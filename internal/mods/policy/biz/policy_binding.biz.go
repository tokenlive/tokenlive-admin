package biz

import (
	"context"
	"time"

	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// PolicyBinding 策略绑定业务逻辑
type PolicyBinding struct {
	Trans            *util.Trans
	PolicyBindingDAL *dal.PolicyBinding
	PolicyRedisSync  *PolicyRedisSync
	AuditLogBIZ      *opsBiz.AuditLog
}

// Query policy bindings from the data access object based on the provided parameters.
func (a *PolicyBinding) Query(ctx context.Context, params schema.PolicyBindingQueryParam) (*schema.PolicyBindingQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyBindingDAL.Query(ctx, params, schema.PolicyBindingQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Get the specified policy binding from the data access object.
func (a *PolicyBinding) Get(ctx context.Context, id string) (*schema.PolicyBindingForm, error) {
	binding, err := a.PolicyBindingDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if binding == nil {
		return nil, errors.NotFound("", "Policy binding not found")
	}
	var form schema.PolicyBindingForm
	if err := binding.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy binding in the data access object.
func (a *PolicyBinding) Create(ctx context.Context, formItem *schema.PolicyBindingForm) (*schema.PolicyBinding, error) {
	// 1. Check unique combination to prevent uk_dimensions_policy database index error.
	existsUnique, err := a.PolicyBindingDAL.ExistsByUniqueKey(ctx, formItem.TenantCode, formItem.UserID, formItem.ModelCode, formItem.PolicyType, formItem.PolicyID)
	if err != nil {
		return nil, err
	} else if existsUnique {
		return nil, errors.BadRequest("", "This specific policy is already bound to the specified dimension")
	}

	// 1.5. Clean logically deleted records that match the unique combination to prevent 1062 database constraint error.
	if err := a.PolicyBindingDAL.CleanDeletedConflict(ctx, formItem.TenantCode, formItem.UserID, formItem.ModelCode, formItem.PolicyType, formItem.PolicyID); err != nil {
		return nil, err
	}

	binding := &schema.PolicyBinding{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	username := util.FromUsername(ctx)
	if username != "" {
		binding.Creator = &username
	}

	if err := formItem.FillTo(binding); err != nil {
		return nil, err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyBindingDAL.Create(ctx, binding)
	})
	if err != nil {
		return nil, err
	}

	// 同步维度到 Redis
	if err := a.PolicyRedisSync.SyncDimension(ctx, binding.TenantCode, binding.UserID, binding.ModelCode); err != nil {
		return nil, err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicyBinding, binding.ID, binding.PolicyID, nil, binding)

	return binding, nil
}

// Update the specified policy binding in the data access object.
func (a *PolicyBinding) Update(ctx context.Context, id string, formItem *schema.PolicyBindingForm) error {
	binding, err := a.PolicyBindingDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if binding == nil {
		return errors.NotFound("", "Policy binding not found")
	}

	// 保存旧维度，如果维度改变需分别重算
	oldTenantCode := binding.TenantCode
	oldUserID := binding.UserID
	oldModelCode := binding.ModelCode

	// If dimension or policy parameters are being updated, ensure constraints are not violated.
	dimChanged := binding.TenantCode != formItem.TenantCode ||
		binding.UserID != formItem.UserID ||
		binding.ModelCode != formItem.ModelCode ||
		binding.PolicyType != formItem.PolicyType

	if dimChanged || binding.PolicyID != formItem.PolicyID {
		existsUnique, err := a.PolicyBindingDAL.ExistsByUniqueKey(ctx, formItem.TenantCode, formItem.UserID, formItem.ModelCode, formItem.PolicyType, formItem.PolicyID)
		if err != nil {
			return err
		} else if existsUnique {
			return errors.BadRequest("", "This specific policy is already bound to the specified dimension")
		}

		// Clean logically deleted records that match the new unique combination to prevent 1062 database constraint error.
		if err := a.PolicyBindingDAL.CleanDeletedConflict(ctx, formItem.TenantCode, formItem.UserID, formItem.ModelCode, formItem.PolicyType, formItem.PolicyID); err != nil {
			return err
		}
	}

	beforePolicy := *binding

	if err := formItem.FillTo(binding); err != nil {
		return err
	}
	binding.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		binding.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyBindingDAL.Update(ctx, binding)
	})
	if err != nil {
		return err
	}

	// 同步新旧维度
	if err := a.PolicyRedisSync.SyncDimension(ctx, binding.TenantCode, binding.UserID, binding.ModelCode); err != nil {
		return err
	}
	if oldTenantCode != binding.TenantCode || oldUserID != binding.UserID || oldModelCode != binding.ModelCode {
		if err := a.PolicyRedisSync.SyncDimension(ctx, oldTenantCode, oldUserID, oldModelCode); err != nil {
			return err
		}
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicyBinding, binding.ID, binding.PolicyID, beforePolicy, binding)

	return nil
}

// ToggleEnabled updates only the enabled status of a policy binding and re-syncs the dimension to Redis.
// Toggling does not change the binding's dimensions, so no exclusive-policy or unique-key checks are needed.
func (a *PolicyBinding) ToggleEnabled(ctx context.Context, id string, formItem *schema.PolicyBindingEnabledForm) error {
	binding, err := a.PolicyBindingDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if binding == nil {
		return errors.NotFound("", "Policy binding not found")
	}

	// No-op if the status is unchanged.
	if binding.Enabled == formItem.Enabled {
		return nil
	}

	beforePolicy := *binding

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyBindingDAL.UpdateEnabled(ctx, id, formItem.Enabled, util.FromUsername(ctx))
	})
	if err != nil {
		return err
	}

	binding.Enabled = formItem.Enabled

	// The aggregated policy snapshot is filtered by enabled = 1, so the dimension must be re-synced.
	if err := a.PolicyRedisSync.SyncDimension(ctx, binding.TenantCode, binding.UserID, binding.ModelCode); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicyBinding, binding.ID, binding.PolicyID, beforePolicy, binding)

	return nil
}

// Delete the specified policy binding from the data access object.
func (a *PolicyBinding) Delete(ctx context.Context, id string) error {
	binding, err := a.PolicyBindingDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if binding == nil {
		return errors.NotFound("", "Policy binding not found")
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyBindingDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 删除绑定后重算维度同步
	if err := a.PolicyRedisSync.SyncDimension(ctx, binding.TenantCode, binding.UserID, binding.ModelCode); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicyBinding, binding.ID, binding.PolicyID, binding, nil)

	return nil
}

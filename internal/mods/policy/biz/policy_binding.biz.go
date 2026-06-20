package biz

import (
	"context"
	"time"

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
}

// isExclusivePolicy checks if the policy type is exclusive (single active instance per dimension).
func isExclusivePolicy(policyType string) bool {
	return policyType == "invocation" || policyType == "loadbalance"
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

	// 2. Industrial Spec: check exclusive policy types (invocation, loadbalance)
	if isExclusivePolicy(formItem.PolicyType) {
		existsDim, err := a.PolicyBindingDAL.ExistsByDimensions(ctx, formItem.TenantCode, formItem.UserID, formItem.ModelCode, formItem.PolicyType)
		if err != nil {
			return nil, err
		} else if existsDim {
			return nil, errors.BadRequest("", "An exclusive policy of type '%s' is already bound to this dimension. Please remove or modify the existing binding first", formItem.PolicyType)
		}
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

	if dimChanged && isExclusivePolicy(formItem.PolicyType) {
		existsDim, err := a.PolicyBindingDAL.ExistsByDimensions(ctx, formItem.TenantCode, formItem.UserID, formItem.ModelCode, formItem.PolicyType)
		if err != nil {
			return err
		} else if existsDim {
			return errors.BadRequest("", "An exclusive policy of type '%s' is already bound to this dimension. Please remove or modify the existing binding first", formItem.PolicyType)
		}
	}

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

	return nil
}

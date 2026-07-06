package biz

import (
	"context"
	"time"

	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Loadbalance policy management
type PolicyLoadbalance struct {
	Trans                *util.Trans
	PolicyLoadbalanceDAL *dal.PolicyLoadbalance
	PolicyRedisSync      *PolicyRedisSync
	ModelDAL             *resourceDal.Model
	DataPermissionDAL    *resourceDal.DataPermission
	AuditLogBIZ          *opsBiz.AuditLog
}

// Query policy loadbalances from the data access object based on the provided parameters and options.
func (a *PolicyLoadbalance) Query(ctx context.Context, params schema.PolicyLoadbalanceQueryParam) (*schema.PolicyLoadbalanceQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyLoadbalanceDAL.Query(ctx, params, schema.PolicyLoadbalanceQueryOptions{
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

// Get the specified policy loadbalance from the data access object.
func (a *PolicyLoadbalance) Get(ctx context.Context, id string) (*schema.PolicyLoadbalanceForm, error) {
	policyLoadbalance, err := a.PolicyLoadbalanceDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyLoadbalance == nil {
		return nil, errors.NotFound("", "Policy loadbalance not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyLoadbalance.ModelID, modelPermissionRead); err != nil {
		return nil, err
	}
	var form schema.PolicyLoadbalanceForm
	if err := policyLoadbalance.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy loadbalance in the data access object.
func (a *PolicyLoadbalance) Create(ctx context.Context, formItem *schema.PolicyLoadbalanceForm) (*schema.PolicyLoadbalance, error) {
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite); err != nil {
		return nil, err
	}

	if exists, err := a.PolicyLoadbalanceDAL.ExistsByName(ctx, formItem.ScopeType, formItem.ScopeCode, formItem.ModelID, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy loadbalance with the same name already exists")
	}

	creator := util.FromUsername(ctx)
	policyLoadbalance := &schema.PolicyLoadbalance{
		ID:        util.NewXID(),
		Deleted:   "0",
		Creator:   &creator,
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyLoadbalance); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLoadbalanceDAL.Create(ctx, policyLoadbalance)
	})
	if err != nil {
		return nil, err
	}
	_ = a.PolicyRedisSync.SyncPolicyChange(ctx, policyLoadbalance.ScopeType, policyLoadbalance.ScopeCode, policyLoadbalance.ModelID)

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyLoadbalance.ID, policyLoadbalance.Name, nil, policyLoadbalance)

	return policyLoadbalance, nil
}

// Update the specified policy loadbalance in the data access object.
func (a *PolicyLoadbalance) Update(ctx context.Context, id string, formItem *schema.PolicyLoadbalanceForm) error {
	policyLoadbalance, err := a.PolicyLoadbalanceDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLoadbalance == nil {
		return errors.NotFound("", "Policy loadbalance not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyLoadbalance.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if err := rejectPolicyKindChange(policyLoadbalance.ModelID, formItem.ModelID); err != nil {
		return err
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyLoadbalance.ScopeType != formItem.ScopeType || policyLoadbalance.ScopeCode != formItem.ScopeCode || policyLoadbalance.ModelID != formItem.ModelID || policyLoadbalance.Name != formItem.Name {
		if exists, err := a.PolicyLoadbalanceDAL.ExistsByName(ctx, formItem.ScopeType, formItem.ScopeCode, formItem.ModelID, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy loadbalance with the same name already exists")
		}
	}

	beforePolicy := *policyLoadbalance

	if err := formItem.FillTo(policyLoadbalance); err != nil {
		return err
	}
	modifier := util.FromUsername(ctx)
	policyLoadbalance.Modifier = &modifier
	policyLoadbalance.UpdatedAt = time.Now()

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLoadbalanceDAL.Update(ctx, policyLoadbalance)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	_ = a.PolicyRedisSync.SyncPolicyChange(ctx, beforePolicy.ScopeType, beforePolicy.ScopeCode, beforePolicy.ModelID)
	if beforePolicy.ScopeType != policyLoadbalance.ScopeType || beforePolicy.ScopeCode != policyLoadbalance.ScopeCode || beforePolicy.ModelID != policyLoadbalance.ModelID {
		_ = a.PolicyRedisSync.SyncPolicyChange(ctx, policyLoadbalance.ScopeType, policyLoadbalance.ScopeCode, policyLoadbalance.ModelID)
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyLoadbalance.ID, policyLoadbalance.Name, beforePolicy, policyLoadbalance)

	return nil
}

// Delete the specified policy loadbalance from the data access object.
func (a *PolicyLoadbalance) Delete(ctx context.Context, id string) error {
	policyLoadbalance, err := a.PolicyLoadbalanceDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLoadbalance == nil {
		return errors.NotFound("", "Policy loadbalance not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyLoadbalance.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLoadbalanceDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	_ = a.PolicyRedisSync.SyncPolicyChange(ctx, policyLoadbalance.ScopeType, policyLoadbalance.ScopeCode, policyLoadbalance.ModelID)

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyLoadbalance.ID, policyLoadbalance.Name, policyLoadbalance, nil)

	return nil
}

// CopyTemplateToModel copies a policy template into a model-owned policy instance.
func (a *PolicyLoadbalance) CopyTemplateToModel(ctx context.Context, templateID string, form *schema.PolicyCopyToModelForm) (*schema.PolicyLoadbalance, error) {
	template, err := a.PolicyLoadbalanceDAL.Get(ctx, templateID)
	if err != nil {
		return nil, err
	} else if template == nil {
		return nil, errors.NotFound("", "Policy loadbalance not found")
	}
	if template.ModelID != "" {
		return nil, errors.BadRequest("", "Only policy templates can be copied to a model")
	}
	if _, err := requireModelPermission(ctx, a.ModelDAL, a.DataPermissionDAL, form.ModelID, modelPermissionWrite); err != nil {
		return nil, err
	}
	name := form.Name
	if name == "" {
		name = template.Name
	}
	name, err = nextPolicyName(ctx, name, form.ModelID, func(ctx context.Context, modelID, name string) (bool, error) {
		// nextPolicyName 内部使用的是没有 tenant_code 和 user_id 的签名，为了兼容我们可以包装下，只根据 model_id 查找
		return a.PolicyLoadbalanceDAL.ExistsByName(ctx, "global", "", modelID, name)
	})
	if err != nil {
		return nil, err
	}

	instance := *template
	instance.ID = util.NewXID()
	instance.ModelID = form.ModelID
	instance.Name = name
	instance.Creator = nil
	instance.Modifier = nil
	instance.CreatedAt = time.Now()
	instance.UpdatedAt = time.Time{}
	instance.Deleted = "0"
	instance.DeletedAt = nil
	if form.ScopeType != nil {
		instance.ScopeType = *form.ScopeType
	}
	if form.ScopeCode != nil {
		instance.ScopeCode = *form.ScopeCode
	}
	if form.Priority != nil {
		instance.Priority = *form.Priority
	}
	if username := util.FromUsername(ctx); username != "" {
		instance.Creator = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLoadbalanceDAL.Create(ctx, &instance)
	})
	if err != nil {
		return nil, err
	}
	_ = a.PolicyRedisSync.SyncPolicyChange(ctx, instance.ScopeType, instance.ScopeCode, instance.ModelID)
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, instance.ID, instance.Name, nil, &instance)
	return &instance, nil
}

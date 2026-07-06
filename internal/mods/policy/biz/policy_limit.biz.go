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

// Limit policy management
type PolicyLimit struct {
	Trans             *util.Trans
	PolicyLimitDAL    *dal.PolicyLimit
	PolicyBindingDAL  *dal.PolicyBinding
	PolicyRedisSync   *PolicyRedisSync
	ModelDAL          *resourceDal.Model
	DataPermissionDAL *resourceDal.DataPermission
	AuditLogBIZ       *opsBiz.AuditLog
}

// Query policy limits from the data access object based on the provided parameters and options.
func (a *PolicyLimit) Query(ctx context.Context, params schema.PolicyLimitQueryParam) (*schema.PolicyLimitQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyLimitDAL.Query(ctx, params, schema.PolicyLimitQueryOptions{
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

// Get the specified policy limit from the data access object.
func (a *PolicyLimit) Get(ctx context.Context, id string) (*schema.PolicyLimitForm, error) {
	policyLimit, err := a.PolicyLimitDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyLimit == nil {
		return nil, errors.NotFound("", "Policy limit not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyLimit.ModelID, modelPermissionRead); err != nil {
		return nil, err
	}
	var policyLimitForm schema.PolicyLimitForm
	if err := policyLimit.ConvertTo(&policyLimitForm); err != nil {
		return nil, err
	}
	return &policyLimitForm, nil
}

// Create a new policy limit in the data access object.
func (a *PolicyLimit) Create(ctx context.Context, formItem *schema.PolicyLimitForm) (*schema.PolicyLimit, error) {
	ownerModel, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite)
	if err != nil {
		return nil, err
	}

	if exists, err := a.PolicyLimitDAL.ExistsByUniqueKey(ctx, formItem.ModelID, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy limit with the same name already exists")
	}

	policyLimit := &schema.PolicyLimit{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyLimit.Creator = &username
	}

	if err := formItem.FillTo(policyLimit); err != nil {
		return nil, err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyLimitDAL.Create(ctx, policyLimit); err != nil {
			return err
		}
		if ownerModel == nil {
			return nil
		}
		return createModelPolicyBinding(ctx, a.PolicyBindingDAL, "limit", policyLimit.ID, ownerModel)
	})
	if err != nil {
		return nil, err
	}
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyLimit.ID, policyLimit.Name, nil, policyLimit)
	return policyLimit, nil
}

// Update the specified policy limit in the data access object.
func (a *PolicyLimit) Update(ctx context.Context, id string, formItem *schema.PolicyLimitForm) error {
	policyLimit, err := a.PolicyLimitDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLimit == nil {
		return errors.NotFound("", "Policy limit not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyLimit.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if err := rejectPolicyKindChange(policyLimit.ModelID, formItem.ModelID); err != nil {
		return err
	}
	model, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite)
	if err != nil {
		return err
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyLimit.ModelID != formItem.ModelID || policyLimit.Name != formItem.Name {
		if exists, err := a.PolicyLimitDAL.ExistsByUniqueKey(ctx, formItem.ModelID, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy limit with the same name already exists")
		}
	}

	beforePolicy := *policyLimit

	if err := formItem.FillTo(policyLimit); err != nil {
		return err
	}
	policyLimit.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyLimit.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyLimitDAL.Update(ctx, policyLimit); err != nil {
			return err
		}
		if model == nil {
			return a.PolicyBindingDAL.DeleteByPolicyID(ctx, "limit", policyLimit.ID)
		}
		return replaceModelPolicyBinding(ctx, a.PolicyBindingDAL, "limit", policyLimit.ID, model)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if policyLimit.ModelID != "" {
		if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "limit", id); err != nil {
			return err
		}
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyLimit.ID, policyLimit.Name, beforePolicy, policyLimit)
	return nil
}

// Delete the specified policy limit from the data access object.
func (a *PolicyLimit) Delete(ctx context.Context, id string) error {
	policyLimit, err := a.PolicyLimitDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLimit == nil {
		return errors.NotFound("", "Policy limit not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyLimit.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyLimitDAL.Delete(ctx, id); err != nil {
			return err
		}
		return a.PolicyBindingDAL.DeleteByPolicyID(ctx, "limit", id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if policyLimit.ModelID != "" {
		if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "limit", id); err != nil {
			return err
		}
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyLimit.ID, policyLimit.Name, policyLimit, nil)
	return nil
}

// CopyTemplateToModel copies a policy template into a model-owned policy instance.
func (a *PolicyLimit) CopyTemplateToModel(ctx context.Context, templateID, modelID, name string) (*schema.PolicyLimit, error) {
	template, err := a.PolicyLimitDAL.Get(ctx, templateID)
	if err != nil {
		return nil, err
	} else if template == nil {
		return nil, errors.NotFound("", "Policy limit not found")
	}
	if template.ModelID != "" {
		return nil, errors.BadRequest("", "Only policy templates can be copied to a model")
	}
	model, err := requireModelPermission(ctx, a.ModelDAL, a.DataPermissionDAL, modelID, modelPermissionWrite)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = template.Name
	}
	name, err = nextPolicyName(ctx, name, modelID, a.PolicyLimitDAL.ExistsByUniqueKey)
	if err != nil {
		return nil, err
	}

	instance := *template
	instance.ID = util.NewXID()
	instance.ModelID = modelID
	instance.Name = name
	instance.Creator = nil
	instance.Modifier = nil
	instance.CreatedAt = time.Now()
	instance.UpdatedAt = time.Time{}
	instance.Deleted = "0"
	instance.DeletedAt = nil
	if username := util.FromUsername(ctx); username != "" {
		instance.Creator = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyLimitDAL.Create(ctx, &instance); err != nil {
			return err
		}
		return createModelPolicyBinding(ctx, a.PolicyBindingDAL, "limit", instance.ID, model)
	})
	if err != nil {
		return nil, err
	}
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, instance.ID, instance.Name, nil, &instance)
	return &instance, nil
}

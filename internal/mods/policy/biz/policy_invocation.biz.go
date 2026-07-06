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

// Invocation policy management
type PolicyInvocation struct {
	Trans               *util.Trans
	PolicyInvocationDAL *dal.PolicyInvocation
	PolicyBindingDAL    *dal.PolicyBinding
	PolicyRedisSync     *PolicyRedisSync
	ModelDAL            *resourceDal.Model
	DataPermissionDAL   *resourceDal.DataPermission
	AuditLogBIZ         *opsBiz.AuditLog
}

// Query policy invocations from the data access object based on the provided parameters and options.
func (a *PolicyInvocation) Query(ctx context.Context, params schema.PolicyInvocationQueryParam) (*schema.PolicyInvocationQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyInvocationDAL.Query(ctx, params, schema.PolicyInvocationQueryOptions{
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

// Get the specified policy invocation from the data access object.
func (a *PolicyInvocation) Get(ctx context.Context, id string) (*schema.PolicyInvocationForm, error) {
	policyInvocation, err := a.PolicyInvocationDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyInvocation == nil {
		return nil, errors.NotFound("", "Policy invocation not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyInvocation.ModelID, modelPermissionRead); err != nil {
		return nil, err
	}
	var form schema.PolicyInvocationForm
	if err := policyInvocation.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy invocation in the data access object.
func (a *PolicyInvocation) Create(ctx context.Context, formItem *schema.PolicyInvocationForm) (*schema.PolicyInvocation, error) {
	model, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite)
	if err != nil {
		return nil, err
	}

	// Check unique key before creating.
	if exists, err := a.PolicyInvocationDAL.ExistsByUniqueKey(ctx, formItem.ModelID, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy invocation with the same name already exists")
	}

	policyInvocation := &schema.PolicyInvocation{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyInvocation); err != nil {
		return nil, err
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyInvocation.Creator = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyInvocationDAL.Create(ctx, policyInvocation); err != nil {
			return err
		}
		if model == nil {
			return nil
		}
		return createModelPolicyBinding(ctx, a.PolicyBindingDAL, "invocation", policyInvocation.ID, model)
	})
	if err != nil {
		return nil, err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyInvocation.ID, policyInvocation.Name, nil, policyInvocation)

	return policyInvocation, nil
}

// Update the specified policy invocation in the data access object.
func (a *PolicyInvocation) Update(ctx context.Context, id string, formItem *schema.PolicyInvocationForm) error {
	policyInvocation, err := a.PolicyInvocationDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyInvocation == nil {
		return errors.NotFound("", "Policy invocation not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyInvocation.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if err := rejectPolicyKindChange(policyInvocation.ModelID, formItem.ModelID); err != nil {
		return err
	}
	model, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite)
	if err != nil {
		return err
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyInvocation.ModelID != formItem.ModelID || policyInvocation.Name != formItem.Name {
		if exists, err := a.PolicyInvocationDAL.ExistsByUniqueKey(ctx, formItem.ModelID, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy invocation with the same name already exists")
		}
	}

	beforePolicy := *policyInvocation

	if err := formItem.FillTo(policyInvocation); err != nil {
		return err
	}
	policyInvocation.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyInvocation.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyInvocationDAL.Update(ctx, policyInvocation); err != nil {
			return err
		}
		if model == nil {
			return a.PolicyBindingDAL.DeleteByPolicyID(ctx, "invocation", policyInvocation.ID)
		}
		return replaceModelPolicyBinding(ctx, a.PolicyBindingDAL, "invocation", policyInvocation.ID, model)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if policyInvocation.ModelID != "" {
		if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "invocation", id); err != nil {
			return err
		}
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyInvocation.ID, policyInvocation.Name, beforePolicy, policyInvocation)

	return nil
}

// Delete the specified policy invocation from the data access object.
func (a *PolicyInvocation) Delete(ctx context.Context, id string) error {
	policyInvocation, err := a.PolicyInvocationDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyInvocation == nil {
		return errors.NotFound("", "Policy invocation not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyInvocation.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyInvocationDAL.Delete(ctx, id); err != nil {
			return err
		}
		return a.PolicyBindingDAL.DeleteByPolicyID(ctx, "invocation", id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if policyInvocation.ModelID != "" {
		if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "invocation", id); err != nil {
			return err
		}
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyInvocation.ID, policyInvocation.Name, policyInvocation, nil)

	return nil
}

// CopyTemplateToModel copies a policy template into a model-owned policy instance.
func (a *PolicyInvocation) CopyTemplateToModel(ctx context.Context, templateID, modelID, name string) (*schema.PolicyInvocation, error) {
	template, err := a.PolicyInvocationDAL.Get(ctx, templateID)
	if err != nil {
		return nil, err
	} else if template == nil {
		return nil, errors.NotFound("", "Policy invocation not found")
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
	name, err = nextPolicyName(ctx, name, modelID, a.PolicyInvocationDAL.ExistsByUniqueKey)
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
		if err := a.PolicyInvocationDAL.Create(ctx, &instance); err != nil {
			return err
		}
		return createModelPolicyBinding(ctx, a.PolicyBindingDAL, "invocation", instance.ID, model)
	})
	if err != nil {
		return nil, err
	}
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, instance.ID, instance.Name, nil, &instance)
	return &instance, nil
}

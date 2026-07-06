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

// Circuit break policy management
type PolicyCircuitBreak struct {
	Trans                 *util.Trans
	PolicyCircuitBreakDAL *dal.PolicyCircuitBreak
	PolicyBindingDAL      *dal.PolicyBinding
	PolicyRedisSync       *PolicyRedisSync
	ModelDAL              *resourceDal.Model
	DataPermissionDAL     *resourceDal.DataPermission
	AuditLogBIZ           *opsBiz.AuditLog
}

// Query policy circuit breaks from the data access object based on the provided parameters and options.
func (a *PolicyCircuitBreak) Query(ctx context.Context, params schema.PolicyCircuitBreakQueryParam) (*schema.PolicyCircuitBreakQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyCircuitBreakDAL.Query(ctx, params, schema.PolicyCircuitBreakQueryOptions{
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

// Get the specified policy circuit break from the data access object.
func (a *PolicyCircuitBreak) Get(ctx context.Context, id string) (*schema.PolicyCircuitBreakForm, error) {
	policyCircuitBreak, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyCircuitBreak == nil {
		return nil, errors.NotFound("", "Policy circuit break not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyCircuitBreak.ModelID, modelPermissionRead); err != nil {
		return nil, err
	}
	var form schema.PolicyCircuitBreakForm
	if err := policyCircuitBreak.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy circuit break in the data access object.
func (a *PolicyCircuitBreak) Create(ctx context.Context, formItem *schema.PolicyCircuitBreakForm) (*schema.PolicyCircuitBreak, error) {
	model, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite)
	if err != nil {
		return nil, err
	}

	// Check unique key before creating.
	if exists, err := a.PolicyCircuitBreakDAL.ExistsByUniqueKey(ctx, formItem.ModelID, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy circuit break with the same name already exists")
	}

	policyCircuitBreak := &schema.PolicyCircuitBreak{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyCircuitBreak); err != nil {
		return nil, err
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyCircuitBreak.Creator = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyCircuitBreakDAL.Create(ctx, policyCircuitBreak); err != nil {
			return err
		}
		if model == nil {
			return nil
		}
		return createModelPolicyBinding(ctx, a.PolicyBindingDAL, "circuit_break", policyCircuitBreak.ID, model)
	})
	if err != nil {
		return nil, err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyCircuitBreak.ID, policyCircuitBreak.Name, nil, policyCircuitBreak)

	return policyCircuitBreak, nil
}

// Update the specified policy circuit break in the data access object.
func (a *PolicyCircuitBreak) Update(ctx context.Context, id string, formItem *schema.PolicyCircuitBreakForm) error {
	policyCircuitBreak, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyCircuitBreak == nil {
		return errors.NotFound("", "Policy circuit break not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyCircuitBreak.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if err := rejectPolicyKindChange(policyCircuitBreak.ModelID, formItem.ModelID); err != nil {
		return err
	}
	model, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite)
	if err != nil {
		return err
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyCircuitBreak.ModelID != formItem.ModelID || policyCircuitBreak.Name != formItem.Name {
		if exists, err := a.PolicyCircuitBreakDAL.ExistsByUniqueKey(ctx, formItem.ModelID, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy circuit break with the same name already exists")
		}
	}

	beforePolicy := *policyCircuitBreak

	if err := formItem.FillTo(policyCircuitBreak); err != nil {
		return err
	}
	policyCircuitBreak.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyCircuitBreak.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyCircuitBreakDAL.Update(ctx, policyCircuitBreak); err != nil {
			return err
		}
		if model == nil {
			return a.PolicyBindingDAL.DeleteByPolicyID(ctx, "circuit_break", policyCircuitBreak.ID)
		}
		return replaceModelPolicyBinding(ctx, a.PolicyBindingDAL, "circuit_break", policyCircuitBreak.ID, model)
	})
	if err != nil {
		return err
	}

	// 级联更新引用此策略的维度到 Redis
	if policyCircuitBreak.ModelID != "" {
		if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "circuit_break", id); err != nil {
			return err
		}
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyCircuitBreak.ID, policyCircuitBreak.Name, beforePolicy, policyCircuitBreak)

	return nil
}

// Delete the specified policy circuit break from the data access object.
func (a *PolicyCircuitBreak) Delete(ctx context.Context, id string) error {
	policyCircuitBreak, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyCircuitBreak == nil {
		return errors.NotFound("", "Policy circuit break not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyCircuitBreak.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyCircuitBreakDAL.Delete(ctx, id); err != nil {
			return err
		}
		return a.PolicyBindingDAL.DeleteByPolicyID(ctx, "circuit_break", id)
	})
	if err != nil {
		return err
	}

	// 级联更新引用此策略的维度到 Redis
	if policyCircuitBreak.ModelID != "" {
		if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "circuit_break", id); err != nil {
			return err
		}
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyCircuitBreak.ID, policyCircuitBreak.Name, policyCircuitBreak, nil)

	return nil
}

// CopyTemplateToModel copies a policy template into a model-owned policy instance.
func (a *PolicyCircuitBreak) CopyTemplateToModel(ctx context.Context, templateID, modelID, name string) (*schema.PolicyCircuitBreak, error) {
	template, err := a.PolicyCircuitBreakDAL.Get(ctx, templateID)
	if err != nil {
		return nil, err
	} else if template == nil {
		return nil, errors.NotFound("", "Policy circuit break not found")
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
	name, err = nextPolicyName(ctx, name, modelID, a.PolicyCircuitBreakDAL.ExistsByUniqueKey)
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
		if err := a.PolicyCircuitBreakDAL.Create(ctx, &instance); err != nil {
			return err
		}
		return createModelPolicyBinding(ctx, a.PolicyBindingDAL, "circuit_break", instance.ID, model)
	})
	if err != nil {
		return nil, err
	}
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, instance.ID, instance.Name, nil, &instance)
	return &instance, nil
}

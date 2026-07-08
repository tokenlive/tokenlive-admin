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

// Tagging policy management
type PolicyTagging struct {
	Trans             *util.Trans
	PolicyTaggingDAL  *dal.PolicyTagging
	PolicyRedisSync   *PolicyRedisSync
	ModelDAL          *resourceDal.Model
	DataPermissionDAL *resourceDal.DataPermission
	AuditLogBIZ       *opsBiz.AuditLog
}

// Query policy taggings from the data access object based on the provided parameters and options.
func (a *PolicyTagging) Query(ctx context.Context, params schema.PolicyTaggingQueryParam) (*schema.PolicyTaggingQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyTaggingDAL.Query(ctx, params, schema.PolicyTaggingQueryOptions{
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

// Get the specified policy tagging from the data access object.
func (a *PolicyTagging) Get(ctx context.Context, id string) (*schema.PolicyTaggingForm, error) {
	policyTagging, err := a.PolicyTaggingDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyTagging == nil {
		return nil, errors.NotFound("", "Policy tagging not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyTagging.ModelID, modelPermissionRead); err != nil {
		return nil, err
	}
	var form schema.PolicyTaggingForm
	if err := policyTagging.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy tagging in the data access object.
func (a *PolicyTagging) Create(ctx context.Context, formItem *schema.PolicyTaggingForm) (*schema.PolicyTagging, error) {
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite); err != nil {
		return nil, err
	}

	if exists, err := a.PolicyTaggingDAL.ExistsByName(ctx, formItem.ScopeType, formItem.ScopeCode, formItem.ModelID, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy tagging with the same name already exists")
	}

	creator := util.FromUsername(ctx)
	policyTagging := &schema.PolicyTagging{
		ID:        util.NewXID(),
		Deleted:   "0",
		Creator:   &creator,
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyTagging); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyTaggingDAL.Create(ctx, policyTagging)
	})
	if err != nil {
		return nil, err
	}
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "tagging", "create", policyTagging.ScopeType, policyTagging.ScopeCode, policyTagging.ModelID)

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyTagging.ID, policyTagging.Name, nil, policyTagging)

	return policyTagging, nil
}

// Update the specified policy tagging in the data access object.
func (a *PolicyTagging) Update(ctx context.Context, id string, formItem *schema.PolicyTaggingForm) error {
	policyTagging, err := a.PolicyTaggingDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyTagging == nil {
		return errors.NotFound("", "Policy tagging not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyTagging.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if err := rejectPolicyKindChange(policyTagging.ModelID, formItem.ModelID); err != nil {
		return err
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyTagging.ModelID != formItem.ModelID || policyTagging.Name != formItem.Name {
		if exists, err := a.PolicyTaggingDAL.ExistsByName(ctx, formItem.ScopeType, formItem.ScopeCode, formItem.ModelID, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy tagging with the same name already exists")
		}
	}

	beforePolicy := *policyTagging

	if err := formItem.FillTo(policyTagging); err != nil {
		return err
	}
	modifier := util.FromUsername(ctx)
	policyTagging.Modifier = &modifier
	policyTagging.UpdatedAt = time.Now()

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyTaggingDAL.Update(ctx, policyTagging)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "tagging", "update_old_dimension", beforePolicy.ScopeType, beforePolicy.ScopeCode, beforePolicy.ModelID)
	if beforePolicy.ScopeType != policyTagging.ScopeType || beforePolicy.ScopeCode != policyTagging.ScopeCode || beforePolicy.ModelID != policyTagging.ModelID {
		_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "tagging", "update_new_dimension", policyTagging.ScopeType, policyTagging.ScopeCode, policyTagging.ModelID)
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyTagging.ID, policyTagging.Name, beforePolicy, policyTagging)

	return nil
}

// Delete the specified policy tagging from the data access object.
func (a *PolicyTagging) Delete(ctx context.Context, id string) error {
	policyTagging, err := a.PolicyTaggingDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyTagging == nil {
		return errors.NotFound("", "Policy tagging not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyTagging.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyTaggingDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "tagging", "delete", policyTagging.ScopeType, policyTagging.ScopeCode, policyTagging.ModelID)

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyTagging.ID, policyTagging.Name, policyTagging, nil)

	return nil
}

// CopyTemplateToModel copies a policy template into a model-owned policy instance.
func (a *PolicyTagging) CopyTemplateToModel(ctx context.Context, templateID string, form *schema.PolicyCopyToModelForm) (*schema.PolicyTagging, error) {
	template, err := a.PolicyTaggingDAL.Get(ctx, templateID)
	if err != nil {
		return nil, err
	} else if template == nil {
		return nil, errors.NotFound("", "Policy tagging not found")
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
		return a.PolicyTaggingDAL.ExistsByName(ctx, "global", "", modelID, name)
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
		return a.PolicyTaggingDAL.Create(ctx, &instance)
	})
	if err != nil {
		return nil, err
	}
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "tagging", "copy_template_to_model", instance.ScopeType, instance.ScopeCode, instance.ModelID)
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, instance.ID, instance.Name, nil, &instance)
	return &instance, nil
}

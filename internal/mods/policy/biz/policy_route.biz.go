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

// Route policy management
type PolicyRoute struct {
	Trans             *util.Trans
	PolicyRouteDAL    *dal.PolicyRoute
	PolicyRedisSync   PolicyChangeSyncer
	ModelDAL          *resourceDal.Model
	DataPermissionDAL *resourceDal.DataPermission
	AuditLogBIZ       *opsBiz.AuditLog
}

// Query policy routes from the data access object based on the provided parameters and options.
func (a *PolicyRoute) Query(ctx context.Context, params schema.PolicyRouteQueryParam) (*schema.PolicyRouteQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyRouteDAL.Query(ctx, params, schema.PolicyRouteQueryOptions{
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

// Get the specified policy route from the data access object.
func (a *PolicyRoute) Get(ctx context.Context, id string) (*schema.PolicyRouteForm, error) {
	policyRoute, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyRoute == nil {
		return nil, errors.NotFound("", "Policy route not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyRoute.ModelID, modelPermissionRead); err != nil {
		return nil, err
	}
	var policyRouteForm schema.PolicyRouteForm
	if err := policyRoute.ConvertTo(&policyRouteForm); err != nil {
		return nil, err
	}
	return &policyRouteForm, nil
}

// Create a new policy route in the data access object.
func (a *PolicyRoute) Create(ctx context.Context, formItem *schema.PolicyRouteForm) (*schema.PolicyRoute, error) {
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite); err != nil {
		return nil, err
	}

	if exists, err := a.PolicyRouteDAL.ExistsByUniqueKey(ctx, formItem.ScopeType, formItem.ScopeCode, formItem.ModelID, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy route with the same name already exists")
	}

	policyRoute := &schema.PolicyRoute{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyRoute.Creator = &username
	}

	if err := formItem.FillTo(policyRoute); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Create(ctx, policyRoute)
	})
	if err != nil {
		return nil, err
	}
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "route", "create", policyRoute.ScopeType, policyRoute.ScopeCode, policyRoute.ModelID)
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyRoute.ID, policyRoute.Name, nil, policyRoute)
	return policyRoute, nil
}

// Update the specified policy route in the data access object.
func (a *PolicyRoute) Update(ctx context.Context, id string, formItem *schema.PolicyRouteForm) error {
	policyRoute, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyRoute == nil {
		return errors.NotFound("", "Policy route not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyRoute.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if err := rejectPolicyKindChange(policyRoute.ModelID, formItem.ModelID); err != nil {
		return err
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, formItem.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyRoute.ModelID != formItem.ModelID || policyRoute.Name != formItem.Name {
		if exists, err := a.PolicyRouteDAL.ExistsByUniqueKey(ctx, formItem.ScopeType, formItem.ScopeCode, formItem.ModelID, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy route with the same name already exists")
		}
	}

	beforePolicy := *policyRoute

	if err := formItem.FillTo(policyRoute); err != nil {
		return err
	}
	policyRoute.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyRoute.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Update(ctx, policyRoute)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "route", "update_old_dimension", beforePolicy.ScopeType, beforePolicy.ScopeCode, beforePolicy.ModelID)
	if beforePolicy.ScopeType != policyRoute.ScopeType || beforePolicy.ScopeCode != policyRoute.ScopeCode || beforePolicy.ModelID != policyRoute.ModelID {
		_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "route", "update_new_dimension", policyRoute.ScopeType, policyRoute.ScopeCode, policyRoute.ModelID)
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyRoute.ID, policyRoute.Name, beforePolicy, policyRoute)
	return nil
}

// ToggleEnabled updates only the enabled status of a policy and re-syncs policy cache.
func (a *PolicyRoute) ToggleEnabled(ctx context.Context, id string, formItem *schema.PolicyEnabledForm) error {
	policyRoute, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyRoute == nil {
		return errors.NotFound("", "Policy route not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyRoute.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if policyRoute.Enabled == formItem.Enabled {
		return nil
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.UpdateEnabled(ctx, id, formItem.Enabled, util.FromUsername(ctx))
	})
	if err != nil {
		return err
	}

	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "route", "toggle_enabled", policyRoute.ScopeType, policyRoute.ScopeCode, policyRoute.ModelID)
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyRoute.ID, policyRoute.Name, map[string]int{"enabled": policyRoute.Enabled}, map[string]int{"enabled": formItem.Enabled})
	return nil
}

// Delete the specified policy route from the data access object.
func (a *PolicyRoute) Delete(ctx context.Context, id string) error {
	policyRoute, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyRoute == nil {
		return errors.NotFound("", "Policy route not found")
	}
	if _, err := requireExistingModelPolicy(ctx, a.ModelDAL, a.DataPermissionDAL, policyRoute.ModelID, modelPermissionWrite); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "route", "delete", policyRoute.ScopeType, policyRoute.ScopeCode, policyRoute.ModelID)

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyRoute.ID, policyRoute.Name, policyRoute, nil)
	return nil
}

// CopyTemplateToModel copies a policy template into a model-owned policy instance.
func (a *PolicyRoute) CopyTemplateToModel(ctx context.Context, templateID string, form *schema.PolicyCopyToModelForm) (*schema.PolicyRoute, error) {
	template, err := a.PolicyRouteDAL.Get(ctx, templateID)
	if err != nil {
		return nil, err
	} else if template == nil {
		return nil, errors.NotFound("", "Policy route not found")
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
		return a.PolicyRouteDAL.ExistsByUniqueKey(ctx, "global", "", modelID, name)
	})
	if err != nil {
		return nil, err
	}

	var details []schema.PolicyRouteDetail
	if err := util.GetDB(ctx, a.PolicyRouteDAL.DB).
		Where("route_id = ? AND deleted = '0'", template.ID).
		Find(&details).Error; err != nil {
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
	instance.Details = nil
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
		if err := a.PolicyRouteDAL.Create(ctx, &instance); err != nil {
			return err
		}
		for _, detail := range details {
			copied := detail
			copied.ID = util.NewXID()
			copied.RouteId = instance.ID
			copied.CreatedAt = time.Now()
			copied.UpdatedAt = time.Time{}
			copied.Deleted = "0"
			copied.DeletedAt = nil
			if err := util.GetDB(ctx, a.PolicyRouteDAL.DB).Create(&copied).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	_ = syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "route", "copy_template_to_model", instance.ScopeType, instance.ScopeCode, instance.ModelID)
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, instance.ID, instance.Name, nil, &instance)
	return &instance, nil
}

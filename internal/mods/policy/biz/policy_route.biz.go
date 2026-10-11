package biz

import (
	"context"
	"time"

	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
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

func (a *PolicyRoute) lifecycle() policyLifecycle {
	return policyLifecycle{
		modelDAL:          a.ModelDAL,
		dataPermissionDAL: a.DataPermissionDAL,
		trans:             a.Trans,
		syncer:            a.PolicyRedisSync,
		audit:             a.AuditLogBIZ,
		store:             routeStore{a.PolicyRouteDAL},
		kind:              "route",
		notFound:          func() error { return errors.NotFound("", "Policy route not found") },
		duplicate:         func() error { return errors.BadRequest("", "Policy route with the same name already exists") },
	}
}

type routeStore struct{ dal *dal.PolicyRoute }

func (s routeStore) get(ctx context.Context, id string) (policyRecord, bool, error) {
	item, err := s.dal.Get(ctx, id)
	if err != nil || item == nil {
		return policyRecord{}, false, err
	}
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled, Creator: item.Creator, Modifier: item.Modifier,
	}, true, nil
}

func (s routeStore) existsName(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error) {
	return s.dal.ExistsByUniqueKey(ctx, scopeType, scopeCode, modelID, name)
}

func (s routeStore) updateEnabled(ctx context.Context, id string, enabled int, modifier string) error {
	return s.dal.UpdateEnabled(ctx, id, enabled, modifier)
}

func (s routeStore) delete(ctx context.Context, id string) error {
	return s.dal.Delete(ctx, id)
}

type routeForm struct{ *schema.PolicyRouteForm }

func (f routeForm) modelID() string   { return f.ModelID }
func (f routeForm) scopeType() string { return f.ScopeType }
func (f routeForm) scopeCode() string { return f.ScopeCode }
func (f routeForm) name() string      { return f.Name }

// Query policy routes from the data access object based on the provided parameters and options.
func (a *PolicyRoute) Query(ctx context.Context, params schema.PolicyRouteQueryParam) (*schema.PolicyRouteQueryResult, error) {
	params.Pagination = false

	return a.PolicyRouteDAL.Query(ctx, params, schema.PolicyRouteQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
}

// Get the specified policy route from the data access object.
func (a *PolicyRoute) Get(ctx context.Context, id string) (*schema.PolicyRouteForm, error) {
	item, ok, err := a.lifecycle().store.get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, a.lifecycle().notFound()
	}
	if err := a.lifecycle().require(ctx, item.ModelID, modelPermissionRead); err != nil {
		return nil, err
	}
	stored, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var form schema.PolicyRouteForm
	if err := stored.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy route in the data access object.
func (a *PolicyRoute) Create(ctx context.Context, formItem *schema.PolicyRouteForm) (*schema.PolicyRoute, error) {
	life := a.lifecycle()
	prepared, err := life.prepareCreate(ctx, routeForm{formItem})
	if err != nil {
		return nil, err
	}
	item := &schema.PolicyRoute{
		ID:        prepared.ID,
		Deleted:   "0",
		Creator:   prepared.Creator,
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(item); err != nil {
		return nil, err
	}
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Create(ctx, item)
	}); err != nil {
		return nil, err
	}
	life.afterCreate(ctx, recordOfRoute(item), item)
	return item, nil
}

// Update the specified policy route in the data access object.
func (a *PolicyRoute) Update(ctx context.Context, id string, formItem *schema.PolicyRouteForm) error {
	life := a.lifecycle()
	before, err := life.beginUpdate(ctx, id, routeForm{formItem})
	if err != nil {
		return err
	}
	item, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return err
	}
	beforeStored := *item
	if err := formItem.FillTo(item); err != nil {
		return err
	}
	item.UpdatedAt = time.Now()
	if modifier, set := life.modifier(ctx); set {
		item.Modifier = modifier
	}
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Update(ctx, item)
	}); err != nil {
		return err
	}
	life.afterUpdate(ctx, before, recordOfRoute(item), &beforeStored, item)
	return nil
}

// ToggleEnabled updates only the enabled status of a policy and re-syncs policy cache.
func (a *PolicyRoute) ToggleEnabled(ctx context.Context, id string, formItem *schema.PolicyEnabledForm) error {
	return a.lifecycle().toggle(ctx, id, formItem.Enabled)
}

// Delete the specified policy route from the data access object.
func (a *PolicyRoute) Delete(ctx context.Context, id string) error {
	life := a.lifecycle()
	stored, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return err
	}
	item, err := life.remove(ctx, id)
	if err != nil {
		return err
	}
	life.afterDelete(ctx, item, stored)
	return nil
}

// CopyTemplateToModel copies a policy template into a model-owned policy instance.
func (a *PolicyRoute) CopyTemplateToModel(ctx context.Context, templateID string, form *schema.PolicyCopyToModelForm) (*schema.PolicyRoute, error) {
	life := a.lifecycle()
	template, ok, err := life.store.get(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, life.notFound()
	}
	copied, err := life.prepareCopy(ctx, template, form)
	if err != nil {
		return nil, err
	}
	stored, err := a.PolicyRouteDAL.Get(ctx, templateID)
	if err != nil {
		return nil, err
	}
	instance := *stored
	instance.ID = copied.ID
	instance.ModelID = copied.ModelID
	instance.Name = copied.Name
	instance.ScopeType = copied.ScopeType
	instance.ScopeCode = copied.ScopeCode
	instance.Priority = copied.Priority
	instance.Creator = copied.Creator
	instance.Modifier = nil
	instance.CreatedAt = copied.CreatedAt
	instance.UpdatedAt = time.Time{}
	instance.Deleted = "0"
	instance.DeletedAt = nil
	instance.Details = nil
	var details []schema.PolicyRouteDetail
	if err := util.GetDB(ctx, a.PolicyRouteDAL.DB).
		Where("route_id = ? AND deleted = '0'", templateID).
		Find(&details).Error; err != nil {
		return nil, err
	}
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.PolicyRouteDAL.Create(ctx, &instance); err != nil {
			return err
		}
		for _, detail := range details {
			copiedDetail := detail
			copiedDetail.ID = util.NewXID()
			copiedDetail.RouteId = instance.ID
			copiedDetail.CreatedAt = time.Now()
			copiedDetail.UpdatedAt = time.Time{}
			copiedDetail.Deleted = "0"
			copiedDetail.DeletedAt = nil
			if err := util.GetDB(ctx, a.PolicyRouteDAL.DB).Create(&copiedDetail).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	life.afterCopy(ctx, recordOfRoute(&instance), &instance)
	return &instance, nil
}

func recordOfRoute(item *schema.PolicyRoute) policyRecord {
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled,
	}
}

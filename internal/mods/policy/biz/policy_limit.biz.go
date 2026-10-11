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

// Limit policy management
type PolicyLimit struct {
	Trans             *util.Trans
	PolicyLimitDAL    *dal.PolicyLimit
	PolicyRedisSync   PolicyChangeSyncer
	ModelDAL          *resourceDal.Model
	DataPermissionDAL *resourceDal.DataPermission
	AuditLogBIZ       *opsBiz.AuditLog
}

func (a *PolicyLimit) lifecycle() policyLifecycle {
	return policyLifecycle{
		modelDAL:          a.ModelDAL,
		dataPermissionDAL: a.DataPermissionDAL,
		trans:             a.Trans,
		syncer:            a.PolicyRedisSync,
		audit:             a.AuditLogBIZ,
		store:             limitStore{a.PolicyLimitDAL},
		kind:              "limit",
		notFound:          func() error { return errors.NotFound("", "Policy limit not found") },
		duplicate:         func() error { return errors.BadRequest("", "Policy limit with the same name already exists") },
	}
}

type limitStore struct{ dal *dal.PolicyLimit }

func (s limitStore) get(ctx context.Context, id string) (policyRecord, bool, error) {
	item, err := s.dal.Get(ctx, id)
	if err != nil || item == nil {
		return policyRecord{}, false, err
	}
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled, Creator: item.Creator, Modifier: item.Modifier,
	}, true, nil
}

func (s limitStore) existsName(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error) {
	return s.dal.ExistsByUniqueKey(ctx, scopeType, scopeCode, modelID, name)
}

func (s limitStore) updateEnabled(ctx context.Context, id string, enabled int, modifier string) error {
	return s.dal.UpdateEnabled(ctx, id, enabled, modifier)
}

func (s limitStore) delete(ctx context.Context, id string) error {
	return s.dal.Delete(ctx, id)
}

type limitForm struct{ *schema.PolicyLimitForm }

func (f limitForm) modelID() string   { return f.ModelID }
func (f limitForm) scopeType() string { return f.ScopeType }
func (f limitForm) scopeCode() string { return f.ScopeCode }
func (f limitForm) name() string      { return f.Name }

// Query policy limits from the data access object based on the provided parameters and options.
func (a *PolicyLimit) Query(ctx context.Context, params schema.PolicyLimitQueryParam) (*schema.PolicyLimitQueryResult, error) {
	params.Pagination = false

	return a.PolicyLimitDAL.Query(ctx, params, schema.PolicyLimitQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
}

// Get the specified policy limit from the data access object.
func (a *PolicyLimit) Get(ctx context.Context, id string) (*schema.PolicyLimitForm, error) {
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
	stored, err := a.PolicyLimitDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var form schema.PolicyLimitForm
	if err := stored.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy limit in the data access object.
func (a *PolicyLimit) Create(ctx context.Context, formItem *schema.PolicyLimitForm) (*schema.PolicyLimit, error) {
	life := a.lifecycle()
	prepared, err := life.prepareCreate(ctx, limitForm{formItem})
	if err != nil {
		return nil, err
	}
	item := &schema.PolicyLimit{
		ID:        prepared.ID,
		Deleted:   "0",
		Creator:   prepared.Creator,
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(item); err != nil {
		return nil, err
	}
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLimitDAL.Create(ctx, item)
	}); err != nil {
		return nil, err
	}
	life.afterCreate(ctx, recordOfLimit(item), item)
	return item, nil
}

// Update the specified policy limit in the data access object.
func (a *PolicyLimit) Update(ctx context.Context, id string, formItem *schema.PolicyLimitForm) error {
	life := a.lifecycle()
	before, err := life.beginUpdate(ctx, id, limitForm{formItem})
	if err != nil {
		return err
	}
	item, err := a.PolicyLimitDAL.Get(ctx, id)
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
		return a.PolicyLimitDAL.Update(ctx, item)
	}); err != nil {
		return err
	}
	life.afterUpdate(ctx, before, recordOfLimit(item), &beforeStored, item)
	return nil
}

// ToggleEnabled updates only the enabled status of a policy and re-syncs policy cache.
func (a *PolicyLimit) ToggleEnabled(ctx context.Context, id string, formItem *schema.PolicyEnabledForm) error {
	return a.lifecycle().toggle(ctx, id, formItem.Enabled)
}

// Delete the specified policy limit from the data access object.
func (a *PolicyLimit) Delete(ctx context.Context, id string) error {
	life := a.lifecycle()
	stored, err := a.PolicyLimitDAL.Get(ctx, id)
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
func (a *PolicyLimit) CopyTemplateToModel(ctx context.Context, templateID string, form *schema.PolicyCopyToModelForm) (*schema.PolicyLimit, error) {
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
	stored, err := a.PolicyLimitDAL.Get(ctx, templateID)
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
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLimitDAL.Create(ctx, &instance)
	}); err != nil {
		return nil, err
	}
	life.afterCopy(ctx, recordOfLimit(&instance), &instance)
	return &instance, nil
}

func recordOfLimit(item *schema.PolicyLimit) policyRecord {
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled,
	}
}

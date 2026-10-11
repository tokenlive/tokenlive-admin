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

// Tagging policy management
type PolicyTagging struct {
	Trans             *util.Trans
	PolicyTaggingDAL  *dal.PolicyTagging
	PolicyRedisSync   PolicyChangeSyncer
	ModelDAL          *resourceDal.Model
	DataPermissionDAL *resourceDal.DataPermission
	AuditLogBIZ       *opsBiz.AuditLog
}

func (a *PolicyTagging) lifecycle() policyLifecycle {
	return policyLifecycle{
		modelDAL:          a.ModelDAL,
		dataPermissionDAL: a.DataPermissionDAL,
		trans:             a.Trans,
		syncer:            a.PolicyRedisSync,
		audit:             a.AuditLogBIZ,
		store:             taggingStore{a.PolicyTaggingDAL},
		kind:              "tagging",
		blankActorIsSet:   true,
		notFound:          func() error { return errors.NotFound("", "Policy tagging not found") },
		duplicate:         func() error { return errors.BadRequest("", "Policy tagging with the same name already exists") },
	}
}

type taggingStore struct{ dal *dal.PolicyTagging }

func (s taggingStore) get(ctx context.Context, id string) (policyRecord, bool, error) {
	item, err := s.dal.Get(ctx, id)
	if err != nil || item == nil {
		return policyRecord{}, false, err
	}
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled, Creator: item.Creator, Modifier: item.Modifier,
	}, true, nil
}

func (s taggingStore) existsName(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error) {
	return s.dal.ExistsByName(ctx, scopeType, scopeCode, modelID, name)
}

func (s taggingStore) updateEnabled(ctx context.Context, id string, enabled int, modifier string) error {
	return s.dal.UpdateEnabled(ctx, id, enabled, modifier)
}

func (s taggingStore) delete(ctx context.Context, id string) error {
	return s.dal.Delete(ctx, id)
}

type taggingForm struct{ *schema.PolicyTaggingForm }

func (f taggingForm) modelID() string   { return f.ModelID }
func (f taggingForm) scopeType() string { return f.ScopeType }
func (f taggingForm) scopeCode() string { return f.ScopeCode }
func (f taggingForm) name() string      { return f.Name }

// Query policy taggings from the data access object based on the provided parameters and options.
func (a *PolicyTagging) Query(ctx context.Context, params schema.PolicyTaggingQueryParam) (*schema.PolicyTaggingQueryResult, error) {
	params.Pagination = false

	return a.PolicyTaggingDAL.Query(ctx, params, schema.PolicyTaggingQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
}

// Get the specified policy tagging from the data access object.
func (a *PolicyTagging) Get(ctx context.Context, id string) (*schema.PolicyTaggingForm, error) {
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
	stored, err := a.PolicyTaggingDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var form schema.PolicyTaggingForm
	if err := stored.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy tagging in the data access object.
func (a *PolicyTagging) Create(ctx context.Context, formItem *schema.PolicyTaggingForm) (*schema.PolicyTagging, error) {
	life := a.lifecycle()
	prepared, err := life.prepareCreate(ctx, taggingForm{formItem})
	if err != nil {
		return nil, err
	}
	item := &schema.PolicyTagging{
		ID:        prepared.ID,
		Deleted:   "0",
		Creator:   prepared.Creator,
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(item); err != nil {
		return nil, err
	}
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyTaggingDAL.Create(ctx, item)
	}); err != nil {
		return nil, err
	}
	life.afterCreate(ctx, recordOfTagging(item), item)
	return item, nil
}

// Update the specified policy tagging in the data access object.
func (a *PolicyTagging) Update(ctx context.Context, id string, formItem *schema.PolicyTaggingForm) error {
	life := a.lifecycle()
	before, err := life.beginUpdate(ctx, id, taggingForm{formItem})
	if err != nil {
		return err
	}
	item, err := a.PolicyTaggingDAL.Get(ctx, id)
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
		return a.PolicyTaggingDAL.Update(ctx, item)
	}); err != nil {
		return err
	}
	life.afterUpdate(ctx, before, recordOfTagging(item), &beforeStored, item)
	return nil
}

// ToggleEnabled updates only the enabled status of a policy and re-syncs policy cache.
func (a *PolicyTagging) ToggleEnabled(ctx context.Context, id string, formItem *schema.PolicyEnabledForm) error {
	return a.lifecycle().toggle(ctx, id, formItem.Enabled)
}

// Delete the specified policy tagging from the data access object.
func (a *PolicyTagging) Delete(ctx context.Context, id string) error {
	life := a.lifecycle()
	stored, err := a.PolicyTaggingDAL.Get(ctx, id)
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
func (a *PolicyTagging) CopyTemplateToModel(ctx context.Context, templateID string, form *schema.PolicyCopyToModelForm) (*schema.PolicyTagging, error) {
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
	stored, err := a.PolicyTaggingDAL.Get(ctx, templateID)
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
		return a.PolicyTaggingDAL.Create(ctx, &instance)
	}); err != nil {
		return nil, err
	}
	life.afterCopy(ctx, recordOfTagging(&instance), &instance)
	return &instance, nil
}

func recordOfTagging(item *schema.PolicyTagging) policyRecord {
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled,
	}
}

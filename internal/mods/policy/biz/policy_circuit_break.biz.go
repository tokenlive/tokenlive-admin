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

// CircuitBreak policy management
type PolicyCircuitBreak struct {
	Trans                 *util.Trans
	PolicyCircuitBreakDAL *dal.PolicyCircuitBreak
	PolicyRedisSync       PolicyChangeSyncer
	ModelDAL              *resourceDal.Model
	DataPermissionDAL     *resourceDal.DataPermission
	AuditLogBIZ           *opsBiz.AuditLog
}

func (a *PolicyCircuitBreak) lifecycle() policyLifecycle {
	return policyLifecycle{
		modelDAL:          a.ModelDAL,
		dataPermissionDAL: a.DataPermissionDAL,
		trans:             a.Trans,
		syncer:            a.PolicyRedisSync,
		audit:             a.AuditLogBIZ,
		store:             circuit_breakStore{a.PolicyCircuitBreakDAL},
		kind:              "circuit_break",
		notFound:          func() error { return errors.NotFound("", "Policy circuit break not found") },
		duplicate:         func() error { return errors.BadRequest("", "Policy circuit break with the same name already exists") },
	}
}

type circuit_breakStore struct{ dal *dal.PolicyCircuitBreak }

func (s circuit_breakStore) get(ctx context.Context, id string) (policyRecord, bool, error) {
	item, err := s.dal.Get(ctx, id)
	if err != nil || item == nil {
		return policyRecord{}, false, err
	}
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled, Creator: item.Creator, Modifier: item.Modifier,
	}, true, nil
}

func (s circuit_breakStore) existsName(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error) {
	return s.dal.ExistsByUniqueKey(ctx, scopeType, scopeCode, modelID, name)
}

func (s circuit_breakStore) updateEnabled(ctx context.Context, id string, enabled int, modifier string) error {
	return s.dal.UpdateEnabled(ctx, id, enabled, modifier)
}

func (s circuit_breakStore) delete(ctx context.Context, id string) error {
	return s.dal.Delete(ctx, id)
}

type circuit_breakForm struct{ *schema.PolicyCircuitBreakForm }

func (f circuit_breakForm) modelID() string   { return f.ModelID }
func (f circuit_breakForm) scopeType() string { return f.ScopeType }
func (f circuit_breakForm) scopeCode() string { return f.ScopeCode }
func (f circuit_breakForm) name() string      { return f.Name }

// Query policy circuit breaks from the data access object based on the provided parameters and options.
func (a *PolicyCircuitBreak) Query(ctx context.Context, params schema.PolicyCircuitBreakQueryParam) (*schema.PolicyCircuitBreakQueryResult, error) {
	params.Pagination = false

	return a.PolicyCircuitBreakDAL.Query(ctx, params, schema.PolicyCircuitBreakQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
}

// Get the specified policy circuit break from the data access object.
func (a *PolicyCircuitBreak) Get(ctx context.Context, id string) (*schema.PolicyCircuitBreakForm, error) {
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
	stored, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var form schema.PolicyCircuitBreakForm
	if err := stored.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy circuit break in the data access object.
func (a *PolicyCircuitBreak) Create(ctx context.Context, formItem *schema.PolicyCircuitBreakForm) (*schema.PolicyCircuitBreak, error) {
	life := a.lifecycle()
	prepared, err := life.prepareCreate(ctx, circuit_breakForm{formItem})
	if err != nil {
		return nil, err
	}
	item := &schema.PolicyCircuitBreak{
		ID:        prepared.ID,
		Deleted:   "0",
		Creator:   prepared.Creator,
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(item); err != nil {
		return nil, err
	}
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyCircuitBreakDAL.Create(ctx, item)
	}); err != nil {
		return nil, err
	}
	life.afterCreate(ctx, recordOfCircuitBreak(item), item)
	return item, nil
}

// Update the specified policy circuit break in the data access object.
func (a *PolicyCircuitBreak) Update(ctx context.Context, id string, formItem *schema.PolicyCircuitBreakForm) error {
	life := a.lifecycle()
	before, err := life.beginUpdate(ctx, id, circuit_breakForm{formItem})
	if err != nil {
		return err
	}
	item, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
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
		return a.PolicyCircuitBreakDAL.Update(ctx, item)
	}); err != nil {
		return err
	}
	life.afterUpdate(ctx, before, recordOfCircuitBreak(item), &beforeStored, item)
	return nil
}

// ToggleEnabled updates only the enabled status of a policy and re-syncs policy cache.
func (a *PolicyCircuitBreak) ToggleEnabled(ctx context.Context, id string, formItem *schema.PolicyEnabledForm) error {
	return a.lifecycle().toggle(ctx, id, formItem.Enabled)
}

// Delete the specified policy circuit break from the data access object.
func (a *PolicyCircuitBreak) Delete(ctx context.Context, id string) error {
	life := a.lifecycle()
	stored, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
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
func (a *PolicyCircuitBreak) CopyTemplateToModel(ctx context.Context, templateID string, form *schema.PolicyCopyToModelForm) (*schema.PolicyCircuitBreak, error) {
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
	stored, err := a.PolicyCircuitBreakDAL.Get(ctx, templateID)
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
		return a.PolicyCircuitBreakDAL.Create(ctx, &instance)
	}); err != nil {
		return nil, err
	}
	life.afterCopy(ctx, recordOfCircuitBreak(&instance), &instance)
	return &instance, nil
}

func recordOfCircuitBreak(item *schema.PolicyCircuitBreak) policyRecord {
	return policyRecord{
		ID: item.ID, ModelID: item.ModelID, ScopeType: item.ScopeType, ScopeCode: item.ScopeCode,
		Priority: item.Priority, Name: item.Name, Enabled: item.Enabled,
	}
}

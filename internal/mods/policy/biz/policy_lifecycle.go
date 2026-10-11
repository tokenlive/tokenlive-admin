package biz

import (
	"context"
	"time"

	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// policyRecord is the identity every policy kind shares. Kind-specific fields
// stay behind the kind's own store and form.
type policyRecord struct {
	ID        string
	ModelID   string
	ScopeType string
	ScopeCode string
	Priority  int
	Name      string
	Enabled   int
	Creator   *string
	Modifier  *string
}

type policyStore interface {
	get(ctx context.Context, id string) (policyRecord, bool, error)
	existsName(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error)
	updateEnabled(ctx context.Context, id string, enabled int, modifier string) error
	delete(ctx context.Context, id string) error
}

type policyForm interface {
	modelID() string
	scopeType() string
	scopeCode() string
	name() string
}

// policyLifecycle is the shared permission, uniqueness, sync and audit path.
// A kind supplies how its row is read and written. It does not decide those.
type policyLifecycle struct {
	modelDAL          *resourceDal.Model
	dataPermissionDAL *resourceDal.DataPermission
	trans             *util.Trans
	syncer            PolicyChangeSyncer
	audit             *opsBiz.AuditLog
	store             policyStore
	kind              string
	notFound          func() error
	duplicate         func() error
	// blankActorIsSet keeps the historical tagging/loadbalance behavior of
	// writing an empty creator. The other kinds omit the field when the
	// username is empty.
	blankActorIsSet bool
}

func (l policyLifecycle) require(ctx context.Context, modelID string, permission uint) error {
	_, err := requireExistingModelPolicy(ctx, l.modelDAL, l.dataPermissionDAL, modelID, permission)
	return err
}

func (l policyLifecycle) creator(ctx context.Context) *string {
	username := util.FromUsername(ctx)
	if username == "" && !l.blankActorIsSet {
		return nil
	}
	return &username
}

func (l policyLifecycle) modifier(ctx context.Context) (*string, bool) {
	username := util.FromUsername(ctx)
	if username == "" && !l.blankActorIsSet {
		return nil, false
	}
	return &username, true
}

func (l policyLifecycle) rejectDuplicate(ctx context.Context, form policyForm) error {
	exists, err := l.store.existsName(ctx, form.scopeType(), form.scopeCode(), form.modelID(), form.name())
	if err != nil {
		return err
	}
	if exists {
		return l.duplicate()
	}
	return nil
}

func (l policyLifecycle) prepareCreate(ctx context.Context, form policyForm) (policyRecord, error) {
	if err := l.require(ctx, form.modelID(), modelPermissionWrite); err != nil {
		return policyRecord{}, err
	}
	if err := l.rejectDuplicate(ctx, form); err != nil {
		return policyRecord{}, err
	}
	return policyRecord{
		ID:      util.NewXID(),
		Creator: l.creator(ctx),
	}, nil
}

func (l policyLifecycle) afterCreate(ctx context.Context, item policyRecord, stored interface{}) {
	_ = syncPolicyChangeAndLog(ctx, l.syncer, l.kind, "create", item.ScopeType, item.ScopeCode, item.ModelID)
	l.audit.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, item.ID, item.Name, nil, stored)
}

func (l policyLifecycle) beginUpdate(ctx context.Context, id string, form policyForm) (policyRecord, error) {
	item, ok, err := l.store.get(ctx, id)
	if err != nil {
		return policyRecord{}, err
	}
	if !ok {
		return policyRecord{}, l.notFound()
	}
	if err := l.require(ctx, item.ModelID, modelPermissionWrite); err != nil {
		return policyRecord{}, err
	}
	if err := rejectPolicyKindChange(item.ModelID, form.modelID()); err != nil {
		return policyRecord{}, err
	}
	if err := l.require(ctx, form.modelID(), modelPermissionWrite); err != nil {
		return policyRecord{}, err
	}
	if item.ModelID != form.modelID() || item.Name != form.name() {
		if err := l.rejectDuplicate(ctx, form); err != nil {
			return policyRecord{}, err
		}
	}
	return item, nil
}

func (l policyLifecycle) afterUpdate(ctx context.Context, before, item policyRecord, beforeStored, stored interface{}) {
	_ = syncPolicyChangeAndLog(ctx, l.syncer, l.kind, "update_old_dimension", before.ScopeType, before.ScopeCode, before.ModelID)
	if before.ScopeType != item.ScopeType || before.ScopeCode != item.ScopeCode || before.ModelID != item.ModelID {
		_ = syncPolicyChangeAndLog(ctx, l.syncer, l.kind, "update_new_dimension", item.ScopeType, item.ScopeCode, item.ModelID)
	}
	l.audit.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, item.ID, item.Name, beforeStored, stored)
}

func (l policyLifecycle) toggle(ctx context.Context, id string, enabled int) error {
	item, ok, err := l.store.get(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return l.notFound()
	}
	if err := l.require(ctx, item.ModelID, modelPermissionWrite); err != nil {
		return err
	}
	if item.Enabled == enabled {
		return nil
	}
	if err := l.trans.Exec(ctx, func(ctx context.Context) error {
		return l.store.updateEnabled(ctx, id, enabled, util.FromUsername(ctx))
	}); err != nil {
		return err
	}
	_ = syncPolicyChangeAndLog(ctx, l.syncer, l.kind, "toggle_enabled", item.ScopeType, item.ScopeCode, item.ModelID)
	l.audit.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, item.ID, item.Name, map[string]int{"enabled": item.Enabled}, map[string]int{"enabled": enabled})
	return nil
}

func (l policyLifecycle) remove(ctx context.Context, id string) (policyRecord, error) {
	item, ok, err := l.store.get(ctx, id)
	if err != nil {
		return policyRecord{}, err
	}
	if !ok {
		return policyRecord{}, l.notFound()
	}
	if err := l.require(ctx, item.ModelID, modelPermissionWrite); err != nil {
		return policyRecord{}, err
	}
	if err := l.trans.Exec(ctx, func(ctx context.Context) error {
		return l.store.delete(ctx, id)
	}); err != nil {
		return policyRecord{}, err
	}
	_ = syncPolicyChangeAndLog(ctx, l.syncer, l.kind, "delete", item.ScopeType, item.ScopeCode, item.ModelID)
	return item, nil
}

func (l policyLifecycle) afterDelete(ctx context.Context, item policyRecord, stored interface{}) {
	l.audit.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, item.ID, item.Name, stored, nil)
}

type policyCopy struct {
	ID        string
	ModelID   string
	Name      string
	ScopeType string
	ScopeCode string
	Priority  int
	Creator   *string
	CreatedAt time.Time
}

func (l policyLifecycle) prepareCopy(ctx context.Context, template policyRecord, form *schema.PolicyCopyToModelForm) (policyCopy, error) {
	if template.ModelID != "" {
		return policyCopy{}, errors.BadRequest("", "Only policy templates can be copied to a model")
	}
	if _, err := requireModelPermission(ctx, l.modelDAL, l.dataPermissionDAL, form.ModelID, modelPermissionWrite); err != nil {
		return policyCopy{}, err
	}
	name := form.Name
	if name == "" {
		name = template.Name
	}
	name, err := nextPolicyName(ctx, name, form.ModelID, func(ctx context.Context, modelID, name string) (bool, error) {
		return l.store.existsName(ctx, "global", "", modelID, name)
	})
	if err != nil {
		return policyCopy{}, err
	}
	copied := policyCopy{
		ID:        util.NewXID(),
		ModelID:   form.ModelID,
		Name:      name,
		ScopeType: template.ScopeType,
		ScopeCode: template.ScopeCode,
		Priority:  template.Priority,
		Creator:   l.creator(ctx),
		CreatedAt: time.Now(),
	}
	if form.ScopeType != nil {
		copied.ScopeType = *form.ScopeType
	}
	if form.ScopeCode != nil {
		copied.ScopeCode = *form.ScopeCode
	}
	if form.Priority != nil {
		copied.Priority = *form.Priority
	}
	return copied, nil
}

func (l policyLifecycle) afterCopy(ctx context.Context, item policyRecord, stored interface{}) {
	_ = syncPolicyChangeAndLog(ctx, l.syncer, l.kind, "copy_template_to_model", item.ScopeType, item.ScopeCode, item.ModelID)
	l.audit.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, item.ID, item.Name, nil, stored)
}

package biz

import (
	"context"

	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DialectPublisher interface {
	PublishModelDialects(ctx context.Context, modelID string) error
}

type PolicyDialect struct {
	Trans             *util.Trans
	ModelDAL          *resourceDal.Model
	DataPermissionDAL *resourceDal.DataPermission
	Publisher         DialectPublisher
	AuditLogBIZ       *opsBiz.AuditLog
}

func (a *PolicyDialect) active(ctx context.Context) *gorm.DB {
	return util.GetDB(ctx, a.Trans.DB).Model(&schema.PolicyDialect{}).Where("deleted = '0'")
}

func (a *PolicyDialect) requireOrdinaryModel(ctx context.Context, id string, permission uint) error {
	model, err := requireModelPermission(ctx, a.ModelDAL, a.DataPermissionDAL, id, permission)
	if err != nil {
		return err
	}
	if model.ModelType == resourceSchema.ModelTypeSmart {
		return errors.BadRequest("", "方言适配仅适用于普通模型")
	}
	return nil
}

func (a *PolicyDialect) Query(ctx context.Context, query schema.PolicyDialectQuery) ([]schema.PolicyDialect, *util.PaginationResult, error) {
	if err := a.requireOrdinaryModel(ctx, query.ModelID, modelPermissionRead); err != nil {
		return nil, nil, err
	}
	query.Pagination = true
	items := make([]schema.PolicyDialect, 0)
	page, err := util.WrapPageQuery(ctx, a.active(ctx).Where("model_id = ?", query.ModelID).Order("implementation, id"),
		query.PaginationParam, util.QueryOptions{}, &items)
	return items, page, err
}

type PolicyDialectCreateResult struct {
	*schema.PolicyDialect
	*resourceSchema.ModelMutationResult
}

func (a *PolicyDialect) Create(ctx context.Context, form schema.PolicyDialectForm) (*PolicyDialectCreateResult, error) {
	switch form.Implementation {
	case schema.DialectDeveloperToSystem, schema.DialectStoreUnsupported, schema.DialectGeminiStripThinkingExtensions:
	default:
		return nil, errors.BadRequest("", "未知的方言适配实现")
	}
	item := &schema.PolicyDialect{
		ID: util.NewXID(), ModelID: form.ModelID, Name: form.Implementation, Implementation: form.Implementation,
		Enabled: form.Enabled,
	}
	if username := util.FromUsername(ctx); username != "" {
		item.Creator = &username
		item.Modifier = &username
	}
	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.requireOrdinaryModel(util.NewRowLock(ctx), item.ModelID, modelPermissionWrite); err != nil {
			return err
		}
		exists, err := util.Exists(ctx, a.active(ctx).Where("model_id = ? AND implementation = ?", item.ModelID, item.Implementation))
		if err != nil {
			return err
		}
		if exists {
			return errors.Conflict("", "该模型已配置此方言适配实现")
		}
		return a.active(ctx).Create(item).Error
	}); err != nil {
		return nil, err
	}
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, item.ID, item.Name, nil, item)
	result := &PolicyDialectCreateResult{PolicyDialect: item}
	if item.Enabled == 1 {
		result.ModelMutationResult = a.publish(ctx, item.ModelID)
	}
	return result, nil
}

func (a *PolicyDialect) ToggleEnabled(ctx context.Context, id string, enabled int) (*resourceSchema.ModelMutationResult, error) {
	action := opsSchema.AuditActionDisable
	if enabled == 1 {
		action = opsSchema.AuditActionEnable
	}
	return a.change(ctx, id, action, &enabled)
}

func (a *PolicyDialect) Delete(ctx context.Context, id string) (*resourceSchema.ModelMutationResult, error) {
	return a.change(ctx, id, opsSchema.AuditActionDelete, nil)
}

func (a *PolicyDialect) get(ctx context.Context, id string) (*schema.PolicyDialect, error) {
	var item schema.PolicyDialect
	found, err := util.FindOne(ctx, a.active(ctx).Where("id = ?", id), util.QueryOptions{}, &item)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.NotFound("", "方言适配策略不存在")
	}
	return &item, nil
}

func (a *PolicyDialect) change(ctx context.Context, id, action string, enabled *int) (*resourceSchema.ModelMutationResult, error) {
	item, err := a.get(ctx, id)
	if err != nil {
		return nil, err
	}
	var before, after *schema.PolicyDialect
	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := a.requireOrdinaryModel(util.NewRowLock(ctx), item.ModelID, modelPermissionWrite); err != nil {
			return err
		}
		var err error
		before, err = a.get(ctx, id)
		if err != nil {
			return err
		}
		if enabled == nil {
			return util.SoftDelete(ctx, a.active(ctx), id)
		}
		updates := map[string]interface{}{"enabled": *enabled}
		if username := util.FromUsername(ctx); username != "" {
			updates["modifier"] = username
		}
		if before.Enabled != *enabled {
			updates["version"] = gorm.Expr("version + 1")
		}
		if err := a.active(ctx).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		after, err = a.get(ctx, id)
		return err
	})
	if err != nil {
		return nil, err
	}

	a.AuditLogBIZ.RecordAction(ctx, action, opsSchema.AuditResourceTypePolicy, id, before.Name, before, after)
	return a.publish(ctx, item.ModelID), nil
}

func (a *PolicyDialect) publish(ctx context.Context, modelID string) *resourceSchema.ModelMutationResult {
	result := &resourceSchema.ModelMutationResult{Saved: true, SyncStatus: "synced"}
	if err := a.Publisher.PublishModelDialects(ctx, modelID); err != nil {
		logging.Context(ctx).Warn("failed to publish model dialects", zap.String("model_id", modelID), zap.Error(err))
		result.SyncStatus = "failed"
		result.Warnings = []string{"方言适配变更已保存，但配置同步失败，请重试模型同步"}
	}
	return result
}

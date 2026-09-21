package biz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func smartReferences(ctx context.Context, db *gorm.DB, id string) ([]*schema.Model, error) {
	var models []*schema.Model
	if err := dal.GetModelDB(ctx, db).Where("model_type = ?", schema.ModelTypeSmart).Find(&models).Error; err != nil {
		return nil, err
	}
	var refs []*schema.Model
	for _, model := range models {
		for _, refID := range model.SmartRouting.ReferenceIDs() {
			if refID == id {
				refs = append(refs, model)
				break
			}
		}
	}
	return refs, nil
}

func (m *Model) validateSmartModel(ctx context.Context, model, before *schema.Model) error {
	if err := m.checkSmartModel(ctx, model); err != nil {
		return err
	}
	if model.ModelType != schema.ModelTypeSmart || model.SmartRouting == nil {
		return nil
	}
	if before != nil && before.ModelType == schema.ModelTypeSmart {
		if model.SmartRouting.Version != smartRoutingVersion(before.SmartRouting) {
			return errors.Conflict("", "智能路由配置已更新，请刷新后重试")
		}
		model.SmartRouting.Version++
	} else {
		model.SmartRouting.Version = 1
	}
	return nil
}

func smartRoutingVersion(routing *schema.SmartRouting) int64 {
	if routing == nil {
		return 0
	}
	return routing.Version
}

func (m *Model) checkSmartModel(ctx context.Context, model *schema.Model) error {
	if model.ModelType != schema.ModelTypeSmart {
		return nil
	}
	if model.SmartRouting == nil {
		if model.Enabled != 0 {
			return errors.BadRequest("", "请先在模型详情的难度路由区间中保存完整配置，再启用智能模型")
		}
	} else if err := model.SmartRouting.Validate(); err != nil {
		return err
	}
	db := util.GetDB(ctx, m.ModelDAL.DB)
	var endpointCount int64
	if err := db.Model(&schema.Endpoint{}).Where("model_id = ? AND deleted = '0'", model.ID).Count(&endpointCount).Error; err != nil {
		return err
	}
	if endpointCount > 0 {
		return errors.BadRequest("", "存在直属端点的模型不能转换为智能模型")
	}
	refs, err := smartReferences(ctx, db, model.ID)
	if err != nil {
		return err
	}
	if len(refs) != 0 {
		return errors.BadRequest("", "模型已被智能模型引用，不能转换为智能模型")
	}
	for _, id := range model.SmartRouting.ReferenceIDs() {
		if id == model.ID {
			return errors.BadRequest("", "智能模型不能引用自身")
		}
		var ref schema.Model
		err := dal.GetModelDB(ctx, db).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&ref).Error
		if err == gorm.ErrRecordNotFound {
			return errors.BadRequest("", "引用模型不存在或无权使用")
		}
		if err != nil {
			return err
		}
		if !util.FromIsRootUser(ctx) {
			if m.DataPermissionBIZ == nil {
				return errors.BadRequest("", "引用模型不存在或无权使用")
			}
			ok, err := m.DataPermissionBIZ.HasReadPermission(ctx, schema.DataPermissionTypeModel, id)
			if err != nil {
				return err
			}
			if !ok {
				return errors.BadRequest("", "引用模型不存在或无权使用")
			}
		}
		if ref.SpaceCode != model.SpaceCode || ref.Enabled != 1 || ref.ModelType == schema.ModelTypeSmart || !supportsChat(ref.RequestTypes) {
			return errors.BadRequest("", "引用模型必须为同空间、已启用且支持 Chat Completions 的普通模型")
		}
	}
	return nil
}

func (m *Model) requireModelMutationPermission(ctx context.Context, model *schema.Model, bit int) error {
	if util.FromIsRootUser(ctx) {
		return nil
	}
	ok, err := util.Exists(ctx, dal.GetDataPermissionDB(ctx, m.ModelDAL.DB).Where(
		"type = ? AND data_id = ? AND user = ? AND tenant = ? AND permission & ? = ?",
		schema.DataPermissionTypeModel, model.ID, util.FromUsername(ctx), util.FromTenant(ctx), bit, bit,
	))
	if err != nil {
		return err
	}
	if !ok {
		return errors.NotFound("", "Model not found")
	}
	return nil
}

type SavedModelSyncError struct {
	Cause error
}

func (e *SavedModelSyncError) Error() string {
	return "模型已删除，但配置同步失败，请通过全量同步重试"
}

func (e *SavedModelSyncError) Unwrap() error { return e.Cause }

func supportsChat(raw string) bool {
	var types []string
	if json.Unmarshal([]byte(raw), &types) != nil {
		return false
	}
	for _, typ := range types {
		if typ == "chat_completion" {
			return true
		}
	}
	return false
}

func (m *Model) updateWithSmartVersion(ctx context.Context, model, before *schema.Model) error {
	db := m.modelWithSmartVersion(ctx, model, before)
	result := db.Select("*").Omit("created_at", "model_code").Updates(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.Conflict("", "智能路由配置已更新，请刷新后重试")
	}
	return nil
}

func (m *Model) modelWithSmartVersion(ctx context.Context, model, before *schema.Model) *gorm.DB {
	db := dal.GetModelDB(ctx, m.ModelDAL.DB).Where("id = ?", model.ID)
	if before.ModelType == schema.ModelTypeSmart {
		db = db.Where("model_type = ?", schema.ModelTypeSmart)
		if before.SmartRouting == nil {
			db = db.Where("(smart_routing IS NULL OR smart_routing = 'null')")
		} else {
			db = db.Where("JSON_EXTRACT(smart_routing, '$.version') = ?", before.SmartRouting.Version)
		}
	} else if model.ModelType == schema.ModelTypeSmart {
		db = db.Where("model_type = ?", schema.ModelTypeNormal)
	}
	return db
}

func (m *Model) fillModelReferences(ctx context.Context, model *schema.Model) error {
	refs, err := smartReferences(ctx, m.ModelDAL.DB, model.ID)
	if err != nil {
		return err
	}
	model.ReferencedBy = nil
	for _, ref := range refs {
		if !util.FromIsRootUser(ctx) {
			if m.DataPermissionBIZ == nil {
				continue
			}
			ok, err := m.DataPermissionBIZ.HasReadPermission(ctx, schema.DataPermissionTypeModel, ref.ID)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		model.ReferencedBy = append(model.ReferencedBy, schema.ModelReference{ID: ref.ID, ModelCode: ref.ModelCode, ModelName: ref.ModelName})
	}
	return util.GetDB(ctx, m.ModelDAL.DB).Model(&schema.Endpoint{}).Where("model_id = ? AND deleted = '0'", model.ID).Count(&model.EndpointCount).Error
}

func (m *Model) mutationResult(ctx context.Context, model *schema.Model, syncErr error) *schema.ModelMutationResult {
	result := &schema.ModelMutationResult{Saved: true, SyncStatus: "synced"}
	if syncErr != nil {
		logging.Context(ctx).Warn("Model saved but configuration publication failed", zap.String("model_id", model.ID), zap.Error(syncErr))
		result.SyncStatus = "failed"
		result.Warnings = append(result.Warnings, "保存成功，但配置同步失败，请使用同步按钮重试")
	}
	if model.Enabled == 0 {
		if err := m.fillModelReferences(ctx, model); err != nil {
			result.Warnings = append(result.Warnings, "模型已停用，但无法读取智能模型引用影响")
		} else if len(model.ReferencedBy) > 0 {
			result.ReferencedBy = model.ReferencedBy
			result.Warnings = append(result.Warnings, fmt.Sprintf("停用将影响 %d 个有权查看的智能模型，网关会将此依赖视为不可用", len(model.ReferencedBy)))
		} else if refs, err := smartReferences(ctx, m.ModelDAL.DB, model.ID); err == nil && len(refs) > 0 {
			result.Warnings = append(result.Warnings, "停用将影响引用此模型的智能模型")
		}
	}
	return result
}

func (m *Model) publishModelChange(ctx context.Context, model, before *schema.Model, affected []*schema.Model) error {
	return m.reconcileModelPublication(ctx, model.ID)
}

func rejectSmartEndpoint(ctx context.Context, modelDAL *dal.Model, modelID string) error {
	if modelDAL == nil {
		return nil
	}
	var model schema.Model
	err := dal.GetModelDB(ctx, modelDAL.DB).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", modelID).First(&model).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if model.ModelType == schema.ModelTypeSmart {
		return errors.BadRequest("", "智能模型不能绑定直属端点")
	}
	return nil
}

func (m *Model) bumpSmartReferences(ctx context.Context, model, before *schema.Model) ([]*schema.Model, error) {
	refs, err := smartReferences(ctx, m.ModelDAL.DB, model.ID)
	if err != nil {
		return nil, err
	}
	if len(refs) > 0 && (model.SpaceCode != before.SpaceCode || !supportsChat(model.RequestTypes)) {
		return nil, errors.BadRequest("", "模型已被智能模型引用，不能修改空间或移除 Chat Completions")
	}
	return nil, nil
}

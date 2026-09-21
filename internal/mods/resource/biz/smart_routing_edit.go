package biz

import (
	"context"
	"encoding/json"
	"time"

	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpdateSmartRouting updates only the detail tab's configuration. Saving a
// complete draft does not enable the model or overwrite its basic information.
func (m *Model) UpdateSmartRouting(ctx context.Context, id string, routing *schema.SmartRouting) (*schema.ModelMutationResult, error) {
	if routing == nil {
		return nil, errors.BadRequest("", "请填写完整的裁决和难度路由区间配置")
	}
	var model, before schema.Model
	err := m.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := dal.GetModelDB(ctx, m.ModelDAL.DB).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).First(&model).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NotFound("", "Model not found")
			}
			return err
		}
		if err := m.requireModelMutationPermission(ctx, &model, 2); err != nil {
			return err
		}
		if model.ModelType != schema.ModelTypeSmart {
			return errors.BadRequest("", "只有智能模型可以配置难度路由区间")
		}
		before = model
		model.SmartRouting = routing.Clone()
		if err := m.validateSmartModel(ctx, &model, &before); err != nil {
			return err
		}
		raw, err := json.Marshal(model.SmartRouting)
		if err != nil {
			return err
		}
		model.Modifier = util.FromUsername(ctx)
		model.UpdatedAt = time.Now()
		result := m.modelWithSmartVersion(ctx, &model, &before).Updates(map[string]any{
			"smart_routing": string(raw),
			"modifier":      model.Modifier,
			"updated_at":    model.UpdatedAt,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.Conflict("", "智能路由配置已更新，请刷新后重试")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := m.mutationResult(ctx, &model, m.publishModelChange(ctx, &model, &before, nil))
	if m.AuditLogBIZ != nil {
		m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeModel,
			model.ID, model.ModelName, before.SmartRouting, model.SmartRouting)
	}
	return result, nil
}

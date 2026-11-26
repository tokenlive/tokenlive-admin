package biz

import (
	"context"

	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func enabledModelDialects(db *gorm.DB, modelIDs []string) (map[string][]string, error) {
	result := make(map[string][]string)
	if len(modelIDs) == 0 {
		return result, nil
	}
	var policies []policySchema.PolicyDialect
	err := db.Select("model_id, implementation").
		Where("model_id IN ? AND enabled = 1 AND deleted = '0'", modelIDs).
		Where("scope_type = 'global' AND scope_code = ''").
		Order("implementation").Find(&policies).Error
	if err != nil {
		return nil, err
	}
	for _, policy := range policies {
		result[policy.ModelID] = append(result[policy.ModelID], policy.Implementation)
	}
	return result, nil
}

// PublishModelDialects reads the current committed model and policy state. The
// model row lock serializes Redis writes with other updates and model deletion.
func (s *ConfigRedisSync) PublishModelDialects(ctx context.Context, modelID string) error {
	ClearGatewayConfigCache()
	var model schema.Model
	if s.RedisClient == nil {
		util.NotifyConfigChanged(ctx, util.ConfigChangeEndpoints)
		return nil
	}
	return (&util.Trans{DB: s.ModelDAL.DB}).Exec(ctx, func(ctx context.Context) error {
		err := util.GetDB(ctx, s.ModelDAL.DB).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted = '0'", modelID).First(&model).Error
		if err == gorm.ErrRecordNotFound {
			// The model deletion path publishes its own invalidation.
			return nil
		}
		if err != nil {
			return err
		}
		return s.SyncModelByCode(ctx, model.ModelCode)
	})
}

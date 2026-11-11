package biz

import (
	"context"
	"fmt"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// RuntimeSmartRouting is shared by HTTP, embedded snapshots and Redis.
type RuntimeSmartRouting = gatewaycontract.RuntimeSmartRouting

type RuntimeSmartRange = gatewaycontract.RuntimeSmartRange

func resolveSmartRuntime(ctx context.Context, db *gorm.DB, model *schema.Model) (*RuntimeSmartRouting, error) {
	if model.SmartRouting == nil || model.SmartRouting.Version < 1 {
		return nil, fmt.Errorf("model %s has no versioned smart routing", model.ModelCode)
	}
	routing := *model.SmartRouting
	if routing.RoutingMode() != compositerouting.Difficulty && !config.C.Gateway.CompositeRoutingV2 {
		return nil, fmt.Errorf("Gateway format 2 rollout is not enabled")
	}
	if err := routing.Validate(); err != nil {
		return nil, err
	}
	var refs []schema.Model
	if err := dal.GetModelDB(ctx, db).Where("id IN ?", routing.ReferenceIDs()).Find(&refs).Error; err != nil {
		return nil, err
	}
	if err := validateCompositePromise(model, refs); err != nil {
		return nil, err
	}
	codes := make(map[string]string)
	for _, ref := range refs {
		// Disabled references remain in the definition so Gateway can upgrade.
		if ref.ModelType == schema.ModelTypeSmart || ref.SpaceCode != model.SpaceCode || !supportsChat(ref.RequestTypes) {
			return nil, fmt.Errorf("model %s has an invalid smart routing dependency", model.ModelCode)
		}
		codes[ref.ID] = ref.ModelCode
	}
	for _, id := range routing.ReferenceIDs() {
		if codes[id] == "" || id == model.ID {
			return nil, fmt.Errorf("model %s has an unresolved smart routing dependency", model.ModelCode)
		}
	}
	result := &RuntimeSmartRouting{
		SchemaVersion: routing.SchemaVersion, Mode: routing.Mode,
		PrimaryModel: codes[routing.PrimaryModelID], DefaultModel: codes[routing.DefaultModelID],
		Version: routing.Version, JudgeModel: codes[routing.JudgeModelID],
		JudgeTimeoutMS: routing.JudgeTimeoutMS, JudgeMaxInputBytes: routing.JudgeMaxInputBytes,
		JudgeMaxOutputTokens: routing.JudgeMaxOutputTokens,
	}
	for _, r := range routing.Rules {
		result.Rules = append(result.Rules, gatewaycontract.RuntimeSmartRule{ID: r.ID, Name: r.Name, Order: r.Order, Enabled: r.Enabled, RelationType: r.RelationType, Conditions: r.Conditions, TargetModel: codes[r.TargetModelID]})
	}
	for _, r := range routing.Ranges {
		result.Ranges = append(result.Ranges, RuntimeSmartRange{Min: r.Min, Max: r.Max, Model: codes[r.ModelID]})
	}
	return result, nil
}

func (s *GatewaySync) smartModelConfigs(ctx context.Context, modelCode string) (map[string]ModelConfig, []string, error) {
	db := dal.GetModelDB(ctx, s.DB).Where("model_type = ? AND enabled = 1", schema.ModelTypeSmart)
	if modelCode != "" {
		db = db.Where("model_code = ?", modelCode)
	}
	var models []schema.Model
	if err := db.Find(&models).Error; err != nil {
		return nil, nil, err
	}
	result := make(map[string]ModelConfig)
	codes := []string{modelCode}
	for _, model := range models {
		runtime, err := resolveSmartRuntime(ctx, util.GetDB(ctx, s.DB), &model)
		if err != nil {
			return nil, nil, err
		}
		result[model.ModelCode] = ModelConfig{
			ModelType: schema.ModelTypeSmart, SmartRouting: runtime,
			RequestTypes: []string{"chat_completion"}, Endpoints: []EndpointConfig{},
		}
		codes = append(codes, runtime.ReferenceCodes()...)
		if model.SmartRouting.RoutingMode() != compositerouting.Difficulty {
			backups, err := compositeFallbackCodes(ctx, s.DB, model.ID, true)
			if err != nil {
				return nil, nil, err
			}
			codes = append(codes, backups...)
		}
	}
	return result, codes, nil
}

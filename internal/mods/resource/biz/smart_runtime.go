package biz

import (
	"context"
	"fmt"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// RuntimeSmartRouting is shared by HTTP, embedded snapshots and Redis.
type RuntimeSmartRouting struct {
	Version              int64               `json:"version"`
	JudgeModel           string              `json:"judge_model"`
	JudgeTimeoutMS       int                 `json:"judge_timeout_ms"`
	JudgeMaxInputBytes   int                 `json:"judge_max_input_bytes"`
	JudgeMaxOutputTokens int                 `json:"judge_max_output_tokens"`
	Ranges               []RuntimeSmartRange `json:"ranges"`
}

type RuntimeSmartRange struct {
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Model string `json:"model"`
}

func resolveSmartRuntime(ctx context.Context, db *gorm.DB, model *schema.Model) (*RuntimeSmartRouting, error) {
	if model.SmartRouting == nil || model.SmartRouting.Version < 1 {
		return nil, fmt.Errorf("model %s has no versioned smart routing", model.ModelCode)
	}
	routing := *model.SmartRouting
	if err := routing.Validate(); err != nil {
		return nil, err
	}
	var refs []schema.Model
	if err := dal.GetModelDB(ctx, db).Where("id IN ?", routing.ReferenceIDs()).Find(&refs).Error; err != nil {
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
		Version: routing.Version, JudgeModel: codes[routing.JudgeModelID],
		JudgeTimeoutMS: routing.JudgeTimeoutMS, JudgeMaxInputBytes: routing.JudgeMaxInputBytes,
		JudgeMaxOutputTokens: routing.JudgeMaxOutputTokens,
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
		codes = append(codes, runtime.JudgeModel)
		for _, r := range runtime.Ranges {
			codes = append(codes, r.Model)
		}
	}
	return result, codes, nil
}

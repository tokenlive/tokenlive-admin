package schema

import (
	"encoding/json"

	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

const (
	ModelTypeNormal = "normal"
	ModelTypeSmart  = "smart"
)

type ModelReference struct {
	ID        string `json:"id"`
	ModelCode string `json:"model_code"`
	ModelName string `json:"model_name"`
}

type ModelMutationResult struct {
	Saved        bool             `json:"saved"`
	SyncStatus   string           `json:"sync_status"`
	Warnings     []string         `json:"warnings,omitempty"`
	ReferencedBy []ModelReference `json:"referenced_by,omitempty"`
}

// SmartRouting keeps stable database identities; publication resolves codes.
type SmartRouting struct {
	Version              int64        `json:"version"`
	JudgeModelID         string       `json:"judge_model_id"`
	JudgeTimeoutMS       int          `json:"judge_timeout_ms"`
	JudgeMaxInputBytes   int          `json:"judge_max_input_bytes"`
	JudgeMaxOutputTokens int          `json:"judge_max_output_tokens"`
	Ranges               []SmartRange `json:"ranges"`
}

// SmartRange is left-inclusive and right-exclusive, except the final 100.
type SmartRange struct {
	Min     int    `json:"min"`
	Max     int    `json:"max"`
	ModelID string `json:"model_id"`
}

func (m *ModelForm) validateSmartRouting() error {
	if m.ModelType == "" {
		m.ModelType = ModelTypeNormal
	}
	switch m.ModelType {
	case ModelTypeNormal:
		if m.SmartRouting != nil {
			return errors.BadRequest("", "普通模型不能包含智能路由配置")
		}
		return nil
	case ModelTypeSmart:
	default:
		return errors.BadRequest("", "model_type 必须为 normal 或 smart")
	}
	var requestTypes []string
	if err := json.Unmarshal([]byte(m.RequestTypes), &requestTypes); err != nil || len(requestTypes) != 1 || requestTypes[0] != "chat_completion" {
		return errors.BadRequest("", "智能模型仅支持 Chat Completions")
	}
	if m.SmartRouting == nil && m.Enabled == 0 {
		return nil // A new smart model remains disabled until configured in its detail tab.
	}
	return m.SmartRouting.Validate()
}

func (s *SmartRouting) Clone() *SmartRouting {
	if s == nil {
		return nil
	}
	result := *s
	result.Ranges = append([]SmartRange(nil), s.Ranges...)
	return &result
}

// Readiness is structural; enabling additionally revalidates references in a transaction.
func (m *Model) IsSmartRoutingReady() bool {
	return m.ModelType == ModelTypeSmart && m.SmartRouting != nil &&
		m.SmartRouting.Version > 0 && m.SmartRouting.Clone().Validate() == nil
}

func (s *SmartRouting) Validate() error {
	if s == nil || s.JudgeModelID == "" {
		return errors.BadRequest("", "智能路由必须配置裁决模型")
	}
	if s.Version < 0 || s.JudgeTimeoutMS < 0 || s.JudgeMaxInputBytes < 0 || s.JudgeMaxOutputTokens < 0 {
		return errors.BadRequest("", "智能路由版本和裁决限制不能为负数")
	}
	if s.JudgeTimeoutMS == 0 {
		s.JudgeTimeoutMS = 5000
	}
	if s.JudgeMaxInputBytes == 0 {
		s.JudgeMaxInputBytes = 65536
	}
	if s.JudgeMaxOutputTokens == 0 {
		s.JudgeMaxOutputTokens = 256
	}
	next := 0
	targets := make(map[string]bool)
	for _, r := range s.Ranges {
		if r.Min != next || r.Max <= r.Min || r.Max > 100 || r.ModelID == "" {
			return errors.BadRequest("", "智能路由区间必须按升序完整、连续且无重叠地覆盖 0–100")
		}
		next = r.Max
		targets[r.ModelID] = true
	}
	if next != 100 || len(targets) < 2 {
		return errors.BadRequest("", "智能路由必须覆盖 0–100 并引用至少两个不同的普通子模型")
	}
	return nil
}

func (s *SmartRouting) ReferenceIDs() []string {
	if s == nil {
		return nil
	}
	ids := []string{s.JudgeModelID}
	seen := map[string]bool{s.JudgeModelID: true}
	for _, r := range s.Ranges {
		if !seen[r.ModelID] {
			ids = append(ids, r.ModelID)
			seen[r.ModelID] = true
		}
	}
	return ids
}

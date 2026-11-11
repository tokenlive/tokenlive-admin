package schema

import (
	"bytes"
	"encoding/json"

	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"

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
	SchemaVersion        int          `json:"schema_version,omitempty"`
	Mode                 string       `json:"mode,omitempty"`
	PrimaryModelID       string       `json:"primary_model_id,omitempty"`
	DefaultModelID       string       `json:"default_model_id,omitempty"`
	Rules                []SmartRule  `json:"rules,omitempty"`
	Version              int64        `json:"version"`
	JudgeModelID         string       `json:"judge_model_id"`
	JudgeTimeoutMS       int          `json:"judge_timeout_ms"`
	JudgeMaxInputBytes   int          `json:"judge_max_input_bytes"`
	JudgeMaxOutputTokens int          `json:"judge_max_output_tokens"`
	Ranges               []SmartRange `json:"ranges"`
}

// UnmarshalJSON rejects typos instead of silently ignoring unsupported routing fields.
func (s *SmartRouting) UnmarshalJSON(data []byte) error {
	type plain SmartRouting
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*s = SmartRouting(value)
	return nil
}

type SmartRule struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name"`
	Order         int                          `json:"order"`
	Enabled       bool                         `json:"enabled"`
	RelationType  string                       `json:"relation_type"`
	Conditions    []compositerouting.Condition `json:"conditions"`
	TargetModelID string                       `json:"target_model_id"`
}

// RoutingMode treats documents without a discriminator as legacy difficulty routing.
func (s *SmartRouting) RoutingMode() string {
	if s == nil || s.Mode == "" {
		return compositerouting.Difficulty
	}
	return s.Mode
}

func (s *SmartRouting) PlanConfig() compositerouting.Config {
	c := compositerouting.Config{Mode: s.RoutingMode(), Primary: s.PrimaryModelID, Default: s.DefaultModelID}
	for _, r := range s.Rules {
		c.Rules = append(c.Rules, compositerouting.Rule{ID: r.ID, Name: r.Name, Order: r.Order, Enabled: r.Enabled, RelationType: r.RelationType, Conditions: r.Conditions, Target: r.TargetModelID})
	}
	return c
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
	result.Rules = append([]SmartRule(nil), s.Rules...)
	for i := range result.Rules {
		result.Rules[i].Conditions = append([]compositerouting.Condition(nil), s.Rules[i].Conditions...)
		for j := range result.Rules[i].Conditions {
			result.Rules[i].Conditions[j].Values = append([]string(nil), s.Rules[i].Conditions[j].Values...)
		}
	}
	return &result
}

// Readiness is structural; enabling additionally revalidates references in a transaction.
func (m *Model) IsSmartRoutingReady() bool {
	return m.ModelType == ModelTypeSmart && m.SmartRouting != nil &&
		m.SmartRouting.Version > 0 && m.SmartRouting.Clone().Validate() == nil
}

func (s *SmartRouting) Validate() error {
	if s == nil {
		return errors.BadRequest("", "请配置复合模型选模方式")
	}
	if s.Version < 0 {
		return errors.BadRequest("", "配置版本不能为负数")
	}
	if s.RoutingMode() != compositerouting.Difficulty {
		if s.SchemaVersion != 2 {
			return errors.BadRequest("", "新选模模式需要 schema_version=2")
		}
		if s.JudgeModelID != "" || len(s.Ranges) > 0 || s.JudgeTimeoutMS != 0 || s.JudgeMaxInputBytes != 0 || s.JudgeMaxOutputTokens != 0 {
			return errors.BadRequest("", "主备/规则模式不能包含裁决配置")
		}
		if err := s.PlanConfig().Validate(); err != nil {
			return errors.BadRequest("", "%s", err.Error())
		}
		return nil
	}
	if s.SchemaVersion != 0 && s.SchemaVersion != 1 || s.PrimaryModelID != "" || s.DefaultModelID != "" || len(s.Rules) != 0 {
		return errors.BadRequest("", "难度模式格式版本或字段不合法")
	}
	if s.JudgeModelID == "" {
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
	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if id != "" && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	add(s.JudgeModelID)
	add(s.PrimaryModelID)
	add(s.DefaultModelID)
	for _, r := range s.Rules {
		add(r.TargetModelID)
	}
	for _, r := range s.Ranges {
		if !seen[r.ModelID] {
			ids = append(ids, r.ModelID)
			seen[r.ModelID] = true
		}
	}
	return ids
}

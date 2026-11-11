package biz

import (
	"encoding/json"
	"strconv"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

// endpointCapabilities is shared by HTTP and Redis publication. Missing limits
// stay unknown; the executor must not turn them into unlimited capacity.
func endpointCapabilities(ep *schema.Endpoint) (int64, int64, []string) {
	var contextLength, maxOutput int64
	var abilities []string
	if ep.Model != nil {
		contextLength, maxOutput = int64(ep.Model.ContextLength), int64(ep.Model.MaxOutputTokens)
		_ = json.Unmarshal([]byte(ep.Model.Abilities), &abilities)
	}
	var metadata map[string]json.RawMessage
	if json.Unmarshal(ep.Metadata, &metadata) == nil {
		for key, dst := range map[string]*int64{"context_length": &contextLength, "max_output_tokens": &maxOutput} {
			if raw, exists := metadata[key]; exists {
				var value int64
				if json.Unmarshal(raw, &value) == nil {
					*dst = value
					continue
				}
				var text string
				if json.Unmarshal(raw, &text) == nil {
					if value, err := strconv.ParseInt(text, 10, 64); err == nil {
						*dst = value
					}
				}
			}
		}
	}
	return contextLength, maxOutput, abilities
}

// validateCompositePromise proves at least one active initial target can meet
// the declared business capacity. Availability and each request remain runtime
// checks; a disabled rule cannot establish this promise.
func validateCompositePromise(model *schema.Model, refs []schema.Model) error {
	if model.SmartRouting == nil || model.SmartRouting.RoutingMode() == compositerouting.Difficulty {
		return nil
	}
	if model.ContextLength <= 0 || model.MaxOutputTokens <= 0 || model.MaxOutputTokens >= model.ContextLength {
		return errors.BadRequest("", "复合模型需要明确的正数上下文及输出上限，且输出上限必须小于上下文")
	}
	var declared []string
	if model.Abilities != "" && json.Unmarshal([]byte(model.Abilities), &declared) != nil {
		return errors.BadRequest("", "复合模型能力配置不合法")
	}
	stream := false
	for _, ability := range declared {
		if ability != "stream" {
			return errors.BadRequest("", "主备/规则模式当前仅支持文本对话及 stream，尚不支持声明其他能力")
		}
		stream = true
	}
	active := map[string]bool{model.SmartRouting.PrimaryModelID: true, model.SmartRouting.DefaultModelID: true}
	for _, rule := range model.SmartRouting.Rules {
		if rule.Enabled {
			active[rule.TargetModelID] = true
		}
	}
	for _, ref := range refs {
		if !active[ref.ID] || ref.ContextLength < model.ContextLength || ref.MaxOutputTokens < model.MaxOutputTokens {
			continue
		}
		if stream {
			var abilities []string
			_ = json.Unmarshal([]byte(ref.Abilities), &abilities)
			found := false
			for _, ability := range abilities {
				found = found || ability == "stream"
			}
			if !found {
				continue
			}
		}
		return nil
	}
	return errors.BadRequest("", "没有有效选模目标能够满足复合模型声明的上下文、输出及流式能力")
}

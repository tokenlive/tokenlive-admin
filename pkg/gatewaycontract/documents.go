package gatewaycontract

import "github.com/tokenlive/tokenlive-admin/pkg/compositerouting"

// ResolvedEndpoint is the Redis routing document for one upstream credential of
// a normal model. Gateway's twin type also accepts auth_type and endpoint_id
// aliases; Admin does not emit either.
type ResolvedEndpoint struct {
	ID                 string            `json:"id,omitempty"`
	Code               string            `json:"code,omitempty"`
	Description        string            `json:"description,omitempty"`
	RealModel          string            `json:"real_model"`
	ProviderName       string            `json:"provider_name"`
	ProviderCode       string            `json:"provider_code,omitempty"`
	ProviderProtocol   string            `json:"provider_protocol"`
	APIKey             string            `json:"api_key"`
	URL                string            `json:"url"`
	Timeout            int64             `json:"timeout"` // milliseconds
	MaxRetries         int               `json:"max_retries"`
	Priority           int               `json:"priority"`
	Weight             int               `json:"weight"`
	Headers            map[string]string `json:"headers,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	RequestTypes       []string          `json:"request_types,omitempty"`
	ContextLength      int64             `json:"context_length,omitempty"`
	MaxOutputTokens    int64             `json:"max_output_tokens,omitempty"`
	Abilities          []string          `json:"abilities,omitempty"`
	InputPrice         *float64          `json:"input_price,omitempty"`
	OutputPrice        *float64          `json:"output_price,omitempty"`
	CachedPrice        *float64          `json:"cached_price,omitempty"`
	CacheCreationPrice *float64          `json:"cache_creation_price,omitempty"`
}

// RuntimeSmartRouting is the smart-routing document shared by the Redis key
// and the HTTP pull payload. Splitting the two would create a third shape.
type RuntimeSmartRouting struct {
	SchemaVersion        int                 `json:"schema_version,omitempty"`
	Mode                 string              `json:"mode,omitempty"`
	PrimaryModel         string              `json:"primary_model,omitempty"`
	DefaultModel         string              `json:"default_model,omitempty"`
	Rules                []RuntimeSmartRule  `json:"rules,omitempty"`
	Version              int64               `json:"version"`
	JudgeModel           string              `json:"judge_model"`
	JudgeTimeoutMS       int                 `json:"judge_timeout_ms"`
	JudgeMaxInputBytes   int                 `json:"judge_max_input_bytes"`
	JudgeMaxOutputTokens int                 `json:"judge_max_output_tokens"`
	Ranges               []RuntimeSmartRange `json:"ranges"`
}

type RuntimeSmartRule struct {
	ID           string                       `json:"id"`
	Name         string                       `json:"name"`
	Order        int                          `json:"order"`
	Enabled      bool                         `json:"enabled"`
	RelationType string                       `json:"relation_type"`
	Conditions   []compositerouting.Condition `json:"conditions"`
	TargetModel  string                       `json:"target_model"`
}

func (r *RuntimeSmartRouting) PlanConfig() compositerouting.Config {
	c := compositerouting.Config{Mode: r.Mode, Primary: r.PrimaryModel, Default: r.DefaultModel}
	for _, rule := range r.Rules {
		c.Rules = append(c.Rules, compositerouting.Rule{ID: rule.ID, Name: rule.Name, Order: rule.Order, Enabled: rule.Enabled, RelationType: rule.RelationType, Conditions: rule.Conditions, Target: rule.TargetModel})
	}
	return c
}

// ReferenceCodes returns all logical dependencies, without empty or duplicate codes.
func (r *RuntimeSmartRouting) ReferenceCodes() []string {
	var result []string
	seen := map[string]bool{}
	add := func(code string) {
		if code != "" && !seen[code] {
			result = append(result, code)
			seen[code] = true
		}
	}
	add(r.JudgeModel)
	add(r.PrimaryModel)
	add(r.DefaultModel)
	for _, rule := range r.Rules {
		add(rule.TargetModel)
	}
	for _, band := range r.Ranges {
		add(band.Model)
	}
	return result
}

// RuntimeSmartRange is one input-size band of a RuntimeSmartRouting.
type RuntimeSmartRange struct {
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Model string `json:"model"`
}

// Billing price keys written as a map, not a struct: a zero price is written,
// which a struct with omitempty would drop.
const (
	PriceInput         = "input_price"
	PriceOutput        = "output_price"
	PriceCached        = "cached_price"
	PriceCacheCreation = "cache_creation_price"
)

package gatewaycontract

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
	InputPrice         *float64          `json:"input_price,omitempty"`
	OutputPrice        *float64          `json:"output_price,omitempty"`
	CachedPrice        *float64          `json:"cached_price,omitempty"`
	CacheCreationPrice *float64          `json:"cache_creation_price,omitempty"`
}

// RuntimeSmartRouting is the smart-routing document shared by the Redis key
// and the HTTP pull payload. Splitting the two would create a third shape.
type RuntimeSmartRouting struct {
	Version              int64               `json:"version"`
	JudgeModel           string              `json:"judge_model"`
	JudgeTimeoutMS       int                 `json:"judge_timeout_ms"`
	JudgeMaxInputBytes   int                 `json:"judge_max_input_bytes"`
	JudgeMaxOutputTokens int                 `json:"judge_max_output_tokens"`
	Ranges               []RuntimeSmartRange `json:"ranges"`
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

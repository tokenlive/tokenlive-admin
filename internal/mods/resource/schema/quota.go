package schema

// ProviderQuotaResult is the unified quota/usage response for oauth providers.
type ProviderQuotaResult struct {
	Provider               string              `json:"provider"` // codex | xai
	Plan                   string              `json:"plan,omitempty"`
	SubscriptionActiveUntil string             `json:"subscription_active_until,omitempty"`
	Windows                []ProviderQuotaWindow `json:"windows"`
	Extras                 map[string]any      `json:"extras,omitempty"`
}

// ProviderQuotaWindow is one usage window / credit bucket.
type ProviderQuotaWindow struct {
	ID               string   `json:"id"`
	Label            string   `json:"label"`
	UsedPercent      *float64 `json:"used_percent,omitempty"`
	RemainingPercent *float64 `json:"remaining_percent,omitempty"`
	ResetAt          string   `json:"reset_at,omitempty"`
	ResetLabel       string   `json:"reset_label,omitempty"`
	// Optional amount labels for billing-style quotas (xAI).
	AmountLabel string `json:"amount_label,omitempty"`
}

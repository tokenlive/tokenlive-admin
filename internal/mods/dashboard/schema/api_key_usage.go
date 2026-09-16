package schema

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

type Query struct {
	TimeRange string
	SortBy    string
	Limit     int
}

func (q Query) Validate() error {
	switch q.TimeRange {
	case "today", "1h", "6h", "24h", "7d":
	default:
		return fmt.Errorf("invalid time range")
	}
	switch q.SortBy {
	case "tokens", "request_count", "cost":
	default:
		return fmt.Errorf("invalid sort field")
	}
	if q.Limit != 10 && q.Limit != 20 && q.Limit != 50 {
		return fmt.Errorf("invalid ranking limit")
	}
	return nil
}

func ParseQuery(values url.Values) (Query, error) {
	q := Query{TimeRange: "today", SortBy: "tokens", Limit: 10}
	for name, items := range values {
		if len(items) != 1 || items[0] == "" {
			return Query{}, fmt.Errorf("invalid query parameter")
		}
		switch name {
		case "time_range":
			q.TimeRange = items[0]
		case "sort_by":
			q.SortBy = items[0]
		case "limit":
			value, err := strconv.Atoi(items[0])
			if err != nil {
				return Query{}, fmt.Errorf("invalid ranking limit")
			}
			q.Limit = value
		default:
			return Query{}, fmt.Errorf("unsupported query parameter")
		}
	}
	return q, q.Validate()
}

type Window struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Timezone string    `json:"timezone"`
}

// ResolveWindow expects a validated query and preserves the server's day boundary.
func ResolveWindow(q Query, end time.Time) Window {
	start := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	durations := map[string]time.Duration{"1h": time.Hour, "6h": 6 * time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour}
	if duration, ok := durations[q.TimeRange]; ok {
		start = end.Add(-duration)
	}
	return Window{Start: start, End: end, Timezone: end.Location().String()}
}

// KeyRef, Candidate, Resolution, Group and Metadata are internal only, never HTTP DTOs.
type KeyRef struct {
	Hash, KeyID, WorkspaceID, UserID, TenantID string
}

type Totals struct {
	Requests, Success, Input, Output, Cached, CacheCreation uint64
	Cost                                                    decimal.Decimal
}

func (t Totals) Tokens() uint64 { return t.Input + t.Output }

func (t *Totals) Add(other Totals) {
	t.Requests += other.Requests
	t.Success += other.Success
	t.Input += other.Input
	t.Output += other.Output
	t.Cached += other.Cached
	t.CacheCreation += other.CacheCreation
	t.Cost = t.Cost.Add(other.Cost)
}

type Candidate struct {
	Ref     KeyRef
	Display string
	Totals  Totals
}

type Resolution struct {
	Keys     map[KeyRef]string
	Warnings []string
}

type Group struct {
	Canonical string
	Ref       KeyRef
	Display   string
	Totals    Totals
}

type Owner struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Metadata struct {
	KeyName, Display, Source, KeyStatus, MetadataStatus string
	Owner                                               Owner
}

type Metrics struct {
	RequestCount        uint64   `json:"request_count"`
	SuccessCount        uint64   `json:"success_count"`
	SuccessRate         float64  `json:"success_rate"`
	InputTokens         uint64   `json:"input_tokens"`
	OutputTokens        uint64   `json:"output_tokens"`
	CachedTokens        uint64   `json:"cached_tokens"`
	CacheCreationTokens uint64   `json:"cache_creation_tokens"`
	TotalTokens         uint64   `json:"total_tokens"`
	TotalCost           string   `json:"total_cost"`
	TokenShare          *float64 `json:"token_share"`
}

type Item struct {
	RowID          string `json:"row_id"`
	KeyName        string `json:"key_name"`
	KeyDisplay     string `json:"key_display"`
	Source         string `json:"source"`
	Owner          Owner  `json:"owner"`
	KeyStatus      string `json:"key_status"`
	MetadataStatus string `json:"metadata_status"`
	Metrics
}

type Summary struct {
	KeyCount uint64 `json:"key_count"`
	Metrics
}

type Response struct {
	State        string     `json:"state"`
	DataSource   string     `json:"data_source"`
	Window       *Window    `json:"window"`
	GeneratedAt  *time.Time `json:"generated_at"`
	Summary      *Summary   `json:"summary"`
	Items        []Item     `json:"items"`
	Unattributed *Metrics   `json:"unattributed"`
	Warnings     []string   `json:"warnings"`
}

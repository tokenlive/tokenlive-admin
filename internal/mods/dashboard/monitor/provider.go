package monitor

import (
	"context"
	"fmt"
	"sort"
	"time"

	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
)

const providerMetricsMaxRange = 7 * 24 * time.Hour
const providerRedisMaxMinutes = 120

// One identity is selected for an entire source/window, including genuine zeros.
func providerStoreKeys(p *rschema.Provider) []string {
	keys := []string{}
	if p.Code != "" {
		keys = append(keys, p.Code)
	}
	if p.Name != "" && p.Name != p.Code {
		keys = append(keys, p.Name)
	}
	return keys
}

func (a *Reader) readProviders(ctx context.Context, sortBy string, limit int, filter string, p windowPlan) ([]ProviderHealth, error) {
	if p.minutes() == 0 {
		return []ProviderHealth{}, nil
	}
	ctx, _ = withPlan(ctx, p)
	key := fmt.Sprintf("provider-health:%s:%d:%s", sortBy, limit, filter)
	if cached, ok := a.getCache(ctx, key); ok {
		return cloneProviderHealth(cached.([]ProviderHealth)), nil
	}
	if a.db == nil {
		return nil, fmt.Errorf("provider catalog is unavailable")
	}
	var providers []rschema.Provider
	query := a.db.WithContext(ctx)
	if filter != "" {
		query = query.Where("id = ? OR code = ?", filter, filter)
	} else {
		query = query.Where("enabled = ?", 1)
	}
	if err := query.Find(&providers).Error; err != nil {
		return nil, err
	}
	items := make([]ProviderHealth, len(providers))
	keys := make([][]string, len(providers))
	for i, provider := range providers {
		items[i] = ProviderHealth{ProviderID: provider.ID, ProviderCode: provider.Code, ProviderName: provider.Name}
		keys[i] = providerStoreKeys(&provider)
	}
	if len(items) == 0 {
		return items, nil
	}
	a.readProviderCalls(ctx, items, keys, p)
	a.readProviderPerformance(ctx, items, keys, p)
	a.readProviderInventory(ctx, items, p)
	if filter == "" {
		filtered := make([]ProviderHealth, 0, len(items))
		for _, item := range items {
			if item.Calls.Value == nil || item.Calls.State != "ok" || item.Calls.Value.Total > 0 {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	sortProviderHealth(items, sortBy)
	if filter == "" && len(items) > limit {
		items = items[:limit]
	}
	cacheable := true
	for _, item := range items {
		for _, state := range providerMetricStates(item) {
			if state == "error" {
				cacheable = false
			}
		}
	}
	if cacheable {
		a.setCache(ctx, key, cloneProviderHealth(items), 5*time.Second)
	}
	return items, nil
}

func providerReading(p windowPlan, items []ProviderHealth, err error) *Reading[[]ProviderHealth] {
	r := &Reading[[]ProviderHealth]{State: "ok", Source: "mixed", Window: p.Window, Data: &items, Coverage: Coverage{Reason: "per_metric_coverage"}}
	if err != nil {
		r.State, r.Error, r.Data = "error", "source_unavailable", nil
		return r
	}
	if p.minutes() == 0 {
		r.State, r.Coverage.Reason, r.Data = "unavailable", "no_completed_minutes", nil
		return r
	}
	if len(items) == 0 {
		r.Source = "catalog"
		r.Coverage = Coverage{Start: &p.Start, End: &p.End}
		return r
	}
	for _, item := range items {
		for _, state := range providerMetricStates(item) {
			if state == "partial" || state == "error" || state == "unavailable" || state == "unsupported" {
				r.State = "partial"
			}
		}
	}
	return r
}

func providerMetricStates(p ProviderHealth) []string {
	return []string{p.Calls.State, p.SuccessRate.State, p.AvgLatencyMs.State, p.AvgTTFTMs.State, p.Otps.State, p.TotalTokens.State, p.TotalCost.State, p.EndpointCount.State, p.OpenBreakerCount.State}
}

func providerRate(calls MetricObservation[ProviderCalls]) MetricObservation[float64] {
	m := MetricObservation[float64]{State: calls.State, Source: calls.Source, Scope: calls.Scope, Window: calls.Window, Coverage: calls.Coverage, Reason: calls.Reason, Identity: calls.Identity}
	if calls.Value == nil {
		return m
	}
	if calls.Value.Total == 0 {
		return missingObservation(m, "no_samples", "zero_denominator")
	}
	value := float64(calls.Value.Success) / float64(calls.Value.Total) * 100
	m.Value = &value
	return m
}

func sortProviderHealth(items []ProviderHealth, by string) {
	value := func(p ProviderHealth) *float64 {
		switch by {
		case "avg_latency":
			return p.AvgLatencyMs.Value
		case "avg_ttft":
			return p.AvgTTFTMs.Value
		case "tokens":
			if p.TotalTokens.Value != nil {
				v := float64(*p.TotalTokens.Value)
				return &v
			}
		case "cost":
			return p.TotalCost.Value
		case "success_rate":
			return p.SuccessRate.Value
		case "otps":
			return p.Otps.Value
		default:
			if p.Calls.Value != nil {
				v := float64(p.Calls.Value.Total)
				return &v
			}
		}
		return nil
	}
	ascending := by == "avg_latency" || by == "avg_ttft"
	sort.SliceStable(items, func(i, j int) bool {
		x, y := value(items[i]), value(items[j])
		if x == nil {
			return false
		}
		if y == nil {
			return true
		}
		if *x == *y {
			return items[i].ProviderCode < items[j].ProviderCode
		}
		if ascending {
			return *x < *y
		}
		return *x > *y
	})
}

func cloneProviderHealth(items []ProviderHealth) []ProviderHealth {
	out := append([]ProviderHealth{}, items...)
	for i := range out {
		p := &out[i]
		p.Calls = cloneObservation(p.Calls)
		p.SuccessRate = cloneObservation(p.SuccessRate)
		p.AvgLatencyMs = cloneObservation(p.AvgLatencyMs)
		p.AvgTTFTMs = cloneObservation(p.AvgTTFTMs)
		p.TotalTokens = cloneObservation(p.TotalTokens)
		p.TotalCost = cloneObservation(p.TotalCost)
		p.Otps = cloneObservation(p.Otps)
		p.EndpointCount = cloneObservation(p.EndpointCount)
		p.OpenBreakerCount = cloneObservation(p.OpenBreakerCount)
	}
	return out
}

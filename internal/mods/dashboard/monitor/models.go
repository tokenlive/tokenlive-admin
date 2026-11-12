package monitor

import (
	"context"
	"fmt"
	"sort"
	"time"

	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
)

func buildModelTokensQuery(promRange string) string {
	return fmt.Sprintf(`sum by (model) (increase(%s{type=~"input|output"}[%s]))`, mTokensTotal, promRange)
}

func buildModelOtpsQuery(promRange string) string {
	return fmt.Sprintf(`sum by (model) (increase(%s{type="output"}[%s])) / sum by (model) (increase(%s[%s]))`, mTokensTotal, promRange, mRequestDurationSum, promRange)
}

// Each measurement owns its source status. A failed query cannot discard a model's
// successful measurements, or turn its unknown latency into a winning zero.
func (a *Reader) readModels(ctx context.Context, sortBy string, limit int, modelFilter string, p windowPlan) ([]ModelRankingItem, error) {
	if p.minutes() == 0 {
		return []ModelRankingItem{}, nil
	}
	ctx, _ = withPlan(ctx, p)
	cacheKey := fmt.Sprintf("ranking:%s:%d:%s", sortBy, limit, modelFilter)
	if cached, ok := a.getCache(ctx, cacheKey); ok {
		return cloneModelRanking(cached.([]ModelRankingItem)), nil
	}
	if a.db == nil {
		return nil, fmt.Errorf("model catalog is unavailable")
	}
	var models []rschema.Model
	query := a.db.WithContext(ctx).Where("enabled = ?", 1)
	if modelFilter != "" {
		query = query.Where("id = ? OR model_code = ?", modelFilter, modelFilter)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]ModelRankingItem, len(models))
	for i, model := range models {
		items[i] = ModelRankingItem{ModelID: model.ID, ModelCode: model.ModelCode, ModelName: model.ModelName}
	}
	if len(items) == 0 {
		return items, nil
	}
	if a.isPrometheusAvailable(ctx) {
		a.readModelPrometheus(ctx, items, p)
	} else {
		a.readModelRequests(ctx, items, p)
		a.readModelMemoryPerformance(items, p)
	}
	cacheable := true
	for _, row := range items {
		for _, state := range modelMetricStates(row) {
			if state == "error" {
				cacheable = false
			}
		}
	}
	if modelFilter == "" {
		filtered := make([]ModelRankingItem, 0, len(items))
		for _, row := range items {
			// no_samples means the source query succeeded over a covered window and
			// the model genuinely had no traffic, so it is a confirmed zero and can
			// be dropped. Other non-ok states keep the row: an uncertain source
			// must not silently hide a model.
			confirmedZero := row.Requests.State == "ok" && row.Requests.Value != nil && row.Requests.Value.Total == 0 ||
				row.Requests.State == "no_samples"
			if !confirmedZero {
				filtered = append(filtered, row)
			}
		}
		items = filtered
	}
	sortModelRankingItems(items, sortBy)
	if modelFilter == "" && len(items) > limit {
		items = items[:limit]
	}
	if cacheable {
		a.setCache(ctx, cacheKey, cloneModelRanking(items), 5*time.Second)
	}
	return items, nil
}

func modelReading(p windowPlan, items []ModelRankingItem, err error) *Reading[[]ModelRankingItem] {
	r := &Reading[[]ModelRankingItem]{State: "ok", Source: "mixed", Window: p.Window, Data: &items, Coverage: Coverage{Reason: "per_metric_coverage"}}
	if err != nil {
		r.State, r.Error, r.Data = "error", "source_unavailable", nil
		return r
	}
	if p.minutes() == 0 {
		r.State, r.Coverage.Reason, r.Data = "unavailable", "no_completed_minutes", nil
		return r
	}
	if len(items) == 0 {
		r.Source, r.Coverage = "catalog", Coverage{Start: &p.Start, End: &p.End}
		return r
	}
	for _, row := range items {
		for _, state := range modelMetricStates(row) {
			if state == "partial" || state == "error" || state == "unavailable" || state == "unsupported" {
				r.State = "partial"
			}
		}
	}
	return r
}

func modelMetricStates(row ModelRankingItem) []string {
	return []string{row.Requests.State, row.SuccessRate.State, row.AvgLatencyMs.State, row.P50LatencyMs.State, row.P95LatencyMs.State, row.P99LatencyMs.State, row.AvgTTFTMs.State, row.P50TTFTMs.State, row.P95TTFTMs.State, row.P99TTFTMs.State, row.TotalTokens.State, row.TotalCost.State, row.Otps.State}
}

func modelRate(requests MetricObservation[RequestCounts]) MetricObservation[float64] {
	m := MetricObservation[float64]{State: requests.State, Source: requests.Source, Scope: requests.Scope, Window: requests.Window, Coverage: requests.Coverage, Reason: requests.Reason, Identity: requests.Identity}
	if requests.Value == nil {
		return m
	}
	if requests.Value.Total == 0 {
		return missingObservation(m, "no_samples", "zero_denominator")
	}
	value := float64(requests.Value.Success) / float64(requests.Value.Total) * 100
	m.Value = &value
	return m
}

func sortModelRankingItems(items []ModelRankingItem, by string) {
	value := func(row ModelRankingItem) *float64 {
		switch by {
		case "avg_latency":
			return row.AvgLatencyMs.Value
		case "avg_ttft":
			return row.AvgTTFTMs.Value
		case "tokens":
			if row.TotalTokens.Value != nil {
				v := float64(*row.TotalTokens.Value)
				return &v
			}
		case "cost":
			return row.TotalCost.Value
		case "success_rate":
			return row.SuccessRate.Value
		case "otps":
			return row.Otps.Value
		default:
			if row.Requests.Value != nil {
				v := float64(row.Requests.Value.Total)
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
			return items[i].ModelCode < items[j].ModelCode
		}
		if ascending {
			return *x < *y
		}
		return *x > *y
	})
}

func cloneModelRanking(items []ModelRankingItem) []ModelRankingItem {
	out := append([]ModelRankingItem{}, items...)
	for i := range out {
		row := &out[i]
		row.Requests = cloneObservation(row.Requests)
		row.SuccessRate = cloneObservation(row.SuccessRate)
		row.AvgLatencyMs = cloneObservation(row.AvgLatencyMs)
		row.P50LatencyMs = cloneObservation(row.P50LatencyMs)
		row.P95LatencyMs = cloneObservation(row.P95LatencyMs)
		row.P99LatencyMs = cloneObservation(row.P99LatencyMs)
		row.AvgTTFTMs = cloneObservation(row.AvgTTFTMs)
		row.P50TTFTMs = cloneObservation(row.P50TTFTMs)
		row.P95TTFTMs = cloneObservation(row.P95TTFTMs)
		row.P99TTFTMs = cloneObservation(row.P99TTFTMs)
		row.TotalTokens = cloneObservation(row.TotalTokens)
		row.TotalCost = cloneObservation(row.TotalCost)
		row.Otps = cloneObservation(row.Otps)
	}
	return out
}

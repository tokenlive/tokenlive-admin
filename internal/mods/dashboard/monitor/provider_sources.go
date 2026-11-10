package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

func providerIdentity(p ProviderHealth, key string) string {
	if key == p.ProviderCode && key != "" {
		return "provider_code"
	}
	if key != "" {
		return "provider_name"
	}
	return ""
}

func (a *Reader) providerMemory(p windowPlan, keys []string) (metrics.EndpointMinutePerf, string, Coverage) {
	c := a.memoryCoverage(p)
	if c.Start == nil {
		return metrics.EndpointMinutePerf{}, "", c
	}
	for _, key := range keys {
		value, found := a.memory().LookupProviderMinutePerf(key, c.Start.Unix()/60, c.End.Unix()/60-1)
		if found {
			return value, key, c
		}
	}
	return metrics.EndpointMinutePerf{}, "", c
}

type providerCallSeries struct {
	Metric  HealthMetric[ProviderCalls]
	Minutes []ProviderCalls
}

func (a *Reader) readProviderCalls(ctx context.Context, items []ProviderHealth, keys [][]string, p windowPlan) {
	for i, row := range a.providerCallsSeries(ctx, items, keys, p) {
		items[i].Calls = row.Metric
		items[i].SuccessRate = providerRate(row.Metric)
	}
}

// One source and one identity supply every minute and both outcome counters.
// The same observations drive KPI totals and the provider traffic chart.
func (a *Reader) providerCallsSeries(ctx context.Context, items []ProviderHealth, keys [][]string, p windowPlan) []providerCallSeries {
	rows := make([]providerCallSeries, len(items))
	c := sourceCoverage(p, p.Observed.Add(-providerRedisMaxMinutes*time.Minute).Truncate(time.Minute).Add(time.Minute))
	redisFailed := false
	if a.redis != nil && c.Start != nil {
		queryKeys := []string{}
		for _, candidates := range keys {
			for _, key := range candidates {
				for minute := c.Start.Unix() / 60; minute < c.End.Unix()/60; minute++ {
					queryKeys = append(queryKeys, gatewaycontract.Keys.Status.Provider(key, minute, gatewaycontract.MetricSuccess), gatewaycontract.Keys.Status.Provider(key, minute, gatewaycontract.MetricFailure))
				}
			}
		}
		values := []interface{}{}
		var err error
		for offset := 0; offset < len(queryKeys); offset += 500 {
			var batch []interface{}
			batch, err = a.redis.MGet(ctx, queryKeys[offset:min(offset+500, len(queryKeys))]...).Result()
			if err != nil {
				break
			}
			values = append(values, batch...)
		}
		if err == nil && len(values) == len(queryKeys) {
			index := 0
			for i := range items {
				calls := healthMetric[ProviderCalls](p, "redis", "upstream_call", c, "")
				found := false
				for _, key := range keys[i] {
					minutes := make([]ProviderCalls, p.minutes())
					candidate := ProviderCalls{}
					exists, valid := false, true
					for minute := c.Start.Unix() / 60; minute < c.End.Unix()/60; minute++ {
						entry := &minutes[int(minute-p.Start.Unix()/60)]
						for j := 0; j < 2; j++ {
							raw := values[index]
							index++
							if raw == nil {
								continue
							}
							exists = true
							count, e := strconv.ParseInt(fmt.Sprint(raw), 10, 64)
							if e != nil || count < 0 {
								valid = false
								continue
							}
							if j == 0 {
								entry.Success = count
								candidate.Success += count
							} else {
								entry.Failure = count
								candidate.Failure += count
							}
						}
						entry.Total = entry.Success + entry.Failure
					}
					if !found && exists {
						found = true
						calls.Identity = providerIdentity(items[i], key)
						candidate.Total = candidate.Success + candidate.Failure
						if valid {
							calls.Value = &candidate
							rows[i].Minutes = minutes
						} else {
							calls = missingHealthMetric(calls, "error", "invalid_source_value")
						}
					}
				}
				if !found {
					calls = missingHealthMetric(calls, "no_samples", "no_source_samples")
				}
				rows[i].Metric = calls
			}
			return rows
		}
		redisFailed = true
	}
	for i := range items {
		perf, key, coverage := a.providerMemory(p, keys[i])
		calls := healthMetric[ProviderCalls](p, "memory", "upstream_call", coverage, providerIdentity(items[i], key))
		if coverage.Start != nil {
			calls.Value = &ProviderCalls{Total: perf.UpstreamSuccess + perf.UpstreamFailure, Success: perf.UpstreamSuccess, Failure: perf.UpstreamFailure}
			rows[i].Minutes = make([]ProviderCalls, p.minutes())
			for minute := coverage.Start.Unix() / 60; minute < coverage.End.Unix()/60; minute++ {
				minutePerf := a.memory().GetProviderMinutePerf(key, minute)
				success, failure := minutePerf.UpstreamSuccess, minutePerf.UpstreamFailure
				rows[i].Minutes[int(minute-p.Start.Unix()/60)] = ProviderCalls{Total: success + failure, Success: success, Failure: failure}
			}
		}
		if perf.LegacyProviderRequests > 0 {
			if calls.Value != nil && calls.Value.Total > 0 {
				calls.State, calls.Reason = "partial", "upstream_samples_unavailable"
			} else {
				calls = missingHealthMetric(calls, "no_samples", "upstream_samples_unavailable")
				rows[i].Minutes = nil
			}
		}
		if redisFailed && calls.Value != nil {
			calls.State = "partial"
			calls.Reason = "redis_unavailable"
		}
		if redisFailed && calls.Value == nil {
			calls = missingHealthMetric(calls, "error", "redis_unavailable")
		}
		rows[i].Metric = calls
	}
	return rows
}

func (a *Reader) readProviderPerformance(ctx context.Context, items []ProviderHealth, keys [][]string, p windowPlan) {
	if a.isPrometheusAvailable(ctx) {
		a.readProviderPrometheus(ctx, items, keys, p)
		return
	}
	for i := range items {
		perf, key, c := a.providerMemory(p, keys[i])
		population := "successful_provider_requests"
		if perf.LegacyProviderRequests > 0 {
			population = "attributed_provider_requests"
		}
		base := healthMetric[float64](p, "memory", population, c, providerIdentity(items[i], key))
		value := func(numerator float64, denominator int64) HealthMetric[float64] {
			m := base
			if c.Start == nil {
				return m
			}
			if denominator <= 0 {
				return missingHealthMetric(m, "no_samples", "no_source_samples")
			}
			v := numerator / float64(denominator)
			m.Value = &v
			return m
		}
		items[i].AvgLatencyMs = value(float64(perf.LatencySumMs), perf.LatencyCount)
		items[i].AvgTTFTMs = value(float64(perf.TTFTSum), perf.TTFTCount)
		items[i].Otps = value(float64(perf.Output)*1000, perf.DurationMs)
		items[i].TotalCost = value(perf.Cost, 1)
		tokens := healthMetric[int64](p, "memory", population, c, base.Identity)
		if c.Start != nil && perf.RequestSamples > 0 {
			v := perf.InputTokens + perf.OutputTokens
			tokens.Value = &v
		} else if c.Start != nil {
			tokens = missingHealthMetric(tokens, "no_samples", "no_source_samples")
		}
		items[i].TotalTokens = tokens
		if perf.RequestSamples == 0 && c.Start != nil {
			items[i].TotalCost = missingHealthMetric(base, "no_samples", "no_source_samples")
		}
	}
}

type providerVector struct {
	values        map[string]*float64
	state, reason string
}

func (a *Reader) providerVector(ctx context.Context, query string, p windowPlan) providerVector {
	c, _ := withPlan(ctx, p) // A failed measurement must not contaminate another one.
	api := config.C.Util.PrometheusServer.Address + "/api/v1/query?query=" + url.QueryEscape(query) + "&time=" + strconv.FormatInt(p.End.Unix(), 10)
	body, err := a.queryPrometheus(c, api, 3*time.Second)
	v := providerVector{values: map[string]*float64{}, state: "ok"}
	if err != nil {
		v.state, v.reason = "error", "source_unavailable"
		return v
	}
	var response prometheusQueryResponse
	if err = json.Unmarshal(body, &response); err != nil || response.Status != "success" || response.Data.ResultType != "vector" {
		v.state, v.reason = "error", "invalid_source_response"
		return v
	}
	missingDimension := false
	for _, row := range response.Data.Result {
		key := row.Metric["provider"]
		if key == "" {
			missingDimension = true
			continue
		}
		v.values[key] = nil
		if len(row.Value) < 2 {
			continue
		}
		n, e := strconv.ParseFloat(fmt.Sprint(row.Value[1]), 64)
		if e == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0 {
			v.values[key] = &n
		}
	}
	if missingDimension && len(v.values) == 0 {
		v.state, v.reason = "unsupported", "provider_dimension_unavailable"
	}
	return v
}

func (a *Reader) readProviderPrometheus(ctx context.Context, items []ProviderHealth, keys [][]string, p windowPlan) {
	duration := fmt.Sprintf("%ds", int64(p.End.Sub(p.Start)/time.Second))
	queries := []string{
		fmt.Sprintf(`sum by (provider) (increase(%s[%s])) / sum by (provider) (increase(%s[%s]))`, mRequestDurationSum, duration, mRequestDurationCount, duration),
		fmt.Sprintf(`sum by (provider) (increase(%s[%s])) / sum by (provider) (increase(%s[%s]))`, mTtftSum, duration, mTtftCount, duration),
		fmt.Sprintf(`sum by (provider) (increase(%s{type=~"input|output"}[%s]))`, mTokensTotal, duration),
		fmt.Sprintf(`sum by (provider) (increase(%s{type="output"}[%s])) / sum by (provider) (increase(%s[%s]))`, mTokensTotal, duration, mRequestDurationSum, duration),
		fmt.Sprintf(`sum by (provider) (increase(%s[%s]))`, mCostTotal, duration),
	}
	vectors := make([]providerVector, len(queries))
	probeCtx, probe := withPlan(ctx, p)
	source(probeCtx, "prometheus")
	var reads sync.WaitGroup
	reads.Add(len(queries) + 1)
	for i, query := range queries {
		go func(index int, expression string) {
			defer reads.Done()
			vectors[index] = a.providerVector(ctx, expression, p)
		}(i, query)
	}
	go func() { defer reads.Done(); a.checkPrometheusCoverage(probeCtx, probe) }()
	reads.Wait()
	coverage := Coverage{Reason: "coverage_unverified"}
	if probe.audit.coverage != nil {
		coverage = *probe.audit.coverage
	}
	for i := range items {
		selected := ""
		for _, key := range keys[i] {
			present := false
			for _, v := range vectors {
				if _, ok := v.values[key]; ok {
					present = true
				}
			}
			if present {
				selected = key
				break
			}
		}
		metric := func(index int, factor float64) HealthMetric[float64] {
			m := healthMetric[float64](p, "prometheus", "final_provider_requests", coverage, providerIdentity(items[i], selected))
			v := vectors[index]
			if v.state != "ok" {
				return missingHealthMetric(m, v.state, v.reason)
			}
			n := v.values[selected]
			if n == nil {
				return missingHealthMetric(m, "no_samples", "no_source_samples")
			}
			number := *n * factor
			m.Value = &number
			if coverage.Start == nil {
				m.State = "partial"
				m.Reason = "coverage_unverified"
			}
			return m
		}
		items[i].AvgLatencyMs = metric(0, 1000)
		items[i].AvgTTFTMs = metric(1, 1000)
		items[i].Otps = metric(3, 1)
		items[i].TotalCost = metric(4, 1)
		tokenValue := metric(2, 1)
		token := HealthMetric[int64]{State: tokenValue.State, Source: tokenValue.Source, Scope: tokenValue.Scope, Identity: tokenValue.Identity, Window: tokenValue.Window, Coverage: tokenValue.Coverage, Reason: tokenValue.Reason}
		if tokenValue.Value != nil {
			if *tokenValue.Value < float64(math.MaxInt64) {
				n := int64(*tokenValue.Value)
				token.Value = &n
			} else {
				token = missingHealthMetric(token, "error", "invalid_source_value")
			}
		}
		items[i].TotalTokens = token
	}
}

func (a *Reader) readProviderInventory(ctx context.Context, items []ProviderHealth, p windowPlan) {
	instant := p
	instant.Window.Start = p.Observed
	instant.Window.End = p.Observed
	c := Coverage{Start: &p.Observed, End: &p.Observed}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ProviderID
		items[i].EndpointCount = healthMetric[int64](instant, "catalog", "current_inventory", c, "")
	}
	var endpoints []rschema.Endpoint
	err := a.db.WithContext(ctx).Select("id", "provider_id").Where("provider_id IN ?", ids).Find(&endpoints).Error
	runtimeSource := "memory"
	open := a.memory().GetOpenEndpoints()
	var runtimeErr error
	if a.redis != nil {
		runtimeSource = "redis"
		open, runtimeErr = a.redis.SMembers(ctx, gatewaycontract.Keys.Circuit.OpenEndpoints).Result()
	}
	openSet := map[string]bool{}
	for _, id := range open {
		openSet[id] = true
	}
	for i := range items {
		m := healthMetric[int64](instant, runtimeSource, "current_inventory", c, "")
		if err != nil {
			items[i].EndpointCount = missingHealthMetric(items[i].EndpointCount, "error", "source_unavailable")
			items[i].OpenBreakerCount = missingHealthMetric(m, "error", "inventory_unavailable")
			continue
		}
		var count, breakers int64
		for _, ep := range endpoints {
			if ep.ProviderID == items[i].ProviderID {
				count++
				if openSet[ep.ID] {
					breakers++
				}
			}
		}
		items[i].EndpointCount.Value = &count
		if runtimeErr != nil {
			m = missingHealthMetric(m, "error", "source_unavailable")
		} else {
			m.Value = &breakers
		}
		items[i].OpenBreakerCount = m
	}
}

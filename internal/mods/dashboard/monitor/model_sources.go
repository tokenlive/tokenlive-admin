package monitor

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

func (a *Reader) readModelRequests(ctx context.Context, items []ModelRankingItem, p windowPlan) {
	coverage := sourceCoverage(p, p.Observed.Add(-120*time.Minute).Truncate(time.Minute).Add(time.Minute))
	redisFailed := false
	if a.redis != nil && coverage.Start != nil {
		minutes := int(coverage.End.Sub(*coverage.Start) / time.Minute)
		keys := make([]string, 0, len(items)*minutes*2)
		for _, row := range items {
			for minute := coverage.Start.Unix() / 60; minute < coverage.End.Unix()/60; minute++ {
				keys = append(keys, gatewaycontract.Keys.Status.Model(row.ModelCode, minute, gatewaycontract.MetricSuccess), gatewaycontract.Keys.Status.Model(row.ModelCode, minute, gatewaycontract.MetricFailure))
			}
		}
		values := []interface{}{}
		var err error
		for offset := 0; offset < len(keys); offset += 500 {
			var batch []interface{}
			batch, err = a.redis.MGet(ctx, keys[offset:min(offset+500, len(keys))]...).Result()
			if err != nil {
				break
			}
			values = append(values, batch...)
		}
		if err == nil && len(values) == len(keys) {
			index := 0
			for i := range items {
				m := observation[RequestCounts](p, "redis", "model_requests", coverage, "model_code")
				counts := RequestCounts{}
				exists, valid := false, true
				for minute := coverage.Start.Unix() / 60; minute < coverage.End.Unix()/60; minute++ {
					for outcome := 0; outcome < 2; outcome++ {
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
						if outcome == 0 {
							if count > math.MaxInt64-counts.Success {
								valid = false
							} else {
								counts.Success += count
							}
						} else if count > math.MaxInt64-counts.Failure {
							valid = false
						} else {
							counts.Failure += count
						}
					}
				}
				if counts.Success > math.MaxInt64-counts.Failure {
					valid = false
				}
				if !exists {
					m = missingObservation(m, "no_samples", "no_source_samples")
				} else if !valid {
					m = missingObservation(m, "error", "invalid_source_value")
				} else {
					counts.Total = counts.Success + counts.Failure
					m.Value = &counts
				}
				items[i].Requests, items[i].SuccessRate = m, modelRate(m)
			}
			return
		}
		redisFailed = true
	}
	coverage = a.memoryCoverage(p)
	for i := range items {
		m := observation[RequestCounts](p, "memory", "model_requests", coverage, "model_code")
		if coverage.Start != nil {
			perf := a.memory().AggregateModelMinutePerf(items[i].ModelCode, coverage.Start.Unix()/60, coverage.End.Unix()/60-1)
			m.Value = &RequestCounts{Total: perf.Success + perf.Fail, Success: perf.Success, Failure: perf.Fail}
		}
		if redisFailed {
			if m.Value == nil {
				m = missingObservation(m, "error", "redis_unavailable")
			} else {
				m.State, m.Reason = "partial", "redis_unavailable"
			}
		}
		items[i].Requests, items[i].SuccessRate = m, modelRate(m)
	}
}

func (a *Reader) readModelMemoryPerformance(items []ModelRankingItem, p windowPlan) {
	coverage := a.memoryCoverage(p)
	for i := range items {
		row := &items[i]
		var perf metrics.EndpointMinutePerf
		if coverage.Start != nil {
			perf = a.memory().AggregateModelMinutePerf(row.ModelCode, coverage.Start.Unix()/60, coverage.End.Unix()/60-1)
		}
		base := observation[float64](p, "memory", "model_requests", coverage, "model_code")
		token := observation[int64](p, "memory", "model_requests", coverage, "model_code")
		redisOnly := perf.Requests == 0 && row.Requests.Source == "redis" && row.Requests.Value != nil
		missing := func() MetricObservation[float64] {
			if redisOnly {
				m := observation[float64](p, "redis", "model_requests", row.Requests.Coverage, row.Requests.Identity)
				return missingObservation(m, "unsupported", "source_unsupported")
			}
			if coverage.Start == nil {
				return base
			}
			return missingObservation(base, "no_samples", "no_source_samples")
		}
		average := func(numerator float64, denominator int64) MetricObservation[float64] {
			if denominator <= 0 {
				return missing()
			}
			m := base
			value := numerator / float64(denominator)
			m.Value = &value
			return m
		}
		row.AvgLatencyMs = average(float64(perf.LatencySumMs), perf.LatencyCount)
		row.AvgTTFTMs = average(float64(perf.TTFTSum), perf.TTFTCount)
		row.Otps = average(float64(perf.Output)*1000, perf.DurationMs)
		row.TotalCost = missing()
		if perf.Requests > 0 {
			cost, tokens := perf.Cost, perf.InputTokens+perf.OutputTokens
			row.TotalCost = base
			row.TotalCost.Value, token.Value = &cost, &tokens
		} else if redisOnly {
			token = observation[int64](p, "redis", "model_requests", row.Requests.Coverage, row.Requests.Identity)
			token = missingObservation(token, "unsupported", "source_unsupported")
		} else if coverage.Start != nil {
			token = missingObservation(token, "no_samples", "no_source_samples")
		}
		row.TotalTokens = token
		histogram := missingObservation(base, "unsupported", "no_histogram")
		row.P50LatencyMs, row.P95LatencyMs, row.P99LatencyMs = histogram, histogram, histogram
		row.P50TTFTMs, row.P95TTFTMs, row.P99TTFTMs = histogram, histogram, histogram
	}
}

func (a *Reader) readModelPrometheus(ctx context.Context, items []ModelRankingItem, p windowPlan) {
	duration := fmt.Sprintf("%ds", int64(p.End.Sub(p.Start)/time.Second))
	queries := []string{
		fmt.Sprintf(`sum by (model) (increase(%s[%s])) / sum by (model) (increase(%s[%s]))`, mRequestDurationSum, duration, mRequestDurationCount, duration),
		fmt.Sprintf(`histogram_quantile(0.50, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, duration),
		fmt.Sprintf(`histogram_quantile(0.95, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, duration),
		fmt.Sprintf(`histogram_quantile(0.99, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, duration),
		fmt.Sprintf(`sum by (model) (increase(%s[%s])) / sum by (model) (increase(%s[%s]))`, mTtftSum, duration, mTtftCount, duration),
		fmt.Sprintf(`histogram_quantile(0.50, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, duration),
		fmt.Sprintf(`histogram_quantile(0.95, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, duration),
		fmt.Sprintf(`histogram_quantile(0.99, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, duration),
		buildModelTokensQuery(duration),
		fmt.Sprintf(`sum by (model) (increase(%s[%s]))`, mCostTotal, duration),
		buildModelOtpsQuery(duration),
	}
	vectors := make([]metricVector, len(queries))
	var requests modelRequestVector
	probeCtx, probe := withPlan(ctx, p)
	source(probeCtx, "prometheus")
	var reads sync.WaitGroup
	reads.Add(len(queries) + 2)
	for i, query := range queries {
		go func(index int, expression string) {
			defer reads.Done()
			vectors[index] = a.prometheusVector(ctx, expression, "model", p)
		}(i, query)
	}
	go func() {
		defer reads.Done()
		query := fmt.Sprintf(`sum by (model, status) (increase(%s{status=~"success|error"}[%s]))`, mRequestTotal, duration)
		requests = a.modelRequestVector(ctx, query, p)
	}()
	go func() { defer reads.Done(); a.checkPrometheusCoverage(probeCtx, probe) }()
	reads.Wait()
	coverage := Coverage{Reason: "coverage_unverified"}
	if probe.audit.coverage != nil {
		coverage = *probe.audit.coverage
	}
	for i := range items {
		row := &items[i]
		r := observation[RequestCounts](p, "prometheus", "model_requests", coverage, "model_code")
		if requests.state != "ok" {
			r = missingObservation(r, requests.state, requests.reason)
		} else if value := requests.values[row.ModelCode]; value != nil {
			r.Value = value
			if coverage.Start == nil {
				r.State, r.Reason = "partial", "coverage_unverified"
			}
		} else {
			r = missingObservation(r, "no_samples", "no_source_samples")
		}
		row.Requests, row.SuccessRate = r, modelRate(r)
		metric := func(index int, factor float64) MetricObservation[float64] {
			m := observation[float64](p, "prometheus", "model_requests", coverage, "model_code")
			v := vectors[index]
			if v.state != "ok" {
				return missingObservation(m, v.state, v.reason)
			}
			value := v.values[row.ModelCode]
			if value == nil {
				return missingObservation(m, "no_samples", "no_source_samples")
			}
			converted := *value * factor
			if math.IsNaN(converted) || math.IsInf(converted, 0) {
				return missingObservation(m, "error", "invalid_source_value")
			}
			m.Value = &converted
			if coverage.Start == nil {
				m.State, m.Reason = "partial", "coverage_unverified"
			}
			return m
		}
		row.AvgLatencyMs, row.P50LatencyMs, row.P95LatencyMs, row.P99LatencyMs = metric(0, 1000), metric(1, 1000), metric(2, 1000), metric(3, 1000)
		row.AvgTTFTMs, row.P50TTFTMs, row.P95TTFTMs, row.P99TTFTMs = metric(4, 1000), metric(5, 1000), metric(6, 1000), metric(7, 1000)
		row.TotalTokens, row.TotalCost, row.Otps = integerObservation(metric(8, 1)), metric(9, 1), metric(10, 1)
	}
}

package monitor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
)

// getOverview preserves live QPS, daily totals and rolling latency as distinct
// measurements. Missing values stay nullable; one failed source does not erase
// successful measurements from another source.
func (a *Reader) getOverview(ctx context.Context) (*Overview, error) {
	var now time.Time
	if s := scope(ctx); s != nil {
		now = s.plan.Cutoff
	} else {
		now = a.now()
	}
	res := &Overview{}
	prom := a.isPrometheusAvailable(ctx)
	if prom {
		res.QPS = a.prometheusScalar(ctx, fmt.Sprintf(`sum(rate(%s[5m]))`, mRequestTotal))
	}
	if res.QPS == nil {
		if s := scope(ctx); s != nil {
			s.qpsFallback = true
		}
		minute := now.Unix()/60 - 1
		if a.redis != nil {
			source(ctx, "redis")
			vals, err := a.redisValues(ctx, gatewaycontract.Keys.Status.Global(minute, gatewaycontract.MetricSuccess), gatewaycontract.Keys.Status.Global(minute, gatewaycontract.MetricFailure))
			if err == nil {
				v := float64(parseMetricInt(vals[0])+parseMetricInt(vals[1])) / 60
				res.QPS = &v
			}
		} else {
			source(ctx, "memory")
			if !time.Unix(minute*60, 0).Before(a.memory().StartedAt()) {
				v := float64(a.memory().GetGlobalMinutePerf(minute).Requests) / 60
				res.QPS = &v
			}
		}
	}
	// Daily keys are written in the gateway/server's local calendar. A different
	// midnight cannot be inferred from those aggregates; expose that gap explicitly.
	local := now.In(time.Local)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	localMidnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
	if midnight.Equal(localMidnight) {
		if a.redis != nil {
			source(ctx, "redis")
			date := now.Format("2006-01-02")
			vals, err := a.redisValues(ctx,
				gatewaycontract.Keys.Status.Daily(gatewaycontract.DailyRequests, date),
				gatewaycontract.Keys.Status.Daily(gatewaycontract.DailyInputTokens, date),
				gatewaycontract.Keys.Status.Daily(gatewaycontract.DailyOutputTokens, date),
				gatewaycontract.Keys.Status.Daily(gatewaycontract.DailyCachedTokens, date),
				gatewaycontract.Keys.Status.Daily(gatewaycontract.DailyCacheCreationTokens, date),
				gatewaycontract.Keys.Status.Daily(gatewaycontract.DailyCost, date))
			if err == nil {
				req, input, output, cached, creation := parseMetricInt(vals[0]), parseMetricInt(vals[1]), parseMetricInt(vals[2]), parseMetricInt(vals[3]), parseMetricInt(vals[4])
				cost := 0.0
				if vals[5] != nil {
					cost, _ = strconv.ParseFloat(fmt.Sprint(vals[5]), 64)
				}
				res.DailyRequests, res.DailyPromptTokens, res.DailyCompletionTokens = &req, &input, &output
				res.DailyCachedTokens, res.DailyCacheCreationTokens, res.DailyCost = &cached, &creation, &cost
			}
		} else {
			source(ctx, "memory")
			stats := a.memory().GetDailyStats(now.Format("2006-01-02"))
			res.DailyRequests, res.DailyPromptTokens, res.DailyCompletionTokens = &stats.ReqCount, &stats.InputTokens, &stats.OutputTokens
			res.DailyCachedTokens, res.DailyCacheCreationTokens, res.DailyCost = &stats.CachedTokens, &stats.CacheCreationTokens, &stats.Cost
		}
	}
	if prom {
		res.AvgLatencyMs = a.prometheusScalar(ctx, fmt.Sprintf(`sum(rate(%s[5m])) / sum(rate(%s[5m]))`, mRequestDurationSum, mRequestDurationCount))
		res.AvgTTFTMs = a.prometheusScalar(ctx, fmt.Sprintf(`sum(rate(%s[5m])) / sum(rate(%s[5m]))`, mTtftSum, mTtftCount))
		if res.AvgLatencyMs != nil {
			*res.AvgLatencyMs *= 1000
		}
		if res.AvgTTFTMs != nil {
			*res.AvgTTFTMs *= 1000
		}
	} else if a.redis == nil {
		source(ctx, "memory")
		end := now.Unix() / 60
		perf := a.memory().AggregateGlobalMinutePerf(end-4, end)
		if perf.LatencyCount > 0 {
			v := float64(perf.LatencySumMs) / float64(perf.LatencyCount)
			res.AvgLatencyMs = &v
		}
		if perf.TTFTCount > 0 {
			v := float64(perf.TTFTSum) / float64(perf.TTFTCount)
			res.AvgTTFTMs = &v
		}
	}
	res.ActiveCircuitBreakers = a.getCircuitBreakers(ctx)
	return res, nil
}

func (a *Reader) getCircuitBreakers(ctx context.Context) []CircuitBreakerInfo {
	var endpoints []string
	var services []string
	if a.redis != nil {
		source(ctx, "redis")
		endpoints, _ = a.redis.SMembers(ctx, gatewaycontract.Keys.Circuit.OpenEndpoints).Result()
		services, _ = a.redis.SMembers(ctx, gatewaycontract.Keys.Circuit.OpenServices).Result()
	} else {
		endpoints = a.memory().GetOpenEndpoints()
		services = a.memory().GetOpenServices()
	}

	var endpointInfos []CircuitBreakerInfo
	if len(endpoints) > 0 && a.db != nil {
		var dbEndpoints []rschema.Endpoint
		err := a.db.WithContext(ctx).
			Preload("Model").
			Preload("Provider").
			Where("id IN ?", endpoints).
			Find(&dbEndpoints).Error
		if err == nil {
			for _, ep := range dbEndpoints {
				info := CircuitBreakerInfo{
					ID:   ep.ID,
					Type: "endpoint",
					URL:  ep.URL,
				}
				if ep.Model != nil {
					info.ModelID = ep.ModelID
					info.ModelName = ep.Model.ModelName
				}
				if ep.Provider != nil {
					info.ProviderID = ep.ProviderID
					info.ProviderName = ep.Provider.Name
				}
				// 组合 Name
				if info.ModelName != "" && info.ProviderName != "" {
					info.Name = fmt.Sprintf("%s (%s)", info.ModelName, info.ProviderName)
				} else if info.ModelName != "" {
					info.Name = info.ModelName
				} else {
					info.Name = ep.URL
				}
				endpointInfos = append(endpointInfos, info)
			}
		}

		// 兜底填充：如果某些端点被逻辑删除（但在 Redis 缓存状态里仍然存活），则使用 ID 填补
		foundMap := make(map[string]bool)
		for _, info := range endpointInfos {
			foundMap[info.ID] = true
		}
		for _, id := range endpoints {
			if !foundMap[id] {
				endpointInfos = append(endpointInfos, CircuitBreakerInfo{
					ID:   id,
					Type: "endpoint",
					Name: id,
				})
			}
		}
	}

	var serviceInfos []CircuitBreakerInfo
	for _, s := range services {
		serviceInfos = append(serviceInfos, CircuitBreakerInfo{
			ID:   s,
			Type: "service",
			Name: s,
		})
	}

	allInfos := make([]CircuitBreakerInfo, 0, len(endpointInfos)+len(serviceInfos))
	allInfos = append(allInfos, endpointInfos...)
	allInfos = append(allInfos, serviceInfos...)

	return allInfos
}

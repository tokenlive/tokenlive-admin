package monitor

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
)

func buildModelTokensQuery(promRange string) string {
	return fmt.Sprintf(
		`sum by (model) (increase(%s{type=~"input|output"}[%s]))`,
		mTokensTotal,
		promRange,
	)
}

func buildModelOtpsQuery(promRange string) string {
	return fmt.Sprintf(
		`sum by (model) (increase(%s{type="output"}[%s])) / sum by (model) (increase(%s[%s]))`,
		mTokensTotal,
		promRange,
		mRequestDurationSum,
		promRange,
	)
}

func (a *Reader) getModelRanking(ctx context.Context, sortBy, timeRange string, limit int, modelFilter string) ([]ModelRankingItem, error) {
	return a.getModelRankingAt(ctx, sortBy, timeRange, limit, modelFilter, a.now())
}

func (a *Reader) getModelRankingAt(ctx context.Context, sortBy, timeRange string, limit int, modelFilter string, end time.Time) ([]ModelRankingItem, error) {
	ctx, plan, planErr := a.queryPlan(ctx, timeRange, end)
	if planErr != nil {
		return nil, planErr
	}
	if plan.minutes() == 0 {
		source(ctx, "catalog")
		return []ModelRankingItem{}, nil
	}
	end = plan.End
	cacheKey := fmt.Sprintf("ranking:%s:%s:%d:%s:%d", sortBy, timeRange, limit, modelFilter, end.Unix())
	if cached, ok := a.getCache(ctx, cacheKey); ok {
		return cached.([]ModelRankingItem), nil
	}

	// 根据 time_range 计算 PromQL 范围和 Redis 窗口
	promRange, redisMinutes := fmt.Sprintf("%ds", int64(plan.End.Sub(plan.Start)/time.Second)), min(plan.minutes(), 120)

	// 1. 从数据库查询启用的模型；指定 model 时只取这一行
	var models []rschema.Model
	if a.db == nil {
		return nil, fmt.Errorf("model catalog is unavailable")
	}
	query := a.db.WithContext(ctx).Where("enabled = ?", 1)
	if modelFilter != "" {
		query = query.Where("id = ? OR model_code = ?", modelFilter, modelFilter)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	if len(models) == 0 {
		source(ctx, "catalog")
		a.setCache(ctx, cacheKey, []ModelRankingItem{}, 5*time.Second)
		return []ModelRankingItem{}, nil
	}

	// 2. 初始化 items
	items := make([]ModelRankingItem, len(models))
	for i, m := range models {
		items[i] = ModelRankingItem{
			ModelID:   m.ID,
			ModelCode: m.ModelCode,
			ModelName: m.ModelName,
		}
	}

	// 3. 尝试从 Prometheus 获取数据
	if a.isPrometheusAvailable(ctx) {
		source(ctx, "prometheus")
		queryValues := func(query, label string) map[string]float64 {
			return a.queryPrometheusMultiValuesAt(ctx, query, label, end)
		}
		// 查询请求计数
		successQuery := fmt.Sprintf(`sum by (model) (increase(%s{status="success"}[%s]))`, mRequestTotal, promRange)
		errorQuery := fmt.Sprintf(`sum by (model) (increase(%s{status="error"}[%s]))`, mRequestTotal, promRange)

		successMap := queryValues(successQuery, "model")
		errorMap := queryValues(errorQuery, "model")

		// 查询延迟指标
		avgLatencyMap := queryValues(fmt.Sprintf(`sum by (model) (rate(%s[%s])) / sum by (model) (rate(%s[%s]))`, mRequestDurationSum, promRange, mRequestDurationCount, promRange), "model")
		p50LatencyMap := queryValues(fmt.Sprintf(`histogram_quantile(0.50, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, promRange), "model")
		p95LatencyMap := queryValues(fmt.Sprintf(`histogram_quantile(0.95, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, promRange), "model")
		p99LatencyMap := queryValues(fmt.Sprintf(`histogram_quantile(0.99, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, promRange), "model")

		// 查询 TTFT 指标
		avgTTFTMap := queryValues(fmt.Sprintf(`sum by (model) (rate(%s[%s])) / sum by (model) (rate(%s[%s]))`, mTtftSum, promRange, mTtftCount, promRange), "model")
		p50TTFTMap := queryValues(fmt.Sprintf(`histogram_quantile(0.50, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, promRange), "model")
		p95TTFTMap := queryValues(fmt.Sprintf(`histogram_quantile(0.95, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, promRange), "model")
		p99TTFTMap := queryValues(fmt.Sprintf(`histogram_quantile(0.99, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, promRange), "model")

		// 查询 Token、OTPS 和费用
		tokensMap := queryValues(buildModelTokensQuery(promRange), "model")
		otpsMap := queryValues(buildModelOtpsQuery(promRange), "model")
		costMap := queryValues(fmt.Sprintf(`sum by (model) (increase(%s[%s]))`, mCostTotal, promRange), "model")

		// 填充数据
		for i, m := range models {
			code := m.ModelCode
			items[i].SuccessCount = int64(successMap[code])
			items[i].FailCount = int64(errorMap[code])
			items[i].RequestCount = items[i].SuccessCount + items[i].FailCount

			if items[i].RequestCount > 0 {
				items[i].SuccessRate = float64(items[i].SuccessCount) / float64(items[i].RequestCount) * 100
			}

			// 延迟转换为毫秒
			items[i].AvgLatencyMs = avgLatencyMap[code] * 1000
			items[i].P50LatencyMs = p50LatencyMap[code] * 1000
			items[i].P95LatencyMs = p95LatencyMap[code] * 1000
			items[i].P99LatencyMs = p99LatencyMap[code] * 1000

			// TTFT 转换为毫秒
			items[i].AvgTTFTMs = avgTTFTMap[code] * 1000
			items[i].P50TTFTMs = p50TTFTMap[code] * 1000
			items[i].P95TTFTMs = p95TTFTMap[code] * 1000
			items[i].P99TTFTMs = p99TTFTMap[code] * 1000

			items[i].TotalTokens = int64(tokensMap[code])
			items[i].Otps = otpsMap[code]
			items[i].TotalCost = costMap[code]
		}
	} else if a.redis == nil {
		source(ctx, "memory")
		currentMin := end.Add(-time.Minute).Unix() / 60
		numMinutes := plan.minutes()
		startMinute := currentMin - int64(numMinutes-1)
		for i, model := range models {
			perf := a.memory().AggregateModelMinutePerf(model.ModelCode, startMinute, currentMin)
			items[i].SuccessCount = perf.Success
			items[i].FailCount = perf.Fail
			items[i].RequestCount = perf.Requests
			availabilityCount := perf.Success + perf.Fail
			if availabilityCount > 0 {
				items[i].SuccessRate = float64(perf.Success) / float64(availabilityCount) * 100
			}
			if perf.LatencyCount > 0 {
				items[i].AvgLatencyMs = float64(perf.LatencySumMs) / float64(perf.LatencyCount)
			}
			if perf.TTFTCount > 0 {
				items[i].AvgTTFTMs = float64(perf.TTFTSum) / float64(perf.TTFTCount)
			}
			items[i].TotalTokens = perf.InputTokens + perf.OutputTokens
			items[i].TotalCost = perf.Cost
			if perf.Output > 0 && perf.DurationMs > 0 {
				items[i].Otps = float64(perf.Output) / (float64(perf.DurationMs) / 1000)
			}
		}
	} else if redisMinutes > 0 {
		var values []interface{}
		var err error
		currentMin := end.Add(-time.Minute).Unix() / 60
		numMinutes := redisMinutes

		if a.redis != nil {
			source(ctx, "redis")
			keys := make([]string, 0, len(models)*numMinutes*2)
			for _, m := range models {
				for i := 0; i < numMinutes; i++ {
					minute := currentMin - int64(numMinutes-1-i)
					keys = append(keys,
						gatewaycontract.Keys.Status.Model(m.ModelCode, minute, gatewaycontract.MetricSuccess),
						gatewaycontract.Keys.Status.Model(m.ModelCode, minute, gatewaycontract.MetricFailure),
					)
				}
			}

			values, err = a.redisValues(ctx, keys...)
		} else {
			values = make([]interface{}, len(models)*numMinutes*2)
			idx := 0
			for _, m := range models {
				for i := 0; i < numMinutes; i++ {
					minute := currentMin - int64(numMinutes-1-i)
					succ, fail := a.memory().GetModelStatus(m.ModelCode, minute)
					if succ > 0 {
						values[idx] = strconv.FormatInt(succ, 10)
					}
					if fail > 0 {
						values[idx+1] = strconv.FormatInt(fail, 10)
					}
					idx += 2
				}
			}
		}
		if err == nil {
			idx := 0
			for i := range models {
				var succ, fail int64
				for j := 0; j < numMinutes; j++ {
					if values != nil {
						if sVal := values[idx]; sVal != nil {
							if sStr, ok := sVal.(string); ok {
								succ += mustParseInt(sStr)
							}
						}
						if fVal := values[idx+1]; fVal != nil {
							if fStr, ok := fVal.(string); ok {
								fail += mustParseInt(fStr)
							}
						}
					}
					idx += 2
				}
				items[i].SuccessCount = succ
				items[i].FailCount = fail
				items[i].RequestCount = succ + fail
				if items[i].RequestCount > 0 {
					items[i].SuccessRate = float64(succ) / float64(items[i].RequestCount) * 100
				}
			}
		}
	}

	// 4. 过滤掉没有请求的模型
	filtered := make([]ModelRankingItem, 0, len(items))
	for _, item := range items {
		if item.RequestCount > 0 {
			filtered = append(filtered, item)
		}
	}

	// 5. 排序
	sortModelRankingItems(filtered, sortBy)

	// 6. 限制结果数量。单模型查询需要完整这一行，不受 Top N 截断。
	if modelFilter == "" && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	a.setCache(ctx, cacheKey, filtered, 5*time.Second)
	return filtered, nil
}

func sortModelRankingItems(items []ModelRankingItem, sortBy string) {
	switch sortBy {
	case "avg_latency":
		sort.Slice(items, func(i, j int) bool {
			return items[i].AvgLatencyMs < items[j].AvgLatencyMs
		})
	case "avg_ttft":
		sort.Slice(items, func(i, j int) bool {
			return items[i].AvgTTFTMs < items[j].AvgTTFTMs
		})
	case "tokens":
		sort.Slice(items, func(i, j int) bool {
			return items[i].TotalTokens > items[j].TotalTokens
		})
	case "cost":
		sort.Slice(items, func(i, j int) bool {
			return items[i].TotalCost > items[j].TotalCost
		})
	case "success_rate":
		sort.Slice(items, func(i, j int) bool {
			return items[i].SuccessRate > items[j].SuccessRate
		})
	case "otps":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Otps > items[j].Otps
		})
	default: // request_count
		sort.Slice(items, func(i, j int) bool {
			return items[i].RequestCount > items[j].RequestCount
		})
	}
}

// resolveTimeRange 将前端传入的 time_range 参数转换为 PromQL 范围字符串和 Redis 窗口分钟数
// promRange: 用于 PromQL 查询的范围（如 "60m", "6h", "24h", "7d"）
// redisMinutes: Redis 降级时查询的分钟数（0 表示 Redis 无法覆盖该范围）

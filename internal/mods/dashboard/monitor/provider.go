package monitor

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

// providerMetricsMaxRange 供应商监控的时间范围上限。
//
// 内存 store 的分钟级指标只保留 7 天（pkg/metrics.minuteMetricRetention），
// 而 Prometheus 通常保留数月。限制在 7 天内可让两条数据源路径行为一致，
// 避免降级时静默返回不完整数据。详见 ADR-0008。
const providerMetricsMaxRange = 7 * 24 * time.Hour

// providerRedisMaxMinutes Redis 状态键的 TTL 为 2 小时（ADR-0009），
// 超出该窗口的查询无法由 Redis 覆盖，需降级到内存 store。
const providerRedisMaxMinutes = 120

// hasProviderLabel 探测 Prometheus 的 request_total 指标是否带 provider label。
//
// Gateway 的埋点无法从 Admin 仓库验证。若直接假定 label 存在并发起
// `sum by (provider)` 查询，label 缺失时会静默返回空结果——接口通了但数据全是空值，
// 这种失败方式非常隐蔽。因此先探测，无 label 则降级到内存 store。结果缓存 30 秒。
func (a *Reader) hasProviderLabel(ctx context.Context) bool {
	a.providerLabelMu.RLock()
	if time.Since(a.providerLabelChk) < 30*time.Second {
		ok := a.providerLabelOK
		a.providerLabelMu.RUnlock()
		return ok
	}
	a.providerLabelMu.RUnlock()

	a.providerLabelMu.Lock()
	defer a.providerLabelMu.Unlock()

	if time.Since(a.providerLabelChk) < 30*time.Second {
		return a.providerLabelOK
	}

	probe := fmt.Sprintf(`count by (provider) (%s)`, mRequestTotal)
	values := a.queryPrometheusMultiValues(ctx, probe, "provider")

	a.providerLabelOK = len(values) > 0
	a.providerLabelChk = time.Now()
	return a.providerLabelOK
}

// providerStoreKeys 返回可能被网关用作指标 key 的候选标识。
//
// ADR-0009 已确立以 Provider 的 code 为端到端主标识，因此 code 优先。
// 仍保留 name 兜底，用于读取网关完成 provider_code 穿透之前写入的历史数据。
func providerStoreKeys(p *rschema.Provider) []string {
	keys := make([]string, 0, 2)
	if p.Code != "" {
		keys = append(keys, p.Code)
	}
	if p.Name != "" && p.Name != p.Code {
		keys = append(keys, p.Name)
	}
	return keys
}

// pickProviderMetric 按候选 key 依次取值，返回首个非零结果。
func pickProviderMetric(values map[string]float64, keys []string) float64 {
	for _, k := range keys {
		if v, ok := values[k]; ok && v != 0 {
			return v
		}
	}
	return 0
}

func (a *Reader) getProviderRankingAt(
	ctx context.Context,
	sortBy, timeRange string,
	limit int,
	providerFilter string,
	end time.Time,
) ([]ProviderRankingItem, error) {
	ctx, plan, planErr := a.queryPlan(ctx, timeRange, end)
	if planErr != nil {
		return nil, planErr
	}
	if plan.minutes() == 0 {
		source(ctx, "catalog")
		return []ProviderRankingItem{}, nil
	}
	end = plan.End
	cacheKey := fmt.Sprintf("provider-ranking:%s:%s:%d:%s:%d", sortBy, timeRange, limit, providerFilter, end.Unix())
	if cached, ok := a.getCache(ctx, cacheKey); ok {
		return cached.([]ProviderRankingItem), nil
	}

	promRange, minutes := fmt.Sprintf("%ds", int64(plan.End.Sub(plan.Start)/time.Second)), plan.minutes()

	// 1. 查询供应商。指定 provider 时按 id 或 code 精确匹配单行。
	var providers []rschema.Provider
	if a.db == nil {
		return nil, fmt.Errorf("provider catalog is unavailable")
	}
	query := a.db.WithContext(ctx)
	if providerFilter != "" {
		query = query.Where("id = ? OR code = ?", providerFilter, providerFilter)
	} else {
		query = query.Where("enabled = ?", 1)
	}
	if err := query.Find(&providers).Error; err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		source(ctx, "catalog")
		a.setCache(ctx, cacheKey, []ProviderRankingItem{}, 5*time.Second)
		return []ProviderRankingItem{}, nil
	}

	items := make([]ProviderRankingItem, len(providers))
	storeKeys := make([][]string, len(providers))
	for i := range providers {
		p := &providers[i]
		items[i] = ProviderRankingItem{
			ProviderID:   p.ID,
			ProviderCode: p.Code,
			ProviderName: p.Name,
		}
		storeKeys[i] = providerStoreKeys(p)
	}

	// 2. 端点数与熔断数：来自 DB 与运行时缓存，与指标源无关。
	a.fillProviderEndpointStats(ctx, items)

	// 3. 指标：Prometheus（需 provider label）→ Redis → 内存 store。
	//    Redis 层只有成功/失败计数（ADR-0009 的 aigw:status:provider:{code} 键），
	//    没有 TTFT/token/费用，且 TTL 仅 2 小时，故只在 2 小时内的窗口使用。
	switch {
	case a.isPrometheusAvailable(ctx) && a.hasProviderLabel(ctx):
		source(ctx, "prometheus")
		a.fillProviderMetricsFromPrometheus(ctx, items, storeKeys, promRange, end)
	case a.redis != nil && minutes <= providerRedisMaxMinutes:
		source(ctx, "redis")
		a.fillProviderStatusFromRedis(ctx, items, storeKeys, minutes, end)
	default:
		source(ctx, "memory")
		a.fillProviderMetricsFromMemory(items, storeKeys, minutes, end)
	}

	// 4. 过滤无流量的供应商。单供应商查询需要保留该行，即使流量为零——
	//    详情页要能展示"这家供应商这段时间没有调用"，而不是空响应。
	filtered := items
	if providerFilter == "" {
		filtered = make([]ProviderRankingItem, 0, len(items))
		for _, item := range items {
			if item.UpstreamCallCount > 0 {
				filtered = append(filtered, item)
			}
		}
	}

	sortProviderRankingItems(filtered, sortBy)

	if providerFilter == "" && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	a.setCache(ctx, cacheKey, filtered, 5*time.Second)
	return filtered, nil
}

// fillProviderEndpointStats 填充每个供应商的端点总数与当前熔断端点数。
func (a *Reader) fillProviderEndpointStats(ctx context.Context, items []ProviderRankingItem) {
	providerIDs := make([]string, 0, len(items))
	indexByID := make(map[string]int, len(items))
	for i, item := range items {
		providerIDs = append(providerIDs, item.ProviderID)
		indexByID[item.ProviderID] = i
	}

	var endpoints []rschema.Endpoint
	if err := a.db.WithContext(ctx).
		Select("id", "provider_id").
		Where("provider_id IN ?", providerIDs).
		Find(&endpoints).Error; err != nil {
		return
	}

	openSet := make(map[string]bool)
	for _, info := range a.getCircuitBreakers(ctx) {
		if info.Type == "endpoint" {
			openSet[info.ID] = true
		}
	}

	for _, ep := range endpoints {
		idx, ok := indexByID[ep.ProviderID]
		if !ok {
			continue
		}
		items[idx].EndpointCount++
		if openSet[ep.ID] {
			items[idx].OpenBreakerCount++
		}
	}
}

func (a *Reader) fillProviderMetricsFromPrometheus(ctx context.Context,
	items []ProviderRankingItem,
	storeKeys [][]string,
	promRange string,
	end time.Time,
) {
	queryValues := func(query string) map[string]float64 {
		return a.queryPrometheusMultiValuesAt(ctx, query, "provider", end)
	}

	successMap := queryValues(fmt.Sprintf(
		`sum by (provider) (increase(%s{status="success"}[%s]))`, mRequestTotal, promRange))
	errorMap := queryValues(fmt.Sprintf(
		`sum by (provider) (increase(%s{status="error"}[%s]))`, mRequestTotal, promRange))
	avgLatencyMap := queryValues(fmt.Sprintf(
		`sum by (provider) (rate(%s[%s])) / sum by (provider) (rate(%s[%s]))`,
		mRequestDurationSum, promRange, mRequestDurationCount, promRange))
	avgTTFTMap := queryValues(fmt.Sprintf(
		`sum by (provider) (rate(%s[%s])) / sum by (provider) (rate(%s[%s]))`,
		mTtftSum, promRange, mTtftCount, promRange))
	tokensMap := queryValues(fmt.Sprintf(
		`sum by (provider) (increase(%s{type=~"input|output"}[%s]))`, mTokensTotal, promRange))
	otpsMap := queryValues(fmt.Sprintf(
		`sum by (provider) (increase(%s{type="output"}[%s])) / sum by (provider) (increase(%s[%s]))`,
		mTokensTotal, promRange, mRequestDurationSum, promRange))
	costMap := queryValues(fmt.Sprintf(
		`sum by (provider) (increase(%s[%s]))`, mCostTotal, promRange))

	for i := range items {
		keys := storeKeys[i]
		items[i].SuccessCount = int64(pickProviderMetric(successMap, keys))
		items[i].FailCount = int64(pickProviderMetric(errorMap, keys))
		items[i].UpstreamCallCount = items[i].SuccessCount + items[i].FailCount
		if items[i].UpstreamCallCount > 0 {
			items[i].UpstreamCallSuccessRate =
				float64(items[i].SuccessCount) / float64(items[i].UpstreamCallCount) * 100
		}
		// Prometheus 的延迟与 TTFT 以秒为单位，转换为毫秒。
		items[i].AvgLatencyMs = pickProviderMetric(avgLatencyMap, keys) * 1000
		items[i].AvgTTFTMs = pickProviderMetric(avgTTFTMap, keys) * 1000
		items[i].TotalTokens = int64(pickProviderMetric(tokensMap, keys))
		items[i].Otps = pickProviderMetric(otpsMap, keys)
		items[i].TotalCost = pickProviderMetric(costMap, keys)
	}
}

// redisMGetBatchSize 单次 MGet 的键数上限。7 天 × 每分钟 2 键 × 多个候选标识
// 很容易堆到上万个键，一次性发出会撑爆单条命令。
const redisMGetBatchSize = 500

// mgetInBatches 分批执行 MGet 并按原顺序拼接结果。
func mgetInBatches(ctx context.Context, client *redis.Client, keys []string) ([]interface{}, error) {
	values := make([]interface{}, 0, len(keys))
	for start := 0; start < len(keys); start += redisMGetBatchSize {
		stop := start + redisMGetBatchSize
		if stop > len(keys) {
			stop = len(keys)
		}
		batch, err := client.MGet(ctx, keys[start:stop]...).Result()
		if err != nil {
			return nil, err
		}
		values = append(values, batch...)
	}
	return values, nil
}

// fillProviderStatusFromRedis 从 ADR-0009 的 aigw:status:provider:{code}:{minute}:[s|f]
// 键读取成功/失败计数。该层只有计数，没有 TTFT、token 与费用——这些指标在此路径下留零值。
func (a *Reader) fillProviderStatusFromRedis(
	ctx context.Context,
	items []ProviderRankingItem,
	storeKeys [][]string,
	minutes int,
	end time.Time,
) {
	currentMinute := end.Add(-time.Minute).Unix() / 60

	// 每个供应商的每个候选 key 都要取 minutes 个分钟的成功与失败两个键。
	keys := make([]string, 0, len(items)*2*minutes)
	for i := range items {
		for _, storeKey := range storeKeys[i] {
			for m := 0; m < minutes; m++ {
				minute := currentMinute - int64(minutes-1-m)
				keys = append(keys,
					gatewaycontract.Keys.Status.Provider(storeKey, minute, gatewaycontract.MetricSuccess),
					gatewaycontract.Keys.Status.Provider(storeKey, minute, gatewaycontract.MetricFailure),
				)
			}
		}
	}
	if len(keys) == 0 {
		return
	}

	values, err := mgetInBatches(ctx, a.redis, keys)
	if err != nil {
		// Redis 不可用时回退到内存 store，而非留下一片空白。
		source(ctx, "memory")
		a.fillProviderMetricsFromMemory(items, storeKeys, minutes, end)
		return
	}

	idx := 0
	for i := range items {
		var success, fail int64
		for range storeKeys[i] {
			var keySuccess, keyFail int64
			for m := 0; m < minutes; m++ {
				if sVal := values[idx]; sVal != nil {
					if sStr, ok := sVal.(string); ok {
						keySuccess += mustParseInt(sStr)
					}
				}
				if fVal := values[idx+1]; fVal != nil {
					if fStr, ok := fVal.(string); ok {
						keyFail += mustParseInt(fStr)
					}
				}
				idx += 2
			}
			// 候选 key 按 code 优先排列，取首个有数据的，避免 code 与 name
			// 同时存在数据时重复累加。
			if success == 0 && fail == 0 {
				success, fail = keySuccess, keyFail
			}
		}

		items[i].SuccessCount = success
		items[i].FailCount = fail
		items[i].UpstreamCallCount = success + fail
		if items[i].UpstreamCallCount > 0 {
			items[i].UpstreamCallSuccessRate =
				float64(success) / float64(items[i].UpstreamCallCount) * 100
		}
	}
}

func (a *Reader) fillProviderMetricsFromMemory(
	items []ProviderRankingItem,
	storeKeys [][]string,
	minutes int,
	end time.Time,
) {
	currentMinute := end.Add(-time.Minute).Unix() / 60
	startMinute := currentMinute - int64(minutes-1)

	for i := range items {
		var perf metrics.EndpointMinutePerf
		// 按候选 key 依次尝试，取首个有数据的聚合结果。
		for _, key := range storeKeys[i] {
			perf = a.memory().AggregateProviderMinutePerf(key, startMinute, currentMinute)
			if perf.Requests > 0 || perf.Success > 0 || perf.Fail > 0 {
				break
			}
		}

		items[i].SuccessCount = perf.Success
		items[i].FailCount = perf.Fail
		items[i].UpstreamCallCount = perf.Success + perf.Fail
		if items[i].UpstreamCallCount > 0 {
			items[i].UpstreamCallSuccessRate =
				float64(perf.Success) / float64(items[i].UpstreamCallCount) * 100
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
}

func sortProviderRankingItems(items []ProviderRankingItem, sortBy string) {
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
			return items[i].UpstreamCallSuccessRate > items[j].UpstreamCallSuccessRate
		})
	case "otps":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Otps > items[j].Otps
		})
	default: // request_count
		sort.Slice(items, func(i, j int) bool {
			return items[i].UpstreamCallCount > items[j].UpstreamCallCount
		})
	}
}

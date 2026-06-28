package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

type CircuitBreakerInfo struct {
	ID           string `json:"id"`
	Type         string `json:"type"`          // "endpoint" 或 "service"
	Name         string `json:"name"`          // 显示名称
	ModelID      string `json:"model_id"`      // 关联模型 ID
	ModelName    string `json:"model_name"`    // 关联模型名称
	ProviderID   string `json:"provider_id"`   // 关联供应商 ID
	ProviderName string `json:"provider_name"` // 关联供应商名称
	URL          string `json:"url"`           // 关联的 URL 地址
}

type cacheVal struct {
	data      interface{}
	expiredAt time.Time
}

type Dashboard struct {
	DB                *gorm.DB
	RedisClient       *redis.Client
	RedisSync         *biz.ConfigRedisSync
	prometheusAvail   bool
	prometheusMu      sync.RWMutex
	prometheusLastChk time.Time
	cacheMu           sync.RWMutex
	cacheMap          map[string]cacheVal
}

func (a *Dashboard) getCache(key string) (interface{}, bool) {
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()
	if a.cacheMap == nil {
		return nil, false
	}
	val, ok := a.cacheMap[key]
	if !ok || time.Now().After(val.expiredAt) {
		return nil, false
	}
	return val.data, true
}

func (a *Dashboard) setCache(key string, data interface{}, ttl time.Duration) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	if a.cacheMap == nil {
		a.cacheMap = make(map[string]cacheVal)
	}
	a.cacheMap[key] = cacheVal{
		data:      data,
		expiredAt: time.Now().Add(ttl),
	}
}

type prometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"` // [timestamp, valueString]
		} `json:"result"`
	} `json:"data"`
}

type prometheusQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"` // [[timestamp, valueString], ...]
		} `json:"result"`
	} `json:"data"`
}

// metricName 根据配置的 MetricPrefix 拼接完整的 Prometheus 指标名
// 例如：metricName("request_total") → "aigateway_gateway_request_total"（当 MetricPrefix="aigateway_gateway_"）
func metricName(suffix string) string {
	prefix := config.C.Util.PrometheusServer.MetricPrefix
	if prefix == "" {
		prefix = "gateway_"
	}
	return prefix + suffix
}

// PromQL 指标名（根据 MetricPrefix 配置动态生成）
var (
	mRequestTotal          = metricName("request_total")
	mRequestDurationSum    = metricName("request_duration_seconds_sum")
	mRequestDurationCount  = metricName("request_duration_seconds_count")
	mRequestDurationBucket = metricName("request_duration_seconds_bucket")
	mTtftSum               = metricName("ttft_seconds_sum")
	mTtftCount             = metricName("ttft_seconds_count")
	mTtftBucket            = metricName("ttft_seconds_bucket")
	mTokensTotal           = metricName("tokens_total")
	mCostTotal             = metricName("cost_total")
)

// isPrometheusAvailable 检查 Prometheus 是否可用，结果缓存 30 秒
func (a *Dashboard) isPrometheusAvailable() bool {
	a.prometheusMu.RLock()
	if time.Since(a.prometheusLastChk) < 30*time.Second {
		avail := a.prometheusAvail
		a.prometheusMu.RUnlock()
		return avail
	}
	a.prometheusMu.RUnlock()

	a.prometheusMu.Lock()
	defer a.prometheusMu.Unlock()

	// 再次检查，避免重复请求
	if time.Since(a.prometheusLastChk) < 30*time.Second {
		return a.prometheusAvail
	}

	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		a.prometheusAvail = false
		a.prometheusLastChk = time.Now()
		return false
	}

	// 尝试访问健康检查端点
	healthURL := promAddr + "/-/healthy"
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		a.prometheusAvail = false
		a.prometheusLastChk = time.Now()
		return false
	}

	username := config.C.Util.PrometheusServer.Username
	password := config.C.Util.PrometheusServer.Password
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		a.prometheusAvail = false
		a.prometheusLastChk = time.Now()
		return false
	}
	defer resp.Body.Close()

	a.prometheusAvail = resp.StatusCode == http.StatusOK
	a.prometheusLastChk = time.Now()
	return a.prometheusAvail
}

// IsPrometheusAvailable 导出的版本，用于测试
func (a *Dashboard) IsPrometheusAvailable() bool {
	return a.isPrometheusAvailable()
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query real-time gateway QPS from Prometheus
// @Success 200 {object} util.ResponseResult{data=float64}
// @Router /api/v1/dashboard/qps [get]
func (a *Dashboard) QueryQPS(c *gin.Context) {
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		// 降级，未配置 Prometheus 地址
		util.ResSuccess(c, 0.0)
		return
	}

	// 构造 QPS PromQL
	query := fmt.Sprintf(`sum(rate(%s[5m]))`, mRequestTotal)
	apiURL := promAddr + "/api/v1/query?query=" + url.QueryEscape(query)

	body, err := a.queryPrometheus(apiURL, 3*time.Second)
	if err != nil {
		// 容错返回 0.0，不抛出异常
		util.ResSuccess(c, 0.0)
		return
	}

	var promResp prometheusQueryResponse
	if err := json.Unmarshal(body, &promResp); err != nil || promResp.Status != "success" || len(promResp.Data.Result) == 0 {
		util.ResSuccess(c, 0.0)
		return
	}

	// 解析返回的最新单点 value 值
	valArr := promResp.Data.Result[0].Value
	if len(valArr) < 2 {
		util.ResSuccess(c, 0.0)
		return
	}

	// 统一用简单直接的 string 转 float 的反序列化方法
	var qpsVal float64
	var valStr string
	valRaw, _ := json.Marshal(valArr[1])
	_ = json.Unmarshal(valRaw, &valStr)
	var valTemp float64
	if err := json.Unmarshal([]byte(`"`+valStr+`"`), &valTemp); err == nil {
		qpsVal = valTemp
	} else {
		_ = json.Unmarshal(valRaw, &qpsVal)
	}

	util.ResSuccess(c, qpsVal)
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query all endpoints that are currently in Open/Isolated state from Redis
// @Success 200 {object} util.ResponseResult{data=[]CircuitBreakerInfo}
// @Router /api/v1/dashboard/circuit-breakers [get]
func (a *Dashboard) QueryCircuitBreakers(c *gin.Context) {
	ctx := c.Request.Context()
	var endpoints []string
	var services []string
	if a.RedisClient != nil {
		endpoints, _ = a.RedisClient.SMembers(ctx, "aigw:cb:open_endpoints").Result()
		services, _ = a.RedisClient.SMembers(ctx, "aigw:cb:open_services").Result()
	} else {
		endpoints = metrics.GlobalStore.GetOpenEndpoints()
		services = metrics.GlobalStore.GetOpenServices()
	}

	var endpointInfos []CircuitBreakerInfo
	if len(endpoints) > 0 {
		var dbEndpoints []rschema.Endpoint
		err := a.DB.WithContext(ctx).
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

	util.ResSuccess(c, allInfos)
}

type OverviewResponse struct {
	QPS                      float64              `json:"qps"`
	DailyRequests            int64                `json:"daily_requests"`
	DailyPromptTokens        int64                `json:"daily_prompt_tokens"`
	DailyCompletionTokens    int64                `json:"daily_completion_tokens"`
	DailyCachedTokens        int64                `json:"daily_cached_tokens"`
	DailyCacheCreationTokens int64                `json:"daily_cache_creation_tokens"`
	DailyCost                float64              `json:"daily_cost"`
	AvgLatencyMs             float64              `json:"avg_latency_ms"`
	AvgTTFTMs                float64              `json:"avg_ttft_ms"`
	ActiveCircuitBreakers    []CircuitBreakerInfo `json:"active_circuit_breakers"`
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query dashboard overview metrics (QPS, daily stats, latency, circuit breakers)
// @Success 200 {object} util.ResponseResult{data=OverviewResponse}
// @Router /api/v1/dashboard/overview [get]
func (a *Dashboard) getOverview(ctx context.Context) (*OverviewResponse, error) {
	cacheKey := "overview"
	if cached, ok := a.getCache(cacheKey); ok {
		return cached.(*OverviewResponse), nil
	}

	var res OverviewResponse

	// 1. 获取 QPS：优先 Prometheus
	qps := a.getQPSFromPrometheus()
	if qps <= 0 {
		if a.RedisClient != nil {
			// 降级：从 Redis 计算 QPS
			minute := time.Now().Unix() / 60
			sKey := fmt.Sprintf("aigw:status:global:%d:s", minute-1)
			fKey := fmt.Sprintf("aigw:status:global:%d:f", minute-1)

			vals, err := a.RedisClient.MGet(ctx, sKey, fKey).Result()
			if err == nil && len(vals) == 2 {
				var succ, fail int64
				if vals[0] != nil {
					succ, _ = strconv.ParseInt(vals[0].(string), 10, 64)
				}
				if vals[1] != nil {
					fail, _ = strconv.ParseInt(vals[1].(string), 10, 64)
				}
				total := succ + fail
				if total <= 0 {
					sKeyCurr := fmt.Sprintf("aigw:status:global:%d:s", minute)
					fKeyCurr := fmt.Sprintf("aigw:status:global:%d:f", minute)
					valsCurr, errCurr := a.RedisClient.MGet(ctx, sKeyCurr, fKeyCurr).Result()
					if errCurr == nil && len(valsCurr) == 2 {
						var succC, failC int64
						if valsCurr[0] != nil {
							succC, _ = strconv.ParseInt(valsCurr[0].(string), 10, 64)
						}
						if valsCurr[1] != nil {
							failC, _ = strconv.ParseInt(valsCurr[1].(string), 10, 64)
						}
						total = succC + failC
					}
				}
				qps = float64(total) / 60.0
			}
		} else {
			// 降级：从内存计算 QPS
			minute := time.Now().Unix() / 60
			succ1, fail1 := metrics.GlobalStore.GetGlobalStatus(minute - 1)
			total := succ1 + fail1
			if total <= 0 {
				succ0, fail0 := metrics.GlobalStore.GetGlobalStatus(minute)
				total = succ0 + fail0
			}
			qps = float64(total) / 60.0
		}
	}
	res.QPS = qps

	// 2. 获取今日自然日累计值 (Redis 或 内存)
	if a.RedisClient != nil {
		dateStr := time.Now().Format("2006-01-02")
		dailyReqKey := fmt.Sprintf("aigw:status:daily:req:%s", dateStr)
		dailyPromptKey := fmt.Sprintf("aigw:status:daily:input_tokens:%s", dateStr)
		dailyCompletionKey := fmt.Sprintf("aigw:status:daily:output_tokens:%s", dateStr)
		dailyCachedKey := fmt.Sprintf("aigw:status:daily:cached_tokens:%s", dateStr)
		dailyCacheCreationKey := fmt.Sprintf("aigw:status:daily:cache_creation_tokens:%s", dateStr)
		dailyCostKey := fmt.Sprintf("aigw:status:daily:cost:%s", dateStr)

		vals, err := a.RedisClient.MGet(ctx, dailyReqKey, dailyPromptKey, dailyCompletionKey, dailyCachedKey, dailyCacheCreationKey, dailyCostKey).Result()
		if err == nil && len(vals) == 6 {
			if vals[0] != nil {
				res.DailyRequests, _ = strconv.ParseInt(vals[0].(string), 10, 64)
			}
			if vals[1] != nil {
				res.DailyPromptTokens, _ = strconv.ParseInt(vals[1].(string), 10, 64)
			}
			if vals[2] != nil {
				res.DailyCompletionTokens, _ = strconv.ParseInt(vals[2].(string), 10, 64)
			}
			if vals[3] != nil {
				res.DailyCachedTokens, _ = strconv.ParseInt(vals[3].(string), 10, 64)
			}
			if vals[4] != nil {
				res.DailyCacheCreationTokens, _ = strconv.ParseInt(vals[4].(string), 10, 64)
			}
			if vals[5] != nil {
				res.DailyCost, _ = strconv.ParseFloat(vals[5].(string), 64)
			}
		}
	} else {
		dateStr := time.Now().Format("2006-01-02")
		stats := metrics.GlobalStore.GetDailyStats(dateStr)
		res.DailyRequests = stats.ReqCount
		res.DailyPromptTokens = stats.InputTokens
		res.DailyCompletionTokens = stats.OutputTokens
		res.DailyCachedTokens = stats.CachedTokens
		res.DailyCacheCreationTokens = stats.CacheCreationTokens
		res.DailyCost = stats.Cost
	}

	// 3. 获取最近 5 分钟的滚动平均延迟与首包延迟 (TTFT)，转换为毫秒
	avgLatency := a.queryPrometheusSingleValue(fmt.Sprintf(`sum(rate(%s[5m])) / sum(rate(%s[5m]))`, mRequestDurationSum, mRequestDurationCount))
	avgTTFT := a.queryPrometheusSingleValue(fmt.Sprintf(`sum(rate(%s[5m])) / sum(rate(%s[5m]))`, mTtftSum, mTtftCount))
	res.AvgLatencyMs = avgLatency * 1000
	res.AvgTTFTMs = avgTTFT * 1000

	// 4. 获取熔断器信息
	res.ActiveCircuitBreakers = a.getCircuitBreakers(ctx)

	a.setCache(cacheKey, &res, 5*time.Second)
	return &res, nil
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query dashboard overview metrics (QPS, daily stats, latency, circuit breakers)
// @Success 200 {object} util.ResponseResult{data=OverviewResponse}
// @Router /api/v1/dashboard/overview [get]
func (a *Dashboard) QueryOverview(c *gin.Context) {
	res, err := a.getOverview(c.Request.Context())
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, res)
}

func (a *Dashboard) getCircuitBreakers(ctx context.Context) []CircuitBreakerInfo {
	var endpoints []string
	var services []string
	if a.RedisClient != nil {
		endpoints, _ = a.RedisClient.SMembers(ctx, "aigw:cb:open_endpoints").Result()
		services, _ = a.RedisClient.SMembers(ctx, "aigw:cb:open_services").Result()
	} else {
		endpoints = metrics.GlobalStore.GetOpenEndpoints()
		services = metrics.GlobalStore.GetOpenServices()
	}

	var endpointInfos []CircuitBreakerInfo
	if len(endpoints) > 0 {
		var dbEndpoints []rschema.Endpoint
		err := a.DB.WithContext(ctx).
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

func (a *Dashboard) queryPrometheusSingleValue(query string) float64 {
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		fmt.Printf("[WARN] PrometheusServer.Address is empty\n")
		return 0.0
	}
	apiURL := promAddr + "/api/v1/query?query=" + url.QueryEscape(query)

	body, err := a.queryPrometheus(apiURL, 2*time.Second)
	if err != nil {
		fmt.Printf("[WARN] Prometheus query failed: %v, URL: %s\n", err, apiURL)
		return 0.0
	}

	var promResp prometheusQueryResponse
	if err := json.Unmarshal(body, &promResp); err != nil || promResp.Status != "success" || len(promResp.Data.Result) == 0 {
		fmt.Printf("[WARN] Prometheus query returned no data: query=%s, status=%s, results=%d\n", query, promResp.Status, len(promResp.Data.Result))
		return 0.0
	}

	valArr := promResp.Data.Result[0].Value
	if len(valArr) < 2 {
		return 0.0
	}

	var valStr string
	switch v := valArr[1].(type) {
	case string:
		valStr = v
	case float64:
		return v
	default:
		valStr = fmt.Sprintf("%v", v)
	}

	floatVal, _ := strconv.ParseFloat(valStr, 64)
	if math.IsNaN(floatVal) || math.IsInf(floatVal, 0) {
		floatVal = 0.0
	}
	return floatVal
}

func (a *Dashboard) queryPrometheusMultiValues(query string, labelKey string) map[string]float64 {
	resMap := make(map[string]float64)
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		return resMap
	}
	apiURL := promAddr + "/api/v1/query?query=" + url.QueryEscape(query)

	body, err := a.queryPrometheus(apiURL, 3*time.Second)
	if err != nil {
		return resMap
	}

	var promResp prometheusQueryResponse
	if err := json.Unmarshal(body, &promResp); err != nil || promResp.Status != "success" {
		return resMap
	}

	for _, result := range promResp.Data.Result {
		key, ok := result.Metric[labelKey]
		if !ok || len(result.Value) < 2 {
			continue
		}
		var valStr string
		switch v := result.Value[1].(type) {
		case string:
			valStr = v
		case float64:
			resMap[key] = v
			continue
		default:
			valStr = fmt.Sprintf("%v", v)
		}
		floatVal, _ := strconv.ParseFloat(valStr, 64)
		if math.IsNaN(floatVal) || math.IsInf(floatVal, 0) {
			floatVal = 0.0
		}
		resMap[key] = floatVal
	}

	return resMap
}

func (a *Dashboard) getQPSFromPrometheus() float64 {
	return a.queryPrometheusSingleValue(fmt.Sprintf(`sum(rate(%s[5m]))`, mRequestTotal))
}

type TrendsSeries struct {
	Label   string  `json:"label"`
	Success []int64 `json:"success"`
	Failure []int64 `json:"failure"`
	Total   []int64 `json:"total"`
}

type TrendsResponse struct {
	Times  []string       `json:"times"`
	Series []TrendsSeries `json:"series"`
}

type trendRangeConfig struct {
	numPoints    int
	stepSeconds  int64
	redisMinutes int
	end          time.Time
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query bucketed gateway traffic success/failure trends
// @Param group_by query string false "Group by: model, provider, tenant, endpoint (default: global)"
// @Param time_range query string false "Time range: 1h, 6h, 24h, 7d, today (default: 1h)"
// @Success 200 {object} util.ResponseResult{data=TrendsResponse}
// @Router /api/v1/dashboard/trends [get]
func (a *Dashboard) getTrends(ctx context.Context, groupBy, timeRange string) (*TrendsResponse, error) {
	cacheKey := fmt.Sprintf("trends:%s:%s", groupBy, timeRange)
	if cached, ok := a.getCache(cacheKey); ok {
		return cached.(*TrendsResponse), nil
	}

	end := time.Now().Truncate(time.Minute)
	rangeConfig := resolveTrendRange(timeRange, end)
	end = rangeConfig.end
	start := end.Add(-time.Duration(rangeConfig.numPoints-1) * time.Duration(rangeConfig.stepSeconds) * time.Second)
	promRange := fmt.Sprintf("%ds", rangeConfig.stepSeconds)

	// 初始化响应
	var res TrendsResponse
	res.Times = make([]string, rangeConfig.numPoints)
	for i := 0; i < rangeConfig.numPoints; i++ {
		res.Times[i] = start.Add(time.Duration(i) * time.Duration(rangeConfig.stepSeconds) * time.Second).Format("15:04")
		if timeRange == "7d" {
			res.Times[i] = start.Add(time.Duration(i) * time.Duration(rangeConfig.stepSeconds) * time.Second).Format("01/02 15:04")
		}
	}

	// 尝试从 Prometheus 获取数据
	if a.isPrometheusAvailable() {
		var successQuery, errorQuery string

		switch groupBy {
		case "model":
			successQuery = fmt.Sprintf(`sum by (model) (increase(%s{status="success"}[%s]))`, mRequestTotal, promRange)
			errorQuery = fmt.Sprintf(`sum by (model) (increase(%s{status="error"}[%s]))`, mRequestTotal, promRange)
		case "provider":
			successQuery = fmt.Sprintf(`sum by (provider) (increase(%s{status="success"}[%s]))`, mRequestTotal, promRange)
			errorQuery = fmt.Sprintf(`sum by (provider) (increase(%s{status="error"}[%s]))`, mRequestTotal, promRange)
		case "tenant":
			successQuery = fmt.Sprintf(`sum by (tenant) (increase(%s{status="success"}[%s]))`, mRequestTotal, promRange)
			errorQuery = fmt.Sprintf(`sum by (tenant) (increase(%s{status="error"}[%s]))`, mRequestTotal, promRange)
		case "endpoint":
			successQuery = fmt.Sprintf(`sum by (endpoint) (increase(%s{status="success"}[%s]))`, mRequestTotal, promRange)
			errorQuery = fmt.Sprintf(`sum by (endpoint) (increase(%s{status="error"}[%s]))`, mRequestTotal, promRange)
		default: // global
			successQuery = fmt.Sprintf(`sum(increase(%s{status="success"}[%s]))`, mRequestTotal, promRange)
			errorQuery = fmt.Sprintf(`sum(increase(%s{status="error"}[%s]))`, mRequestTotal, promRange)
		}

		if groupBy == "" {
			// 全局汇总
			successData, err1 := a.queryPrometheusRange(successQuery, start.Unix(), end.Unix(), rangeConfig.stepSeconds)
			errorData, err2 := a.queryPrometheusRange(errorQuery, start.Unix(), end.Unix(), rangeConfig.stepSeconds)

			if err1 == nil && err2 == nil {
				series := TrendsSeries{
					Label:   "global",
					Success: make([]int64, rangeConfig.numPoints),
					Failure: make([]int64, rangeConfig.numPoints),
					Total:   make([]int64, rangeConfig.numPoints),
				}

				// 使用时间戳对齐填充
				alignPrometheusData(series.Success, successData, start.Unix(), rangeConfig.stepSeconds)
				alignPrometheusData(series.Failure, errorData, start.Unix(), rangeConfig.stepSeconds)

				for i := 0; i < rangeConfig.numPoints; i++ {
					series.Total[i] = series.Success[i] + series.Failure[i]
				}

				res.Series = []TrendsSeries{series}
				a.setCache(cacheKey, &res, 5*time.Second)
				return &res, nil
			}
		} else {
			// 按标签分组
			labelKey := groupBy
			successMap, err1 := a.queryPrometheusRangeMulti(successQuery, start.Unix(), end.Unix(), rangeConfig.stepSeconds, labelKey)
			errorMap, err2 := a.queryPrometheusRangeMulti(errorQuery, start.Unix(), end.Unix(), rangeConfig.stepSeconds, labelKey)

			if err1 == nil && err2 == nil {
				// 合并所有标签
				allLabels := make(map[string]bool)
				for label := range successMap {
					allLabels[label] = true
				}
				for label := range errorMap {
					allLabels[label] = true
				}

				res.Series = make([]TrendsSeries, 0, len(allLabels))
				for label := range allLabels {
					series := TrendsSeries{
						Label:   label,
						Success: make([]int64, rangeConfig.numPoints),
						Failure: make([]int64, rangeConfig.numPoints),
						Total:   make([]int64, rangeConfig.numPoints),
					}

					if successValues, ok := successMap[label]; ok {
						alignPrometheusData(series.Success, successValues, start.Unix(), rangeConfig.stepSeconds)
					}

					if errorValues, ok := errorMap[label]; ok {
						alignPrometheusData(series.Failure, errorValues, start.Unix(), rangeConfig.stepSeconds)
					}

					for i := 0; i < rangeConfig.numPoints; i++ {
						series.Total[i] = series.Success[i] + series.Failure[i]
					}

					res.Series = append(res.Series, series)
				}

				a.setCache(cacheKey, &res, 5*time.Second)
				return &res, nil
			}
		}
	}

	// Redis 或 内存 降级路径：只返回全局汇总
	if rangeConfig.redisMinutes > 0 {
		var vals []interface{}
		var err error
		if a.RedisClient != nil {
			minute := end.Unix() / 60
			keys := make([]string, rangeConfig.redisMinutes*2)
			for i := 0; i < rangeConfig.redisMinutes; i++ {
				ts := minute - int64(rangeConfig.redisMinutes-1-i)
				keys[i*2] = fmt.Sprintf("aigw:status:global:%d:s", ts)
				keys[i*2+1] = fmt.Sprintf("aigw:status:global:%d:f", ts)
			}
			vals, err = a.RedisClient.MGet(ctx, keys...).Result()
		} else {
			minute := end.Unix() / 60
			vals = make([]interface{}, rangeConfig.redisMinutes*2)
			for i := 0; i < rangeConfig.redisMinutes; i++ {
				ts := minute - int64(rangeConfig.redisMinutes-1-i)
				succ, fail := metrics.GlobalStore.GetGlobalStatus(ts)
				if succ > 0 {
					vals[i*2] = strconv.FormatInt(succ, 10)
				}
				if fail > 0 {
					vals[i*2+1] = strconv.FormatInt(fail, 10)
				}
			}
		}

		if err == nil && len(vals) == rangeConfig.redisMinutes*2 {
			series := TrendsSeries{
				Label:   "global",
				Success: make([]int64, rangeConfig.numPoints),
				Failure: make([]int64, rangeConfig.numPoints),
				Total:   make([]int64, rangeConfig.numPoints),
			}

			aggregateRedisTrendValues(&series, vals, end.Unix()/60, start.Unix()/60, rangeConfig.stepSeconds/60)

			res.Series = []TrendsSeries{series}
		} else {
			res.Series = []TrendsSeries{
				{
					Label:   "global",
					Success: make([]int64, rangeConfig.numPoints),
					Failure: make([]int64, rangeConfig.numPoints),
					Total:   make([]int64, rangeConfig.numPoints),
				},
			}
		}
	} else {
		res.Series = []TrendsSeries{
			{
				Label:   "global",
				Success: make([]int64, rangeConfig.numPoints),
				Failure: make([]int64, rangeConfig.numPoints),
				Total:   make([]int64, rangeConfig.numPoints),
			},
		}
	}

	a.setCache(cacheKey, &res, 5*time.Second)
	return &res, nil
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query bucketed gateway traffic success/failure trends
// @Param group_by query string false "Group by: model, provider, tenant, endpoint (default: global)"
// @Param time_range query string false "Time range: 1h, 6h, 24h, 7d, today (default: 1h)"
// @Success 200 {object} util.ResponseResult{data=TrendsResponse}
// @Router /api/v1/dashboard/trends [get]
func (a *Dashboard) QueryTrends(c *gin.Context) {
	groupBy := c.Query("group_by")
	timeRange := c.DefaultQuery("time_range", "1h")
	res, err := a.getTrends(c.Request.Context(), groupBy, timeRange)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, res)
}

func resolveTrendRange(timeRange string, now time.Time) trendRangeConfig {
	switch timeRange {
	case "6h":
		return trendRangeConfig{numPoints: 72, stepSeconds: 5 * 60, end: now}
	case "24h":
		return trendRangeConfig{numPoints: 96, stepSeconds: 15 * 60, end: now}
	case "7d":
		return trendRangeConfig{numPoints: 84, stepSeconds: 2 * 60 * 60, end: now}
	case "today":
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		minutes := int(now.Sub(midnight).Minutes())
		if minutes < 1 {
			return trendRangeConfig{numPoints: 1, stepSeconds: 60, redisMinutes: 1, end: now}
		}

		stepMinutes := 1
		for _, candidate := range []int{1, 5, 10, 15} {
			stepMinutes = candidate
			if (minutes+candidate-1)/candidate <= 120 {
				break
			}
		}

		numPoints := minutes / stepMinutes
		if numPoints < 1 {
			numPoints = 1
		}
		return trendRangeConfig{
			numPoints:    numPoints,
			stepSeconds:  int64(stepMinutes * 60),
			redisMinutes: min(minutes, 120),
			end:          midnight.Add(time.Duration(numPoints*stepMinutes) * time.Minute),
		}
	default:
		return trendRangeConfig{numPoints: 60, stepSeconds: 60, redisMinutes: 60, end: now}
	}
}

func aggregateRedisTrendValues(series *TrendsSeries, values []interface{}, currentMinute, startMinute, stepMinutes int64) {
	numMinutes := len(values) / 2
	for i := 0; i < numMinutes; i++ {
		var succ, fail int64
		if values[i*2] != nil {
			succ, _ = strconv.ParseInt(fmt.Sprint(values[i*2]), 10, 64)
		}
		if values[i*2+1] != nil {
			fail, _ = strconv.ParseInt(fmt.Sprint(values[i*2+1]), 10, 64)
		}

		valueMinute := currentMinute - int64(numMinutes-1-i)
		diff := valueMinute - startMinute
		idx := int64(0)
		if diff > 0 {
			idx = (diff + stepMinutes - 1) / stepMinutes
		}
		if idx >= 0 && idx < int64(len(series.Total)) {
			series.Success[idx] += succ
			series.Failure[idx] += fail
			series.Total[idx] += succ + fail
		}
	}
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary One-click synchronization of database configuration to Redis
// @Success 200 {object} util.ResponseResult
// @Router /api/v1/dashboard/sync-redis [post]
func (a *Dashboard) SyncRedis(c *gin.Context) {
	if !util.FromIsRootUser(c.Request.Context()) {
		util.ResError(c, fmt.Errorf("only admin is allowed to perform sync operation"))
		return
	}

	if a.RedisSync == nil {
		util.ResError(c, fmt.Errorf("redis sync service is not initialized"))
		return
	}

	err := a.RedisSync.SyncAllToRedis(c.Request.Context())
	if err != nil {
		util.ResError(c, err)
		return
	}

	util.ResSuccess(c, nil)
}

type ModelRankingItem struct {
	ModelID      string  `json:"model_id"`
	ModelCode    string  `json:"model_code"`
	ModelName    string  `json:"model_name"`
	RequestCount int64   `json:"request_count"`
	SuccessCount int64   `json:"success_count"`
	FailCount    int64   `json:"fail_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	AvgTTFTMs    float64 `json:"avg_ttft_ms"`
	P50TTFTMs    float64 `json:"p50_ttft_ms"`
	P95TTFTMs    float64 `json:"p95_ttft_ms"`
	P99TTFTMs    float64 `json:"p99_ttft_ms"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query model usage ranking with detailed metrics
// @Param sort_by query string false "Sort by: request_count, avg_latency, avg_ttft, tokens, cost, success_rate (default: request_count)"
// @Param limit query int false "Limit results (default: 10)"
// @Success 200 {object} util.ResponseResult{data=[]ModelRankingItem}
// @Router /api/v1/dashboard/model-ranking [get]
func (a *Dashboard) getModelRanking(ctx context.Context, sortBy, timeRange string, limit int) ([]ModelRankingItem, error) {
	cacheKey := fmt.Sprintf("ranking:%s:%s:%d", sortBy, timeRange, limit)
	if cached, ok := a.getCache(cacheKey); ok {
		return cached.([]ModelRankingItem), nil
	}

	// 根据 time_range 计算 PromQL 范围和 Redis 窗口
	promRange, redisMinutes := resolveTimeRange(timeRange)

	// 1. 从数据库查询所有启用的模型
	var models []rschema.Model
	if err := a.DB.WithContext(ctx).Where("enabled = ?", 1).Find(&models).Error; err != nil {
		return nil, err
	}

	if len(models) == 0 {
		a.setCache(cacheKey, []ModelRankingItem{}, 5*time.Second)
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
	if a.isPrometheusAvailable() {
		// 查询请求计数
		successQuery := fmt.Sprintf(`sum by (model) (increase(%s{status="success"}[%s]))`, mRequestTotal, promRange)
		errorQuery := fmt.Sprintf(`sum by (model) (increase(%s{status="error"}[%s]))`, mRequestTotal, promRange)

		successMap := a.queryPrometheusMultiValues(successQuery, "model")
		errorMap := a.queryPrometheusMultiValues(errorQuery, "model")

		// 查询延迟指标
		avgLatencyMap := a.queryPrometheusMultiValues(fmt.Sprintf(`sum by (model) (rate(%s[%s])) / sum by (model) (rate(%s[%s]))`, mRequestDurationSum, promRange, mRequestDurationCount, promRange), "model")
		p50LatencyMap := a.queryPrometheusMultiValues(fmt.Sprintf(`histogram_quantile(0.50, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, promRange), "model")
		p95LatencyMap := a.queryPrometheusMultiValues(fmt.Sprintf(`histogram_quantile(0.95, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, promRange), "model")
		p99LatencyMap := a.queryPrometheusMultiValues(fmt.Sprintf(`histogram_quantile(0.99, sum by (model, le) (rate(%s[%s])))`, mRequestDurationBucket, promRange), "model")

		// 查询 TTFT 指标
		avgTTFTMap := a.queryPrometheusMultiValues(fmt.Sprintf(`sum by (model) (rate(%s[%s])) / sum by (model) (rate(%s[%s]))`, mTtftSum, promRange, mTtftCount, promRange), "model")
		p50TTFTMap := a.queryPrometheusMultiValues(fmt.Sprintf(`histogram_quantile(0.50, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, promRange), "model")
		p95TTFTMap := a.queryPrometheusMultiValues(fmt.Sprintf(`histogram_quantile(0.95, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, promRange), "model")
		p99TTFTMap := a.queryPrometheusMultiValues(fmt.Sprintf(`histogram_quantile(0.99, sum by (model, le) (rate(%s[%s])))`, mTtftBucket, promRange), "model")

		// 查询 Token 和费用
		tokensMap := a.queryPrometheusMultiValues(fmt.Sprintf(`sum by (model) (increase(%s[%s]))`, mTokensTotal, promRange), "model")
		costMap := a.queryPrometheusMultiValues(fmt.Sprintf(`sum by (model) (increase(%s[%s]))`, mCostTotal, promRange), "model")

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
			items[i].TotalCost = costMap[code]
		}
	} else if redisMinutes > 0 {
		var values []interface{}
		var err error
		currentMin := time.Now().Unix() / 60
		numMinutes := redisMinutes

		if a.RedisClient != nil {
			keys := make([]string, 0, len(models)*numMinutes*2)
			for _, m := range models {
				for i := 0; i < numMinutes; i++ {
					minute := currentMin - int64(numMinutes-1-i)
					keys = append(keys,
						fmt.Sprintf("aigw:status:model:%s:%d:s", m.ModelCode, minute),
						fmt.Sprintf("aigw:status:model:%s:%d:f", m.ModelCode, minute),
					)
				}
			}

			values, err = a.RedisClient.MGet(ctx, keys...).Result()
		} else {
			values = make([]interface{}, len(models)*numMinutes*2)
			idx := 0
			for _, m := range models {
				for i := 0; i < numMinutes; i++ {
					minute := currentMin - int64(numMinutes-1-i)
					succ, fail := metrics.GlobalStore.GetModelStatus(m.ModelCode, minute)
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
	switch sortBy {
	case "avg_latency":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].AvgLatencyMs < filtered[j].AvgLatencyMs
		})
	case "avg_ttft":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].AvgTTFTMs < filtered[j].AvgTTFTMs
		})
	case "tokens":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].TotalTokens > filtered[j].TotalTokens
		})
	case "cost":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].TotalCost > filtered[j].TotalCost
		})
	case "success_rate":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].SuccessRate > filtered[j].SuccessRate
		})
	default: // request_count
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].RequestCount > filtered[j].RequestCount
		})
	}

	// 6. 限制结果数量
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	a.setCache(cacheKey, filtered, 5*time.Second)
	return filtered, nil
}

// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query model usage ranking with detailed metrics
// @Param sort_by query string false "Sort by: request_count, avg_latency, avg_ttft, tokens, cost, success_rate (default: request_count)"
// @Param limit query int false "Limit results (default: 10)"
// @Success 200 {object} util.ResponseResult{data=[]ModelRankingItem}
// @Router /api/v1/dashboard/model-ranking [get]
func (a *Dashboard) QueryModelRanking(c *gin.Context) {
	sortBy := c.DefaultQuery("sort_by", "request_count")
	timeRange := c.DefaultQuery("time_range", "1h")
	limit := 10
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil && l > 0 {
		limit = l
	}
	res, err := a.getModelRanking(c.Request.Context(), sortBy, timeRange, limit)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, res)
}

// resolveTimeRange 将前端传入的 time_range 参数转换为 PromQL 范围字符串和 Redis 窗口分钟数
// promRange: 用于 PromQL 查询的范围（如 "60m", "6h", "24h", "7d"）
// redisMinutes: Redis 降级时查询的分钟数（0 表示 Redis 无法覆盖该范围）
func resolveTimeRange(timeRange string) (string, int) {
	switch timeRange {
	case "6h":
		return "6h", 0 // Redis TTL 只有 2h，无法覆盖
	case "24h":
		return "24h", 0
	case "7d":
		return "7d", 0
	case "today":
		// 计算从今天零点到现在的分钟数
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		minutes := int(now.Sub(midnight).Minutes())
		if minutes < 1 {
			minutes = 1
		}
		// PromQL 使用 "today" 时，用精确的分钟数作为范围
		return fmt.Sprintf("%dm", minutes), minutes // Redis 也可以覆盖今日（只要不超过 2h TTL）
	default: // "1h"
		return "60m", 60
	}
}

func mustParseInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// alignPrometheusData 将 Prometheus Range 查询返回的 values 按时间戳对齐填充到 dst 数组中
func alignPrometheusData(dst []int64, values [][]interface{}, startUnix int64, step int64) {
	numPoints := int64(len(dst))
	for _, v := range values {
		if len(v) < 2 {
			continue
		}

		// 解析时间戳
		var ts float64
		switch tVal := v[0].(type) {
		case float64:
			ts = tVal
		case string:
			ts, _ = strconv.ParseFloat(tVal, 64)
		}
		timestamp := int64(ts)

		// 解析数值
		var val int64
		if valStr, ok := v[1].(string); ok {
			if fVal, err := strconv.ParseFloat(valStr, 64); err == nil {
				val = int64(math.Round(fVal))
			}
		}

		// 计算该时间戳对应的目标数组索引 (四舍五入对齐)
		diff := timestamp - startUnix
		idx := (diff + step/2) / step

		// 边界处理
		if idx >= 0 && idx < numPoints {
			dst[idx] = val
		}
	}
}

func (a *Dashboard) queryPrometheus(apiURL string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	username := config.C.Util.PrometheusServer.Username
	password := config.C.Util.PrometheusServer.Password
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// queryPrometheusRange 执行 Prometheus range 查询，返回 [timestamp, value] 数组
func (a *Dashboard) queryPrometheusRange(query string, start, end, step int64) ([][]interface{}, error) {
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		return nil, fmt.Errorf("prometheus address not configured")
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start, 10))
	params.Set("end", strconv.FormatInt(end, 10))
	params.Set("step", strconv.FormatInt(step, 10))

	apiURL := promAddr + "/api/v1/query_range?" + params.Encode()

	body, err := a.queryPrometheus(apiURL, 5*time.Second)
	if err != nil {
		return nil, err
	}

	var promResp prometheusQueryRangeResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, err
	}

	if promResp.Status != "success" || len(promResp.Data.Result) == 0 {
		return nil, fmt.Errorf("no data returned")
	}

	return promResp.Data.Result[0].Values, nil
}

// queryPrometheusRangeMulti 执行 Prometheus range 查询，返回按标签分组的数据
func (a *Dashboard) queryPrometheusRangeMulti(query string, start, end, step int64, labelKey string) (map[string][][]interface{}, error) {
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		return nil, fmt.Errorf("prometheus address not configured")
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start, 10))
	params.Set("end", strconv.FormatInt(end, 10))
	params.Set("step", strconv.FormatInt(step, 10))

	apiURL := promAddr + "/api/v1/query_range?" + params.Encode()

	body, err := a.queryPrometheus(apiURL, 5*time.Second)
	if err != nil {
		return nil, err
	}

	var promResp prometheusQueryRangeResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, err
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("query failed")
	}

	result := make(map[string][][]interface{})
	for _, r := range promResp.Data.Result {
		label, ok := r.Metric[labelKey]
		if !ok {
			continue
		}
		result[label] = r.Values
	}

	return result, nil
}

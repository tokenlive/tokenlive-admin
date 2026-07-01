package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelCatalog management
type ModelCatalog struct {
	ModelCatalogBIZ   *biz.ModelCatalog
	prometheusAvail   bool
	prometheusMu      sync.RWMutex
	prometheusLastChk time.Time
}

// metricName 根据配置的 MetricPrefix 拼接完整的 Prometheus 指标名
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
)

// prometheusQueryResponse Prometheus 查询响应结构
type prometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Query model catalog list
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param slug query string false "Slug (like)"
// @Param status query string false "Status (available/paused)"
// @Param visibility query string false "Visibility (public/private)"
// @Param featured query bool false "Featured"
// @Param model_code query string false "Model code"
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs [get]
func (m *ModelCatalog) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.ModelCatalogQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelCatalogBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Get model catalog by ID
// @Param id path string true "Model ID"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id} [get]
func (m *ModelCatalog) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelCatalogBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Get model catalog by slug
// @Param slug path string true "Slug"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/slug/{slug} [get]
func (m *ModelCatalog) GetBySlug(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelCatalogBIZ.GetBySlug(ctx, c.Param("slug"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Create model catalog
// @Param body body schema.ModelCatalogForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalog}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs [post]
func (m *ModelCatalog) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelCatalogBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Update model catalog
// @Param id path string true "Model ID"
// @Param body body schema.ModelCatalogForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id} [put]
func (m *ModelCatalog) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelCatalogBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Publish model catalog
// @Param id path string true "Model ID"
// @Param body body schema.ModelCatalogPublishForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id}/publish [put]
func (m *ModelCatalog) Publish(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogPublishForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelCatalogBIZ.Publish(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Delete model catalog
// @Param id path string true "Model ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id} [delete]
func (m *ModelCatalog) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelCatalogBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Query public model catalogs (for portal)
// @Param limit query int false "Limit" default(50)
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/public [get]
func (m *ModelCatalog) QueryPublic(c *gin.Context) {
	ctx := c.Request.Context()
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	result, err := m.ModelCatalogBIZ.QueryPublic(ctx, limit)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// isPrometheusAvailable 检查 Prometheus 是否可用，结果缓存 30 秒
func (m *ModelCatalog) isPrometheusAvailable() bool {
	m.prometheusMu.RLock()
	if time.Since(m.prometheusLastChk) < 30*time.Second {
		avail := m.prometheusAvail
		m.prometheusMu.RUnlock()
		return avail
	}
	m.prometheusMu.RUnlock()

	m.prometheusMu.Lock()
	defer m.prometheusMu.Unlock()

	// 再次检查，避免重复请求
	if time.Since(m.prometheusLastChk) < 30*time.Second {
		return m.prometheusAvail
	}

	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		m.prometheusAvail = false
		m.prometheusLastChk = time.Now()
		return false
	}

	// 尝试访问健康检查端点
	healthURL := promAddr + "/-/healthy"
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		m.prometheusAvail = false
		m.prometheusLastChk = time.Now()
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
		m.prometheusAvail = false
		m.prometheusLastChk = time.Now()
		return false
	}
	defer resp.Body.Close()

	m.prometheusAvail = resp.StatusCode == http.StatusOK
	m.prometheusLastChk = time.Now()
	return m.prometheusAvail
}

// queryPrometheus executes a Prometheus query
func (m *ModelCatalog) queryPrometheus(apiURL string, timeout time.Duration) ([]byte, error) {
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

// queryPrometheusSingleValue executes a Prometheus query and returns a single value
func (m *ModelCatalog) queryPrometheusSingleValue(query string) float64 {
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		return 0
	}

	apiURL := promAddr + "/api/v1/query?query=" + url.QueryEscape(query)
	body, err := m.queryPrometheus(apiURL, 2*time.Second)
	if err != nil {
		return 0
	}

	var promResp prometheusQueryResponse
	if err := json.Unmarshal(body, &promResp); err != nil || promResp.Status != "success" || len(promResp.Data.Result) == 0 {
		return 0
	}

	valArr := promResp.Data.Result[0].Value
	if len(valArr) < 2 {
		return 0
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

// queryPrometheusModelValue executes a Prometheus query filtered by model and returns a single value
func (m *ModelCatalog) queryPrometheusModelValue(query string, modelCode string) float64 {
	fullQuery := fmt.Sprintf(query, modelCode)
	return m.queryPrometheusSingleValue(fullQuery)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Get model metrics from Prometheus
// @Description 获取模型的服务质量指标（可用性、成功率、TTFT、响应速度等）
// @Param id path string true "Model ID or Slug"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalogMetricResponse}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id}/metrics [get]
func (m *ModelCatalog) GetMetrics(c *gin.Context) {
	ctx := c.Request.Context()
	modelID := c.Param("id")

	// 先尝试获取 model_catalog 信息，用于获取 model_code
	catalog, err := m.ModelCatalogBIZ.Get(ctx, modelID)
	if err != nil {
		// 如果 ID 找不到，尝试用 slug 查找
		catalog, err = m.ModelCatalogBIZ.GetBySlug(ctx, modelID)
		if err != nil {
			util.ResError(c, err)
			return
		}
	}

	// 使用 model_code 作为 Prometheus 查询的 model 标签
	// 如果 model_code 为空，使用 model_id
	modelCode := catalog.ModelCode
	if modelCode == "" {
		modelCode = catalog.ModelID
	}

	// 检查 Prometheus 是否可用
	if !m.isPrometheusAvailable() {
		// Prometheus 不可用时返回空数组
		util.ResSuccess(c, schema.ModelCatalogMetricResponse{})
		return
	}

	// 查询多个时间窗口的指标
	windows := []struct {
		label     string
		promRange string
	}{
		{"1h", "1h"},
		{"24h", "24h"},
		{"7d", "7d"},
	}

	metrics := make(schema.ModelCatalogMetricResponse, 0, len(windows))
	now := time.Now()

	for _, w := range windows {
		metric := schema.ModelCatalogMetric{
			Window:    w.label,
			UpdatedAt: now.Format("2006-01-02 15:04:05"),
		}

		// 1. 查询成功/失败请求数，计算成功率
		successQuery := fmt.Sprintf(`sum(increase(%s{model="%s",status="success"}[%s]))`, mRequestTotal, modelCode, w.promRange)
		errorQuery := fmt.Sprintf(`sum(increase(%s{model="%s",status="error"}[%s]))`, mRequestTotal, modelCode, w.promRange)

		successCount := m.queryPrometheusSingleValue(successQuery)
		errorCount := m.queryPrometheusSingleValue(errorQuery)
		totalCount := successCount + errorCount

		metric.SampleCount = int64(totalCount)

		if totalCount > 0 {
			metric.SuccessRate = successCount / totalCount
			metric.Availability = successCount / totalCount // 使用成功率作为可用性
		}

		// 2. 查询 TTFT p50 和 p95
		ttftP50Query := fmt.Sprintf(`histogram_quantile(0.50, sum by (le) (rate(%s{model="%s"}[%s])))`, mTtftBucket, modelCode, w.promRange)
		ttftP95Query := fmt.Sprintf(`histogram_quantile(0.95, sum by (le) (rate(%s{model="%s"}[%s])))`, mTtftBucket, modelCode, w.promRange)

		ttftP50 := m.queryPrometheusSingleValue(ttftP50Query)
		ttftP95 := m.queryPrometheusSingleValue(ttftP95Query)

		// 转换为毫秒
		metric.TtftP50Ms = ttftP50 * 1000
		metric.TtftP95Ms = ttftP95 * 1000

		// 3. 查询响应速度（tokens per second）
		// 使用 tokens_total 的增长速率
		tokensQuery := fmt.Sprintf(`sum(rate(%s{model="%s"}[%s]))`, mTokensTotal, modelCode, w.promRange)
		metric.ResponseSpeed = m.queryPrometheusSingleValue(tokensQuery)

		metrics = append(metrics, metric)
	}

	util.ResSuccess(c, metrics)
}

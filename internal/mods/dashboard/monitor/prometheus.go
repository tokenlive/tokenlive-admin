package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

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
func (a *Reader) isPrometheusAvailable(ctx context.Context) bool {
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
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
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

func (a *Reader) prometheusScalar(ctx context.Context, query string) *float64 {
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		return nil
	}
	apiURL := promAddr + "/api/v1/query?query=" + url.QueryEscape(query)
	if s := scope(ctx); s != nil {
		apiURL += "&time=" + strconv.FormatInt(s.plan.Cutoff.Unix(), 10)
	}
	body, err := a.queryPrometheus(ctx, apiURL, 2*time.Second)
	if err != nil {
		return nil
	}
	var response prometheusQueryResponse
	if err = json.Unmarshal(body, &response); err != nil || response.Status != "success" {
		sourceError(ctx, "prometheus", fmt.Errorf("invalid prometheus response"))
		return nil
	}
	if len(response.Data.Result) == 0 || len(response.Data.Result[0].Value) < 2 {
		return nil
	}
	value, err := strconv.ParseFloat(fmt.Sprint(response.Data.Result[0].Value[1]), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	seen(ctx, 0)
	return &value
}

func (a *Reader) queryPrometheusMultiValuesAt(ctx context.Context, query string, labelKey string, end time.Time) map[string]float64 {
	resMap := make(map[string]float64)
	promAddr := config.C.Util.PrometheusServer.Address
	if promAddr == "" {
		return resMap
	}
	apiURL := promAddr + "/api/v1/query?query=" + url.QueryEscape(query)
	if !end.IsZero() {
		apiURL += "&time=" + strconv.FormatInt(end.Unix(), 10)
	}

	body, err := a.queryPrometheus(ctx, apiURL, 3*time.Second)
	if err != nil {
		return resMap
	}

	var promResp prometheusQueryResponse
	if err := json.Unmarshal(body, &promResp); err != nil || promResp.Status != "success" {
		sourceError(ctx, "prometheus", fmt.Errorf("invalid prometheus response"))
		return resMap
	}

	if len(promResp.Data.Result) > 0 {
		seen(ctx, 0)
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

func (a *Reader) queryPrometheus(ctx context.Context, apiURL string, timeout time.Duration) ([]byte, error) {
	source(ctx, "prometheus")
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
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
		sourceError(ctx, "prometheus", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("status code %d", resp.StatusCode)
		sourceError(ctx, "prometheus", err)
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

// queryPrometheusRange 执行 Prometheus range 查询，返回 [timestamp, value] 数组
func (a *Reader) queryPrometheusRangeRaw(ctx context.Context, query string, start, end, step int64) ([][]interface{}, error) {
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

	body, err := a.queryPrometheus(ctx, apiURL, 5*time.Second)
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

	if len(promResp.Data.Result) == 0 {
		return nil, nil
	}

	for _, v := range promResp.Data.Result[0].Values {
		if len(v) > 0 {
			if at, ok := v[0].(float64); ok {
				seen(ctx, int64(at))
			}
		}
	}
	return promResp.Data.Result[0].Values, nil
}

// queryPrometheusRangeMulti 执行 Prometheus range 查询，返回按标签分组的数据
func (a *Reader) queryPrometheusRangeMultiRaw(ctx context.Context, query string, start, end, step int64, labelKey string) (map[string][][]interface{}, error) {
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

	body, err := a.queryPrometheus(ctx, apiURL, 5*time.Second)
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
		for _, v := range r.Values {
			if len(v) > 0 {
				if at, ok := v[0].(float64); ok {
					seen(ctx, int64(at))
				}
			}
		}
		result[label] = r.Values
	}

	return result, nil
}

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDashboard(t *testing.T) (*api.Dashboard, *gin.Engine) {
	// 使用 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&rschema.Model{}, &rschema.Endpoint{}, &rschema.Provider{})
	assert.NoError(t, err)

	// 创建 Redis 客户端（使用测试 Redis 或 mock）
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	dashboard := &api.Dashboard{
		DB:          db,
		RedisClient: redisClient,
		RedisSync:   nil,
	}

	// 创建 Gin 引擎
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	return dashboard, engine
}

func TestQueryOverview(t *testing.T) {
	dashboard, engine := setupTestDashboard(t)

	// 注册路由
	engine.GET("/api/v1/dashboard/overview", dashboard.QueryOverview)

	// 创建请求
	req, _ := http.NewRequest("GET", "/api/v1/dashboard/overview", nil)
	w := httptest.NewRecorder()

	// 执行请求
	engine.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var response util.ResponseResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	// 验证数据结构
	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)

	// 检查必需的字段
	_, hasQPS := data["qps"]
	_, hasDailyRequests := data["daily_requests"]
	_, hasAvgLatency := data["avg_latency_ms"]
	_, hasAvgTTFT := data["avg_ttft_ms"]
	_, hasCircuitBreakers := data["active_circuit_breakers"]

	assert.True(t, hasQPS)
	assert.True(t, hasDailyRequests)
	assert.True(t, hasAvgLatency)
	assert.True(t, hasAvgTTFT)
	assert.True(t, hasCircuitBreakers)
}

func TestQueryTrends(t *testing.T) {
	dashboard, engine := setupTestDashboard(t)

	// 注册路由
	engine.GET("/api/v1/dashboard/trends", dashboard.QueryTrends)

	// 测试全局汇总
	t.Run("Global", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/dashboard/trends", nil)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response util.ResponseResult
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)

		data, ok := response.Data.(map[string]interface{})
		assert.True(t, ok)

		// 检查 Times 和 Series
		_, hasTimes := data["times"]
		_, hasSeries := data["series"]
		assert.True(t, hasTimes)
		assert.True(t, hasSeries)
	})

	// 测试按模型分组
	t.Run("GroupByModel", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/dashboard/trends?group_by=model", nil)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response util.ResponseResult
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})
}

func TestQueryTrendsIncludesLatestBucket(t *testing.T) {
	type queryInfo struct {
		span  int64
		step  int64
		query string
	}
	var queries []queryInfo
	var mu sync.Mutex
	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/-/healthy" {
			w.WriteHeader(http.StatusOK)
			return
		}

		start, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
		assert.NoError(t, err)
		end, err := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
		assert.NoError(t, err)

		mu.Lock()
		queries = append(queries, queryInfo{
			span:  end - start,
			step:  mustParseTestInt(t, r.URL.Query().Get("step")),
			query: r.URL.Query().Get("query"),
		})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[%d,"1"],[%d,"9"]]}]}}`, start, end)
	}))
	defer prometheus.Close()

	oldAddress := config.C.Util.PrometheusServer.Address
	oldUsername := config.C.Util.PrometheusServer.Username
	oldPassword := config.C.Util.PrometheusServer.Password
	config.C.Util.PrometheusServer.Address = prometheus.URL
	config.C.Util.PrometheusServer.Username = ""
	config.C.Util.PrometheusServer.Password = ""
	defer func() {
		config.C.Util.PrometheusServer.Address = oldAddress
		config.C.Util.PrometheusServer.Username = oldUsername
		config.C.Util.PrometheusServer.Password = oldPassword
	}()

	dashboard, engine := setupTestDashboard(t)
	engine.GET("/api/v1/dashboard/trends", dashboard.QueryTrends)

	tests := []struct {
		timeRange string
		points    int
		step      int64
	}{
		{timeRange: "1h", points: 60, step: 60},
		{timeRange: "6h", points: 72, step: 300},
		{timeRange: "24h", points: 96, step: 900},
		{timeRange: "7d", points: 84, step: 7200},
	}
	for _, tt := range tests {
		t.Run(tt.timeRange, func(t *testing.T) {
			mu.Lock()
			queries = nil
			mu.Unlock()

			req, _ := http.NewRequest("GET", "/api/v1/dashboard/trends?time_range="+tt.timeRange, nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response util.ResponseResult
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			data := response.Data.(map[string]interface{})
			times := data["times"].([]interface{})
			series := data["series"].([]interface{})
			total := series[0].(map[string]interface{})["total"].([]interface{})

			assert.Len(t, times, tt.points)
			assert.LessOrEqual(t, len(times), 120)
			assert.Equal(t, float64(18), total[len(total)-1])

			mu.Lock()
			captured := append([]queryInfo(nil), queries...)
			mu.Unlock()
			assert.Len(t, captured, 2)
			for _, query := range captured {
				assert.Equal(t, int64(tt.points-1)*tt.step, query.span)
				assert.Equal(t, tt.step, query.step)
				assert.True(t, strings.Contains(query.query, fmt.Sprintf("[%ds]", tt.step)))
			}
		})
	}
}

func mustParseTestInt(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := strconv.ParseInt(value, 10, 64)
	assert.NoError(t, err)
	return parsed
}

func TestQueryModelRanking(t *testing.T) {
	dashboard, engine := setupTestDashboard(t)

	// 注册路由
	engine.GET("/api/v1/dashboard/model-ranking", dashboard.QueryModelRanking)

	// 创建测试数据
	ctx := context.Background()
	testModel := rschema.Model{
		ModelCode: "test-model",
		ModelName: "Test Model",
		Enabled:   1,
	}
	dashboard.DB.WithContext(ctx).Create(&testModel)

	// 测试默认排序
	t.Run("DefaultSort", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/dashboard/model-ranking", nil)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response util.ResponseResult
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)

		// 验证数据是数组
		_, ok := response.Data.([]interface{})
		assert.True(t, ok)
	})

	// 测试按延迟排序
	t.Run("SortByLatency", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/dashboard/model-ranking?sort_by=avg_latency&limit=5", nil)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response util.ResponseResult
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})
}

func TestIsPrometheusAvailable(t *testing.T) {
	dashboard := &api.Dashboard{}

	// 测试未配置 Prometheus
	t.Run("NotConfigured", func(t *testing.T) {
		oldAddr := config.C.Util.PrometheusServer.Address
		config.C.Util.PrometheusServer.Address = ""
		defer func() {
			config.C.Util.PrometheusServer.Address = oldAddr
		}()

		available := dashboard.IsPrometheusAvailable()
		assert.False(t, available)
	})
}

func TestQueryPrometheusRange(t *testing.T) {
	// 创建一个模拟的 Prometheus 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求参数
		query := r.URL.Query().Get("query")
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		step := r.URL.Query().Get("step")

		assert.NotEmpty(t, query)
		assert.NotEmpty(t, start)
		assert.NotEmpty(t, end)
		assert.NotEmpty(t, step)

		// 返回模拟数据
		response := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "matrix",
				"result": []map[string]interface{}{
					{
						"metric": map[string]string{},
						"values": [][]interface{}{
							{1234567890, "100"},
							{1234567950, "200"},
						},
					},
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	_ = &api.Dashboard{}

	// 注意：这个测试需要能够设置 PrometheusServer.Address
	// 由于配置是全局的，我们可能需要调整测试方法
	// 这里只是展示测试结构
}

func TestQueryCircuitBreakers(t *testing.T) {
	dashboard, engine := setupTestDashboard(t)

	// 注册路由
	engine.GET("/api/v1/dashboard/circuit-breakers", dashboard.QueryCircuitBreakers)

	req, _ := http.NewRequest("GET", "/api/v1/dashboard/circuit-breakers", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response util.ResponseResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	// 验证数据是数组
	_, ok := response.Data.([]interface{})
	assert.True(t, ok)
}

func TestQueryQPS(t *testing.T) {
	dashboard, engine := setupTestDashboard(t)

	// 注册路由
	engine.GET("/api/v1/dashboard/qps", dashboard.QueryQPS)

	req, _ := http.NewRequest("GET", "/api/v1/dashboard/qps", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response util.ResponseResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	// 验证数据是数字
	_, ok := response.Data.(float64)
	assert.True(t, ok)
}

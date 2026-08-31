package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

type overviewRedisHook struct{}

func (overviewRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (overviewRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		switch typedCmd := cmd.(type) {
		case *redis.SliceCmd:
			typedCmd.SetVal([]interface{}{"103", "1000", "2000", "300", "400", "11.027258"})
		case *redis.StringSliceCmd:
			typedCmd.SetVal([]string{})
		}
		return nil
	}
}

func (overviewRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(_ context.Context, _ []redis.Cmder) error {
		return nil
	}
}

type performanceRedisHook struct {
	values map[string]interface{}
}

func (performanceRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h performanceRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		sliceCmd, ok := cmd.(*redis.SliceCmd)
		if !ok {
			return nil
		}
		args := cmd.Args()
		values := make([]interface{}, 0, len(args)-1)
		for _, arg := range args[1:] {
			values = append(values, h.values[fmt.Sprint(arg)])
		}
		sliceCmd.SetVal(values)
		return nil
	}
}

func (performanceRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(_ context.Context, _ []redis.Cmder) error {
		return nil
	}
}

func TestQueryOverviewUsesRedisDailyCost(t *testing.T) {
	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1781457000,"3.25"]}]}}`))
	}))
	defer prometheus.Close()

	oldAddress := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = prometheus.URL
	defer func() {
		config.C.Util.PrometheusServer.Address = oldAddress
	}()

	redisClient := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	redisClient.AddHook(overviewRedisHook{})
	dashboard := &Dashboard{RedisClient: redisClient}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/overview", nil)

	dashboard.QueryOverview(ctx)

	var response util.ResponseResult
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 11.027258, data["daily_cost"])
}

func TestResolveTrendRangeToday(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 3, 0, 0, time.Local)

	config := resolveTrendRange("today", now)

	assert.Equal(t, 60, config.numPoints)
	assert.Equal(t, int64(10*60), config.stepSeconds)
	assert.Equal(t, time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local), config.end)
	assert.LessOrEqual(t, config.numPoints, 120)
}

func TestAggregateRedisTrendValues(t *testing.T) {
	series := TrendsSeries{
		Success: make([]int64, 3),
		Failure: make([]int64, 3),
		Total:   make([]int64, 3),
	}
	values := []interface{}{
		"1", "2",
		"3", "4",
		"5", "6",
		"7", "8",
		"9", "10",
	}

	aggregateRedisTrendValues(&series, values, 100, 96, 2)

	assert.Equal(t, []int64{1, 8, 16}, series.Success)
	assert.Equal(t, []int64{2, 10, 18}, series.Failure)
	assert.Equal(t, []int64{3, 18, 34}, series.Total)
}

func TestBuildModelTokensQueryCountsInputAndOutputOnly(t *testing.T) {
	query := buildModelTokensQuery("24h")

	expected := `sum by (model) (increase(` + mTokensTotal + `{type=~"input|output"}[24h]))`
	assert.Equal(t, expected, query)
}

func TestBuildTrendQueriesHomepageUnchangedWithoutModel(t *testing.T) {
	successQuery, errorQuery := buildTrendQueries("", "60s", "")
	assert.Equal(t, `sum(increase(`+mRequestTotal+`{status="success"}[60s]))`, successQuery)
	assert.Equal(t, `sum(increase(`+mRequestTotal+`{status="error"}[60s]))`, errorQuery)

	successQuery, errorQuery = buildTrendQueries("endpoint", "60s", "")
	assert.Equal(t, `sum by (endpoint) (increase(`+mRequestTotal+`{status="success"}[60s]))`, successQuery)
	assert.Equal(t, `sum by (endpoint) (increase(`+mRequestTotal+`{status="error"}[60s]))`, errorQuery)
}

func TestBuildTrendQueriesFiltersModelAndEscapesLabel(t *testing.T) {
	successQuery, errorQuery := buildTrendQueries("", "60s", `gpt-4.1"`)
	assert.Equal(t, `sum(increase(`+mRequestTotal+`{status="success",model="gpt-4.1\""}[60s]))`, successQuery)
	assert.Equal(t, `sum(increase(`+mRequestTotal+`{status="error",model="gpt-4.1\""}[60s]))`, errorQuery)

	successQuery, errorQuery = buildTrendQueries("endpoint", "300s", "gpt-4")
	assert.Equal(t, `sum by (endpoint) (increase(`+mRequestTotal+`{status="success",model="gpt-4"}[300s]))`, successQuery)
	assert.Equal(t, `sum by (endpoint) (increase(`+mRequestTotal+`{status="error",model="gpt-4"}[300s]))`, errorQuery)
}

func TestResolveTimeRangeAcceptsCustomDuration(t *testing.T) {
	promRange, redisMinutes := resolveTimeRange("90m")

	assert.Equal(t, "90m", promRange)
	assert.Equal(t, 90, redisMinutes)

	promRange, redisMinutes = resolveTimeRange("3h30m")
	assert.Equal(t, "3h30m", promRange)
	assert.Equal(t, 0, redisMinutes)
}

func TestResolveTrendRangeUsesCustomDurationAndSelectedEnd(t *testing.T) {
	end := time.Date(2026, 8, 24, 12, 34, 0, 0, time.Local)
	config := resolveTrendRange("90m", end)

	assert.Equal(t, end, config.end)
	assert.Equal(t, 90, config.numPoints)
	assert.Equal(t, int64(60), config.stepSeconds)
	assert.Equal(t, 90, config.redisMinutes)
}

func TestGetTrendsModelFilterSkipsRedisFallback(t *testing.T) {
	dashboard := &Dashboard{}
	res, err := dashboard.getTrends(context.Background(), "", "1h", "gpt-4")
	assert.NoError(t, err)
	assert.Len(t, res.Series, 1)
	assert.Equal(t, "gpt-4", res.Series[0].Label)
	assert.Equal(t, 60, len(res.Series[0].Success))
	for _, value := range res.Series[0].Total {
		assert.Equal(t, int64(0), value)
	}

	grouped, err := dashboard.getTrends(context.Background(), "endpoint", "1h", "gpt-4")
	assert.NoError(t, err)
	assert.Empty(t, grouped.Series)
}

func TestBuildModelOtpsQueryUsesOutputOverRequestDuration(t *testing.T) {
	query := buildModelOtpsQuery("24h")

	expected := `sum by (model) (increase(` + mTokensTotal + `{type="output"}[24h])) / sum by (model) (increase(` + mRequestDurationSum + `[24h]))`
	assert.Equal(t, expected, query)
}

func TestModelRankingItemJSONIncludesOtpsWithoutDroppingExistingFields(t *testing.T) {
	raw, err := json.Marshal(ModelRankingItem{
		ModelID:      "m1",
		ModelCode:    "gpt-test",
		ModelName:    "GPT Test",
		RequestCount: 10,
		SuccessCount: 9,
		FailCount:    1,
		SuccessRate:  90,
		AvgLatencyMs: 120,
		TotalTokens:  4000,
		TotalCost:    1.25,
		Otps:         32.5,
	})
	assert.NoError(t, err)

	var payload map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, "m1", payload["model_id"])
	assert.Equal(t, "gpt-test", payload["model_code"])
	assert.Equal(t, float64(10), payload["request_count"])
	assert.Equal(t, float64(4000), payload["total_tokens"])
	assert.Equal(t, 1.25, payload["total_cost"])
	assert.Equal(t, 32.5, payload["otps"])
}

func TestGetModelRankingSingleModelNoTrafficReturnsEmpty(t *testing.T) {
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&rschema.Model{}))
	require.NoError(t, db.Create(&rschema.Model{
		ID:        "model-1",
		ModelName: "GPT Test",
		ModelCode: "gpt-test",
		SpaceCode: "default",
		Enabled:   1,
	}).Error)

	dashboard := &Dashboard{DB: db}
	items, err := dashboard.getModelRanking(context.Background(), "request_count", "1h", 10, "gpt-test")
	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestSortModelRankingItemsByOtpsDescendingLeavesRequestCountDefault(t *testing.T) {
	items := []ModelRankingItem{
		{ModelCode: "slow", RequestCount: 100, Otps: 5},
		{ModelCode: "fast", RequestCount: 10, Otps: 40},
		{ModelCode: "mid", RequestCount: 50, Otps: 20},
	}

	sortModelRankingItems(items, "otps")
	assert.Equal(t, []string{"fast", "mid", "slow"}, []string{items[0].ModelCode, items[1].ModelCode, items[2].ModelCode})

	sortModelRankingItems(items, "request_count")
	assert.Equal(t, []string{"slow", "mid", "fast"}, []string{items[0].ModelCode, items[1].ModelCode, items[2].ModelCode})
}

func TestQueryPrometheusRangeReturnsNilOnEmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	}))
	defer ts.Close()

	origAddr := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = ts.URL
	defer func() { config.C.Util.PrometheusServer.Address = origAddr }()

	dashboard := &Dashboard{}
	values, err := dashboard.queryPrometheusRange("sum(increase(test[60s]))", 1000, 2000, 60)
	assert.NoError(t, err)
	assert.Nil(t, values)
}

func TestGetTrendsPreservesSuccessTrafficWhenErrorQueryReturnsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		query := r.URL.Query().Get("query")
		if strings.Contains(query, `status="error"`) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
			return
		}
		// Return 1 success point
		start := r.URL.Query().Get("start")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[%s,"5"]]}]}}`, start)))
	}))
	defer ts.Close()

	origAddr := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = ts.URL
	defer func() { config.C.Util.PrometheusServer.Address = origAddr }()

	dashboard := &Dashboard{}
	res, err := dashboard.getTrendsAt(context.Background(), "", "1h", "glm-5.3", time.Unix(2000, 0))
	assert.NoError(t, err)
	require.Len(t, res.Series, 1)
	assert.Equal(t, "glm-5.3", res.Series[0].Label)
	assert.Equal(t, int64(5), res.Series[0].Success[0])
	assert.Equal(t, int64(0), res.Series[0].Failure[0])
	assert.Equal(t, int64(5), res.Series[0].Total[0])
}

func TestBuildModelPerformanceQueriesFiltersModelAndEscapesLabel(t *testing.T) {
	ttftQuery, otpsQuery := buildModelPerformanceQueries(`gpt-4.1"`, "300s")

	assert.Equal(t,
		`sum(increase(`+mTtftSum+`{model="gpt-4.1\""}[300s])) / sum(increase(`+mTtftCount+`{model="gpt-4.1\""}[300s])) * 1000`,
		ttftQuery,
	)
	assert.Equal(t,
		`sum(increase(`+mTokensTotal+`{model="gpt-4.1\"",type="output"}[300s])) / sum(increase(`+mRequestDurationSum+`{model="gpt-4.1\""}[300s]))`,
		otpsQuery,
	)
}

func TestAlignPrometheusFloatDataPreservesFractionsAndMissingPoints(t *testing.T) {
	values := [][]interface{}{
		{float64(1000), "125.5"},
		{float64(1060), "NaN"},
		{float64(1120), "32.25"},
	}
	dst := make([]*float64, 4)

	alignPrometheusFloatData(dst, values, 1000, 60)

	require.NotNil(t, dst[0])
	assert.Equal(t, 125.5, *dst[0])
	assert.Nil(t, dst[1])
	require.NotNil(t, dst[2])
	assert.Equal(t, 32.25, *dst[2])
	assert.Nil(t, dst[3])
}

func TestModelPerformanceResponseMarshalsUnavailablePointsAsNull(t *testing.T) {
	value := 180.5
	raw, err := json.Marshal(ModelPerformanceTrendsResponse{
		Times:     []string{"10:00", "10:01"},
		AvgTTFTMs: []*float64{&value, nil},
		Otps:      []*float64{nil, &value},
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"times":["10:00","10:01"],
		"avg_ttft_ms":[180.5,null],
		"otps":[null,180.5]
	}`, string(raw))
}

func TestGetModelPerformanceTrendsUsesTrafficTimeGridWhenUnavailable(t *testing.T) {
	end := time.Date(2026, 8, 31, 12, 0, 0, 0, time.Local)
	dashboard := &Dashboard{}

	res, err := dashboard.getModelPerformanceTrendsAt(context.Background(), "gpt-test", "6h", end)
	require.NoError(t, err)

	rangeConfig := resolveTrendRange("6h", end)
	require.Len(t, res.Times, rangeConfig.numPoints)
	require.Len(t, res.AvgTTFTMs, rangeConfig.numPoints)
	require.Len(t, res.Otps, rangeConfig.numPoints)
	assert.Equal(t, end.Add(-time.Duration(rangeConfig.numPoints-1)*time.Duration(rangeConfig.stepSeconds)*time.Second).Format("15:04"), res.Times[0])
	for i := range res.Times {
		assert.Nil(t, res.AvgTTFTMs[i])
		assert.Nil(t, res.Otps[i])
	}
}

func TestQueryModelPerformanceTrendsRequiresModel(t *testing.T) {
	dashboard := &Dashboard{}
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/model-performance-trends?time_range=1h", nil)

	dashboard.QueryModelPerformanceTrends(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response util.ResponseResult
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotNil(t, response.Error)
	assert.Equal(t, "bad_request", response.Error.ID)
}

func TestGetModelPerformanceTrendsFallsBackToRedisForOneHour(t *testing.T) {
	end := time.Date(2026, 8, 31, 12, 0, 0, 0, time.Local)
	minute := end.Unix() / 60
	modelCode := "gpt-test"
	values := map[string]interface{}{
		fmt.Sprintf("aigw:status:model:%s:%d:ttft_sum", modelCode, minute): "500",
		fmt.Sprintf("aigw:status:model:%s:%d:ttft_cnt", modelCode, minute): "2",
		fmt.Sprintf("aigw:status:model:%s:%d:out", modelCode, minute):      "80",
		fmt.Sprintf("aigw:status:model:%s:%d:dur_ms", modelCode, minute):   "2000",
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	redisClient.AddHook(performanceRedisHook{values: values})
	dashboard := &Dashboard{RedisClient: redisClient}

	res, err := dashboard.getModelPerformanceTrendsAt(context.Background(), modelCode, "1h", end)
	require.NoError(t, err)
	require.Len(t, res.AvgTTFTMs, 60)
	require.NotNil(t, res.AvgTTFTMs[59])
	assert.Equal(t, 250.0, *res.AvgTTFTMs[59])
	require.NotNil(t, res.Otps[59])
	assert.Equal(t, 40.0, *res.Otps[59])
	assert.Nil(t, res.AvgTTFTMs[58])
	assert.Nil(t, res.Otps[58])
}

func TestGetModelPerformanceTrendsFallsBackPerMetricWithoutOverwritingPrometheus(t *testing.T) {
	end := time.Now().Truncate(time.Minute)
	modelCode := "partial-prom-model"
	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()
	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:         end.Unix(),
		Model:        modelCode,
		Success:      true,
		OutputTokens: 60,
		TTFTMs:       100,
		DurationMs:   2000,
	})

	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/-/healthy" {
			w.WriteHeader(http.StatusOK)
			return
		}
		query := r.URL.Query().Get("query")
		if strings.Contains(query, mTtftSum) {
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[%d,"350.5"]]}]}}`,
				end.Unix(),
			)))
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	}))
	defer prometheus.Close()

	originalAddress := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = prometheus.URL
	defer func() { config.C.Util.PrometheusServer.Address = originalAddress }()

	dashboard := &Dashboard{}
	res, err := dashboard.getModelPerformanceTrendsAt(context.Background(), modelCode, "1h", end)
	require.NoError(t, err)
	require.NotNil(t, res.AvgTTFTMs[59])
	assert.Equal(t, 350.5, *res.AvgTTFTMs[59])
	require.NotNil(t, res.Otps[59])
	assert.Equal(t, 30.0, *res.Otps[59])
}

func TestGetModelPerformanceTrendsDoesNotPartiallyFallbackLongWindow(t *testing.T) {
	end := time.Now().Truncate(time.Minute)
	modelCode := "long-window-model"
	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()
	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:         end.Unix(),
		Model:        modelCode,
		Success:      true,
		OutputTokens: 60,
		TTFTMs:       100,
		DurationMs:   2000,
	})

	dashboard := &Dashboard{}
	res, err := dashboard.getModelPerformanceTrendsAt(context.Background(), modelCode, "6h", end)
	require.NoError(t, err)
	for i := range res.Times {
		assert.Nil(t, res.AvgTTFTMs[i])
		assert.Nil(t, res.Otps[i])
	}
}

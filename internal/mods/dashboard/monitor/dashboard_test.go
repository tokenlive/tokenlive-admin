package monitor

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

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
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

func TestGetOverviewFallsBackToMemoryLatencyAndTTFT(t *testing.T) {
	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()

	originalAddress := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = ""
	defer func() { config.C.Util.PrometheusServer.Address = originalAddress }()

	now := time.Now()
	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:       now.Add(-time.Minute).Unix(),
		Model:      "gpt-test",
		Success:    true,
		TTFTMs:     100,
		DurationMs: 1000,
	})
	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:       now.Unix(),
		Model:      "gpt-test",
		Success:    true,
		TTFTMs:     300,
		DurationMs: 3000,
	})

	dashboard := &Reader{}
	overview, err := dashboard.getOverview(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2000.0, *overview.AvgLatencyMs)
	assert.Equal(t, 200.0, *overview.AvgTTFTMs)
}

func TestAggregateRedisTrendValues(t *testing.T) {
	series := trendSeries{
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

	assert.Equal(t, []int64{4, 12, 9}, series.Success)
	assert.Equal(t, []int64{6, 14, 10}, series.Failure)
	assert.Equal(t, []int64{10, 26, 19}, series.Total)
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

func TestGetTrendsModelFilterSkipsRedisFallback(t *testing.T) {
	dashboard := &Reader{}
	res, err := readTrafficAt(t, dashboard, context.Background(), "", "1h", "gpt-4", time.Now())
	assert.NoError(t, err)
	assert.Len(t, res.Series, 1)
	assert.Equal(t, "gpt-4", res.Series[0].Label)
	assert.Equal(t, 60, len(res.Series[0].Success))
	for _, value := range res.Series[0].Total {
		assert.Equal(t, int64(0), value)
	}

	grouped, err := readTrafficAt(t, dashboard, context.Background(), "endpoint", "1h", "gpt-4", time.Now())
	assert.NoError(t, err)
	assert.Empty(t, grouped.Series)
}

func TestGetTrendsFallsBackToSevenDayMemoryDataGroupedByModelAndProvider(t *testing.T) {
	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()

	originalAddress := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = ""
	defer func() { config.C.Util.PrometheusServer.Address = originalAddress }()

	end := time.Now().Truncate(time.Minute)
	for _, metric := range []metrics.RequestMetric{
		{Time: end.Add(-6 * 24 * time.Hour).Unix(), Model: "gpt-a", Provider: "openai", Success: true},
		{Time: end.Add(-time.Minute).Unix(), Model: "gpt-b", Provider: "anthropic", Success: false},
	} {
		metrics.GlobalStore.Record(metric)
	}

	dashboard := &Reader{}
	models, err := readTrafficAt(t, dashboard, context.Background(), "model", "7d", "", end)
	require.NoError(t, err)
	require.Len(t, models.Series, 2)
	assert.Equal(t, "gpt-a", models.Series[0].Label)
	assert.Equal(t, int64(1), sumInt64(models.Series[0].Success))
	assert.Equal(t, "gpt-b", models.Series[1].Label)
	assert.Equal(t, int64(1), sumInt64(models.Series[1].Failure))

	providers, err := readTrafficAt(t, dashboard, context.Background(), "provider", "7d", "", end)
	require.NoError(t, err)
	require.Len(t, providers.Series, 2)
	assert.Equal(t, "anthropic", providers.Series[0].Label)
	assert.Equal(t, int64(1), sumInt64(providers.Series[0].Failure))
	assert.Equal(t, "openai", providers.Series[1].Label)
	assert.Equal(t, int64(1), sumInt64(providers.Series[1].Success))
}

func sumInt64(values []int64) int64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return total
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

	dashboard := &Reader{db: db}
	items, err := readModelsAt(t, dashboard, context.Background(), "request_count", "1h", 10, "gpt-test", time.Now())
	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestGetModelRankingFallsBackToDetailedMemoryMetricsForTwentyFourHours(t *testing.T) {
	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()

	originalAddress := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = ""
	defer func() { config.C.Util.PrometheusServer.Address = originalAddress }()

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

	end := time.Now().Truncate(time.Minute)
	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:         end.Add(-23 * time.Hour).Unix(),
		Model:        "gpt-test",
		Provider:     "openai",
		Success:      true,
		InputTokens:  10,
		OutputTokens: 20,
		Cost:         0.1,
		TTFTMs:       100,
		DurationMs:   1000,
	})
	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:         end.Add(-time.Minute).Unix(),
		Model:        "gpt-test",
		Provider:     "openai",
		Success:      true,
		InputTokens:  30,
		OutputTokens: 40,
		Cost:         0.2,
		TTFTMs:       300,
		DurationMs:   3000,
	})

	dashboard := &Reader{db: db}
	items, err := readModelsAt(t, dashboard, context.Background(), "request_count", "24h", 10, "", end)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, int64(2), items[0].RequestCount)
	assert.Equal(t, 100.0, items[0].SuccessRate)
	assert.Equal(t, 2000.0, items[0].AvgLatencyMs)
	assert.Equal(t, 200.0, items[0].AvgTTFTMs)
	assert.Equal(t, int64(100), items[0].TotalTokens)
	assert.InDelta(t, 0.3, items[0].TotalCost, 0.000001)
	assert.Equal(t, 15.0, items[0].Otps)
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

	dashboard := &Reader{}
	values, err := dashboard.queryPrometheusRange(context.Background(), "sum(increase(test[60s]))", 1000, 2000, 60)
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

	dashboard := &Reader{}
	res, err := readTrafficAt(t, dashboard, context.Background(), "", "1h", "glm-5.3", time.Unix(2000, 0))
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
	raw, err := json.Marshal(Performance{
		AvgTTFTMs: []*float64{&value, nil},
		Otps:      []*float64{nil, &value},
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"avg_ttft_ms":[180.5,null],
		"otps":[null,180.5]
	}`, string(raw))
}

func TestGetModelPerformanceTrendsFallsBackToRedisForOneHour(t *testing.T) {
	end := time.Date(2026, 8, 31, 12, 0, 0, 0, time.Local)
	minute := end.Add(-time.Minute).Unix() / 60
	modelCode := "gpt-test"
	values := map[string]interface{}{
		gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricTTFTSum):      "500",
		gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricTTFTCount):    "2",
		gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricOutputTokens): "80",
		gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricDurationMs):   "2000",
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	redisClient.AddHook(performanceRedisHook{values: values})
	dashboard := &Reader{redis: redisClient}

	res, err := readPerformanceAt(t, dashboard, context.Background(), modelCode, "1h", end)
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
		Time:         end.Add(-time.Minute).Unix(),
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

	dashboard := &Reader{}
	res, err := readPerformanceAt(t, dashboard, context.Background(), modelCode, "1h", end)
	require.NoError(t, err)
	require.NotNil(t, res.AvgTTFTMs[59])
	assert.Equal(t, 350.5, *res.AvgTTFTMs[59])
	require.NotNil(t, res.Otps[59])
	assert.Equal(t, 30.0, *res.Otps[59])
}

func TestGetModelPerformanceTrendsFallsBackToMemoryForLongWindow(t *testing.T) {
	end := time.Now().Truncate(time.Minute)
	modelCode := "long-window-model"
	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()
	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:         end.Add(-time.Minute).Unix(),
		Model:        modelCode,
		Success:      true,
		OutputTokens: 60,
		TTFTMs:       100,
		DurationMs:   2000,
	})

	dashboard := &Reader{}
	res, err := readPerformanceAt(t, dashboard, context.Background(), modelCode, "6h", end)
	require.NoError(t, err)
	require.NotNil(t, res.AvgTTFTMs[len(res.AvgTTFTMs)-1])
	assert.Equal(t, 100.0, *res.AvgTTFTMs[len(res.AvgTTFTMs)-1])
	require.NotNil(t, res.Otps[len(res.Otps)-1])
	assert.Equal(t, 30.0, *res.Otps[len(res.Otps)-1])
}

func TestGetTrendsEndpointFallsBackToRedis(t *testing.T) {
	end := time.Date(2026, 8, 31, 12, 0, 0, 0, time.Local)
	minute := end.Add(-time.Minute).Unix() / 60
	modelCode := "gpt-test"
	endpointID := "ep-1"

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&rschema.Model{}, &rschema.Endpoint{}))
	require.NoError(t, db.Create(&rschema.Model{
		ID:        "model-1",
		ModelName: "GPT Test",
		ModelCode: modelCode,
		SpaceCode: "default",
		Enabled:   1,
		Deleted:   "0",
	}).Error)
	require.NoError(t, db.Create(&rschema.Endpoint{
		ID:      endpointID,
		Code:    "ep-test",
		ModelID: "model-1",
		Enabled: 1,
		Deleted: "0",
	}).Error)

	values := map[string]interface{}{
		gatewaycontract.Keys.Status.Endpoint(endpointID, minute, gatewaycontract.MetricSuccess): "5",
		gatewaycontract.Keys.Status.Endpoint(endpointID, minute, gatewaycontract.MetricFailure): "1",
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	redisClient.AddHook(performanceRedisHook{values: values})
	dashboard := &Reader{db: db, redis: redisClient}

	res, err := readTrafficAt(t, dashboard, context.Background(), "endpoint", "1h", modelCode, end)
	require.NoError(t, err)
	require.Len(t, res.Series, 1)
	assert.Equal(t, endpointID, res.Series[0].Label)
	assert.Equal(t, int64(5), res.Series[0].Success[59])
	assert.Equal(t, int64(1), res.Series[0].Failure[59])
	assert.Equal(t, int64(6), res.Series[0].Total[59])
}

func TestGetTrendsEndpointFallsBackToMemory(t *testing.T) {
	end := time.Now().Truncate(time.Minute)
	modelCode := "mem-model"
	endpointID := "ep-mem-1"

	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()

	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:    end.Add(-time.Minute).Unix(),
		Model:   modelCode,
		Success: true,
		Attempts: []struct {
			EndpointID   string `json:"endpoint_id"`
			Provider     string `json:"provider,omitempty"`
			ProviderCode string `json:"provider_code,omitempty"`
			Success      bool   `json:"success"`
		}{
			{EndpointID: endpointID, Success: true},
			{EndpointID: endpointID, Success: false},
		},
	})

	dashboard := &Reader{}
	res, err := readTrafficAt(t, dashboard, context.Background(), "endpoint", "1h", modelCode, end)
	require.NoError(t, err)
	require.Len(t, res.Series, 1)
	assert.Equal(t, endpointID, res.Series[0].Label)
	assert.Equal(t, int64(1), res.Series[0].Success[59])
	assert.Equal(t, int64(1), res.Series[0].Failure[59])
	assert.Equal(t, int64(2), res.Series[0].Total[59])
}

func TestGetTrendsModelFallsBackToRedis(t *testing.T) {
	end := time.Date(2026, 8, 31, 12, 0, 0, 0, time.Local)
	minute := end.Add(-time.Minute).Unix() / 60
	modelCode := "gpt-test-model"

	values := map[string]interface{}{
		gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricSuccess): "10",
		gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricFailure): "2",
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	redisClient.AddHook(performanceRedisHook{values: values})
	dashboard := &Reader{redis: redisClient}

	res, err := readTrafficAt(t, dashboard, context.Background(), "", "1h", modelCode, end)
	require.NoError(t, err)
	require.Len(t, res.Series, 1)
	assert.Equal(t, modelCode, res.Series[0].Label)
	assert.Equal(t, int64(10), res.Series[0].Success[59])
	assert.Equal(t, int64(2), res.Series[0].Failure[59])
	assert.Equal(t, int64(12), res.Series[0].Total[59])
}

func sourceWindow(t *testing.T, ctx context.Context, span string, end time.Time) (context.Context, windowPlan) {
	t.Helper()
	p, err := compileWindow(WindowQuery{Range: span}, end, end, end.Location())
	require.NoError(t, err)
	ctx, _ = withPlan(ctx, p)
	return ctx, p
}
func readTrafficAt(t *testing.T, r *Reader, ctx context.Context, group, span, model string, end time.Time) (*trafficResult, error) {
	t.Helper()
	ctx, p := sourceWindow(t, ctx, span, end)
	return r.readTraffic(ctx, group, model, p)
}
func readPerformanceAt(t *testing.T, r *Reader, ctx context.Context, model, span string, end time.Time) (*Performance, error) {
	t.Helper()
	ctx, p := sourceWindow(t, ctx, span, end)
	return r.readPerformance(ctx, model, p)
}
func readModelsAt(t *testing.T, r *Reader, ctx context.Context, sort, span string, limit int, model string, end time.Time) ([]ModelRankingItem, error) {
	t.Helper()
	ctx, p := sourceWindow(t, ctx, span, end)
	return r.readModels(ctx, sort, limit, model, p)
}

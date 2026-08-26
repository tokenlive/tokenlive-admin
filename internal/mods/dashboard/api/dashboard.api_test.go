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


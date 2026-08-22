package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
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

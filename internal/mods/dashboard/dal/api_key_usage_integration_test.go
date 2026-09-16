package dal

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

// This test never uses a production configuration. Its explicit test account
// must be allowed to create/drop a dedicated, randomly named test database.
func TestUsageReaderClickHouseIntegration(t *testing.T) {
	addr := os.Getenv("API_KEY_USAGE_CH_TEST_ADDR")
	if addr == "" {
		t.Skip("API_KEY_USAGE_CH_TEST_ADDR is not configured; no live ClickHouse verification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Username: os.Getenv("API_KEY_USAGE_CH_TEST_USER"),
			Password: os.Getenv("API_KEY_USAGE_CH_TEST_PASSWORD"),
		},
		DialTimeout: 3 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	database := "api_key_usage_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.Regexp(t, `^api_key_usage_test_[a-f0-9]{32}$`, database)
	require.NoError(t, conn.Exec(ctx, "CREATE DATABASE "+database))
	t.Cleanup(func() {
		if !regexp.MustCompile(`^api_key_usage_test_[a-f0-9]{32}$`).MatchString(database) {
			t.Fatal("refusing to clean an unowned database")
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		require.NoError(t, conn.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+database))
	})
	// Same fields and replacing/sorting key as Gateway; omit retention so fixed
	// fixture timestamps are not removed by background TTL processing.
	require.NoError(t, conn.Exec(ctx, `CREATE TABLE `+database+`.access_logs (
		request_id String, time DateTime64(3), tenant_id LowCardinality(String),
		user_id LowCardinality(String), session_id String, api_key String,
		workspace_id LowCardinality(String), api_key_id LowCardinality(String), api_key_hash String,
		client_ip String, original_model LowCardinality(String), model LowCardinality(String),
		provider LowCardinality(String), endpoint_id LowCardinality(String), is_stream UInt8,
		attempts UInt8, fallback_chain Array(String), status_code Int16, latency_ms UInt32,
		ttft_ms UInt32, error_message String, input_tokens UInt32, output_tokens UInt32,
		cached_tokens UInt32, cache_creation_tokens UInt32, cost Decimal(18,9)
	) ENGINE = ReplacingMergeTree(time) PARTITION BY toYYYYMMDD(time)
	ORDER BY (tenant_id, model, request_id, time)`))
	start := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	type row struct {
		id, hash, user   string
		seconds          int
		status           int16
		input, output    uint32
		cached, creation uint32
		cost             string
	}
	rows := []row{
		{"a", "hash-a", "u", 0, 200, 10, 20, 5, 2, "0.1"},
		{"a", "hash-a", "u", 0, 200, 10, 20, 5, 2, "0.1"},
		{"a", "hash-b", "u", 1, 200, 40, 30, 0, 0, "0.2"},
		{"c", "hash-a", "other", 2, 499, 0, 0, 0, 0, "0"},
		{"d", "", "", 3, 200, 0, 0, 0, 0, "0"},
		{"end", "excluded", "", 3600, 200, 999, 999, 0, 0, "9"},
	}
	for _, row := range rows {
		require.NoError(t, conn.Exec(ctx, `INSERT INTO `+database+`.access_logs
			(request_id,time,api_key_hash,user_id,api_key,status_code,input_tokens,output_tokens,cached_tokens,cache_creation_tokens,cost)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			row.id, start.Add(time.Duration(row.seconds)*time.Second), row.hash, row.user, "sk-****same", row.status,
			row.input, row.output, row.cached, row.creation, decimal.RequireFromString(row.cost)))
	}
	reader, cleanup := NewClickHouseReader(config.ClickHouseConfig{
		Enabled: true, Addr: []string{addr}, Database: database,
		Username: os.Getenv("API_KEY_USAGE_CH_TEST_USER"), Password: os.Getenv("API_KEY_USAGE_CH_TEST_PASSWORD"),
	})
	t.Cleanup(cleanup)
	result, err := reader.Read(ctx, schema.Window{Start: start, End: start.Add(time.Hour), Timezone: "UTC"})
	require.NoError(t, err)
	var all schema.Totals
	for _, candidate := range result {
		all.Add(candidate.Totals)
	}
	require.Equal(t, uint64(4), all.Requests)
	require.Equal(t, uint64(3), all.Success)
	require.Equal(t, uint64(100), all.Tokens())
	require.Equal(t, uint64(5), all.Cached)
	require.True(t, decimal.RequireFromString("0.3").Equal(all.Cost))
}

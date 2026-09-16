package config

import (
	"testing"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/require"
)

func TestClickHouseDefaultsAndSafeConfigPrint(t *testing.T) {
	cfg := new(Config)
	require.NoError(t, defaults.Set(cfg))
	require.False(t, cfg.Storage.ClickHouse.Enabled)
	require.Equal(t, 3, cfg.Storage.ClickHouse.DialTimeoutSeconds)
	require.Equal(t, 5, cfg.Storage.ClickHouse.QueryTimeoutSeconds)
	cfg.Storage.ClickHouse.Password = "fixture-clickhouse-secret"
	require.NotContains(t, cfg.String(), "fixture-clickhouse-secret")
	require.Equal(t, "fixture-clickhouse-secret", cfg.Storage.ClickHouse.Password)
}

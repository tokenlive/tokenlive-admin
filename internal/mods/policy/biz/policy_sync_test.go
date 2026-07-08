package biz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveRedisKeyAndField_NoGlobalPolicyKey(t *testing.T) {
	key, field, ok := resolveRedisKeyAndField("", "", "")

	require.False(t, ok)
	require.Empty(t, key)
	require.Empty(t, field)
}

func TestResolveRedisKeyAndField_ModelAndScopedKeys(t *testing.T) {
	key, field, ok := resolveRedisKeyAndField("", "", "gpt-5")
	require.True(t, ok)
	require.Equal(t, "aigw:policies:model:gpt-5", key)
	require.Equal(t, "*", field)

	key, field, ok = resolveRedisKeyAndField("tenant-a", "", "gpt-5")
	require.True(t, ok)
	require.Equal(t, "aigw:policies:tenant:tenant-a", key)
	require.Equal(t, "gpt-5", field)

	key, field, ok = resolveRedisKeyAndField("tenant-a", "", "")
	require.True(t, ok)
	require.Equal(t, "aigw:policies:tenant:tenant-a", key)
	require.Equal(t, "*", field)
}

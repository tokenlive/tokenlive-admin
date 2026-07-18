package adminapp_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestNotifyConfigChanged_InvokesListeners(t *testing.T) {
	util.ResetConfigChangeListeners()
	t.Cleanup(util.ResetConfigChangeListeners)

	var n atomic.Int32
	util.OnConfigChanged(func(ctx context.Context, kind util.ConfigChangeKind, keys ...string) {
		n.Add(1)
		require.Equal(t, util.ConfigChangeAPIKeys, kind)
	})
	util.NotifyConfigChanged(context.Background(), util.ConfigChangeAPIKeys)
	require.Equal(t, int32(1), n.Load())
}

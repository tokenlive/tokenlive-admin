package biz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

func TestSyncPolicyChangeAndLogCallsSyncer(t *testing.T) {
	syncer := &fakePolicyChangeSyncer{}

	err := syncPolicyChangeAndLog(context.Background(), syncer, "loadbalance", "update", "tenant", "AAA", "model-1")

	require.NoError(t, err)
	require.Equal(t, []fakePolicyChangeCall{{
		scopeType: "tenant",
		scopeCode: "AAA",
		modelID:   "model-1",
	}}, syncer.calls)
}

func TestSyncPolicyChangeAndLogReturnsSyncError(t *testing.T) {
	wantErr := errors.InternalServerError("", "sync failed")
	syncer := &fakePolicyChangeSyncer{err: wantErr}

	err := syncPolicyChangeAndLog(context.Background(), syncer, "loadbalance", "update", "tenant", "AAA", "model-1")

	require.ErrorIs(t, err, wantErr)
}

type fakePolicyChangeSyncer struct {
	calls []fakePolicyChangeCall
	err   error
}

type fakePolicyChangeCall struct {
	scopeType string
	scopeCode string
	modelID   string
}

func (f *fakePolicyChangeSyncer) SyncPolicyChange(ctx context.Context, scopeType, scopeCode, modelID string) error {
	f.calls = append(f.calls, fakePolicyChangeCall{
		scopeType: scopeType,
		scopeCode: scopeCode,
		modelID:   modelID,
	})
	return f.err
}

package biz

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type PolicyChangeSyncer interface {
	SyncPolicyChange(ctx context.Context, scopeType, scopeCode, modelID string) error
}

func syncPolicyChangeAndLog(ctx context.Context, syncer PolicyChangeSyncer, policyType, operation, scopeType, scopeCode, modelID string) (err error) {
	start := time.Now()
	fields := []zap.Field{
		zap.String("policy_type", policyType),
		zap.String("operation", operation),
		zap.String("scope_type", scopeType),
		zap.String("scope_code", scopeCode),
		zap.String("model_id", modelID),
	}
	policyRedisSyncLogger(ctx).Info("Policy change triggers Redis sync", fields...)
	defer func() {
		doneFields := append(fields, zap.Duration("cost", time.Since(start)))
		if err != nil {
			doneFields = append(doneFields, zap.Error(err))
			policyRedisSyncLogger(ctx).Error("Policy change Redis sync returned error", doneFields...)
			return
		}
		policyRedisSyncLogger(ctx).Info("Policy change Redis sync returned success", doneFields...)
	}()

	if syncer == nil {
		policyRedisSyncLogger(ctx).Warn("Policy change Redis sync skipped because syncer is nil", fields...)
		return nil
	}
	return syncer.SyncPolicyChange(ctx, scopeType, scopeCode, modelID)
}

package biz

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// toPtrString marshals v to JSON and returns a *string pointer, or nil on error/nil input.
func toPtrString(v interface{}) *string {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// AuditLog business logic layer
type AuditLog struct {
	Trans       *util.Trans
	AuditLogDAL *dal.AuditLog
}

// Query audit logs.
func (a *AuditLog) Query(ctx context.Context, params schema.AuditLogQueryParam) (*schema.AuditLogQueryResult, error) {
	params.Pagination = true
	return a.AuditLogDAL.Query(ctx, params, schema.AuditLogQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
}

// Get the specified audit log.
func (a *AuditLog) Get(ctx context.Context, id string) (*schema.AuditLog, error) {
	log, err := a.AuditLogDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if log == nil {
		return nil, errors.NotFound("", "Audit log not found")
	}
	return log, nil
}

// Create creates a new audit log entry.
func (a *AuditLog) Create(ctx context.Context, formItem *schema.AuditLogForm) (*schema.AuditLog, error) {
	log := &schema.AuditLog{
		ID:        util.NewXID(),
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(log); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.AuditLogDAL.Create(ctx, log)
	})
	if err != nil {
		return nil, err
	}
	return log, nil
}

// RecordAction is a convenience method to record an audit action.
func (a *AuditLog) RecordAction(ctx context.Context, action, resourceType, resourceID, resourceName string, beforeData, afterData interface{}) {
	a.RecordActionWithTenant(ctx, util.FromTenant(ctx), action, resourceType, resourceID, resourceName, beforeData, afterData)
}

// RecordActionWithTenant records an audit action for a specific resource tenant.
func (a *AuditLog) RecordActionWithTenant(ctx context.Context, tenantCode, action, resourceType, resourceID, resourceName string, beforeData, afterData interface{}) {
	form := &schema.AuditLogForm{
		TenantCode:   tenantCode,
		ActorUserID:  util.FromUserID(ctx),
		ActorName:    util.FromUsername(ctx),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		BeforeData:   toPtrString(beforeData),
		AfterData:    toPtrString(afterData),
		IP:           util.FromClientIP(ctx),
		UserAgent:    util.FromUserAgent(ctx),
	}
	// Audit logging failure should not break the main flow
	_, _ = a.Create(ctx, form)
}

package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetAuditLogDB returns the database instance for AuditLog.
func GetAuditLogDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.AuditLog))
}

// AuditLog data access layer
type AuditLog struct {
	DB *gorm.DB
}

// Query audit logs from the database.
func (a *AuditLog) Query(ctx context.Context, params schema.AuditLogQueryParam, opts ...schema.AuditLogQueryOptions) (*schema.AuditLogQueryResult, error) {
	var opt schema.AuditLogQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetAuditLogDB(ctx, a.DB)
	if v := params.TenantCode; len(v) > 0 {
		db = db.Where("tenant_code = ?", v)
	}
	if v := params.ActorUserID; len(v) > 0 {
		db = db.Where("actor_user_id = ?", v)
	}
	if v := params.Action; len(v) > 0 {
		db = db.Where("action = ?", v)
	}
	if v := params.ResourceType; len(v) > 0 {
		db = db.Where("resource_type = ?", v)
	}
	if v := params.ResourceID; len(v) > 0 {
		db = db.Where("resource_id = ?", v)
	}
	if v := params.TraceID; len(v) > 0 {
		db = db.Where("trace_id = ?", v)
	}
	if v := params.StartTime; len(v) > 0 {
		db = db.Where("created_at >= ?", v)
	}
	if v := params.EndTime; len(v) > 0 {
		db = db.Where("created_at <= ?", v)
	}

	var list schema.AuditLogs
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.AuditLogQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get the specified audit log from the database.
func (a *AuditLog) Get(ctx context.Context, id string) (*schema.AuditLog, error) {
	item := new(schema.AuditLog)
	ok, err := util.FindOne(ctx, GetAuditLogDB(ctx, a.DB).Where("id=?", id), util.QueryOptions{}, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Create a new audit log entry.
func (a *AuditLog) Create(ctx context.Context, item *schema.AuditLog) error {
	result := GetAuditLogDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

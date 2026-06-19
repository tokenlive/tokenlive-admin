package dal

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetEventLogDB returns the base DB query for event_log records.
func GetEventLogDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.EventLog))
}

// EventLog data access layer.
type EventLog struct {
	DB *gorm.DB
}

// Query event logs from the database based on the provided parameters and options.
func (a *EventLog) Query(ctx context.Context, params schema.EventQueryParam, opts ...schema.EventQueryOptions) (*schema.EventQueryResult, error) {
	var opt schema.EventQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetEventLogDB(ctx, a.DB)
	if v := params.EventType; v != "" {
		db = db.Where("event_type = ?", v)
	}
	if v := params.TenantCode; v != "" {
		db = db.Where("tenant_code = ?", v)
	}
	if v := params.ModelCode; v != "" {
		db = db.Where("model_code = ?", v)
	}
	if v := params.EndpointID; v != "" {
		db = db.Where("endpoint_id = ?", v)
	}
	if v := params.PolicyID; v != "" {
		db = db.Where("policy_id = ?", v)
	}
	if v := params.StartTime; v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			db = db.Where("event_time >= ?", t)
		}
	}
	if v := params.EndTime; v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			db = db.Where("event_time <= ?", t)
		}
	}

	var list schema.EventLogs
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.EventQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Create inserts a single event log record.
func (a *EventLog) Create(ctx context.Context, item *schema.EventLog) error {
	result := GetEventLogDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// CreateInBatches inserts multiple event log records in batches.
func (a *EventLog) CreateInBatches(ctx context.Context, items []*schema.EventLog) error {
	if len(items) == 0 {
		return nil
	}
	result := GetEventLogDB(ctx, a.DB).CreateInBatches(items, 100)
	return errors.WithStack(result.Error)
}

// DeleteBefore deletes event logs older than the given cutoff time (physical delete).
func (a *EventLog) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := GetEventLogDB(ctx, a.DB).Where("event_time < ?", cutoff).Delete(&schema.EventLog{})
	if result.Error != nil {
		return 0, errors.WithStack(result.Error)
	}
	return result.RowsAffected, nil
}

// CountByEventType counts events grouped by event_type within the given time range.
func (a *EventLog) CountByEventType(ctx context.Context, startTime, endTime time.Time) ([]schema.EventTypeCount, error) {
	var counts []schema.EventTypeCount
	err := GetEventLogDB(ctx, a.DB).
		Select("event_type, COUNT(*) as count").
		Where("event_time >= ? AND event_time <= ?", startTime, endTime).
		Group("event_type").
		Scan(&counts).Error
	return counts, errors.WithStack(err)
}

// CountTotal counts total events within the given time range.
func (a *EventLog) CountTotal(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	var count int64
	err := GetEventLogDB(ctx, a.DB).
		Where("event_time >= ? AND event_time <= ?", startTime, endTime).
		Count(&count).Error
	return count, errors.WithStack(err)
}

// TrendByHour returns hourly event counts for the given time range.
func (a *EventLog) TrendByHour(ctx context.Context, startTime, endTime time.Time) ([]schema.TrendPoint, error) {
	var points []schema.TrendPoint
	err := GetEventLogDB(ctx, a.DB).
		Select(`DATE_FORMAT(event_time, '%Y-%m-%d %H:00:00') as time,
			SUM(CASE WHEN event_type = 'circuit_break' THEN 1 ELSE 0 END) as circuit_break,
			SUM(CASE WHEN event_type = 'rate_limit' THEN 1 ELSE 0 END) as rate_limit,
			SUM(CASE WHEN event_type = 'invocation_fail' THEN 1 ELSE 0 END) as invocation_fail,
			SUM(CASE WHEN event_type = 'lb_switch' THEN 1 ELSE 0 END) as lb_switch`).
		Where("event_time >= ? AND event_time <= ?", startTime, endTime).
		Group("time").
		Order("time ASC").
		Scan(&points).Error
	return points, errors.WithStack(err)
}

// TrendByDay returns daily event counts for the given time range.
func (a *EventLog) TrendByDay(ctx context.Context, startTime, endTime time.Time) ([]schema.TrendPoint, error) {
	var points []schema.TrendPoint
	err := GetEventLogDB(ctx, a.DB).
		Select(`DATE_FORMAT(event_time, '%Y-%m-%d') as time,
			SUM(CASE WHEN event_type = 'circuit_break' THEN 1 ELSE 0 END) as circuit_break,
			SUM(CASE WHEN event_type = 'rate_limit' THEN 1 ELSE 0 END) as rate_limit,
			SUM(CASE WHEN event_type = 'invocation_fail' THEN 1 ELSE 0 END) as invocation_fail,
			SUM(CASE WHEN event_type = 'lb_switch' THEN 1 ELSE 0 END) as lb_switch`).
		Where("event_time >= ? AND event_time <= ?", startTime, endTime).
		Group("time").
		Order("time ASC").
		Scan(&points).Error
	return points, errors.WithStack(err)
}

// TopTenants returns the top N tenants by event count.
func (a *EventLog) TopTenants(ctx context.Context, startTime, endTime time.Time, limit int) ([]schema.RankingItem, error) {
	var items []schema.RankingItem
	err := GetEventLogDB(ctx, a.DB).
		Select("tenant_code as name, COUNT(*) as count").
		Where("event_time >= ? AND event_time <= ? AND tenant_code != ''", startTime, endTime).
		Group("tenant_code").
		Order("count DESC").
		Limit(limit).
		Scan(&items).Error
	return items, errors.WithStack(err)
}

// TopModels returns the top N models by event count.
func (a *EventLog) TopModels(ctx context.Context, startTime, endTime time.Time, limit int) ([]schema.RankingItem, error) {
	var items []schema.RankingItem
	err := GetEventLogDB(ctx, a.DB).
		Select("model_code as name, COUNT(*) as count").
		Where("event_time >= ? AND event_time <= ? AND model_code != ''", startTime, endTime).
		Group("model_code").
		Order("count DESC").
		Limit(limit).
		Scan(&items).Error
	return items, errors.WithStack(err)
}

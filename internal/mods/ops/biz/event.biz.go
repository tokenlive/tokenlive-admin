package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// EventBiz handles business logic for event operations.
type EventBiz struct {
	EventDAL *dal.EventLog
}

// QueryEvents queries event logs with pagination and filtering.
func (a *EventBiz) QueryEvents(ctx context.Context, params schema.EventQueryParam) (*schema.EventQueryResult, error) {
	params.Pagination = true
	return a.EventDAL.Query(ctx, params, schema.EventQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "event_time", Direction: util.DESC},
			},
		},
	})
}

// GetStatistics returns aggregated event statistics for the given time range.
func (a *EventBiz) GetStatistics(ctx context.Context, timeRange string) (*schema.EventStatistics, error) {
	startTime, endTime, trendFunc := resolveTimeRange(timeRange)

	// Count by event type
	typeCounts, err := a.EventDAL.CountByEventType(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	stats := &schema.EventStatistics{}
	for _, tc := range typeCounts {
		stats.TotalEvents += tc.Count
		switch tc.EventType {
		case schema.EventTypeCircuitBreak:
			stats.CircuitBreakCount = tc.Count
		case schema.EventTypeRateLimit:
			stats.RateLimitCount = tc.Count
		case schema.EventTypeInvocationFail:
			stats.InvocationFailCount = tc.Count
		case schema.EventTypeLBSwitch:
			stats.LBSwitchCount = tc.Count
		}
	}

	// Trend data
	stats.Trend, err = trendFunc(ctx, a.EventDAL, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Rankings (top 10)
	stats.TenantRanking, err = a.EventDAL.TopTenants(ctx, startTime, endTime, 10)
	if err != nil {
		return nil, err
	}

	stats.ModelRanking, err = a.EventDAL.TopModels(ctx, startTime, endTime, 10)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// resolveTimeRange returns start/end times and the appropriate trend function.
func resolveTimeRange(timeRange string) (time.Time, time.Time, func(ctx context.Context, dal *dal.EventLog, start, end time.Time) ([]schema.TrendPoint, error)) {
	now := time.Now()
	endTime := now

	var startTime time.Time
	var trendFunc func(ctx context.Context, dal *dal.EventLog, start, end time.Time) ([]schema.TrendPoint, error)

	switch timeRange {
	case "1h":
		startTime = now.Add(-1 * time.Hour)
		trendFunc = func(ctx context.Context, d *dal.EventLog, start, end time.Time) ([]schema.TrendPoint, error) {
			return d.TrendByHour(ctx, start, end)
		}
	case "6h":
		startTime = now.Add(-6 * time.Hour)
		trendFunc = func(ctx context.Context, d *dal.EventLog, start, end time.Time) ([]schema.TrendPoint, error) {
			return d.TrendByHour(ctx, start, end)
		}
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		trendFunc = func(ctx context.Context, d *dal.EventLog, start, end time.Time) ([]schema.TrendPoint, error) {
			return d.TrendByHour(ctx, start, end)
		}
	case "7d":
		startTime = now.AddDate(0, 0, -7)
		trendFunc = func(ctx context.Context, d *dal.EventLog, start, end time.Time) ([]schema.TrendPoint, error) {
			return d.TrendByDay(ctx, start, end)
		}
	default: // "24h"
		startTime = now.Add(-24 * time.Hour)
		trendFunc = func(ctx context.Context, d *dal.EventLog, start, end time.Time) ([]schema.TrendPoint, error) {
			return d.TrendByHour(ctx, start, end)
		}
	}

	return startTime, endTime, trendFunc
}

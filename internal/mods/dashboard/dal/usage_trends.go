package dal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

const usageTrendsSQL = `
SELECT
    toUInt64(intDiv(toUnixTimestamp64Milli(time) - ?, ?)) AS bucket_index,
    count(), countIf(status_code >= 200 AND status_code < 400),
    sum(input_tokens), sum(output_tokens), sum(cached_tokens), sum(cache_creation_tokens),
    sum(latency_ms), sumIf(ttft_ms, is_stream = 1 AND ttft_ms > 0),
    countIf(is_stream = 1 AND ttft_ms > 0)
FROM access_logs FINAL
WHERE time >= fromUnixTimestamp64Milli(?) AND time < fromUnixTimestamp64Milli(?)
GROUP BY bucket_index
ORDER BY bucket_index`

func (r *ClickHouseReader) ReadUsageTrends(ctx context.Context, window schema.Window, bucketSeconds int64) ([]schema.UsageTrendsRow, error) {
	if !r.Enabled() {
		return nil, nil
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if bucketSeconds <= 0 || window.End.Before(window.Start) {
		return nil, errors.New("invalid trends window")
	}
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Plain time.Time parameters are bound at second precision by the driver.
	// Bind epoch milliseconds explicitly so filtering and bucket origins agree.
	rows, err := r.query(ctx, usageTrendsSQL, window.Start.UnixMilli(), bucketSeconds*1000, window.Start.UnixMilli(), window.End.UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("read usage trends: %w", err)
	}
	defer rows.Close()
	count := uint64((window.End.Sub(window.Start) + time.Duration(bucketSeconds)*time.Second - 1) / (time.Duration(bucketSeconds) * time.Second))
	result := make([]schema.UsageTrendsRow, 0)
	for rows.Next() {
		var row schema.UsageTrendsRow
		if err := rows.Scan(&row.Index, &row.Requests, &row.Success, &row.Input, &row.Output,
			&row.Cached, &row.CacheCreation, &row.LatencySumMs, &row.TTFTSumMs, &row.TTFTCount); err != nil {
			return nil, errors.New("invalid usage trends row")
		}
		if row.Index >= count || row.Success > row.Requests || row.TTFTCount > row.Requests || len(result) >= 120 {
			return nil, errors.New("invalid usage trends aggregate")
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage trends: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

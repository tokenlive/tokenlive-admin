package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	"golang.org/x/sync/singleflight"
)

type UsageTrendsReader interface {
	Enabled() bool
	ReadUsageTrends(context.Context, schema.Window, int64) ([]schema.UsageTrendsRow, error)
}

type usageTrendsCacheEntry struct {
	response *schema.UsageTrendsResponse
	expires  time.Time
}

type UsageTrendsService struct {
	reader  UsageTrendsReader
	now     func() time.Time
	mu      sync.Mutex
	cache   map[string]usageTrendsCacheEntry
	flights singleflight.Group
}

func NewUsageTrendsService(reader UsageTrendsReader, now func() time.Time) *UsageTrendsService {
	if now == nil {
		now = time.Now
	}
	return &UsageTrendsService{reader: reader, now: now, cache: make(map[string]usageTrendsCacheEntry)}
}

func (s *UsageTrendsService) cached(key string) *schema.UsageTrendsResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.cache[key]
	if !s.now().Before(entry.expires) {
		delete(s.cache, key)
		return nil
	}
	return entry.response
}

func (s *UsageTrendsService) Query(ctx context.Context, q schema.UsageTrendsQuery) (*schema.UsageTrendsResponse, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.reader == nil || !s.reader.Enabled() {
		return &schema.UsageTrendsResponse{State: "disabled", DataSource: "clickhouse", Buckets: []schema.UsageTrendsBucket{}}, nil
	}
	if response := s.cached(q.TimeRange); response != nil {
		return response, nil
	}
	result := s.flights.DoChan(q.TimeRange, func() (any, error) {
		if response := s.cached(q.TimeRange); response != nil {
			return response, nil
		}
		workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 9*time.Second)
		defer cancel()
		window := q.Window(s.now())
		seconds := q.BucketSeconds(window)
		rows, err := s.reader.ReadUsageTrends(workCtx, window, seconds)
		if err != nil {
			return nil, err
		}
		step := time.Duration(seconds) * time.Second
		buckets := make([]schema.UsageTrendsBucket, 0, 120)
		for start := window.Start; start.Before(window.End); start = start.Add(step) {
			end := start.Add(step)
			if end.After(window.End) {
				end = window.End
			}
			buckets = append(buckets, schema.UsageTrendsBucket{Start: start, End: end})
		}
		var totals schema.UsageTrendsTotals
		seen := make(map[uint64]bool, len(rows))
		for _, row := range rows {
			if row.Index >= uint64(len(buckets)) || seen[row.Index] || row.Success > row.Requests || row.TTFTCount > row.Requests {
				return nil, fmt.Errorf("invalid usage trends aggregate")
			}
			seen[row.Index] = true
			buckets[row.Index].UsageTrendsMetrics = row.UsageTrendsTotals.Metrics()
			totals.Add(row.UsageTrendsTotals)
		}
		if err := workCtx.Err(); err != nil {
			return nil, err
		}
		generated := s.now()
		summary := totals.Metrics()
		response := &schema.UsageTrendsResponse{State: "ready", DataSource: "clickhouse", Window: &window,
			GeneratedAt: &generated, BucketSeconds: seconds, Summary: &summary, Buckets: buckets}
		expires := generated.Add(30 * time.Second)
		if q.TimeRange == "today" && window.Start.AddDate(0, 0, 1).Before(expires) {
			expires = window.Start.AddDate(0, 0, 1)
		}
		s.mu.Lock()
		s.cache[q.TimeRange] = usageTrendsCacheEntry{response: response, expires: expires}
		s.mu.Unlock()
		return response, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case value := <-result:
		if value.Err != nil {
			return nil, value.Err
		}
		return value.Val.(*schema.UsageTrendsResponse), nil
	}
}

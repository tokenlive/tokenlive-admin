package biz

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	"golang.org/x/sync/singleflight"
)

type usageCacheEntry struct {
	Response *schema.Response
	Expires  time.Time
}

// Each service is permanently bound to one data source; caches never cross deployments.
type APIKeyUsageService struct {
	reader   Reader
	resolver Resolver
	now      func() time.Time
	mu       sync.Mutex
	cache    map[schema.Query]usageCacheEntry
	flights  singleflight.Group
}

func NewAPIKeyUsageService(reader Reader, resolver Resolver, now func() time.Time) *APIKeyUsageService {
	if now == nil {
		now = time.Now
	}
	if resolver == nil {
		resolver = NewIdentityResolver(nil, nil, "", now)
	}
	return &APIKeyUsageService{reader: reader, resolver: resolver, now: now, cache: make(map[schema.Query]usageCacheEntry)}
}

func (s *APIKeyUsageService) cached(query schema.Query) *schema.Response {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cache[query]
	if ok && s.now().Before(entry.Expires) {
		return entry.Response
	}
	delete(s.cache, query)
	return nil
}

func (s *APIKeyUsageService) Query(ctx context.Context, query schema.Query) (*schema.Response, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.reader == nil || !s.reader.Enabled() {
		return &schema.Response{State: "disabled", DataSource: "clickhouse", Items: []schema.Item{}, Warnings: []string{}}, nil
	}
	if cached := s.cached(query); cached != nil {
		return cloneUsageResponse(cached), nil
	}
	key := query.TimeRange + ":" + query.SortBy + ":" + strconv.Itoa(query.Limit)
	result := s.flights.DoChan(key, func() (any, error) {
		if cached := s.cached(query); cached != nil {
			return cached, nil
		}
		// One caller leaving must not cancel other authorized callers sharing this
		// bounded read. The owned task always ends within nine seconds.
		workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 9*time.Second)
		defer cancel()
		response, err := s.load(workCtx, query)
		if err != nil {
			return nil, err
		}
		expires := s.now().Add(30 * time.Second)
		if query.TimeRange == "today" {
			start := response.Window.Start
			midnight := start.AddDate(0, 0, 1)
			if midnight.Before(expires) {
				expires = midnight
			}
		}
		s.mu.Lock()
		s.cache[query] = usageCacheEntry{Response: response, Expires: expires}
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
		return cloneUsageResponse(value.Val.(*schema.Response)), nil
	}
}

func cloneUsageMetrics(metrics schema.Metrics) schema.Metrics {
	if metrics.TokenShare != nil {
		value := *metrics.TokenShare
		metrics.TokenShare = &value
	}
	return metrics
}

func cloneUsageResponse(source *schema.Response) *schema.Response {
	result := *source
	if source.Window != nil {
		window := *source.Window
		result.Window = &window
	}
	if source.GeneratedAt != nil {
		generated := *source.GeneratedAt
		result.GeneratedAt = &generated
	}
	if source.Summary != nil {
		summary := *source.Summary
		summary.Metrics = cloneUsageMetrics(summary.Metrics)
		result.Summary = &summary
	}
	if source.Unattributed != nil {
		unknown := cloneUsageMetrics(*source.Unattributed)
		result.Unattributed = &unknown
	}
	result.Items = make([]schema.Item, len(source.Items))
	for i, item := range source.Items {
		result.Items[i] = item
		result.Items[i].Metrics = cloneUsageMetrics(item.Metrics)
	}
	result.Warnings = append([]string{}, source.Warnings...)
	return &result
}

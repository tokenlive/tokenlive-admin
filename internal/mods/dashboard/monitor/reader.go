package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"gorm.io/gorm"
)

type Reader struct {
	store             *metrics.MemoryStore
	clock             func() time.Time
	db                *gorm.DB
	redis             *redis.Client
	prometheusAvail   bool
	prometheusMu      sync.RWMutex
	prometheusLastChk time.Time
	providerLabelOK   bool
	providerLabelMu   sync.RWMutex
	providerLabelChk  time.Time
	cacheMu           sync.RWMutex
	cacheMap          map[string]cacheVal
}

type Dependencies struct {
	DB     *gorm.DB
	Redis  *redis.Client
	Memory *metrics.MemoryStore
	Clock  func() time.Time
}

func New(deps Dependencies) *Reader {
	return &Reader{db: deps.DB, redis: deps.Redis, store: deps.Memory, clock: deps.Clock}
}

func (a *Reader) now() time.Time {
	if a.clock != nil {
		return a.clock()
	}
	return time.Now()
}

func (a *Reader) memory() *metrics.MemoryStore {
	if a.store != nil {
		return a.store
	}
	return metrics.GlobalStore
}

// Read is the external query seam for HTTP, WebSocket and interface tests.
// It captures the clock once, validates the whole request before I/O, and keeps
// failures local to the selected reading. Window plans never escape the module.
func (a *Reader) Read(ctx context.Context, q Query) (*Snapshot, error) {
	observed := a.now()
	loc, err := queryLocation(q.Timezone)
	if err != nil {
		return nil, err
	}
	cutoff := observed.In(loc)
	if q.At != nil {
		cutoff = q.At.In(loc)
		if cutoff.After(observed) {
			return nil, fmt.Errorf("monitoring cutoff is in the future")
		}
	}
	if !q.Overview && q.Traffic == nil && q.Performance == nil && q.Models == nil && q.Providers == nil {
		return nil, fmt.Errorf("no monitoring reading selected")
	}
	if q.Overview && q.At != nil && !cutoff.Truncate(time.Minute).Equal(observed.Truncate(time.Minute)) {
		return nil, fmt.Errorf("overview readings are live only")
	}
	plans := map[string]windowPlan{}
	add := func(name string, w WindowQuery) error {
		p, err := compileWindow(w, cutoff, observed, loc)
		if err == nil {
			plans[name] = p
		}
		return err
	}
	if q.Overview {
		err = add("overview", WindowQuery{Range: "today"})
	}
	if q.Traffic != nil && err == nil {
		if !oneOf(q.Traffic.GroupBy, "", "model", "provider", "tenant", "endpoint") {
			return nil, fmt.Errorf("invalid traffic grouping")
		}
		err = add("traffic", q.Traffic.WindowQuery)
	}
	if q.Performance != nil && err == nil {
		if strings.TrimSpace(q.Performance.Model) == "" {
			return nil, fmt.Errorf("model is required")
		}
		err = add("performance", q.Performance.WindowQuery)
	}
	if q.Models != nil && err == nil {
		err = validateRanking(q.Models.Sort, q.Models.Limit)
		if err == nil {
			err = add("models", q.Models.WindowQuery)
		}
	}
	if q.Providers != nil && err == nil {
		err = validateRanking(q.Providers.Sort, q.Providers.Limit)
		if err == nil {
			w := q.Providers.WindowQuery
			if d, ok := parseRangeDuration(w.Range); ok && d > providerMetricsMaxRange {
				w.Range = "7d"
			}
			if w.Start != nil && w.Start.Before(cutoff.Truncate(time.Minute).Add(-providerMetricsMaxRange)) {
				start := cutoff.Truncate(time.Minute).Add(-providerMetricsMaxRange)
				w.Start = &start
			}
			err = add("providers", w)
		}
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res := &Snapshot{Cutoff: cutoff}
	if q.Overview {
		c, s := a.readContext(ctx, plans["overview"])
		data, e := a.getOverview(c)
		res.Overview = &Reading[Overview]{State: "ok", Source: "mixed", Window: plans["overview"].Window, Data: data}
		r := res.Overview
		if e != nil || data == nil {
			r.State, r.Error = "error", "source_unavailable"
		} else {
			if data.QPS == nil || data.DailyRequests == nil || data.AvgLatencyMs == nil || data.AvgTTFTMs == nil {
				r.State, r.Coverage.Reason = "partial", "missing_measurements"
			}
			if a.redis == nil && a.memory().StartedAt().After(plans["overview"].Start) {
				r.State, r.Coverage.Reason = "partial", "outside_retention"
			}
			start, end := plans["overview"].Start, cutoff
			if a.redis == nil && a.memory().StartedAt().After(start) {
				start = a.memory().StartedAt()
			}
			if data.DailyRequests != nil {
				r.Coverage.Start, r.Coverage.End = &start, &end
			}
		}
		res.Overview.Windows = map[string]Window{
			"daily":   {Start: plans["overview"].Start, End: cutoff, Timezone: loc.String()},
			"qps":     {Start: cutoff.Add(-5 * time.Minute), End: cutoff, Timezone: loc.String()},
			"latency": {Start: cutoff.Add(-5 * time.Minute), End: cutoff, Timezone: loc.String()},
		}
		if s.qpsFallback {
			res.Overview.Windows["qps"] = Window{Start: cutoff.Truncate(time.Minute).Add(-time.Minute), End: cutoff.Truncate(time.Minute), Timezone: loc.String()}
		}
	}
	if q.Traffic != nil {
		p := plans["traffic"]
		c, s := a.readContext(ctx, p)
		raw, e := a.getTrendsAt(c, q.Traffic.GroupBy, q.Traffic.Range, q.Traffic.Model, p.End)
		data := &Traffic{Series: []TrafficSeries{}}
		if raw != nil {
			for _, row := range raw.Series {
				data.Series = append(data.Series, TrafficSeries{Label: row.Label, Success: nullableCounts(row.Success), Failure: nullableCounts(row.Failure), Total: nullableCounts(row.Total)})
			}
		}
		a.checkPrometheusCoverage(c, s)
		res.Traffic = result(s, data, e)
		maskTraffic(res.Traffic, s)
	}
	if q.Performance != nil {
		p := plans["performance"]
		c, s := a.readContext(ctx, p)
		raw, e := a.getModelPerformanceTrendsAt(c, q.Performance.Model, q.Performance.Range, p.End)
		var data *Performance
		if raw != nil {
			data = &Performance{AvgTTFTMs: cloneFloats(raw.AvgTTFTMs), Otps: cloneFloats(raw.Otps)}
		}
		a.checkPrometheusCoverage(c, s)
		res.Performance = result(s, data, e)
		if r := res.Performance; r.Data != nil {
			for i, b := range r.Buckets {
				if !bucketCovered(b, r.Coverage) || !sourceBucketCovered(b, s) {
					r.Data.AvgTTFTMs[i], r.Data.Otps[i] = nil, nil
				}
			}
		}
	}
	if q.Models != nil {
		p := plans["models"]
		c, s := a.readContext(ctx, p)
		data, e := a.getModelRankingAt(c, defaultSort(q.Models.Sort), q.Models.Range, defaultLimit(q.Models.Limit), q.Models.Model, p.End)
		data = append([]ModelRankingItem{}, data...)
		a.checkPrometheusCoverage(c, s)
		res.Models = result(s, &data, e)
	}
	if q.Providers != nil {
		p := plans["providers"]
		c, s := a.readContext(ctx, p)
		data, e := a.getProviderRankingAt(c, defaultSort(q.Providers.Sort), q.Providers.Range, defaultLimit(q.Providers.Limit), q.Providers.Provider, p.End)
		data = append([]ProviderRankingItem{}, data...)
		a.checkPrometheusCoverage(c, s)
		res.Providers = result(s, &data, e)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
func validateRanking(sort string, limit int) error {
	if !oneOf(sort, "", "request_count", "avg_latency", "avg_ttft", "tokens", "cost", "success_rate", "otps") || limit < 0 || limit > 100 {
		return fmt.Errorf("invalid monitoring ranking options")
	}
	return nil
}
func defaultSort(sort string) string {
	if sort == "" {
		return "request_count"
	}
	return sort
}
func defaultLimit(limit int) int {
	if limit == 0 {
		return 10
	}
	return limit
}

package monitor

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type scopeKey struct{}
type readScope struct {
	plan        windowPlan
	audit       *readAudit
	memoryStart time.Time
	qpsFallback bool
}
type readAudit struct {
	mu         sync.Mutex
	source     string
	failed     map[string]bool
	seen       map[int64]bool
	coverage   *Coverage
	incomplete bool
	minutes    map[int64]bool
}

func scope(ctx context.Context) *readScope {
	v, _ := ctx.Value(scopeKey{}).(*readScope)
	return v
}

func withPlan(ctx context.Context, p windowPlan) (context.Context, *readScope) {
	s := &readScope{plan: p, audit: &readAudit{failed: map[string]bool{}, seen: map[int64]bool{}}}
	return context.WithValue(ctx, scopeKey{}, s), s
}

func source(ctx context.Context, name string) {
	if s := scope(ctx); s != nil {
		s.audit.mu.Lock()
		s.audit.source = name
		s.audit.mu.Unlock()
	}
}

func sourceError(ctx context.Context, name string, err error) {
	if err != nil {
		if s := scope(ctx); s != nil {
			s.audit.mu.Lock()
			s.audit.failed[name] = true
			s.audit.mu.Unlock()
		}
	}
}

func seen(ctx context.Context, at int64) {
	if s := scope(ctx); s != nil {
		s.audit.mu.Lock()
		s.audit.seen[at] = true
		s.audit.mu.Unlock()
	}
}

func (a *Reader) redisValues(ctx context.Context, keys ...string) ([]interface{}, error) {
	values, err := a.redis.MGet(ctx, keys...).Result()
	sourceError(ctx, "redis", err)
	return values, err
}

func result[T any](s *readScope, data *T, err error) *Reading[T] {
	p := s.plan
	r := &Reading[T]{State: "ok", Source: s.audit.source, Window: p.Window, Buckets: append([]Bucket(nil), p.Buckets...), Data: data}
	if err != nil || s.audit.failed[r.Source] {
		r.State, r.Error, r.Data = "error", "source_unavailable", nil
		return r
	}
	start, end := p.Start, p.End
	if !start.Before(end) {
		r.State, r.Coverage.Reason, r.Data = "unavailable", "no_completed_minutes", nil
		return r
	}
	switch r.Source {
	case "redis":
		floor := p.Observed.Add(-2 * time.Hour).Truncate(time.Minute).Add(time.Minute)
		if start.Before(floor) {
			start = floor
		}
	case "memory":
		floor := p.Observed.Add(-7 * 24 * time.Hour).Truncate(time.Minute).Add(time.Minute)
		if !s.memoryStart.IsZero() && s.memoryStart.After(floor) {
			floor = s.memoryStart.Truncate(time.Minute).Add(time.Minute)
		}
		if start.Before(floor) {
			start = floor
		}
	case "catalog":
	case "prometheus":
		if len(s.audit.seen) == 0 {
			r.State, r.Coverage.Reason, r.Data = "unavailable", "no_source_samples", nil
			return r
		}
	default:
		r.State, r.Coverage.Reason, r.Data = "unavailable", "no_source", nil
		return r
	}
	if !start.Before(end) && p.Start.Before(p.End) {
		r.State, r.Coverage.Reason, r.Data = "unavailable", "outside_retention", nil
		return r
	}
	if r.Source == "prometheus" && s.audit.coverage != nil {
		r.Coverage = *s.audit.coverage
		if r.Coverage.Start == nil {
			r.State, r.Data = "unavailable", nil
			return r
		}
		if s.audit.incomplete {
			r.State = "partial"
		}
		return r
	}
	r.Coverage.Start, r.Coverage.End = &start, &end
	if start.After(p.Start) {
		r.State, r.Coverage.Reason = "partial", "outside_retention"
	}
	return r
}

func nullableCounts(values []int64) []*int64 {
	res := make([]*int64, len(values))
	for i, v := range values {
		n := v
		res[i] = &n
	}
	return res
}
func maskTraffic(r *Reading[Traffic], s *readScope) {
	if r.Data == nil {
		return
	}
	for i, b := range r.Buckets {
		covered := bucketCovered(b, r.Coverage) && sourceBucketCovered(b, s)
		if r.Source == "prometheus" && !s.audit.seen[b.End.Unix()] {
			covered = false
		}
		if !covered {
			if r.State == "ok" {
				r.State, r.Coverage.Reason = "partial", "missing_buckets"
			}
			for j := range r.Data.Series {
				row := &r.Data.Series[j]
				if i < len(row.Total) {
					row.Total[i], row.Success[i], row.Failure[i] = nil, nil, nil
				}
			}
		}
	}
}

func (a *Reader) readContext(ctx context.Context, p windowPlan) (context.Context, *readScope) {
	ctx, s := withPlan(ctx, p)
	s.memoryStart = a.memory().StartedAt()
	return ctx, s
}

func bucketCovered(b Bucket, c Coverage) bool {
	return c.Start != nil && c.End != nil && !b.Start.Before(*c.Start) && !b.End.After(*c.End)
}

// A PromQL increase alone cannot certify retention: the engine can extrapolate
// from a small tail of samples. Inspect source presence separately at minute
// resolution before claiming a complete historical window.
func (a *Reader) checkPrometheusCoverage(ctx context.Context, s *readScope) {
	if s.audit.source != "prometheus" || s.audit.failed["prometheus"] || len(s.plan.Buckets) == 0 {
		return
	}
	p := s.plan
	probeCtx, _ := withPlan(ctx, p)
	const cacheKey = "prometheus-coverage"
	type probe struct {
		Coverage   Coverage
		Incomplete bool
		Minutes    map[int64]bool
	}
	if v, ok := a.getCache(probeCtx, cacheKey); ok {
		v := v.(probe)
		s.audit.coverage = &v.Coverage
		s.audit.incomplete = v.Incomplete
		s.audit.minutes = v.Minutes
		return
	}
	values, err := a.queryPrometheusRangeRaw(probeCtx, fmt.Sprintf("count(count_over_time(%s[60s]))", mRequestTotal), p.Start.Add(time.Minute).Unix(), p.End.Unix(), 60)
	c := Coverage{Reason: "no_source_samples"}
	missing := true
	if err != nil {
		c.Reason = "coverage_unverified"
	} else {
		first, last := int64(0), int64(0)
		n := 0
		s.audit.minutes = map[int64]bool{}
		for _, v := range values {
			if len(v) < 2 {
				continue
			}
			stamp, ok := v[0].(float64)
			if !ok {
				continue
			}
			count, _ := strconv.ParseFloat(fmt.Sprint(v[1]), 64)
			if count <= 0 {
				continue
			}
			at := int64(stamp)
			s.audit.minutes[at-60] = true
			if first == 0 || at < first {
				first = at
			}
			if at > last {
				last = at
			}
			n++
		}
		if first != 0 {
			start, end := time.Unix(first-60, 0).In(p.Start.Location()), time.Unix(last, 0).In(p.End.Location())
			c.Start, c.End = &start, &end
			missing = n < p.minutes()
			if missing {
				c.Reason = "missing_source_minutes"
			} else {
				c.Reason = ""
			}
		}
	}
	s.audit.coverage, s.audit.incomplete = &c, missing
	if err == nil {
		a.setCache(probeCtx, cacheKey, probe{c, missing, s.audit.minutes}, 5*time.Second)
	}
}

func sourceBucketCovered(b Bucket, s *readScope) bool {
	if s.audit.source != "prometheus" || s.audit.minutes == nil {
		return true
	}
	for cursor := b.Start; cursor.Before(b.End); cursor = cursor.Add(time.Minute) {
		if !s.audit.minutes[cursor.Unix()] {
			return false
		}
	}
	return true
}
func cloneFloats(values []*float64) []*float64 {
	out := make([]*float64, len(values))
	for i, v := range values {
		if v != nil {
			n := *v
			out[i] = &n
		}
	}
	return out
}

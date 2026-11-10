package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// windowPlan is immutable after compilation and is never exposed to adapters'
// callers. All source lookbacks and bucket indexes derive from this geometry.
type windowPlan struct {
	Window
	Observed time.Time
	Cutoff   time.Time
	Buckets  []Bucket
	Step     time.Duration
}

func queryLocation(name string) (*time.Location, error) {
	if name == "" {
		return time.Local, nil
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc, nil
	}
	// RFC3339 offsets are also useful when the caller has no IANA zone name.
	if len(name) == 6 && (name[0] == '+' || name[0] == '-') && name[3] == ':' {
		h, e1 := strconv.Atoi(name[1:3])
		m, e2 := strconv.Atoi(name[4:6])
		if e1 == nil && e2 == nil && name[1] >= '0' && name[1] <= '9' && name[4] >= '0' && name[4] <= '9' && h >= 0 && m >= 0 && h <= 14 && m < 60 && (h < 14 || m == 0) {
			offset := (h*60 + m) * 60
			if name[0] == '-' {
				offset = -offset
			}
			return time.FixedZone(name, offset), nil
		}
	}
	return nil, fmt.Errorf("invalid monitoring timezone")
}

func parseDuration(value string) (time.Duration, error) {
	d, ok := parseRangeDuration(value)
	if !ok || d < time.Minute || d > 7*24*time.Hour {
		return 0, fmt.Errorf("invalid monitoring range")
	}
	return d, nil
}

// parseRangeDuration 解析前端传入的时间范围。
//
// time.ParseDuration 不支持 "d" 单位，而前端的快捷范围里就有 "7d"——
// 直接交给它解析会让 "7d" 静默退化成默认的 1 小时。
func parseRangeDuration(timeRange string) (time.Duration, bool) {
	if days, found := strings.CutSuffix(timeRange, "d"); found {
		n, err := strconv.Atoi(days)
		if err != nil || n <= 0 || int64(n) > int64((1<<63-1)/(24*time.Hour)) {
			return 0, false
		}
		return time.Duration(n) * 24 * time.Hour, true
	}
	duration, err := time.ParseDuration(timeRange)
	if err != nil {
		return 0, false
	}
	return duration, true
}

func compileWindow(q WindowQuery, cutoff, observed time.Time, loc *time.Location) (windowPlan, error) {
	end := cutoff.In(loc).Truncate(time.Minute)
	start := end.Add(-time.Hour)
	if q.Range == "today" {
		start = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
	} else if q.Range != "" {
		duration, err := parseDuration(q.Range)
		if err != nil {
			return windowPlan{}, err
		}
		start = end.Add(-duration)
	}
	if q.Start != nil {
		start = q.Start.In(loc)
	}
	if !start.Equal(start.Truncate(time.Minute)) {
		start = start.Truncate(time.Minute).Add(time.Minute)
	}
	if start.After(end) || end.Sub(start) > 7*24*time.Hour {
		return windowPlan{}, fmt.Errorf("invalid monitoring window")
	}
	span := end.Sub(start)
	step := time.Minute
	switch q.Range {
	case "6h":
		step = 5 * time.Minute
	case "24h":
		step = 15 * time.Minute
	case "7d":
		step = 2 * time.Hour
	default:
		for _, candidate := range []time.Duration{time.Minute, 5 * time.Minute, 10 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour} {
			step = candidate
			if (span+step-1)/step <= 120 {
				break
			}
		}
	}
	if (span+step-1)/step > 120 {
		step = ((span/time.Minute + 119) / 120) * time.Minute
	}
	p := windowPlan{Window: Window{Start: start, End: end, Timezone: loc.String()}, Cutoff: cutoff, Observed: observed, Step: step, Buckets: []Bucket{}}
	for cursor := start; cursor.Before(end); cursor = cursor.Add(step) {
		bucketEnd := cursor.Add(step)
		if bucketEnd.After(end) {
			bucketEnd = end
		}
		p.Buckets = append(p.Buckets, Bucket{Start: cursor, End: bucketEnd})
	}
	return p, nil
}

func (p windowPlan) identity() string {
	return fmt.Sprintf("window-v1:%d:%d:%s:%d", p.Start.Unix(), p.End.Unix(), p.Timezone, p.Step/time.Second)
}

func (p windowPlan) minutes() int { return int(p.End.Sub(p.Start) / time.Minute) }

func (a *Reader) queryPlan(ctx context.Context, span string, end time.Time) (context.Context, windowPlan, error) {
	if s := scope(ctx); s != nil {
		return ctx, s.plan, nil
	}
	p, err := compileWindow(WindowQuery{Range: span}, end, end, end.Location())
	if err != nil {
		return ctx, p, err
	}
	ctx, _ = withPlan(ctx, p)
	return ctx, p, nil
}

func (p windowPlan) trendConfig() trendRangeConfig {
	copy := p
	return trendRangeConfig{plan: &copy, minutes: p.minutes(), numPoints: len(p.Buckets), stepSeconds: int64(p.Step / time.Second), redisMinutes: min(p.minutes(), 120), end: p.End}
}

func (p windowPlan) index(minute int64) int {
	t := time.Unix(minute*60, 0)
	if t.Before(p.Start) || !t.Before(p.End) {
		return -1
	}
	return int(t.Sub(p.Start) / p.Step)
}

package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type gridQuery struct {
	query            string
	start, end, step int64
}

// gridQueries keeps the final partial bucket's lookback equal to its actual
// width. Prometheus's inclusive query_range endpoints must not create an extra
// point at the requested interval start.
func gridQueries(ctx context.Context, query string, start, end, step int64) []gridQuery {
	s := scope(ctx)
	if s == nil || len(s.plan.Buckets) == 0 {
		return []gridQuery{{query, start, end, step}}
	}
	p := s.plan
	last := p.Buckets[len(p.Buckets)-1]
	width := int64(last.End.Sub(last.Start) / time.Second)
	if width == step {
		return []gridQuery{{query, p.Buckets[0].End.Unix(), p.End.Unix(), step}}
	}
	var parts []gridQuery
	if len(p.Buckets) > 1 {
		parts = append(parts, gridQuery{query, p.Buckets[0].End.Unix(), last.Start.Unix(), step})
	}
	partial := strings.ReplaceAll(query, fmt.Sprintf("[%ds]", step), fmt.Sprintf("[%ds]", width))
	return append(parts, gridQuery{partial, last.End.Unix(), last.End.Unix(), max(width, 1)})
}

func gridValues(ctx context.Context, values [][]interface{}, step int64) [][]interface{} {
	s := scope(ctx)
	if s == nil || len(s.plan.Buckets) == 0 {
		return values
	}
	p := s.plan
	last := p.Buckets[len(p.Buckets)-1]
	for _, point := range values {
		if len(point) < 2 {
			continue
		}
		stamp, ok := point[0].(float64)
		if !ok {
			continue
		}
		// Internal alignment uses a regular index grid; actual bucket bounds
		// remain in the immutable plan and public result.
		if int64(stamp) == last.End.Unix() && last.End.Sub(last.Start) < p.Step {
			point[0] = float64(p.Buckets[0].End.Unix() + int64(len(p.Buckets)-1)*step)
		}
	}
	return values
}

func (a *Reader) queryPrometheusRange(ctx context.Context, query string, start, end, step int64) ([][]interface{}, error) {
	var values [][]interface{}
	for _, part := range gridQueries(ctx, query, start, end, step) {
		points, err := a.queryPrometheusRangeRaw(ctx, part.query, part.start, part.end, part.step)
		if err != nil {
			sourceError(ctx, "prometheus", err)
			return nil, err
		}
		values = append(values, points...)
	}
	return gridValues(ctx, values, step), nil
}

func (a *Reader) queryPrometheusRangeMulti(ctx context.Context, query string, start, end, step int64, label string) (map[string][][]interface{}, error) {
	values := map[string][][]interface{}{}
	for _, part := range gridQueries(ctx, query, start, end, step) {
		rows, err := a.queryPrometheusRangeMultiRaw(ctx, part.query, part.start, part.end, part.step, label)
		if err != nil {
			sourceError(ctx, "prometheus", err)
			return nil, err
		}
		for key, points := range rows {
			values[key] = append(values[key], points...)
		}
	}
	for key, points := range values {
		values[key] = gridValues(ctx, points, step)
	}
	return values, nil
}

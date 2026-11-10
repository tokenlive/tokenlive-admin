package monitor

import (
	"math"
	"time"
)

func observation[T any](p windowPlan, source, population string, c Coverage, identity string) MetricObservation[T] {
	m := MetricObservation[T]{State: "ok", Source: source, Scope: population, Window: p.Window, Coverage: c, Identity: identity}
	if c.Start == nil || c.End == nil {
		m.State = "unavailable"
		m.Reason = c.Reason
	} else if c.Start.After(p.Start) || c.End.Before(p.End) || c.Reason != "" {
		m.State = "partial"
		m.Reason = c.Reason
	}
	return m
}

func missingObservation[T any](m MetricObservation[T], state, reason string) MetricObservation[T] {
	m.Value = nil
	m.State = state
	m.Reason = reason
	return m
}

func sourceCoverage(p windowPlan, floor time.Time) Coverage {
	start, end := p.Start, p.End
	if start.Before(floor) {
		start = floor
	}
	if !start.Before(end) {
		return Coverage{Reason: "outside_retention"}
	}
	c := Coverage{Start: &start, End: &end}
	if start.After(p.Start) {
		c.Reason = "outside_retention"
	}
	return c
}

func (a *Reader) memoryCoverage(p windowPlan) Coverage {
	floor := p.Observed.Add(-7 * 24 * time.Hour).Truncate(time.Minute).Add(time.Minute)
	started := a.memory().StartedAt()
	if started.After(floor) {
		floor = started.Truncate(time.Minute)
		if !started.Equal(floor) {
			floor = floor.Add(time.Minute)
		}
	}
	return sourceCoverage(p, floor)
}

func cloneObservation[T any](m MetricObservation[T]) MetricObservation[T] {
	if m.Value != nil {
		value := *m.Value
		m.Value = &value
	}
	if m.Coverage.Start != nil {
		at := *m.Coverage.Start
		m.Coverage.Start = &at
	}
	if m.Coverage.End != nil {
		at := *m.Coverage.End
		m.Coverage.End = &at
	}
	return m
}

func integerObservation(m MetricObservation[float64]) MetricObservation[int64] {
	n := MetricObservation[int64]{State: m.State, Source: m.Source, Scope: m.Scope, Window: m.Window, Coverage: m.Coverage, Reason: m.Reason, Identity: m.Identity}
	if m.Value == nil {
		return n
	}
	value := *m.Value
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= float64(math.MaxInt64) {
		return missingObservation(n, "error", "invalid_source_value")
	}
	v := int64(value)
	n.Value = &v
	return n
}

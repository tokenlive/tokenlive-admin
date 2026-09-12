package updatecheck

import (
	"context"
	"errors"
	"time"
)

// SourceState describes the latest attempt and, when present, its candidate.
// Candidates in stale or non-ready states are history, not actionable targets.
type SourceState struct {
	Status      string     `json:"status"`
	Candidate   *Candidate `json:"candidate,omitempty"`
	LastAttempt time.Time  `json:"last_attempt"`
	LastSuccess time.Time  `json:"last_success"`
	Stale       bool       `json:"stale"`
	ErrorCode   string     `json:"error_code,omitempty"`
}

// CheckResult is a snapshot of the shared update checks.
type CheckResult struct {
	Enabled           bool                   `json:"enabled"`
	Sources           map[string]SourceState `json:"sources"`
	RetryAfterSeconds int                    `json:"retry_after_seconds"`
}

func (c *Checker) snapshotLocked(now time.Time) CheckResult {
	return cloneResult(CheckResult{
		Enabled:           c.opts.Enabled,
		Sources:           c.states,
		RetryAfterSeconds: c.retryAfterLocked(now),
	}, now, c.opts.Interval)
}

func (c *Checker) retryAfterLocked(now time.Time) int {
	if !c.hasStart {
		return 0
	}
	remaining := c.opts.Cooldown - now.Sub(c.lastStart)
	if remaining <= 0 {
		return 0
	}
	seconds := remaining / time.Second
	if remaining%time.Second != 0 {
		seconds++
	}
	return int(seconds)
}

func cloneResult(result CheckResult, now time.Time, interval time.Duration) CheckResult {
	states := make(map[string]SourceState, len(result.Sources))
	for name, state := range result.Sources {
		if state.Candidate != nil {
			candidate := *state.Candidate
			state.Candidate = &candidate
			if now.Sub(state.LastSuccess) >= interval {
				state.Stale = true
			}
		}
		states[name] = state
	}
	result.Sources = states
	return result
}

// Keep error details local: upstream messages may contain sensitive URLs.
func sourceErrorCode(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "check_failed"
	}
}

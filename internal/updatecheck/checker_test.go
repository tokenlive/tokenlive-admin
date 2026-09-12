package updatecheck

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type sourceFunc func(context.Context) (Candidate, error)

func (f sourceFunc) Latest(ctx context.Context) (Candidate, error) { return f(ctx) }

func TestDisabledNeverCallsSource(t *testing.T) {
	var calls atomic.Int32
	c, err := NewChecker(context.Background(), Options{Enabled: false}, map[string]Source{
		"admin": sourceFunc(func(context.Context) (Candidate, error) {
			calls.Add(1)
			return Candidate{Version: "v1.2.4"}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Start()
	result, err := c.Check(context.Background(), true)
	if !errors.Is(err, ErrDisabled) || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
	if result.Enabled || result.Sources["admin"].Status != "disabled" {
		t.Fatalf("disabled result = %+v", result)
	}
}

// These doubles replace external requests, not the checker's scheduling/state.
type checkerClock struct {
	mu  sync.Mutex
	now time.Time
}

func newCheckerClock() *checkerClock {
	return &checkerClock{now: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)}
}

func (c *checkerClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *checkerClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestChecker(t *testing.T, opts Options, sources map[string]Source) *Checker {
	t.Helper()
	c, err := NewChecker(context.Background(), opts, sources)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestCheckerRejectsNegativeDurations(t *testing.T) {
	for name, opts := range map[string]Options{
		"interval": {Interval: -time.Nanosecond},
		"timeout":  {Timeout: -time.Nanosecond},
		"cooldown": {Cooldown: -time.Nanosecond},
	} {
		t.Run(name, func(t *testing.T) {
			c, err := NewChecker(context.Background(), opts, nil)
			if c != nil {
				c.Close()
			}
			if err == nil {
				t.Fatal("negative duration accepted")
			}
		})
	}
}

func TestCheckerSuccessAndSnapshotDoNotFetch(t *testing.T) {
	clock := newCheckerClock()
	var calls atomic.Int32
	sources := map[string]Source{
		"admin": sourceFunc(func(context.Context) (Candidate, error) {
			calls.Add(1)
			return Candidate{Version: "v1.2.4", ReleaseURL: "https://example.invalid/v1.2.4"}, nil
		}),
	}
	c := newTestChecker(t, Options{Enabled: true, Now: clock.Now}, sources)
	delete(sources, "admin")
	initial := c.Snapshot()
	if initial.Sources["admin"].Status != "unchecked" || calls.Load() != 0 {
		t.Fatalf("initial snapshot = %+v, calls = %d", initial, calls.Load())
	}
	result, err := c.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	state := result.Sources["admin"]
	if !result.Enabled || state.Status != "ready" || state.Stale || state.ErrorCode != "" {
		t.Fatalf("successful result = %+v", result)
	}
	if state.Candidate == nil || state.Candidate.Version != "v1.2.4" || state.Candidate.ReleaseURL != "https://example.invalid/v1.2.4" {
		t.Fatalf("candidate = %+v", state.Candidate)
	}
	if !state.LastAttempt.Equal(clock.Now()) || !state.LastSuccess.Equal(clock.Now()) {
		t.Fatalf("timestamps = %+v", state)
	}
	c.Snapshot()
	c.Snapshot()
	if calls.Load() != 1 {
		t.Fatalf("snapshots initiated requests: %d", calls.Load())
	}
}

func TestManualCooldownBoundaryAndAutomaticBypass(t *testing.T) {
	clock := newCheckerClock()
	var calls atomic.Int32
	c := newTestChecker(t, Options{Enabled: true, Now: clock.Now}, map[string]Source{
		"admin": sourceFunc(func(context.Context) (Candidate, error) {
			calls.Add(1)
			return Candidate{Version: "v1.2.4"}, nil
		}),
	})
	if _, err := c.Check(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	clock.Advance(59 * time.Second)
	result, err := c.Check(context.Background(), true)
	if !errors.Is(err, ErrCooldown) || result.RetryAfterSeconds != 1 || calls.Load() != 1 {
		t.Fatalf("59s result = %+v, err = %v, calls = %d", result, err, calls.Load())
	}
	if result.Sources["admin"].Status != "ready" {
		t.Fatalf("cooldown lost cached state: %+v", result)
	}
	clock.Advance(time.Second)
	if _, err := c.Check(context.Background(), true); err != nil || calls.Load() != 2 {
		t.Fatalf("60s err = %v, calls = %d", err, calls.Load())
	}
	if _, err := c.Check(context.Background(), false); err != nil || calls.Load() != 3 {
		t.Fatalf("automatic check err = %v, calls = %d", err, calls.Load())
	}
	clock.Advance(500 * time.Millisecond)
	result, err = c.Check(context.Background(), true)
	if !errors.Is(err, ErrCooldown) || result.RetryAfterSeconds != 60 {
		t.Fatalf("retry duration was not rounded up: %+v, %v", result, err)
	}
}

func TestCustomCooldownAndZeroClock(t *testing.T) {
	clock := &checkerClock{}
	c := newTestChecker(t, Options{
		Enabled: true, Now: clock.Now, Cooldown: 10 * time.Second,
	}, nil)
	if _, err := c.Check(context.Background(), true); err != nil {
		t.Fatalf("initial zero clock check: %v", err)
	}
	clock.Advance(9 * time.Second)
	result, err := c.Check(context.Background(), true)
	if !errors.Is(err, ErrCooldown) || result.RetryAfterSeconds != 1 {
		t.Fatalf("zero clock bypassed cooldown: %+v, %v", result, err)
	}
	clock.Advance(time.Second)
	if _, err := c.Check(context.Background(), true); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateHistoryIsStaleAfterFailureOrWithdrawal(t *testing.T) {
	for name, sourceErr := range map[string]error{
		"failure":   errors.New("private upstream credentials must not be exposed"),
		"withdrawn": fmt.Errorf("release withdrawn: %w", ErrNoCandidate),
		"timed_out": context.DeadlineExceeded,
	} {
		t.Run(name, func(t *testing.T) {
			clock := newCheckerClock()
			var calls atomic.Int32
			c := newTestChecker(t, Options{Enabled: true, Now: clock.Now}, map[string]Source{
				"admin": sourceFunc(func(context.Context) (Candidate, error) {
					if calls.Add(1) == 2 {
						return Candidate{}, sourceErr
					}
					return Candidate{Version: "v1.2.4"}, nil
				}),
			})
			if _, err := c.Check(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			successTime := clock.Now()
			clock.Advance(time.Minute)
			result, err := c.Check(context.Background(), false)
			if err != nil {
				t.Fatal(err)
			}
			state := result.Sources["admin"]
			wantStatus := "unavailable"
			if errors.Is(sourceErr, ErrNoCandidate) {
				wantStatus = "no_candidate"
			}
			if state.Status != wantStatus || !state.Stale || state.Candidate == nil || state.Candidate.Version != "v1.2.4" {
				t.Fatalf("historical candidate became actionable/lost: %+v", state)
			}
			if !state.LastSuccess.Equal(successTime) || !state.LastAttempt.Equal(clock.Now()) {
				t.Fatalf("failure changed success timestamp: %+v", state)
			}
			if state.ErrorCode == sourceErr.Error() {
				t.Fatalf("raw upstream error exposed: %q", state.ErrorCode)
			}
			if wantStatus == "unavailable" && state.ErrorCode == "" {
				t.Fatal("unavailable result lacks a safe error code")
			}
			clock.Advance(time.Minute)
			result, err = c.Check(context.Background(), false)
			if err != nil || result.Sources["admin"].Stale || result.Sources["admin"].Status != "ready" || result.Sources["admin"].ErrorCode != "" {
				t.Fatalf("recovery result = %+v, err = %v", result, err)
			}
		})
	}
}

func TestMissingCandidateWithoutHistory(t *testing.T) {
	c := newTestChecker(t, Options{Enabled: true}, map[string]Source{
		"admin": sourceFunc(func(context.Context) (Candidate, error) {
			return Candidate{}, ErrNoCandidate
		}),
		"gateway": sourceFunc(func(context.Context) (Candidate, error) {
			return Candidate{}, errors.New("offline")
		}),
	})
	result, err := c.Check(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	for name, status := range map[string]string{"admin": "no_candidate", "gateway": "unavailable"} {
		state := result.Sources[name]
		if state.Status != status || state.Candidate != nil || !state.LastSuccess.IsZero() {
			t.Fatalf("%s result = %+v", name, state)
		}
	}
}

func TestSnapshotExpiresCandidateAtInterval(t *testing.T) {
	for name, interval := range map[string]time.Duration{"default": 0, "custom": 2 * time.Hour} {
		t.Run(name, func(t *testing.T) {
			clock := newCheckerClock()
			c := newTestChecker(t, Options{Enabled: true, Now: clock.Now, Interval: interval}, map[string]Source{
				"admin": sourceFunc(func(context.Context) (Candidate, error) {
					return Candidate{Version: "v1.2.4"}, nil
				}),
			})
			if _, err := c.Check(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			if interval == 0 {
				interval = 6 * time.Hour
			}
			clock.Advance(interval - time.Nanosecond)
			if c.Snapshot().Sources["admin"].Stale {
				t.Fatal("candidate expired early")
			}
			clock.Advance(time.Nanosecond)
			state := c.Snapshot().Sources["admin"]
			if !state.Stale || state.Candidate == nil || state.Candidate.Version != "v1.2.4" {
				t.Fatalf("expired candidate = %+v", state)
			}
		})
	}
}

func TestAllReturnedSnapshotsAreDeepCopies(t *testing.T) {
	clock := newCheckerClock()
	c := newTestChecker(t, Options{Enabled: true, Now: clock.Now}, map[string]Source{
		"admin": sourceFunc(func(context.Context) (Candidate, error) {
			return Candidate{Version: "v1.2.4", ReleaseURL: "https://example.invalid/release"}, nil
		}),
	})
	check, err := c.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := c.Snapshot()
	cooldown, err := c.Check(context.Background(), true)
	if !errors.Is(err, ErrCooldown) {
		t.Fatal(err)
	}
	for _, result := range []CheckResult{check, snapshot, cooldown} {
		result.Sources["admin"].Candidate.Version = "corrupted"
		result.Sources["admin"].Candidate.ReleaseURL = "corrupted"
		delete(result.Sources, "admin")
		result.Sources["unexpected"] = SourceState{Status: "ready"}
	}
	state := c.Snapshot()
	if len(state.Sources) != 1 || state.Sources["admin"].Candidate.Version != "v1.2.4" || state.Sources["admin"].Candidate.ReleaseURL != "https://example.invalid/release" {
		t.Fatalf("caller changed checker state: %+v", state)
	}
}

type checkOutcome struct {
	result CheckResult
	err    error
}

func checkAsync(c *Checker, ctx context.Context, manual bool) <-chan checkOutcome {
	result := make(chan checkOutcome, 1)
	go func() {
		value, err := c.Check(ctx, manual)
		result <- checkOutcome{value, err}
	}()
	return result
}

func receiveCheckValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for checker")
		var zero T
		return zero
	}
}

type waitingContext struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func observeWait(ctx context.Context) *waitingContext {
	return &waitingContext{Context: ctx, waiting: make(chan struct{})}
}

func (c *waitingContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestConcurrentChecksJoinAndCallerCancellationIsIsolated(t *testing.T) {
	var calls atomic.Int32
	started := make(chan context.Context, 8)
	release := make(chan struct{})
	c := newTestChecker(t, Options{Enabled: true}, map[string]Source{
		"admin": sourceFunc(func(ctx context.Context) (Candidate, error) {
			calls.Add(1)
			started <- ctx
			select {
			case <-release:
				return Candidate{Version: "v1.2.4"}, nil
			case <-ctx.Done():
				return Candidate{}, ctx.Err()
			}
		}),
	})
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	first := checkAsync(c, firstCtx, true)
	sourceCtx := receiveCheckValue(t, started)
	secondCtx := observeWait(context.Background())
	second := checkAsync(c, secondCtx, true)
	receiveCheckValue(t, secondCtx.waiting)
	thirdCtx := observeWait(context.Background())
	third := checkAsync(c, thirdCtx, false)
	receiveCheckValue(t, thirdCtx.waiting)
	cancelFirst()
	if outcome := receiveCheckValue(t, first); !errors.Is(outcome.err, context.Canceled) {
		t.Fatalf("canceled caller err = %v", outcome.err)
	}
	if sourceCtx.Err() != nil {
		t.Fatalf("caller cancellation reached shared source: %v", sourceCtx.Err())
	}
	close(release)
	secondResult := receiveCheckValue(t, second)
	thirdResult := receiveCheckValue(t, third)
	if secondResult.err != nil || thirdResult.err != nil || calls.Load() != 1 {
		t.Fatalf("joined check errors = %v / %v, calls = %d", secondResult.err, thirdResult.err, calls.Load())
	}
	secondResult.result.Sources["admin"].Candidate.Version = "corrupted"
	delete(secondResult.result.Sources, "admin")
	if thirdResult.result.Sources["admin"].Candidate.Version != "v1.2.4" || c.Snapshot().Sources["admin"].Candidate.Version != "v1.2.4" {
		t.Fatal("joined callers share mutable snapshots")
	}
}

func TestSourcesRunConcurrentlyWithOneDeadline(t *testing.T) {
	for name, timeout := range map[string]time.Duration{"default": 0, "custom": time.Minute} {
		t.Run(name, func(t *testing.T) {
			started := make(chan context.Context, 2)
			release := make(chan struct{})
			source := sourceFunc(func(ctx context.Context) (Candidate, error) {
				started <- ctx
				select {
				case <-release:
					return Candidate{Version: "v1.2.4"}, nil
				case <-ctx.Done():
					return Candidate{}, ctx.Err()
				}
			})
			c := newTestChecker(t, Options{Enabled: true, Timeout: timeout}, map[string]Source{"admin": source, "gateway": source})
			before := time.Now()
			result := checkAsync(c, context.Background(), true)
			first := receiveCheckValue(t, started)
			second := receiveCheckValue(t, started)
			after := time.Now()
			d1, ok1 := first.Deadline()
			d2, ok2 := second.Deadline()
			if timeout == 0 {
				timeout = 5 * time.Second
			}
			if !ok1 || !ok2 || !d1.Equal(d2) || d1.Before(before.Add(timeout)) || d1.After(after.Add(timeout)) {
				t.Fatalf("source deadlines = %v / %v, expected one %v deadline", d1, d2, timeout)
			}
			close(release)
			outcome := receiveCheckValue(t, result)
			if outcome.err != nil || outcome.result.Sources["admin"].LastSuccess.Before(before) || outcome.result.Sources["admin"].LastSuccess.After(time.Now()) {
				t.Fatalf("default clock result = %+v, err = %v", outcome.result, outcome.err)
			}
		})
	}
}

func TestSharedTimeoutIsRecordedInSourceStates(t *testing.T) {
	started := make(chan context.Context, 2)
	source := sourceFunc(func(ctx context.Context) (Candidate, error) {
		started <- ctx
		<-ctx.Done()
		return Candidate{}, ctx.Err()
	})
	c := newTestChecker(t, Options{Enabled: true, Timeout: 10 * time.Millisecond}, map[string]Source{"admin": source, "gateway": source})
	result := checkAsync(c, context.Background(), true)
	receiveCheckValue(t, started)
	receiveCheckValue(t, started)
	outcome := receiveCheckValue(t, result)
	if outcome.err != nil {
		t.Fatalf("source deadline became caller failure: %v", outcome.err)
	}
	for name, state := range outcome.result.Sources {
		if state.Status != "unavailable" || state.ErrorCode != "timeout" {
			t.Fatalf("%s timeout state = %+v", name, state)
		}
	}
}

func TestResultAfterDeadlineCannotBecomeFresh(t *testing.T) {
	c := newTestChecker(t, Options{Enabled: true, Timeout: 10 * time.Millisecond}, map[string]Source{
		"admin": sourceFunc(func(ctx context.Context) (Candidate, error) {
			<-ctx.Done()
			// Even a misbehaving source cannot publish a late "success".
			return Candidate{Version: "v1.2.4"}, nil
		}),
	})
	result, err := c.Check(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	state := result.Sources["admin"]
	if state.Status != "unavailable" || state.ErrorCode != "timeout" || state.Candidate != nil || !state.LastSuccess.IsZero() {
		t.Fatalf("late response became a fresh candidate: %+v", state)
	}
}

func TestSourceResultsPublishIndependently(t *testing.T) {
	clock := newCheckerClock()
	gatewayStarted := make(chan struct{})
	releaseGateway := make(chan struct{})
	c := newTestChecker(t, Options{Enabled: true, Now: clock.Now}, map[string]Source{
		"admin": sourceFunc(func(context.Context) (Candidate, error) {
			return Candidate{Version: "v1.2.4"}, nil
		}),
		"gateway": sourceFunc(func(ctx context.Context) (Candidate, error) {
			close(gatewayStarted)
			select {
			case <-releaseGateway:
				return Candidate{}, errors.New("gateway is unavailable")
			case <-ctx.Done():
				return Candidate{}, ctx.Err()
			}
		}),
	})
	result := checkAsync(c, context.Background(), false)
	receiveCheckValue(t, gatewayStarted)
	deadline := time.Now().Add(2 * time.Second)
	for c.Snapshot().Sources["admin"].Status != "ready" {
		if time.Now().After(deadline) {
			t.Fatal("completed source was not published while other source was blocked")
		}
		runtime.Gosched()
	}
	state := c.Snapshot()
	if state.Sources["gateway"].Status != "checking" || state.Sources["admin"].Candidate.Version != "v1.2.4" {
		t.Fatalf("partial snapshot = %+v", state)
	}
	clock.Advance(time.Second)
	close(releaseGateway)
	outcome := receiveCheckValue(t, result)
	if outcome.err != nil || outcome.result.Sources["admin"].Status != "ready" || outcome.result.Sources["gateway"].Status != "unavailable" {
		t.Fatalf("partial failure result = %+v, err = %v", outcome.result, outcome.err)
	}
}

func TestRecheckPreservesFreshHistoryUntilResult(t *testing.T) {
	clock := newCheckerClock()
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	c := newTestChecker(t, Options{Enabled: true, Now: clock.Now}, map[string]Source{
		"admin": sourceFunc(func(ctx context.Context) (Candidate, error) {
			if calls.Add(1) == 2 {
				close(started)
				select {
				case <-release:
				case <-ctx.Done():
					return Candidate{}, ctx.Err()
				}
			}
			return Candidate{Version: "v1.2.4"}, nil
		}),
	})
	if _, err := c.Check(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	firstSuccess := clock.Now()
	clock.Advance(time.Minute)
	result := checkAsync(c, context.Background(), false)
	receiveCheckValue(t, started)
	state := c.Snapshot().Sources["admin"]
	if state.Status != "checking" || state.Stale || state.Candidate == nil || !state.LastSuccess.Equal(firstSuccess) {
		t.Fatalf("starting recheck prematurely invalidated fresh history: %+v", state)
	}
	if !state.LastAttempt.Equal(clock.Now()) {
		t.Fatalf("attempt start time = %v", state.LastAttempt)
	}
	clock.Advance(6*time.Hour - time.Minute)
	if !c.Snapshot().Sources["admin"].Stale {
		t.Fatal("history did not expire while recheck was running")
	}
	close(release)
	outcome := receiveCheckValue(t, result)
	if outcome.err != nil || !outcome.result.Sources["admin"].LastSuccess.Equal(clock.Now()) || outcome.result.Sources["admin"].Stale {
		t.Fatalf("completion timestamp = %+v, err = %v", outcome.result, outcome.err)
	}
	if _, err := c.Check(context.Background(), true); err != nil {
		t.Fatalf("cooldown was measured from completion instead of start: %v", err)
	}
}

type scheduledCheck struct {
	delay time.Duration
	tick  chan time.Time
}

func TestStartIsAsyncIdempotentAndSchedulesChecks(t *testing.T) {
	for name, interval := range map[string]time.Duration{"default": 0, "custom": time.Hour} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			clock := newCheckerClock()
			scheduled := make(chan scheduledCheck, 8)
			started := make(chan struct{}, 8)
			release := make(chan struct{}, 8)
			c := newTestChecker(t, Options{
				Enabled: true, Now: clock.Now, Interval: interval,
				After: func(d time.Duration) <-chan time.Time {
					tick := make(chan time.Time, 1)
					scheduled <- scheduledCheck{d, tick}
					return tick
				},
			}, map[string]Source{
				"admin": sourceFunc(func(ctx context.Context) (Candidate, error) {
					calls.Add(1)
					started <- struct{}{}
					select {
					case <-release:
						return Candidate{Version: "v1.2.4"}, nil
					case <-ctx.Done():
						return Candidate{}, ctx.Err()
					}
				}),
			})
			startReturned := make(chan struct{})
			go func() {
				c.Start()
				c.Start()
				close(startReturned)
			}()
			receiveCheckValue(t, startReturned)
			receiveCheckValue(t, started)
			release <- struct{}{}
			schedule := receiveCheckValue(t, scheduled)
			if interval == 0 {
				interval = 6 * time.Hour
			}
			if schedule.delay != interval || calls.Load() != 1 {
				t.Fatalf("schedule = %v, calls = %d", schedule.delay, calls.Load())
			}
			clock.Advance(interval)
			schedule.tick <- clock.Now()
			receiveCheckValue(t, started)
			if calls.Load() != 2 {
				t.Fatalf("scheduled calls = %d", calls.Load())
			}
			release <- struct{}{}
			next := receiveCheckValue(t, scheduled)
			c.Close()
			next.tick <- clock.Now()
			c.Start()
			if _, err := c.Check(context.Background(), false); !errors.Is(err, context.Canceled) {
				t.Fatalf("post-close check = %v", err)
			}
			if calls.Load() != 2 {
				t.Fatalf("close allowed more requests: %d", calls.Load())
			}
		})
	}
}

func TestCloseCancelsAndWaitsForSourceCleanup(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	releaseCleanup := make(chan struct{})
	defer close(releaseCleanup)
	c := newTestChecker(t, Options{Enabled: true}, map[string]Source{
		"admin": sourceFunc(func(ctx context.Context) (Candidate, error) {
			close(started)
			<-ctx.Done()
			close(canceled)
			<-releaseCleanup
			return Candidate{}, ctx.Err()
		}),
	})
	result := checkAsync(c, context.Background(), false)
	receiveCheckValue(t, started)
	closed := make(chan struct{})
	go func() {
		c.Close()
		close(closed)
	}()
	receiveCheckValue(t, canceled)
	select {
	case <-closed:
		t.Fatal("Close returned before source cleanup")
	default:
	}
	if outcome := receiveCheckValue(t, result); !errors.Is(outcome.err, context.Canceled) {
		t.Fatalf("waiting caller err = %v", outcome.err)
	}
	releaseCleanup <- struct{}{}
	receiveCheckValue(t, closed)
	c.Close()
}

func TestCanceledContextsAndClosedCheckerNeverStartSources(t *testing.T) {
	for _, scenario := range []string{"caller", "parent", "parent_deadline", "closed"} {
		t.Run(scenario, func(t *testing.T) {
			var calls atomic.Int32
			parent, cancelParent := context.WithCancel(context.Background())
			if scenario == "parent_deadline" {
				cancelParent()
				parent, cancelParent = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			}
			defer cancelParent()
			c, err := NewChecker(parent, Options{Enabled: true}, map[string]Source{
				"admin": sourceFunc(func(context.Context) (Candidate, error) {
					calls.Add(1)
					return Candidate{}, ErrNoCandidate
				}),
			})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			caller, cancelCaller := context.WithCancel(context.Background())
			defer cancelCaller()
			switch scenario {
			case "caller":
				cancelCaller()
			case "parent":
				cancelParent()
				c.Start()
			case "parent_deadline":
				c.Start()
			case "closed":
				c.Close()
				c.Start()
			}
			wantErr := error(context.Canceled)
			if scenario == "parent_deadline" {
				wantErr = context.DeadlineExceeded
			}
			if _, err := c.Check(caller, true); !errors.Is(err, wantErr) {
				t.Fatalf("canceled check err = %v", err)
			}
			if calls.Load() != 0 {
				t.Fatalf("canceled checker initiated %d requests", calls.Load())
			}
		})
	}
}

func TestParentCancellationStopsRunningChecks(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan context.Context, 1)
	c, err := NewChecker(parent, Options{Enabled: true}, map[string]Source{
		"admin": sourceFunc(func(ctx context.Context) (Candidate, error) {
			started <- ctx
			<-ctx.Done()
			return Candidate{}, ctx.Err()
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	result := checkAsync(c, context.Background(), true)
	sourceCtx := receiveCheckValue(t, started)
	cancel()
	outcome := receiveCheckValue(t, result)
	if !errors.Is(outcome.err, context.Canceled) || sourceCtx.Err() != context.Canceled {
		t.Fatalf("parent cancellation err = %v, source err = %v", outcome.err, sourceCtx.Err())
	}
	c.Close()
	if state := c.Snapshot().Sources["admin"]; state.Status != "unavailable" || state.ErrorCode != "canceled" {
		t.Fatalf("canceled source state = %+v", state)
	}
}

func TestConcurrentStartCheckClose(t *testing.T) {
	for iteration := 0; iteration < 30; iteration++ {
		var calls atomic.Int32
		c := newTestChecker(t, Options{Enabled: true}, map[string]Source{
			"admin": sourceFunc(func(ctx context.Context) (Candidate, error) {
				calls.Add(1)
				<-ctx.Done()
				return Candidate{}, ctx.Err()
			}),
		})
		begin := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < 9; i++ {
			wg.Add(1)
			go func(operation int) {
				defer wg.Done()
				<-begin
				switch operation % 3 {
				case 0:
					c.Start()
				case 1:
					c.Check(context.Background(), true)
				case 2:
					c.Close()
				}
				c.Snapshot()
			}(i)
		}
		close(begin)
		finished := make(chan struct{})
		go func() { wg.Wait(); close(finished) }()
		receiveCheckValue(t, finished)
		if calls.Load() > 1 {
			t.Fatalf("racing starts created %d flights", calls.Load())
		}
		before := calls.Load()
		c.Start()
		if _, err := c.Check(context.Background(), false); !errors.Is(err, context.Canceled) || calls.Load() != before {
			t.Fatalf("post-close work: err = %v, calls = %d", err, calls.Load())
		}
	}
}

package updatecheck

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Options controls background checks, the shared deadline and manual cooldown.
type Options struct {
	Enabled                     bool
	Interval, Timeout, Cooldown time.Duration
	Now                         func() time.Time
	After                       func(time.Duration) <-chan time.Time
}

var (
	ErrDisabled = errors.New("update checks disabled")
	ErrCooldown = errors.New("check cooldown")
)

type flight struct {
	done   chan struct{}
	result CheckResult
	err    error
}

// Checker owns the lifecycle and state shared by automatic and manual checks.
type Checker struct {
	mu        sync.Mutex
	wg        sync.WaitGroup
	startOnce sync.Once
	closeOnce sync.Once
	ctx       context.Context
	cancel    context.CancelFunc
	opts      Options
	sources   map[string]Source
	states    map[string]SourceState
	running   *flight
	lastStart time.Time
	hasStart  bool
	closed    bool
}

// NewChecker borrows its sources and ties their shared work to ctx, not to the
// context of an individual caller. Zero durations use the production defaults.
func NewChecker(ctx context.Context, opts Options, sources map[string]Source) (*Checker, error) {
	if opts.Interval < 0 || opts.Timeout < 0 || opts.Cooldown < 0 {
		return nil, errors.New("update check durations must not be negative")
	}
	if opts.Interval == 0 {
		opts.Interval = 6 * time.Hour
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.Cooldown == 0 {
		opts.Cooldown = time.Minute
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.After == nil {
		opts.After = time.After
	}
	ctx, cancel := context.WithCancel(ctx)
	c := &Checker{
		ctx: ctx, cancel: cancel, opts: opts,
		sources: make(map[string]Source, len(sources)),
		states:  make(map[string]SourceState, len(sources)),
	}
	status := "unchecked"
	if !opts.Enabled {
		status = "disabled"
	}
	for name, source := range sources {
		c.sources[name] = source
		c.states[name] = SourceState{Status: status}
	}
	return c, nil
}

// Start begins one asynchronous check, then schedules checks after each interval.
func (c *Checker) Start() {
	c.startOnce.Do(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if !c.opts.Enabled || c.closed || c.ctx.Err() != nil {
			return
		}
		// Every Add is protected by mu, as is Close's transition to closed.
		// Once Close begins waiting, neither Start nor Check can add work.
		c.wg.Add(1)
		go c.schedule()
	})
}

func (c *Checker) schedule() {
	defer c.wg.Done()
	c.Check(c.ctx, false)
	for c.ctx.Err() == nil {
		select {
		case <-c.ctx.Done():
			return
		case <-c.opts.After(c.opts.Interval):
			c.Check(c.ctx, false)
		}
	}
}

// Check joins an in-flight check before applying the instance-wide manual
// cooldown. Canceling ctx stops only this caller's wait.
func (c *Checker) Check(ctx context.Context, manual bool) (CheckResult, error) {
	c.mu.Lock()
	now := c.opts.Now()
	if !c.opts.Enabled {
		result := c.snapshotLocked(now)
		c.mu.Unlock()
		return result, ErrDisabled
	}
	if err := c.ctx.Err(); err != nil {
		result := c.snapshotLocked(now)
		c.mu.Unlock()
		return result, err
	}
	if err := ctx.Err(); err != nil {
		result := c.snapshotLocked(now)
		c.mu.Unlock()
		return result, err
	}
	f := c.running
	if f == nil {
		if manual && c.retryAfterLocked(now) > 0 {
			result := c.snapshotLocked(now)
			c.mu.Unlock()
			return result, ErrCooldown
		}
		c.lastStart, c.hasStart = now, true
		for name, state := range c.states {
			state.Status = "checking"
			state.LastAttempt = now
			state.ErrorCode = ""
			c.states[name] = state
		}
		f = &flight{done: make(chan struct{})}
		c.running = f
		// Create exactly one timeout context before dispatching any sources.
		flightCtx, cancel := context.WithTimeout(c.ctx, c.opts.Timeout)
		c.wg.Add(1)
		go c.runFlight(flightCtx, cancel, f)
	}
	c.mu.Unlock()
	return c.wait(ctx, f)
}

func (c *Checker) wait(ctx context.Context, f *flight) (CheckResult, error) {
	select {
	case <-ctx.Done():
		return c.Snapshot(), ctx.Err()
	case <-c.ctx.Done():
		return c.Snapshot(), c.ctx.Err()
	case <-f.done:
		// Cancellation may have raced with flight completion.
		if err := ctx.Err(); err != nil {
			return c.Snapshot(), err
		}
		if err := c.ctx.Err(); err != nil {
			return c.Snapshot(), err
		}
		return cloneResult(f.result, c.opts.Now(), c.opts.Interval), f.err
	}
}

func (c *Checker) runFlight(ctx context.Context, cancel context.CancelFunc, f *flight) {
	defer c.wg.Done()
	defer cancel()
	var workers sync.WaitGroup
	for name, source := range c.sources {
		workers.Add(1)
		go func() {
			defer workers.Done()
			var candidate Candidate
			err := ctx.Err()
			if err == nil {
				candidate, err = source.Latest(ctx)
				// A result arriving after cancellation/deadline is not fresh.
				if ctx.Err() != nil {
					err = ctx.Err()
				}
			}
			c.mu.Lock()
			defer c.mu.Unlock()
			state := c.states[name]
			switch {
			case err == nil:
				state.Status = "ready"
				state.Candidate = &candidate
				state.LastSuccess = c.opts.Now()
				state.Stale = false
				state.ErrorCode = ""
			case errors.Is(err, ErrNoCandidate):
				state.Status = "no_candidate"
				state.Stale = state.Candidate != nil
				state.ErrorCode = ""
			default:
				state.Status = "unavailable"
				state.Stale = state.Candidate != nil
				state.ErrorCode = sourceErrorCode(err)
			}
			c.states[name] = state
		}()
	}
	workers.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = nil
	f.result = c.snapshotLocked(c.opts.Now())
	f.err = c.ctx.Err()
	close(f.done)
}

// Snapshot returns an isolated, age-adjusted view without contacting a source.
func (c *Checker) Snapshot() CheckResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotLocked(c.opts.Now())
}

// Close cancels and waits for the scheduler and all registered source work.
// It never closes a source's borrowed HTTP client.
func (c *Checker) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.cancel()
		c.mu.Unlock()
		c.wg.Wait()
	})
}

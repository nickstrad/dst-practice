// Package retry runs an operation until it succeeds, waiting between
// attempts with capped exponential backoff and full jitter.
//
// The schedule looks like this with InitialDelay=100ms, Multiplier=2,
// MaxDelay=1s:
//
//	attempt 1 fails -> sleep somewhere in [0, 100ms)
//	attempt 2 fails -> sleep somewhere in [0, 200ms)
//	attempt 3 fails -> sleep somewhere in [0, 400ms)
//	attempt 4 fails -> sleep somewhere in [0, 800ms)
//	attempt 5 fails -> sleep somewhere in [0, 1s)     (capped)
//	attempt 6 fails -> sleep somewhere in [0, 1s)     (capped)
//	...
//
// "Full jitter" means the actual sleep is a uniformly random value between
// zero and the computed backoff. This spreads out clients that all failed
// at the same moment, so they do not retry in lockstep. See
// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
//
// The retrier never reads the wall clock and never spawns a goroutine. It
// gets a Clock and a Rand at construction so tests can drive it
// deterministically.
package retry

import (
	"errors"
	"fmt"
	"math"
	"time"

	"dstpractice/sim/clock"
)

// Rand is the source of jitter. Float64 must return a value in [0, 1).
// *math/rand/v2.Rand satisfies this.
type Rand interface {
	Float64() float64
}

// Policy describes how long to keep trying and how long to wait between
// attempts.
type Policy struct {
	// InitialDelay is the upper bound of the first sleep. Must be > 0.
	InitialDelay time.Duration

	// MaxDelay caps the upper bound of every sleep. Must be >= InitialDelay.
	MaxDelay time.Duration

	// Multiplier grows the upper bound after each failed attempt.
	// 2 doubles it every time. Must be >= 1.
	Multiplier float64

	// MaxAttempts is the total number of times the operation may run,
	// including the first. Must be >= 1.
	MaxAttempts int

	// Timeout is the total time budget measured from the start of Do.
	// Zero means no time limit. When the next sleep would end after the
	// deadline, Do gives up instead of sleeping.
	Timeout time.Duration
}

// DefaultPolicy is a reasonable starting point for calling a flaky
// network service.
var DefaultPolicy = Policy{
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	Multiplier:   2,
	MaxAttempts:  5,
	Timeout:      30 * time.Second,
}

// Retrier runs operations according to a Policy.
type Retrier struct {
	policy Policy
	clock  clock.Clock
	rand   Rand
}

// New builds a Retrier. It panics on an invalid policy because a bad
// policy is a programming error, not a runtime condition.
func New(policy Policy, clk clock.Clock, rnd Rand) *Retrier {
	if err := policy.validate(); err != nil {
		panic("retry: " + err.Error())
	}
	return &Retrier{policy: policy, clock: clk, rand: rnd}
}

func (p Policy) validate() error {
	switch {
	case p.InitialDelay <= 0:
		return errors.New("InitialDelay must be > 0")
	case p.MaxDelay < p.InitialDelay:
		return errors.New("MaxDelay must be >= InitialDelay")
	// NaN fails every comparison, so "< 1" alone would let it through.
	case p.Multiplier < 1 || math.IsNaN(p.Multiplier):
		return errors.New("Multiplier must be >= 1")
	case p.MaxAttempts < 1:
		return errors.New("MaxAttempts must be >= 1")
	case p.Timeout < 0:
		return errors.New("Timeout must be >= 0")
	}
	return nil
}

// Do runs op until it returns nil, returns a permanent error, or the
// policy is exhausted.
//
// The returned error is:
//   - nil if op eventually succeeded
//   - the op's own error, unwrapped, if op returned Stop(err)
//   - an *ExhaustedError if attempts or time ran out. It wraps both the
//     reason (ErrMaxAttempts or ErrTimeout) and the last error op returned.
func (r *Retrier) Do(op func() error) error {
	// A zero deadline means "no time limit".
	var deadline time.Time
	if r.policy.Timeout > 0 {
		deadline = r.clock.Now().Add(r.policy.Timeout)
	}

	for attempt := 1; ; attempt++ {
		err := op()
		if err == nil {
			return nil
		}

		if perm, ok := errors.AsType[*permanent](err); ok {
			return perm.err
		}

		if attempt >= r.policy.MaxAttempts {
			return &ExhaustedError{Attempts: attempt, Reason: ErrMaxAttempts, Err: err}
		}

		sleep := r.nextSleep(attempt)

		wakeUp := r.clock.Now().Add(sleep)
		if !deadline.IsZero() && wakeUp.After(deadline) {
			return &ExhaustedError{Attempts: attempt, Reason: ErrTimeout, Err: err}
		}

		r.clock.Sleep(sleep)
	}
}

// nextSleep picks how long to wait after the given failed attempt
// (1-based). It is the backoff formula plus full jitter.
func (r *Retrier) nextSleep(attempt int) time.Duration {
	upper := Backoff(r.policy, attempt)
	jittered := float64(upper) * r.rand.Float64()
	return time.Duration(jittered)
}

// Backoff returns the upper bound of the sleep after the given failed
// attempt (1-based), before jitter is applied. It is exported so tests can
// check the schedule without running anything.
//
//	Backoff(p, 1) == InitialDelay
//	Backoff(p, 2) == InitialDelay * Multiplier
//	Backoff(p, 3) == InitialDelay * Multiplier^2
//	... capped at MaxDelay.
func Backoff(p Policy, attempt int) time.Duration {
	delay := float64(p.InitialDelay)
	for i := 1; i < attempt; i++ {
		delay *= p.Multiplier
		if delay >= float64(p.MaxDelay) {
			return p.MaxDelay
		}
	}
	return time.Duration(delay)
}

// Stop marks err as permanent, meaning retrying will not help, for example
// a 400 Bad Request. Returning Stop(err) from an operation makes Do give up
// immediately and return err unchanged.
func Stop(err error) error {
	return &permanent{err: err}
}

// permanent is the wrapper Stop puts around an error. Do looks for it with
// errors.As and unwraps it, so callers never see this type.
type permanent struct {
	err error
}

func (p *permanent) Error() string { return p.err.Error() }

// Why Do gave up. Check with errors.Is(err, retry.ErrTimeout).
var (
	ErrMaxAttempts = errors.New("max attempts reached")
	ErrTimeout     = errors.New("timeout reached")
)

// ExhaustedError is returned when Do gave up. It wraps two errors: the
// Reason (ErrMaxAttempts or ErrTimeout) and the last error the operation
// returned. errors.Is finds either one.
type ExhaustedError struct {
	Attempts int   // how many times op ran
	Reason   error // ErrMaxAttempts or ErrTimeout
	Err      error // the last error op returned
}

func (e *ExhaustedError) Error() string {
	return fmt.Sprintf("retry: gave up after %d attempt(s): %v: %v", e.Attempts, e.Reason, e.Err)
}

// Unwrap returns both wrapped errors. errors.Is and errors.As walk every
// error in the slice.
func (e *ExhaustedError) Unwrap() []error {
	return []error{e.Reason, e.Err}
}

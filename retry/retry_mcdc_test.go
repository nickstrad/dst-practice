package retry_test

// Decision tables. One table per decision in retry.go that has more than
// one condition or a boundary worth pinning. Each row changes one
// condition from a neighbor, so the table shows that every condition
// moves the outcome on its own. That is the MC/DC idea done by hand,
// since Go ships no MC/DC tool. See docs/testing.md.

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"dstpractice/retry"
	"dstpractice/sim/clock"
)

// TestDeadlineDecision covers the compound decision in Do:
//
//	!deadline.IsZero() && wakeUp.After(deadline)
//
// Every sleep is exactly 100ms here, so wake-up times are round numbers.
func TestDeadlineDecision(t *testing.T) {
	base := retry.Policy{
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   1,
		MaxAttempts:  3,
	}
	rows := []struct {
		name       string
		timeout    time.Duration
		wantReason error
		wantCalls  int
	}{
		// deadline zero: the second condition is never evaluated.
		{"no deadline", 0, retry.ErrMaxAttempts, 3},
		// deadline set, every wake-up lands before it.
		{"deadline set, wake before", time.Second, retry.ErrMaxAttempts, 3},
		// deadline set, a wake-up lands exactly on it. After is false.
		{"deadline set, wake exactly at", 200 * time.Millisecond, retry.ErrMaxAttempts, 3},
		// deadline set, the second wake-up would land after it.
		{"deadline set, wake after", 150 * time.Millisecond, retry.ErrTimeout, 2},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			p := base
			p.Timeout = row.timeout
			clk := clock.NewFake(epoch)
			r := retry.New(p, clk, halfRand)

			op := &flakyOp{failures: 100}
			err := r.Do(op.Do)

			if !errors.Is(err, row.wantReason) {
				t.Errorf("Do returned %v, want %v", err, row.wantReason)
			}
			if op.calls != row.wantCalls {
				t.Errorf("op ran %d times, want %d", op.calls, row.wantCalls)
			}
		})
	}
}

// TestMaxAttemptsDecision pins the boundary of attempt >= MaxAttempts.
func TestMaxAttemptsDecision(t *testing.T) {
	rows := []struct {
		name        string
		maxAttempts int
		failures    int
		wantErr     bool
		wantCalls   int
	}{
		{"one attempt, fails", 1, 100, true, 1},
		{"one attempt, succeeds", 1, 0, false, 1},
		{"two attempts, fails twice", 2, 100, true, 2},
		{"two attempts, succeeds on second", 2, 1, false, 2},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			p := retry.DefaultPolicy
			p.MaxAttempts = row.maxAttempts
			p.Timeout = 0
			clk := clock.NewFake(epoch)
			r := retry.New(p, clk, halfRand)

			op := &flakyOp{failures: row.failures}
			err := r.Do(op.Do)

			if (err != nil) != row.wantErr {
				t.Errorf("Do returned %v, want error=%v", err, row.wantErr)
			}
			if op.calls != row.wantCalls {
				t.Errorf("op ran %d times, want %d", op.calls, row.wantCalls)
			}
		})
	}
}

// TestBackoffCapDecision covers the cap check inside the Backoff loop:
//
//	delay >= float64(p.MaxDelay)
func TestBackoffCapDecision(t *testing.T) {
	rows := []struct {
		name     string
		maxDelay time.Duration
		attempt  int
		want     time.Duration
	}{
		// attempt 1 never enters the loop.
		{"first attempt", time.Second, 1, 100 * time.Millisecond},
		// 100 -> 200 -> 400, all below the cap.
		{"below cap", time.Second, 3, 400 * time.Millisecond},
		// 100 -> 200 -> 400, and 400 equals the cap.
		{"exactly at cap", 400 * time.Millisecond, 3, 400 * time.Millisecond},
		// 100 -> 200 -> 400, and 400 passes the cap of 300.
		{"over cap", 300 * time.Millisecond, 3, 300 * time.Millisecond},
		// the cap equals InitialDelay, so the first multiply hits it.
		{"cap at start", 100 * time.Millisecond, 5, 100 * time.Millisecond},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			p := retry.Policy{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     row.maxDelay,
				Multiplier:   2,
				MaxAttempts:  1,
			}
			if got := retry.Backoff(p, row.attempt); got != row.want {
				t.Errorf("Backoff(attempt=%d) = %v, want %v", row.attempt, got, row.want)
			}
		})
	}
}

// TestPolicyValidationDecisions covers the switch in Policy.validate.
// The first row is valid. Every other row breaks exactly one field and
// expects the panic message to name that field.
func TestPolicyValidationDecisions(t *testing.T) {
	valid := retry.Policy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2,
		MaxAttempts:  3,
		Timeout:      time.Minute,
	}
	rows := []struct {
		name     string
		mutate   func(p *retry.Policy)
		wantWord string // empty means no panic
	}{
		{"valid", func(p *retry.Policy) {}, ""},
		{"InitialDelay zero", func(p *retry.Policy) { p.InitialDelay = 0 }, "InitialDelay"},
		{"MaxDelay below InitialDelay", func(p *retry.Policy) { p.MaxDelay = p.InitialDelay - 1 }, "MaxDelay"},
		{"MaxDelay equals InitialDelay", func(p *retry.Policy) { p.MaxDelay = p.InitialDelay }, ""},
		{"Multiplier below one", func(p *retry.Policy) { p.Multiplier = 0.5 }, "Multiplier"},
		{"Multiplier exactly one", func(p *retry.Policy) { p.Multiplier = 1 }, ""},
		{"Multiplier NaN", func(p *retry.Policy) { p.Multiplier = math.NaN() }, "Multiplier"},
		{"MaxAttempts zero", func(p *retry.Policy) { p.MaxAttempts = 0 }, "MaxAttempts"},
		{"Timeout negative", func(p *retry.Policy) { p.Timeout = -1 }, "Timeout"},
		{"Timeout zero", func(p *retry.Policy) { p.Timeout = 0 }, ""},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			p := valid
			row.mutate(&p)
			msg := panicMessage(func() { retry.New(p, clock.NewFake(epoch), halfRand) })

			if row.wantWord == "" && msg != "" {
				t.Errorf("New panicked with %q, want no panic", msg)
			}
			if row.wantWord != "" && !strings.Contains(msg, row.wantWord) {
				t.Errorf("New panicked with %q, want a message naming %s", msg, row.wantWord)
			}
		})
	}
}

// panicMessage runs fn and returns the panic text, or "" if it did not
// panic.
func panicMessage(fn func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = r.(string)
		}
	}()
	fn()
	return ""
}

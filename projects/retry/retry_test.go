package retry_test

// Spec tests. Each test is one named behavior with exact numbers, so a
// reader learns what the retrier does from the test names and bodies.

import (
	"errors"
	"slices"
	"testing"
	"time"

	"dstpractice/projects/retry"
	"dstpractice/sim/clock"
)

// I8: the schedule starts at InitialDelay, multiplies, and caps.
func TestBackoffSchedule(t *testing.T) {
	want := []time.Duration{
		200 * time.Millisecond,  // attempt 1
		400 * time.Millisecond,  // attempt 2
		800 * time.Millisecond,  // attempt 3
		1600 * time.Millisecond, // attempt 4
		2 * time.Second,         // attempt 5, capped
		2 * time.Second,         // attempt 6, still capped
	}
	for i, w := range want {
		attempt := i + 1
		if got := retry.Backoff(schedulePolicy, attempt); got != w {
			t.Errorf("Backoff(attempt=%d) = %v, want %v", attempt, got, w)
		}
	}
}

// I2, I5: one call, no sleep, nil error.
func TestSucceedsFirstTry(t *testing.T) {
	clk := clock.NewFake(epoch)
	r := retry.New(retry.DefaultPolicy, clk, halfRand)

	op := &flakyOp{failures: 0}
	if err := r.Do(op.Do); err != nil {
		t.Fatalf("Do returned %v, want nil", err)
	}
	if op.calls != 1 {
		t.Errorf("op ran %d times, want 1", op.calls)
	}
	if len(clk.Sleeps) != 0 {
		t.Errorf("slept %v, want no sleeps", clk.Sleeps)
	}
}

// I3: with jitter fixed at one half, the sleeps are half the schedule.
func TestSleepsFollowSchedule(t *testing.T) {
	clk := clock.NewFake(epoch)
	r := retry.New(schedulePolicy, clk, halfRand)

	op := &flakyOp{failures: 5}
	if err := r.Do(op.Do); err != nil {
		t.Fatalf("Do returned %v, want nil", err)
	}
	if op.calls != 6 {
		t.Errorf("op ran %d times, want 6", op.calls)
	}

	want := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1 * time.Second,
	}
	if !slices.Equal(clk.Sleeps, want) {
		t.Errorf("sleeps = %v, want %v", clk.Sleeps, want)
	}
}

// I1, I2, I6: the attempt cap holds and the error wraps the op's error.
func TestGivesUpAfterMaxAttempts(t *testing.T) {
	p := retry.Policy{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1,
		MaxAttempts:  3,
	}
	clk := clock.NewFake(epoch)
	r := retry.New(p, clk, halfRand)

	op := &flakyOp{failures: 100}
	err := r.Do(op.Do)

	var exhausted *retry.ExhaustedError
	if !errors.As(err, &exhausted) {
		t.Fatalf("Do returned %v, want *ExhaustedError", err)
	}
	if !errors.Is(err, retry.ErrMaxAttempts) {
		t.Errorf("reason = %v, want ErrMaxAttempts", exhausted.Reason)
	}
	if exhausted.Attempts != 3 || op.calls != 3 {
		t.Errorf("attempts = %d, calls = %d, want 3 and 3", exhausted.Attempts, op.calls)
	}
	if !errors.Is(err, errFlaky) {
		t.Errorf("errors.Is(err, errFlaky) = false, want true")
	}
	// Three attempts means two sleeps between them. No sleep after the last.
	if len(clk.Sleeps) != 2 {
		t.Errorf("slept %d times, want 2", len(clk.Sleeps))
	}
}

// I4: the retrier stops before a sleep that would cross the deadline.
func TestGivesUpAtDeadline(t *testing.T) {
	p := retry.Policy{
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   1,
		MaxAttempts:  1000,
		Timeout:      250 * time.Millisecond,
	}
	clk := clock.NewFake(epoch)
	r := retry.New(p, clk, halfRand) // every sleep is exactly 100ms

	op := &flakyOp{failures: 100}
	err := r.Do(op.Do)

	if !errors.Is(err, retry.ErrTimeout) {
		t.Fatalf("Do returned %v, want ErrTimeout", err)
	}
	// Timeline: t=0 attempt 1, sleep to 100ms, attempt 2, sleep to 200ms,
	// attempt 3, next sleep would end at 300ms > 250ms, so stop.
	if op.calls != 3 {
		t.Errorf("op ran %d times, want 3", op.calls)
	}
	if clk.Now().Sub(epoch) > p.Timeout {
		t.Errorf("clock advanced to %v past start, beyond timeout %v", clk.Now().Sub(epoch), p.Timeout)
	}
}

// I4: time the op itself burns counts against the deadline.
func TestDeadlineCountsTimeSpentInsideOp(t *testing.T) {
	p := retry.Policy{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1,
		MaxAttempts:  1000,
		Timeout:      1 * time.Second,
	}
	clk := clock.NewFake(epoch)
	r := retry.New(p, clk, halfRand)

	calls := 0
	err := r.Do(func() error {
		calls++
		clk.Advance(600 * time.Millisecond) // a slow RPC
		return errFlaky
	})

	if !errors.Is(err, retry.ErrTimeout) {
		t.Fatalf("Do returned %v, want ErrTimeout", err)
	}
	// Two slow calls burn 1.2s, so the retrier must stop after the second.
	if calls != 2 {
		t.Errorf("op ran %d times, want 2", calls)
	}
}

// I7: Stop ends the loop at once and Do returns the unwrapped error.
func TestStopEndsRetryingImmediately(t *testing.T) {
	clk := clock.NewFake(epoch)
	r := retry.New(retry.DefaultPolicy, clk, halfRand)

	errBadRequest := errors.New("400 bad request")
	calls := 0
	err := r.Do(func() error {
		calls++
		return retry.Stop(errBadRequest)
	})

	if err != errBadRequest {
		t.Errorf("Do returned %v, want the exact stopped error", err)
	}
	if calls != 1 {
		t.Errorf("op ran %d times, want 1", calls)
	}
	if len(clk.Sleeps) != 0 {
		t.Errorf("slept %v, want no sleeps", clk.Sleeps)
	}
}

// I9: a bad policy is a programming error, so New panics.
func TestInvalidPolicyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New did not panic on invalid policy")
		}
	}()
	retry.New(retry.Policy{}, clock.NewFake(epoch), halfRand)
}

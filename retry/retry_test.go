package retry_test

import (
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"dstpractice/retry"
	"dstpractice/sim/clock"
)

// Every test starts the fake clock here. The exact instant does not matter.
var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

var errFlaky = errors.New("flaky")

// schedulePolicy doubles from 200ms and caps at 2s. With halfRand every
// sleep is exactly half the backoff: 100ms, 200ms, 400ms, 800ms, 1s, 1s...
var schedulePolicy = retry.Policy{
	InitialDelay: 200 * time.Millisecond,
	MaxDelay:     2 * time.Second,
	Multiplier:   2,
	MaxAttempts:  10,
}

// fixedRand always returns the same value, so a test can predict the
// exact sleep instead of getting a random one.
type fixedRand struct{ value float64 }

func (f fixedRand) Float64() float64 { return f.value }

// halfRand makes every sleep exactly half its backoff upper bound.
var halfRand = fixedRand{0.5}

// flakyOp is an operation that fails a fixed number of times, then
// succeeds. Pass op.Do to the retrier and read op.calls afterwards.
type flakyOp struct {
	failures int // how many calls return an error before succeeding
	calls    int // how many times Do has run
}

func (op *flakyOp) Do() error {
	op.calls++
	if op.calls <= op.failures {
		return errFlaky
	}
	return nil
}

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

func TestInvalidPolicyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New did not panic on invalid policy")
		}
	}()
	retry.New(retry.Policy{}, clock.NewFake(epoch), halfRand)
}

// Flags for the seeded test below.
//
//	go test ./retry -run TestInvariantsAcrossSeeds -seed 42   # replay one seed
//	go test ./retry -runs 10000                                # try more seeds
var (
	seedFlag = flag.Uint64("seed", 0, "run only this seed (0 means many random seeds)")
	runsFlag = flag.Int("runs", 1000, "how many random seeds to try when -seed is not set")
)

// TestInvariantsAcrossSeeds is the DST-style test. It runs many seeded
// simulations with real jitter and checks properties that must hold for
// every seed. On failure it prints the seed so you can replay just that one.
func TestInvariantsAcrossSeeds(t *testing.T) {
	if *seedFlag != 0 {
		checkInvariants(t, *seedFlag)
		return
	}
	for range *runsFlag {
		checkInvariants(t, rand.Uint64())
	}
}

func checkInvariants(t *testing.T, seed uint64) {
	t.Helper()
	rng := rand.New(rand.NewPCG(seed, 0))

	// Randomize the policy too, within sane bounds, so we cover more than
	// one shape of schedule.
	p := retry.Policy{
		InitialDelay: time.Duration(1+rng.IntN(500)) * time.Millisecond,
		Multiplier:   1 + rng.Float64()*3, // [1, 4)
		MaxAttempts:  1 + rng.IntN(20),
	}
	p.MaxDelay = p.InitialDelay * time.Duration(1+rng.IntN(20))
	if rng.IntN(2) == 0 {
		p.Timeout = time.Duration(rng.IntN(10000)) * time.Millisecond
	}

	clk := clock.NewFake(epoch)
	r := retry.New(p, clk, rng)
	op := &flakyOp{failures: rng.IntN(30)}
	err := r.Do(op.Do)

	// Every failure message starts with enough to replay this run.
	run := fmt.Sprintf("seed=%d policy=%+v failures=%d", seed, p, op.failures)

	// Invariant: never more than MaxAttempts calls.
	if op.calls > p.MaxAttempts {
		t.Errorf("%s: op ran %d times, more than MaxAttempts", run, op.calls)
	}

	// Invariant: exactly one fewer sleep than calls.
	if len(clk.Sleeps) != op.calls-1 {
		t.Errorf("%s: %d sleeps for %d calls", run, len(clk.Sleeps), op.calls)
	}

	// Invariant: every sleep is within [0, Backoff(attempt)).
	for i, s := range clk.Sleeps {
		attempt := i + 1
		upper := retry.Backoff(p, attempt)
		if s < 0 || s >= upper {
			t.Errorf("%s: sleep %d = %v, outside [0, %v)", run, i, s, upper)
		}
	}

	// Invariant: never sleep past the deadline.
	if p.Timeout > 0 && clk.Now().Sub(epoch) > p.Timeout {
		t.Errorf("%s: clock ended at %v past start, beyond timeout %v", run, clk.Now().Sub(epoch), p.Timeout)
	}

	// Invariant: the result matches what the model says.
	// The model: success iff the op succeeded on some attempt we made.
	succeeded := op.calls > op.failures
	if succeeded && err != nil {
		t.Errorf("%s: op succeeded on call %d but Do returned %v", run, op.calls, err)
	}
	if !succeeded && err == nil {
		t.Errorf("%s: op never succeeded but Do returned nil", run)
	}
	if !succeeded && !errors.Is(err, errFlaky) {
		t.Errorf("%s: Do returned %v, which does not wrap the op's error", run, err)
	}
}

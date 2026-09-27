package retry_test

// Shared doubles and the invariant oracle for every test file in this
// package. docs/testing.md says what each test file is for.

import (
	"errors"
	"testing"
	"time"

	"dstpractice/projects/retry"
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

// checkRun drives one retry to completion against the fake clock and
// checks every invariant in invariants.md that a single run can show.
// Both the seeded test and the fuzz target call it, so the invariants are
// written down once. label is printed first on every failure so the run
// can be replayed.
func checkRun(t *testing.T, label string, p retry.Policy, failures int, rng retry.Rand) {
	t.Helper()

	clk := clock.NewFake(epoch)
	r := retry.New(p, clk, rng)
	op := &flakyOp{failures: failures}
	err := r.Do(op.Do)

	// I1: never more than MaxAttempts calls.
	if op.calls > p.MaxAttempts {
		t.Errorf("%s: op ran %d times, more than MaxAttempts", label, op.calls)
	}

	// I2: exactly one fewer sleep than calls.
	if len(clk.Sleeps) != op.calls-1 {
		t.Errorf("%s: %d sleeps for %d calls", label, len(clk.Sleeps), op.calls)
	}

	// I3: every sleep is within [0, Backoff(attempt)).
	for i, s := range clk.Sleeps {
		attempt := i + 1
		upper := retry.Backoff(p, attempt)
		if s < 0 || s >= upper {
			t.Errorf("%s: sleep %d = %v, outside [0, %v)", label, i, s, upper)
		}
	}

	// I4: never sleep past the deadline.
	if p.Timeout > 0 && clk.Now().Sub(epoch) > p.Timeout {
		t.Errorf("%s: clock ended at %v past start, beyond timeout %v", label, clk.Now().Sub(epoch), p.Timeout)
	}

	// I5: Do returns nil exactly when some attempt succeeded.
	succeeded := op.calls > op.failures
	if succeeded && err != nil {
		t.Errorf("%s: op succeeded on call %d but Do returned %v", label, op.calls, err)
	}
	if !succeeded && err == nil {
		t.Errorf("%s: op never succeeded but Do returned nil", label)
	}

	// I6: when Do gives up, the error wraps the op's last error.
	if !succeeded && !errors.Is(err, errFlaky) {
		t.Errorf("%s: Do returned %v, which does not wrap the op's error", label, err)
	}
}

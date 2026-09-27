package retry_test

// Fuzz targets. A plain `go test` replays only the seed corpus given with
// f.Add plus anything saved under testdata/fuzz. `make fuzz` lets the
// fuzzer search for new inputs, guided by coverage.

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"dstpractice/retry"
)

// maxExact is the largest delay Backoff computes exactly. Backoff
// multiplies in float64, which holds every integer only below 2^53.
// That is about 104 days, far beyond any real retry delay, so the fuzz
// targets stay inside it. invariants.md lists this under Not promised.
const maxExact = time.Duration(1 << 53)

// foldDuration maps any int64 into [1, maxExact).
func foldDuration(raw int64) time.Duration {
	return time.Duration(uint64(raw)%uint64(maxExact-1)) + 1
}

// foldCount maps any int into [1, limit].
func foldCount(raw int, limit int) int {
	return int(uint(raw)%uint(limit)) + 1
}

// validPolicy restates the rules from the Policy field comments. The
// fuzz targets use it as the oracle for New and to skip inputs that
// would panic on purpose.
func validPolicy(p retry.Policy) bool {
	return p.InitialDelay > 0 &&
		p.MaxDelay >= p.InitialDelay &&
		p.Multiplier >= 1 &&
		!math.IsNaN(p.Multiplier) &&
		p.MaxAttempts >= 1 &&
		p.Timeout >= 0
}

// FuzzBackoff checks I8: the schedule starts at InitialDelay, never
// decreases, and never passes MaxDelay.
func FuzzBackoff(f *testing.F) {
	f.Add(int64(200*time.Millisecond), int64(2*time.Second), 2.0, 5)
	f.Add(int64(time.Millisecond), int64(time.Millisecond), 1.0, 1)
	f.Add(int64(time.Second), int64(time.Hour), 1.0000001, 999)
	f.Add(int64(time.Second), int64(time.Hour), math.Inf(1), 2)

	f.Fuzz(func(t *testing.T, rawInitial, rawMax int64, multiplier float64, rawAttempt int) {
		p := retry.Policy{
			InitialDelay: foldDuration(rawInitial),
			MaxDelay:     foldDuration(rawMax),
			Multiplier:   multiplier,
			MaxAttempts:  1,
		}
		if p.MaxDelay < p.InitialDelay {
			p.InitialDelay, p.MaxDelay = p.MaxDelay, p.InitialDelay
		}
		if !validPolicy(p) {
			t.Skip()
		}
		// Bound the attempt so a Multiplier of 1 cannot loop for seconds.
		attempt := foldCount(rawAttempt, 1000)

		got := retry.Backoff(p, attempt)
		next := retry.Backoff(p, attempt+1)
		label := fmt.Sprintf("policy=%+v attempt=%d", p, attempt)

		if attempt == 1 && got != p.InitialDelay {
			t.Errorf("%s: first backoff = %v, want InitialDelay", label, got)
		}
		if got < p.InitialDelay || got > p.MaxDelay {
			t.Errorf("%s: backoff = %v, outside [InitialDelay, MaxDelay]", label, got)
		}
		if next < got {
			t.Errorf("%s: backoff fell from %v to %v on the next attempt", label, got, next)
		}
	})
}

// FuzzPolicyValidation checks I9: New panics exactly when validPolicy
// says the policy is bad.
func FuzzPolicyValidation(f *testing.F) {
	f.Add(int64(time.Millisecond), int64(time.Second), 2.0, 3, int64(0))
	f.Add(int64(0), int64(0), 0.0, 0, int64(0))
	f.Add(int64(time.Millisecond), int64(time.Second), math.NaN(), 3, int64(0))
	f.Add(int64(time.Millisecond), int64(time.Second), 2.0, 3, int64(-1))

	f.Fuzz(func(t *testing.T, initial, maxDelay int64, multiplier float64, attempts int, timeout int64) {
		p := retry.Policy{
			InitialDelay: time.Duration(initial),
			MaxDelay:     time.Duration(maxDelay),
			Multiplier:   multiplier,
			MaxAttempts:  attempts,
			Timeout:      time.Duration(timeout),
		}
		want := validPolicy(p)
		got := newDoesNotPanic(p)
		if got != want {
			t.Errorf("policy=%+v: New accepted=%v, want %v", p, got, want)
		}
	})
}

// newDoesNotPanic reports whether New accepts the policy.
func newDoesNotPanic(p retry.Policy) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	retry.New(p, nil, nil)
	return true
}

// FuzzDo checks I1 through I6 on one run whose policy, failure count, and
// jitter seed all come from the fuzzer instead of from a random draw.
func FuzzDo(f *testing.F) {
	f.Add(uint64(1), int64(200*time.Millisecond), int64(2*time.Second), 2.0, 10, int64(0), 5)
	f.Add(uint64(2), int64(200*time.Millisecond), int64(200*time.Millisecond), 1.0, 1000, int64(250*time.Millisecond), 100)
	f.Add(uint64(3), int64(time.Millisecond), int64(time.Millisecond), 1.0, 1, int64(0), 0)

	f.Fuzz(func(t *testing.T, seed uint64, rawInitial, rawMax int64, multiplier float64, rawAttempts int, rawTimeout int64, rawFailures int) {
		p := retry.Policy{
			InitialDelay: foldDuration(rawInitial),
			MaxDelay:     foldDuration(rawMax),
			Multiplier:   1 + math.Abs(multiplier),
			MaxAttempts:  foldCount(rawAttempts, 1000),
			Timeout:      time.Duration(uint64(rawTimeout) % uint64(maxExact)),
		}
		if p.MaxDelay < p.InitialDelay {
			p.InitialDelay, p.MaxDelay = p.MaxDelay, p.InitialDelay
		}
		if !validPolicy(p) {
			t.Skip()
		}
		failures := int(uint(rawFailures) % 2000)

		rng := rand.New(rand.NewPCG(seed, 0))
		label := fmt.Sprintf("seed=%d policy=%+v failures=%d", seed, p, failures)
		checkRun(t, label, p, failures, rng)
	})
}

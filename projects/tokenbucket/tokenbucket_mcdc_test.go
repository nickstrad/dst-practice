// Decision tables exercise each single-condition guard and its boundary.
package tokenbucket_test

import (
	"math"
	"testing"
	"time"

	"dstpractice/projects/tokenbucket"
	"dstpractice/sim/clock"
)

var decisionStart = time.Unix(0, 0)

// I7, I8: each policy guard rejects its excluded values and accepts the
// adjacent supported values. The quoted conditions are the source guards.
func TestPolicyValidationDecisions(t *testing.T) {
	t.Run("Every guard: policy.Every <= 0", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			every time.Duration
			panic bool
		}{
			{name: "negative", every: -time.Nanosecond, panic: true},
			{name: "zero", every: 0, panic: true},
			{name: "positive", every: time.Nanosecond},
		} {
			t.Run(tc.name, func(t *testing.T) {
				assertNewPanics(t, tokenbucket.Policy{Every: tc.every, Burst: 1}, tc.panic)
			})
		}
	})

	t.Run("Burst guard: policy.Burst < 1", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			burst int
			panic bool
		}{
			{name: "negative", burst: -1, panic: true},
			{name: "zero", burst: 0, panic: true},
			{name: "positive", burst: 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				assertNewPanics(t, tokenbucket.Policy{Every: time.Nanosecond, Burst: tc.burst}, tc.panic)
			})
		}
	})

	t.Run("capacity guard: int64(policy.Burst) > math.MaxInt64/int64(policy.Every)", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			p     tokenbucket.Policy
			panic bool
		}{
			{name: "largest exact capacity", p: tokenbucket.Policy{Every: time.Duration(math.MaxInt64), Burst: 1}},
			{name: "one nanosecond beyond capacity", p: tokenbucket.Policy{Every: time.Duration(math.MaxInt64/2 + 1), Burst: 2}, panic: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				assertNewPanics(t, tc.p, tc.panic)
			})
		}
	})
}

func assertNewPanics(t *testing.T, policy tokenbucket.Policy, wantPanic bool) {
	t.Helper()
	panicked := false
	func() {
		defer func() {
			panicked = recover() != nil
		}()
		tokenbucket.New(policy, clock.NewFake(decisionStart))
	}()
	if panicked != wantPanic {
		t.Fatalf("New(%+v) panic = %t, want %t", policy, panicked, wantPanic)
	}
}

// I1-I3: the source cap decision is `if elapsed >= room`. After draining
// the capped credit, a 9ns retry must still fail; leaked 1ns surplus in the
// above-room row would make that retry succeed. Below room, retained credit
// makes the same retry succeed.
func TestRefillCapDecisions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		elapsed   time.Duration
		want      []bool
		afterNine bool
	}{
		{name: "below available room", elapsed: 19 * time.Nanosecond, want: []bool{true, false, false}, afterNine: true},
		{name: "equal to available room", elapsed: 20 * time.Nanosecond, want: []bool{true, true, false}},
		{name: "above available room", elapsed: 21 * time.Nanosecond, want: []bool{true, true, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewFake(decisionStart)
			b := tokenbucket.New(tokenbucket.Policy{Every: 10 * time.Nanosecond, Burst: 2}, clk)
			if !b.Allow() || !b.Allow() || b.Allow() {
				t.Fatal("could not establish an empty bucket")
			}
			clk.Advance(tc.elapsed)
			for i, want := range tc.want {
				if got := b.Allow(); got != want {
					t.Fatalf("call %d after %v refill = %t, want %t", i+1, tc.elapsed, got, want)
				}
			}
			clk.Advance(9 * time.Nanosecond)
			if got := b.Allow(); got != tc.afterNine {
				t.Fatalf("Allow after another 9ns = %t, want %t", got, tc.afterNine)
			}
		})
	}
}

// I2: the source admission decision is `if b.credit < b.every`.
// Equality admits; one unit on either side shows the branch boundary and
// the residual credit through a later public Allow call.
func TestAdmissionDecisions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		elapsed   time.Duration
		first     bool
		afterNine bool
	}{
		{name: "credit below token cost", elapsed: 9 * time.Nanosecond, first: false, afterNine: true},
		{name: "credit equal to token cost", elapsed: 10 * time.Nanosecond, first: true, afterNine: false},
		{name: "credit above token cost", elapsed: 11 * time.Nanosecond, first: true, afterNine: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewFake(decisionStart)
			b := tokenbucket.New(tokenbucket.Policy{Every: 10 * time.Nanosecond, Burst: 2}, clk)
			if !b.Allow() || !b.Allow() {
				t.Fatal("could not spend the initial burst")
			}
			clk.Advance(tc.elapsed)
			if got := b.Allow(); got != tc.first {
				t.Fatalf("Allow with %v refill = %t, want %t", tc.elapsed, got, tc.first)
			}
			clk.Advance(9 * time.Nanosecond)
			if got := b.Allow(); got != tc.afterNine {
				t.Fatalf("Allow after another 9ns = %t, want %t", got, tc.afterNine)
			}
		})
	}
}

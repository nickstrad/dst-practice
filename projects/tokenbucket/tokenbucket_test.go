// Spec tests use fixed clock times to show each promised behavior.
package tokenbucket_test

import (
	"math"
	"math/big"
	"testing"
	"time"

	"dstpractice/projects/tokenbucket"
	"dstpractice/sim/clock"
)

var start = time.Unix(0, 0)

// I1: construction provides the whole burst at one instant.
func TestStartsFull(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}, clk)
	for i, want := range []bool{true, true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("call %d at construction = %t, want %t", i+1, got, want)
		}
	}
}

// I2: a token appears at the exact refill boundary, not one nanosecond before.
func TestRefillAtBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		at   time.Duration
		want bool
	}{
		{"one nanosecond before", 100*time.Millisecond - time.Nanosecond, false},
		{"at boundary", 100 * time.Millisecond, true},
		{"one nanosecond after", 100*time.Millisecond + time.Nanosecond, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewFake(start)
			b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 1}, clk)
			if !b.Allow() {
				t.Fatal("initial token denied")
			}
			clk.Advance(tc.at)
			if got := b.Allow(); got != tc.want {
				t.Errorf("Allow at %v = %t, want %t", tc.at, got, tc.want)
			}
		})
	}
}

// I2: denied calls account for elapsed time without throwing away a fraction.
func TestDeniedCallsPreserveFractionalRefill(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}, clk)
	for _, want := range []bool{true, true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("Allow at start = %t, want %t", got, want)
		}
	}
	clk.Advance(40 * time.Millisecond)
	if b.Allow() {
		t.Fatal("40ms after depletion was admitted")
	}
	clk.Advance(59 * time.Millisecond)
	if b.Allow() {
		t.Fatal("99ms after depletion was admitted")
	}
	clk.Advance(time.Millisecond)
	if !b.Allow() {
		t.Fatal("100ms after depletion was denied")
	}
}

// I2: an admission spends one whole token, including after fractional refill.
func TestAdmissionSpendsOneToken(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}, clk)
	for call := 1; call <= 2; call++ {
		if !b.Allow() {
			t.Fatalf("initial burst call %d was denied", call)
		}
	}
	clk.Advance(150 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("first token after 150ms was denied")
	}
	if b.Allow() {
		t.Fatal("same-time call spent fractional credit")
	}
	clk.Advance(50 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("remaining 50ms did not complete a token")
	}
}

// I3: a long idle period restores at most Burst tokens.
func TestIdleRefillCapsAtBurst(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}, clk)
	for call, want := range []bool{true, true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("initial burst call %d = %t, want %t", call+1, got, want)
		}
	}
	clk.Advance(time.Second)
	for i, want := range []bool{true, true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("call %d after idle = %t, want %t", i+1, got, want)
		}
	}
}

// I3: time spent while full cannot be used after the full burst is consumed.
func TestFullBucketDiscardsIdleCredit(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}, clk)
	clk.Advance(time.Second)
	for call, want := range []bool{true, true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("call %d after idle = %t, want %t", call+1, got, want)
		}
	}
	clk.Advance(time.Nanosecond)
	if b.Allow() {
		t.Fatal("one nanosecond recovered discarded idle credit")
	}
}

// I4: tied endpoints count in a closed interval's burst-plus-refill bound.
func TestAdmissionEnvelope(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}, clk)
	admitted := 0
	for i := 0; i < 3; i++ {
		if b.Allow() {
			admitted++
		}
	}
	if admitted != 2 { // d=0: Burst + floor(0/Every) = 2.
		t.Fatalf("closed interval [0,0] admitted %d, want 2", admitted)
	}
	clk.Advance(100 * time.Millisecond)
	for i := 0; i < 3; i++ {
		if b.Allow() {
			admitted++
		}
	}
	if admitted != 3 { // d=100ms: Burst + floor(d/Every) = 3.
		t.Fatalf("closed interval [0,100ms] admitted %d, want 3", admitted)
	}
}

// I5: after denial, a quiet interval of Every permits the next attempt.
func TestDeniedCallerProgressAfterQuietPeriod(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 1}, clk)
	for call, want := range []bool{true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("initial call %d = %t, want %t", call+1, got, want)
		}
	}
	clk.Advance(100 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("denied caller could not retry after one quiet interval")
	}
}

type observedClock struct {
	now    time.Time
	reads  int
	sleeps int
}

func (c *observedClock) Now() time.Time {
	c.reads++
	return c.now
}

func (c *observedClock) Sleep(time.Duration) { c.sleeps++ }

// I6: Allow neither sleeps nor advances injected time.
func TestAllowDoesNotSleepOrAdvanceTime(t *testing.T) {
	clk := &observedClock{now: start}
	b := tokenbucket.New(tokenbucket.Policy{Every: time.Second, Burst: 1}, clk)
	for i := 0; i < 2; i++ {
		before := clk.now
		b.Allow()
		if clk.now != before || clk.sleeps != 0 {
			t.Fatalf("call %d: now=%v, sleeps=%d; want unchanged time and no sleep", i+1, clk.now, clk.sleeps)
		}
	}
}

// I7: every excluded policy panics before the supplied clock is read.
func TestInvalidPolicyPanics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy tokenbucket.Policy
	}{
		{"zero Every", tokenbucket.Policy{Every: 0, Burst: 1}},
		{"negative Every", tokenbucket.Policy{Every: -1, Burst: 1}},
		{"zero Burst", tokenbucket.Policy{Every: 1, Burst: 0}},
		{"negative Burst", tokenbucket.Policy{Every: 1, Burst: -1}},
		{"capacity overflow", tokenbucket.Policy{Every: time.Duration(math.MaxInt64), Burst: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := &observedClock{now: start}
			func() {
				defer func() {
					if recover() == nil {
						t.Error("New did not panic")
					}
				}()
				tokenbucket.New(tc.policy, clk)
			}()
			if clk.reads != 0 || clk.sleeps != 0 {
				t.Errorf("invalid policy used clock: reads=%d, sleeps=%d", clk.reads, clk.sleeps)
			}
		})
	}
}

// I7, I8: the largest representable capacity is accepted without overflow.
func TestLargestValidCapacity(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: time.Duration(math.MaxInt64), Burst: 1}, clk)
	for call, want := range []bool{true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("initial call %d = %t, want %t", call+1, got, want)
		}
	}
	clk.Advance(time.Duration(math.MaxInt64) - time.Nanosecond)
	if b.Allow() {
		t.Fatal("MaxInt64-nanosecond refill completed one nanosecond early")
	}
	clk.Advance(time.Nanosecond)
	if !b.Allow() {
		t.Fatal("MaxInt64-nanosecond refill did not complete at its boundary")
	}
	// A large Burst needs no draining to show that validation accepts its edge.
	burstLimit := int64(math.MaxInt64 / 3)
	maxInt := int64(^uint(0) >> 1)
	if burstLimit > maxInt {
		burstLimit = maxInt
	}
	b = tokenbucket.New(tokenbucket.Policy{Every: 3, Burst: int(burstLimit)}, clk)
	if !b.Allow() {
		t.Fatal("largest Burst for Every=3ns was rejected")
	}
}

// I8: nanosecond refill retains the exact smallest positive interval.
func TestNanosecondRefill(t *testing.T) {
	clk := clock.NewFake(start)
	b := tokenbucket.New(tokenbucket.Policy{Every: time.Nanosecond, Burst: 1}, clk)
	for call, want := range []bool{true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("initial call %d = %t, want %t", call+1, got, want)
		}
	}
	clk.Advance(time.Nanosecond)
	for call, want := range []bool{true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("call %d after one nanosecond = %t, want %t", call+1, got, want)
		}
	}
}

// I8: a gap beyond time.Duration's range still fills a valid capacity.
func TestLargeIdleGapSaturates(t *testing.T) {
	first := time.Date(1700, time.January, 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(2300, time.January, 1, 0, 0, 0, 0, time.UTC)
	if last.Sub(first) != time.Duration(math.MaxInt64) {
		t.Fatal("test times do not exceed time.Duration's range")
	}
	clk := &observedClock{now: first}
	b := tokenbucket.New(tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}, clk)
	for call, want := range []bool{true, true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("initial burst call %d = %t, want %t", call+1, got, want)
		}
	}
	clk.now = last
	for call, want := range []bool{true, true, false} {
		if got := b.Allow(); got != want {
			t.Fatalf("call %d after large idle gap = %t, want %t", call+1, got, want)
		}
	}
}

// I1, I6: the finish-time model starts with exactly the full burst.
func TestReferenceModelInitialBurst(t *testing.T) {
	p := tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}
	events := []event{{Call: true}, {Call: true}, {Call: true}}
	checkModelExample(t, "initial burst", p, events, []bool{true, true, false})
}

// I2, I6: fractional elapsed time accumulates to exact boundaries.
func TestReferenceModelFractionalRefill(t *testing.T) {
	p := tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 1}
	events := []event{
		{Call: true},
		{Advance: 25 * time.Millisecond, Call: true},
		{Advance: 50 * time.Millisecond, Call: true},
		{Advance: 25 * time.Millisecond, Call: true},
		{Advance: 50 * time.Millisecond, Call: true},
		{Advance: 50 * time.Millisecond, Call: true},
	}
	checkModelExample(t, "fractional refill", p, events,
		[]bool{true, false, false, true, false, true})
}

// I2, I5, I6: a denial leaves the model's debt unchanged until admission.
func TestReferenceModelDenialPreservesDebt(t *testing.T) {
	p := tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 1}
	events := []event{
		{Call: true},
		{Advance: 60 * time.Millisecond, Call: true},
		{Advance: 39 * time.Millisecond, Call: true},
		{Advance: time.Millisecond, Call: true},
		{Call: true},
		{Advance: 100 * time.Millisecond, Call: true},
	}
	checkModelExample(t, "denial preserves debt", p, events,
		[]bool{true, false, false, true, false, true})
}

// I3, I6: a long idle period restores the burst and discards the surplus.
func TestReferenceModelIdleSaturation(t *testing.T) {
	p := tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}
	events := []event{
		{Call: true},
		{Advance: time.Second, Call: true},
		{Call: true},
		{Call: true},
		{Advance: time.Nanosecond, Call: true},
	}
	checkModelExample(t, "idle saturation", p, events,
		[]bool{true, true, true, false, false})
}

// I4, I6: all admissions tied at either endpoint belong to the window.
func TestReferenceModelTiedWindowEndpoints(t *testing.T) {
	p := tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 2}
	events := []event{
		{Call: true}, {Call: true}, {Call: true},
		{Advance: 100 * time.Millisecond, Call: true}, {Call: true},
		{Advance: 100 * time.Millisecond, Call: true}, {Call: true},
	}
	checkModelExample(t, "tied endpoints", p, events,
		[]bool{true, true, false, true, false, true, false})

	// A hypothetical fourth admission in [0,100ms] exceeds 2+floor(100/100).
	groups := []admissionGroup{
		{at: big.NewInt(0), count: 2},
		{at: big.NewInt(int64(100 * time.Millisecond)), count: 2},
	}
	if envelopeViolation(groups, p) == "" {
		t.Fatal("closed-window checker missed four admissions with tied endpoints")
	}
	if envelopeViolation([]admissionGroup{{at: big.NewInt(0), count: 3}}, p) == "" {
		t.Fatal("zero-length checker missed three tied admissions")
	}
}

// I2, I5, I6: equality admits, and a denied caller progresses after quiet time.
func TestReferenceModelBoundaryEqualityAndProgress(t *testing.T) {
	p := tokenbucket.Policy{Every: 100 * time.Millisecond, Burst: 1}
	events := []event{
		{Call: true}, {Call: true},
		{Advance: 99 * time.Millisecond, Call: true},
		{Advance: time.Millisecond, Call: true},
		{Call: true},
		{Advance: 100 * time.Millisecond, Call: true},
	}
	checkModelExample(t, "boundary and progress", p, events,
		[]bool{true, false, false, true, false, true})
}

// Shared test helpers use a finish-time model and check whole traces.
package tokenbucket_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"dstpractice/projects/tokenbucket"
	"dstpractice/sim/clock"
)

// Advance is nonnegative and occurs before the optional call.
// Call=false represents idle time alone. Actor is a diagnostic label.
type event struct {
	Advance time.Duration
	Call    bool
	Actor   int
}

// finishModel represents admitted work as a virtual finish time. Its state
// is debt in nanoseconds since construction, not available bucket credit.
type finishModel struct {
	finish   big.Int
	every    big.Int
	capacity big.Int
}

func newFinishModel(p tokenbucket.Policy) *finishModel {
	m := &finishModel{}
	m.every.SetInt64(int64(p.Every))
	m.capacity.Mul(big.NewInt(int64(p.Burst)), &m.every)
	return m
}

func (m *finishModel) allow(at *big.Int) bool {
	base := at
	if m.finish.Cmp(at) > 0 {
		base = &m.finish
	}
	// Fresh destinations keep the model's finish and the caller's time
	// unchanged when an attempted request is denied.
	candidate := new(big.Int).Add(base, &m.every)
	limit := new(big.Int).Add(at, &m.capacity)
	if candidate.Cmp(limit) > 0 {
		return false
	}
	m.finish.Set(candidate)
	return true
}

type admissionGroup struct {
	at    *big.Int
	count int
}

// envelopeViolation checks windows ending at the latest admitted time.
// Earlier windows were checked when their last admission arrived. Each group
// includes all calls tied at its time, including zero-length windows.
func envelopeViolation(groups []admissionGroup, p tokenbucket.Policy) string {
	if len(groups) == 0 {
		return ""
	}
	every := big.NewInt(int64(p.Every))
	last := len(groups) - 1
	count := 0
	for first := last; first >= 0; first-- {
		count += groups[first].count
		span := new(big.Int).Sub(groups[last].at, groups[first].at)
		bound := new(big.Int).Quo(span, every)
		bound.Add(bound, big.NewInt(int64(p.Burst)))
		if big.NewInt(int64(count)).Cmp(bound) > 0 {
			return fmt.Sprintf("[%s,%s] contains %d admissions; bound is %s", groups[first].at, groups[last].at, count, bound)
		}
	}
	return ""
}

// checkRun compares every decision with the independent model, checks all
// closed admission windows, then witnesses progress after a quiet interval.
func checkRun(t *testing.T, label string, p tokenbucket.Policy, events []event) {
	t.Helper()
	if len(events) > 128 || p.Every <= 0 || p.Every > time.Second || p.Burst < 1 || p.Burst > 16 {
		t.Fatalf("%s: unbounded test input: policy=%+v events=%d", label, p, len(events))
	}

	clk := clock.NewFake(start)
	b := tokenbucket.New(p, clk)
	model := newFinishModel(p)
	var at big.Int
	groups := make([]admissionGroup, 0)
	trace := make([]event, 0, len(events)+p.Burst+2)

	apply := func(e event) bool {
		if e.Advance < 0 {
			t.Fatalf("%s: negative advance at event %d: policy=%+v prefix=%v", label, len(trace), p, append(trace, e))
		}
		trace = append(trace, e)
		at.Add(&at, big.NewInt(int64(e.Advance)))
		clk.Advance(e.Advance)
		if !e.Call {
			return false
		}

		before := clk.Now()
		sleepCount := len(clk.Sleeps)
		want := model.allow(&at)
		got := b.Allow()
		// I6: Allow cannot move the injected clock or ask it to sleep.
		if !clk.Now().Equal(before) || len(clk.Sleeps) != sleepCount {
			t.Fatalf("%s: Allow changed clock at event %d: policy=%+v prefix=%v", label, len(trace)-1, p, trace)
		}
		// I1-I3: the debt model decides each admission independently.
		if got != want {
			t.Fatalf("%s: event %d actor=%d policy=%+v prefix=%v: Allow=%t, model=%t", label, len(trace)-1, e.Actor, p, trace, got, want)
		}
		if got {
			if len(groups) > 0 && groups[len(groups)-1].at.Cmp(&at) == 0 {
				groups[len(groups)-1].count++
			} else {
				groups = append(groups, admissionGroup{at: new(big.Int).Set(&at), count: 1})
			}
			// I4: include all admissions tied at either closed endpoint.
			if violation := envelopeViolation(groups, p); violation != "" {
				t.Fatalf("%s: event %d actor=%d policy=%+v prefix=%v: envelope %s", label, len(trace)-1, e.Actor, p, trace, violation)
			}
		}
		return got
	}

	for _, e := range events {
		apply(e)
	}
	// No more than Burst calls at one instant can leave any credit to spend.
	for i := 0; i < p.Burst; i++ {
		apply(event{Call: true, Actor: -1})
	}
	// I1 and I3: a burst of same-time calls must empty any remaining credit.
	if apply(event{Call: true, Actor: -1}) {
		t.Fatalf("%s: draining did not reach denial: policy=%+v prefix=%v", label, p, trace)
	}
	// I5: one quiet refill interval lets the denied caller through.
	if !apply(event{Advance: p.Every, Call: true, Actor: -1}) {
		t.Fatalf("%s: denied caller did not progress after Every: policy=%+v prefix=%v", label, p, trace)
	}
}

// checkModelExample checks hand-calculated decisions before comparing the
// same trace with the bucket. That order keeps the oracle accountable.
func checkModelExample(t *testing.T, label string, p tokenbucket.Policy, events []event, want []bool) {
	t.Helper()
	model := newFinishModel(p)
	var at big.Int
	call := 0
	for i, e := range events {
		at.Add(&at, big.NewInt(int64(e.Advance)))
		if !e.Call {
			continue
		}
		if call >= len(want) {
			t.Fatalf("%s: event %d has no expected result", label, i)
		}
		if got := model.allow(&at); got != want[call] {
			t.Fatalf("%s: model call %d at %s = %t, want %t", label, call, &at, got, want[call])
		}
		call++
	}
	if call != len(want) {
		t.Fatalf("%s: got %d calls, want %d results", label, call, len(want))
	}
	checkRun(t, label, p, events)
}

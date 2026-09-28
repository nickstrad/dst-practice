// Deterministic simulation tests replay seeded requests against the fake clock.
package tokenbucket_test

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"dstpractice/projects/tokenbucket"
)

var (
	seedFlag = flag.Uint64("seed", 0, "run one seed; zero runs seeds 1 through -runs")
	runsFlag = flag.Int("runs", 1000, "number of deterministic seeds when -seed=0")
)

// I1-I6: each seeded trace checks decisions, the closed admission bound,
// injected-clock behavior, and progress after competition stops.
func TestInvariantsAcrossSeeds(t *testing.T) {
	// Regenerating a trace checks the replay contract without a brittle
	// golden sequence tied to the random library's exact output.
	p1, events1 := generatedTrace(42)
	p2, events2 := generatedTrace(42)
	if p1 != p2 || !slices.Equal(events1, events2) {
		t.Fatal("seed=42 did not reproduce the same policy and event trace")
	}

	if *seedFlag != 0 {
		checkSeed(t, *seedFlag)
		return
	}
	if *runsFlag <= 0 {
		t.Fatalf("-runs must be positive when -seed=0; got %d", *runsFlag)
	}
	for seed := 1; seed <= *runsFlag; seed++ {
		checkSeed(t, uint64(seed))
	}
}

func checkSeed(t *testing.T, seed uint64) {
	t.Helper()
	p, events := generatedTrace(seed)
	label := fmt.Sprintf("seed=%d policy=%+v replay: go test ./projects/tokenbucket -run '^TestInvariantsAcrossSeeds$' -seed %d -count=1 -v", seed, p, seed)
	checkRun(t, label, p, events)
}

func generatedTrace(seed uint64) (tokenbucket.Policy, []event) {
	rng := rand.New(rand.NewPCG(seed, 0))
	p := tokenbucket.Policy{
		Every: time.Duration(1 + rng.Int64N(int64(time.Second))),
		Burst: 1 + rng.IntN(16),
	}
	// The fixed shapes are present in every seed. Actor labels describe
	// serialized competitors; no actor has a fairness promise.
	events := make([]event, 0, 128)
	for i := 0; i < p.Burst+2; i++ {
		events = append(events, event{Call: true, Actor: i % 4})
	}
	// The competitor takes each new token before the other actor retries.
	// This tests repeated denial under competition, not caller fairness.
	for i := 0; i < 3; i++ {
		events = append(events,
			event{Advance: p.Every, Call: true, Actor: 1},
			event{Call: true, Actor: 0},
		)
	}
	events = append(events,
		event{Advance: p.Every - time.Nanosecond, Call: true, Actor: 0},
		event{Advance: time.Nanosecond, Call: true, Actor: 0},
		event{Advance: p.Every * time.Duration(p.Burst+1)}, // idle beyond capacity
		event{Call: true, Actor: 1},
		event{Call: true, Actor: 2},
	)

	for n := 48 + rng.IntN(33); n > 0; n-- {
		actor := rng.IntN(4)
		switch rng.IntN(8) {
		case 0: // another request at the same time
			events = append(events, event{Call: true, Actor: actor})
		case 1: // less than one refill interval
			events = append(events, event{Advance: time.Duration(rng.Int64N(int64(p.Every))), Call: true, Actor: actor})
		case 2: // equality at one refill boundary
			events = append(events, event{Advance: p.Every, Call: true, Actor: actor})
		case 3: // idle time beyond a full bucket
			events = append(events, event{Advance: p.Every * time.Duration(p.Burst+1), Actor: actor})
		case 4: // idle time without a caller
			events = append(events, event{Advance: p.Every / 2, Actor: actor})
		case 5: // just past the boundary
			events = append(events, event{Advance: p.Every + time.Nanosecond, Call: true, Actor: actor})
		case 6: // smallest clock movement
			events = append(events, event{Advance: time.Nanosecond, Call: true, Actor: actor})
		case 7: // a request may or may not follow this interval
			events = append(events, event{Advance: time.Duration(rng.Int64N(int64(p.Every) + 1)), Call: rng.IntN(3) != 0, Actor: actor})
		}
	}
	return p, events
}

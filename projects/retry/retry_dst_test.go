package retry_test

// Deterministic simulation test. Many seeded runs against the fake clock,
// each checked against the invariants in invariants.md. A failure prints
// the seed so one run can be replayed.

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"dstpractice/retry"
)

// Flags for the seeded test.
//
//	go test ./retry -run TestInvariantsAcrossSeeds -seed 42   # replay one seed
//	go test ./retry -runs 10000                                # try more seeds
var (
	seedFlag = flag.Uint64("seed", 0, "run only this seed (0 means many random seeds)")
	runsFlag = flag.Int("runs", 1000, "how many random seeds to try when -seed is not set")
)

// TestInvariantsAcrossSeeds checks I1 through I6 across many seeds with
// real jitter and a randomized policy.
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
	failures := rng.IntN(30)

	// Every failure message starts with enough to replay this run.
	label := fmt.Sprintf("seed=%d policy=%+v failures=%d", seed, p, failures)
	checkRun(t, label, p, failures, rng)
}

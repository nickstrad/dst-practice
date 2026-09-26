// Command retrydemo shows the retrier running against the real clock with
// an operation that fails a few times before succeeding.
//
//	go run ./cmd/retrydemo
package main

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"dstpractice/retry"
	"dstpractice/sim/clock"
)

func main() {
	policy := retry.Policy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2,
		MaxAttempts:  6,
		Timeout:      10 * time.Second,
	}

	// Production wiring: the real clock and a randomly seeded PRNG.
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	r := retry.New(policy, clock.Real{}, rng)

	start := time.Now()
	attempt := 0
	err := r.Do(func() error {
		attempt++
		fmt.Printf("t=%-8v attempt %d\n", time.Since(start).Round(time.Millisecond), attempt)
		if attempt < 4 {
			return errors.New("service unavailable")
		}
		return nil
	})

	if err != nil {
		fmt.Println("failed:", err)
		return
	}
	fmt.Println("succeeded after", attempt, "attempts")
}

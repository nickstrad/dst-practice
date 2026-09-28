// Package tokenbucket admits requests as time refills a fixed burst.
//
// With Every=100ms and Burst=2, two calls at construction succeed and a
// third is denied. A call 40ms later is denied, but one at 100ms succeeds.
// Waiting longer restores at most two tokens.
package tokenbucket

import (
	"math"
	"time"

	"dstpractice/sim/clock"
)

// Policy sets the refill interval and the largest immediate burst.
type Policy struct {
	Every time.Duration // >0; one token refills per Every.
	Burst int           // >=1; Burst*Every must fit time.Duration.
}

// Bucket holds available refill time. Calls to one bucket must be serialized.
type Bucket struct {
	clock    clock.Clock
	every    time.Duration
	capacity time.Duration
	credit   time.Duration
	last     time.Time
}

// New starts full and panics for an invalid policy. The clock must be
// non-nil and must never move backward.
func New(policy Policy, clk clock.Clock) *Bucket {
	if policy.Every <= 0 {
		panic("tokenbucket: Every must be positive")
	}
	if policy.Burst < 1 {
		panic("tokenbucket: Burst must be positive")
	}
	if int64(policy.Burst) > math.MaxInt64/int64(policy.Every) {
		panic("tokenbucket: capacity exceeds time.Duration")
	}
	capacity := time.Duration(policy.Burst) * policy.Every
	return &Bucket{
		clock:    clk,
		every:    policy.Every,
		capacity: capacity,
		credit:   capacity,
		last:     clk.Now(),
	}
}

// Allow samples injected time and consumes one token when credit is enough.
// It returns immediately without sleeping.
func (b *Bucket) Allow() bool {
	now := b.clock.Now()
	elapsed := now.Sub(b.last)
	room := b.capacity - b.credit
	// Saturate before addition so even the largest supported capacity is safe.
	if elapsed >= room {
		b.credit = b.capacity
	} else {
		b.credit += elapsed
	}
	b.last = now

	if b.credit < b.every {
		return false
	}
	b.credit -= b.every
	return true
}

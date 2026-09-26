// Package clock defines the Clock interface that every system under test
// receives, plus two implementations: Real (wall clock) and Fake (virtual
// time the test controls).
package clock

import "time"

// Clock is the only way the system under test learns what time it is or
// waits for time to pass. Production code uses Real. Tests use Fake.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// Real reads the wall clock.
type Real struct{}

func (Real) Now() time.Time        { return time.Now() }
func (Real) Sleep(d time.Duration) { time.Sleep(d) }

// Fake is a virtual clock. Sleep does not block. It moves time forward by
// the requested amount and records the request so tests can inspect it.
//
// Fake is not safe for concurrent use. That is on purpose: the systems
// under test in this repo never spawn goroutines.
type Fake struct {
	now    time.Time
	Sleeps []time.Duration // every duration passed to Sleep, in order
}

// NewFake returns a fake clock whose time starts at the given instant.
func NewFake(start time.Time) *Fake {
	return &Fake{now: start}
}

func (f *Fake) Now() time.Time { return f.now }

func (f *Fake) Sleep(d time.Duration) {
	f.Sleeps = append(f.Sleeps, d)
	f.Advance(d)
}

// Advance moves the clock forward without recording a sleep. Use it to
// simulate time passing inside an operation, for example a slow RPC.
func (f *Fake) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}

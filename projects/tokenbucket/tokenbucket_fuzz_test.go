// Fuzz targets mutate raw policies and bounded request traces.
package tokenbucket_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"dstpractice/projects/tokenbucket"
	"dstpractice/sim/clock"
)

// I7: New accepts exactly the representable positive policy domain.
func FuzzPolicyValidation(f *testing.F) {
	f.Add(int64(0), 1)
	f.Add(int64(-1), 1)
	f.Add(int64(math.MinInt64), 1)
	f.Add(int64(1), 0)
	f.Add(int64(1), -1)
	f.Add(int64(1), 1)
	f.Add(int64(math.MaxInt64), 1)
	f.Add(int64(math.MaxInt64), 2)
	// For this Every, Burst=3 is the largest valid value and 4 overflows.
	f.Add(int64(math.MaxInt64/3), 3)
	f.Add(int64(math.MaxInt64/3), 4)
	f.Add(int64(1), int(^uint(0)>>1))

	f.Fuzz(func(t *testing.T, rawEvery int64, rawBurst int) {
		// Short-circuit before division; multiplying first could overflow.
		want := rawEvery > 0 && rawBurst >= 1 &&
			int64(rawBurst) <= math.MaxInt64/rawEvery
		p := tokenbucket.Policy{Every: time.Duration(rawEvery), Burst: rawBurst}
		got := policyAccepted(p)
		if got != want {
			t.Fatalf("FuzzPolicyValidation rawEvery=%d rawBurst=%d: accepted=%t, want %t", rawEvery, rawBurst, got, want)
		}
	})
}

func policyAccepted(p tokenbucket.Policy) (accepted bool) {
	defer func() {
		if recover() != nil {
			accepted = false
		}
	}()
	tokenbucket.New(p, clock.NewFake(start))
	return true
}

// I1-I6: mutated bytes choose clock advances, calls, and serialized actors.
func FuzzAllow(f *testing.F) {
	// The folds below add one, so these raw values produce Every=100ms.
	rawEvery := int64(100*time.Millisecond - time.Nanosecond)
	f.Add(rawEvery, int64(1), []byte{})                                   // empty prefix, Burst=2
	f.Add(rawEvery, int64(1), []byte{0x08, 0x18, 0x28})                   // tied calls, Burst=2
	f.Add(rawEvery, int64(0), []byte{0x08, 0x0d, 0x0d})                   // fractional denial, Burst=1
	f.Add(rawEvery, int64(1), []byte{0x08, 0x08, 0x04, 0x08, 0x08, 0x08}) // saturated idle, Burst=2
	f.Add(rawEvery, int64(0), []byte{0x08, 0x08, 0x0a})                   // quiet progress, Burst=1

	f.Fuzz(func(t *testing.T, rawEvery, rawBurst int64, data []byte) {
		// Unsigned folding handles MinInt64 without abs or negation.
		p := tokenbucket.Policy{
			Every: time.Duration(uint64(rawEvery)%uint64(time.Second)) + 1,
			Burst: int(uint64(rawBurst)%16) + 1,
		}
		// One byte is one event. Bits 0..2 choose its advance, bit 3
		// chooses whether to call, bits 4..5 name actor 0..3, and bits
		// 6..7 vary the small advances. Ignore bytes after event 128.
		used := data
		if len(used) > 128 {
			used = used[:128]
		}
		events := make([]event, 0, len(used))
		for _, b := range used {
			e := event{Call: b&8 != 0, Actor: int((b >> 4) & 3)}
			switch b & 7 {
			case 0:
				e.Advance = 0
			case 1:
				e.Advance = p.Every - time.Nanosecond
			case 2:
				e.Advance = p.Every
			case 3:
				e.Advance = p.Every + time.Nanosecond
			case 4:
				e.Advance = p.Every * time.Duration(p.Burst+1)
			case 5:
				e.Advance = p.Every / 2
			case 6:
				e.Advance = p.Every * time.Duration(b>>6) / 4
			case 7:
				e.Advance = time.Duration(b >> 6)
			}
			events = append(events, e)
		}
		// The largest decoded advance is 17s. Even 128 such events total
		// under 37 minutes, far below time.Duration's roughly 292 years.
		label := fmt.Sprintf("FuzzAllow rawEvery=%d rawBurst=%d rawData=%x rawLen=%d policy=%+v", rawEvery, rawBurst, used, len(data), p)
		checkRun(t, label, p, events)
	})
}

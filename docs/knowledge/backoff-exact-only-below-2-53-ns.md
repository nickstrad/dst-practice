# Backoff is exact only below 2^53 nanoseconds

## Context
Writing `FuzzBackoff`. A property test on "Backoff(p, 1) equals
InitialDelay" fails for delays near `math.MaxInt64`.

## Learning
`retry.Backoff` multiplies in float64. float64 holds every integer only
up to 2^53, so a `time.Duration` above that, about 104 days, rounds when
converted to float and back. `InitialDelay` of `MaxInt64-1` comes back as
`MaxInt64`. Any Go code that round-trips a `time.Duration` through
float64 has the same limit.

## Why it matters
A fuzz target that feeds raw int64 durations will report this as a bug.
It is a documented limit, not a defect: no retry waits 104 days.

## How to apply
Fold fuzz inputs into `[1, 2^53)` with a named constant and say why in a
comment, as `retry_fuzz_test.go` does with `maxExact`. List the limit
under `Not promised` in the package's `invariants.md`. If a future
system needs exact huge durations, compute in integers instead.

## Evidence
`Backoff(1)` returned 9223372036854775807 for an InitialDelay of
9223372036854775806 on 2026-09-25.

## Related
`projects/retry/invariants.md` Not promised,
`projects/retry/retry_fuzz_test.go`.

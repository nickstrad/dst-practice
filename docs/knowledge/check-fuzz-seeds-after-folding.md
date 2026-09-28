# Check fuzz seeds after folding

## Context

The token bucket fuzz target folds raw policy values into a small valid
domain. Its initial fixed seeds used the intended policy values as raw
inputs, then the fold added one to each value.

## Learning

A fixed seed's label must describe the decoded input, not its raw value.
With `Burst = rawBurst%16 + 1`, raw `1` creates a two-token bucket.
The intended fractional-denial example then admitted its second call,
so a passing corpus did not prove that it exercised the named case.

## Why it matters

Input normalization can silently remove the boundary a seed was meant
to test. Successful fuzz runs do not establish that corpus labels are
accurate.

## How to apply

Walk each fixed seed through the decoder before naming its behavior.
For a value inside a modulo range followed by `+1`, use the desired
value minus one as the raw seed. Check the resulting event sequence too.

## Evidence

Review of `FuzzAllow` on 2026-09-28 caught this in the fractional-denial
seed. Its corrected raw burst is zero, giving `Burst=1`. With
`Every=100ms`, calls at 0ms, 50ms, and 100ms admit, deny, then admit.
Replay the fixed corpus with:

    go test ./projects/tokenbucket -run '^FuzzAllow$' -count=1 -v

## Related

- [Token bucket fuzz targets](../../projects/tokenbucket/tokenbucket_fuzz_test.go).

# NaN passes a less-than validation check

## Context
Adding fuzz and decision-table tests to `projects/retry/`. A seed input
with `Multiplier: math.NaN()` made `retry.New` accept the policy.

## Learning
Every comparison with NaN is false. A guard written as `if x < 1 { reject }`
lets NaN through, and the code then computes with it. A NaN backoff
converts to an implementation-defined integer, which was 0 on this
machine.

## Why it matters
Any float field validated with a single comparison has this hole. The
seeded DST test never draws NaN, so only a deliberate seed or the fuzzer
finds it.

## How to apply
Validate floats with the comparison plus `math.IsNaN`, or write the guard
as `!(x >= 1)` when the style allows. Add a NaN row to the decision table
and a NaN seed to the fuzz corpus for every float field.

## Evidence
`FuzzPolicyValidation` seed 2 and `TestPolicyValidationDecisions/Multiplier_NaN`
in `projects/retry/`, fixed in `Policy.validate` on 2026-09-25.

## Related
`docs/testing.md`, `projects/retry/invariants.md` I9.

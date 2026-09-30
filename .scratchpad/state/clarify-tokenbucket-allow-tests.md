# Clarify token bucket Allow tests

Shape: state only

## Goal

Restore the full initial burst in TestAdmissionSpendsOneToken and replace compound Allow assertions with visible call sequences.

## Current state

The user clarified that the uncommitted one-call edit should be fixed. TestAdmissionSpendsOneToken now drains both initial tokens with an explicit loop. Other compound Allow conditions now use visible call sequences. The focused test, make test, make vet, and git diff --check pass.

## Next step

No work remains. The user can delete this state file when it is no longer useful.

## Open questions

None.

## Reflection

Each Allow call changes bucket credit. A short-circuit expression can hide
which call spent a token. Indexed sequences show the order and report the
call that differs from the expected result. The one-call setup left a
whole token in the bucket, so the same-time call succeeded. No new
knowledge entry is needed because the token bucket invariant and the
existing test already explain the credit rule.

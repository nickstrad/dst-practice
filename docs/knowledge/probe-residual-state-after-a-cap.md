# Probe residual state after a cap

## Context

The token bucket refill table tested 19ns, 20ns, and 21ns of refill in an
empty bucket with a 20ns capacity and 10ns token cost.

## Learning

Immediate outcomes can hide incorrect residual state. Both a correctly
capped 20ns balance and an uncapped 21ns balance admit two calls and deny
a third. A later 9ns advance makes the states observably different:
the correct bucket denies, while the bucket retaining 1ns admits.

## Why it matters

A boundary row can execute a branch without detecting its defect.
Statement coverage alone cannot show that a test observes the branch's
effect on later behavior.

## How to apply

After testing a cap, choose a later input that turns any discarded surplus
into a different public result. Keep the check outside private state so
the test remains about the contract.

## Evidence

Review on 2026-09-28 added this observation to
`TestRefillCapDecisions/above_available_room`. The original row could not
distinguish the two balances. No production defect was found.

    go test ./projects/tokenbucket -run '^TestRefillCapDecisions$' -count=1 -v

## Related

- [Token bucket decision tables](../../projects/tokenbucket/tokenbucket_mcdc_test.go).

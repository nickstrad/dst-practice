# Illustrate invariants

Shape: state only

## Goal

Remove test-name lists from invariant statements. Add a small text-art example for each invariant and update the repo guidance.

## Current state

The documentation changes were committed as 3c1f8c2. The user then asked to fix the token bucket test failure before pushing. TestRefillCapDecisions had tried to empty a Burst=2 bucket with one successful call. Its setup now spends both tokens and checks that a third call is denied. The focused test, make test, and make vet all pass with GOCACHE under /private/tmp.

## Next step

Commit the test fix and updated state, then push both commits on main.

## Open questions

None.

## Reflection

The deadline wording was non-obvious: Do can skip a late sleep but cannot
interrupt an operation. The old invariant overstated that guarantee. See
[A retry deadline cannot interrupt an operation](../../docs/knowledge/retry-deadline-cannot-interrupt-op.md).

The first verification failed because the sandbox could not read the
default Go cache. Set GOCACHE=/private/tmp/dst-practice-go-cache to replay.
With that cache, make vet passed. make test exposed a token bucket test
setup failure in TestRefillCapDecisions. A Burst=2 bucket needs two
successful calls before a denial can prove it is empty.

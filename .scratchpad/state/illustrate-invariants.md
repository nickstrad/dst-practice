# Illustrate invariants

Shape: state only

## Goal

Remove test-name lists from invariant statements. Add a small text-art example for each invariant and update the repo guidance.

## Current state

Updated both project invariant files with one text example per ID. Updated AGENTS.md, sim/AGENTS.md, docs/testing.md, and the clock architecture. Corrected retry I4 and recorded the finding in docs/knowledge/retry-deadline-cannot-interrupt-op.md. With GOCACHE under /private/tmp, make vet passes and retry tests pass. make test fails in the unchanged TestRefillCapDecisions: its setup expects a Burst=2 bucket to be empty after one call.

## Next step

Commit and push the documentation changes on main. No further edits are planned.

## Open questions

None.

## Reflection

The deadline wording was non-obvious: Do can skip a late sleep but cannot
interrupt an operation. The old invariant overstated that guarantee. See
[A retry deadline cannot interrupt an operation](../../docs/knowledge/retry-deadline-cannot-interrupt-op.md).

The first verification failed because the sandbox could not read the
default Go cache. Set GOCACHE=/private/tmp/dst-practice-go-cache to replay.
With that cache, make vet passed. make test exposed an existing token bucket
test setup failure in TestRefillCapDecisions; no Go files changed.

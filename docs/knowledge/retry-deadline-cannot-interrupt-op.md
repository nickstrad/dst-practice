# A retry deadline cannot interrupt an operation

## Context

While adding examples to retry invariants, I checked what `Do` does when an
operation uses time from the injected clock.

## Learning

`Do` checks the deadline after an operation returns. It can skip a sleep
that would wake too late, but it cannot stop an operation that runs past
the deadline.

## Why it matters

An invariant that says `Do` always returns by the deadline promises more
than the code can enforce. A test using a slow operation could fail even
though the retrier made no late sleep.

## How to apply

State the sleep boundary in the invariant. List operation interruption
under `Not promised`. Test an operation that consumes part of the time
budget before the retrier chooses a sleep.

## Evidence

`projects/retry/retry.go` checks `wakeUp.After(deadline)` after `op()`.
`projects/retry/invariants.md` I4 shows a 60ms operation and a skipped
50ms sleep before a 100ms deadline.

# retry invariants

The retrier runs an operation until it succeeds, sleeping between attempts
with capped exponential backoff and full jitter. It never runs the
operation more often than the policy allows, never sleeps past the
deadline, and never reads the wall clock or spawns a goroutine.

## Invariants

- I1. `Do` calls the operation at most `MaxAttempts` times. Checked by
  `TestGivesUpAfterMaxAttempts`, `TestMaxAttemptsDecision`,
  `TestInvariantsAcrossSeeds`, `FuzzDo`.
- I2. `Do` sleeps exactly once between consecutive attempts and never
  after the last one. Checked by `TestSucceedsFirstTry`,
  `TestGivesUpAfterMaxAttempts`, `TestInvariantsAcrossSeeds`, `FuzzDo`.
- I3. The sleep after failed attempt n lies in `[0, Backoff(p, n))`.
  Checked by `TestSleepsFollowSchedule`, `TestInvariantsAcrossSeeds`,
  `FuzzDo`.
- I4. With a `Timeout`, the clock never passes start plus `Timeout` when
  `Do` returns, and time spent inside the operation counts. Checked by
  `TestGivesUpAtDeadline`, `TestDeadlineCountsTimeSpentInsideOp`,
  `TestDeadlineDecision`, `TestInvariantsAcrossSeeds`, `FuzzDo`.
- I5. `Do` returns nil exactly when some attempt returned nil. Checked by
  `TestSucceedsFirstTry`, `TestInvariantsAcrossSeeds`, `FuzzDo`.
- I6. When `Do` gives up, the returned error wraps the operation's last
  error and reports why it stopped. Checked by
  `TestGivesUpAfterMaxAttempts`, `TestInvariantsAcrossSeeds`, `FuzzDo`.
- I7. An error wrapped with `Stop` ends `Do` at once, with no sleep, and
  `Do` returns the unwrapped error. Checked by
  `TestStopEndsRetryingImmediately`.
- I8. `Backoff(p, 1)` equals `InitialDelay`, `Backoff` never decreases as
  the attempt grows, and it never exceeds `MaxDelay`. Checked by
  `TestBackoffSchedule`, `TestBackoffCapDecision`, `FuzzBackoff`.
- I9. `New` panics exactly when the policy breaks a rule in the `Policy`
  field comments. Checked by `TestInvalidPolicyPanics`,
  `TestPolicyValidationDecisions`, `FuzzPolicyValidation`.

## Not promised

- Wall-clock accuracy. `clock.Real` may oversleep, and nothing here
  measures it.
- Exact schedules for delays of 2^53 nanoseconds or more, about 104 days.
  `Backoff` multiplies in float64, which loses integer precision there.
- Safe concurrent use of one `Retrier`, unless the `Rand` it holds is
  safe for concurrent use.

## Related

- `sim/clock/architecture.md`: the fake clock that makes I2 through I4
  checkable with exact numbers and no waiting.
- `docs/testing.md`: which test file checks what, and how to run each.

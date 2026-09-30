# retry invariants

The retrier runs an operation until it succeeds, sleeping between attempts
with capped exponential backoff and full jitter. It never runs the
operation more often than the policy allows, never sleeps past the
deadline, and never reads the wall clock or spawns a goroutine.

## Invariants

- I1. `Do` calls the operation at most `MaxAttempts` times.

  ```text
  MaxAttempts=3, all fail:  op 1 -> op 2 -> op 3 -> stop
  ```

- I2. `Do` sleeps exactly once between consecutive attempts and never
  after the last one.

  ```text
  op fails -> sleep -> op fails -> sleep -> op succeeds
               1                      2       no sleep
  ```

- I3. The sleep after failed attempt n lies in `[0, Backoff(p, n))`.

  ```text
  Backoff(p, 2)=200ms:  0ms <= sleep < 200ms
  ```

- I4. With a `Timeout`, `Do` skips any sleep that would wake after the
  deadline, counting time spent inside the operation.

  ```text
  start=0  op uses 60ms  proposed sleep=50ms  deadline=100ms
     0 ----------- 60 ---- 100 ---- 110
                  stop; the proposed wake would pass the deadline
  ```

- I5. `Do` returns nil exactly when some attempt returned nil.

  ```text
  op: error -> error -> nil    Do: nil
  op: error -> error          Do: non-nil error at attempt cap
  ```

- I6. When `Do` gives up, the returned error wraps the operation's last
  error and reports why it stopped.

  ```text
  last op error ----+
                    +--> ExhaustedError
  ErrMaxAttempts ---+     reason: attempt cap
  ```

- I7. An error wrapped with `Stop` ends `Do` at once, with no sleep, and
  `Do` returns the unwrapped error.

  ```text
  op -> Stop(err) -> Do returns err
                    sleeps: []
  ```

- I8. `Backoff(p, 1)` equals `InitialDelay`, `Backoff` never decreases as
  the attempt grows, and it never exceeds `MaxDelay`.

  ```text
  InitialDelay=100ms, Multiplier=2, MaxDelay=250ms
  attempt:   1      2      3      4
  Backoff: 100ms  200ms  250ms  250ms
  ```

- I9. `New` panics exactly when the policy breaks a rule in the `Policy`
  field comments.

  ```text
  All other fields valid:
  InitialDelay=100ms, MaxDelay=100ms -> New succeeds
  InitialDelay=100ms, MaxDelay= 99ms -> New panics
  ```

## Not promised

- Wall-clock accuracy. `clock.Real` may oversleep, and nothing here
  measures it.
- Exact schedules for delays of 2^53 nanoseconds or more, about 104 days.
  `Backoff` multiplies in float64, which loses integer precision there.
- A deadline that interrupts an operation. An operation may run past the
  deadline before it returns control to `Do`.
- Safe concurrent use of one `Retrier`, unless the `Rand` it holds is
  safe for concurrent use.

## Related

- `sim/clock/architecture.md`: the fake clock that makes I2 through I4
  checkable with exact numbers and no waiting.
- `docs/testing.md`: which test file checks what, and how to run each.

# tokenbucket invariants

A bucket admits one request per token. It starts with `Burst` tokens,
refills one token per `Every`, and keeps fractional refill credit between
calls. The caller controls time through the supplied clock.

## Invariants

- I1. A new bucket admits exactly `Burst` calls at its initial timestamp
  before denying another.

  ```text
  Burst=2, time=0:  allow -> allow -> deny
  ```

- I2. At each call, `Allow` admits exactly when capped available credit,
  including elapsed fractional refill, can pay for one token; only an
  admission spends that token.

  ```text
  Every=10ms, Burst=1
  t=0: allow, credit=0  ->  t=4ms: deny, credit=4ms
                         -> t=10ms: allow, credit=0
  ```

- I3. Idle time never stores credit beyond `Burst` tokens.

  ```text
  Burst=2, Every=10ms:  idle 100ms -> 2 tokens
  same time:               allow -> allow -> deny
  ```

- I4. Every closed interval of length `d` contains at most
  `Burst + floor(d/Every)` admissions, counting ties at both endpoints.

  ```text
  Burst=2, Every=10ms, interval [0, 10ms]
  t=0: allow, allow        t=10ms: allow     total=3, bound=3
  ```

- I5. With no successful intervening call, a denied caller succeeds on
  a retry at least `Every` later.

  ```text
  Every=10ms:  t=0 deny -> no admissions -> t=10ms allow
  ```

- I6. `Allow` uses the injected time and never sleeps or advances that
  clock.

  ```text
  fake clock at 10ms -> Allow() -> fake clock still at 10ms
                                   recorded sleeps: []
  ```

- I7. Given a valid clock, `New` accepts exactly policies with `Every > 0`,
  `Burst >= 1`, and `Burst*Every <= MaxInt64` nanoseconds; it rejects an
  invalid policy before accessing the clock.

  ```text
  Every=10ns, Burst=2 -> New reads clock and succeeds
  Every= 0ns, Burst=2 -> New panics before reading clock
  ```

- I8. The supported transitions remain exact at representable numeric
  boundaries and after an idle gap longer than `time.Duration` can
  represent.

  ```text
  Every=1ns, Burst=1:  t=0 allow -> t=1ns allow
  idle gap > MaxInt64 ns: credit stops at capacity, never wraps
  ```

## Not promised

The bucket does not guarantee fairness among competing callers or safe
concurrent use. The clock must be non-nil and never move backward; neither
nil clocks nor clock regression have defined behavior. A zero-value
`Bucket` is not usable. Rates finer than one token per nanosecond and
capacities beyond `MaxInt64` nanoseconds are outside the policy domain.
The admission bound permits the initial burst; it is not a strict
sliding-window quota. Wall-clock scheduling accuracy is outside this
package's control.

## Related

- [sim/clock/architecture.md](../../sim/clock/architecture.md) describes
  the fake clock that lets tests advance time without waiting.

## Commands

From the repository root, run the package checks and replayable simulation
and fuzz commands:

    go test ./projects/tokenbucket
    go vet ./projects/tokenbucket
    make test-seeds PROJECT=tokenbucket RUNS=10000
    make seed PROJECT=tokenbucket SEED=42
    make fuzz PROJECT=tokenbucket FUZZTIME=1s

# tokenbucket invariants

A bucket admits one request per token. It starts with `Burst` tokens,
refills one token per `Every`, and keeps fractional refill credit between
calls. The caller controls time through the supplied clock.

## Invariants

- I1. A new bucket admits exactly `Burst` calls at its initial timestamp
  before denying another. Checked by `TestStartsFull`,
  `TestReferenceModelInitialBurst`, `TestInvariantsAcrossSeeds`, `FuzzAllow`.
- I2. At each call, `Allow` admits exactly when capped available credit,
  including elapsed fractional refill, can pay for one token; only an
  admission spends that token. Checked by `TestRefillAtBoundary`,
  `TestDeniedCallsPreserveFractionalRefill`, `TestAdmissionSpendsOneToken`,
  `TestReferenceModelFractionalRefill`, `TestReferenceModelDenialPreservesDebt`,
  `TestReferenceModelBoundaryEqualityAndProgress`, `TestInvariantsAcrossSeeds`,
  `TestRefillCapDecisions`, `TestAdmissionDecisions`, `FuzzAllow`.
- I3. Idle time never stores credit beyond `Burst` tokens. Checked by
  `TestIdleRefillCapsAtBurst`, `TestFullBucketDiscardsIdleCredit`,
  `TestReferenceModelIdleSaturation`, `TestInvariantsAcrossSeeds`,
  `TestRefillCapDecisions`, `FuzzAllow`.
- I4. Every closed interval of length `d` contains at most
  `Burst + floor(d/Every)` admissions, counting ties at both endpoints.
  Checked by `TestAdmissionEnvelope`, `TestReferenceModelTiedWindowEndpoints`,
  `TestInvariantsAcrossSeeds`, `FuzzAllow`.
- I5. With no successful intervening call, a denied caller succeeds on
  a retry at least `Every` later. Checked by
  `TestDeniedCallerProgressAfterQuietPeriod`, `TestReferenceModelDenialPreservesDebt`,
  `TestReferenceModelBoundaryEqualityAndProgress`, `TestInvariantsAcrossSeeds`,
  `FuzzAllow`.
- I6. `Allow` uses the injected time and never sleeps or advances that
  clock. Checked by `TestAllowDoesNotSleepOrAdvanceTime`,
  `TestReferenceModelInitialBurst`, `TestReferenceModelFractionalRefill`,
  `TestReferenceModelDenialPreservesDebt`, `TestReferenceModelIdleSaturation`,
  `TestReferenceModelTiedWindowEndpoints`,
  `TestReferenceModelBoundaryEqualityAndProgress`, `TestInvariantsAcrossSeeds`,
  `FuzzAllow`.
- I7. Given a valid clock, `New` accepts exactly policies with `Every > 0`,
  `Burst >= 1`, and `Burst*Every <= MaxInt64` nanoseconds; it rejects an
  invalid policy before accessing the clock. Checked by
  `TestInvalidPolicyPanics`, `TestLargestValidCapacity`,
  `TestPolicyValidationDecisions`, `FuzzPolicyValidation`.
- I8. The supported transitions remain exact at representable numeric
  boundaries and after an idle gap longer than `time.Duration` can
  represent. Checked by `TestLargestValidCapacity`, `TestNanosecondRefill`,
  `TestLargeIdleGapSaturates`, `TestPolicyValidationDecisions`.

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

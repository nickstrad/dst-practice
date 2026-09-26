# Work state

Purpose: what is done, what is in flight, and what is open, so a fresh
session (or a cleared context) can pick up without re-reading everything.
Update this file at the end of every work session. Keep entries short.

Last updated: 2026-09-25

## Conventions

- Module name: `dstpractice`. Import paths look like `dstpractice/retry`.
- Every system under test (SUT) takes `clock.Clock` (and `Rand` etc.) at
  construction. No goroutines, no wall clock, no real I/O in the SUT.
- Style: readable over clever. Plain structs, comments that explain the
  why, tests that read like a spec.
- Seeded tests print `seed=N` on failure. Replay with
  `go test ./<pkg> -run <Test> -seed N`.
- Commit related changes in clear batches with clear messages when asked.
- Knowledge store lives in docs/knowledge/. Add learnings with the
  update-knowledge skill (.claude/skills and .codex/skills are mirrors,
  edit both).

## Done

- [x] 2026-09-25 Go module initialized, `.gitignore`, `README.md`.
- [x] 2026-09-25 `sim/clock`: `Clock` interface, `Real`, `Fake`
      (records sleeps, `Advance` for time spent inside an op).
- [x] 2026-09-25 Project 1 `retry/`: `Policy`, `Retrier.Do`, exported
      `Backoff` schedule, full jitter, `Stop` for permanent errors,
      `ExhaustedError` wrapping `ErrMaxAttempts` or `ErrTimeout` plus the last error.
- [x] 2026-09-25 `retry` tests: schedule, max attempts, deadline (including
      time spent inside op), permanent error, invalid policy, and a
      seeded invariant test with `-seed` and `-runs` flags.
- [x] 2026-09-25 `cmd/retrydemo`: real-clock demo.
- [x] 2026-09-25 `Makefile`: test, test-seeds, seed, vet, fmt, demo, clean.
- [x] 2026-09-25 Simplify pass on retry: typed `ErrMaxAttempts`/`ErrTimeout`,
      unexported permanent wrapper, honest test doubles, flakyOp struct.
- [x] 2026-09-25 Docs: `AGENTS.md` at root, `docs/` with coding-style,
      index, AGENTS.md, and `docs/knowledge/`; `sim/AGENTS.md` norm and
      `sim/clock/architecture.md` as the reference example.
- [x] 2026-09-25 `update-knowledge` skill under `.claude/skills` and
      `.codex/skills`.

## In progress

- (nothing)

## Open items

- [ ] Decide whether `Do` should accept a `context.Context`. Left out on
      purpose: a context deadline reads the wall clock, which breaks the
      DST rule. If needed, add a `DoCtx` that only checks `ctx.Err()`
      between attempts.
- [ ] `sim/clock.Fake` has no timers yet. Project 2 (token bucket) needs
      only `Now`; project 5 onward needs timers. Add when first needed.
- [ ] Consider a `sim/rand` package (`Chance`, `Pick`, `Shuffle`) once a
      second project needs it. Do not build it ahead of time.
- [ ] Next project: option 2, token bucket rate limiter (see `options.md`).

## How to verify

    make test
    make test-seeds RUNS=100000
    make seed SEED=42
    make demo

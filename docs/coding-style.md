# Coding style

This is a learning repo. Every rule below favors code a reader can learn
from over code that is clever or production hardened. Treat the code here
as reference code.

## Readability beats cleverness

Rule: write the plain version, even when a shorter trick exists.

A reader should follow each function top to bottom on the first read. If a
trick saves three lines but costs a reader a minute, skip the trick.

## Plain structs and explicit loops

Rule: prefer plain structs and explicit loops over math tricks.

`retry.Backoff` multiplies in a loop instead of calling `math.Pow`. The loop
stops at the cap, so it cannot overflow, and it reads top to bottom.

```go
delay := float64(p.InitialDelay)
for i := 1; i < attempt; i++ {
	delay *= p.Multiplier
	if delay >= float64(p.MaxDelay) {
		return p.MaxDelay
	}
}
```

## Comments explain why

Rule: a comment says why the code does something, not what it does.

The code already says what. A package doc comment goes further and shows a
worked example of the schedule or behavior, like the backoff table at the
top of `projects/retry/retry.go`.

## Tests read like a spec

Rule: a reader should learn the behavior from the test names and bodies.

Inline the idiom at each call site when the idiom is the lesson. Write
`errors.As` in every test that needs it rather than hiding it in a helper.

Spec tests are one of three kinds. `docs/testing.md` describes the other
two and the file name each kind uses.

## Test doubles obey the interface

Rule: a fake returns only values the interface allows.

If `Rand.Float64` promises a value in `[0, 1)`, the fake never returns `1.0`
to make a test easy. A fake that breaks the contract tests a system that
cannot exist.

## Typed errors and sentinels

Rule: return typed errors and sentinel values, not free-form strings.

Tests check errors with `errors.Is` and `errors.As`. Comparing strings
breaks the moment someone rewords a message.

```go
if !errors.Is(err, retry.ErrTimeout) { ... }
```

## Small exported API

Rule: export one entry point per idea.

`retry.Stop` is exported. The `permanent` wrapper type it returns is not.
Callers learn one function, and the package keeps the freedom to change the
wrapper.

## No abstraction before the second user

Rule: build a shared helper only when a second caller needs it.

Until then, write the plan down in the Open items section of the nearest
`architecture.md`, or in the state file for the current work under
`.scratchpad/state/`. The `sim/rand` package waits in `docs/options.md`
now.

## The DST design rule

Rule: the system under test never spawns a goroutine, never reads the wall
clock, and never touches network or disk on its own.

It takes a `Clock`, `Rand`, `Storage`, and `Network` at construction.
Production passes real ones. Tests pass fakes. This rule makes every run
repeatable from a seed.

## Seeded tests

Rule: a seeded test prints `seed=N` on failure and accepts `-seed` to
replay it.

A failure you cannot replay is a failure you cannot fix.

    go test ./projects/retry -run TestInvariantsAcrossSeeds -seed 42

## Clean before done

Rule: run `make vet` before you call a change done.

It checks gofmt and runs `go vet ./...`. Without make, run `gofmt -l .` and
`go vet ./...`.

# Testing

This repo uses three kinds of test. They answer different questions, so a
package may have all three. Each kind lives in its own file with a fixed
name suffix, so a reader can tell what any test file is for from its name.

## File naming

For a package `foo`:

| File                    | Holds                                   |
| ----------------------- | --------------------------------------- |
| `foo_test.go`           | Spec tests. One named behavior each.    |
| `foo_dst_test.go`       | Seeded simulation runs over invariants. |
| `foo_fuzz_test.go`      | Fuzz targets, `Fuzz*` functions.        |
| `foo_mcdc_test.go`      | Decision tables, one per decision.      |
| `foo_helpers_test.go`   | Shared doubles and the invariant oracle.|

Every file starts with a comment that says which kind it is. A test that
checks an invariant names the invariant's ID in a comment. The IDs come
from the package's `invariants.md`, or from the `What it proves` section
of a sim package's `architecture.md`.

`retry/` is the reference example. Read its five test files in the order
above.

## Spec tests

Question: does the code do the specific thing the name says?

One behavior per test, exact numbers, a fixed jitter so every sleep is
predictable. A reader learns the behavior from the test names and bodies.
This is the kind `docs/coding-style.md` describes under "Tests read like
a spec".

## Deterministic simulation tests

Question: do the invariants hold across many schedules and fault patterns?

The test seeds a random source, draws a policy and a failure pattern from
it, drives the system under test against the fake clock, and checks every
invariant. It repeats for many seeds. A failure prints the seed, and the
same seed replays the exact run.

    make test-seeds RUNS=100000
    make seed SEED=42

The invariant checks live in one function in the helpers file, so the DST
test and the fuzz target share them. This is the main focus of the repo.
`docs/options.md` lists the patterns each project practices.

## Fuzz tests

Question: is there any input at all that crashes the code or breaks a
property?

Go's fuzzer mutates the inputs of a `Fuzz*` function and watches code
coverage, so it steers toward branches the seeded test never reaches. It
found that `NaN` passed the policy check, because `NaN < 1` is false.

Rules for a fuzz target:

- Give it a seed corpus with `f.Add`. A plain `go test` replays only
  those seeds plus any saved failures under `testdata/fuzz/`, so the
  target costs nothing in the normal run.
- Fold raw inputs into the domain the code promises, and `t.Skip()`
  inputs that would panic by design. Say why in a comment.
- Check properties, not exact values. For `Do`, call the same invariant
  oracle the DST test uses.
- When the fuzzer finds a failure it writes the input to
  `testdata/fuzz/<Target>/`. Commit that file. It becomes a permanent
  regression test.

Run the search on purpose, one target at a time:

    make fuzz                 # every target, 10s each
    make fuzz FUZZTIME=1m

`go test -fuzz` accepts one target per run, so the Makefile loops.

## MC/DC decision tables

Question: does the test suite show that each condition in a compound
decision changes the outcome on its own?

MC/DC stands for modified condition/decision coverage. It is a coverage
criterion, not a test style. For a decision with N conditions it needs
N+1 rows, where each row differs from a neighbor in one condition and
the outcome flips. Single-condition decisions only need both branches.

Go has no MC/DC tool. `go test -cover` reports statement coverage only.
Commercial tools exist, but this repo does the analysis by hand, which
suits a learning repo: the table is the coverage record and it reads as
a spec.

Rules for a decision table:

- One table per decision. Quote the decision in the test comment.
- One row per condition change, plus the all-true or all-false row.
  Add a row for any equality boundary, such as a wake-up landing exactly
  on the deadline.
- Name each row after the condition it changes.
- Assert the outcome that the decision controls, not something
  downstream.

To see which statements no test touches:

    make cover

## Which kind to write

- New behavior: a spec test first, then add the invariant to
  `invariants.md` and to the oracle so the DST and fuzz runs cover it.
- New compound decision or boundary: a decision table.
- New pure function with a clear property: a fuzz target.
- New failure the fuzzer or a seed found: commit the corpus file or add
  the seed as a spec test, and record the learning in `docs/knowledge/`.

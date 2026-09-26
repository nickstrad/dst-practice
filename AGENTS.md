# AGENTS.md

## What this repo is

This repo holds small Go systems built to practice deterministic simulation
testing. See `docs/options.md` for the project list and `state.md` for current
work.

## Always do first

- Scan `docs/knowledge/index.md` before you start any task. Read every entry
  that touches the task.
- Read `state.md` for current work and open items. Update it when you
  finish.

## Rules

- Follow `docs/coding-style.md`.
- Use the On Writing Well skill whenever you write in this repo, including
  chat threads, comments, docs, and commit messages.
- Each sim module has an `architecture.md` next to its code. See
  `sim/AGENTS.md`.
- Each system under test outside `sim/` has an `invariants.md` next to its
  code. Sim packages state their invariants in `architecture.md` instead.
  See the Invariants section below.
- Start at `docs/index.md` for all other docs.
- Put scripts, research notes, and experiments in `.scratchpad/`, organized
  by dated topic folder. Write scripts in Go or Bash unless another language
  is necessary. See `.scratchpad/AGENTS.md`. Review it periodically for
  knowledge or code worth promoting into the repo.
- Use the `do-work` skill for any task that may outlive one context window.
  It keeps a resumable state file in `.scratchpad/state/`.
- Do not commit unless asked.
- When asked to commit, group related changes into clear batches and write a
  clear commit message for each batch.

## Invariants

Everything in this repo exists to state invariants and test them. Each
invariant lives in exactly one document and is checked by at least one
named test.

- A system under test outside `sim/`, such as `retry/`, owns its
  invariants in `invariants.md` next to its Go source.
- A package under `sim/` states its invariants in the `What it proves`
  section of its `architecture.md`. Do not add an `invariants.md` there.
- Create or update the document in the same change that adds or changes
  the behavior. A new invariant lands with the test that checks it.
- Each invariant has an ID, one sentence, and the name of the test that
  checks it. A test that checks an invariant names the ID in a comment.
- A test file checks invariants. It does not define them. If a test
  asserts something the document does not state, add it to the document
  or drop the assertion.

Template for `invariants.md`:

```
# <package> invariants

One paragraph: what the system promises, in plain words.

## Invariants

- I1. <One sentence that is always true.> Checked by `TestName`.
- I2. <...> Checked by `TestName`, `TestOther`.

## Not promised

What the system does not guarantee, so nobody writes a test for it.

## Related

Sim packages whose fakes make these checks possible, with a link to
their `architecture.md`.
```

## Verify

    make test
    make vet

Without make:

    go test ./...
    go vet ./...

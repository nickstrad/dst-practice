# AGENTS.md

## What this repo is

This repo holds small Go systems built to practice deterministic simulation
testing. Each system lives in its own folder under `projects/`. See
`docs/options.md` for the project list and `.scratchpad/state/` for work
in flight.

## Always do first

- Scan `docs/knowledge/index.md` before you start any task. Read every entry
  that touches the task.
- Check `.scratchpad/state/` for a file on the task you were given. If one
  exists, resume from it. If not, and the task may outlive one context
  window, start one with the `do-work` skill.

## Rules

- Follow `docs/coding-style.md`.
- Follow `docs/testing.md` for which kind of test to write and what to
  name the file.
- Use the On Writing Well skill whenever you write in this repo, including
  chat threads, comments, docs, and commit messages.
- Each sim module has an `architecture.md` next to its code. See
  `sim/AGENTS.md`.
- Each project lives in its own folder under `projects/`. See
  `projects/AGENTS.md`.
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

## Plans with subagents

Use `plan-agent-flow-state` when asked for a detailed plan with agent
assignments and persistent state. Use `do-work` to maintain the same file
at `.scratchpad/state/<task>.md`; do not create a second state file.

Reuse these roles from `.claude/agents/*.md` or `.codex/agents/*.toml`:

| Role | Model and effort | Responsibility |
|---|---|---|
| `plan-coordinator` | Fable high / Astra high | Own design, task assignments, state, validation, and escalation. |
| `plan-builder-strong` | Opus medium / Sol medium | Build stateful logic and complex tests. |
| `plan-builder-fast` | Sonnet high / Luna high | Build bounded tables, commands, and documentation. |
| `plan-reviewer` | Fable high / Astra high | Independently review plans, coordinator work, and final integration. |

These repo defaults replace the skill's generic role installation and
commit workflow. Do not generate task-specific role copies. The coordinator
reads submissions and reruns acceptance checks; an independent reviewer
checks the coordinator's own work. Keep the execution table and append-only
log current, including owners, models, attempts, findings, evidence, and
next actions. Reuse accepted uncommitted work after a context clear, and
check for live agents before reassigning their files.

A request to plan produces a reviewed plan. A request to build or implement
authorizes execution through the coordinator without another approval
step. Never commit unless asked; leave plans and role definitions
uncommitted unless the user explicitly changes that restriction.

Example: `$plan-agent-flow-state build <feature>`.
Resume: `Continue .scratchpad/state/<task>.md`.

## Invariants

Everything in this repo exists to state invariants and test them. Each
invariant lives in exactly one document and is checked by at least one
named test.

- A system under test outside `sim/`, such as `projects/retry/`, owns its
  invariants in `invariants.md` next to its Go source.
- A package under `sim/` states its invariants in the `What it proves`
  section of its `architecture.md`. Do not add an `invariants.md` there.
- Create or update the document in the same change that adds or changes
  the behavior. A new invariant lands with the test that checks it.
- Give each invariant an ID, one sentence, and a small text-art example in
  a fenced block directly below it. Keep the example concrete and show
  the boundary or sequence that makes the rule visible.
- Do not list test names in an invariant document. A test that checks an
  invariant names its ID in a comment. Those comments are the mapping
  from tests to invariants.
- A test file checks invariants. It does not define them. If a test
  asserts something the document does not state, add it to the document
  or drop the assertion.

Template for `invariants.md`:

```
# <package> invariants

One paragraph: what the system promises, in plain words.

## Invariants

- I1. <One sentence that is always true.>

  ```text
  <Small example with concrete values and the observed result.>
  ```

- I2. <One sentence that is always true.>

  ```text
  <Small example with a sequence or boundary.>
  ```

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

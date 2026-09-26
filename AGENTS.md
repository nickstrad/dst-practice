# AGENTS.md

## What this repo is

This repo holds small Go systems built to practice deterministic simulation
testing. See `options.md` for the project list and `state.md` for current
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
- Start at `docs/index.md` for all other docs.
- Do not commit unless asked.
- When asked to commit, group related changes into clear batches and write a
  clear commit message for each batch.

## Verify

    make test
    make vet

Without make:

    go test ./...
    go vet ./...

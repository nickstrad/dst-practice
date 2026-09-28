# Docs index

The `docs/` folder holds documentation for people and agents. It holds no
code. Update this index whenever you add a file to `docs/`.

- `options.md`: the project list and the deterministic simulation testing
  patterns each project covers.
- `coding-style.md`: the rules for code in this repo. Read it before you
  write or review Go code.
- `testing.md`: the three kinds of test, the file name for each, and how
  to run them. Read it before you add or change a test.
- `knowledge/`: learnings from past work, such as gotchas, decisions, and
  patterns. Start at `knowledge/index.md` and read it before any task.
- `knowledge/check-fuzz-seeds-after-folding.md`: why fixed seeds must match
  their decoded policies and event sequences.
- `knowledge/probe-residual-state-after-a-cap.md`: how later calls reveal
  incorrect surplus that immediate cap observations can miss.
- `AGENTS.md`: instructions for AI agents that edit files in `docs/`. Read
  it before you add or change a doc.

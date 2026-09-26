# sim/AGENTS.md

Every package under `sim/` gets an `architecture.md` next to its Go source.
This includes `sim/clock`, and the packages `options.md` lists as future
work: `sim/rand`, `sim/disk`, `sim/net`, `sim/sched`, `sim/trace`,
`sim/check`.

Read `sim/clock/architecture.md` first. It is the reference example. Copy
its structure for every new package.

## Required template

```
# <package> architecture

## Point
One paragraph: the problem this fake solves and the DST pattern IDs from
options.md it carries. Example: P2 fake clock.

## Shape
Text-art diagram of the interface, the real implementation, and the fake,
and who holds which.

## How it is used
A text-art timeline or sequence: a system under test drives the fake while
a test controls time or faults. Follow it with a short paragraph or a few
bullets.

## What it proves
The invariants a test can check because this fake exists. State its limits
too.

## Open items
What the fake does not do yet, and which future project needs it.
```

## Rules

- Create `architecture.md` in the same change that creates a new `sim`
  package. Update it in the same change that changes the package's public
  surface.
- Draw diagrams as plain text art in fenced code blocks. Skip image files
  and mermaid, so the diagram renders in any terminal and diffs cleanly.
  Keep each diagram under about 25 lines and 72 columns.
- Follow `docs/coding-style.md` and the writing-well rules: active voice,
  short sentences.
- Explain mechanism and intent. Skip a line-by-line account of the code.
  Link the pattern IDs in `options.md`.
- Keep the whole file readable in one or two screens.

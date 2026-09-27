# projects/AGENTS.md

Every system under test from `docs/options.md` gets its own folder here.
The folder name is the package name: `projects/retry` holds package
`retry`. Only the fakes in `sim/` live outside this folder.

Read `projects/retry` first. It is the reference example for the layout,
the test files, and the `invariants.md` every project carries.

## Rules

- One project per folder. Do not share code between projects until a
  second project needs it. Then move the shared part into `sim/`.
- Each project owns its invariants in an `invariants.md` next to its Go
  source. The top-level `AGENTS.md` holds the template.
- Name test files as `docs/testing.md` says.
- A demo that runs a project against the real clock goes under `cmd/`,
  not here. `cmd/retrydemo` is the example.

# Knowledge

This folder holds general learnings that help future agents work in this
repo: gotchas, decisions and why we made them, patterns that worked, and
things that failed.

## Rules

- A learning is either a standalone `.md` file or a folder with a
  `README.md`.
- Use a folder when the learning has supporting artifacts such as scripts,
  traces, or data. Those artifacts live inside that folder.
- Every learning gets one line in the list below, in this form:
  `- [Title](path) - one line hook`
- Use the `update-knowledge` skill to add an entry. It lives at
  `.claude/skills/update-knowledge` and `.codex/skills/update-knowledge`.

## Entries

- [A retry deadline cannot interrupt an operation](retry-deadline-cannot-interrupt-op.md) - the retrier can skip a late sleep but cannot stop an operation already running
- [Backoff is exact only below 2^53 nanoseconds](backoff-exact-only-below-2-53-ns.md) - fold fuzz durations into [1, 2^53) and list the limit under Not promised
- [Check fuzz seeds after folding](check-fuzz-seeds-after-folding.md) - verify decoded policies and events before naming fixed corpus cases
- [make and git can fail with an Xcode license error](make-and-git-need-xcode-license.md) - Apple shims stop working until sudo xcodebuild -license accept; use go commands directly
- [NaN passes a less-than validation check](nan-passes-less-than-check.md) - guard float fields with math.IsNaN too; add a NaN row and seed
- [Probe residual state after a cap](probe-residual-state-after-a-cap.md) - use a later public observation to expose incorrectly retained surplus

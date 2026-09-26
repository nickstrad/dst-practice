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

- [make and git can fail with an Xcode license error](make-and-git-need-xcode-license.md) - Apple shims stop working until sudo xcodebuild -license accept; use go commands directly

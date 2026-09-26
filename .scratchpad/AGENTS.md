# .scratchpad

## What this folder is

A working area for scripts, research notes, experiments, traces, and other
material that does not belong in the codebase yet. Agents and the user put
work-in-progress here instead of in `/tmp`, so it survives the session and
can be mined later.

## Layout

- `state/` holds one file per unit of work, written by the `do-work` skill.
  A state file lets an agent clear its context and pick the work back up.
  See `.claude/skills/do-work/SKILL.md`.
- Everything else goes in a dated topic subfolder, `YYYY-MM-DD-short-topic/`,
  with a short `README.md` that says what the work was for and what came
  of it.

## Rules

- Write scripts in Go or Bash. Use another language only when the task
  needs it, and say why in the README.
- Keep scripts runnable. Note the command that runs them in the README.
- Do not put secrets here.
- Nothing here is load-bearing. Code in the repo must never import from or
  depend on this folder.

## Periodic review

Every so often, scan this folder for two things:

- Knowledge worth keeping: a gotcha, a decision and its reason, a failed
  approach, a replayable seed. Move it into `docs/knowledge/` with the
  `update-knowledge` skill.
- Code worth keeping: a helper or tool that proved useful more than once.
  Promote it into the repo under the normal coding style.

After promoting something, delete it here or leave a one-line pointer in
its README to where it went.

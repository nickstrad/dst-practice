---
name: do-work
description: This skill should be used when the user types "/do-work", says "use do-work", or asks to "start tracked work", "plan this in the scratchpad", "make a state file for this", or "work on this so I can clear context later". Creates one file in .scratchpad/state/ that holds either the resumable state of the work or its plan and work log, keeps that file current while working, and ends with a reflection step that records learnings with the update-knowledge skill.
---

<!-- Mirror of the other copy under .claude/skills or .codex/skills. Edit both. -->

# Do work

Track a unit of work in one file under `.scratchpad/state/` so that an
agent can clear its context at any point and restart from that file alone.

## Procedure

1. Read `AGENTS.md` and `docs/knowledge/index.md`. Read every knowledge
   entry that touches the task.
2. Check `.scratchpad/state/` for an existing file on this work. If one
   exists, read it and continue from its last entry. Do not create a
   second file.
3. Otherwise create `.scratchpad/state/<slug>.md`. Use kebab-case and name
   the work, not the date. Pick one of the two shapes below and say which
   at the top of the file.
4. Do the work. After each meaningful step, update the file before doing
   anything else. Treat the file as the only memory that survives.
5. Verify with `make test` and `make vet`, or the direct `go` commands
   when make is unavailable. Record the result in the file.
6. Reflect. Answer these in the file under a `## Reflection` heading:
   what was non-obvious, what failed and why, what a future agent should
   know before touching this area, and any seed or command that replays a
   finding.
7. For each item in the reflection that is a real learning, run the
   `update-knowledge` skill. Add a pointer in the reflection to each entry
   you wrote.
8. Finish by printing the state file path and the knowledge entries you
   added.

## The two shapes

Pick the smaller shape that still lets a fresh agent resume.

**State only.** For short or linear work. The file holds only the current
state, rewritten in place each time. Keep it under one screen.

```markdown
# <Title>

Shape: state only

## Goal
One or two sentences.

## Current state
Where the work stands right now. What is done, what is verified.

## Next step
The single next action.

## Open questions
Anything blocked on the user.
```

**Plan and log.** For work with several steps or design choices. The plan
is written once and edited only when the design changes. The log is
append only.

```markdown
# <Title>

Shape: plan and log

## Goal
One or two sentences.

## Plan
1. Step, with the files it touches.
2. Step.

## Log
- <YYYY-MM-DD HH:MM> Started step 1. ...
- <YYYY-MM-DD HH:MM> Step 1 done. Tests pass. Decided X because Y.

## Reflection
Filled in at the end. See step 6.
```

## Rules

- Write in active voice with short sentences. Use no em-dashes and no
  parentheticals.
- Write dates as absolute dates, never "today" or "yesterday".
- Record decisions with their reason. A future agent needs the why more
  than the what.
- Keep scratch scripts and research for this work in a dated topic folder
  under `.scratchpad/`, and link to it from the state file. Write them in
  Go or Bash unless another language is necessary.
- Do not commit unless the user asked.
- When the work is done and its learnings are recorded, tell the user the
  state file can be deleted. Do not delete it yourself.

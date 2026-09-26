---
name: update-knowledge
description: This skill should be used when the user asks to "save this learning", "add to knowledge", "record what we learned", "update the knowledge store", or types "/update-knowledge". It should also be used when an agent finishes a task that uncovered something non-obvious, such as a gotcha, a decision and its reason, a failed approach, or a replayable seed. Adds one learning to docs/knowledge/ and indexes it in docs/knowledge/index.md.
---

<!-- Mirror of the other copy under .claude/skills or .codex/skills. Edit both. -->

# Update knowledge

The knowledge store in `docs/knowledge/` holds general learnings that help
future agents work in this repo. Each learning gets one entry and one index
line.

## Procedure

1. Read `docs/knowledge/index.md`. Check whether an existing entry already
   covers this learning. If one does, update that entry and skip step 5
   unless its title or hook changed.
2. Choose the shape. A learning with no artifacts is a single file,
   `docs/knowledge/<slug>.md`. A learning with any artifact, such as a
   script, trace, seed file, data, or diagram, is a folder,
   `docs/knowledge/<slug>/`, with `README.md` as the entry and every
   artifact inside that folder. Never leave artifacts loose in
   `docs/knowledge/`.
3. Pick the slug. Use kebab-case. Name the learning, not the date. Good:
   `fake-clock-needs-advance-inside-op`. Bad: `2026-09-25-notes`.
4. Write the entry from this template. Keep it short. Record one learning
   per entry. Omit `Evidence` when there is none.

   ```markdown
   # <Title>

   ## Context
   What we were doing when this came up.

   ## Learning
   The fact or rule, in one to five sentences.

   ## Why it matters
   What goes wrong if an agent does not know this.

   ## How to apply
   Concrete steps or a short snippet.

   ## Evidence
   Seed, command, file paths, or test name.

   ## Related
   Links to other entries or code paths.
   ```

5. Add one line to the list section of `docs/knowledge/index.md`:
   `- [Title](path) - one line hook`. Use `<slug>.md` for a file and
   `<slug>/README.md` for a folder. Keep the list alphabetical by title.
6. Finish by printing the entry path and the index line you added.

## Rules

- Write in active voice with short sentences. Use no em-dashes and no
  parentheticals.
- Record only what was non-obvious. Skip anything the code or git history
  already says.
- Do not commit unless the user asked.

---
type: task
status: done
feature: sketch-blueprints
scope: medium
review: done
depends_on: [blueprint-notes]
summary: Notes are draft until y marks them ready, sketch check gates a build, and sketch done deletes a feature's notes.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T19:33:44Z
---

# Status, check and done

## Description

New notes start with `status: draft`, and `y` in the explorer switches the selected note between draft and ready, with drafts dimmed and the header counting drafts, ready notes, open questions and dangling links. `sketch check [--feature X]` lists every problem and exits non-zero on any, and `sketch done X` deletes that feature's notes and lists them, leaving the commit to the caller. It gets a review because `check` is the gate `implement` will trust.

## Files

- internal/sketch/note/note.go
- internal/sketch/explorer/explorer.go
- internal/sketch/explorer/settings.go
- internal/sketch/command/command.go
- internal/sketch/command/command_test.go

## Acceptance criteria

- [x] A note made with `a` has `status: draft` in its front matter.
- [x] `y` turns the selected note ready, and pressing it again turns it back to draft.
- [x] Draft notes show dimmed in the tree, and the header shows the readiness counts.
- [x] `sketch check --feature X` exits 0 only with every note ready, every question answered and every link resolved, and otherwise lists each problem.
- [x] `sketch done X` deletes exactly the notes linked to X.

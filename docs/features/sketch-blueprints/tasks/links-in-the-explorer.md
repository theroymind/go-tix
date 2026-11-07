---
type: task
status: todo
feature: sketch-blueprints
scope: medium
review: none
depends_on: [blueprint-notes]
summary: r picks a file and adds -> path to the note, and the footer shows uses and used-by.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T18:47:01Z
---

# Links in the explorer

## Description

`r` opens the fuzzy picker over the repo's files and planned `+` targets and appends `-> path` to the selected note. A line under the tree shows the selected note's `uses →` and `used by ←`, each marked `+`, `~`, or `!` when the path is neither a file nor a planned target. It replaces the old symbol picker, so `refs`' name scan goes if nothing else uses it. No review, since the parsing lives in blueprint-notes.

## Files

- internal/sketch/explorer/picker.go
- internal/sketch/explorer/explorer.go
- internal/sketch/explorer/settings.go
- internal/sketch/explorer/explorer_test.go

## Acceptance criteria

- [ ] `r` on a note lists repo files and planned targets, and picking one appends `-> path`.
- [ ] The footer shows `uses →` for the note's links and `used by ←` for notes linking to it.
- [ ] A link to a path with no file and no note shows `!`.

---
type: task
status: todo
feature: sketch-blueprints
scope: large
review: pending
depends_on: []
summary: Notes carry their whole body, status, links and multi-line answers, with item parsing and marks removed.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T18:47:01Z
---

# Blueprint notes

## Description

Rewrite `contracts/sketch` at schema version 2 around a note's body: `Note` gains `Status`, `Body` and `Edges` (`{To, Label, Mark}`) and loses `Items`, `ItemType`, `Task` and `UnknownFeature`. The parser reads `-> path : label`, the path from the repo root or from the note's folder when it starts with `./` or `../`, and a `>` answer that runs over several lines and may follow one blank line. The `marks` package goes, tix becomes optional with a warning on stderr, and the marker convention moves into `sketch help` and the README as suggestions. It gets a review because it rewrites a public contract.

## Files

- contracts/sketch/sketch.go
- contracts/sketch/settings.go
- contracts/sketch/sketch_test.go
- internal/sketch/note/note.go
- internal/sketch/note/note_test.go
- internal/sketch/marks/marks.go
- internal/sketch/questions/questions.go
- internal/sketch/command/command.go
- internal/sketch/explorer/explorer.go
- README.md

## Acceptance criteria

- [ ] `sketch ls --json` prints each note at schema version 2 with its `body`, `status`, `edges` and questions, and no items.
- [ ] A `>` answer over two lines, or one blank line below its question, closes the question.
- [ ] `sketch questions --json --all` includes answered questions with their answers.
- [ ] With no `tix` on the PATH, `sketch ls --json` lists the notes and warns on stderr.
- [ ] `sketch help` shows a short example note and suggests the marker convention.
- [ ] `internal/sketch/marks` is gone, and `make check` passes.

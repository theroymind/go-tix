---
type: task
status: todo
feature: sketch
scope: medium
review: pending
depends_on: [one-shot-grammar-and-show]
summary: List and answer open questions, mark items new or changed from the target file, and resolve features and tasks through tix.
created_at: 2026-11-06T02:46:19Z
updated_at: 2026-11-06T02:46:19Z
---

# Questions, change marks and tix refs

## Description

This adds the three things that make a sketch a conversation rather than a note. A question stays open until it's answered. An item is marked changed when its name already appears in its target file, found by a plain text search, which is enough for Odin and TypeScript alike. A feature name resolves through tix's store when a `.tix/` is present. It gets a review because it reaches into tix's backend from a second app.

## Files

- `internal/sketch/questions/questions.go`
- `internal/sketch/marks/marks.go`
- `internal/sketch/refs/refs.go`

## Acceptance criteria

- [ ] `sketch questions FEATURE --json` lists every unanswered question with its context, and `sketch answer FEATURE N TEXT` records the answer and closes it.
- [ ] `show` marks an item `~` when its name appears in its target file, and `+` otherwise. A file that doesn't exist yet marks everything `+`.
- [ ] Inside a tix repo, a feature that isn't a ticket is refused, naming it, and the sketch lives beside the feature's note wherever tix keeps it. Outside one, any name works.

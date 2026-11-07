---
type: task
status: todo
feature: sketch-explorer
scope: medium
review: none
depends_on: [add-attach-edit-and-delete-notes]
summary: r picks a name from an attached note's target, f filters to one tix feature, and ? jumps between notes with open questions.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T03:21:03Z
---

# Symbol refs, feature filter and question jumps

## Description

Add the keys that make it a sketching tool rather than just an explorer: `r` on an attached note lists the identifiers in its target with a filter as you type and adds the pick as a line in the note, `f` picks a feature from tix's features and the ones notes name and filters the tree to its notes, and `?` moves to the next note with an open question. The identifiers come from the same plain scan the old completer used, not treesitter. No review, since it's wiring over parts the earlier tasks built and reviewed.

## Files

- `internal/sketch/explorer/explorer.go`
- `internal/sketch/explorer/explorer_test.go`
- `internal/sketch/explorer/picker.go`
- `internal/sketch/refs/refs.go`

## Acceptance criteria

- [ ] `r` on `~ useCamera.ts` lists its names, typing filters them, and `enter` appends ``- uses `make_camera` `` to the note.
- [ ] `f` lists features, picking one shows only its notes and their folders, and the header names the filter.
- [ ] `?` moves to the next note with an open question and wraps around, and the header shows the open-question count.

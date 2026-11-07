---
type: task
status: done
feature: sketch-explorer
scope: medium
review: pending
depends_on: [explorer-tree-with-ghost-nodes]
summary: Keys to create placeholder and attached notes, open them in $EDITOR or the parent nvim, and delete them.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T03:46:44Z
---

# Add, attach, edit and delete notes

## Description

Make the explorer write notes: `a` on a folder asks for a file name and creates `<name>.sketch.md` there, `A` on a file creates its attached note, and on a node that already has a note either key opens it. New notes start with front matter carrying the active feature filter and a `#` heading. `e` suspends the TUI through `tea.ExecProcess` and runs `$EDITOR`, or sends the file to `$NVIM` with `--remote` when it's set, then reloads the tree. It gets a review for its child processes and file-creation edge cases: names with folders, an existing note, and cancelling.

## Files

- `internal/sketch/explorer/explorer.go`
- `internal/sketch/explorer/explorer_test.go`
- `internal/sketch/explorer/prompt.go`
- `internal/sketch/editor/editor.go`
- `internal/sketch/editor/editor_test.go`

## Acceptance criteria

- [x] `a` on `src/composables/` and typing `useOrbit.ts` creates `src/composables/useOrbit.ts.sketch.md`, shows a `+` ghost node, and opens it in the editor.
- [x] `A` on an existing file creates its note and shows `~`, and on a node that already has a note, `a` or `A` opens it instead.
- [x] `e` opens the note in `$EDITOR`, or in the parent nvim when `$NVIM` is set, and the tree and preview reload afterwards.
- [x] `d` asks before deleting a note, and the node goes back to plain or disappears.
- [x] `esc` cancels a name prompt without creating anything.

---
type: task
status: todo
feature: sketch-explorer
scope: large
review: pending
depends_on: [marks-questions-and-tix-features]
summary: A read-only Bubble Tea explorer showing the repo tree with notes as + ghost nodes or ~ badges, a preview pane, and n for notes only.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T03:21:03Z
---

# Explorer tree with ghost nodes

## Description

Add Bubble Tea, Lip Gloss and Bubbles, and make bare `sketch` open the explorer. `internal/sketch/tree` builds the tree model from `git ls-files -co --exclude-standard`, falling back to a directory walk outside git, and folds each note into its target's node, a ghost when the target is missing and a `~` badge when it exists, so notes never show as their own `.sketch.md` rows. `internal/sketch/explorer` renders the tree and the preview pane, with arrow and vim keys, expand and collapse, and `n` for notes only; keeping the tree model apart from the Bubble Tea view lets it be tested without a terminal. It gets a review because it adds the dependency and carries cursor and expansion state.

## Files

- `go.mod`
- `internal/sketch/tree/tree.go`
- `internal/sketch/tree/tree_test.go`
- `internal/sketch/explorer/explorer.go`
- `internal/sketch/explorer/explorer_test.go`
- `internal/sketch/explorer/settings.go`
- `internal/sketch/command/command.go`

## Acceptance criteria

- [ ] `sketch` in sketch-trial opens the tree with `node_modules` and `dist` hidden.
- [ ] A note for a missing file shows as a `+` ghost node in its folder, and a note for an existing file shows as `~` on that file.
- [ ] `j`/`k`, the arrow keys, `enter` and `h`/`l` move, expand and collapse, and `q` quits.
- [ ] The preview pane shows the selected node's note, or the file's first lines when it has none.
- [ ] `n` shows only notes and the folders above them, and pressing it again restores the tree with the cursor kept.

---
type: task
status: done
feature: sketch-blueprints
scope: medium
review: pending
depends_on: [explorer-keys-and-naming]
summary: Inside nvim, sketch shows the selected note in nvim's main window instead of its own preview pane, and LazyVim opens it as a left split.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T19:16:40Z
---

# Paired nvim editor

## Description

When `$NVIM` is set, moving onto a note opens it in that nvim's main window without taking focus, `a`, `enter`, `e` and `>` focus it, and the preview pane hides so the tree takes the split. Without `$NVIM`, the preview pane and `$EDITOR` work as now, and the README's LazyVim snippet changes from a float to a left split. It gets a review because driving another process over `--remote` has edge cases: an unsaved buffer, a closed window, a note deleted underneath.

## Files

- internal/sketch/editor/editor.go
- internal/sketch/editor/editor_test.go
- internal/sketch/explorer/explorer.go
- README.md
- ~/.config/nvim/lua/plugins/sketch.lua

## Acceptance criteria

- [x] `<leader>k` in LazyVim opens sketch as a left split beside the main window.
- [x] Moving onto a note shows it in the main window while focus stays in sketch.
- [x] `enter` on a note moves focus to it in the main window.
- [x] Outside nvim, the preview pane shows and `e` opens `$EDITOR`.

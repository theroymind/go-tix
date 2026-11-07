---
type: task
status: done
feature: sketch-explorer
scope: small
review: none
depends_on: [add-attach-edit-and-delete-notes]
summary: A LazyVim plugin spec that opens sketch in a float on <leader>k, documented in the README.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T03:54:34Z
---

# LazyVim float

## Description

Add `~/.config/nvim/lua/plugins/sketch.lua`, which opens `sketch` in a Snacks terminal float on `<leader>k` the way LazyVim opens lazygit (LazyVim already maps `<leader>S` to its scratch picker), and document the same snippet in the README for other setups. Snacks is used because it's LazyVim's default and already installed. No review, since it's a config file.

## Files

- `~/.config/nvim/lua/plugins/sketch.lua`
- `README.md`

## Acceptance criteria

- [x] `<leader>k` in LazyVim opens the explorer in a float at the project root, and `q` closes it.
- [x] `e` in the float opens the note in that nvim and closes the float.
- [x] The README shows the snippet and the note format.

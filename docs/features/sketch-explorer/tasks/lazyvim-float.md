---
type: task
status: todo
feature: sketch-explorer
scope: small
review: none
depends_on: [add-attach-edit-and-delete-notes]
summary: A LazyVim plugin spec that opens sketch in a float on <leader>S, documented in the README.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T03:21:03Z
---

# LazyVim float

## Description

Add `~/.config/nvim/lua/plugins/sketch.lua`, which opens `sketch` in a Snacks terminal float on `<leader>S` the way LazyVim opens lazygit, and document the same snippet in the README for other setups. Snacks is used because it's LazyVim's default and already installed. No review, since it's a config file.

## Files

- `~/.config/nvim/lua/plugins/sketch.lua`
- `README.md`

## Acceptance criteria

- [ ] `<leader>S` in LazyVim opens the explorer in a float at the project root, and `q` closes it.
- [ ] `e` in the float opens the note in that nvim and closes the float.
- [ ] The README shows the snippet and the note format.

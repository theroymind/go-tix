---
type: feature
status: done
summary: A terminal file explorer for sketching, with plain .sketch.md notes beside the files they plan, shown as + or ~ in the repo tree, and ls and questions as JSON for agents.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T18:49:26Z
---

# Sketch explorer

## Description

`sketch` becomes a terminal file explorer for roughing out a feature before it's built. You browse the repo as you would in lazygit or a sidebar explorer, and to plan a file you drop a note beside it: `src/composables/useOrbit.ts.sketch.md`. A note for a file that doesn't exist yet shows in the tree as a ghost `+` node, and one for an existing file shows that file as `~`. Notes hold whatever you type, and a few optional markers give structure: `feature:` and `task:` in front matter link the note to tix, `#fn useOrbit target Ref<Vec3>` (or `#type`, `#component`, `#hook`, `#flow`, `#tune`, `#rule`, `#test`) names an item, `~` when the target already holds that name and `+` otherwise, and `? question` is an open question until a `> answer` line sits right under it. Claude reads and edits notes as plain files, and gets the overview from `sketch ls --json` and `sketch questions --json`. It works in any terminal, and LazyVim opens it in a float the way it opens lazygit.

It replaces the old sketch in place: same `sketch` binary, same `internal/sketch/` home. `contracts/sketch` is rewritten around `Note`, `Item` and `Question`, `refs` (tix feature lookup) and `marks` (whole-word name search) carry over, and the builder, grammar, YAML store, `merge` and liner are deleted, along with the old sketch epic's notes. The explorer is built on Bubble Tea, Lip Gloss and Bubbles, which are approved. The tree lists files through `git ls-files -co --exclude-standard`, so ignored folders like `node_modules` stay hidden, and walks the directory skipping hidden folders outside git. `e` opens a note in `$EDITOR`, or in the parent nvim through `--remote` when `$NVIM` is set, so nvim never nests inside nvim.

Decided along the way: notes are committed on the feature branch and deleted as each file is built, questions are answered by editing the note rather than through a command, and the explorer opens on the whole tree with `n` narrowing it to notes. Turned down: the command builder, which hides the tree while you type; a Neovim-only plugin, which ties the tool to one editor; and treesitter for symbol picking, for now, since a plain identifier scan needs no cgo.

## Acceptance criteria

- [x] `sketch` opens the explorer on the repo tree, with ignored files hidden, notes shown as `+` ghost nodes or `~` on their targets, and the selected note in a preview pane.
- [x] `n` toggles between the whole tree and notes only, with the folders above each note.
- [x] `a` on a folder creates a placeholder note, `A` on a file attaches one, both open it in the editor, and `d` deletes a note after you confirm.
- [x] `r` on an attached note lists the names in its target file and adds the one you pick to the note.
- [x] `f` filters the tree to one tix feature's notes, and `?` jumps between notes with open questions.
- [x] `L` on a note picks a feature and links the note to it in its front matter.
- [x] `sketch ls --json` lists each note with its target, `+` or `~`, feature, task and items, and `sketch questions --json` lists the open ones.
- [x] `<leader>k` in LazyVim opens the explorer in a float, and `e` in it opens the note in that nvim.
- [ ] In sketch-trial, you sketch a feature's files in the explorer, and Claude builds from the notes after answering their questions.

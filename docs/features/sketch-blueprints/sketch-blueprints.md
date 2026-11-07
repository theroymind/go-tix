---
type: feature
status: active
summary: Sketch notes become free-form blueprints for agents, with links, a ready status, a paired nvim editor and a whole-feature plan view with a Mermaid graph.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T18:54:39Z
---

# Sketch blueprints

## Description

You sketch a feature by writing plain notes beside the files they plan, and Claude builds from them. Notes are blueprints for an agent to read, not a format sketch parses: `#fn`, `#type`, `#component`, `#flow` and `#test` are a suggested, language-neutral way to describe intent, documented in `sketch help`, and the whole body goes to agents as written. sketch reads only `feature:` and `status: draft|ready` in front matter, `? question` / `> answer`, and `-> path : label` links between files. From those it can tell whether a feature is ready to build, draw how its files connect, and hand Claude one brief.

The explorer pairs with a real nvim rather than opening an editor per edit: in LazyVim it opens as a left split, moving onto a note shows it in the main window, and `a`, `enter` or `e` focuses it there. Outside nvim it keeps its preview pane and `$EDITOR`. `a` normalizes whatever you type to `<target>.sketch.md`, allowing a name with no extension when the file type isn't settled. `r` picks a file to link to, `y` marks a note ready, `>` jumps to answer a question, `]`/`[` move between open questions, `?` shows help, and `v` shows the feature's notes joined in build order. `y` is always the human's last step.

Claude runs `sketch check --feature X`, which fails on draft notes, open questions and dangling links, then builds from `sketch plan --feature X`: markdown with a Mermaid graph, the notes in build order, and answered questions as decisions. Notes stay on the branch as the spec until `sketch done X` deletes them; marking the tix feature done is left to `implement`. `contracts/sketch` moves to schema version 2, dropping items, item marks, `task:` and `unknown_feature` and adding body, status and links; nothing outside go-tix imports it. tix becomes optional, with a warning. Turned down: an nvim embedded in the pane (needs a terminal-emulator dependency), an explicit `order:` key (until links prove out), and the map view for now, kept as an idea.

## Acceptance criteria

- [ ] A note with pseudo-code and no markers shows its whole body in `sketch ls --json`.
- [ ] Typing `useOrbit.ts`, `useOrbit.ts.sketch.md` or `useOrbit.sketch.ts` at `a` creates `useOrbit.ts.sketch.md`, and `useOrbit` creates `useOrbit.sketch.md`.
- [ ] In LazyVim, `<leader>k` opens sketch as a left split, and moving onto a note shows it in the main window.
- [ ] `r` adds `-> path` to a note, and the footer shows `uses →` and `used by ←` for the selected note.
- [ ] `y` turns a draft note ready, and `sketch check --feature X` passes only when every note is ready, every question answered and every link resolves.
- [ ] `sketch plan --feature X` prints a Mermaid graph GitHub renders, then the notes in build order.
- [ ] `v` shows the feature's notes joined in build order.
- [ ] `sketch ls` works in a repo with no `tix` on the PATH, with a warning.
- [ ] In sketch-trial, you sketch a feature in a paired nvim, mark it ready, and Claude builds it from `sketch plan`.

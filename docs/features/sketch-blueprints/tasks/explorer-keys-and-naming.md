---
type: task
status: todo
feature: sketch-blueprints
scope: medium
review: none
depends_on: [blueprint-notes]
summary: a normalizes names, enter/e create or open a note, ? shows help, ]/[ jump questions, > jumps to answer, and the feature filter sticks per branch.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T18:47:01Z
---

# Explorer keys and naming

## Description

`a` prompts `plan file: <folder>/` with folder tab-completion and normalizes what you type to `<target>.sketch.md`, stripping a stray `.sketch` or `.sketch.md`. `enter` or `e` on a file creates its note or opens the one it has, replacing `A`; `?` becomes a help overlay, `]`/`[` jump between questions, and `>` adds a `> ` line under the first open question and opens the editor on it. The feature filter is saved per branch under the git dir, and the explorer opens notes-only when notes exist. No review, since it's keys inside one package.

## Files

- internal/sketch/explorer/explorer.go
- internal/sketch/explorer/prompt.go
- internal/sketch/explorer/names.go
- internal/sketch/explorer/filter.go
- internal/sketch/explorer/settings.go
- internal/sketch/editor/editor.go
- internal/sketch/explorer/explorer_test.go

## Acceptance criteria

- [ ] `useOrbit.ts`, `useOrbit.ts.sketch.md` and `useOrbit.sketch.ts` all create `useOrbit.ts.sketch.md`, and `useOrbit` creates `useOrbit.sketch.md`.
- [ ] `enter` on a file with no note creates one and opens it, and `A` is gone.
- [ ] `?` shows every key, and `]`/`[` move between notes with open questions.
- [ ] `>` adds `> ` under the first open question and opens the editor on that line.
- [ ] A feature filter set with `f` is still set after quitting and reopening sketch on the same branch.

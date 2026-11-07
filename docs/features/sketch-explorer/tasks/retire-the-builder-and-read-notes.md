---
type: task
status: done
feature: sketch-explorer
scope: large
review: pending
depends_on: []
summary: Delete the old builder, grammar and YAML store, and parse .sketch.md notes into the new contract, listed by sketch ls --json.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T03:26:50Z
---

# Retire the builder and read notes

## Description

Delete the old sketch, then rebuild its core around notes: `contracts/sketch` gets `Note` (target path, feature, task, items, questions) with `Validate`, `internal/sketch/note` parses a note file and maps its path to its target, and `sketch ls [--json]` finds every `*.sketch.md` under the repo root. The parser is deliberately lenient, so anything that isn't front matter, an item marker, or a `?` or `>` line is free text and never an error. It gets a review because it replaces a public contract and deletes most of the package tree.

## Files

- `contracts/sketch/sketch.go`
- `contracts/sketch/sketch_test.go`
- `internal/sketch/note/note.go`
- `internal/sketch/note/note_test.go`
- `internal/sketch/command/command.go`
- `internal/sketch/command/command_test.go`
- `go.mod`
- `docs/features/sketch/`
- `docs/ideas/share-the-yaml-and-atomic-write-helpers-across-apps.md`

## Acceptance criteria

- [x] `internal/sketch/{builder,grammar,store,merge,show}`, liner, the old sketch epic's notes and the YAML-helpers idea are gone, and `make check` passes.
- [x] A note parses into front matter, items (`#fn name rest`) and questions (`?` with an optional `>` answer), and any other line is free text.
- [x] `useOrbit.ts.sketch.md` maps to the target `useOrbit.ts` in the same folder.
- [x] `sketch ls` prints a table of notes and their targets, and `--json` prints notes that pass `Validate`.
- [x] `sketch help` documents the note format, so an agent can learn it from the binary.

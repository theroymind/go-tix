---
type: task
status: done
feature: sketch
scope: medium
review: passed
depends_on: [sketch-schema-and-store]
summary: Parse the verbs and their aliases into edits on a sketch, run them one-shot under -C, and print the sketch as a table, JSON or markdown.
created_at: 2026-11-06T02:46:19Z
updated_at: 2026-11-06T03:32:15Z
---

# One-shot grammar and show

## Description

A single parser turns a line like `fn score_strike` or `in actor Combatant_Id` into an edit on the sketch at a context. Both modes go through it, so the builder can never drift from the one-shot commands. `-C fight-camera/score-strikes/client/camera.odin` sets the context for a one-shot run. It gets a review because the parser and context rules carry the edge cases.

## Files

- `cmd/sketch/main.go`
- `internal/sketch/command/command.go`
- `internal/sketch/command/command_test.go`
- `internal/sketch/grammar/grammar.go`
- `internal/sketch/grammar/grammar_test.go`
- `internal/sketch/grammar/context.go`
- `internal/sketch/grammar/settings.go`
- `internal/sketch/show/show.go`
- `internal/sketch/show/show_test.go`
- `internal/sketch/store/store.go`
- `internal/sketch/store/store_test.go`
- `internal/sketch/store/lock_unix.go`
- `internal/sketch/store/lock_other.go`

## Acceptance criteria

- [x] The parser accepts each verb and its alias (`f`, `t`, `i`, `o`, `c`, `a`, `r`), and rejects a line verb outside an item or an item verb outside a file, naming the context it needs.
- [x] `sketch -C REF/TASK/FILE VERB ARGS` adds, renames or removes an item or line, and saves.
- [x] `sketch show FEATURE` prints a table of tasks, files and items, and `--json` and `--md` print the same sketch for a program and a reader.

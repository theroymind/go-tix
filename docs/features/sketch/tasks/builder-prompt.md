---
type: task
status: done
feature: sketch
scope: medium
review: passed
depends_on: [one-shot-grammar-and-show]
summary: An interactive prompt over the grammar with a context stack, history, tab completion, undo and edit in $EDITOR.
created_at: 2026-11-06T02:46:19Z
updated_at: 2026-11-06T04:08:43Z
---

# Builder prompt

## Description

`sketch feature X` opens a prompt that holds the context stack and feeds each line through task 2's parser, saving after every edit. Tab completion offers verbs, repo paths after `file`, existing names, and tix tasks after `task`. `undo` steps back through the session's saves. It adds `github.com/peterh/liner`, which is approved, and it gets a review because the context stack and undo carry state.

## Files

- `internal/sketch/builder/builder.go`
- `internal/sketch/builder/builder_test.go`
- `internal/sketch/builder/session.go`
- `internal/sketch/builder/complete.go`
- `internal/sketch/builder/settings.go`
- `internal/sketch/merge/merge.go`
- `internal/sketch/merge/merge_test.go`
- `internal/sketch/command/command.go`
- `internal/sketch/refs/refs.go`
- `internal/sketch/grammar/context.go`
- `contracts/sketch/sketch.go`
- `go.mod`

## Acceptance criteria

- [x] The prompt shows the current context, `..` pops one level, `/` returns to the feature, and an item verb opens that item.
- [x] Up-arrow history and tab completion of verbs, paths, names and tix tasks work.
- [x] `undo` reverts the last edit of the session, and `edit` opens the current item's YAML in `$EDITOR`, keeping the change only when it validates.

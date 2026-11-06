---
type: task
status: todo
feature: sketch
scope: medium
review: true
depends_on: [one-shot-grammar-and-show]
summary: An interactive prompt over the grammar with a context stack, history, tab completion, undo and edit in $EDITOR.
created_at: 2026-11-06T02:46:19Z
updated_at: 2026-11-06T02:46:19Z
---

# Builder prompt

## Description

`sketch feature X` opens a prompt that holds the context stack and feeds each line through task 2's parser, saving after every edit. Tab completion offers verbs, repo paths after `file`, existing names, and tix tasks after `task`. `undo` steps back through the session's saves. It adds `github.com/peterh/liner`, which is approved, and it gets a review because the context stack and undo carry state.

## Files

- `internal/sketch/builder/builder.go`
- `internal/sketch/builder/builder_test.go`
- `cmd/sketch/main.go`

## Acceptance criteria

- [ ] The prompt shows the current context, `..` pops one level, `/` returns to the feature, and an item verb opens that item.
- [ ] Up-arrow history and tab completion of verbs, paths, names and tix tasks work.
- [ ] `undo` reverts the last edit of the session, and `edit` opens the current item's YAML in `$EDITOR`, keeping the change only when it validates.

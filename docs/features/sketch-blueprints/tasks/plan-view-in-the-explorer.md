---
type: task
status: done
feature: sketch-blueprints
scope: medium
review: none
depends_on: [sketch-plan-and-graph]
summary: v shows the feature's files beside every note joined in build order.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T19:18:32Z
---

# Plan view in the explorer

## Description

`v` swaps the tree for the plan view: the feature's files on the left with their marks and link counts, and on the right the notes joined in build order under the tix summary, scrolled to the selected file. It reuses `plan`'s ordering, and `v` or `esc` goes back. No review, since it's a view over sketch-plan-and-graph.

## Files

- internal/sketch/explorer/planview.go
- internal/sketch/explorer/explorer.go
- internal/sketch/explorer/explorer_test.go

## Acceptance criteria

- [x] `v` shows the files on the left and the notes joined in build order on the right.
- [x] Moving through the file list scrolls the right pane to that note.
- [x] `v` or `esc` returns to the tree on the same node.

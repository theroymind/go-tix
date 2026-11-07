---
type: task
status: todo
feature: sketch-blueprints
scope: medium
review: pending
depends_on: [blueprint-notes]
summary: sketch plan prints a feature's brief as markdown with a Mermaid graph and notes in build order, and sketch graph prints the graph alone.
created_at: 2026-11-07T18:47:01Z
updated_at: 2026-11-07T18:47:01Z
---

# sketch plan and graph

## Description

`sketch plan [--feature X]` prints the feature's summary and acceptance criteria from tix when tix is there, a Mermaid `flowchart LR` with one subgraph per folder, new and changed nodes styled apart and labelled links, the notes in build order, and answered questions as decisions; `sketch graph --mermaid` prints the graph alone. Build order is a topological sort over links that falls back to path order inside a cycle. It gets a review for that ordering and for Mermaid escaping.

## Files

- internal/sketch/plan/plan.go
- internal/sketch/plan/plan_test.go
- internal/sketch/command/command.go
- internal/sketch/command/command_test.go

## Acceptance criteria

- [ ] `sketch graph --mermaid` prints a flowchart that renders on GitHub, with a subgraph per folder.
- [ ] `sketch plan --feature X` lists a note after every note it links to.
- [ ] A cycle of links still prints every note once, in path order.
- [ ] Answered questions appear under a decisions heading.

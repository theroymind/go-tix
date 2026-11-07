---
type: task
status: todo
feature: sketch-explorer
scope: medium
review: pending
depends_on: [retire-the-builder-and-read-notes]
summary: Mark notes and items + or ~ against their targets, list open questions, and resolve a note's feature through tix.
created_at: 2026-11-07T03:20:52Z
updated_at: 2026-11-07T03:21:03Z
---

# Marks, questions and tix features

## Description

Add what makes `ls` useful to an agent: each note is `+` when its target is missing and `~` when it exists, each item is marked the same way through `marks`, and `sketch questions [--json]` lists the open questions with their note path and line. Inside a tix repo, `refs` checks each note's `feature:` against tix, and `ls` flags one that doesn't resolve rather than failing, so a typo doesn't hide the rest of the sketch. It gets a review because it crosses into tix through a child process.

## Files

- `internal/sketch/marks/marks.go`
- `internal/sketch/questions/questions.go`
- `internal/sketch/questions/questions_test.go`
- `internal/sketch/refs/refs.go`
- `internal/sketch/refs/refs_test.go`
- `internal/sketch/command/command.go`

## Acceptance criteria

- [ ] `ls` shows `+` for a note whose target is missing and `~` for one whose target exists, and the same per item.
- [ ] `sketch questions --json` lists each open question with its note, line number and text, and leaves out answered ones.
- [ ] Inside a tix repo, a note whose `feature:` isn't a ticket is flagged in `ls`, and the other notes still list.
- [ ] Outside a tix repo, any `feature:` value is accepted.

---
type: task
status: done
feature: sketch
scope: small
review: passed
depends_on: [one-shot-grammar-and-show]
summary: Read a batch of sketch commands from stdin under -C, saved once, and lock the repo root instead of the feature's directory.
created_at: 2026-11-06T03:43:32Z
updated_at: 2026-11-06T03:44:53Z
---

# Stdin batches and a root lock

## Description

Claude sends several sketch -C calls in parallel, and each one reads, edits and writes sketch.yml, so overlapping runs depend on the lock to keep every edit. A batch read from stdin applies every command in one process, in order and all or nothing, the way a builder session reads, so the normal path has nothing to overlap. The lock stays as a backstop for the builder and other sessions, and moves to the repo root, which always exists, so the store no longer creates and removes the feature's directories around a refused edit.

## Files

- `internal/sketch/command/command.go`
- `internal/sketch/command/command_test.go`
- `internal/sketch/store/store.go`
- `internal/sketch/store/store_test.go`
- `internal/sketch/store/lock_unix.go`

## Acceptance criteria

- [x] sketch -C CONTEXT - reads one command per line from stdin, each starting at the context the one before it left, and saves once.
- [x] A batch with a failing line writes nothing and names the line number and its error.
- [x] store.Update locks the repo root, and a refused edit creates no directories.

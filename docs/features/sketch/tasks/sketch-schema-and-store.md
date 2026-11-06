---
type: task
status: todo
feature: sketch
scope: medium
review: true
depends_on: []
summary: The public sketch schema with Validate, and loading and saving a feature's sketch.yml atomically.
created_at: 2026-11-06T02:46:19Z
updated_at: 2026-11-06T02:46:19Z
---

# Sketch schema and store

## Description

This adds the sketch's data model as a public contract, so tools other than sketch, like Herdr, can read it, along with the store that reads and writes it. Items keep one generic shape across languages: a type, a name and ordered typed lines, rather than a struct per language. It gets a review because it's a new public type.

## Files

- `contracts/sketch/sketch.go`
- `contracts/sketch/sketch_test.go`
- `internal/sketch/store/store.go`
- `internal/sketch/store/store_test.go`

## Acceptance criteria

- [ ] `contracts/sketch` holds the Sketch, Task, File, Item and Line types, each with `Validate`. That rejects an unknown item or line type, an empty name, and a duplicate item name in one file.
- [ ] The store loads and saves `sketch.yml` in a feature's directory, `docs/features/<feature>/` from the repo root by default, writing through a temp file and a rename.
- [ ] A saved sketch loads back unchanged, and the YAML keeps the order you typed.

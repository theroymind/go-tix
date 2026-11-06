---
type: feature
status: todo
summary: A sibling CLI to tix that roughs out a feature's files, types and functions through short commands, in a builder prompt or one-shot, stored as YAML that Claude reads, challenges and builds against.
created_at: 2026-11-06T02:46:19Z
updated_at: 2026-11-06T02:46:19Z
---

# Sketch

## Description

`sketch` lets you rough out the shape of code before it's built: the files, types, functions, inputs, outputs and open questions. You type short commands instead of writing markdown. `sketch feature fight-camera` opens a builder prompt that keeps track of where you are (feature, task, file, item). So `file client/camera.odin`, then `fn score_strike`, `in actor Combatant_Id`, `out Camera_Move` and `ask who anchors a kill`, build up the sketch a line at a time. Every builder command also runs one-shot, so Claude reads the sketch as JSON, answers its questions and adds counter-items through the same grammar. The settled sketch becomes the contract a task is built against, in any language: Odin here, React at work.

It lives in `cmd/sketch` inside the go-tix module, with its packages under `internal/sketch/`. The schema is public in `contracts/sketch`, and `railyard-cli` handles the args, subcommands and JSON. Each feature's sketch is one YAML file at `sketches/<feature>.yml`, holding tasks, then files by repo path, then items. An item has a type (`type`, `fn`, `component`, `hook`, `flow`, `tune`, `rule` or `test`), a name, and typed lines (`field`, `in`, `out`, `prop`, `case`, `ask`, `rule`). Searching the target file for the item's name marks it new or changed. Inside a tix repo, `sketch feature X` resolves X as a ticket and offers its child tasks. The builder's history and tab completion come from `github.com/peterh/liner`, the one new dependency, which needs approval before task 4.

Decided along the way: one file per feature, so the view across tasks sits in one place, to be tested in use. The builder and one-shot modes share one grammar, so only `undo` is builder-only, since it steps back through the session. Change marks are detected rather than typed, and a sketch is deleted with its feature. Turned down: sketches written by hand inside task notes, which bloat the task and say nothing about where files go, and a mirrored tree of markdown files, which is still syntax to remember.

## Acceptance criteria

- [ ] `sketch feature X` opens a builder whose prompt shows the current context, with `..` and `/` to move up.
- [ ] Builder and one-shot commands, with `-C feature/task/file` setting the context, write the same YAML.
- [ ] An item whose name already appears in its target file shows as changed. Any other item shows as new.
- [ ] `sketch questions --json` lists the open questions, and `sketch answer` resolves one.
- [ ] `sketch show` prints a table, and also JSON with `--json` and markdown with `--md`.
- [ ] Inside a tix repo, the feature must resolve as a tix ticket, and its tasks tab-complete. Outside one, any name works.
- [ ] In odin-tbc, you sketch fight-camera task 2 in the builder in a few minutes without opening an editor, and Claude answers its questions through `sketch answer`.

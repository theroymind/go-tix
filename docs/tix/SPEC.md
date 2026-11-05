# Spec: tix v1

> Imported from tix v1 on 2026-10-25, then developed inside Railyard and split back out into go-tix on 2026-11-04. The `.tix/` format, commands and rules here are what tix keeps compatible with. The build commands and code layout below describe v1, not the current repo. See `CLAUDE.md` for those.

## Objective

tix is a per-repo CLI ticket tracker. Tickets are YAML files in `.tix/` at the repo root, committed with the code. Humans and coding agents both use it: `tix ready` answers "what do I pick up next?".

v1 is done when tix can hold its own remaining work and the balance-sim slice 1 breakdown (README), including the hierarchy, dependencies and checkpoints, and `ready` returns the right next ticket.

## Tech Stack

- Go 1.26.3, module `github.com/theroymind/go-tix`
- Standard library only (`flag`, `os`, `path/filepath`, `text/tabwriter`, `encoding/json`, `crypto/rand`, `slices`, `errors`)
- `go.yaml.in/yaml/v3` v3.0.5, the only dependency. It's already in the module cache, so no download is needed.

## Commands

```text
make build                 # go build -o tix ./cmd/tix
make test                  # go test ./...
go test ./internal/graph   # one package
go vet ./...
gofmt -l .                 # must print nothing
```

## Storage

```text
<repo>/.tix/
  config.yml        prefix: bs
  bs-k3f9.yml       one file per ticket, named <id>.yml
```

- Commands find `.tix/` by walking up from the working directory, the way git finds `.git`. If none is found, the error is `not in a tix repo (run tix init)`.
- Writes are atomic: write `<id>.yml.tmp`, then `os.Rename`.
- Field order in the file follows the struct order. Empty optional fields are omitted, so diffs stay small.

## Ticket schema

```yaml
id: bs-k3f9
title: Purchasables and greedy-cheapest policy
type: task
status: open
priority: 1
tags: [engine, slice1]
dependencies: [bs-2m7q]
parent: bs-x8vd
description: |
  Free-form markdown.
acceptance:
  - Policy buys the cheapest affordable purchasable each tick
created_at: 2026-10-14T21:00:00-04:00
updated_at: 2026-10-14T21:00:00-04:00
```

| Field | Rule |
| --- | --- |
| `id` | `<prefix>-<4 chars>`. The chars come from the ID characters `0123456789abcdefghjkmnpqrstvwxyz` (lowercase, no `i l o u`). |
| `title` | Required and non-empty after trimming. |
| `type` | Required: `epic`, `story` or `task`. |
| `status` | `draft`, `open`, `in_progress`, `completed` or `canceled` (already in `internal/ticket/status.go`). |
| `priority` | An integer from 0 (highest) to 3. Defaults to 2. |
| `tags` | Optional list of strings. |
| `dependencies` | Optional list of ticket IDs. They must exist, can't include the ticket itself, and can't form a cycle. |
| `parent` | Optional ticket ID. It must exist, the type rule must allow the pairing, and the parent chain can't form a cycle. |
| `description` | Optional markdown. |
| `acceptance` | Optional list of strings. |
| `created_at`, `updated_at` | Set by tix. `updated_at` changes on every write. |

### Hierarchy rule

| Parent type | Allowed child type |
| --- | --- |
| `epic` | `story` |
| `story` | `task` |
| `task` | `task` (a subtask is a task whose parent is a task, at any depth) |

Any type may be a root ticket (no parent).

### Resolution

- A status is **resolved** when it's `completed` or `canceled`.
- A dependency is satisfied when the ticket it points to is resolved.
- **Ready** means status `open`, every dependency satisfied, and no unresolved children. Leaves surface first, so an epic doesn't show up as "ready" while its stories are still open.
- **Blocked** means status `open` or `in_progress` with at least one unsatisfied dependency.

### IDs

- On `add`, tix generates 4 random characters with `crypto/rand`. Each random byte maps to the ID characters with `byte % 32`. Since 32 divides 256, there's no bias. On a collision it retries, up to 10 attempts, and then fails.
- Every command that takes an ID accepts a **reference**, resolved in this order:
  1. An exact ID.
  2. If the reference doesn't start with `<prefix>-`, prepend it (so `k3f9` → `bs-k3f9`).
  3. A unique prefix match (`bs-k3` or `k3`).

  No match fails with `no ticket matches "k3"`. More than one match fails with `"k3" is ambiguous: bs-k3f9, bs-k3aa`.

## CLI

Flags come before positional arguments, because Go's `flag` stops at the first non-flag argument. All commands follow the `runX(arguments []string) error` shape from emmerplan.

| Command | Behavior |
| --- | --- |
| `tix init --prefix bs` | Creates `.tix/config.yml` in the working directory. The prefix must match `^[a-z][a-z0-9]{0,7}$`. Fails if `.tix/` exists. |
| `tix init` | Without `--prefix`, asks on standard input for the prefix, the backend, and how to search. On the vault backend it also asks for the docs directory (`root`). The search answer is `haiku` (the default), which writes `claude: {model: haiku}`, `none`, which writes no search settings, or an embeddings endpoint URL, which then asks for its model and writes `embeddings`. The vault profile takes the default `type_property` and `criteria_heading`, one `task` ticket note type with the statuses `todo=open`, `in_progress=in_progress` and `done=completed`, and no doc types. Edit `.tix/config.yml` afterwards to change them. A blank answer takes the default shown in brackets. An invalid answer is asked again. Writes nothing if the input ends early. |
| `tix add --type task [--parent REF] [--priority N] [--tag T]... [--dependency REF]... [--acceptance A]... [--description D] [--draft] "title"` | Validates, saves, and prints only the new ID. |
| `tix show REF [--json]` | Prints the ticket as YAML, the same as the file. On a backend with docs, a reference that matches no ticket shows the doc, as `tix doc show` does. |
| `tix ls [--status S] [--type T] [--tag T] [--parent REF] [--all] [--json]` | Table of `ID TYPE STATUS PRIORITY TITLE`. Hides resolved tickets unless `--all` or `--status` is given. |
| `tix dep REF DEPENDENCY_REF` / `tix undep REF DEPENDENCY_REF` | Adds or removes a dependency. `dep` rejects cycles and names the path. |
| `tix ready [--type T] [--json]` | Ready tickets, sorted. |
| `tix blocked [--json]` | Blocked tickets, each with the IDs blocking it. |
| `tix start REF` / `tix done REF` / `tix cancel REF` | Sets the status to `in_progress` / `completed` / `canceled`. |
| `tix reparent REF PARENT_REF` / `tix reparent --root REF` | Changes the parent and runs the parent validation. |
| `tix edit REF` | Copies the ticket to a temp file, opens `$EDITOR` (falling back to `vi`), then reloads and validates. If it's valid it saves. If it's invalid, it prints the error, leaves the original untouched, and prints the temp path. Changing `id` isn't allowed. |
| `tix tree [REF] [--all]` | Indented hierarchy, two spaces per level: `ID TYPE STATUS TITLE`. Hides resolved subtrees unless `--all`. |
| `tix check` | Validates every ticket and reference (dangling IDs, type rule, cycles). Prints each problem and exits 1 if there are any. |

- **Sort order** everywhere: priority ascending, then `created_at`, then ID.
- **Exit codes:** 0 on success, 1 on failure, 2 on usage errors (`exitUsage` already exists).
- **`--json`** emits an array (or a single object for `show`) using the same field names as the YAML.

### Railyard extensions

These commands and rules are Railyard's, beyond v1. Files they write still match v1's format: field order, flow lists for `tags` and `dependencies`, and empty optional fields omitted.

| Command | Behavior |
| --- | --- |
| `tix dep REF DEPENDENCY_REF...` / `tix undep REF DEPENDENCY_REF...` | v1's `dep` and `undep`, taking several references and writing once. `dep` skips a dependency the ticket already has. `undep` removes a dependency by the exact id held, so one on a deleted ticket can go, or by a reference to a held ticket. Anything else fails, naming it. |
| `tix set REF KEY=VALUE...` | A raw write to a ticket or doc. `KEY=[a, b]` replaces the value with that list, and `KEY=[]` empties it (on `.tix/` files an empty `tags` list drops the key). A wikilink such as `[[slug]]` stays text, and any other value is a scalar. `dependencies` and `acceptance` are refused: change them with `dep`/`undep` and `ac set`/`ac add`. |
| `tix unset REF KEY...` | Removes each key, all of them or none. A required field (`.tix/`: `id`, `title`, `type`, `status`, `priority`, `created_at`, `updated_at`; a vault note: `status` and its type property) is refused, and so is a key the file doesn't hold, naming it. On a vault ticket, `description` is the note's body, not a property, so `unset` refuses it. On a doc, it removes only the frontmatter key. |
| `tix tick` / `untick` / `ac set` / `ac add` | Tick, untick, reword and add acceptance criteria by the number `show` prints. |
| `tix context REF [--depth N] [--json]` | Walks breadth first from the ticket or doc, default 2 hops, over its parent, dependencies, outgoing wikilinks and backlinks (the yaml backend: parent and dependencies only), and prints one card per note at its shortest distance: id, type, title, summary, `has_summary`, distance, relation (`start`, `parent`, `dependency`, `wikilink` or `backlink`: the hop that first reached the note, strongest first when one note reaches it several ways), commits and files. The Profile's `summary_property` (default `summary`) and `commits_property` (default `commits`) name the properties, and `files_heading` the section whose comma-separated or listed paths are the files; with no `files_heading`, files are empty. A note whose type is in the Profile's `context_stop_types` (default none; each must be a ticket or doc type), such as a vault's home note, gets its card but the walk doesn't go on through it, unless it is REF. `--json` is a bare array. |
| `tix index` | Embeds each ticket's and doc's title and summary (the title alone when there is no summary) through the `embeddings` endpoint in `.tix/config.yml` (`endpoint`, the base URL that `/embeddings` is appended to, such as `http://localhost:11434/v1`, and `model`) and writes the vectors, with the model's name, to one JSON file per repository under `${XDG_STATE_HOME:-~/.local/state}/railyard/index/`, keyed by the repository's directory name and a hash of its path. Only a note whose title or summary changed since the last run is embedded again, a deleted note drops out, and a changed model rebuilds the whole index. When the endpoint fails it exits 1 naming the endpoint and leaves the previous index as it was. Prints `indexed N notes (M embedded) at PATH`. When the config names `claude` instead, it exits 1, since that search needs no index. |
| `tix search QUERY [--type T] [--top N] [--json]` | When `.tix/config.yml` names `claude` (`model`, such as `haiku`), sends QUERY and every kept note's id, type, title and summary to `claude -p` with that model and no tools. The model scores the notes that match from 1 to 3, and a hit's score is that number divided by 3. Ids the model makes up are dropped, and an answer that isn't the JSON asked for exits 1. Otherwise it embeds QUERY through the same endpoint and prints the N notes (default 10) closest to it by cosine similarity over the index, best first, ties by id, as a `SCORE ID TYPE SUMMARY` table. `--type` keeps one note type, or a ticket type and the project's names for it. It exits 1 when `.tix/config.yml` names neither `embeddings` nor `claude`, when there is no index (`run tix index`), when the index was built with another model than the config names, or when the endpoint is down, naming it. `--json` is a bare array of `{id, type, title, summary, has_summary, score}`. |
| `tix lint (REF... \| --all)` | Checks each ticket and doc note the references name, or every one with `--all`, by plain-code rules: `section`, each heading the Profile's `required_sections` lists for the note's type (a map of ticket or doc type to heading texts, matched exactly at any level; default none) is present; `summary`, the `summary_property` is not empty; `link`, every slug the note names through a wikilink or a link property such as a parent or dependencies is a note's slug or another file's name under the vault root. Prints `PATH: RULE: MESSAGE` for each problem, PATH relative to the directory holding `.tix`, and exits 1 when there is any, 0 when there is none. A note that wouldn't map onto a ticket or doc is still checked. |
| `tix gate (REF... \| --staged)` | The pre-commit entry point. With `--staged`, takes the ticket and doc notes `git diff --cached` lists as added or modified under the working directory; else the REFs. Runs `tix lint` on them and exits with its code. It reads the working tree's text of each note, not the staged blob. Nothing staged passes. |

## Project Structure

```text
cmd/tix/main.go          dispatch switch, usage, exit codes
cmd/tix/<command>.go     one runX per command (init.go, add.go, ...)
internal/ticket/         pure types + single-ticket rules: Ticket, Status, Type, Priority, ID generation, Sort
internal/store/          filesystem: Find, Init, Load, Save, Resolve
internal/graph/          pure multi-ticket rules: Index, parent validation, cycles, Ready, Blocked, Tree, Check
docs/                    SPEC.md, PLAN.md, intent/
```

`graph` and `ticket` never touch the filesystem. `store` never makes graph decisions. Commands glue them together.

## Code Style

Match `internal/ticket/status.go`:

```go
type Type string

const (
	TypeEpic  Type = "epic"
	TypeStory Type = "story"
	TypeTask  Type = "task"
)

func (ticketType Type) IsValid() bool {
	switch ticketType {
	case TypeEpic, TypeStory, TypeTask:
		return true
	default:
		return false
	}
}
```

- No comments, except a genuinely non-obvious "why".
- Named constants for every semantic value (statuses, types, file names, limits).
- Full-word names (`dependencies`, not `deps`). Booleans read as questions (`IsValid`, `IsResolved`, `CanHaveChild`).
- `type`, never `kind`.
- Errors are sentinel vars (`var ErrTitleRequired = errors.New("title is required")`), wrapped with `fmt.Errorf("...: %w", ...)` and checked with `errors.Is`.

## Testing Strategy

- `testing` from the standard library, table-driven like `status_test.go`, with tests next to the code.
- Test real logic only: validation, the hierarchy rule, ID generation and collision retry, reference resolution, walk-up discovery, load/save round-trip, cycle detection, ready/blocked, tree ordering and check.
- Store tests use `t.TempDir()`. ID tests use a deterministic `io.Reader` (`bytes.NewReader`).
- Derive expected values from the constants (for example `MaximumIDAttempts`) rather than hardcoding numbers.
- CLI commands are verified manually with the command listed in each plan step. There are no CLI tests.

## Boundaries

- **Always:** `make test`, `go vet ./...` and `gofmt -l .` stay clean after every step. Validate before every save.
- **Ask first:** adding any dependency, changing the schema or a field name, changing a decision in this spec.
- **Never:** have tix run git, write outside `.tix/`, add code comments that restate the code, or use `kind`.

## Success Criteria

1. In a temp dir, `tix init --prefix t`, then add an epic → story → task → subtask chain. `tix tree` shows it indented, and `tix add --type task --parent <epic>` is rejected.
2. Two tasks where B depends on A: `ready` shows A only. After `done A`, it shows B. `dep A B` is rejected as a cycle.
3. `show`, `start`, `done` and `dep` accept a 2-character reference when it's unique.
4. A hand-broken file (dangling parent) makes `tix check` exit 1 and name the ticket.
5. The balance-sim slice 1 is filed in `balance-sim/.tix`, and `tix ready` there returns ticket 1 (engine core model types).

## Deferred (out of scope for v1)

- Git integration. tix never stages or commits.
- A compiled single YAML of all tickets and relations.
- Status rollup (auto-completing a parent when its children finish).
- Deleting tickets (delete the file by hand, then run `tix check`).
- Status transition rules. Any status can move to any status.

## Open Questions

None blocking. Each of these has a default; change it in this spec if you disagree:

- `canceled` counts as resolved, so it unblocks dependents.
- `ready` excludes tickets with unresolved children.
- `ls` and `tree` hide resolved tickets by default.

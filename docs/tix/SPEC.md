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
| `status` | `draft`, `open`, `in_progress`, `in_review`, `completed` or `canceled` (in `contracts/ticket/status.go`). `in_review` is a Railyard extension, not in v1: built and being reviewed. Like `in_progress` it's neither ready nor resolved. |
| `priority` | An integer from 0 (highest) to 3. Defaults to 2. |
| `tags` | Optional list of strings. |
| `dependencies` | Optional list of ticket IDs. They must exist, can't include the ticket itself, and can't form a cycle. |
| `parent` | Optional ticket ID. It must exist, the type rule must allow the pairing, and the parent chain can't form a cycle. |
| `description` | Optional markdown. |
| `acceptance` | Optional list of strings. |
| `external_id`, `external_url` | Optional. A Railyard extension, not in v1. `external_id` is the ticket's id in an outside tracker, such as `INO-343`. `external_url` is a link to it there, and must be an `http` or `https` URL with a host. In a file they go after `parent`. A ticket without them writes no key, so its file matches v1. |
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
| `tix init --prefix bs [--backend vault\|yaml] [--docs DIR]` | Creates `.tix/config.yml` in the working directory. The prefix must match `^[a-z][a-z0-9]{0,7}$`. Fails if `.tix/` exists. The backend defaults to `vault`, which writes the starter vault (see [Writing tickets](#writing-tickets)) rooted at `--docs`, `docs` by default, a directory inside this one. `--backend yaml` writes the prefix alone, as tix v1 did, and refuses `--docs`. |
| `tix init` | Without `--prefix`, asks on standard input for the prefix and the backend, `vault` by default. On the vault backend it also asks for the docs directory (`root`), and writes the starter vault there. A blank answer takes the default shown in brackets. An invalid answer is asked again. Writes nothing if the input ends early. |
| `tix add --type task [--parent REF] [--priority N] [--tag T]... [--dependency REF]... [--acceptance A]... [--description D] [--draft] "title"` | Validates, saves, and prints only the new ID. |
| `tix show REF [--json]` | Prints the ticket as YAML, the same as the file. On a backend with docs, a reference that matches no ticket shows the doc, as `tix doc show` does. |
| `tix path REF` | Prints the absolute path of the file holding the ticket, so another app, such as sketch, can find what sits beside it without reading the backend's layout. |
| `tix ls [--status S] [--type T] [--tag T] [--parent REF] [--all] [--json]` | Table of `ID TYPE STATUS PRIORITY TITLE`. Hides resolved tickets unless `--all` or `--status` is given. The `PRIORITY` column shows only when a listed ticket has a priority. On a vault, each property in the Profile's `list_properties`, such as `scope`, is a column before `TITLE`, with a dash for a ticket that lacks it; `--json` is unchanged. The table lists each ticket after its parent, its `ID` indented two spaces per level, so an epic's tasks and their subtasks nest under it; a ticket whose parent isn't listed starts a new tree. `--json` keeps the flat order. `tix list` is an alias. |
| `tix dep REF DEPENDENCY_REF` / `tix undep REF DEPENDENCY_REF` | Adds or removes a dependency. `dep` rejects cycles and names the path. |
| `tix ready [--type T] [--parent REF] [--json]` | Ready tickets, sorted, in the table `ls` prints. `--parent` keeps the ones whose parent is REF. |
| `tix blocked [--json]` | Blocked tickets, each with the IDs blocking it. |
| `tix start REF` / `tix review REF` / `tix done REF` / `tix cancel REF` | Sets the status to `in_progress` / `in_review` / `completed` / `canceled`. On a vault, it writes the note status the type maps to that status, and refuses one the type doesn't map. |
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
| `tix add ... [--external-id ID] [--external-url URL] TITLE` | v1's `add`, which also takes the ticket's external id and URL. `show --json` and `ls --json` print them as `external_id` and `external_url` when a ticket has them. `set` and `unset` take them as keys. |
| Vault external fields | A vault ticket's `external_id` and `external_url` are frontmatter properties. A ticket type's `fields:` map can name the property that holds each one, as it does for `parent` and `dependencies`, such as `fields: {tracker_id: external_id}`. With no property mapped, the property is named after the field: `external_id` or `external_url`. Their values are plain text, not links, so `tix lint` doesn't check them. |
| Vault description | `tix add --description` writes into the section under the Profile's `description_heading` (default `Description`) when the template has one, and under the note's H1 otherwise. |
| Vault note placement | A new ticket note goes in the directory `ticket_directories` names for its type, and a new doc in its doc type's `directory`. A `ticket_directories` entry can hold `{slug}`, the new note's slug, and `{parent}`, its parent note's directory, or the Profile's `uncategorized_directory` for a note with no parent; any other placeholder, or one that leads outside the vault root, is rejected. Without an entry, a note goes where most notes of its type already are, the first by name on a tie, or in the vault's root when the type has none yet. |
| Vault parent | `tix set REF parent=ID`, or the property the Profile maps onto the parent such as `feature=ID`, makes another ticket the note's parent, refusing an id that matches no ticket and the note's own. When the type's `ticket_directories` entry gives a different directory under the new parent, the note moves there, and each directory it leaves empty is removed. |
| `tix tick` / `untick` / `ac set` / `ac add` | Tick, untick, reword and add acceptance criteria by the number `show` prints. `tick REF --all` and `untick REF --all` mark every criterion in one write; numbers with `--all` are a usage error. |
| `tix ac ls [--open] REF` | Prints each criterion as `N. [x] TEXT` or `N. [ ] TEXT`, numbered as `tick` takes them, or with `--open` only the unticked ones, keeping their numbers. A backend whose criteria carry no done state shows each unticked. `acceptance-criteria` is `ac`'s long name for every subcommand. |
| `tix start` / `review` / `done REF...` | v1's status commands, taking several references. Every reference resolves before any ticket changes. `done` checks every ticket's criteria first and, without `--force`, changes none while any is unmet, naming each ticket and criterion. |
| `tix add ... [--file PATH]... [--set KEY=VALUE]... TITLE` | `--file` lists each path, one per item, under the Profile's `files_heading`, in place of the template's text; a backend without one refuses it before creating anything. `--set` writes each pair as `tix set` does. A malformed pair creates nothing; the files and pairs go in one write after the ticket is created, and a pair the ticket refuses fails naming the created id. |
| `tix show REF --with-parent [--json]` | Shows the ticket, then its parent, each under a `==> ID <==` line, as `head` prints several files; with `--json`, an array of the two. A ticket without a parent shows alone, as plain `show` does. |
| `tix help [COMMAND]`, `-h`, `--help` | `-h` or `--help` before a command, or `help` alone, prints the full usage. `-h` or `--help` among a command's arguments, before any `--`, or `help COMMAND`, prints only that command's usage lines, without running it. A usage error is followed by the command's usage, not the full one. |
| `tix context REF [--depth N] [--json]` | Walks breadth first from the ticket or doc, default 2 hops, over its parent, dependencies, outgoing wikilinks and backlinks (the yaml backend: parent and dependencies only), and prints one card per note at its shortest distance: id, type, title, summary, `has_summary`, distance, relation (`start`, `parent`, `dependency`, `wikilink` or `backlink`: the hop that first reached the note, strongest first when one note reaches it several ways), commits and files. The Profile's `summary_property` (default `summary`) and `commits_property` (default `commits`) name the properties, and `files_heading` the section whose comma-separated or listed paths are the files; with no `files_heading`, files are empty. A note whose type is in the Profile's `context_stop_types` (default none; each must be a ticket or doc type), such as a vault's home note, gets its card but the walk doesn't go on through it, unless it is REF. `--json` is a bare array. |
| `tix lint (REF... \| --all)` | Checks each ticket and doc note the references name, or every one with `--all`, by plain-code rules: `section`, each heading the Profile's `required_sections` lists for the note's type (a map of ticket or doc type to heading texts, matched exactly at any level; default none) is present; `template`, no required section's text is still, word for word, the text under that heading in the template the type starts from; `summary`, the `summary_property` is not empty; `link`, every slug the note names through a wikilink or a link property such as a parent or dependencies is a note's slug or another file's name under the vault root. Prints `PATH: RULE: MESSAGE` for each problem, PATH relative to the directory holding `.tix`, and exits 1 when there is any, 0 when there is none. A note that wouldn't map onto a ticket or doc is still checked. |
| `tix gate (REF... \| --staged)` | The pre-commit entry point. With `--staged`, takes the ticket and doc notes `git diff --cached` lists as added or modified under the working directory; else the REFs. Runs `tix lint` on them and exits with its code. It reads the working tree's text of each note, not the staged blob. Nothing staged passes. |

## Writing tickets

`tix init` sets up a starter vault: `feature` notes as epics, each in its own `features/<feature>/` directory, `task` notes as tasks in that feature's `tasks/` directory, with `feature` as the parent and `depends_on` as the dependencies, and `idea` docs in `ideas/`. Feature and task notes say `status: todo`, `in-progress`, `in-review` or `done`, for `open`, `in_progress`, `in_review` and `completed`. A task with no feature goes in `features/uncategorized/tasks/`. Each starts from a template that init writes into `<root>/templates/`, refusing before it writes anything when one is already there. The templates' source is `internal/tix/starter/templates/`. The profile sets `files_heading: Files` and `list_properties: [scope, review]`, and requires the sections below. Edit `.tix/config.yml` and the templates to fit the project. Each part of a ticket has one job, and nothing says twice what another part already says.

| Part | Job | Read by |
| --- | --- | --- |
| `summary` | One sentence saying what the note is about. | `tix context` cards, lists |
| `## Description` | A brief for whoever builds it. A feature says what the user gets, why it's worth it, how it works and what was decided, in at most three short paragraphs. A task says what changes, why, and the one decision that isn't obvious, in two to four sentences. Prose only, no lists. | The builder, human or agent |
| `## Files` | The paths a task expects to touch, one per item. The Profile's `files_heading` names this section. | `tix context`, as each card's `files` |
| `## Acceptance criteria` | The checklist and the progress model. Each criterion is one thing you can see or test, small enough to tick as soon as it lands, listed in roughly the order it gets built. Three to seven per task. A check by hand is a criterion, such as "In the boss fight you see the camera glide over the mage's shoulder". | `tix tick`, `tix done`, and progress views that count ticked criteria |

- **Criteria, not subtasks, show progress.** Tick each criterion when it lands, not all at the end, so a progress view moves while the work does. Make a subtask only when a piece needs its own dependency edge, its own review, or could be built in parallel.
- **A description doesn't restate the criteria.** If a sentence would also pass as a criterion, make it one and cut it from the description.
- **Properties hold facts a program reads.** Scope, review flags and links are properties, set with `tix set` and `tix dep`. The reason behind one goes in the description in half a sentence.

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

1. In a temp dir, `tix init --prefix t --backend yaml`, then add an epic → story → task → subtask chain. `tix tree` shows it indented, and `tix add --type task --parent <epic>` is rejected.
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

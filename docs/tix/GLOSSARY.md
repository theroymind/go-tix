# tix

> Imported from tix v1 on 2026-10-25, then developed inside Railyard and split back out into go-tix on 2026-11-04. tix uses these terms. Code paths mentioned here are from v1.

A per-repo CLI ticket tracker. Tickets live as YAML files in `.tix/`, committed with the code they describe.

## Language

### Tickets

**Ticket**:
One unit of tracked work, stored as `.tix/<id>.yml`.
_Avoid_: issue, item, card

**Type**:
A ticket's level in the hierarchy: `epic`, `story` or `task`.
_Avoid_: kind, category, level

**Status**:
Where a ticket is in its life: `draft`, `open`, `in_progress`, `completed` or `canceled`.
_Avoid_: state

**Priority**:
An integer from 0 (highest) to 3. Lower numbers sort first.
_Avoid_: severity, rank

**Prefix**:
The short repo-wide code that starts every ID (`bs` in `bs-k3f9`), set by `tix init`.
_Avoid_: namespace, project key

**ID**:
A ticket's full identifier: `<prefix>-<4 chars>`.
_Avoid_: key, number

**Reference**:
What a user types to name a ticket: an exact ID, an ID without its prefix, or a unique prefix of either.
_Avoid_: ref (in code), handle, selector

### Hierarchy

**Parent**:
The one ticket a ticket sits under. The hierarchy rule decides which types may pair.
_Avoid_: owner, container

**Child**:
A ticket whose parent is the ticket in question.
_Avoid_: sub-ticket

**Subtask**:
A task whose parent is a task.
_Avoid_: sub-item

**Root ticket**:
A ticket with no parent.
_Avoid_: top-level ticket, orphan

**Leaf**:
A ticket with no unresolved children.

### Dependencies and readiness

**Dependency**:
A ticket that must be resolved before the ticket listing it can be ready.
_Avoid_: deps, blocker (for the stored link), prerequisite

**Satisfied**:
A dependency is satisfied when the ticket it points to is resolved.

**Resolved**:
Status `completed` or `canceled`.
_Avoid_: done, closed, finished

**Ready**:
Status `open`, every dependency satisfied, and no unresolved children.
_Avoid_: available, next, actionable

**Blocked**:
Status `open` or `in_progress` with at least one unsatisfied dependency.
_Avoid_: stuck, waiting

**Checkpoint**:
An ordinary task that depends on the work before it and marks a verifiable milestone.
_Avoid_: milestone, gate

## Relationships

- A **Ticket** has at most one **Parent** and any number of **Dependencies**.
- A **Ticket** is **Ready** only once every **Dependency** is **Satisfied** and it is a **Leaf**.
- **Parent** and **Dependency** are separate links: parent is the hierarchy, dependency is ordering.

## Flagged ambiguities

- "dependency" also means a Go module (`go.yaml.in/yaml/v3`) in SPEC.md's Tech Stack and Boundaries. Resolved: in code and tickets, **Dependency** means the ticket link; call a Go one a "module".
- "checkpoint" also names the `### Checkpoint N` milestones in `docs/PLAN.md`. Resolved: those are plan sections, not tickets; a **Checkpoint** in `.tix/` is a ticket.
- Parent as a dependency type (`child_of` / `parent_of`). Resolved: the hierarchy lives only in **Parent**, never as a **Dependency**.
- `DependencyType` in `internal/ticket/dependency.go` adds link types the spec doesn't define (`relates`, `clones`, ...). Unresolved: SPEC.md treats every dependency as blocking.

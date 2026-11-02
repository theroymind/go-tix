---
type: home
---

# odin-tbc vault

The project's design notes, as an Obsidian vault. Open `docs/` as the vault.

- [[intent]]: the confirmed intent.
- [[pillars]]: the gates every new feature must pass before it gets a spec.
- [[combat-loop]]: the first spec, the encounter loop the prototype exists to test.
- [[mana-pool]] and [[enemy-goals]]: the second and third specs.
- [[architecture]]: principles, module map, core types and the turn.
- [[ui-library]]: the `ui` widget library carried over from odin-proto.
- Phases, in order: [[first-fight]], [[enemy-puzzle]].
- [[tasks.base]] and [[decisions.base]]: queryable tables of every task and decision.

## Notes and properties

Every note is named by a kebab-case slug and is referred to by a wikilink to that name, such as `[[turn-order-is-a-wait-counter]]`. A note name is the only identifier, so decisions and tasks carry no numbers.

| Folder | `type` | Properties |
|---|---|---|
| `decisions/` | `decision` | `status` (`proposed`, `accepted`, `deferred`, `superseded`), `decided_by` (`user`, `agent`, `coordinator`), `date`, `tasks`, and `superseded_by` when superseded |
| `tasks/` | `task` | `status` (`todo`, `done`), `phase`, `scope` (`small`, `medium`, `large`), `depends_on` |
| `phases/` | `phase` | `status`. The body lists the phase's tasks in order and ends with its checkpoint |
| `ideas/` | `idea` | brainstorms that led to a decision |
| `pillars.md` | `pillars` | `status`. One note: the gates a feature passes before `spec` |

New notes start from `templates/`. A decision holds one choice, with what it rejected under `## Rejected`. When a decision changes, write a new note, set the old one's `status: superseded` and `superseded_by`, and leave its text as it was. A decision lists the tasks it shapes in `tasks`, and a task's backlinks show its decisions, so the link is stored once. A task is `done` once every acceptance criterion is ticked.

Outside Obsidian, query the properties with grep, for example `grep -l 'status: proposed' docs/decisions/*.md`, or find a note's backlinks with `grep -rl '\[\[note-name' docs`.

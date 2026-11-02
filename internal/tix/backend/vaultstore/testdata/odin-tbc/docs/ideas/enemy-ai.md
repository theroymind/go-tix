---
type: idea
status: open
---

# Enemy AI: targeting, behaviour and planning

A brainstorm logged 2026-11-01 while specifying [[mana-pool]], which is the first spec to give an enemy a choice between two actions. Layers 1 and 2 are now specced in [[enemy-goals]]; the planner stays unspecced. It records where enemy choice could grow, so the [[mana-pool]] shape leaves room for it without building it.

## Where enemy choice stands

`choose_enemy_command` in `sim/enemy_choice.odin` switches on enemy type and returns one `Attack`. Targeting is already a small rule made of a `Target_Stat` and a `Target_Preference`. [[mana-pool]] turns both into data on the enemy definition: a targeting rule, which gains a weighted roll, and a behaviour profile, an ordered list of actions with the first affordable one chosen.

## Three layers, smallest first

1. **Targeting rule:** who to hit. Data: stat, preference, or a weighted roll. In [[mana-pool]].
2. **Behaviour profile:** what to do, as a personality. Starts as an ordered action list. Later grows weights or modifiers such as aggressive, cautious or support-first, applied multiplicatively so that three layers of modifiers stay predictable. The goals stay generic and the profile supplies the personality.
3. **Planner:** GOAP or scripted patterns, for bosses. Not before the prototype has a boss.

## From the GOAP brainstorm, what carries over

- **Goals stay generic, profiles create personality.** Each enemy picks the goals it uses, and the profile scales their priority. A phase change swaps the modifiers. This maps onto the behaviour profile above and is the next step after an ordered list.
- **Scripted patterns compete with goals, not inside them.** A pattern is a fixed sequence, such as Bio, Lance, Full-Life, Flare, with its own priority. When its next step's preconditions fail, the goals take over, and the pattern resumes once they hold again. Resumption needs a decision: retry the interrupted step, skip to the next step, or reset the pattern. Skipping risks dropping a load-bearing step.
- **Replan only when invalidated:** the next step's preconditions fail, a high-priority reactive goal fires, the plan completes, or the world changes the plan's expected effects.
- **The planner reads a query over the world, not the world itself:** health percents, weaknesses, whether an action is affordable or on cooldown. In this codebase that is a set of read-only procs over `World`, like the timeline.
- **State stays copyable so the AI can simulate "what if".** `World` is already a plain value with no singletons, and the timeline already steps a copy forward. A planner gets lookahead by applying commands to a copy.
- **Edge cases to test:** every action unaffordable or on cooldown, and a permanent status that blocks the pattern's next step. An enemy must always have a legal action; in [[mana-pool]] attack is free, which guarantees one.

## What doesn't carry over

- A `BattleManager`, `TurnQueue`, event bus and per-combatant controllers: `apply_command`, the wait counter and the client's `fight` already own those roles. See [[turn-order-is-a-wait-counter]].
- A persistent `PlayerState` snapshotted into battle state: progression and persistence are out of prototype scope in [[intent]].
- Aeons, Banish, Silence, Stun, charging spells and cooldowns: each is its own mechanic to weigh against [[pillars]] first.

## Gates to weigh before any of this is specced

- **Readable before it resolves:** a planner whose choice the player cannot anticipate fails this. Patterns help, because they can be learned. A planned move may need a telegraph, such as the next action shown on the timeline.
- **The enemy is a puzzle:** behaviour profiles pass if each one asks a different answer of the party.
- **One system:** goals and patterns must emit ordinary `Command`s through `check_command`, never a side door.
- **Combat first:** a boss with patterns passes. Cross-fight learning does not.

---
type: task
status: done
phase: enemy-puzzle
scope: small
depends_on: []
---

# Sim queries

A `sim/queries.odin` of read-only questions over `World` that goals score from and that a planner, a damage preview or a telegraph can read later: `predicted_damage`, `is_killing_blow`, `missing_health_percent`, `has_buff` and `opponent_stat_spread`. Shape in [[enemy-goals]].

## Acceptance criteria

- [x] `predicted_damage(world, actor, action, target)` returns what `resolve_attack`, `resolve_cast` or `resolve_skill` would deal, through the existing damage procs, and a test applies the command to a copy and asserts the health drop matches.
- [x] `is_killing_blow` is true exactly when predicted damage reaches the target's health, tested at one below, equal and above.
- [x] `missing_health_percent` is 0 at full health and 100 at zero; `has_buff` reads `active_buffs`; `opponent_stat_spread` returns the lowest and highest value of a `Target_Stat` among living opponents.
- [x] Nothing in `queries` takes a `^rand` state or writes `World`.

## Verification

Test and Check pass. Manual check: none.

## Files likely touched

sim/queries.odin, sim/queries_test.odin

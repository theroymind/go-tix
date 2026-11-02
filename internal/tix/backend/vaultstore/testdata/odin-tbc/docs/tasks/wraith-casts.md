---
type: task
status: todo
phase: first-fight
scope: medium
depends_on: [mana, enemy-behaviour-profiles]
---

# Wraith casts

The Wraith casts Ice while it can afford it and attacks once dry. Enemy definitions gain a spell set that `can_cast` reads like a role's, and the Wraith's profile becomes Ice by a weighted roll on health, then Attack by highest wait. Shape in [[mana-pool]].

## Acceptance criteria

- [x] `can_cast` reads the spell set of a role or an enemy type; the Wraith's is {Ice}, and Golem and Harpy have none.
- [x] A Wraith with enough MP casts Ice on a rolled party member and spends its cost; with less than the cost, it attacks the party member with the highest wait.
- [x] The determinism test's scripted fight includes Wraith casts and still matches on both worlds.

## Verification

Test and Check pass. Manual check: the user watches the Wraith's MP bar drain over its casts and sees it switch to attacks when empty.

## Files likely touched

sim/roster.odin, sim/balance.odin, sim/enemy_choice_test.odin, sim/determinism_test.odin

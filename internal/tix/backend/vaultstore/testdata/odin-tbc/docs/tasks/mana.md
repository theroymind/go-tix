---
type: task
status: done
phase: first-fight
scope: medium
depends_on: [sim-actions]
---

# Mana

Every combatant carries MP and a maximum, full at the start of the fight, and every action but attack and defend spends it. `check_command` refuses what the actor cannot afford. MP was weighed against [[pillars]] and passed; the gates and shape are in [[mana-pool]].

## Acceptance criteria

- [x] `Combatant` carries `mana` and `max_mana`, filled from each role's and enemy type's maximum in `balance.odin`; Golem and Harpy have 0. `#assert`s hold the mage and Wraith to about twelve casts and the warrior and ranger to three to five skills.
- [x] `action_mana_cost` reads each spell's, buff's and role skill's own cost from `balance.odin`, with attack and defend at 0 and the heal within one cost step of the elemental spells; `can_afford` compares it with the actor's MP.
- [x] `apply_command` subtracts the cost, and `check_command` refuses an unaffordable command with `Not_Enough_Mana` without changing the world.

## Verification

Test and Check pass. Manual check: none until [[mana-hud]].

## Files likely touched

sim/balance.odin, sim/world.odin, sim/roster.odin, sim/actions.odin, sim/commands.odin

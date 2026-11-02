---
type: phase
status: active
---

# First fight

Get one playable encounter on screen and judge it against [[combat-loop]].

1. [[turn-order-and-timeline]]: wait counters scaled by speed, next actor, timeline on read, with tests.
2. [[roster-and-favoured-types]]: the three roles, the enemy types, favoured-type and element multipliers in one formula.
3. [[enemy-choice]]: enemy actions chosen in the sim by a per-type rule.
4. [[fight-scene]]: flat-shaded low-resolution 3D scene, camera cuts, attack animation.
5. [[fight-hud]]: timeline widget, health bars, favoured types, action menu.
6. [[skills-and-buffs]]: spells, a heal, one skill per role and two buffs.
7. [[combat-screen]]: the HUD reshaped to the combat-screen design over a gridded floor.
8. [[disabled-controls]]: `ui` calls an unusable control disabled.
9. [[sim-actions]]: one `sim.Action` behind the command menu.
10. [[enemy-turn-command]]: enemies act through `Enemy_Turn`, recorded in `last_action`.

   Checkpoint: Test and Check pass, and a played fight looks the same as before.

11. [[enemy-behaviour-profiles]]: enemy choice as data, with targeting rules.
12. [[mana]]: MP on every combatant, spent and checked in the sim, per [[mana-pool]].
13. [[mana-hud]]: MP bars and costs, with unaffordable actions disabled.

   Checkpoint: Test and Check pass, and the mage runs dry in a played fight.

14. [[wraith-casts]]: the Wraith casts Ice until dry, on a weighted roll.

Checkpoint: play the same fight twice. If the second time still felt like deciding, write the progression intent. If not, say which pillar the loop failed.

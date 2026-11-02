---
type: task
status: todo
phase: enemy-puzzle
scope: medium
depends_on: ["[[goal-scoring]]"]
---

# Weighted step choice

`Behaviour_Step` gains `weights` and `Enemy_Definition` a `disposition`. `choose_enemy_action` scores every affordable step whose command passes `check_command` and takes the highest, ties to the earlier step, in place of the first-affordable walk. A `Weighted_Roll` step is scored on `scoring_target` and rolled only once it has won. Today's three profiles get weights and dispositions that reproduce today's choices, so play does not change in this task. Shape in [[enemy-goals]].

## Acceptance criteria

- [x] `scoring_target` returns the `Pick` target for a `Pick` rule and the lowest-stat candidate for a `Weighted_Roll`, tested on both.
- [ ] The argmax takes the higher score, the earlier step on a tie, and falls through to Attack when every other step is unaffordable or fails `check_command`; each case has a test on a constructed world.
- [ ] Only the winning step draws from the random state, tested by comparing the state after a choice to one draw.
- [ ] A parity test runs the determinism test's scripted fight and asserts every `last_action` matches the pre-change choices, and the roster test still asserts Attack in every profile.

## Verification

Test and Check pass. Manual check: none, play is unchanged.

## Files likely touched

sim/roster.odin, sim/targeting.odin, sim/enemy_choice.odin, sim/enemy_choice_test.odin, sim/balance.odin, sim/determinism_test.odin

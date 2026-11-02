---
type: task
status: done
phase: enemy-puzzle
scope: medium
depends_on: [sim-queries]
---

# Goal scoring

`sim/goals.odin`: the `Goal` enum, its `Goal_Category`, the `Disposition` enum, `Goal_Weights`, `Goal_Context`, `score_goal` and `score_step`. Every number comes from `balance.odin`. Shape in [[enemy-goals]].

## Acceptance criteria

- [x] `score_goal` returns an integer in `0 ..= GOAL_SCORE_MAXIMUM` for every goal and asserts it; tests check each goal at both ends and a midpoint: Finish_Weak on a kill, on full health and on half health; Protect_Ally on a target holding the buff, on a full-health unbuffed ally and on a hurt one; Punish_Exposed at the preferred extreme, the other extreme and with all opponents equal.
- [x] `score_step` is the sum over goals of `score_goal × weight × DISPOSITION_PERCENTS[disposition][goal_category(goal)] / 100`, tested with a zero weight ignoring a goal and with Aggressive and Cautious moving an offensive step in opposite directions.
- [x] `goal_category` maps Finish_Weak and Punish_Exposed to Offensive and Protect_Ally to Supportive; `Balanced` is 100 for both categories, asserted in `balance.odin`.

## Verification

Test and Check pass. Manual check: none.

## Files likely touched

sim/goals.odin, sim/goals_test.odin, sim/balance.odin

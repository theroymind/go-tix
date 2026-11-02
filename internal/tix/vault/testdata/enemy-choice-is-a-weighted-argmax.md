---
type: decision
status: accepted
decided_by: user
date: "2026-11-01"
tasks: [sim-queries, goal-scoring, weighted-step-choice, golem-guards, harpy-hastes]
---

An enemy chooses the affordable, legal behaviour step with the highest score, ties to the earlier step. A step's score is the sum over a fixed set of generic goals of the goal's score times the step's weight for it times the enemy's disposition percent for that goal's category. Goals read the world only through read-only query procs in `sim/queries`, and a `Weighted_Roll` step is scored on its lowest-stat candidate with the real target rolled only once the step has won, so no random draw ever decides the action. `Finish_Weak` scores on predicted damage through the existing formulas, so "would this kill" is the rule the player learns.

## Rejected

- roll the target first and score the rolled target: simpler, but the roll could flip the action, which fails Readable before it resolves
- dropping the Wraith's roll so all targeting is a `Pick`: removes the seam but reverses a [[mana-pool]] decision
- queries inside `goals`: one file fewer, but the planner, a damage preview and a telegraph will read the same procs, so the boundary is visible from day one
- `Finish_Weak` on raw HP percent: no dependency on combat formulas, but the Golem would guard with a kill available
- disposition as a per-goal percent table: more knobs, and each new goal needs a column in every disposition
- disposition replacing step weights: fewest numbers, but no per-step emphasis, so Wraith parity would need a disposition of its own

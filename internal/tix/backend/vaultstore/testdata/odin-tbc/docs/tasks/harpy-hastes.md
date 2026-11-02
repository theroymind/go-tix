---
type: task
status: todo
phase: "[[enemy-puzzle]]"
scope: small
depends_on: [weighted-step-choice]
---

# Harpy hastes

The Harpy gains a `Give_Buff Haste` step on an ally weighted on Protect_Ally, before its Attack weighted on Punish_Exposed by defense, with a Cautious disposition and an MP pool that buys two to three Hastes. It hastes an unhasted living ally while it can afford to, then attacks the lowest-defense member. Shape in [[enemy-goals]].

## Acceptance criteria

- [x] `HARPY_MANA` is above zero and `balance.odin` asserts `HARPY_HASTE_BUDGET_MINIMUM <= HARPY_MANA / HASTE_MANA_COST <= HARPY_HASTE_BUDGET_MAXIMUM`; the zero-MP assert now covers the Golem only.
- [x] On a constructed world the Harpy hastes an unhasted ally, prefers a hurt one over a full one, attacks when every living ally is hasted, and attacks when dry; each is a test.
- [x] The Haste step's targeting is a `Pick` on allies by lowest health, the actor included, scored on the lowest-health ally lacking Haste, so the Harpy skips a hasted ally for an unhasted one and hastes itself when it is the only unhasted ally; a test covers each case.
- [x] The determinism test's scripted fight includes a Harpy haste and still matches on both worlds.

## Verification

Test and Check pass. Manual check: the user sees the Harpy haste the Golem or Wraith early, the Haste tag appear on that row and the enemy climb the timeline, then the Harpy switch to attacks once its MP bar is empty.

## Files likely touched

sim/roster.odin, sim/balance.odin, sim/targeting.odin, sim/enemy_choice_test.odin, sim/determinism_test.odin

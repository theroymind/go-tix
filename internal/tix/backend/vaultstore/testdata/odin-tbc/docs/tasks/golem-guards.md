---
type: task
status: todo
phase: enemy-puzzle
scope: small
depends_on: [weighted-step-choice]
---

# Golem guards

The Golem gains a Defend step weighted on Protect_Ally beside its Attack weighted on Finish_Weak, with an Aggressive disposition. It guards itself when no party member is in killing range and no Guard is held, and attacks the lowest-HP member otherwise. Shape in [[enemy-goals]].

## Acceptance criteria

- [x] On a constructed world with a party member in killing range, the Golem attacks that member; with none in range and no Guard, it defends; with none in range and Guard held, it attacks the lowest-HP member.
- [x] `GOLEM_ATTACK_WEIGHTS` and `GOLEM_DEFEND_WEIGHTS` live in `balance.odin`, and an assertion there states that a killing-blow Attack outscores Defend under the Golem's disposition.
- [x] The determinism test's scripted fight includes a Golem guard and still matches on both worlds.

## Verification

Test and Check pass. Manual check: the user sees the Golem guard, with its Guard tag on its row, when nobody is low, and go for a low member when one is.

## Files likely touched

sim/roster.odin, sim/balance.odin, sim/enemy_choice_test.odin, sim/determinism_test.odin

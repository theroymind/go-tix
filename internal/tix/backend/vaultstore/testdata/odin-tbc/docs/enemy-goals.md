---
type: spec
status: confirmed
---

# Enemy goals

The third spec, after [[mana-pool]]. Each regular enemy chooses its action by scoring a few generic goals against the current world, scaled by a per-type profile, so the Golem, Harpy and Wraith each demand a different answer from the party. Grown from [[enemy-ai]], whose boss layer stays parked.

## Objective

The three enemies feel the same: each takes one attack at one target by a fixed rule. The player has nothing to anticipate and nothing to answer. This spec gives each enemy a second action and a personality made of weighted goals, so that after a few turns the player can say what each enemy wants and play against it. It also lays the priority system a later boss pattern competes with.

## Gates

- **Every turn is a choice:** passes if each enemy's goal creates a party decision, such as healing the low member before the Golem acts or spending a turn to strip the Harpy's Haste. Fails if the goals only reorder attacks.
- **Readable before it resolves:** passes if choice is a deterministic argmax over visible state, so a consistent rule can be learned by watching. No roll decides the action. The Wraith's weighted target roll stays as decided in [[mana-pool]]. No telegraph is added; see Open questions.
- **The enemy is a puzzle:** passes if each enemy has a named counter and ignoring it visibly costs the party. The Golem finishes the weak, answered by healing or guarding. The Harpy hastes allies and punishes the exposed, answered by killing it first. The Wraith casts at the hurt and attacks the slow, answered by outlasting its pool.
- **Combat first:** passes. Goals read the current world only and nothing persists.
- **One system:** passes. Enemies act through existing actions, `Defend` and `Give_Buff`, the existing MP pool and the existing `Targeting_Rule`. Goals replace only the first-affordable walk in `choose_enemy_action`. The goal scorer pays for itself on the boss, whose pattern will compete with these goals by priority.

## Success criteria

- [x] A goal is a generic rule that scores an action and target candidate from the world as an integer in a fixed range, the same range for every goal, with the range constants in `balance.odin`. The set is `Finish_Weak` (rises as a living opponent's HP falls, highest when the action would kill), `Protect_Ally` (rises when a living ally is hurt or lacks the buff the action gives, zero for a self-target that already holds it), and `Punish_Exposed` (rises with the opponent's wait or falls with its defense, as the step's targeting stat says).
- [x] A behaviour profile becomes a list of steps, each an action, a targeting rule and one integer weight per goal. Each enemy definition also carries a disposition, Aggressive, Cautious or Balanced, that scales offensive and supportive goals by a percent. The score of a step is the sum over goals of goal score times weight times the disposition percent for that goal's category. The enemy takes the highest-scoring affordable step that passes `check_command`, with ties broken by step order. Attack, always affordable, is in every profile and a roster test asserts it.
- [x] Choice is deterministic: for a given world, the same step is chosen every time, and the determinism test covers a fight in which every enemy type takes its non-attack action at least once.
- [x] The Golem has `Defend` and `Attack`. It guards itself when no opponent is in killing range and attacks the lowest-HP opponent otherwise. Counter: heal or guard the low member before its turn.
- [x] The Harpy has `Give_Buff Haste` on an ally and `Attack` on the lowest defense. It hastes an unhasted living ally while it can afford to, then attacks. Its MP maximum rises from 0 to a budget of two to three Hastes, asserted in `balance.odin`. Counter: kill the Harpy first or absorb a faster Golem.
- [x] The Wraith keeps `Cast Ice` by weighted roll on HP, then `Attack` on highest wait, now expressed as goal weights that reproduce the same choices. A test asserts its choices match today's for the same world and seed.
- [x] Every goal and weight constant lives in `balance.odin`. Profiles stay data on `Enemy_Definition` with no switch on enemy type in `choose_enemy_action`.
- [x] Enemy `Defend` and `Give_Buff` present in the client without an attack animation: the actor rests and the HUD shows the buff on the target as it does for the party.
- [x] Tests cover each goal's scoring at its ends and midpoint, the argmax and its tie break, every-step-unaffordable falling through to Attack, and each enemy's named behaviour on a constructed world. Expected values derive from `balance.odin`.

## Out of scope

- Boss and scripted patterns: the next feature, built on this priority system. Logged in [[enemy-ai]].
- GOAP or any multi-turn planning: goals score the current world only. Deferred to the boss feature, where patterns are tried first.
- Telegraphing the next enemy action on the timeline: the player learns the rule by playing. Revisit if the checkpoint says enemy moves feel unannounced.
- New command types, spells, elements or statuses such as silence, stun, cooldowns and charging: each is its own pillar pass.
- Exploit-weakness goal: party weaknesses do not exist, so it would score on favoured type alone. Added when party weaknesses are.
- Mid-fight disposition swaps and phase changes: a boss concern. The disposition field exists so that swap is one assignment later.
- Enemy use of `Use_Skill` and the heal: enemies have no role skill, and an enemy heal changes fight length, which [[mana-pool]] has not measured yet.

## Constraints

- `sim` stays deterministic and reads no wall clock. Action choice uses no random draw; only the Wraith's existing target roll touches `World.random_state`.
- Goals reach the world only through read-only query procs over `World`, so a later planner can run them on a copy.
- Every chosen action is built with `make_command` and passes `check_command`. No side door.
- Every number is in `balance.odin`: score range, per-goal weights per enemy, disposition percents, the Harpy's MP and its Haste budget assertions. The `GOLEM_MANA == 0 && HARPY_MANA == 0` assertion changes to match.
- Changing `MAXIMUM_BEHAVIOUR_STEPS` or `Enemy_Definition` is expected; adding fields to `Combatant` or `World` is not, and an implementation agent that needs one stops and reports.

## Open questions

- Should a profile's weights be visible to the player anywhere, such as a bestiary line on the enemy row? Guess: no, learning by watching is the point. Reconsider at the checkpoint.
- Does the Golem guarding itself read as "it saw nobody to finish", or as a dead turn? Guess: it reads once Guard shows on its row. If not, swap Defend for Empower on itself.
- Is a two-to-three Haste budget for the Harpy enough to matter? Guess: yes, since each Haste moves an enemy up the timeline for its duration. Measure at the checkpoint.
- Does one disposition per enemy plus per-step weights over-fit three enemies? Guess: no while tests pin behaviour rather than numbers. If tuning one breaks another, collapse step weights to a goal set and let disposition carry the weighting.

## Shape

Decided in [[enemy-choice-is-a-weighted-argmax]]. An implementation agent that needs a different shape stops and reports it.

### Modules

| Module | Depends on | Owns |
|---|---|---|
| `sim/queries` *(new)* | world, combat, actions, targeting | read-only questions over `World`: `predicted_damage`, `is_killing_blow`, `missing_health_percent`, `has_buff`, `opponent_stat_spread` |
| `sim/goals` *(new)* | queries, targeting, balance | `Goal`, `Goal_Category`, `Disposition`, `Goal_Weights`, `Goal_Context`, `score_goal`, `score_step` |
| `sim/targeting` | world, balance | `+= rule_stat`, `rule_preference` (a Weighted_Roll reads as Lowest), `scoring_target`: the deterministic target a rule is scored on; `pick_target_lacking_buff`: the Pick among candidates that do not hold a buff |
| `sim/roster` | goals, targeting, balance | `Behaviour_Step += weights`; `Enemy_Definition += disposition`; Golem gains a Defend step, Harpy a Haste step |
| `sim/enemy_choice` | goals, targeting, actions, commands | argmax over steps replaces the first-affordable walk |
| `sim/balance` | | `GOAL_SCORE_MAXIMUM`, `DISPOSITION_PERCENTS`, per-enemy step weights, `HARPY_MANA`, Haste budget asserts |

### Types

| Type | Shape | Notes |
|---|---|---|
| `Goal` | `enum: Finish_Weak \| Protect_Ally \| Punish_Exposed` | one `score_goal` switch, no proc table |
| `Goal_Category` | `enum: Offensive \| Supportive` | `goal_category(goal)`: Finish_Weak and Punish_Exposed offensive, Protect_Ally supportive |
| `Disposition` | `enum: Aggressive \| Cautious \| Balanced` | data on `Enemy_Definition`; a boss phase change swaps it later |
| `Goal_Weights` | `[Goal]int` | zero ignores the goal |
| `Goal_Context` | `{ world: ref World, actor, action: Action, target: maybe Combatant_Id, rule: maybe Targeting_Rule }` | the only surface a goal reads; target and rule nil for Defend; Punish_Exposed reads the rule's stat and preference through `rule_stat` and `rule_preference` |
| `Behaviour_Step` | `+= weights: Goal_Weights` | |
| `Behaviour_Profile` | unchanged, `MAXIMUM_BEHAVIOUR_STEPS` stays 2 | |
| `Enemy_Definition` | `+= disposition: Disposition` | Golem Aggressive, Harpy Cautious, Wraith Balanced |
| `score_goal(goal, context) → int` | `0 ..= GOAL_SCORE_MAXIMUM` | asserted in range |
| `score_step(context, weights, disposition) → int` | `Σ score_goal × weight × DISPOSITION_PERCENTS[disposition][category] / 100` | weights must be non-negative; zero skips the goal |
| `scoring_target(world, actor, side, rule) → Combatant_Id` | the Pick on `rule_stat` and `rule_preference` | |
| `pick_target_lacking_buff(world, actor, side, rule: Pick, buff) → maybe Combatant_Id` | the Pick skipping candidates that hold `buff`; nil when every candidate holds it | a Give_Buff step scores on it, falling back to `scoring_target`; a Give_Buff step must use a Pick |
| `Step_Candidate` | `{ index: int, target: maybe Combatant_Id, score: int }` | internal to `enemy_choice` |
| `predicted_damage(world, actor, action, target) → int` | attack, spell or skill through the existing formulas | no random state |
| `is_killing_blow(world, actor, action, target) → bool` | `predicted_damage >= target.health` | |

Balance constants:

- `GOAL_SCORE_MAXIMUM`, `PROTECT_ALLY_BASE_SCORE`
- `DISPOSITION_PERCENTS: [Disposition][Goal_Category]int`
- `GOLEM_ATTACK_WEIGHTS`, `GOLEM_DEFEND_WEIGHTS`, `HARPY_HASTE_WEIGHTS`, `HARPY_ATTACK_WEIGHTS`, `WRAITH_ICE_WEIGHTS`, `WRAITH_ATTACK_WEIGHTS`
- `HARPY_MANA`, `HARPY_HASTE_BUDGET_MINIMUM`, `HARPY_HASTE_BUDGET_MAXIMUM`, with the zero-MP assert narrowed to the Golem

### Flows

- **Enemy turn:** `apply_enemy_turn` → `choose_enemy_action` → for each step: skip if unaffordable → `scoring_target`, or for a Give_Buff step the Pick among allies lacking the buff → build the command → skip if `check_command` fails → `score_step` → keep a strictly higher score, so ties go to the earlier step → if the winner's rule is `Weighted_Roll`, roll the real target now → `Resolved_Action`.
- **Finish_Weak:** `is_killing_blow` gives the maximum; otherwise the maximum scaled by `missing_health_percent` of the target.
- **Protect_Ally:** the target is the ally, or the actor for Defend. Zero if the target already holds the buff the action gives; otherwise a base for a full-health unbuffed ally, rising with missing health.
- **Punish_Exposed:** the target's position in the living opponents' spread on the step's stat, maximum at the preferred extreme, maximum when all are equal.
- **Golem:** Attack weights Finish_Weak, Defend weights Protect_Ally, Aggressive. A reachable kill outscores guarding; no kill and no Guard held means it guards.
- **Harpy:** Haste weights Protect_Ally, Attack weights Punish_Exposed on defense, Cautious. The Haste step scores on the lowest-health living ally lacking Haste, itself included, which outscores attacking while MP lasts; a dry Harpy or all-hasted allies fall to Attack.
- **Wraith:** Ice weights Finish_Weak scored on the lowest-HP party member, target rolled after; Attack weights Punish_Exposed on wait, Balanced. A parity test compares to the old walk.
- **Determinism:** the only draw is the chosen step's roll, taken once, so the Wraith's draw sequence is unchanged.
- **Presentation:** `begin_action_presentation` already rests on Defend and Give_Buff, and enemy rows already draw buff tags.

### Seams

- Generic: `queries` serves goals now, and the planner, a damage preview on the target menu and a timeline telegraph later. `Goal`, `Goal_Weights` and `Disposition` serve the three enemies now and the boss's goal layer and phase swaps later. `Goal_Context` is the one surface a goal reads, so a planner can hand it a copied world.
- Specific: the argmax loop in `enemy_choice`. The boss adds a pattern that competes with it; extracting a `choose_by_priority` then costs one proc move.
- Flagged: `scoring_target` has one consumer and exists only because the Wraith rolls. If the roll goes, it folds back into `choose_target`.

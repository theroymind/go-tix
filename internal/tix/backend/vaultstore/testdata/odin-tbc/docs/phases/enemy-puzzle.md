---
type: phase
status: active
---

# Enemy puzzle

Make each enemy ask a different answer of the party, per [[enemy-goals]], then judge the fight against [[pillars]].

1. [[sim-queries]]: read-only questions over the world that goals score from.
2. [[goal-scoring]]: goals, categories, dispositions and the step score.
3. [[weighted-step-choice]]: the argmax replaces the first-affordable walk, play unchanged.

   Checkpoint: Test and Check pass, and a played fight looks the same as before.

4. [[golem-guards]]: the Golem guards when nobody is in killing range.
5. [[harpy-hastes]]: the Harpy hastes allies until dry.

Checkpoint: play the fight and name what each enemy wants and how you answered it. If an enemy's want cannot be named after three of its turns, it fails Readable before it resolves and the goals or the telegraph question in [[enemy-goals]] reopen. If naming it does not change your play, it fails Every turn is a choice and the counter is too weak.

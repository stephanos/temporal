---
satisfies: [R7, R5]
---
# fn-137-capabilities-read-phase-roles.2 Lifter reads the projection from the Phased parent

## Description
The IR generator learns to find the projection on the machine's or composition's `Phased[...](...)` parent, or a derived machine's source. It keeps the `Rules(...)` argument as a fallback until task 4. The lifted IR must be identical for both spellings.

**Size:** M
**Files:** model/irgen/Declarations.scala, model/irgen/Order.scala, model/irgen/Compositions.scala, model/irgen/Context.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/lifts/Rules.scala (+ new Phased fixtures and expected JSON)
**Touches:** [model/irgen/Declarations.scala, model/irgen/Order.scala, model/irgen/Compositions.scala, model/irgen/Context.scala, model/irgen/test/**, model/irgen/testdata/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `ruleSteps(machine, state, rules)` (Declarations.scala:424-426) reads `parentArguments(rules)`. Pass it the projection resolved from the machine object's parents instead, falling back to the `Rules` argument. Resolve a `Derived` object's projection from its source (as derived compositions read `parentArguments(c)`, Compositions.scala:201).
- `phaseOf` (:617-621) keeps its one-parameter-lambda check for the `Phased` argument.
- The refusal at :472-473 names `Phased[State, Phase](_.phase)`.
- Order.scala (:33-34, :86 `appliesAtOnce`, :175, :204-205) treats the `Rules` constructor as running the projection at once. Teach it that `Phased`'s argument is a constructor argument of the object, initialized before `rules`, and update the comments.
- Memory note (channel-catalogs-and-visible-results): carry the projection faithfully or refuse the form. Never fall back silently to "no projection".

### Investigation targets
**Required:**
- model/irgen/Declarations.scala:420-495, 610-625
- model/irgen/Order.scala:25-40, 80-90, 170-210
- model/irgen/Compositions.scala:195-240
- model/irgen/test/Fixtures.test.scala:160-170, 360-370, 500-525

**Optional:**
- model/irgen/testdata/crossed/Rules.scala: the fn-126 R16 "no projection" refusal fixture

### Acceptance
- [ ] A fixture machine written with `Phased` plus argument-less `Rules` lifts to JSON byte-identical to the same machine written with `Rules(_.phase)` (expected-JSON fixture shared by both).
- [ ] `in(...)` with no projection anywhere is refused, naming `Phased[State, Phase](_.phase)`, at its position.
- [ ] The lifter resolves a derived machine's and a derived composition's projection to its source's, for tasks 5 and 6. Derivation rules name no phase, so the test asserts the resolved projection directly.
- [ ] The initialization-order check accepts `Phased` plus `rules` and still refuses the reads it refused before.
- [ ] `make umpire-check-model` passes with no IR change.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R4, R5]
---
# fn-137-capabilities-read-phase-roles.1 Phased mixin; argument-less Rules reads it beside the old form

## Description
Adds the `Phased[S, P](projection)` mixin to the framework and lets `object rules extends Rules:` read the phase projection and type from it. `Rules(projection)` keeps working for now, so no Model or fixture changes yet. This is the early proof point: a given from a trait parent can pin `P` for argument-less `Rules`.

**Size:** M
**Files:** model/umpire/Syntax.scala, model/umpire/Machine.scala, model/umpire/Compose.scala, model/umpire/Rules.test.scala, model/check/SyntaxRule.scala
**Touches:** [model/umpire/Syntax.scala, model/umpire/Machine.scala, model/umpire/Compose.scala, model/umpire/Rules.test.scala, model/check/SyntaxRule.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `Phased` is author-surface sugar, so define it in Syntax.scala with a `Core form:` comment (enforced by check/SyntaxRule.scala:55; add `Phased` to its sugar list).
- Give it a protected given carrying the projection and `P`, modelled on `machineOwner` (Machine.scala:289) and `compositionOwner` (Compose.scala:50). It must work for both `Machine` and `Composition`; their common parent is `Declares[S]` (Machine.scala:80).
- `Rules` (Syntax.scala:266) resolves that given when no constructor projection is passed. While both exist, the constructor argument wins. `in(...)` (:322, :326) and the overlap check (`bind`, :298) call whichever applies.
- Update the `PhasesOf` implicitNotFound message (:353-358) to name `Phased[State, Phase](_.phase)`.
- `Derived` (Machine.scala:369) and `Composition(derivation)` (Compose.scala:43) refuse a `Phased` mixed into themselves at initialization, naming the object.
- Carry the phase type through derivations: `rebind` on a `Phased` machine and `withMember` on a `Phased` composition return a derivation typed with `P`, and `Derived(...)`/`Composition(derivation)` re-export the typed given. Task 6 depends on this to resolve role witnesses on `trustingActivityRecord` and the trusting/lossy compositions.
- Both `in(...)` forms (:322 phases, :326 named set) take evidence that `P` is not `Nothing`. Without it, `in(set: P => Boolean)` accepts any predicate on a non-`Phased` machine, because function inputs are contravariant.
- Preserve the existing comments. Update the doc comments on `Rules`, `Owner` (Machine.scala:144) and the machine example (:250-253) to the new form.

### Investigation targets
**Required:**
- model/umpire/Syntax.scala:240-365: `Rules`, `in`, `PhasesOf`
- model/umpire/Machine.scala:140-165, 268-300, 365-385: `Owner`, `Machine`, `Derived`
- model/umpire/Compose.scala:31-60, 101: `Composition`, `Composer`
- model/umpire/Rules.test.scala: the Switch/Overlapping/Unfelt fixtures

### Key context
The planning probe on Scala 3.9.0: `extends Machine[State], Phased(_.phase)` fails with `value phase is not a member of Any`, and `Phased[State, Phase](_.phase)` compiles. Use the explicit type arguments.

### Acceptance
- [ ] A `Phased[State, Phase](_.phase)` machine with `object rules extends Rules:` fires `in(p1, p2)` and `in(states.x)` exactly as `Rules(_.phase)` does (Rules.test.scala: same table for both forms).
- [ ] The overlap check refuses an overlap through the `Phased` projection with the same message as today.
- [ ] Both `in(p1, p2)` and `in(states.x)` in a non-`Phased` machine with argument-less `Rules` fail to compile, and the message names `Phased[State, Phase](_.phase)` (one compile-error test per form).
- [ ] A derived machine and a derived composition expose their source's projection with its phase type at compile time (a test summons the typed given on each).
- [ ] A derived machine or derived composition that mixes in `Phased` is refused at initialization, naming it.
- [ ] `mise exec -- scala-cli test model/umpire`, `make lint-model` and `make umpire-check-model` pass with no IR change.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

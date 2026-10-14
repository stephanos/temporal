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
Adds the `Phased[S, P](projection)` mixin. A machine that mixes it in can write `object rules extends Rules:`, and its `in(p1, p2)` and `in(states.x)` read the mixed-in projection. `Rules(projection)` still works, and where both are present the constructor argument wins. The early proof point holds: a given from a trait parent pins `P` for argument-less `Rules`, and the overlap check runs through it.

Lane C, commit e29b84c06 (base 0f25754d6). Tier: lane C, IMPLEMENTER claude-opus-5-5 at high.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at the batch end)

### What changed
- `model/umpire/Syntax.scala`: `trait Phased[S, P](projection) extends Declares[S]`, with a protected given `phased: Phasing[S, P]`, plus a companion that gives `Inherited` evidence for a Phased source. `Rules` now takes `(using owner, phasing: Phasing[S, ? <: P])(phase: S => P = phasing.projection)`. Both `in` forms take `PhasesOf` evidence, which requires `NotGiven[P =:= Nothing]`. The message reads "in names phases of Q, and these rules read phases of P: mix the projection the phases are of into the machine, `Phased[State, Phase](_.phase)`".
- `model/umpire/Machine.scala` (core; it cannot name `Phased`, per SyntaxRule):
  - `Phasing[S, P]` has a fallback given, `Phasing.unphased[S]: Phasing[S, Nothing]`.
  - `DerivedFrom[+M]` records a derivation's source.
  - `Inherited[-T, S, P]` is the evidence that a source projects onto `P`. Its instances: derivation chains, derived machines, and a low-priority `Unphased` fallback to `Nothing` that `Phased`'s companion extends.
  - `Declares.declaresPhase` lets core code see that an object mixes in Phased.
  - All derivations (`restrict`, `rebind`, `extend`, `refining`, `assuming`, `unmonitored`) now return `Machine[S, O, F] & DerivedFrom[this.type]`. `Built` carries its source.
  - `Derived[S, O, F, P](derivation)(using Inherited[derivation.type, S, P])` re-exports the source's typed `Phasing`, and it refuses `declaresPhase` at initialization.
  - Doc comments on `Owner` and the machine example now show the new form.
- `model/umpire/Compose.scala`: a derived composition, `Shape.Of`, that mixes in Phased is refused at initialization.
- `model/check/SyntaxRule.scala`: `Phased` is added to `sugarNames`.
- Tests are in `model/umpire/Rules.test.scala`:
  - One table compares `PhasedLamp` (`Phased` + `Rules:`) with `Projected` (`Rules(_.light)`) over every state, knob and action.
  - An overlap through `Phased` is refused with the existing message.
  - Two compile-error tests, one per `in` form, check that the message names `Phased[State, Phase](_.phase)`.
  - The typed given is summoned on `PhasedStuck` (a `rebind`) and on `PhasedStucker` (a `restrict` of a derived machine).
  - `Rephased` (a derived machine) and `Reprojected` (a derived composition) are refused at initialization.

### Deviations, exceptions, open items
- **Derived composition typed given: not built (DESIGN_CONFLICT, partial AC 4).** A derived composition's object type is `Composition[S]`. Its parents carry no trace of the `withMember` argument's static type, so no member of the object can be typed with the source's `P`. The only routes are these:
  - (a) Add a phase type parameter to `Composition`. That breaks about 50 explicit `Composition[S](...)` sites in Models and fixtures.
  - (b) Scala's `tracked` parameters. I tried this: `scala.language.experimental.modularity` marks every definition in the file `@experimental`, which spreads to the whole build unless the build compiles with `-experimental`.
  - (c) Change how derived compositions are declared, which is a Model-surface change.
  - I recommend (a) or (c), decided by the owner before task 6. Task 6 needs the typed given on the trusting and lossy derived compositions: `trustingRecordOverQueue`, `trustingRecordOverMatching` and `recordOverLossyMatching`.
  - The refusal half of the AC, a derived composition mixing in `Phased`, is built.
- **Derived machines: `Derived` gained a fourth type parameter `P`.** Inference fills it everywhere (`extends Derived(x.op(...))`). The only explicit spelling was `Rejects.scala`'s `LoopFirst`/`LoopSecond`.
- **Touches exceptions** (both necessary to keep the lifter fixtures green with no IR change):
  - `model/irgen/Declarations.scala`, two patterns. `objectMachine` accepts `List(List(derivation), _)`, since `Derived` now has a using clause. The `in(set)` heading accepts `List(List(set), _)`, since there is now an evidence clause.
  - `model/irgen/testdata/lifts/Rejects.scala`: `LoopFirst` spells `Derived[Lamp, Outcome, Nothing, Nothing]` across two lines, and `LoopSecond` is inferred. The blank line before the "Two machines derived from each other" comment was dropped so that every later line number is unchanged, including the `Rejects.scala:419` refusal in rejects.txt and 97 other references. Both files are in task .2's Touches, in this lane.
- The `in(set)` evidence only proves `P` is not `Nothing`. A non-Phased machine that keeps `Rules(_.light)` still names phases, as before.

### Expected IR delta
None. The Models are unchanged and the lifter reads the same trees, so the batch regeneration should show no lines from this task.

### Tests
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal`: 32 passed.
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`: 94 passed, no expected file changed.
- `make lint-model-syntax`, `lint-model-models`, `lint-model-check`, `lint-model-irgen-lifts` and `lint-model-irgen`: each rc=0.
- scalafmt `--check` over model/: rc=0.
- `scala-cli test model/check`: INCONCLUSIVE. 68 passed. The GateSuite "a check lints the settled IR..." test hit munit's 30s timeout twice. That test runs the gate four times, and this diff's only change in model/check is the sugar list.
- GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration

### For later tasks
- **Spelling:**
  - `object M extends Machine[S, O, F], Phased[S, P](_.phase)`. Write both type arguments; `Phased(_.phase)` does not compile.
  - Inside sections the given is `Phasing[S, P]` (core, `private[umpire] val projection`).
  - Phased exposes `private[umpire] val projection` as well.
- **Lifter (.2):** the parent tree is `Apply(TypeApply(Select(New(Phased), <init>), [S, P]), [lambda])`. The `Rules` parent is now `Rules(using owner, phasing)(proj)`, and argument-less `Rules` still has two clauses: the default argument is a call of `Rules.$lessinit$greater$default$...`, not a lambda. `ruleSteps` already finds the projection by searching for a lambda among the flattened arguments, so a default getter is simply absent. A derived machine's parent is `Derived(derivation)(using inherited)`.
- **Derived machines (.5/.6):** inside a `Derived` object, `summon[Phasing[S, P]]` resolves to the source's projection and type. That holds through `rebind`, `restrict`, `extend`, `refining`, `assuming` and `unmonitored`, and through a derived machine used as a source (for example `TrustingRecordMember` = `Derived(TrustingActivityRecord.unmonitored)`). Derived compositions have no typed given. See the deviation above and decide before task 6.
- A non-Phased object's sections see `Phasing[S, Nothing]` (`Phasing.unphased`), so a capability that needs a role witness will fail to resolve there. That is the intended compile-time refusal path.
- **Task .4** (retiring `Rules(projection)`): drop the explicit `phase` clause and the `? <: P` wildcard in `Rules`.
- `Phasing`, `DerivedFrom` and `Inherited` are core names in Machine.scala. Core files must keep from naming `Phased` (SyntaxRule).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a37d8f8b58
- Tests: baseline: green (mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire, pre-edit, lane-C clone), make model/build/model-scala.jar (packaging only) and scala-cli compile --print-class-path > model/build/model-scala.classpath (the gate's classpath file, for the irgen Fixtures), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal (32 passed), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (check mode: 94 passed, no expected file changed), make lint-model-syntax lint-model-models lint-model-check lint-model-irgen-lifts lint-model-irgen (each rc=0, one by one), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check (rc=0), INCONCLUSIVE: mise exec -- scala-cli test model/check: 68 passed, 1 failed twice on a 30s munit timeout (GateSuite 'a check lints the settled IR before the Go checks...', runs the gate 4 times with fake tools); unrelated to this diff (only SyntaxRule's sugar list changed there), not re-run on the base, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration
- PRs:
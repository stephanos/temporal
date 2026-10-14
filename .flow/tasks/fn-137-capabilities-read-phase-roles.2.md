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
The lifter now reads the phase projection from the machine's `Phased[...](...)` parent, and keeps the `Rules(...)` argument as a fallback until task 4. Reads of a Phased projection are recorded at the rules' declaration line. That is where `Rules(_.phase)` writes the projection, so both spellings lift to the same IR byte for byte. The projection of a derived machine or derived composition resolves to its source's.

Lane C, commit 880a8a8c1 (base e29b84c06). Tier: lane C, IMPLEMENTER claude-opus-5-5 at high.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at the batch end)

### What changed
- `model/irgen/Declarations.scala`:
  - `phaseProjection(cls, at)` returns the argument of the `Phased` parent, or follows a `Derived(m.op(...))` chain or a `Composition(c.withMember(...))` chain back to the source object's projection. It is guarded against cycles.
  - `ruleSteps(..., owner)` uses the rules' own lambda first and `phaseProjection(owner)` otherwise. A projection that comes from `Phased` is registered in `phaseReaders`, and `phaseOf` lifts it under `placing(pos(rules))`.
  - The no-projection refusal now reads "in names phases, and <machine> reads no phase projection: mix it into the machine, `Phased[State, Phase](_.phase)`".
  - `recordPhase` fills `phases`, keyed by object name with the shown projection, for declared, derived and composition objects.
- `model/irgen/Context.scala`: `placing(at)(body)` places the positions of lifted expressions. `pos` honours it; `where` and `fail` still name the written position.
- `model/irgen/Compositions.scala`: records each composition object's projection.
- `model/irgen/Order.scala`: when a machine's `rules` initialize, they read the body of the machine's Phased projection (`phaseRead`), as they read `Rules(_.phase)` in the past. The machine's own initialization treats the projection lambda as deferred, as before. The header comment is updated.
- `model/irgen/Lift.scala` (**Touches exception**): `Lifter.phases` carries the record out of a lift run so the test can read it. Nothing new is written to IR or to disk.
- Fixtures (`model/irgen/testdata/phasedRules`, `phasedMixin`, and tests in `test/Fixtures.test.scala`):
  - The two lamps are written on the same lines, one with `Rules(_.light)` and one with `Phased` plus `Rules:`. Both lift `fixture.phased.Switch` to the same bytes, checked against `phasedRules/expected.json`. Both lifts are mapped to phasedRules' positions.
  - `Unprojected` uses `Rules[Bulb, Outcome, Nothing, Light]`, so `in` compiles with no projection anywhere. Lifting it is refused at `Phased.scala:61` with the new message.
  - A lift in the test JVM reads `Lifter.phases`: `stiffer` (`Derived(Stiff.unmonitored)`, where `Stiff = Derived(Switch.rebind(...))`) equals `switch`, and `lopsided` (`Twins.withMember(...)`) equals the `Phased` projection of `twins`.

### Deviations and exceptions
- **Positions:** a Phased projection's reads are recorded at the line of the rules object's declaration, not at the `Phased` argument. That is the only way the migration in task 3 leaves model/ir byte-identical, because IR expressions carry lines. Refusals still name the line where the tree is written.
- Touches exception: `model/irgen/Lift.scala` (the `phases` field and one line in `liftRoots`).
- `phaseOf`'s refusal of a projection that is not a lambda now names `Phased[State, Phase](_.phase)`.
- The AC "initialization-order check accepts `Phased` plus rules" is shown by phasedMixin lifting with the Order lint on, over its whole source. The existing initOrder fixtures show that the earlier refusals still hold.

### Expected IR delta
None. Models still use `Rules(_.phase)`, and the lifter's output for that spelling is unchanged: every lifts/expected file passed in check mode.

### Tests
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`: 97 passed.
- Red first: the byte-identity test failed on the lines (22 against 31) before the placement change, then on the missing expected.json.
- `make lint-model-syntax lint-model-irgen lint-model-irgen-lifts`: rc=0. scalafmt `--check`: rc=0.
- GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration

### For later tasks
- **Task 3** (migrating the Models): `extends Machine[...], Phased[State, Phase](_.phase)` plus `object rules extends Rules:`. The IR should not change, provided `object rules` stays on the line where `Rules(_.phase)` was. It always does, because the change is in place. Compositions: `extends Composition[S](...), Phased[S, P](_.x.phase)`. Scalafmt wraps a long header like this: `object X\n    extends Composition[S](...),\n      Phased[...](...):`.
- **Tasks 5 and 6:** call `phaseProjection(cls, at)` (Declarations) to get the projection `Term` of any object form, including derived ones. `Lifter.phases` and `recordPhase` exist only to make that observable.
- **Task 4:** drop the `Rules(...)` lambda fallback in `ruleSteps`. The `phaseReaders` placement then always applies. Keep it so the IR stays as it is.
- fn-137.1's open item still applies: the framework gives derived compositions no typed `Phasing` given. The lifter side resolves them already.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a86f0314db
- Tests: baseline: green via fn-137.1's final runs in this lane (model/irgen 94 passed at e29b84c06), red-first: the new byte-identity test failed first on the Phased projection's positions (line 22 against Rules(_.light)'s line 31), and again on the missing phasedRules/expected.json, mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (check mode: 97 passed, no lifts/expected file changed), phasedRules/expected.json copied from the test's own lift output (phasedRules-out.json, cmp-equal to phasedMixin-out.json), not an UMPIRE_LIFTER_UPDATE run, make lint-model-syntax lint-model-irgen lint-model-irgen-lifts (each rc=0), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check (rc=0), GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration
- PRs:
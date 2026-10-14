---
satisfies: [R8, R9]
---
# fn-135-effect-and-is-blocks-for-effects-and.5 Status declarations on phase cases: derived facts, lifting and refusals

## Description
Add status declarations on phase enum cases, the derived status fact in `effect { }` blocks at run time, its lifting, and its two refusals, with a fixture machine that shows R9 at run time and in lifted IR. Split from .4 so the framework and lifter part lands in the DSL batch's step 1 beside .1-.3, and .4 only adopts it in `ActivityProduct`.

**Size:** M
**Files:** `model/umpire/Syntax.scala` (status declaration trait, draft derivation), `model/umpire/Effects.test.scala` (from .1, extend), `model/irgen/Syntax.scala` and/or `model/irgen/Declarations.scala` (the .3 effect-block walker), `model/irgen/Constants.scala` or `model/irgen/Types.scala` (reading a case's declared status), new `model/irgen/testdata/lifts/StatusFacts.scala` and `StatusFactRejects.scala` with `expected/` output and `rejects.txt` lines, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/umpire/Syntax.scala, model/umpire/Machine.scala, model/umpire/Effects.test.scala, model/irgen/*.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Decision this task implements (see Open question)
The spec's "a step that changes the declared status" is implemented as **an effect block that assigns the status-declaring field**: the step records the declared fact of the assigned case, after the explicit facts; a block that does not assign that field records no status fact; a rejecting block records none. This reading is static, so the lifted IR is a constant fact list identical to the hand-written `enter(s.copy(phase = X), statusX)` (R10). A value comparison (record only when the new case differs from the old) would put a conditional in the IR and would also drop `statusCancelRequested` from `ActivityProduct`'s `requestCancel` step out of `cancelRequested` (its rule fires `in(scheduled, started, paused, cancelRequested)`), which R10 forbids. If the owner rules the other way before this task starts, stop and re-plan.

### Approach
- **Declaration.** A framework trait over the fact type that a phase enum extends through a constructor parameter, e.g. `enum Phase(val status: Fact) extends Recorded[Fact]` with `case started extends Phase(Fact.statusStarted)`. A case without the argument does not compile because the parameter has no default (R8's first refusal; no lint needed). The form must coexist with fn-136's role mixins (`case started extends Phase(Fact.statusStarted), Held`); keep `derives Finite` working (check `Domain.scala` Finite derivation over a parameterized enum). Add a `Core form:` doc naming the hand-written `record(...)` it stands for, and add the trait name to `sugarNames` (`model/check/SyntaxRule.scala:37`) if the syntax rule requires it.
- **Run-time derivation.** In the `effect { }` draft from .1: when the block assigns a field whose value is `Recorded`, append that value's `status` after the explicit facts when the step is built. A setter that only hands the draft an `S => S` cannot tell it which field was assigned, least of all on a same-value assignment, so a setter for a status-declaring field uses a second fixed shape that also passes the assigned value (e.g. `d.set(p)(_.copy(phase = p))`). The lifter recognizes exactly that shape and refuses others, as .3 does; other fields keep .1's shape. Method-form effects (`enter`, `stay`, `reject(o, s)`) and `(S, A) => …` effects derive nothing: they record what they write, so R7 holds trivially.
- **Explicit derived fact refused (R8's second refusal).** A block that both assigns the status field and `record`s the fact that assignment derives: refused by the lifter, naming the effect and the fact (`fail(tree, msg)`, as .3's refusals), and by the draft at step construction with the same message, so munit tests reach it without a lift.
- **Lifting.** Extend .3's effect-block walker: for an assignment of the status field to an enum case literal, resolve that case's constructor argument from the case's TASTy tree (`sym.tree`; no existing code reads enum case args — `Types.scala:112-116` reads only `sym.children`, and `Declarations.scala:143-147` `literalValue` handles case refs) and append it as a fact literal after the `record` facts. An assignment of a non-literal value to the status field is refused, naming it. Status declarations must not change the enum's lifted `ir.Enum` (value cases carry no fields; the enum's constructor `val` must not surface as a field), so existing IR is untouched.
- **Fixture machine (R9).** `StatusFacts.scala`: a two-field state (a phase enum with declared statuses, plus a second field such as an attempt count) and a twin machine written in method form with hand-written facts; effects: assignment that changes the phase, assignment of the same phase (still records, per the decision), assignment of the other field only (no status fact), phase assignment plus an explicit non-status `record` (explicit fact first, then the status fact), reject (no fact). Require one IR for the pair but for names and positions, as `Sugar.scala` does. Mirror the cases in `Effects.test.scala` at run time.
- **Refusal fixtures.** `StatusFactRejects.scala`: explicit derived fact, non-literal status assignment, each in `rejects.txt`. A case without a status is a build-failure fixture through `refusals(fixture)` (`Fixtures.test.scala:76`), one negative fixture per refusal (memory: paired-level-validators-must-check-each).

### Investigation targets
**Required** (read before coding):
- `model/umpire/Syntax.scala:10-30` and .1's draft — where the derivation hooks in
- `model/irgen/Syntax.scala:19-60` — how `enter` lifts its fact list
- `model/irgen/Types.scala:100-120`, `model/irgen/Declarations.scala:140-150`, `model/irgen/Constants.scala:25` — enum lifting, case literals, val RHS resolution
- .3's effect-block walker and `model/irgen/testdata/lifts/Sugar.scala` — the paired-fixture pattern
- `model/irgen/test/Fixtures.test.scala:76` and `:192-292` — build and lift refusals

**Optional** (reference as needed):
- `.flow/specs/fn-136-*.md` — role mixins on the same enum cases
- `model/umpire/Domain.scala:39-60` — `Finite` derivation

### Key context
- A machine whose phase enum declares no status keeps today's behavior (spec, Derived status facts); every existing lifter fixture and IR file stays unchanged.
- The System's `pauseRequested` question and deriving the refinement mapping or `Closable`'s projection stay out of scope (spec, Boundaries, Parked unknowns).

### Open question
- R9's literal wording ("a step that keeps the status records no status fact") versus this task's assignment reading. Owner to confirm; recorded in the spec's Parked unknowns.

## Acceptance
- [ ] A phase enum declares each case's status fact on the case; a case without one does not compile (build-failure fixture)
- [ ] At run time, an effect block that assigns the status field records the assigned case's declared fact after its explicit facts; one that does not assign it, and a rejecting one, record no status fact; a same-value assignment still records (munit, `Effects.test.scala`)
- [ ] The `StatusFacts` fixture lifts to IR identical to its hand-written method-form twin but for names and positions, for every case listed in Approach (R9)
- [ ] An explicitly recorded derived fact and a non-literal status assignment are refused by the lifter, each message naming the effect, the fact or statement, and its position; the draft refuses the explicit derived fact at run time
- [ ] Every existing lifter fixture passes unchanged, and an enum with status declarations lifts to the same `ir.Enum` as without (R7)
- [ ] `make lint-model-syntax lint-model` and the umpire and irgen munit suites pass


## Done summary
Each case of a phase enum can now declare the status fact it is recorded as, through the framework trait `Recorded[F]` (`enum Phase(val status: Fact) extends Recorded[Fact]`). An `effect { }` block that assigns such a field records the assigned case's status after the block's own facts, both at run time and in the lifted IR. The static assignment reading from the task's Decision is implemented: a same-value assignment still records, and a block that does not assign the field or that rejects records none.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at the batch end)

Tier: IMPLEMENTER claude-opus-5-5 at high (actual_model: claude-opus-5-5)

Baseline: green before the first edit. `scala-cli test model/project.scala model/umpire` rc=0, and `scala-cli test model/irgen` rc=0.

### What changed
- `model/umpire/Syntax.scala`:
  - New `trait Recorded[F] { def status: F }`, with a `Core form:` doc.
  - `Draft` gains the overload `set(value: Recorded[F])(update: S => S)`, which keeps a list of assigned statuses.
  - `Draft.step` appends those statuses after the recorded facts. With a `require`, it refuses a block that also records one of them: "an effect block records X, which its assignment of a status already records: drop the record(X)".
- Setter shape for a status field: `def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit = d.set(p)(_.copy(phase = p))`. The fact type must be concrete, because a wildcard `F` cannot accept `Recorded[Fact]`. Other fields keep the `Draft[State, ?, ?]` shape from .1.
- `model/irgen/Expressions.scala`: `setterField` now delegates to the new `setterShape(fn): Option[(field, recorded)]`, which also recognizes the status shape. The value passed must be the setter's own parameter.
- `model/irgen/Syntax.scala` (`effectSteps`):
  - `Assign` carries a `recorded` flag.
  - `declaredStatus(caseSym)` finds the enum case's call to the enum constructor in the case's own tree, and takes the argument at the position of the parameter named `status`.
  - Each status assignment of a case literal appends that fact after the `record` facts.
  - Three refusals, each naming the effect, the statement and its position:
    - an explicit `record` of the derived fact;
    - a status assignment that names no case;
    - a plain-shape setter for a field whose values are `Recorded`. I added this third one: without it such a setter would silently derive nothing.

### Acceptance
- A case without a status does not compile. `testdata/crossed/Statuses.scala:12:3` is in the crossed refusals list.
- Run time: `Effects.test.scala` adds the `Tasker` machine with two tests:
  - derivation, compared over every state with the method form: phase change, same-value assignment, other field only, explicit fact plus status, reject;
  - the draft's refusal of an explicitly recorded derived fact.
  
  Red-first: with the derivation and the refusal disabled, both tests failed.
- IR (R9): in `lifts/StatusFacts.scala`, the blocks of `Derived` sit beside the hand-written defs of `Written`. The new Fixtures test "an effect block's derived status facts lift as the facts its method form writes" requires machines and functions to be identical apart from names and positions. It also requires `Phase` to lift as `{"cases":[{"name":"idle"},{"name":"running"},{"name":"paused"}]}`, an enum whose cases carry no fields. Lifted fact lists: start/resume `[statusRunning]`, retry `[]`, pause `[attempted, statusPaused]`, refuse `[]`. I did not run this IR test red first. The expected JSON shows the derived facts, which the method twin writes by hand.
- Refusals: `lifts/StatusFactRejects.scala` adds the roots `ExplicitStatus` (line 20), `ComputedStatus` (line 30) and `PlainStatusSetter` (line 39), each with a line in `expected/rejects.txt`.
- Every existing lifter fixture is unchanged. The update run changed only `expected/rejects.txt` (3 added lines) and the new `expected/statusFacts.json`.

### Tests run
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire` rc=0
- `make model/build/model-scala.jar` (needed so the lifts build sees the framework change; this is not an IR regeneration)
- `UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen` rc=0
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen` (check mode) rc=0
- `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check` rc=0
- `make lint-model-models`, `lint-model-irgen`, `lint-model-irgen-lifts`, `lint-model-check` and `lint-model-syntax`: each rc=0 (`lint-model-syntax` after a fix to the Core form doc, which changed only a comment)
- GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration

### Declared IR delta
None for `model/ir`, `model/cases` or the Model lift expectations. No Model's phase enum extends `Recorded`, and the new paths fire only for a setter of the status shape. Deliberate fixture changes: the new `expected/statusFacts.json` and 3 new lines in `expected/rejects.txt`.

### Touches
- `model/irgen/testdata/crossed/Statuses.scala` is outside the declared Touches (`testdata/lifts/**`). The task asks for a build-failure fixture through `refusals(fixture)`. A file that fails to compile cannot live under `lifts/`, which builds as one project, so it joins the existing `crossed` build-refusal fixture.
- `sugarNames` (`model/check/SyntaxRule.scala`) is unchanged: the syntax rule does not require `Recorded` there, and the file is outside Touches.

### For later tasks
- fn-135.4 (ActivityProduct):
  - Declare `enum Phase(val status: Fact) extends Recorded[Fact] derives Finite` with `case started extends Phase(statusStarted)` and so on. The parameter must be named `status`, because the lifter finds the constructor argument by that name.
  - The phase setter must be exactly `def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit = d.set(p)(_.copy(phase = p))`, with the concrete fact type in the `Draft`. A plain-shape phase setter is refused by the lifter.
  - Assign the phase a case literal (`phase = Phase.started`, or an imported case). Anything else is refused.
  - Drop the `record(statusX)` calls: recording a derived fact explicitly is refused.
  - `requestCancel` from `cancelRequested` still records `statusCancelRequested`, because a same-value assignment records (R10).
  - Run-time order is the explicit facts, then the status.
- If fn-136's role mixins come later (`case started extends Phase(statusStarted), Held`), the lifter still finds the enum constructor's call among the case's parents.

### Follow-ups (not built)
- `View`/`Draft`/`Recorded` are not in `sugarNames` (carried over).
- The draft's run-time refusal cannot name the effect val: the framework has no name for it at run time, unlike the lifter's message.
- No MILESTONES.md edit is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 68f19659153a7a345c1e4050167e44e23ad43c55
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire, UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen, mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check, make lint-model-models, make lint-model-irgen, make lint-model-irgen-lifts, make lint-model-check, make lint-model-syntax, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration
- PRs:
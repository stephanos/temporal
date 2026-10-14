---
satisfies: [R1, R2, R3, R4]
---
# fn-136-phase-roles-on-lifecycle-enums.1 Role traits and role-test lowering in the lifter

## Description
Adds the role vocabulary to the framework and teaches the lifter to lower role tests and refuse conflicting roles. No model changes, so `model/ir` is unchanged apart from lifter fixtures. This is the early proof point: it shows a role test lowers to exactly the case-set membership a hand-written `in(...)` gives.

**Size:** M
**Files:** `model/umpire/Roles.scala` (new: the eleven role traits), `model/irgen/Roles.scala` (new: per-enum role closure and conflicts), `model/irgen/Expressions.scala`, `model/irgen/Types.scala`, `model/irgen/testdata/lifts/Roles.scala` (new positive fixture), `model/irgen/testdata/lifts/expected/roles.json`, `model/irgen/testdata/lifts/Rejects.scala` (or a new `RoleRejects.scala`), `model/irgen/testdata/lifts/expected/rejects.txt`, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/umpire/Roles.scala, model/irgen/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Role traits are plain member-less traits in `model/umpire` with the inheritance of spec R1. They name no Temporal kind, so `CapabilityVocabularySuite` stays green.
- A role is a framework role trait or a trait extending one; a stand-alone trait is not a role. Build one deep module in `model/irgen/Roles.scala`: given an enum symbol, return for every role its cases carry (framework roles and model roles extending them) the cases having it (inheritance closed via the case's base classes, starting from `sym.children` as `Types.scala:113-115` does) and the list of conflicting cases. Both lowering and the R4 check call it; give it its own unit tests over the fixture enums.
- Expression form: lower `x.isInstanceOf[R]` (R a role) to `OP_CONTAINS(x, cases)` by reusing the `in(...)` lowering at `model/irgen/Syntax.scala:52-60`. Cases in enum declaration order.
- Pattern form: at `model/irgen/Expressions.scala:729` the `TypedOrTest(inner, tpt)` branch passes every type test through. The compiler also inserts `TypedOrTest` around constructor patterns (`case Fact.statusTimedOut(_)`, `case Reply.handlerError(r)`, `case Some(x)`); those keep passing through unchanged. Only `TypedOrTest(Wildcard | Bind(_, Wildcard), tpt)` is a user-written type pattern: lower it to the alternatives of the matching case literals when `tpt` is a role (the Alternatives/Literal machinery at 700-730), and refuse it otherwise instead of lifting it as a wildcard.
- R4: run the conflict check when an enum is declared (`declareType`, `Types.scala:113-115`), for every enum that has role parents, on roles after closure. Refuse with `fail(tree, msg)` naming the case and roles.
- Refusals R3 names: role test on a non-enum value, role no case has, non-role type test (including a stand-alone trait).

### Investigation targets
**Required** (read before coding):
- `model/irgen/Syntax.scala:52-60` - `in(...)` to OP_CONTAINS lowering to reuse
- `model/irgen/Expressions.scala:362-371,700-730` - match and pattern lifting, the silent TypedOrTest drop at 729
- `model/irgen/Types.scala:12-15,113-115` - enum case detection and IR case construction
- `model/irgen/test/Fixtures.test.scala:112-140,216-330,398-428` - positive fixtures and reject roots
- `model/project.scala:6-7` - `-Werror -unchecked`

**Optional** (reference as needed):
- `model/irgen/testdata/lifts/Patterns.scala` - existing match fixtures
- `model/check/test/CapabilityVocabulary.test.scala` - framework/Temporal vocabulary guard

### Key context
- Enum cases with role mixins keep `derives Finite` and `values` unchanged (verified on Scala 3.9.0 against the framework during capture); still assert it in the positive fixture.
- Update expected files with `UMPIRE_LIFTER_UPDATE=1` (the gate's `--update`). Pin each refusal's message in `rejects.txt`, one line per declaration; do not share one fixture across error kinds.
- `isInstanceOf[Closed]` on a concrete enum type must not trigger an unchecked or always-true warning under `-Werror`; cover a single-case scrutinee in the fixture.

## Acceptance
- [ ] The eleven role traits exist in `model/umpire` with the R1 inheritance; a fixture enum with role mixins has the same `Finite` instance and `values` as without them.
- [ ] A fixture lifts `p.isInstanceOf[Closed]` and `case _: Closed` to the same case set, in declaration order, as the equivalent hand-written `p.in(...)`, including an inherited role (`Retrying` is `Waiting` and `Live`) and a model role extending a framework role (its test lowers to the cases carrying it).
- [ ] Existing constructor patterns under a compiler-inserted type test (enum-case unapply, `Some`) lift exactly as before (existing fixtures unchanged).
- [ ] Refusal fixtures pin one message each for: role test on a non-enum value, role no case has, non-role type test on an enum (previously a silent wildcard), test against a stand-alone trait, case both `Live` and `Closed`, two of `Waiting`/`Held`/`Suspended`, two closure roles. `Retrying` with `Waiting` is accepted.
- [ ] The role-closure module has unit tests independent of any model.
- [ ] `make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model` and `make lint-model` pass; no file under `model/ir` other than lifter fixtures changes.

## Done summary
Adds the eleven phase role traits to the framework (`model/umpire/Roles.scala`). The lifter now lowers `p.isInstanceOf[R]` to an `in(...)` of the role's cases, and lowers `case _: R` / `case q: R` to the alternatives of those case literals (one case gives the bare literal), both in declaration order. It also refuses role-carrying enums whose roles conflict. All of this sits behind one role-closure module, `model/irgen/Roles.scala`, which has its own unit tests.

stage: impl-review - skipped(config: REVIEW_MODE=none - DSL batch: reviews run once at the batch's end)
Tier: lane B, IMPLEMENTER claude-opus-5-5 at high

### What changed
- `model/umpire/Roles.scala`: `Live`; `Waiting`, `Held`, `Suspended` extend `Live`; `Retrying` extends `Waiting`; `Closed`; `Succeeded`, `Failed`, `Canceled`, `Terminated`, `TimedOut` extend `Closed`. None of them names a Temporal kind.
- `model/irgen/Roles.scala`: the pure `Roles.closure` (class names in, role-to-cases map and conflicts out) and the `PhaseRoles` lifter trait (`checkRoles`, `roleTest`, `rolePattern`), mixed into `Lifting`.
- `Types.declareType` runs `checkRoles` on every enum it declares (R4).
- `Expressions.lift` handles `isInstanceOf`. `Expressions.pattern` handles `TypedOrTest(Wildcard | Bind(_, Wildcard), tpt)`. A compiler-inserted `TypedOrTest` around an `Unapply` still lifts as before.
- Refusals, each pinned in `rejects.txt` from `lifts/RoleRejects.scala`: a role test on a value of no enum, a role no case has, a non-role type pattern on an enum (was a silent wildcard), a stand-alone trait, the three conflict kinds, and one guard not in the ACs: a role carried by a case with fields. Such a case is no single value, so `OP_CONTAINS` over its bare literal would be wrong IR.
- Fixtures: `lifts/Roles.scala` (`Roled` with role tests, `Listed` with the hand-written cases). The test requires identical machines and functions, but for names and positions, and requires the IR type of the role-carrying `Phase` to equal the role-less `Bare`. `Retrying` together with an explicit `Waiting` is accepted.
- `model/umpire/Roles.test.scala`: role mixins leave `Finite` values and their order unchanged; each case tests as its declared and implied roles.

### Outside the declared Touches (necessary)
- `model/umpire/Syntax.scala`: `PhasesOf` now accepts `Q <: P` (was `P =:= Q`). A case that takes a role has the type `Phase & Role`, so a rule `in(Phase.done)` on such a case did not compile (verified). Without this change R2 does not hold. The no-projection refusal is unchanged, since `P` is `Nothing` there.
- `model/umpire/Roles.test.scala`: a new framework test for the `Finite` acceptance criterion.

### Expected IR delta at the batch regeneration
None. A lift of every Model before and after this commit is byte-identical (`diff -r` empty). Only lifter fixtures changed.

### For later tasks
- **fn-136.2/.3 (models):** the lint `DisableSyntax.noIsInstanceOf` refuses `p.isInstanceOf[Closed]` in Models (`make lint-model-models`). A Model's role test must use fn-136.4's short spelling or a type pattern (`p match { case _: Closed => true; case _ => false }`, which does not lower to `OP_CONTAINS`, so it is not byte-equal to an `in(...)`), or carry `// scalafix:ok DisableSyntax.isInstanceOf`. The fixtures use `scalafix:off/on`. For R7 byte-equality, use fn-136.4's spelling.
- A role-carrying case has the type `Enum & Role`. Anything that infers a type from a single case (`val x = Phase.done`, a `List(Phase.done)`) gets that intersection type. `PhasesOf` now handles this for `Rules.in`; other inference sites may need an ascription.
- Lowering puts every role literal at the test's position. The hand-written `in(a, b)` had per-member positions. That is a position-only diff.
- A role on a case with fields is refused at its test.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 954c96d536
- Tests: baseline: green (mise exec -- scala-cli test model/irgen at 0f25754d6, after building model/build/model-scala.jar and model-scala.classpath in the clone), UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test model/irgen (writes expected/roles.json and the RoleRejects lines of expected/rejects.txt), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (suite_rc=0, 99 passed), mise exec -- scala-cli test model/project.scala model/umpire --test-only umpire.RolesSuite (green), Models lift into a scratch dir before and after the change: diff -r /tmp/laneB-ir-base /tmp/laneB-ir-1 is empty (no model/ir change), scala-cli fmt --check model/umpire model/irgen model/check model/temporal: clean, make lint-model-irgen, lint-model-irgen-lifts, lint-model-models, lint-model-syntax: rc=0 each (run one by one; lint-model-check untouched), GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: no model gate per task (worker-notes.md)
- PRs:
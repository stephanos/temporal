---
satisfies: [R6]
---
# fn-135-effect-and-is-blocks-for-effects-and.1 effect, is, record and reject(outcome) sugar in umpire

## Description
Add the author-facing sugar: `effect { }`, `is { }`, `record(facts*)`, the one-argument `reject(outcome)` and the draft they work on, plus the run-time half of R6. Split first because the lifter tasks need the forms to exist and compile.

**Size:** M
**Files:** `model/umpire/Syntax.scala`, `model/umpire/Machine.scala` (only if the forms are typed through the machine), `model/check/SyntaxRule.scala`, a new `model/umpire/Effects.test.scala`
**Touches:** [model/umpire/Syntax.scala, model/umpire/Machine.scala, model/check/SyntaxRule.scala, model/umpire/Effects.test.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Put the forms beside `enter`/`stay`/`reject` (`model/umpire/Syntax.scala:13-30`), each with a `Core form:` doc naming the method form it stands for, and add `effect`, `is` and `record` to `sugarNames` (`model/check/SyntaxRule.scala:37`). `reject` is already listed; the new arity overloads the existing `reject(outcome, s)` at `Syntax.scala:27`, which stays.
- Type the forms by the machine's `S`, `O`, `F` so no author writes type arguments: either members of `Machine[S, O, F]` (`model/umpire/Machine.scala`, class at ~:174) or functions over the existing `Owner[S, O, F]` given. `Ok[O]` is a feature-level given (`Standalone.scala:129`), so `effect` takes it with `using`, as `enter` does.
- `effect` returns a plain `S => List[Step[S, O, F]]`: core `StepBinding` casts by arity, so a wrapper type would break rule binding.
- The draft is created per call and holds the state, the facts and an optional rejection. Its mutable field needs a `scalafix:ok` comment, following the precedent `private var open` in `Syntax.scala`.
- Two context types: a read-only view of the state, which `is { }` provides and getters require, and the effect draft, which extends the view and which `effect { }` provides. `record`, the one-argument `reject` and setters require the draft, so they compile neither outside a block nor inside `is { }`. Give the missing-given cases readable `@implicitNotFound` messages.
- The draft API is what the fixed accessor shape (spec, Architecture) calls: a getter reads one field through the view (`v.get(_.phase)`-style) and a setter replaces the draft's state with a one-field copy (`d.set(_.copy(phase = p))`-style). Keep it that small: the lifter in .2/.3 recognizes exactly this shape.
- Field accessors are per state type and hand-written (spec, Architecture). This task defines the view/draft API and writes the test state's accessors in that shape; the activity's accessors land in .4 and the lifter fixtures' in .2/.3.
- No `inline`, no macros on the author surface (`.plans/DSL_OPERATORS.md:227`, rule 5).

### Investigation targets
**Required** (read before coding):
- `model/umpire/Syntax.scala:1-60` — sugar conventions, `Ok`, `enter`, `reject`, `Core form:` docs
- `model/check/SyntaxRule.scala:30-60` — what the syntax rule checks
- `model/umpire/Domain.scala:39-60` — `Finite` derivation, for the state-type conventions
- `/tmp/effect-proto/Proto.scala` — the working prototype of draft, setter and `record` (if still present)

**Optional** (reference as needed):
- `model/umpire/Rules.test.scala` — munit style for machine-level tests
- `.plans/DSL_OPERATORS.md` — rules 5 and 35

### Key context
- `Step[S, O, +F]` is covariant in `F`; keep the fact type the machine's `F` so `List[Step[S, O, F]]` infers.
- A field getter `def phase(using Draft…)` and setter `def phase_=(…)(using Draft…)` in scope make `phase = x` compile (Scala 3.9.0, verified by the prototype).

### Acceptance
- [ ] `effect`, `is`, `record` and the one-argument `reject` exist with `Core form:` docs; `make lint-model-syntax` passes
- [ ] Run-time tests over a two-field test state: an effect block yields the same steps as its method equivalent for assign+record, partial assignment, several `record` calls (call order), assign-only, empty, and reject; an `is` block answers as its predicate
- [ ] Compile-failure tests (`compileErrors` assertions): `record`, one-argument `reject` and a setter do not compile outside a block, nor inside `is { }`
- [ ] Existing umpire and model tests pass (`make umpire-check-model`)

## Acceptance
- [ ] TBD

## Done summary
Added the effect-block sugar to `model/umpire/Syntax.scala`: `effect { }` builds a state-only step function on a fresh per-call `Draft[S, O, F]`, `is { }` builds a predicate over a read-only `View[S]`, and `record(first, rest*)` and the one-argument `reject(outcome)` take the draft as a leading using clause, so they don't compile outside an effect block or inside `is { }`. `effect`, `is` and `record` join `sugarNames`. `model/umpire/Effects.test.scala` checks the run-time half of R6 over a two-field `Job` state. It also has `compileErrors` tests that assert the `@implicitNotFound` text.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at batch end)

Baseline: green (`scala-cli test model/project.scala model/umpire`, `make lint-model-syntax`), recorded before the first edit.

Not run (batch rules): `make umpire-check-model`, regeneration, the full Go suite. The irgen/lifter tests were not run either: the lifter is untouched, and its sugar match on `("reject", List(List(outcome, state)))` is keyed by arity, so the new overload doesn't reach it.

Declared IR delta: none. The task adds framework sugar and a munit fixture. No Model or lift fixture uses the new forms yet.

### For later tasks
- Accessor shape the lifter (.2/.3) must recognize, exactly:
  `def phase(using v: View[Job]): Phase = v.get(_.phase)` and
  `def phase_=(p: Phase)(using d: Draft[Job, ?, ?]): Unit = d.set(_.copy(phase = p))`.
  The getter takes a `View[<State>]`. The setter takes a `Draft[<State>, ?, ?]` with wildcards.
- Signatures: `is[S, O, F](using Owner[S, O, F])(body: View[S] ?=> Boolean): S => Boolean`.
  `effect[S, O, F](using Owner[S, O, F], Ok[O])(body: Draft[S, O, F] ?=> Unit): S => List[Step[S, O, F]]`.
  `record[S, O, F](using Draft[S, O, F])(first: F, rest: F*)`.
  `reject[S, O, F](using Draft[S, O, F])(outcome: O)`.
  In TASTy each one is the `umpire.Syntax$package$` member with a leading using arg list. The two-argument `reject(outcome, s)` is unchanged.
- At run time an effect returns `List(Step(o, s))` when it rejects (the original state, no facts). Otherwise it returns `List(Step(ok, assigned, facts))`, with facts in call order. Getters inside an effect read the assigned (current) state.
- A `View`/`Draft` given never comes from the machine. The blocks need the machine's `Owner` given, which is in scope inside a machine object's sections. A test outside a machine must supply `given Owner[...] = Owner(machine)`.
- `UpTo(1)` comparisons inside a block need the bound written out (`UpTo[2](1)`).
- `View` and `Draft` are not in `sugarNames`. The task listed only effect/is/record. Add them later if the rule should cover them.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 615066f80a1c6bab54f956e581774a3e902bdae4
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire, make lint-model-syntax, make lint-model-models, make lint-model-check, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check, mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check
- PRs:
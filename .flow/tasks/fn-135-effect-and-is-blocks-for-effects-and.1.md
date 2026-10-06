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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

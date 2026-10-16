# Models are plain Scala: state as vars, effects as assignments, lifted by the plugin, evaluated on the IR

## Conversation Evidence

> user (2026-10-16): "if we wanted to mirror Quint's simplicity where they just say `var heatingOn: bool` and `heatingOn' = false` instead of our s.copy and such. could we do that?"
> user: "why would we need state() and :=? if we write it as regular Scala code; it would be even clearer"
> user: "do we need to actually run an effect handler or just parse it via the compiler plugin?"
> user: "I'm committed to this new DSL - file a new flow spec and break it into tasks."

## Goal & Context

A Model today declares its state as a case class, threads it through every effect and guard as a
parameter, and updates it with `copy`. Quint declares `var heatingOn: bool` and writes
`heatingOn' = false`. The gap is not expressiveness, it is noise: `Snapshot`, `s =>`, `s.copy(…)`,
`after.state.` on every line of every Model, and the `Phased` projection and writer that exist only
because a phase is a field the facet cannot name.

Two facts make the Quint form available in Scala without inventing syntax. First, the compiler
plugin lifts whatever the author writes, and a block of assignments to vars lifts to the `copy`
message the IR already has. Second, nothing in production executes an effect body in Scala: the
Scala step tables were retired (fn-173.16), the Go checks are the oracle (owner, 2026-10-13), and
the only in-process evaluation left is the native sample tests. So an effect body needs to
type-check and to lift; it does not need to run.

This spec makes a Model plain Scala: state is `var`s on the Machine, an effect is a method whose
body assigns to them, a guard or an expectation reads them by name, and the state record, its
schema and `init` are derived from the declarations by the plugin. The DSL keeps constructing
declarations (actions, transitions, claims) as runtime values; bodies become opaque
callables that only the plugin reads. The native sample tests move to an evaluator over the IR so
the fast loop survives without Scala execution.

Owner's standing decisions carried: byte identity is not required; the Go checks' answers are the
oracle; Temporal sources stay excluded; the framework samples are the proving ground.

## Architecture & Data Models

**State.** A Machine declares its state as `var`s in its object, in declaration order. Their
declared types are the schema; their initialisers are `init`. The plugin synthesises the IR record
type named `<machine>.State` with one field per var. There is no author-written state class and
`Machine` no longer takes a state type parameter: `Machine[O, F]`.

**Bounds as types that erase.** `UpTo[N]` and the other bounded forms become transparent aliases
of `Int` whose argument the plugin reads from the var's declared type tree. `temperature += 1` on a
`var temperature: UpTo[140] = UpTo(60)` type-checks because the alias is `Int`; the bound lives
in the schema only. Collections and enums keep their types as today.

**Effects.** An effect is a `def` in the Machine's `effects` object whose body is a block of
statements over the state vars and its own parameters: assignments `x = e`, compound assignments
`x += e`, `if`/`match` with such blocks in each arm, `choose(a -> block, b -> block)`, and the
step vocabulary `reject(outcome)`, `records(fact)`, `crash`. An effect with an input takes it as an
ordinary parameter. Scala's sequential semantics is the semantics: a read after a write in the same
block sees the write; the lifter substitutes to produce one `copy` per path. `stay` is an empty
body. `enter` goes.

**Transitions as control flow.** A transition is `on(action)` followed by a block of plain control flow whose leaves fire the
action's effect: `if g then e()` is a guarded transition, with no `else` meaning disabled; a `match` on
`phase` is the phase table, `case a | b` for several phases, `case a if g` for a phase with a
guard, `case _: Role` for a role over phases, `case _ => ()` for disabled otherwise; a bare `e()`
is always; `reject(r)` at a leaf is a rejection. A leaf is a call to a named effect, never an
assignment: the state change lives in `effects`, the transition says when it fires (owner, 2026-10-16,
enforced). First match wins, as in Scala. An input is the block's parameter:
`on(client.send): s => if s.amount <= supply then effects.send(s)`. `where`, `~>`, `when`,
`always`, `rejects` and `enters` as transition vocabulary are retired; the section is `transitions`
(`rules` renamed, owner 2026-10-16) and the marker trait `Transitions`; `disabled(a, b)` stays, it is a
declaration about actions. The overlap lint becomes a warning that arms are exclusive; the
compiler's exhaustiveness error covers a missing phase.

**Guards and expectations.** A guard reads vars by name in the transition's `if` or `case` condition,
with an input as a parameter when the action has one. A same-step expectation reads the after-state
by name; a transition expectation reads the before-state through `before(x)`. `after.state.x` goes.

**Claims by kind, in sections (fn-191 defines the sections).** A Machine states its claims in
sections named for their kind, and the section is the capture scope, as `effects` is for effects:
a member of `invariants` is a block of `check` statements that must hold in every state; a member
of `postconditions` is one that relates the state a step started from, `before`, to the state now,
for every step or, under `after(action):`, for steps of that action; `monitors` are unchanged;
`progress` holds `leadsTo` claims. No per-member kind word: inside `invariants` a claim is
`val x = if beeping then check(!heatingOn)`, and inside `postconditions` the only word is
`after(action):`. A bare name is the state now; `before` is the State record a step started from,
so `before.temperature` is field access. `check(c)` is an assertion reported on its own; `if`
guards a check; `oneOf`/`branch` stay for alternatives; `records(fact)` stays for facts. `property`,
`expect`, `across`, `invariant`, `transition`, `after.state` and the `properties`/`safety`/`liveness`
groups are retired. There is no `implies` and no default `Limits`: a query names its limits.

**Progress claims and the environment.** A progress claim is a promise with a bound and nothing
else: `leadsTo(from = p, to = q, within = n)` in the `progress` section, named after its `val`.
The environment a claim is checked in is the Scenario, which already says where a run starts and
which actions it admits; it gains the two remaining statements about an environment: `only(actions)`
admits those actions and no others, and `fair(actions)` makes admitted actions weakly fair, an
action that stays enabled is eventually taken. Both compose with `starts` and each other; a scenario
with neither admits every path and assumes nothing. A progress claim is checked by the same
`query verify claim in scenario limits bounds` as any claim, and answers verified (with the longest
`from`-to-`to` path), deadlock, cycle (naming the admitted action that stayed enabled and was never
taken, and the two remedies) or missed deadline. `assume`, `Assumption`, the assumption vararg,
the name string and `Derived(Model.restrict(…))` as the way to prove progress are retired. No
action is fair unless a scenario says so.

**Claims are checked by default; `queries` is for witnesses and realizations.** Every claim is
checked from every start over the whole reachable graph: an invariant or postcondition is
verified, a progress claim is verified under its environment. A claim says `in scenario` only
when its environment is not the default, with the scenario inline (`in only(a, b)`,
`in fair(a)`, `in starts(x = v)`) or named. The checker reports vacuity on every run: actions that
never fire and `after(action)` postconditions never triggered are warnings. A negative control, a
claim expected to be refuted, is a test, not a Model: `assertRefuted(Model, invariant(...))` in the
sample's test file; the derived negative-control machines (`Leaky`, `Naive`, `EagerCommit`) move
to tests the same way. The `queries` section remains only for what a default cannot express: a
witness, a pinned path, and the checks a realization names to generate Cases; a Model without
Cases has no `queries` section. `verify`/`find` keep their forms there. There is no `Limits` in
the model: a bound is either a promise (`within`) or a budget, and a budget belongs to the caller.
The checker takes its budget from the gate's configuration, with per-feature overrides beside the
feature and never in the Model; every result records the budget and whether exploration completed,
and a cut exploration answers unknown, never verified. `query`, `limits` and `Limits` are retired
from the DSL and the bounds field from the IR `Query` message.

**Actors and the system's own steps.** An action is declared inside its owner: `object user extends
Actor` holds `val pressButton = action`, a stimulus a test drives; `object heater extends Internal`
holds `val heat = step`, a transition the system takes on its own, which a test only observes. The
container says which it is, so `action(this)` loses its argument and `internal` becomes `step`. An
`action` outside an `Actor` object, a `step` outside an `Internal` object, or either in the other,
is refused.

**Nondeterministic starts.** A var initialiser may be `oneOf(domain)`, where the domain is an
ordinary Scala collection or range (`oneOf(-40 to 40)`, `oneOf(Lit.values)`); the set of starts is
the product of the initial domains, lifted to the start list the IR already carries (`starts` is
a list with one element today). `init` is the case where every domain is a singleton. A pinned
witness or a realization names one start with `starts(x = v)`. A correlated start predicate
(`starts where ...`) is not added until a Model needs it.

A claim, scenario or check is a `val`: a declaration with identity, named after its val
and referenced by it (`query verify invariants.beepsOnlyWhenOff`). A `def` is for a parametrised
helper that produces such declarations; a use that needs a name binds the result to a `val`. A
claim member's body evaluates to nothing at construction: `check` outside an evaluation is a
no-op, and the plugin lifts the member's tree.

**Sections.** The section order is `transitions`, `effects`, `invariants`, `postconditions`,
`monitors`, `progress`, `capabilities`, `queries`, inside the Machine: transitions first, since
they are the table of contents and the effects the glossary they point into (owner, 2026-10-16).
fn-191 defines the sections, lints them and applies them to today's samples ahead of this spec;
the lint takes this order when this spec lands.

**Facets and phases.** A phase is a var. A facet trait declares `var phase: Phase` abstractly and
the Machine object defines it; the facet's transitions read and write it like any var. `Phased`, the
projection, the writer and the contextual-phase machinery in the plugin go. This resolves the facet
item fn-189 left out.

**Derivations.** `rebind`/`extend` stay declarations and compose at the IR level; they no longer
adapt Scala callbacks.

**No execution.** `effectOf`, `guardHolds`, the input-effect adapters, `Finite` enumeration,
`StateData`, `Domain` and the native budgets' runtime are removed from the DSL. A binding holds a
reference to its effect for the plugin, not a callable the framework invokes. Without the plugin a
Model is declarations only.

**Plugin.** The body lifter gains one rule family: a var read lifts to `field(s, name)`, an
assignment block lifts to `copy(base, updates)` per path with substitution for sequential reads,
`before(x)` lifts to a read of the transition's first parameter. The lifter refuses an assignment
to a state var anywhere but an effect body, and a state var read outside an effect, guard,
expectation, monitor or refinement. The schema phase of fn-189 R4 is re-cut: the record and its
schema come from the var declarations in the plugin, and `init` is lifted from the initialisers.

**Evaluator for tests.** A Scala interpreter over `ir.Model`, `ir.Function` and the IR's values in
`model/framework/check`, sufficient for the sample tests: evaluate a guard, an effect and a
predicate on a given state and input. It is a test tool, not a second oracle; the Go checks stay the
oracle for Query answers.

## API Contracts

Kettle, complete:

```scala
object Kettle:
  val boilingTemperature = 100

  object user extends Actor:          // outside the kettle: a test drives these
    val pressButton = action
    val depressButton = action

  object heater extends Internal:     // part of the kettle: a test only observes these
    val heat = step
    val failover = step

  object Model extends Machine[Outcome, Nothing]:
    var temperature = oneOf(-40 to 40)   // any start temperature, as Quint's initNondet
    var heatingOn = false
    var beeping = false

    object transitions extends Transitions:
      on(user.pressButton):
        if !heatingOn then effects.pressButton()
      on(heater.failover):
        if heatingOn && temperature >= boilingTemperature then effects.failover()
      on(heater.heat):
        if heatingOn && temperature < boilingTemperature then effects.heat()
      on(user.depressButton):
        if heatingOn then effects.depressButton()

    object effects:
      def pressButton() =
        heatingOn = true
        beeping = false
      def failover() =
        heatingOn = false
        beeping = true
      def heat() =
        temperature += 1
      def depressButton() =
        heatingOn = false
        beeping = false

    object invariants:
      val beepsOnlyWhenOff = if beeping then check(!heatingOn)
      val neverBeeps = check(!beeping)

    object postconditions:
      val warmsOnlyWhenOn = if !before.heatingOn then check(temperature <= before.temperature)
      val failsOverAtBoiling = after(heater.failover):
        check(temperature == boilingTemperature)
        check(beeping)

    object progress:
      // left to itself the kettle beeps within 81 steps; the user is out, since a depress
      // disables heat rather than leaving it enabled, so fairness on heat would not help
      val boils = leadsTo(from = heatingOn, to = beeping, within = 81) in only(heater.heat, heater.failover)
```

An effect with an input, from Bank: `def send(s: Send) = …` and
`on(client.send): s => if s.amount <= supply then effects.send(s)`. A phase table, from the
activity's retry facet (the largest transitions block in the repo), is the contract for `match` transitions:

```scala
on(activityWorker.respondFailed(Failure.retryable)):
  phase match
    case Phase.started if retriesRemaining                 => effects.backOff()
    case Phase.cancelRequested                             => cancellation.effects.cancel()
    case Phase.pauseRequested if retriesRemaining          => effects.backOffPaused()
    case Phase.started | Phase.pauseRequested if exhausted => effects.fail()
    case _                                                 => ()
on(completionClient.respondFailedByID):
  phase match
    case Phase.unstarted | Phase.scheduled | Phase.paused  => reject(Rejection.notFound)
    case _: Closed                                         => reject(Rejection.notFound)
    case Phase.resetRequested | Phase.resetKeepingPause    => reset.effects.applyReset()
    case _                                                 => ()
``` Kettle has no `queries` section: every claim is checked by default, and its negative control
lives in `Kettle.test.scala` as `assertRefuted(Model, invariant(check(!beeping)))`. A scenario,
where one is needed, names the vars it changes from the start with `starts(temperature = 20)`; a
scenario with no `only`/`actions` admits every action.
A result, exact shape: `<machine>.<check>: <verified|found|refuted|unknown|deadlock|cycle|missed
deadline> (budget <states> states, <depth> steps; exploration <complete|cut>)`.

Compiler refusals, exact texts:

- assignment to a state var outside an effect: `heatingOn is state: it is assigned in an effect only`
- read of a state var outside a lifted scope: `heatingOn is state: read it in an effect, guard, expectation, monitor or refinement`
- a claim member with no `check`, `records` or `oneOf`: today's text, `a requirement block must
  not be empty or contain only helper vals`
- `before` used outside `postconditions`: `before is the state a step started from; use it in a
  postcondition`
- a `verify`/`find` naming something that is not a claim: `verify takes an invariant, a
  postcondition or a progress claim`
- `action` outside an `Actor` object, or `step` outside an `Internal` object: `an action is
  declared in an Actor object; a step the system takes on its own in an Internal object`
- `in` on a claim with something that is not a scenario: `in takes a scenario: only(...),
  fair(...), starts(...) or a named one`
- `oneOf(domain)` as an initialiser with a domain that has no schema: the var schema refusal
- vacuity warnings, shapes: `<machine>: <action> never fires` and `<machine>.<postcondition>:
  <action> never fires, the postcondition is vacuous`
- `limits` or `Limits` in a Model: `a bound is a promise (within) or a budget (the checker's
  configuration); a Model declares no Limits`
- a `leadsTo` outside `progress`, or a `check` block in `progress`: `a progress claim belongs in
  progress` / `an invariant or postcondition belongs in invariants or postconditions`
- the cycle report of a failed progress claim, shape: `<query>: cycle <a>, <b>, …: <action> stays
  enabled and is never taken; add fair(<action>) to the scenario, or admit fewer actions with only`
- a realization naming a query whose scenario has `only` or `fair`: `<query> runs on the model
  only: its scenario restricts or assumes the environment`
- a transition leaf that is not an effect call, `reject(r)` or `()`:
  `a transition ends in an effect call, a rejection or (): <what it found> is none of these`
- an assignment to a state var inside `transitions`: `a transition says when; write the state change in effects`
- a state var whose declared type has no schema: `temperature: Double has no IR schema; use Int, a bounded alias, Boolean, an enum, a record, Option, Set, Map, List or Inbox`

## Edge Cases & Constraints

- Sequential reads: `a = 1; b = a` lifts `b` as `1`. A write to a var followed by a read of it in
  a guard of a nested `if` in the same block is substituted the same way.
- `choose` arms are blocks; each lifts to its own `copy`. The step-cardinality rule for helpers
  called from alternatives is unchanged.
- A helper `def` in `effects` or a `Lifted` object that assigns to state is an effect fragment and
  lifts inline where called; a helper that only reads is a function as today.
- A var declared in a trait (facet) is abstract and defined once in the Machine; two facets
  declaring the same var name is an ordinary Scala error.
- Records in state (`var order: Order`) are updated with `order = order.copy(…)` or field-wise
  `order.paid = true` if the record's fields are vars; both lift to nested `copy`.
- Collections: `seen += x` on a `Set` var is `seen = seen + x`, which lifts to the existing
  collection operations.
- Transitions: first match wins; the lifter lifts an `if`/`match` chain as nested `if` in the IR,
  guards in source order. An effect is always a named `def` in `effects`; a transition leaf calls it. A `match` with a missing phase is Scala's exhaustiveness error under the
  strict flags. The overlap lint reports non-exclusive arms as a warning. `choose` inside a leaf
  keeps the one-step-per-alternative rule.
- Inputs are never ambient; they are parameters, so a transition with an input is `on(a): in => …`.
- A claim member lifts as the requirement list it is today: `check`s in order, short-circuit on
  the first failure, each reported on its own; `if` guards the checks under it; `oneOf`/`branch`
  are alternatives. In `postconditions`, `before` lifts to the step's first parameter and bare
  names to the second; in `invariants`, `before` is refused.
- A claim that needs a hole (`???`) for an illustration keeps it; the lifter treats the hole as
  today.
- `fair(a)` on an action with classes makes every class fair; a composition's scenario selects
  composed steps for `only` and `fair` as `actions` does today (`c.synced(...)`, `c.own(...)`).
  `only` with a `starts` keeps the start; `only` and `actions` on one scenario is refused.
- A verified progress answer reports the longest path from a `from` state to a `to` state, so a
  loose `within` is visible.
- Every claim is checked by default, named `<machine>.<claim>`, from every start, under its own
  `in` environment if it has one; an explicit `verify`/`find` in `queries` is an additional check
  (a witness, a pinned path, a Case source), never a replacement. A realization names a claim or a
  `queries` check by its `val`.
- `oneOf` starts multiply the reachable graph by the number of starts; the caller's budget, not
  the Model, bounds that. The IR evaluator for tests picks a start by named updates.
- The Go explorer must iterate the start list rather than read its head; verified in fn-190.7.
- The gate's budget configuration lists a default and per-feature overrides; the samples run on
  the default. A result whose exploration was cut reports unknown and the budget; the inventory of
  intended answers records the budget each answer was produced under.
- Monitors, refinement maps and evidence read vars by name inside their blocks and lift as today.
- Tests that need a concrete state build it through the IR evaluator from `init` plus named
  updates; there is no Scala state value to construct.
- Backward compatibility: none owed. Retained oracles under `export/testdata` are regenerated once
  from the rewritten samples; the equivalence claim is on Query answers through the Go checks when
  they run, and on the reachable graph otherwise.
- The editor without the plugin: a Model type-checks and constructs its declarations; nothing
  evaluates. Same as today for IR.

## Acceptance Criteria

- **R1:** A design note in `model/design` records a prototype of the plain-Scala form on Kettle
  alone: vars, assignment-block effects, bare reads, `before(x)`, `UpTo` as an erasing alias, and
  a scenario start by named updates, the four claims as members of `invariants` and
  `postconditions` with no kind word, `after(action):` in a postcondition, `before` as the State
  record, the `boils` progress claim with named `leadsTo` parameters and `in only(...)`, a
  `oneOf(-40 to 40)` start, no `queries` section, the negative control as a test, `heater extends
  Internal` with `step` and `action` without `this`, the four transitions as
  `if` and one as a `match` on a phase var with `case a | b if g`, `case _: Role`, `reject` and
  `case _ => ()`, lifted by the plugin to the existing IR shapes (`field`,
  `copy`), with the prototype kept under `.flow/tmp` with a README and a rerun script. It states the
  verdict on sequential-read substitution and on bounded aliases, and what the IR type name is.
  Errors: a prototype finding that one of those does not work is recorded as a verdict with the
  alternative chosen; the note is not optional.
- **R2:** `Machine[O, F]` declares state as vars; the plugin synthesises the record `<machine>.State`
  from the vars in declaration order and lifts `init` from the initialisers; a Model has no
  author-written state class. Errors: the third refusal text in API Contracts for a var with no
  schema; a Machine with no vars is refused with `a Machine declares at least one state var`.
- **R3:** Effects are methods whose bodies assign to state vars; the lifter produces one `copy` per
  path with sequential substitution; `choose`, `if`, `match`, `reject`, `records` and `crash`
  work inside; an effect with an input takes it as a parameter. Errors: the first refusal text in
  API Contracts for an assignment outside an effect; an assignment to a non-state var or a
  parameter is ordinary Scala and is not lifted (refused as "not state" at the lift).
- **R4:** Guards, same-step expectations, transition expectations (`before(x)`), monitors,
  refinement and evidence read vars by name and lift to `field` reads; `after.state` and the
  `s =>` parameter forms are gone from the DSL. Errors: the second refusal text in API Contracts.
- **R5:** Facets declare their phase and any shared state as abstract vars; `Phased`, the projection
  and writer, and the plugin's contextual-phase machinery are removed; the Facets fixture is
  rewritten and lifts. Errors: a facet var the Machine does not define is Scala's own abstract
  member error.
- **R6:** The DSL executes nothing: `effectOf`, `guardHolds`, input-effect adapters, `Finite`
  enumeration, `StateData`, `Domain` runtime and native budget runtime are removed; a binding holds
  a reference for the plugin only. Errors: none beyond compile errors for removed vocabulary.
- **R7:** An IR evaluator in `model/framework/check` evaluates guards, effects and predicates of an
  `ir.Model` on IR values, and the four native sample suites (Expect, NativeHistory, NativeMaps,
  NativeVoting) pass against it with their assertions preserved. Errors: an IR construct the
  evaluator does not support fails the test naming the construct; no silent skip.
- **R8:** All six samples and the native-migration fixtures are rewritten in the plain form,
  follow the sections convention, compile with the plugin under the native strict flags, and lift.
  Retained oracles are regenerated once; the reachable graphs and, when the Go checks run, the
  Query answers of the six samples are unchanged from the previous oracles. Errors: a changed
  answer is a finding, not a regeneration.
- **R9:** Bounded forms are erasing aliases carrying their bound in the type tree; the schema of a
  `var x: UpTo[140]` is the same `intRange` as before. Errors: an alias argument that is not a
  literal is refused at the declaration.
- **R10:** fn-189 is reconciled: its R4 task is superseded by R2 here (schema from declarations in
  the plugin, `init` lifted), its facet boundary is closed by R5, and its counts receipt is
  re-recorded after this spec. No error surface.
- **R11:** Claims are members of `invariants` and `postconditions`: blocks of `check`
  statements with no kind word, `after(action):` the only word in a postcondition, declared as
  `val`s (a `def` only when parametrised), evaluating nothing at construction; bare names read the
  state now and `before` is the State record in a postcondition; `property`, `expect`, `across`,
  `invariant`, `transition`, `after.state` and the `properties`/`safety`/`liveness` groups are
  retired; `check`, `oneOf`, `branch`, `records` and holes stay; a check refers to a claim as
  `invariants.x`, `postconditions.x` or `progress.x`. Errors: the claim refusal texts in API
  Contracts, including `before` in an invariant.
- **R12:** The section is `transitions` (renamed from `rules`; marker trait `Transitions`); an `on(action)` block is plain control flow: `if`/`match` chains whose leaves are a
  call to a named effect, `reject(r)` or `()`, never an assignment (effects and transitions stay
  separate, enforced); `case a | b`, `case a if g`,
  `case _: Role` and `case _ => ()` are the phase-table forms; an input is the block's
  parameter; first match wins and lifts as nested `if`; `where`, `~>`, `when`, `always`,
  `rejects` and `enters` are retired from the public DSL; `disabled` stays. The retry block in API
  Contracts lifts to the same arms as its `when` form did, except positions. Errors: the leaf
  refusal texts in API Contracts; a non-exhaustive `match` is the compiler's error.
- **R13:** A progress claim is `leadsTo(from = p, to = q, within = n)` in the `progress`
  section, named after its `val`, carrying no assumptions; the Scenario builder gains
  `only(actions)` and `fair(actions)`, composable with `starts`; a progress claim is checked by
  `verify … in …` and answers verified with its longest path, deadlock, cycle
  with the never-taken action and both remedies, or missed deadline, in the shapes in API
  Contracts; `assume`, `Assumption`, the assumption vararg, the name string and `restrict` as the
  progress idiom are retired; nothing is fair by default. Kettle carries `boils` and
  `heatingAlone`. Errors: the placement refusals and the model-only realization refusal in API
  Contracts; `within` not a positive literal is refused at the declaration; `only` together with
  `actions` on one scenario is refused.
- **R14:** There is no `Limits` in the model: `query`, `limits` and `Limits` are retired from the
  DSL and the bounds field from the IR `Query` message; the checker takes its budget from the
  gate's configuration (default plus per-feature overrides outside the Model), every result carries
  the budget and completion in the shape in API Contracts, and a cut exploration answers unknown.
  The six samples' intended answers are reproduced on the default budget. Errors: the `Limits`
  refusal text in API Contracts; a per-feature override naming no feature is a gate configuration
  error.
- **R15:** Every claim is checked by default from every start over the reachable graph, under its
  own `in` environment when it has one (inline `only`/`fair`/`starts` or a named scenario); a var
  initialiser may be `oneOf(domain)` over an ordinary Scala collection or range, and the set of
  starts is the product, lifted to the IR start list; the checker reports vacuity (actions that
  never fire, `after(action)` postconditions never triggered) in the shapes in API Contracts;
  negative controls and the derived negative-control machines are tests, not Models; `queries`
  holds only witnesses, pinned paths and realization checks, and the six samples have no `queries`
  section unless a witness needs one. Errors: the `in` and `oneOf` refusal texts in API Contracts;
  a claim named by a realization that has an `only`/`fair` environment keeps the model-only
  refusal of R13.
- **R16:** Actions are declared inside their owner: `action` (no argument) in an `object … extends
  Actor`, `step` in an `object … extends Internal`; `internal` and `action(this)` are retired; the
  six samples and the fixtures use the two containers. Errors: the placement refusal text in API
  Contracts.

## Boundaries

- No change to the IR schema beyond type naming; `field` and `copy` are the carriers.
- Go consumers change only where R14 and R15 require: the checker takes a budget and reports it
  with completion, iterates the start list, and reports vacuity; the IR `Query` loses its bounds
  field. No other Go change.
- Temporal sources stay disabled; the kit is not migrated here.
- The IR evaluator is a test tool; it is not a checker, not a second oracle and has no CLI.
- Declaration-site naming of commands (`scripted(alias)`) stays as it is.
- No new syntax: no `state()`, no `:=`, no macros for assignment.

## Decision Context

Three forms were considered. `state()` fields with `:=` keep the state immutable at runtime and
need no evaluator, but add vocabulary the Quint does not have and keep a runtime that nothing in
production uses. Plain vars with load/store around each call keep in-process execution, at the
cost of a state record derived from vars at runtime, a single-state-at-a-time Machine and a
`before` thunk. Plain vars with no execution, chosen here, is the only one of the three where the
author writes nothing but Scala and the framework keeps nothing but declarations; the cost is an IR
evaluator for the sample tests, which is a bounded interpreter over a closed IR and is useful on
its own for debugging lifts.

The reason the no-execution choice is available at all is that the Scala step tables were retired
and the owner made the Go checks the oracle; this spec follows those decisions to their end.

fn-189 is not blocked by this spec and most of it still applies: single compilation, one carrier,
`given Family`, `Lifted`, uncurried helpers, inline sugar, `Captured[F]`. Its R4 is re-cut by R10
here rather than done twice. fn-191 moves today's samples into sections first, so R8 here changes
bodies, not layout.

Sections by kind (R11, owner 2026-10-16): the `safety`/`liveness` grouping under `properties`
was replaced by sections named for what a developer has to understand to write one, `invariants`,
`postconditions`, `monitors`, `progress`; design-by-contract vocabulary, with the `transitions` guards
as the preconditions. The kind word per member went with it: the section says it, as `effects`
does. `postconditions` was chosen over `transitions`/`steps`/`changes` because developers know
the word and `after(action)` reads as the postcondition of that action.

Fairness (R13, owner 2026-10-16, after stepping back): every earlier form attached fairness to
the claim, including an `assuming = fair(a)` parameter drafted the same day. In TLA⁺ fairness is
part of the environment specification, not of the property, and the DSL already has the object
that describes a check's environment, the Scenario. So `only` and `fair` are scenario builders,
the claim is a promise with a bound, and a result names its scenario, which keeps the assumption
traceability the old `Assumption` names gave. Implicit fairness of `internal` actions was
considered and rejected as a hidden assumption. `Derived(Model.restrict(…))` is no longer needed
for progress; `rebind` keeps the negative controls.

Checks and budgets (R14/R15, owner 2026-10-16): Quint records predicates in the model and the
question (which invariant, how many steps) on the command line; we record the question because a
result is an artifact realizations and Cases depend on, but the common question was being written
by hand on every claim. A check adds only an environment and a Case to a claim; both have
defaults, so every claim is checked by default and `queries` keeps only what a default cannot
express. A nondeterministic start (`oneOf`, the design note's finding 7) removes the last reason
Kettle had a scenario; vacuity reporting removes the hand-written `find`; negative controls test
the checker, not the Model, and move to tests. `oneOf(domain)` takes a plain Scala collection or
range, not a generator: a start set is enumerated exhaustively by the explorer, where a
property-based generator samples; ScalaCheck-style `Gen`/`Arbitrary` were considered and rejected
for starts, since sampling is the wrong semantics for a model check, but they are the right tool
for the IR evaluator's tests (fn-190.3), which may use them. `Limits` mixed a promise
(`within`, kept on the progress claim) with a budget (depth, schedule length, state count), and
since the reachable graph is the only table a budget no longer changes what verified means; it is
the caller's, recorded on every result, with a cut exploration answering unknown.

Actors and internals (R16, owner 2026-10-16): `object heater:` holding `internal` vals made the
actor/system distinction invisible, and that distinction is what decides whether a Case performs
a step or waits for it. Naming the other container, `Internal`, and letting both containers say
what their members are (`action`, `step`) applies the same rule as the claim sections.

Prototype items, not criteria (fn-190.1 tries them on Kettle and the Facets fixture): `object
transitions:` without `extends Transitions`, with facet order taken from the Machine's `extends` clause instead
of `Facets(...)`; samples as top-level definitions under `package samples.<name>` with the Machine
named after the sample; outcomes and facts as member types with defaults so the plain case is
`extends Machine`. Each becomes a criterion only if the prototype finds it clean.

Property syntax (R11, owner 2026-10-16): a Boolean form (`invariant(beeping implies
!heatingOn)`, `transition(before(x) || …)`, `after(a)(…)`) was drafted and rejected as too clever
for regular developers: `implies`, implication spelled as `||`, and a second argument list are
logicians' moves. A block of `check`s reads like a unit test, which every developer has written,
keeps per-requirement diagnostics without any `&&` splitting, and leaves only the kind word and
`before` to learn. A default `Limits` through a `given` was rejected for the same reason: explicit
`limits` on each query is one more word and no hidden source.

Transitions as control flow (R12, owner 2026-10-16): `rules` is renamed `transitions`, the state-machine word, which also frees "rule" for the lints. `where`/`~>`/`when` existed to hand the lifter a
guard and an effect separately; with the plugin reading the tree, `if`/`match` give it the same
arms, Scala's exhaustiveness check replaces the gap lint for phase tables, and the largest transitions
block in the repo (the activity retry facet) reads as a state table. First-match order is a
semantic change from today's disjoint arms, accepted because it is what a Scala developer expects;
the overlap lint stays as a warning. Inline assignment blocks as rule leaves were considered and
rejected (owner, 2026-10-16): effects and transitions stay separate so "what a step does" and "when it
fires" read apart, an effect keeps a name in the IR, and facets and `rebind` share effects by name;
a lint enforces it rather than a style rule.

Order of work: R1, then R2+R9 and R7 in parallel, then R3+R4+R11+R12+R13+R14+R15+R16, R6, R5, R8, R10.

## Requirement coverage

| Req | Task(s) | Gap justification |
| --- | --- | --- |
| R1 | fn-190.1 | — |
| R2 | fn-190.2 | — |
| R3 | fn-190.4 | — |
| R4 | fn-190.4 | — |
| R5 | fn-190.6 | — |
| R6 | fn-190.5 | — |
| R7 | fn-190.3 | — |
| R8 | fn-190.7 | after fn-191.1 puts the samples in sections |
| R9 | fn-190.2 | — |
| R10 | fn-190.2, fn-190.8 | — |
| R11 | fn-190.1, fn-190.4 | — |
| R12 | fn-190.1, fn-190.4 | — |
| R13 | fn-190.1, fn-190.4 | — |
| R14 | fn-190.1, fn-190.4, fn-190.7 | the Go budget argument and result shape land with the sample rewrite in .7 |
| R15 | fn-190.1, fn-190.4, fn-190.7 | start-list iteration and vacuity reporting in the checker land in .7 |
| R16 | fn-190.1, fn-190.4 | — |

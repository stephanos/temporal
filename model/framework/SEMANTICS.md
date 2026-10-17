# Modelling with the Umpire DSL

This is the design guide to the DSL in `model/framework`: how a Model is built, one concept at a
time, in the order you write it. Each step says what the concept *means* (what the Go reader does
with it), how it is *written*, and what is *refused* and by whom. The README covers the pipeline,
the file layout and the commands; this guide covers the language.

A Model is ordinary Scala 3. The compiler checks its types, the framework checks its rules as they
are constructed, the IR generator (`model/framework/export`) reads the compiled trees and refuses what the IR
cannot hold, and the Go reader (`tools/umpire`) derives every table and answers every question. The
IR carries no Scala: everything a reader knows is in the feature's `generated/ir.json`.

The running example is the committed **mini activity**, `model/temporal/features/activity_mini/`:
its Product says what a client sees, its Protocol adds attempts, backoff and dispatch designs,
and `realization/MiniActivityRun.scala` binds two live Queries to typed commands and evidence.
Every Scala block below is a cited source excerpt, read in its original package and imports;
excerpts are not standalone programs. The feature header names its complete hard-instance inventory.
The shared framework and Closable definition are cited where the mini consumes their contracts.

The mini's `.live` values are authored expectations, not observed Verdicts. Its source compiles
and lifts; Case lowering and live execution remain held in fn-166.4, the complete Make loop in
fn-166.5, and production custom-rule loading in fn-166.10. Real features remain behind during the
Scala DSL batch. [README, Writing a Model](../README.md#writing-a-model) documents the focused route.

Contents:

1. [Types](#1-types): finite domains, catalogs, counters
2. [Phases and roles](#2-phases-and-roles): the lifecycle vocabulary
3. [The signature](#3-the-signature): entities, actors, actions, inputs, timers
4. [The machine](#4-the-machine): the object, its header and its sections
5. [Effects](#5-effects): what a step is and how it is written
6. [Rules](#6-rules): when each action fires
7. [The table](#7-the-table): what the reader derives, and what lint reads off it
8. [Properties](#8-properties): the promises
9. [Scenarios and Queries](#9-scenarios-and-queries): the bounded questions
10. [Monitors and assumptions](#10-monitors-and-assumptions)
11. [Progress](#11-progress): bounded liveness
12. [Holes](#12-holes): unknown on purpose
13. [Refinement](#13-refinement): Product and Protocol
14. [Derivations](#14-derivations): one thing changed
15. [Facets](#15-facets): one machine, assembled
16. [Capabilities](#16-capabilities): shared promises by binding
17. [Compositions](#17-compositions): several entities, one Model
18. [Channels](#18-channels): messages in flight
19. [Realizations](#19-realizations): running a path against a server
20. [What the tools answer](#20-what-the-tools-answer): results, levels, admission
21. [Core and sugar](#21-core-and-sugar): the shape of the DSL itself

---

## 1. Types

**Meaning.** Everything a machine ranges over is finite and enumerable. A *catalog* is the ordered
list of a type's values; the reader builds every table by walking catalogs, so the order is part of
the semantics:

- `Boolean` is `false, true`;
- an integer range `0..n` is its integers ascending;
- an enum is its cases in declaration order, a case with fields once per assignment of them;
- a case class is the product of its fields in declaration order, the last field varying fastest;
- `Option[A]` is `None`, then `Some(a)` for each `a` in order;
- a channel's contents are a catalog of their own ([Channels](#18-channels)).

Equality is structural. Values are keyed in the IR by spelling: a case is its name, a case with
fields `C-v1-v2`, a record `v1-v2`, a list `[v1,v2]`.

**Writing it.** Plain enums and case classes; bound integers with `UpTo[N]`:

Source: [temporal/features/activity_mini/protocol/Protocol.scala:24](../temporal/features/activity_mini/protocol/Protocol.scala#L24).

```scala
final case class State(phase: Phase, attempts: UpTo[2], scheduleToStart: Timeout,
    startToClose: Timeout, maxAttempts: MaxAttempts)
```

Nothing is declared about finiteness: every enum and case class is a domain, and the compiler
derives its catalog (`framework.dsl.Finite`) where one is first needed, at the `Machine`, `input` or
`Refinement` that takes the type. `UpTo[2]` has the values 0, 1, 2 and reads as an `Int`; write one
with `UpTo(n)`. A type whose catalog is not structural declares its own `given Finite[T]`, which
wins over the derived one: a named opaque integer type with its own `Finite.upTo(n)` range,
or a state that holds a channel's `Inbox` (step 18). A plain `Int` field is refused at its declaration,
even with a local `Finite[Int]`: write `UpTo[N]`. Optional bounded values retain `N` in their
synthetic enum identity, so `Option[UpTo[1]]` and `Option[UpTo[3]]` have independent catalogs.

The build runs with `-language:strictEquality`, so comparing two values needs a `CanEqual`. Every
`Finite` type gets one from `import framework.dsl.Finite.given`; a file that only uses it in a
comparison keeps the import alive with `val _ = () => summon[CanEqual[Phase, Phase]]`. Comparing
two levels' phases (`product.Phase == protocol.Phase`) does not compile, which is the point.

**Refused.** A state field with no finite catalog (a list, an unbounded `Int`), by the compiler
at the first use that needs the catalog (`State has no finite catalog`, with the explanation
ending at the member that has none), and by the reader at admission; a type that is neither an
enum nor a case class, by the IR generator. A comparison of two values typed by a *role* rather
than by the phase (`if c then Phase.paused else Phase.scheduled` is a `Phase & Live`) needs the
phase spelled, `val landing: Phase = ...`, since only the phase has a catalog.

## 2. Phases and roles

**Meaning.** A *phase* is the field of the state that says where the entity is in its lifecycle. A
*role* is what a phase says about the entity, independent of the entity: alive or over, waiting or
held. Roles let shared Properties ([Capabilities](#16-capabilities)) and rule headings speak of
"any closed phase" without naming the entity's cases. A role is metadata: it adds no state, and the
IR has no role construct. Every role test lowers to membership in the listed cases that carry it.

The roles (`framework/dsl/Roles.scala`):

| Role | Means | Narrower roles |
| --- | --- | --- |
| `Live` | exists and is not over | `Waiting` (no attempt held), `Held` (an attempt is held), `Suspended` (paused); `Retrying extends Waiting` |
| `Closed` | is over | `Succeeded`, `Failed`, `Canceled`, `Terminated`, `TimedOut` |

A phase with no role is one where the entity does not exist yet (`unscheduled`).

**Writing it.** A case takes roles in its own extends clause. A phase enum may also declare, per
case, the *status fact* a step that enters it records:

Source: [temporal/features/activity_mini/protocol/Protocol.scala:14](../temporal/features/activity_mini/protocol/Protocol.scala#L14).

```scala
enum Phase(val status: Fact) extends Recorded[Fact]:
  case scheduled extends Phase(Fact.statusScheduled), Waiting
  case backingOff extends Phase(Fact.statusScheduled), Retrying
  case started extends Phase(Fact.statusStarted), Held
  case cancelRequested extends Phase(Fact.statusCancelRequested), Held
  case completed extends Phase(Fact.statusCompleted), Succeeded
  case failed extends Phase(Fact.statusFailed), Failed
  case canceled extends Phase(Fact.statusCanceled), Canceled
  case timedOut extends Phase(Fact.statusTimedOut), TimedOut
```

A role test is written dotted: `s.phase.in[Closed]` on a value, `when[Closed]` in a rule, and
`case _: Closed` in a pattern. A Model may declare a role of its own by extending one:
`trait Expired extends TimedOut`.

**Refused.** A case that is both `Live` and `Closed`, two of `Waiting`/`Held`/`Suspended`, or two
closure roles (IR generator, at the enum); `isInstanceOf` (lint); a role test against a type that
is no role, or a role no case has (IR generator). A `Recorded` case without a status does not
compile, since the parameter has no default.

**Design note.** Roles replace the named phase sets Models used to declare (`terminal`, `paused`).
A `states` predicate that only renames a role is not written; rules use `when[R]` and expressions
`p.in[R]`. Composite sets such as `created` (`Live` or `Closed`) stay named predicates.

## 3. The signature

The signature is who acts and what they can do. It is declared once per feature, at the top level
of the feature file, and every level's machine binds the same actions.

### Entities and observations

An *entity* is what a machine keeps state for; a realization keys recorded evidence to it.

Source: [temporal/features/activity_mini/MiniActivity.scala:37](../temporal/features/activity_mini/MiniActivity.scala#L37).

```scala
val activity = Entity(key = "activityId")
```

An observation stands for a value no recorded event carries (an attempt count read back through a
describe call); the realization's typed read does the reading.

### Actors and actions

An *actor* is an object whose members are the actions it takes. `Actor.system` is reserved for the
steps the system takes of its own: timers, internal steps, channel deliveries.

Source: [temporal/features/activity_mini/MiniActivity.scala:42](../temporal/features/activity_mini/MiniActivity.scala#L42).

```scala
object client extends Client:
  val start = action(this)
    .input(scheduleToStart)
    .input(startToClose)
    .input(maxAttempts)
    .creates(activity)
  val cancel = action(this).on(activity).results("Delivery")
```

Every declaration is named after its `val`; a name is written as a string only where no `val`
declares it or it must differ (`action("start", Actor("caller"))`). A feature that adds actions to
an actor another feature declares puts them in an object named for that actor and imports the
shared one under another name (`import actors.worker.{worker as process}`).

### Inputs and classes

An action may take finite *inputs*. Each assignment of the inputs is one *class* of the action:
`activityWorker.respondFailed(Failure.fatal)` and
`activityWorker.respondFailed(Failure.retryable)` are two classes, and the table has one column per
class. An action with no input has one class.

Source: [temporal/features/activity_mini/MiniActivity.scala:38](../temporal/features/activity_mini/MiniActivity.scala#L38).

```scala
val scheduleToStart = input[Timeout](default = Timeout.unset)
val startToClose = input[Timeout](default = Timeout.unset)
val maxAttempts = input[MaxAttempts](default = MaxAttempts.one)
```

The mini selects its two-attempt start by `client.start(maxAttempts := MaxAttempts.two)`
in `protocol/Protocol.scala:73`; omitted inputs take their token's explicit default.
`client.start()` in `protocol/Protocol.scala:78` uses Timeout.unset, Timeout.unset and MaxAttempts.one.
`Input[A]` carries `Option[A]`; `input[A]` without a default is required, even when its catalog is
nonempty. Defaults can be finite records, field-bearing enum cases and Option values, including
false and zero. Catalog order governs enumeration, never omission. `:=` is sugar for the positional call. An action that takes an input may carry
*examples*, `.example(Reply.handlerError(false), "BadRequest")`: the author's claim that every
realized value of that class behaves alike, with the one the generated Case runs.

**Refused.** A token that is not the action's, or supplied twice (framework, at the call); a value
of the wrong type (compiler); a step function whose arity does not match the action's inputs
(compiler: `~>` is typed per arity). An empty or partial named call omitting a required input
refuses with its actual declaration name while the containing machine, Scenario or realization
initializes; the lifter refuses independently at the call's source position. Named string slots
without tokens stay required. Zero-input actions and timers still have their one empty class.
An explicitly supplied required non-Timeout input in a deadline preset needs no default; Timeout
inputs must declare the default against which the deadline contract checks the preset.
The deadline helper requires semantic `unset` as that default and selects `expires` by name,
independent of enum order. An `expires` default remains valid for ordinary action classes.

### Outcomes

Every step answers an *outcome*. A machine declares its own enum, or adopts the shared one:

Source: [temporal/features/activity_mini/product/Completion.scala:8](../temporal/features/activity_mini/product/Completion.scala#L8).

```scala
import framework.dsl.outcomes.{Outcome, Rejection}
```

`enter` and `stay` answer the outcome one `given Ok[Outcome] = Ok(Outcome.accepted)` names,
declared once beside the outcome type; the shared outcomes come with theirs. A machine that answers
nothing a caller reads (a worker, a queue) still names an outcome type, since every step has one.

## 4. The machine

**Meaning.** A machine is a transition relation over a finite state type: for every state and every
action class, the list of steps that class may take there. Its start, its ends, and the steps are
all the reader needs to build the table ([The table](#7-the-table)).

**Writing it.** A machine is an object that *is* the machine, named after the object with its first
letter lowered (`object ActivityProduct` is `activityProduct`), stating its three types once:

Source: [temporal/features/activity_mini/product/Product.scala:28](../temporal/features/activity_mini/product/Product.scala#L28).

```scala
object ActivityProduct
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Dispatching,
      Completion:
  val init = product.State(Phase.scheduled)
  override def end(s: State) = s.phase.in[Closed]
  val entity = activity
```

The header:

- `Phased[State, Phase](projection, writer)` names the phase field once. Rules may then say
  `when(Phase.x)`, `when[Role]`; `enters(Phase.x)` writes through the writer. A machine with no
  phase is a plain `Machine` and its rules use `where` and `always` only.
- `init` is the one start state (named as TLA+ and Quint name it).
- `end` says where a path may stop. With `Phased` the default is "phase has a `Closed` role"; a
  machine whose stopping point differs overrides it. A machine with no `Closed` case and no
  explicit `end` is refused.
- `evidence` lists only the exceptions: a fact is confirmed by evidence of its own name unless a
  line says otherwise, and a fact case with fields needs a line that covers all its values.
- `entity` is derived from the actions' `on`/`creates` and is written only where that is ambiguous;
  `unobservable` lists timers whose step records nothing a Run can read.

The sections are objects named for what they hold and come in the order above; the source checks'
declaration-order lint refuses any other order and any section nested elsewhere. The two designated
claim groups are the exception: `safety` and `liveness` sit directly inside `properties`. Inside the
object `State` names the state type and `property`, `scenario` name the machine implicitly.

**Refused.** Two machines of one name in a package; a machine whose object name differs from its
file's (structure lint); a section object of any other name; a `val` read before it is declared
(the initialization-order lint, since an object's `val`s initialize top to bottom).

## 5. Effects

Native var-based Models declare their effects as methods in `object effects`. Every method there,
including an unused or private helper, must return exactly `Unit`. Infer the result type and write
the body on the following line; the author lint rejects explicit return annotations and inline
bodies. Parameter type annotations are allowed. The record-state forms below remain reference for
the held integration work.

**Meaning.** A *step* is one result of an action in one state: the outcome it answers, the state it
lands in, the facts it records, and an optional explanation.

Source: [framework/dsl/Machine.scala:10](../framework/dsl/Machine.scala#L10).

```scala
final case class Step[S, O, +F](outcome: O, state: S, facts: List[F] = Nil, because: String = "")
```

A step function gives a *list* of steps: empty means the action is disabled there, one step is the
ordinary case, several steps are a branch the system may take either way. Only a step function and
the functions it calls may make a step; a start, an `end`, a Property or a monitor that constructs
one is at the wrong *level* and is refused ([Levels](#levels)).

**Writing it.** The core form is `List(Step(...))`; every Model writes the sugar:

| Form | Core form | Use |
| --- | --- | --- |
| `enter(s.copy(phase = p), f1, f2)` | `List(Step(ok, state, List(f1, f2)))` | one ok step recording facts |
| `stay(s)` | `List(Step(ok, s))` | keep the state, record nothing |
| `reject(Outcome.rejected(why), s)` | `List(Step(outcome, s))` | keep the state, answer otherwise |
| `rejects(Rejection.notFound)` | `s => reject(Outcome.rejected(notFound), s)` | the same as a rule's effect, shared outcomes |
| `steps.because("...")` | `Step(..., because = "...")` | explain the row; no check reads it |
| `disabled` | `Nil` | no step; in an effect it is refused, in a rule use `disabled(a)` |
| `enters(Phase.p)` | `enter(s.copy(phase = p), p.status)` | the step that only enters a phase; records the status its case declares, if any |

Effects live in `object effects` as defs of the state (and the class's inputs), named for what
they do and never giving `Nil`:

Source: [temporal/features/activity_mini/protocol/Dispatching.scala:24](../temporal/features/activity_mini/protocol/Dispatching.scala#L24).

```scala
def schedule(s: State, scheduleToStart: Timeout, startToClose: Timeout, maxAttempts: MaxAttempts) =
  enter(s.copy(scheduleToStart = scheduleToStart, startToClose = startToClose,
    maxAttempts = maxAttempts), Fact.statusScheduled)
def startAttempt(s: State) = enter(s.copy(phase = Phase.started,
  attempts = UpTo((s.attempts + 1).min(2))), Fact.statusStarted)
```

An effect of the state alone may be a block, which reads and assigns fields by name through the
accessors the level file declares (`def phase(using v: View[State]) = v.get(_.phase)`,
`def phase_=(p: Phase)(using d: Draft[State, ?, Fact]) = d.set(p)(_.copy(phase = p))`):

The mini writes named effect defs rather than `effect` or `is` blocks; the accessor and block rules above describe the supported sugar, not an additional mini declaration.

A block holds straight-line field assignments (each field at most once, not read after), `record`
calls, or one `reject`; assigning a `Recorded` phase records its status after the block's own facts.
An effect whose whole work is entering a phase is not declared at all: the rule writes `~>
enters(Phase.done)`.

### Named choices

A step that can go more than one way names each way with `choose`. This is core: the names are
what nothing else says.

Source: [temporal/features/activity_mini/product/Completion.scala:16](../temporal/features/activity_mini/product/Completion.scala#L16).

```scala
val retryScheduled = choice
val retryExhausted = choice
def serviceRetry(s: State) = choose(
```

Each alternative is one step written out, or a call of a function that gives at most one step (an
alternative whose function gives none is not taken). The name rides on the step record as
metadata: no key, equality, Property, monitor or refinement reads it, and the table is the same
with and without names. The IR generator refuses several results written without `choose`
(`List(Step, Step)`, `++`), a `choose` of one alternative (does not compile), a name used twice,
and an alternative that is not one step.

### What a step function may contain

`if`, `match` (case, alternative, binding and wildcard patterns; a wildcard over a whole Model enum
is refused by lint, name the cases), local `val`s, `copy`, constructors, comparisons, arithmetic,
list literals and `++`, calls of other defs, `require` (which becomes the function's precondition:
a call outside it is an error, not a disabled row). A `var`, a loop, or anything else stops the
lift at its line. A function may not call itself.

## 6. Rules

**Meaning.** The IR binds each action to one step function. A Model writes *when* an action fires
and *what* it does apart, and the IR generator lowers the two into that function: the cases of an
action, in the order written, become `if g1(s) then e1(s, i) else if g2(s) then e2(s, i) else Nil`.
Where no case holds, the action is disabled; there is no catch-all.

The rules of one action class are *disjoint*: no state is covered by two of them. The framework
checks this as the rules object constructs, over every state of the catalog and every class, and
refuses the second rule of an overlap naming the machine, the class, both rules and a witness
state. Because they are disjoint, their order changes no row. A step that can go two ways is one
effect with `choose`, never two rules.

**Writing it.**

Source: [temporal/features/activity_mini/protocol/Dispatching.scala:30](../temporal/features/activity_mini/protocol/Dispatching.scala#L30).

```scala
object rules extends Rules:
  on(client.start) {
    when(Phase.scheduled).where(states.firstSchedule) ~> effects.schedule
  }
  on(activityWorker.poll) {
    when(Phase.scheduled).where(ActivityProtocol.states.retriesRemaining) ~> effects.startAttempt
  }
  on(timers.backoff) {
    when(Phase.backingOff) ~> enters(Phase.scheduled)
```

The case headings:

| Heading | Holds while |
| --- | --- |
| `when(Phase.a, Phase.b)` | the phase is one listed; needs `Phased` |
| `when(states.someSet)` | a named predicate of the phase holds |
| `when[Role]` | the phase has the role |
| `where(g)` | `g(s)` holds; the only form a machine without `Phased` has, with `always` |
| `when(...).where(g)` | both |
| `always` | every state |

Rules of spelling, which change no table: every target names its action through its declarer,
`on(activityWorker.poll)`, never bare; every phase through its type, `Phase.scheduled` (no `import Phase.*`);
every `on` is brace-delimited with one case per line; `from(actor) { on(actor.a) { ... } }` only
groups and imports nothing. The old `in(...)` rule heading is retired and refused. The lifter and
native rule construction independently refuse bare action targets, including imported names and
class forms whose arguments are dotted. Each refusal names the target, its source coordinate and
the qualified form. Native construction also refuses imports it can inspect in a `from` body.

The phase type is the `P` of `Phased[S, P]`, including a derived or composed machine's
inherited projection. The lifter refuses wildcard, selective and renamed imports of its members
at every authored level and same-line position. Imports of unrelated enums stay valid.
A bare phase or eta-expanded named set in phase-form `when` is refused; qualified sets and
`when[Role]` remain valid. Property-builder `when` selects actions. Phase-form varargs
`when` has no runtime qualification check.

### Modalities: what a row means

A machine's table says what *may* happen; the claims say what *must*.

- A **row** is permission with fixed results: the class may happen in that state with exactly the
  results listed.
- A **disabled pair** for a system action (timer, internal step) is prohibition: the system does
  not take it there.
- A **disabled pair** for an actor's action is *silence*: the actor can always try, and the Model
  does not say what the system answers. A row with a rejecting outcome and a `because` says it.
- A **hole row** is neither ([Holes](#12-holes)).

Lint reports where these leave a gap (`silent-rejection`, `never-enabled`, `stuck-state`,
[below](#7-the-table)). A rule with a rejecting effect is how a Model is explicit about what it
refuses.

## 7. The table

**Meaning.** From a machine's declaration the reader derives:

1. the states: the catalog of the state type; the outcomes and facts: their catalogs;
2. the action classes: for each bound action, every assignment of its inputs in catalog order,
   keyed `name` or `name-key(i1)-...`;
3. the rows, states-major: for each state and class, the list the step function gives. Empty is a
   disabled pair; a result state outside the catalog is an error; a hole makes a hole row;
4. the starts (the `init`), the ends (the states `end` accepts);
5. the evidence line per fact;
6. the refinement check where one is declared ([Refinement](#13-refinement));
7. the monitors and assumptions it names.

Reachability, stuck states, Definition IDs and the Behavior Fingerprint are computed from this
table. `go run ./tools/umpire/cmd/umpire-lint --tables model/temporal/features/<feature>/generated/ir.json` prints it by class:
each cell MAY with its results, MUST NOT with its guard, or `?` for a hole.

**What lint reads off it** (`make umpire-check-lint`, part of the gate), each at the Scala line:

| Finding | Reported for | Fix |
| --- | --- | --- |
| `never-enabled` | a class no reachable state enables | bind a rule, or drop the action |
| `silent-rejection` | an actor's action disabled in a reachable non-end state | a rule whose outcome says the answer |
| `stuck-state` | a reachable non-end state with no enabled class at all, with a shortest path | the missing timer or internal step, or make it an end |

A finding fails the gate until fixed or accepted with a reason in the feature's `generated/ir.lint.json`.

`@gap(subjects*)(because = "...", until = "...")` is the staged source declaration of a
known lint gap (`framework/dsl/Gaps.scala`). It is metadata, with typed subject references and
one annotation per reason. Source admission checks placement, constructor ownership, a
non-empty subject list and a non-blank literal reason, preserving ordinary compiler annotations.
Extraction keeps authored order and collapses only synthetic object/module mirrors. The lift
resolves each subject to the lint's `(kind, owner, subject)` by placement (README, Declaring
source gaps), refuses a subject that resolves to nothing, a key declared twice and an `until`
naming no unique Flow spec (exact canonical id, else a unique short id `fn-N`; a requirement
suffix is a reference only), and collects `except`/`overriding` waivers as `capability-waiver`
metadata. It serializes none of it to the shared IR; Go still judges the existing acceptance
file. The annotation does not change a machine's rules, a claim's predicate or any IR position.

A derived machine's lift also records its provenance (fn-167.3): the machine its chain starts
from, that machine's own origins, and per operation the bindings kept, rebound and added, the
actions, monitors, refinement and unobservable actions dropped, a replaced refinement and the
assumptions added. It describes the derivation and changes nothing written. A gap on a derived
machine is refused, naming the base, only where every operation provably keeps that finding:
`assuming` keeps every table kind; `unmonitored` keeps the kinds no monitor pins and no refined
state field keys (silent-rejection, disabled-by-default, never-enabled, unproduced,
untaken-choice, unreachable-value); a chain names its root. A rebound or added action's gap is
the derived machine's own; any other operation leaves the finding unproven and the gap kept.
Equal bindings alone prove nothing, and no Go finding is suppressed by this stage.

**Design note.** The table is the one artifact everything else reads; the step tables
(`framework/dsl/StepTables.scala`) dump it from Scala for the equivalence check, so a refactor that
keeps the Models' meaning can be sealed by comparing tables while the IR is stale.

## 8. Properties

**Meaning.** A Property is one promise, read on steps. A *same-step* Property reads one step record
(outcome, state after, facts); a *transition* Property reads the state before and the step. A
selector narrows which steps it is about: every step, the steps of one class (`when`), or every
class of an action (`whenAction`). An unrelated step stays on the path but neither reads nor
exercises the Property.

**Writing it.** In `object properties.safety`, named after the `val`:

Source: [temporal/features/activity_mini/product/Product.scala:39](../temporal/features/activity_mini/product/Product.scala#L39).

```scala
object properties:
  object safety:
    val terminalRemains = property.once(_.phase.in[Closed]).keeps(_.phase)
```

The checker classifies a declaration by its resolved kind, never by its name or predicate body.
A `Property` and a reference that attaches a safety monitor belong in `safety`; a `Progress`
belongs in `liveness`. A claim directly in `properties`, or in the other group, is refused at its
authored source line with the binding, resolved kind and expected group. Plain helper defs are not
claims and may remain directly in `properties`. Omit an empty group, or the whole section when it
has no claims; an omitted group neither invents nor duplicates a shared law or monitor attachment.

The patterns are sugar over the two core forms, `holds(after => ...)` and
`holdsAcross((before, after) => ...)`:

| Pattern | Core form |
| --- | --- |
| `once(over).keeps(_.x)` | `holdsAcross((b, a) => !over(b) \|\| a.state.x == b.x)` |
| `never(to)` | `holds(a => !to(a))` |
| `never(to).from(before)` | `holdsAcross((b, a) => !before(b) \|\| !to(a))` |
| `stays(p)` | `holdsAcross((b, a) => !p(b) \|\| p(a.state))` |
| `stays(p).unless(release)` | `... \|\| release(a)` |

Helpers in a predicate: `after.records(fact)` is `after.facts.contains(fact)`; `a implies b` is
`!a || b` and reads `b` only where `a` holds (so a hole in `b` is not reached otherwise);
`x.in(a, b)` is membership. `keeps` takes a field path, or a def whose body is one.

A Property's Definition ID is `<family>.<name>`, so two machines of one family may each declare
`terminalIsFinal` and share it; a result names the machine as well, which keeps them apart.

**Properties written once for many machines.** A function over `Declares[S]` (the supertype of a
machine and a composition) declares the same Property on whichever model it is called with. Its
state-dependent parts are parameters, each a def of the lifted sources, never a lambda:

Source: [temporal/features/activity_mini/protocol/AnswerDispatch.scala:29](../temporal/features/activity_mini/protocol/AnswerDispatch.scala#L29).

```scala
def oldAnswerKeepsAttempt(m: Machine[AnswerState, Outcome, AnswerFact]): Property[AnswerState] =
  m.property("oldAnswerKeepsAttempt") when answers.deliver(Answer.firstCompleted) holdsAcross (
    (before, after) =>
      before.attempts != 2 || (after.state.phase == before.phase &&
        after.state.attempts == before.attempts && after.outcome == Outcome.rejected(
          Rejection.notFound
        ))
  )
```

Each Property is declared once, in the facet or machine whose promise it states, and every Query
that asks it names that declaration. An author's Property of a machine whose selector and predicate
equal another's of the same machine, up to names and positions, is refused, naming both
declarations; a capability's generated Properties keep their own identities and are not compared.

A case class whose every field is a Property, Scenario or Query is a *bundle*; built in such a
function it lifts as its claims and `bundle.field` reads one back.

## 9. Scenarios and Queries

**Meaning.** A *Scenario* is where to look: a start state and either a pinned schedule of classes or
`free`, any action at every step. A *Query* is a bounded question that joins a Property and a
Scenario: `verify` asks whether the Property holds on every path, `find` asks for one path on which
it holds, and that path is the Query's *witness*. `.live` marks an authored Query for live Case
generation; without it the Query is model-only. Capability expectation bindings make their
generated Queries live.

**Writing it.** In `object queries`, every Scenario before the Query that uses it:

Source: [temporal/features/activity_mini/protocol/Protocol.scala:69](../temporal/features/activity_mini/protocol/Protocol.scala#L69).

```scala
object queries:
  capabilities.bound(six)
  val any = scenario.free
  val terminalThroughProduct = query verify product.ActivityProduct.properties.safety.terminalRemains in any limits six
  val retryCompleted = witness(client.start(maxAttempts := MaxAttempts.two), activityWorker.poll,
    activityWorker.respondFailed(Failure.retryable), timers.backoff, activityWorker.poll,
    activityWorker.respondCompleted).records(Fact.statusCompleted).ends(_.attempts == 2).limits(six)
    .live(inconclusive(Reason.explanationsDisagree,
      because = "The first delivered attempt and final status do not establish the second delivery or retry backoff; explaining executions can disagree on the attempt count."))
  val protocolCompleted = witness(client.start(), activityWorker.poll, activityWorker.respondCompleted)
    .records(Fact.statusCompleted).limits(three).live(satisfied)
```

A Query named by its `val`, by `query("name")`, or, inside a list or a def over a machine, by
default `<machine>.<scenario>.<property>`.

A source witness names a pinned path and its last step's claim in one declaration:
`witness(class, ...).records(fact)`, optionally `.ends(statePredicate)`, `.limits(bounds)`,
`.explore(space)` and `.live(expectation)`. It lifts to an ordinary Scenario, same-step Property
and `find` Query, all named after the Query and owned by the witness's machine or composition.
No new IR declaration kind is added; these use ordinary same-step Property semantics. The
Property selects the last listed class and tests `records(fact) && statePredicate` on its
resulting state, or `records(fact)` alone without `.ends`. A composition lists its `own` or
`synced` classes and records a typed member fact with
`.records(_.member -> MemberMachine, fact)`.

Derived Limits have `steps = actions = path length` and `search = 512`; `.limits` replaces them.
A repeated last class is refused because a same-step Property would test its earlier occurrences
too. Keep a shared Property or Scenario, and `query verify`, as a triple. A `val` names the Query;
`witness("name")(...)` supplies an explicit name when no val does, including list or helper
results. `.live` is the authored Query switch for Case generation; without it the Query is
model-only. Capability expectation bindings make their generated Queries live.

**Limits** bound the search: `steps` the depth, `actions` the schedule length, `search` the product
states the search may visit before it reports `limit-reached`. Every bound is declared once in
`temporal/Bounds.scala` (`two`, `three`, ..., `twelve`, `fourWide`) and imported by name, so a
receipt's bound name means one budget.

**Totals.** Every Query carries a *total*, its static combination count, which the IR generator
computes and the reader checks: for a pinned Scenario the state catalog × `min(steps, scheduled)`,
for a free one states × classes × steps. An author may state it, `limits three total 384`, as a
check of their own reading; it is a review figure, not a prediction, and nothing else reads it.

**Through a refinement.** A Query may verify a Property of the machine its Scenario's machine
refines: `terminalThroughProduct` in the mini Protocol's `queries` above. The Property is read through the refinement map; only a
`verify` may, and only where the refinement holds. A Property of an unrelated machine is refused at
the Query's line.

**Expectations.** A `find` that generates a live Case states what its Run should be judged as:
`.live(satisfied)`, or `.live(inconclusive(Reason.neverEvaluated, because = "..."))`. Each value of
the expectation (contract verdict, disposition, cleanup, conformance, Property outcome, one entry
per monitor) is declared, none inferred; the live test compares a Run with it value by value. A
`find` with `.explore(Exploration(...))` also declares finite alternatives for positions of its
schedule, which the exploration tooling enumerates ([Realizations](#19-realizations)).

Convenience expectations and full `RunExpectation` values whose Property or Contract outcome is
not satisfied require non-blank `because` text, as does each non-satisfied nested
`MonitorExpectation`. A full Run whose Property and Contract are both satisfied cannot supply
that text, even if its conformance is short of conformant; satisfied monitor values also take
none. This text explains the author's expectation only and remains source-only: it adds no IR,
Case or assessment field and cannot supply or change judge explanations. Existing judge reason
IDs and `expected_run` metadata are still emitted and compared. Capability expectation bindings
use the same expectation values and the same source-text rule.

## 10. Monitors and assumptions

**Meaning.** A *monitor* is a finite state of its own that every step advances and a verdict read at
an evaluation point. It watches, never disables: the machine's rows are the same with and without
it. Its state is part of what a check explores, so two paths to one machine state with different
monitor states stay two. It is how a Model states a law about history ("no run records two
outcomes") that no single step can see.

An *assumption* is a name every result that relies on it carries: that a machine stands for an
opaque provider, that it includes a fault only this assumption allows, or the fairness a progress
claim rests on. `fair` makes actions *weakly fair*: a path that keeps one of their classes enabled
forever takes it.

**Writing it.** In `object monitors`:

Source: [temporal/features/activity_mini/protocol/Protocol.scala:62](../temporal/features/activity_mini/protocol/Protocol.scala#L62).

```scala
object monitors:
  val attemptsNeverRewind = monitor[State, Outcome, Fact, Boolean](false)(
    (broken, before, after) => broken || after.state.attempts < before.attempts)(_ == true)
```

`monitor(initial)(next)(violated)` is core; `.readAtEnds` and `.readAfter(p)` move the evaluation
point from "after every step". `sticky(p)` and `stickyAcross(p)` are sugar for the Boolean monitor
that remembers one breach.

**Refused.** Two monitors of one name; a monitor whose next state leaves its catalog (declaration
error); a Query over a composition one of whose members names monitors (unsupported: whether a
member's monitors watch the composition is not defined).

## 11. Progress

**Meaning.** A progress claim is bounded liveness: from every reachable state `from` accepts, a
state `to` accepts follows within `within` steps, on every path the named assumptions admit. It is
violated by a *deadlock* (a state with no row before a `to` state), a *fair non-progress cycle* (a
reachable cycle avoiding `to` on which every fair class enabled throughout is taken), or a *missed
deadline* (`within` steps with no `to` state). Each is reported apart. A search cut by a limit
proves none of them.

Umpire's temporal claims are these, and safety: no unbounded `eventually`. Weak fairness bounds
nothing by itself: a finite bound holds only where the graph leaves no path that spends its steps
elsewhere, which is why the dispatch designs *restrict* the actions that could, rather than assume
them away.

**Writing it.** In `object properties.liveness`, always named, with an explicit finite bound and
the assumptions on which the claim relies:

Source: [temporal/features/activity_mini/protocol/DispatchingProgress.scala:12](../temporal/features/activity_mini/protocol/DispatchingProgress.scala#L12).

```scala
val workerFair = assume.fair(activityWorker.poll)

object DispatchingProgress extends Derived(ActivityProtocol.restrict(activityWorker.poll)):
  object states:
    def waiting(s: State): Boolean = s.phase == Phase.scheduled
    def started(s: State): Boolean = s.phase == Phase.started
  object properties:
    object liveness:
      val workerStarts = DispatchingProgress.leadsTo("workerStarts")(
        states.waiting, states.started, within = Bounds.dispatchSteps, workerFair)
```

A safety postcondition such as `Queue.properties.safety.deliveredWasScheduled` assumes no future action: it
must hold whenever its selected response step occurs, but does not say that response will occur.
The progress claim is universal over every admitted path from `from`; it states its `within` bound
and fairness or recovery assumptions separately. A `find` Query supplies one witness path and is
therefore not evidence that this universal progress claim holds.

A composition's claim reads the composed state and counts composed steps, a sync's once. Its
assumptions select composed steps by structure: `assume.fair(c.synced(_.activity -> worker.poll),
c.own(_.queue, queue.persist))`. The Scala authoring, lifting and model-check path supports this
composition form today. Progress is a model check only: no Case is generated from it, and that
source support does not by itself close fn-150's pending generated-consumer, gate or live evidence.

## 12. Holes

**Meaning.** A declared hole names behavior the Model leaves unknown on purpose. A row that reaches
one is neither disabled nor an error: a check that explores it reports its result *incomplete*,
naming the hole and the row, and keeps everything else it established. A violation found elsewhere
stays a violation. A `match` with no case for a value is an *undeclared* hole: the same, with no
declaration to account for it.

A hole inside a claim's predicate is unknown evidence at that step: the claim neither holds nor
fails there, and the search does not continue through it. A hole reached outside any step (a start,
an `end`, a refinement map) leaves that machine without a table, and everything that depends on it
is incomplete; every other machine is checked as if the hole were not there.

**Writing it.**

Source: [temporal/features/activity_mini/protocol/AnswerDispatch.scala:23](../temporal/features/activity_mini/protocol/AnswerDispatch.scala#L23).

```scala
val cancelDuringBackoff = hole
```

Source: [temporal/features/activity_mini/protocol/Answering.scala:53](../temporal/features/activity_mini/protocol/Answering.scala#L53).

```scala
def interrupt(@unused s: AnswerState): List[Step[AnswerState, Outcome, AnswerFact]] =
  AnswerDispatch.monitors.cancelDuringBackoff.reached
```

A stuck-state finding is not reported for a state with a hole: the hole already says unmodeled
behavior may happen there.

## 13. Refinement

**Meaning.** A feature is read at two levels: the *Product*, what a caller sees, and the *Protocol*,
how the server gets there, which *refines* the Product. A refinement is a map from the refining
machine's states to the refined machine's, and the check is: every start maps to a start, and
every row result is carried by a product row from the mapped source to the mapped target with the
same outcome and the facts the product sees, or else source and target map to one product state
and the step is a *stutter*. Where both phases carry roles, closedness is preserved as well: a
state is `Closed` exactly when its image is (checked in the model tests, since the IR has no roles).

A refinement narrows permission (the Protocol does nothing the Product forbids) and does not by
itself carry obligation: a Product Property holds of the Protocol only where a `verify` Query reads
it `through` the refinement, and a progress claim only where the Protocol declares its own.

**Writing it.** `object refinement`, the second section:

Source: [temporal/features/activity_mini/protocol/Protocol.scala:50](../temporal/features/activity_mini/protocol/Protocol.scala#L50).

```scala
object refinement extends Refinement(product.ActivityProduct):
  def toProduct(s: State): product.State = product.State(s.phase match
    case Phase.scheduled | Phase.backingOff => product.Phase.scheduled
    case Phase.started => product.Phase.started
    case Phase.cancelRequested => product.Phase.cancelRequested
    case Phase.completed => product.Phase.completed
    case Phase.failed => product.Phase.failed
    case Phase.canceled => product.Phase.canceled
    case Phase.timedOut => product.Phase.timedOut)
  def visibleOutcomes(o: Outcome): Boolean = o != Outcome.accepted
  val unobservable = List(timers.backoff)
```

`visible` narrows both halves of the rule: a carrying product result records, by name, every fact
the result records that the product sees, and a stutter records none of them. A refinement that
names none reads every fact. `visibleOutcomes` narrows the stutter: its outcome is none of them.

**Refused.** A `visible` on a machine that refines none; a machine refining two. A rejected
refinement is a result of its own, `refinement-rejected`, with the rule broken and a witness, and
every Query read through it answers with that rejection.

**Design note.** The Product is written first and never imports the Protocol; the Protocol is
written against the same actions, so a Property proved on the Product is carried to the Protocol
by the refinement rather than restated.

## 14. Derivations

**Meaning.** A derived machine is another machine with one thing changed, keeping everything else
(start, end, evidence, monitors, assumptions, refinement, phase projection). It is how a Model
holds a design beside its variants: a negative control that must fail, a failure model that must
still hold, a restriction that isolates one path.

**Writing it.** An object of its own, in its package's family; the derivation is one expression:

Source: [temporal/features/activity_mini/protocol/AnswerTrusting.scala:13](../temporal/features/activity_mini/protocol/AnswerTrusting.scala#L13).

```scala
object AnswerTrusting extends Derived(AnswerDispatch.rebind(answers.deliver ~> AnswerDispatch.answering.effects.trust)), NegativeControl:
```

Source: [temporal/features/activity_mini/protocol/ProtocolMember.scala:5](../temporal/features/activity_mini/protocol/ProtocolMember.scala#L5).

```scala
object ProtocolMember extends Derived(ActivityProtocol.unmonitored)
```

| Derivation | Keeps | Changes |
| --- | --- | --- |
| `restrict(actions*)` | state, start, end, monitors, assumptions | keeps only the named actions' rows; drops the refinement |
| `rebind(a ~> e)` | the action's rules, their guards and classes | their effect; refused where the source's rules of one action have different effects (unless each fires one class) |
| `rebind(on(actor.a) { where(g) ~> e })` | the rest | replaces the action's rules; cases are `where`/`always` only |
| `extend(on(actor.b) { ... })` | everything | adds rules for an action the source does not bind |
| `refining(M)(map)` | everything | replaces the refinement |
| `assuming(a*)` | everything | appends assumptions, each once |
| `unmonitored` | the transitions | drops monitors and refinement, for a composition member |

A derived object adds only its own `states`, `properties`, `capabilities` and `queries`, and mixes
in no `Phased` of its own: it reads its source's.

**Markers.** `NegativeControl` marks a deliberately wrong design the checks must refute; the marker
check refuses one no lifted Query can refute (a `verify`, or a Run expected violated), one that
something refines, or one that declares a refinement. `FailureModel` marks the real design under a
fault the environment can cause; it must bind a fault (an action of the actor `fault`) and not every
Query of it may expect violation. A machine that binds a fault and is marked neither is refused.
Markers change no name, ID, table or answer.
The checks in `model/framework/lint` read the shared typed-tree index and the lifted Models after
lifting and before any file is written, through the gate and the direct lift entry point alike.

## 15. Facets

**Meaning.** A facet is one behaviour area of one machine at one level, named as a noun
(`Dispatching`, `Completion`). A machine is assembled from its facets, in facet order, and lifts as if
every facet's sections were written in it: same rows, same outcomes and facts; only function names,
positions and binding order differ. A facet *owns* the actions it binds, all their rules and the
effects they use; two facets binding one action are refused.

**Writing it.** A trait in its own file, holding one object named after it:

Source: [temporal/features/activity_mini/product/Completion.scala:12](../temporal/features/activity_mini/product/Completion.scala#L12).

```scala
trait Completion extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Dispatching =>
  object completion:
```

The machine mixes them in and lists their rules and capabilities:

Source: [temporal/features/activity_mini/product/Product.scala:37](../temporal/features/activity_mini/product/Product.scala#L37).

```scala
object rules extends Facets(dispatching.rules, completion.rules)

object properties:
  object safety:
    val terminalRemains = property.once(_.phase.in[Closed]).keeps(_.phase)

object capabilities extends FacetCapabilities(dispatching.capabilities, completion.capabilities)
```

Two lifter switches hold the Temporal Models to this: `facets-only` (a declared `Machine` keeps
rules, effects and capabilities only in facets) and `facet-vocabulary` (each feature file names
its facets once, `enum FacetName`, and an action bound at both levels is owned by the same-named
facet at both). What stays machine-wide: phases, `State`, `init`, `end`, the projection, the
refinement, monitors and whole-feature claims.

Core form: the machine's sections are the union of its listed facets' sections in listed order,
including capability waivers and overrides. Qualified effect paths identify effects: two distinct
namespaces may both declare `effects.settle`; exposing one namespace twice conflicts in Scala.
Native aggregation checks the rule declaration's enclosing facet rather than guessing from a JVM
class name. It refuses ordinary non-facet rule books and duplicate ordinary, timer, fault or
disabled owners with the actual action and facet names; the lifter checks ownership independently.

Each recognized level file has one facet named after its file, and the facet namespace's sections
obey the usual declaration places. Actions remain in the signature. Machine `FacetCapabilities`
lists facet sections and has no capability or waiver declarations of its own. A declared machine's
core `Bindings` is refused by `facets-only`; derived machines and compositions retain their
exemption. Paired primary levels are checked through their explicit refinement even with state
types named `QueueView`/`QueueDetail`. A separate design subject with its own state keeps its
own facets; the spelling of a state type alone cannot decide pairing.

## 16. Capabilities

**Meaning.** A capability binds a reusable protocol to one machine's vocabulary: which outcome
rejects a closed request, which class pauses, which timer expires. Its companion defines the
Properties that use those bindings. Declaring the capability *brings* those Properties, each
expanded into a Property, a Scenario and a Query named `<machine>.<property>` (or
`<machine>.<val>.<property>` for a capability declared several times), with the companion def as its
`origin`. A Property that reads two kinds is brought only where both are declared. Nobody calls it.

A facet *declares* capabilities; a capability is named as an adjective (`Pausable`) and never names
a facet. Temporal's split by level: `capabilities/product` (`Closable`, `Pausable`, `Pollable`,
`Terminable`, `Describable`) and `capabilities/system` (`Retryable`, `Expirable`, and the named
laws `Exclusive`, `Fenced`, `Idempotent`).

**Writing it.** A case class extending `CapabilityOf[S, O, F]` with a companion `CapabilityKind`
whose defs take the model, then fields by name:

Source: [temporal/capabilities/product/Closable.scala:11](../temporal/capabilities/product/Closable.scala#L11).

```scala
final case class Closable[S, P, O](rejected: O)(using
    @unused phasing: Phasing[S, P],
    @unused finite: Finite[P],
    @unused witness: TypeTest[P, Closed],
    @unused phaseType: ClassTag[P]
) extends CapabilityOf[S, O, Nothing]
```

Declaring, waiving and bounding:

Source: [temporal/features/activity_mini/product/Completion.scala:48](../temporal/features/activity_mini/product/Completion.scala#L48).

```scala
object capabilities extends Capabilities:
  val closable: Capability = Closable(rejected = Outcome.rejected(Rejection.notFound))
  except(Closable.closedIsRejectedUniformly[State, Phase, Outcome],
    because = "The Product omits late worker RPC replies; only client cancellation models NotFound after closure.")
```

Source: [temporal/features/activity_mini/protocol/Completion.scala:51](../temporal/features/activity_mini/protocol/Completion.scala#L51).

```scala
object capabilities extends Capabilities:
  val retryable: Capability = Retryable[State, Phase, 2](
    failure = activityWorker.respondFailed(Failure.retryable), retryable = true,
    attemptCount = ActivityProtocol.states.attemptCount,
    maximumAttempts = ActivityProtocol.states.maximumAttempts,
    retriesRemaining = ActivityProtocol.states.retriesRemaining,
    pendingCancel = Some(ActivityProtocol.states.pendingCancel))
  overriding(Retryable.failureReturnsToWaiting[State, Phase] -> properties.retryBacksOff,
    because = "The mini protocol distinguishes retry backoff from dispatch-ready waiting while reporting scheduled for both.")
  val scheduleDeadline: Capability = Expirable[State, Phase, Waiting, Fact](
    timer = deadline.scheduleToStart, armed = ActivityProtocol.states.scheduleArmed,
    timeout = Fact.statusTimedOut, timeoutFacts = Seq(Fact.statusTimedOut))
```

`except` lifts nothing for that Property; `overriding` lifts the machine's own def under the shared
name; both need a reason and are recorded as waivers (`<file>.waivers.json`, forwarded into
`.lint.json`, not findings). A Query of the machine's own reads a brought Property with
`capabilities.claim(Kind.property[...])`. Every field that takes a function names a def of the
lifted sources, or a member's def through `through` on a composition; a lambda is refused at its
line. A Query brought by a same-step Property is a `find` through the capability's `reach` path and
expects the Run its field names; every other brought Property is a free `verify`.

## 17. Compositions

**Meaning.** A composition is one Model from machines of *different entities*, for a claim no one
of them can state: an activity over its task queue, an operation with its worker. Its state is a
record with one field per member. A member's own action steps that member alone and is keyed
`<field>_<class>`; a *sync* pairs two members' actions into one step keyed by the sync's name
followed by both classes' inputs, whose results are the product of the members' results with the
first member's outcome and both members' facts. Only the rows reachable from the
members' starts belong to it. Its outcome type is `String`, the composed outcome key. Its fact type
is an enum declared by the composition, with one case per member carrying that member's fact.
Native claims read those tagged values directly; lifting and Go lowering still use the legacy
`<field>_<fact>` keys until the composition migration's second phase.

A member that *replaces* a machine stands in for it within the composition: a detailed provider in
place of an opaque one. Its machine must declare a refinement of the replaced machine that holds,
and the composition relies on none of the replaced machine's assumptions.

**Writing it.** An object with field selectors; `->` pairs a member with its value and never means
a transition:

The source excerpt retains the held record-state spelling. A native composition declares its own
`Fact` enum with per-member cases, such as `case activity(value: AdmissionFact)` and
`case queue(value: QueueFact)`.

Source: [temporal/features/activity_mini/protocol/Queue.scala:13](../temporal/features/activity_mini/protocol/Queue.scala#L13).

```scala
final case class OverQueue(activity: State, queue: QueueView)

object taskQueue:
  val schedule = internal.input(scheduleToStart).on(foundations.taskqueue.taskQueueEntity)

object Queue extends Composition[OverQueue](
    _.activity -> ProtocolMember, _.queue -> QueueScheduled):
  def end(s: State) = s.activity.phase.in[Closed]
  object syncs extends Syncs:
    sync(_.activity -> activityWorker.poll, _.queue -> foundationQueue.deliver)
  object properties:
    object safety:
      val deliveredWasScheduled = property.whenAction(synced(_.activity -> activityWorker.poll))
        .holds(_.state.queue.outstanding.in(Outstanding.deliveredOnce, Outstanding.deliveredTwice))
  object queries:
    val delivery = scenario.actions(own(_.activity, client.start()), own(_.queue, taskQueue.schedule()),
      synced(_.activity -> activityWorker.poll))
    val queueDelivery = query verify properties.safety.deliveredWasScheduled in delivery limits three
```

`synced` finds the one sync that pairs that member's action; `own` an action the member binds and
no sync pairs. `withMember` replaces one member, keeping syncs, end and order; `DerivedComposition`
is the form that also keeps the source's fact type. A claim that reads a member's named state
set calls its helper directly, `ActivityRecord.states.twoActive(activity)`. A member fact is wrapped
in its case, `records(Fact.activity(AdmissionFact.admissionCommitFailed))`; witnesses use the same
value in `.records`, `.excludes` and `.recordsWhen`. An unwrapped member fact is a type error.
A native composed step carries those same tagged values in its facts, for example
`Step("accepted", joined, List(Fact.activity(AdmissionFact.admissionCommitFailed)))`.
The framework's own composition starts from each member's first start; the reader composes every start.

**Refused.** A selector naming no field; a member of another state type; a sync of an action the
member does not bind; a `synced` matching none or several; an `own` of a paired action; a
replacement by a machine that refines nothing; a Scenario or `whenAction` over a key that is no
composed class.

## 18. Channels

**Meaning.** A channel is the messages one machine's state holds between their send and their
delivery: a bounded, ordered-or-not, reliable-or-lossy buffer that may deliver a message once more
when its acknowledgment was lost. The reader *derives* the delivery and loss rows from the
declaration: the `deliver` action's row at state `s` and message `m` removes the first entry
carrying `m` (first entry at all for FIFO), applies the bound function to the result, and, where
the entry's redeliveries are fewer than `duplicates`, adds the same results again with the entry
put back, explained as "the channel delivers the message again". The `lose` action is bound exactly
when the channel is lossy. A send to a full channel lands outside the catalog, so an effect checks
`isFull` first.

**Writing it.**

Source: [temporal/features/activity_mini/protocol/AnswerDispatch.scala:22](../temporal/features/activity_mini/protocol/AnswerDispatch.scala#L22).

```scala
val answers = channel[Answer](capacity = 1, order = Order.fifo, loss = Loss.lossy)
```

Source: [temporal/features/activity_mini/protocol/AnswerFaults.scala:18](../temporal/features/activity_mini/protocol/AnswerFaults.scala#L18).

```scala
given Finite[AnswerState] =
  given Finite[Inbox[Answer]] = AnswerDispatch.monitors.answers.contents
  new Finite[AnswerState]:
    def values: IndexedSeq[AnswerState] = Finite.derived[AnswerState].values
```

A state holds each channel in at most one field; the message type is a named finite type, Boolean
or `UpTo[N]` (including named opaque integer types with their own range). A plain `Int` message
is refused at the channel declaration even with `Finite.upTo(n)`; send bounded values with `UpTo(n)`.
Each `Inbox[UpTo[N]]` associates with the channel of that bound. The mini uses this channel in its
`AnswerDispatch` FailureModel. `Answering.effects.send`
enqueues a first- or second-attempt answer; `receive` rejects a stale answer, while
`AnswerTrusting` rebinds delivery to `trust` as the NegativeControl. The task queue foundation
separately models its one outstanding message as an enum.

## 19. Realizations

**Meaning.** A realization says how the `find` Queries of one machine run against a real system:
which command performs each action class, which recorded data is evidence of each fact, and what
the controller does around them. The lowering (`tools/umpire/lower`) takes a `find` Query's
witness through it into a Testpilot Case: a Program of instructions and a Contract the recorded
evidence must satisfy. A realization carries no Case and decides nothing a path decides. A `verify`
realizes nothing. A `find` runs through the realization that names it in its header's `runs`
(`Realizes(machine, runs = Seq(...))`: Queries, lists of them, or a `queries` or `capabilities`
section of its machine), and a `find` no realization names is model-only. So one machine may have
several realizations, each running the Queries it names, and a Query needs no copy of its machine
to choose one. A realization naming a Query of another machine, or a Query two realizations name,
is refused, naming both. A Case is assessed against its Query's Property and the monitors of its
Scenario's machine, whichever realization runs it.

The pieces, each a declaration the IR carries as written:

- **roles**: the participants commands address, with the environment bindings a run supplies
  (the kit's `workflowService`, `taskQueue`, `caseWorker`, `nexusEndpoint`);
- **scripts**: one per activation, the controller or a worker (`Workflow`, `NexusHandler`,
  `Activity`). Each is an ordered list of items: `everyCase(command)`, a command every Case
  carries; `onPath(classes*)(command)`, one only the Cases whose path takes one of those classes
  carry; `perform(class -> command, ...)`, where the path's steps of those classes land, in path
  order. A command is named after its `val` in kebab case; `rpc(role, METHOD) { field(_.x) := v }`
  opens a typed request scope; `await(evidence, calls)(condition) {}` is the one wait form, whose
  interval and deadline the lowering derives; `call.withFields { }` keeps a call's name,
  `call.extended { }` is a command of its own;
- **evidence**: kinds of recorded data that confirm facts, each keyed to its operation and saying
  what it commits to (what a caller was told, or a durable commit). A kind may be *exhaustive*: its
  source reports every occurrence, so absence means the step did not happen, closed by one read
  every Case carries (`closes = ...`). The kit's modules (`HistoryEvidence`, `DescribedStatus`,
  `answered`, `delivered`) declare the common shapes once;
- **learned values** and **observations**: a handle one command binds and others read; a protobuf
  message a run records;
- **controls**: actuators beyond commands, such as holding what a step dispatched;
- **server steps** and **API behavior**: the classes no command performs (a delivery, a timeout)
  with the kind of cause each is, and the kit's hints of when a write is visible to a read, from
  which every read's wait is derived;
- a **status table**, `statusTable(fact -> value, ...)`, read when the IR generator lifts.

**Writing it.** An object typed by its machine, with sections in this order:

Source: [temporal/features/activity_mini/realization/MiniActivityRun.scala:12](../temporal/features/activity_mini/realization/MiniActivityRun.scala#L12).

```scala
object MiniActivityRun extends Realizes(
  ActivityProtocol,
  runs = Seq(ActivityProtocol.queries.protocolCompleted, ActivityProtocol.queries.retryCompleted)
):
  object controller extends Controller(
    deadlines[StartActivityExecutionRequest](
      client.start(maxAttempts := MaxAttempts.one), startOne, duration(deadlineSeconds),
      unset = Some(startToClose -> startOneUnreached)
    )(
      scheduleToStart.sets(_.getScheduleToStartTimeout),
      startToClose.sets(_.getStartToCloseTimeout)
    ),
    deadlines[StartActivityExecutionRequest](
      client.start(maxAttempts := MaxAttempts.two), startTwo, duration(deadlineSeconds),
      unset = Some(startToClose -> startTwoUnreached)
    )(
      scheduleToStart.sets(_.getScheduleToStartTimeout),
      startToClose.sets(_.getStartToCloseTimeout)
    ),
    onPath(deadline.scheduleToStart)(holdDispatchBeforeTimeout),
    perform(client.cancel -> requestCancelActivity),
    onPath(activityWorker.respondCompleted)(described.await(protocol.Fact.statusCompleted)),
    onPath(activityWorker.respondFailed(Failure.fatal))(described.await(protocol.Fact.statusFailed)),
    onPath(client.cancel)(described.await(protocol.Fact.statusCanceled)),
    onPath(deadline.scheduleToStart, deadline.startToClose)(described.await(protocol.Fact.statusTimedOut))
  )

  object workers extends Workers(attempts)

  object evidence extends Evidences(
    answered(protocol.Fact.statusScheduled, startActivity),
    delivered(protocol.Fact.statusStarted, attempts, attempt = 1, after = startActivity,
      Taking(activityWorker.poll, 1)),
    answered(protocol.Fact.statusCancelRequested, requestCancelActivity),
    described(protocol.Fact.statusCompleted),
    described(protocol.Fact.statusFailed),
    described(protocol.Fact.statusCanceled),
    described(protocol.Fact.statusTimedOut)
  )

  object controls extends Controls(unstartedDispatch)
```

A derived realization uses `DerivesFrom` and holds `changes` with
`inserting(after = step)(items*)` and `replacing(step)(items*)`; the mini has no derived realization. Protobuf data is named through
generated types and typed selectors (`field(_.getTaskQueue.name)`); Scala never builds or sends a
request, and Go checks every path against the descriptors when it lowers.

Typed realization records retain their types, fields and core data representation. Their
constructors, generated `apply` and `copy` are private to their realization package; an author
uses `Instruction.rpc` / scoped `rpc`, `Instruction.readUntil` / scoped `readUntil`, the `Recorded`
factories, `Assignment.typed`, `ResponseRead.typed`, `EvidenceField.typed` and
`WorkflowHistory.event`. These factories keep their request, response and projected-root checks.

**How a Query lowers**, in short: the witness path fixes which items the Case carries and where
performances land; each step is confirmed by one kind of evidence (a step that records nothing is
a Known Gap); the Property is lowered to Contract clauses over the machine's table; a path that
needs something the runtime has no primitive for is listed `unsupported` with its reason at the
declaration. The README's "Following the example to a Verdict" records an existing Nexus Run;
it is not live evidence for the mini. The mini selects `protocolCompleted` and `retryCompleted`
only, with seven evidence kinds and deadline presets retaining the one/two-attempt policy.
`retryCompleted` expects an inconclusive Property with `Reason.explanationsDisagree` because
first-delivery and final-status evidence does not establish the second delivery or retry backoff.
That source explanation is not assessment output, and no mini live Verdict is claimed.

## 20. What the tools answer

### Levels

Every expression sits at one of five levels, by what it is computed from, and the IR generator
keeps them apart:

| Level | Places |
| --- | --- |
| value | starts, a Scenario's start, evidence, `visible`, a monitor's initial state and `violated` |
| reading of a state | `end`, a refinement map, a progress claim's `from` and `to` |
| step | the function a rule names and what it calls |
| reading of a step | a Property's `holds`, a monitor's `next` and `after` |
| claim over a path | a Query, a monitor's verdict, a progress claim |

Only a step makes a step record. A `Step(...)`, `enter`, `stay` or a call of a step function in
any other place is refused at its line.

### Expressions the reader evaluates

#### Liftable subset

The typed author lint and the lifter use `framework.inspect.LiftableSubset` for the library
allowlist. A step, guard or predicate is a pure expression in the rows below. Other Scala tree
forms and library operations are refused at the author's line, naming the construct and this
table. Declaration computation is a different context: a library operation in a declaration
does not thereby become legal in its captured function. The table records the currently supported
source export, including the temporary declaration readers that constructed-value export retires.
Adding an operation requires a row and its fixture; the fixture names below are functions in
`framework/export/testdata/lifts/Subset.scala` unless another path is stated.

| Source tree or operation | Context and IR lowering | Fixture |
| --- | --- | --- |
| Boolean literal | Body: Boolean `Literal` | `booleanLiteral` |
| Integer literal | Body: integer `Literal` | `integerLiteral` |
| String literal | Body: text `Literal` | `textLiteral` |
| Parameter or local reference | Body: `Var`, or its bound argument | `reference` |
| Field selection | Body: `Field` | `field` |
| Source function application | Body: `Call` of the indexed function | `called` |
| Case-class or enum constructor | Body: `Construct`, fields in declaration order | `constructed` |
| Typed/type-applied/named/inlined wrappers | Body: underlying expression; type and argument binding retained | `wrapped` |
| Local immutable `val` block | Body: ordered `Let` | `localValue` |
| Structural lambda | Callback: `Function`; expression: `Lambda` | `predicate` |
| Empty draft block | `effect` capture only: unchanged state with the declared ok outcome; Unit is not a body value | `EmptyDraft.effects.keep` |
| `if` | Body: `If`, only the selected branch is read | `conditional` |
| `match` and literal, case, bind, wildcard, alternative patterns | Body: `Match` and matching IR patterns; enum wildcard restrictions still apply | `matched` |
| `List(...)` / `Nil` / repeated arguments | Body: `List` in written order | `listed` |
| `Some(...)` / `None` | Body: synthetic finite Option enum | `optional` |
| `Step(...)` | Step body only: core step `Construct` | `stepRecord` |
| `==` | Body: structural `OP_EQ` | `equal` |
| `!=` | Body: structural `OP_NE` | `unequal` |
| `&&` | Body: short-circuit `OP_AND` | `conjoined` |
| `||` | Body: short-circuit `OP_OR` | `disjoined` |
| `!` | Body: `OP_NOT` | `negated` |
| `<` | Body: integer `OP_LT` | `less` |
| `<=` | Body: integer `OP_LE` | `lessEqual` |
| `>` | Body: integer `OP_GT` | `greater` |
| `>=` | Body: integer `OP_GE` | `greaterEqual` |
| `+` | Body: integer `OP_ADD` | `added` |
| `-` | Body: integer `OP_SUB` | `subtracted` |
| `contains` | Body, Scala sequence receiver: list membership `OP_CONTAINS`; String and Java collection receivers are refused | `contained` |
| `copy` | Body, case-class receiver: `Copy`; default getters preserve unchanged fields | `copied` |
| `min` | Body, integer receiver: `If` with `OP_LE` | `minimum` |
| `max` | Body, integer receiver: `If` with `OP_GE` | `maximum` |
| `++` | Body, Scala sequence receiver: `OP_CONCAT`; unnamed step branching remains refused | `concatenated` |
| `+:` | Body, Scala sequence receiver: prepend a literal item or `OP_CONCAT` | `prepended` |
| `toList` | Body, Scala sequence receiver: identity of the already lifted list | `converted` |
| `forall` | Capability's bound fact-family parameter only: ordered conjunction of the recorded-fact guards; no dynamic collection lowering | `lifts/DeadlineCapabilities.scala`; `lifts/DeadlineCapabilityRejects.scala` |
| `map` | Declaration sequence reader: substitute each written item into a structural mapper; no body lowering | `LiftableSubsetFixtures` declaration control; body refusal control |
| `toVector` | Declaration sequence reader: underlying written items; no body lowering | `LiftableSubsetFixtures` declaration control |
| `find` | Query declaration: `FORM_FIND`; collection `find` has no body lowering | `lifts/Declarations.scala` find Query; `LiftableSubsetFixtures` body refusal control |
| `require` | Function contract: precondition | `contracted` |
| `ensuring` | Function contract: body; postcondition is not an IR assertion | `contracted` |
| Builder `name`/`when`, optional/sequence `isEmpty` | Declaration precondition only: read already-bound core metadata, then check the definition's `require`; no Option body lowering | `DefinitionExpansionTest` restricted helper/core twin; existing `whenStays` refusal |
| String interpolation | Constant values only: text `Literal` | `interpolated` |
| Channel `empty` | Channel literal | `lifts/Channels.scala` |
| Channel `send` | `Inbox` send | `lifts/Channels.scala` |
| Channel `isEmpty` | `Inbox` empty test | `lifts/Channels.scala` |
| Channel `isFull` | `Inbox` full test | `lifts/Channels.scala` |
| Hole `reached` | Body: declared `Hole` read | `lifts/Hints.scala` |
| Role test through a marked definition | Body: membership in the enum's role cases | `lifts/RolesCompatibility.scala` |
| `choose` | Core choice naming | `lifts/Choices.scala` |
| `because` | Core explanation | `lifts/Sugar.scala` |
| `UpTo` | Bounded integer | `lifts/Inputs.scala` |
| `through` | Projected source `Call` | `lifts/Members.scala` |

`var`, assignment, loops, `return`, exceptions and comprehensions inside these bodies are outside
the subset. A declaration may compute its declarations; the functions it hands them remain held
to this table. The author-time refusal controls are `framework.lint.LiftableSubsetFixtures`.

The IR holds functions as expression trees: literals, variables, field reads, calls (after the
callee's precondition), constructors, `copy`, `not`/`and`/`or` (short-circuit), `=`/`≠`
(structural), integer comparison and arithmetic, list concatenation and membership, `if`, `let`,
list literals, lambdas, `match` with wildcard, binding, literal, case and alternative patterns, a
declared hole, and the channel operations. A hole propagates; a branch not taken is not reached.

### Results

A check gives one of these, kept apart and listed in precedence order where one result is made of
several:

| Result | Means |
| --- | --- |
| `admission-error` | the Model is rejected before any check (unknown version, undeclared name, wrong arity, no finite catalog, a channel or Limits out of range, a wrong total, two declarations under one key, recursion, a realization that is not well-formed, ...) |
| `declaration-error` | a declaration cannot be read as the IR says: a Boolean function returning otherwise, a monitor state outside its catalog, a `find` of a transition Property |
| `replay-failed` | a witness did not replay through a second interpretation |
| `refinement-rejected` | with the rule broken and a witness |
| `counterexample` / `found` | a witness, which holes elsewhere do not take away |
| `resource-limit`, `limit-reached`, `unresolved` | a ceiling or bound stopped the work; proves nothing |
| `incomplete` | no witness, and a hole was read; named with the row |
| `unsupported` | listed, not checked (a `find` with `through`; a composition with a monitored member) |
| `verified-within-limits` / `not-found` | holds on everything the scope reached, no hole read |

A result names the table it read (Definition ID and Behavior Fingerprint), the limits, the
assumptions it relies on and the work it took.

### Who refuses what

| Where | What |
| --- | --- |
| the compiler | a step bound to an action of other inputs, a value of the wrong type, two levels' types compared, a `choose` of one, a `Recorded` case without status, a capability over another machine's types |
| the framework, as objects construct | overlapping rules, a bare rule target, two facets binding one action, an input token not the action's, a phase writer that does not write the projection, a role with no case |
| the source checks, at the Scala line | declaration order, section and file structure, typed conventions and marker requirements, including negative controls nothing refutes; a refusal writes no IR |
| the IR generator, at the Scala line | unsupported constructs, wrong levels, unnamed branching, a lambda where a def is required, names that would collide, totals that are no literal, every realization form it can see |
| the reader, at admission | the IR-level rules above: catalogs, channels, Limits, totals, keys, recursion, realization consistency |
| the reader, as it checks | holes, refinements, counterexamples, limits |

## 21. Core and sugar

The DSL is kept in two layers so it can be changed with confidence. **Core** is what the IR needs
declared: `Machine`, `Derived`, `Composition`, `Phased`, the derivations, `Actor`, `action` and its
classes, `input`, `UpTo`, `Bindings` and `~>`, `Step` and `because`, `Declares[S]`, `property` with
`holds`/`holdsAcross`/`when`, `monitor`, `leadsTo`, `Syncs` with `sync`/`synced`/`own`/`withMember`,
`through`, `scenario`, `query`, `Limits`, `total`, `Capabilities`, `choose`, `channel`, `hole`,
`assume`, and the realization declarations with their script helpers. **Sugar** is any form whose
meaning a core form already says: `Rules` with `on`/`from`/`when`/`where`/`always`/`disabled`,
`enter`/`stay`/`reject`/`rejects`/`enters`/`disabled`, the `is { }` and `effect { }` blocks,
`implies`, `in`, `records`, the claim patterns, `witness` (the same-named Property, Scenario and
`find` Query triple), `sticky`/`stickyAcross`, and both spellings of `:=`.
Sugar lives only in the framework and kit `Syntax.scala` files; each form documents its
`Core form:`, and a fixture lifts it beside that spelling and requires the same IR. No core file
imports sugar; `make lint-model` checks it.

Function-body `enter`, `stay`, `reject`, `disabled`, both membership forms, `implies` and step
`records`, and the claim patterns and sticky monitors, are marked `@sugar`. The lifter expands their indexed definitions with the supplied
types, values and functions bound; adding a marked kit definition requires no exporter registration.
Generated nodes take the call's position, and explicit arguments keep their source positions.
An unsupported definition names its definition position; recursive expansion names the cycle.
Observable by-value arguments retain eager, once-only evaluation in caller order through core
`let` bindings where substitution would drop, delay, duplicate or reorder a read. By-name arguments
remain under the definition's short-circuit or conditional control.

Composition `records(_.member, fact)` has a thin inline selector capture point. It records the
typed field path, without executing or lifting the selector. The core capture validates the selected
member and its fact type; a plain marked definition builds the canonical exact member key and tests
the step's facts. Equal states and identically named facts of another member cannot satisfy it.
Capture requires a direct field-selector expression at the call site. Plain selector parameters
and selector `val` references are opaque and refused, rather than assigning invented metadata.
The declaration expansion follows marked constructors and their core parent declarations, binding
their type parameters and fields. Structural callback bindings preserve the originating method's
function name; a selector-shaped generic callback retains its field path. Core `holds` and `monitor`
read those bindings, without a registry of pattern or monitor names. Caller positions and the direct
builder-chain refusals are retained. The syntax lint checks lifter references in code and dispatch
strings, including its remaining `Syntax.scala`, against the temporary owners below.
Definition preconditions are read from their `require` statements and core builder metadata;
the framework patterns declare their no-`when` policy there. An unrelated marked helper over
restricted `holds` has the same meaning as its core spelling.

| Remaining source reduction | Removal owner |
| --- | --- |
| Named action inputs (`:=`, action application) | fn-141.9 |
| Request field assignment/scopes, protobuf literal scopes, script helpers | fn-141.8 |
| RuleBook headings and disabled actions, rule effects (`enters`/`rejects`), phase writers and `is`/`effect` blocks, captured `Ok` outcomes | fn-141.10 |
| Composition selectors, syncs, own/synced and replacements | fn-141.11 |
| Witness path, primary records/excludes, recordsWhen clauses and modifiers; claim/query builders | fn-141.12 |
| Capability bound fact-family and Property expansion | fn-141.12/.13 |

These owners are the bounded exceptions in `SyntaxRule.remainingExportSugar`. They are not
permanent name-dispatch exemptions. Claim patterns and sticky monitors have no such exception.

`Bindings` has no current mini/shared/kit/sample author use. Its core fixtures are reserved for
fn-141.3/.4's sugar equivalence proofs, with retirement still open at constructed `RuleBook` export
in fn-141.10/.14. The fn-141.2 checkpoint does not close that obligation.

Three symbols, one meaning each: `~>` binds an action to its effect, `->` pairs a key with its
value, `:=` gives a named slot a value. Everything else the DSL adds is a word.

**Naming.** A declaration is named after its `val` or object and takes its Definition ID from where
it sits: package, enclosing objects, own name. A machine's *family* is its package; its claims and
Queries hang off it (`<family>.query.<name>`). Moving a declaration between files of one package
changes nothing; moving it into another object or package changes its IDs, which is accepted and
can be sealed with the equivalence check (README, "Sealing a batch").

**Where to go next.** README "Writing a Model" for the file layout, the reading order and the
lints; `model/framework/export/testdata/layout/lamp/` for the template to copy; `model/temporal/features` for
the Models this guide abstracts from.

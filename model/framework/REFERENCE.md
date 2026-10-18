# Umpire DSL reference

A lookup of the DSL's API: each form, where it may be written, what it means, and what refuses it.
[SEMANTICS.md](SEMANTICS.md) walks through the same language as a guide; this file lists it.

## Scope and status

- **Current form.** A Model is a plain-form `Machine`: state as `var`s, `object transitions extends
  Transitions`, `object effects`, and the claim sections (fn-190). The samples in `samples/` are the
  canonical examples; every example here is current unless marked otherwise.
- **Checked natively.** Today the plain form is checked by the Scala checker.
  Lifting a plain-form Machine to the IR is phase 2 of fn-190 (fn-190.9
  and on) and not done: the exporter (`export/`) and several lints still read the old record form.
- **Status words.** *Current*: write it. *Legacy*: public, but only the old record form uses it; do
  not write it in a new Model. *Retired*: removed or `private[framework]`; [Retired
  forms](#retired-forms) maps each to its replacement. *Pending*: specified, not yet implemented.
- **Who refuses.** Every refusal names its layer:
  - *compile*: the Scala compiler or a DSL macro, at the author's line;
  - *plugin*: the reify compiler plugin (`reify/plugin`);
  - *runtime*: a `require` or exception as objects construct or a check runs (`require` texts carry
    the prefix `requirement failed: `);
  - *lint*: `make lint-model` (`lint/`, `check/lint/`);
  - *export*: the IR generator, at the Scala line;
  - *reader*: the Go reader (`tools/umpire`) at admission.

Contents:

1. [Conventions](#1-conventions)
2. [Domains and data](#2-domains-and-data)
3. [The signature](#3-the-signature)
4. [The Machine](#4-the-machine)
5. [Transitions](#5-transitions)
6. [Effects](#6-effects)
7. [Invariants and postconditions](#7-invariants-and-postconditions)
8. [Monitors](#8-monitors)
9. [Progress](#9-progress)
10. [Scenarios](#10-scenarios)
11. [Queries](#11-queries)
12. [Checking and results](#12-checking-and-results)
13. [Holes and gaps](#13-holes-and-gaps)
14. [Refinement](#14-refinement)
15. [Derivations and markers](#15-derivations-and-markers)
16. [Facets](#16-facets)
17. [Capabilities](#17-capabilities)
18. [Compositions](#18-compositions)
19. [Channels and inboxes](#19-channels-and-inboxes)
20. [Faults, crashes and durability](#20-faults-crashes-and-durability)
21. [Realizations](#21-realizations)
22. [The IR boundary](#22-the-ir-boundary)
23. [Lints](#23-lints)
24. [Retired forms](#retired-forms)
25. [Known gaps](#known-gaps)
26. [Public surface not in this reference](#public-surface-not-in-this-reference)

---

## 1. Conventions

| Rule | Detail | Enforced by |
|---|---|---|
| Imports | `import framework.dsl.*` and `import framework.dsl.Finite.given`; a Model on the shared outcomes adds `import framework.dsl.outcomes.Outcome` | compile |
| Strict equality | The build uses `-language:strictEquality`. `Finite.given` (or `Data.given`) supplies `==` between values of one domain; comparing two levels' types, unions or widened types does not compile | compile |
| Names | A Machine, actor, action, scenario, hole and claim is named after its `object` or `val`. A Machine's name is its object name with the first letter lowered: `object Model` is `model` | runtime |
| One body per line | Every `val`/`def` member of a native Model and its sections starts its body on the next line: `R4.declaration-multiline: <name> starts its body on the definition line: put the body on the next line`. `var`s, givens, locals and test files are exempt | lint |
| Colon blocks | `on`, `from`, `oneOf`, `branch` and the other block words take a colon block, not braces: `block-form rule: use a colon block for \`<name>\`, not optional braces` | lint |
| Comments | `//` only | lint |
| Layout | In `framework/samples` and `temporal/features`, scalafmt keeps the author's line breaks (`newlines.source = keep`), so a long `if` may break after `then` | `make fmt-model` |

Section order inside a Machine, as the samples write it: vars, `refinement`, `transitions`,
`effects`, `invariants`, `postconditions`, `monitors`, `progress`, `queries`. A named scenario is a
Machine-level `val` just before the section that uses it. No lint holds the plain-form order yet
([Known gaps](#known-gaps)).

---

## 2. Domains and data

### Type classes

| Form | Meaning | Required for | Refusal (compile) |
|---|---|---|---|
| `Data[T]` | Structural total order and equality. Never enumerates | state, monitors, comparisons | `${T} has no structural data comparison; its members must also support Data.` |
| `StateData[T]` | Admits `T` as native state: `Boolean`, `Int`, `UpTo[N]`, `Option`, `List`, `Set`, `Map`, `Inbox`, `Nothing`, and enums and case classes of these. `String`, `Long`, `Double` and opaque types have none | state vars | `${T} is not supported native state data; use enums, records, Boolean, Int, UpTo, Option, Set, Map, List or Inbox with supported members` |
| `Finite[T]` | A catalog: the ordered list of a type's values, `values: IndexedSeq[T]` | outcomes, facts, action inputs, channel messages. Not state vars | `${T} has no finite catalog. A finite domain is an enum, a case class whose fields are finite domains, Boolean, Option of a finite domain or UpTo[N].` |

### Catalog order

Catalog order is semantic: it orders an action's classes. Defaults never come from it.

| Type | Catalog |
|---|---|
| `Boolean` | `false, true` |
| `UpTo[N]` | `0 .. N` |
| `Option[A]` | `None`, then `Some(a)` in `A`'s order |
| enum | cases in declaration order; a case with fields once per assignment of them |
| case class | product of its fields in declaration order, the last field varying fastest |
| `Nothing` | empty: the fact type of a machine that records no facts |
| `Finite.of(v*)` | as written; duplicates and emptiness are not checked |
| `Finite.upTo(n)` | `0 .. n`, for a named opaque integer type |

No `derives` clause is needed: enums and case classes derive their catalog where one is first
needed. A more specific given wins: `Option`, `UpTo`, or a Model's own `given Finite[T]`.

### Forms

| Form | Signature | Meaning | Refusals |
|---|---|---|---|
| `UpTo[N]` | `opaque type UpTo[N <: Int] <: Int`; `UpTo(n)` | A counter `0 .. N` that reads as an `Int`. Being opaque, `x += 1` does not type-check: write `x = UpTo(x + 1)` | runtime `requirement failed: <n> is outside 0..<N>`; on a var, compile `<v>: <T> takes a literal bound, as UpTo[3] does` |
| `Finite.of[T](vs*)` | `Finite[T]` | Explicit catalog | — |
| `Finite.upTo(hi)` | `Finite[Int]` | `0 .. hi`. Pattern: `opaque type Retries = Int; object Retries: given Finite[Retries] = Finite.upTo(2)`. Such a type is an outcome, fact, input or message, not a state var | — |
| `domains.oneOf(v*)`, `domains.oneOf(range)` | `Finite[A]`, `Finite[Int]` | An input catalog, in written order (a range in its iteration order). A collection argument is one value: `domains.oneOf(Set(2, 0))` has one value | runtime `an explicit input domain must not be empty`; `an explicit input domain must not contain duplicate values` |

Values are keyed by spelling: a case by its name, a case or record with fields by its prefix and
fields joined with `-`.

### Collection operations

What a guard, effect or claim may call on a `List`, `Set`, `Map` or `Option` depends on the layer
that reads it:

- **Native checker: any Scala.** Effects, guards and claims run as Scala, so every method of the
  standard collections works, and the samples use many of them.
- **IR lift today: a few `List` forms.** The record-form lifter accepts only the rows of the
  [Liftable subset](#liftable-subset); `Set` and `Map` are not liftable at all.
- **IR lift planned: the builtins table.** fn-192 (open) makes one table, `model/builtins.json`, the
  complete list of operators the lifter, the Go reader and the Quint export know; "a builtin not in
  it does not exist". Its initial `set` and `map` families are fn-173.4's native collection
  operations.

| Operation | Native | Liftable today | fn-192 builtin | Used in samples |
|---|---|---|---|---|
| `contains` | yes | `List` (`OP_CONTAINS`) | `list`, `set`, `map` | Inputs, ReliableBroadcast, TwoPhaseCommit, Voting |
| `isEmpty` | yes | — | `list`, `set`, `map`, `option` | Bank, ReliableBroadcast, Voting |
| `size` | yes | — | `list`, `set`, `map` | ReliableBroadcast |
| `incl`, `excl` | yes | — | `set` | Inputs, TwoPhaseCommit, Voting (`incl`) |
| `get`, `getOrElse` | yes | — | `map`; `option` (`getOrElse`) | Bank, ReliableBroadcast |
| `updated`, `removed` | yes | — | `map` | Bank, ReliableBroadcast, TwoPhaseCommit, Voting |
| `++` | yes | `List` (`OP_CONCAT`) | `list.concat` | — |
| `+:` | yes | `List` | `list.prepend` | — |
| `isDefined` | yes | — | `option` | — |
| `min`, `max` (on `Int`) | yes | yes | `int` | — |
| `toList` | yes | `List` (identity) | — | ReliableBroadcast |
| `forall`, `exists` | yes | `forall` only for a capability's fact family | — | Bank, ReliableBroadcast, TwoPhaseCommit, Voting |
| `map`, `filter`, `sum`, `values`, `toSet`, `head`, `nonEmpty` | yes | — | — | Bank, ReliableBroadcast, Inputs, Voting |

The last two rows are used by the samples, but no body lifts them today and fn-192's initial table
does not hold them: once plain-form Models are lifted, a sample that keeps them is refused unless
the table grows or the sample is rewritten ([Known gaps](#known-gaps)).

---

## 3. The signature

### Actors and steps

| Form | Where | Meaning |
|---|---|---|
| `object x extends Actor` | feature object | Something outside the system. Its actions are stimuli a test drives |
| `val a = action` | in an `Actor` | An action of that actor, named after its `val`; the actor is named after its object |
| `object x extends Internal` | feature object | The system itself |
| `val s = step` | in an `Internal` | A step the system takes on its own, which a test only observes. Its actor is `system` |
| `val t = timer` | in an `Internal` | A timer step |
| `Actor(name)` | anywhere | An actor named in place |
| `Actor.system` | — | The reserved actor of internal steps, timers, and channel deliveries and losses |

Refusal (compile), for `action` outside an `Actor`, `action` in an `Internal`, or `step`/`timer`
outside an `Internal`: `an action is declared in an Actor object; a step the system takes on its own
in an Internal object`.

Kettle example:

```scala
object user extends Actor:
  val pressButton = action
  val depressButton = action

object heater extends Internal:
  val heat = step
  val failover = step
```

### Action modifiers

Each returns the action with its input shape kept.

| Modifier | Meaning | Refusals (runtime) |
|---|---|---|
| `.crash`, `.responseLoss`, `.storageLoss` | A fault of that kind. The actor must be the object named `fault` ([Faults](#20-faults-crashes-and-durability)) | `fault metadata requires the fault actor`; `a fault requires exactly one kind` |
| `.modelOnly(because)` | A fault with no native control: no realization may perform it | `modelOnly is configured twice`; `modelOnly requires a nonempty reason` |
| `.on(entity)`, `.creates(entity)` | The entity an action acts on or creates | — |
| `.results(name)` | Names the action's result | — |
| `.example(value, text)` | A documented input example | export `action <a> example does not belong to its native action input catalog` |
| `.schema[M]` | Retired | compile: `` `.schema[T]` is retired: what a class carries is derived from the realization's binding of it (`perform(class -> instruction)`); delete the schema `` |

`Entity(name, key, refer)` and `Observation(name, on)` describe what a machine keeps state for and a
derived read; no sample uses them.

### Inputs

| Form | Signature | Meaning |
|---|---|---|
| scalar | `action.input[A](name = "n")`; also `default = v`, `domain = d`, or both | One labelled input over `A`'s catalog or the given domain. Without `default` the input is required. Works on `step` too |
| record | `action.inputs[In]` | The case class `In`'s catalog; field names and defaults come from the class |
| record, per field | `action.inputs[In](inputField[In](_.f).in(domain), ...)` | Named fields take the given domain; the others their `Finite`. Classes are the product, last field fastest |
| record, whole | `action.inputs[In](domain, bindings*)` | An explicit joint catalog, which keeps its correlations and order |
| field | `inputField[In](_.f)`, `rec.field(_.f)` | A typed descriptor of one direct field |
| bind | `field.in(domain)` | The field's catalog |

Bank example:

```scala
object client extends Actor:
  val send = action.inputs[Send](inputField[Send](_.amount).in(domains.oneOf(-1 to supply + 1)))
```

Refusals:

| Text | Layer |
|---|---|
| `a scalar input requires an explicit name`; `inputs are already declared` | runtime |
| `default for input <n> is outside its declared domain` | runtime |
| `inputs[In] requires a case class record`; `input records require structural comparison`; `input <f> has no structural comparison` | compile |
| `an input domain is supplied twice`; `an input domain names an unknown field`; `input <f> has no finite catalog` | runtime |
| `whole-record and per-field input domains cannot be combined` | runtime |
| `an input field requires a concrete case class record`; `an input field requires its declared field type`; `a member selector must name exactly one direct state field` | compile |
| `action <a> input <i> default does not belong to its native action input catalog` | export |

### Classes

A class is one assignment of an action's inputs. `type ClassRef = Class | Action[EmptyTuple]`.

| Form | Meaning | Refusals |
|---|---|---|
| `a()` | A zero-input action's one class, or every input at its default | runtime `input <n> of <a> has no default: supply it` (scalar); `input <n> has no default: supply it` (record) |
| `a(value)` | The class of that value: `rm.prepare(ResourceManager.rm1)`. A one-field record takes the bare field value | compile `supply the action's declared input type`; runtime `input value of <a> is outside its declared domain` |
| `a(f = v, ...)` | Built through the record's `apply`, so omitted fields take their defaults: `client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1)` | compile `named inputs require a record action`; `an input is supplied twice`; `unknown input <n>`; `missing argument for parameter <f>` |

### Phases and roles

Role traits mix into enum cases as metadata and add no state: `Live`, `Waiting`,
`Retrying`, `Held`, `Suspended`, `Closed`, `Succeeded`, `Failed`, `Canceled`, `Terminated`,
`TimedOut`. A Model may declare its own, `trait Expired extends TimedOut`.

| Form | Meaning |
|---|---|
| `case backingOff extends Roled, Retrying` | A phase with a role |
| `phase.in[Closed]`, `case _: Closed` | Membership in a role |
| `phase.in(a, b, ...)` | Membership in listed cases |
| `a implies b` | `!a \|\| b` |

Refusals: lint `R4.dotted-in` (`write the membership test dotted, value.in(...)`), `R9.phase-import`
(an `import Phase.*`), `R4.enum-wildcard` (a whole-case wildcard over a phase enum); plugin `a phase
is live or closed, never both`, `a live phase is at most one of Waiting, Held and Suspended`, `a
closed phase has at most one closure role of Succeeded, Failed, Canceled, Terminated and TimedOut`;
runtime `<owner>'s phase type <T> has no <role> case`.

---

## 4. The Machine

### `Machine[O, F]`

```scala
object Model extends Machine[Outcome, Nothing]:
  var temperature = oneOf(coldestTemperature to 40)
  var heatingOn = false
  var beeping = false
```

- **Type parameters.** `O` is the outcome type, `F` the fact type; both need a `Finite`. `Nothing`
  records no facts.
- **`given Ok[O]`.** The outcome of a step that fires and does not reject. `Transitions` needs it.
  Declare it beside the outcome enum, `given Ok[Outcome] = Ok(Outcome.accepted)`; the shared
  `outcomes.Outcome` brings its own.
- **Shared outcomes.** `framework.dsl.outcomes`:
  `enum Outcome { accepted; rejected(why: Rejection) }`,
  `enum Rejection { notFound, alreadyExists, failedPrecondition, invalidArgument }`.

### State vars

| Form | Meaning |
|---|---|
| `var x = e`, `var x: T = e` | One field of the state, in declaration order. The declared type is its schema, the initialiser its start value. Types: anything with `StateData`, including unbounded `Int`, `Map`, `Set`, `List`, `Option` |
| `var x = oneOf(domain)` | A nondeterministic start over the elements of a collection or range. Duplicates drop; `init` takes the first element |
| reading `x` | In guards, effects, claims, monitors, the refinement, `end` and `at`. The checker restores a state into the vars before each step or claim; between evaluations they read as `init` |

The starts are the product of every `oneOf` domain, in var declaration order, the last var varying
fastest. Every claim is checked from every start.

Refusals:

| Text | Layer |
|---|---|
| `a Machine declares at least one state var` | compile |
| `<v>: <T> has no IR schema; use Int, a bounded alias, Boolean, an enum, a record, Option, Set, Map, List or Inbox` | compile |
| `<v>: <T> takes a literal bound, as UpTo[3] does` | compile |
| `a Machine's vars are read where its object extends Machine` | compile |
| `<machine>: <v> has no public accessors` (a `private var`) | runtime |
| `<v>: oneOf takes a non-empty domain`; `<v>: oneOf declares one initial domain` | runtime |
| `oneOf(domain) initializes a state var` (a `oneOf` outside a var initialiser) | compile |

### Members

| Member | Signature | Meaning |
|---|---|---|
| `init` | `final def init: State` | The vars' initial values. Final: `val init = ...` does not compile |
| `initialStates` | `Seq[State]` | Every start |
| `end` | `def end: Boolean = true` | Where a run may stop, read from the vars. Default: anywhere. Override with `override def end = ...` |
| `at` | `final def at[A](state: State)(evaluate: => A): A` | Evaluates `evaluate` with `state` in the vars, then restores them. Nests |
| `vars` | `Vector[Var]` | Each var's name, declared type as written, bound and Inbox message type |
| `name` | `String` | The object name, first letter lowered |
| `State` | `final class State extends Selectable` | The state record. `s.temperature` reads a field; `Model.init.copy(heatingOn = true)` updates by name |
| `current` | — | Not a DSL member: samples define `def current = Transaction(...)` to bundle the vars for helpers |

`State.copy` refusals (runtime): `<x> is not a state var of <machine>`; `<x>: <v> is not a <T>`;
`<x>: <n> is out of UpTo[N]`; `State.copy.<m> is not an update of a state`;
`<v> is a state of another machine than <m>`.

### Sections

A Machine holds these objects. Each is found by name.

| Section | Holds | See |
|---|---|---|
| `object transitions extends Transitions` | when each action fires. Required, may be empty | [5](#5-transitions) |
| `object effects` | what each step changes | [6](#6-effects) |
| `object refinement extends Refinement(product)` | the map to the machine it refines | [14](#14-refinement) |
| `object invariants` | claims about every reached state | [7](#7-invariants-and-postconditions) |
| `object postconditions` | claims about every step | [7](#7-invariants-and-postconditions) |
| `object monitors` | claims about a path's history | [8](#8-monitors) |
| `object progress` | claims that something follows | [9](#9-progress) |
| `object capabilities extends Capabilities` | shared promises by binding (legacy) | [17](#17-capabilities) |
| `object queries` | `verify`/`find` checks and their targets | [11](#11-queries) |

`states`, `rules` and `properties` (with `safety`/`liveness`) are retired names.

### Markers

| Marker | Meaning |
|---|---|
| `NegativeControl` | A deliberately wrong design. R15 moves negative controls into tests, where the samples now keep them |
| `FailureModel` | A design that binds faults: `object Model extends Machine[Outcome, Nothing], FailureModel` |

Lint refusals:
`<n> binds the fault <a> and is marked neither a FailureModel ... nor a NegativeControl ...: mix in the one it is`;
`<n> is a failure model and binds no fault: bind an action of the actor \`fault\` that some state enables`;
`<n> is marked a failure model and a negative control: it is one`;
`<n> is a negative control that nothing the run checks can refute: ...`.

---

## 5. Transitions

```scala
object transitions extends Transitions:
  on(user.pressButton):
    if !heatingOn then effects.pressButton()
  on(heater.heat):
    if heatingOn && temperature < boilingTemperature then effects.heat()
```

### Forms

| Form | Meaning |
|---|---|
| `on(a): body` | The block of an action with no input |
| `on(a): in => body` | The block of an action with an input, given as the parameter |
| `on(a(value)): body` | The block of one class |
| `on(a, b): body` (up to four) | One block for several actions or classes; takes no input |
| `from(declarer): on(...) ...` | Groups the blocks of one declarer's actions |
| `disabled(a, ...)` | Actions that never fire |
| `crashes(fault.a, outcome, facts*)` | A generated crash ([20](#20-faults-crashes-and-durability)) |

### What a block may contain

- **Control flow:** `if`/`else`, `match` (with `case a | b`, guards, `case _: Role`, `case _ =>`),
  local `val`s.
- **Leaves**, one of:
  - a call of an effect, a `Unit` method of an object named `effects` (also another object's or
    facet's);
  - `reject(r)`;
  - `()`.
- **Meaning of a leaf:** an effect call fires one step with the `Ok` outcome into the state the
  effect leaves; `reject(r)` answers `r` in the start state; `()`, or an `if` with no `else`, is
  disabled there: no step.
- **Order:** first match wins, as in Scala. Several blocks for one class each become a rule; the
  first that yields a step wins. An action's row in the table is its first mention.
- **Not bound:** an action no block names is not bound by the Machine. There is no construction-time
  overlap check.

### Refusals

| Text | Layer |
|---|---|
| `a transition says when; write the state change in effects` (an assignment in a block) | compile |
| `a transition ends in an effect call, a rejection or (): <source> is none of these` | compile |
| `on(<a>) names its action bare at <pos>: name it through the object that declares it, \`on(<declarer>.<a>)\`` | runtime |
| `on(<a>) sits in on(<b>): a block holds cases alone` | runtime |
| `<a> is named twice in one on: name each action, or class of it, once` | runtime |
| `<a> is disabled and fired by a rule`; `an action is disabled twice, or disabled and bound by a rule` | runtime |
| `on(<a>) sits in from(<d>), and <d> declares no <a>: a from holds the blocks of the actions its declarer declares` | runtime |
| `on(<a>) reads an input it does not declare` | runtime |
| a non-exhaustive `match` | compile (strict flags) |

---

## 6. Effects

```scala
object effects:
  // turn on the heating element
  def pressButton() =
    heatingOn = true
    beeping = false
  def heat() =
    temperature += 1
```

- **Where:** an object named exactly `effects`, directly in a Machine, a facet's namespace or a
  `Derived`.
- **What:** a method whose body assigns the vars, with `if`, `match`, local `val`s, imports and
  calls of other effects. An input is an ordinary parameter, `def send(in: Send) =`. Scala's order
  holds: a read after a write sees the write. An empty body, `()`, fires a step into an unchanged
  state.
- **When:** an effect runs only as the leaf of a transition.

### Statements

| Statement | Meaning | Refusals |
|---|---|---|
| `reject(outcome)` | The step answers `outcome` and keeps the start state; its assignments and facts are dropped. Execution continues; the last `reject` wins | compile `this rejects, which only an \`effect:\` block does, or an effect of a Machine that declares its state as vars: ...`; runtime `<machine>: a rejection is a statement of an effect, which its transition runs` |
| `reject(why: Rejection)` | On the shared outcomes: answers `rejected(why)` | compile `reject(Rejection) answers the shared framework.dsl.outcomes.Outcome, so it rejects in a Machine on the shared outcomes: ...` |
| `records(fact)` | The step records the fact, in call order | runtime `a fact an effect records is a statement of the step its transition runs` |
| `record(fact, more*)` | The same, several at once | runtime `<machine>: a recorded fact is a statement of an effect, which its transition runs` |
| `val t = choice`; `choose(t1 -> { ... }, t2 -> { ... })` | Named alternatives: one step per alternative, the transition re-run with each pick | runtime `requirement failed: a choice names one alternative of a choose`; `choose is a statement of an effect, which its transition runs` |

Do not use in an effect:

- **`oneOf:`/`branch:`** are claim alternatives: in an effect they compile and do nothing. Use
  `choose`.
- **`check`** compiles and does nothing.
- **`before`, `outcome`, `after(a):`** are refused (compile):
  `before is the state a step started from; use it in a postcondition`,
  `outcome is what a step answered: read it in a postcondition or a monitor`,
  `after(action) is a postcondition: write it in postconditions`.

### Definition rules

| Rule | Text | Layer |
|---|---|---|
| Every `def` in `effects` returns `Unit`; a non-`Unit` helper lives outside `effects` | `<name>: effect definitions must return Unit` | plugin |
| The `Unit` result is inferred | `R4.effect-return-type: <name> declares a return type: let its Unit result be inferred` | lint |
| The body starts on the next line | `R4.declaration-multiline: ...` | lint |

---

## 7. Invariants and postconditions

```scala
object invariants:
  val beepsOnlyWhenOff =
    if beeping then
      check(!heatingOn)

object postconditions:
  val warmsOnlyWhenOn =
    if !before.heatingOn then
      check(temperature <= before.temperature)
  val failsOverAtBoiling =
    after(heater.failover):
      check(temperature == boilingTemperature)
      check(beeping)
```

### Sections

| Section | Read on | Members |
|---|---|---|
| `object invariants` | every reached state, starts included | `val`s of type `Unit`, or `ScopedClaim` when they end in `in env` |
| `object postconditions` | every step, from its start to the state it reaches; a rejection reaches its start | as above |

A member is a block of claim words. A member that is not a claim, such as `val limit = 3`, fails the
check (runtime): `<machine>.<kind>.<member> holds <value>: a member of <kind> is a claim, a block of
check, records, if or oneOf, and a requirement block must not be empty or contain only helper vals`.

### Claim words

| Word | Where | Meaning |
|---|---|---|
| `check(cond)` | any claim block | One assertion. The first failing assertion stops its member; other members are independent |
| `if c then check(...)` | any claim | Plain Scala: skips some states or steps |
| `after(a):` | `postconditions`, `queries` | The block judges only steps of action `a` (any class), or of one class with `after(a(input)):`. Other steps do not judge the member |
| `before` | `postconditions`, `monitors`, `queries` | The State record the step started from, typed by the vars: `before.temperature` |
| `outcome` | `postconditions`, `monitors`, `queries` | The step's outcome, typed `O` |
| `records(fact)` | any claim | The step recorded `fact` |
| `oneOf:` with `branch:` blocks | any claim | Holds when one branch holds with no hole; at least two branches |
| `x.in(a, b)`, `a implies b` | anywhere | Boolean sugar |
| `{ ... } in env`, `(after(a): ...) in env` | `invariants`, `postconditions`, `queries` | The claim is checked in its own [scenario](#10-scenarios), not the shared exploration |

`in` scopes only the statement it ends. A claim of several statements is scoped whole, with braces
or parentheses. The block-form lint refuses a member whose block of several statements ends in
`... in env`:
`` block-form rule: `<name>`: `in` scopes only its own statement; scope the whole claim: `{ ... } in env` ``.

### When a claim gets a word

A claim starts with a DSL word only when the word changes what is checked: `after(a)` changes which
steps are read, `whenever` when the later state is read. A claim that only skips some states or
steps is a plain `if`.

### Refusals (compile)

| Text | Trigger |
|---|---|
| `before is the state a step started from; use it in a postcondition` | `before` in `invariants` or `progress` |
| `before is the state a step of a Machine started from` | `before` outside a Machine |
| `outcome is what a step answered: read it in a postcondition or a monitor` | `outcome` in `invariants` |
| `<word> reads a step of a Machine: write it in one` | `outcome` or `records` outside a Machine |
| `after(action) is a postcondition: write it in postconditions` | `after` in `invariants`, `progress` or `monitors` |
| `an invariant or postcondition belongs in invariants or postconditions` | `check` in `progress`, or a block `in env` outside the three sections |
| `a requirement block must not be empty or contain only helper vals` | `after`, `branch` or `... in env` with no `check`, `records` or `oneOf` |
| `oneOf requires at least two branches` | one branch |
| `branch is an alternative of oneOf: write it inside oneOf` | a stray `branch` |
| `in takes a scenario: only(...), fair(...), starts(...) or a named one` | `in` with no scenario |
| `value <y> is not a member of framework.dsl.State` | `before.y` for no var `y` |

A claim section of a class instance (a facet's) is refused when checked:
`<cls> is a claim section of an instance, such as a facet's, which no native check evaluates: declare the claims in the Machine object's own sections`.

---

## 8. Monitors

```scala
object monitors:
  val agreement =
    monitor(false)(_ => !agreed(current))(identity).readAtEnds
```

| Form | Signature | Meaning |
|---|---|---|
| `monitor(initial)(next)(violated)` | `monitor[M](initial: M)(next: M => M)(violated: M => Boolean)` | A history: `next` reads the vars after each step, plus `before`, `outcome`, `facts` and `recorded(fact)` of that step. Named after its `val` |
| `sticky(promise)` | `sticky(promise: => Boolean)` | Once broken, stays broken |
| `.readAtEnds` | — | The verdict is read only in states where `end` holds, starts included |
| `.readAfter(at)` | `at: Step => Boolean` | The verdict is read only after the steps `at` accepts |
| `facts`, `recorded(fact)` | — | The watched step's facts |

The default reading point is after every step. A monitor runs on the whole machine: it takes no `in`
and is no `verify`/`find` target. Checking explores (state, monitor state) pairs from every start;
the budget counts the pairs. Verdicts: `refuted` with `counterexample: <path>`, `verified`, or
`unknown` (a cut, or a hole in `next`, `violated` or `readAfter`). A monitor over an unbounded
history, such as a growing `List`, is `unknown` at any budget.

Legacy: the three-argument `monitor[S, O, F, M](initial)((m, state, step) => ...)(violated)` and
`stickyAcross`.

---

## 9. Progress

```scala
val neverSwitchedOff =
  without(user.depressButton)

object progress:
  val boils =
    (whenever(heatingOn):
      eventually(beeping)
    ) in neverSwitchedOff
  val agreement =
    whenever(!agreed(current)):
      eventually(agreed(current), within = 1)
```

### Forms

| Form | Meaning |
|---|---|
| `whenever(p): eventually(q)` | From every reached state where `p` holds and `q` does not, every admitted path reaches a state where `q` holds. Unbounded; checked under the scenario's weak fairness |
| `whenever(p): eventually(q, within = n)` | The same within `n` steps. `n` is a positive constant: a literal, or arithmetic over `inline val`s |
| `leadsTo(from = p, to = q, within = n)` | The bounded claim, older spelling |
| `(...) in env` | The claim's environment, a [scenario](#10-scenarios). Default: every path from the Machine's starts, nothing fair |

A claim is named after its `val`. The parentheses before `in` are required: inside the block, `in`
would scope only the `eventually`, and after the block a word operator cannot start a line.

`LeadsTo` is the claim value: `name`, `machine`, `from`, `to`, `within: Option[Int]`, `scenario`,
and `in(scenario)` to replace the environment. `Deadline` is what `eventually` builds, and
`Whenever` the evidence only a `whenever` block supplies.

### Verdicts

The region is the reached states where `from` holds and `to` does not, closed under successors that
are not `to` states. Tested in this order:

| Verdict | When |
|---|---|
| `deadlock` | A region state has no successor (and reaches no hole). `end` is ignored |
| `cycle` | A cyclic part of the region that no fair class leaves. A fair class leaves a part when it is enabled at every state of it and never taken inside it. With no fairness, any cycle |
| `missed deadline` | Bounded: every cycle is left by a fair class, yet a cycle can still avoid `to` for any number of steps; or the longest path reaches the bound |
| `verified` | Otherwise, on a complete exploration. `longest` is the longest path to the first `to` state, `Some(0)` for an empty region, and `None` when an unbounded claim's region cycles |
| `unknown` | The exploration was cut, or `from` or `to` reached a hole |

A bound gains nothing from fairness: any cycle that avoids `to` fails a bounded claim. State a bound
only where it is the promise ("repaired within one step"), not a count of the model's steps; the
checker reports the longest path anyway.

Detail lines:

- `deadlock: <a>, <b>, ... ends in <state>`
- `<subject>: cycle <a>, <b>, ...: <action> stays enabled and is never taken; add fair(<action>) to the scenario, or admit fewer actions with only`
- `<subject>: cycle <a>, ...: every action enabled on it is taken; admit fewer actions with only`
- `<subject>: a cycle can avoid a to state for any number of steps`
- `<subject>: a path of <n> steps reaches no to state`

### Refusals (compile)

| Text | Trigger |
|---|---|
| `whenever, a progress claim, belongs in progress` | `whenever` outside `progress` |
| `a progress claim belongs in progress` | `leadsTo` outside `progress` |
| `eventually is the deadline of a whenever block` | `eventually` outside a `whenever` block |
| `within is a positive number of steps, written as a literal` | `within` zero, negative, or not a constant |
| `in takes a scenario: only(...), without(...), fair(...), starts(...) or a named one` | `in` with no scenario |
| `a progress claim is verified, not found` | `find(progress.x)` |

---

## 10. Scenarios

| Builder | Meaning |
|---|---|
| `only(a, ...)` | Admits these and no others: whole actions, or single classes `a(input)` |
| `without(a, ...)` | Admits every action but these. Dropping one class leaves the action's other classes admitted. Accumulates |
| `fair(a, ...)` | Makes these weakly fair: one that stays enabled is eventually taken. A whole action makes every class fair. Admits every action. Accumulates. Nothing is fair by default |
| `starts(x = v, ...)` | Starts from `init` with the named vars changed. Other `oneOf` vars take their first value |
| `starts()` | The Machine's own starts, unchanged |
| `actions(c, ...)` | A pinned schedule: exactly these classes, in order |

Builders chain: `only(a, b).fair(a).starts(heatingOn = true)`. A scenario is named after its `val`
and belongs to the enclosing Machine; an unnamed one takes the name of the claim it scopes.
`modelOnly` is true when it says `only`, `without` or `fair`: a realization cannot drive such an
environment.

Refusals (runtime):

| Text | Trigger |
|---|---|
| `a scenario admits actions with only or pins them with actions, not both` | `only` with `actions` |
| `a scenario admits actions with only, drops them with without or pins them with actions, one of the three` | `without` with `only` or `actions` |
| `a scenario says only once: <name>` | `only` twice |
| `starts.<m> is not a start` | `starts.foo(...)` |
| `starts(<vars>) names vars, and <m> declares none` | `starts` on a machine without vars |
| `<x> is not a state var of <machine>`; `<x>: <v> is not a <T>` | unknown var, wrong type |

Legacy: `scenario.free`, `scenario.actions(...)`, `scenario starts s`, `scenario("name")`.

---

## 11. Queries

```scala
object queries:
  val peachesGetsBananas =
    after(client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1)):
      check(balances(Addr.king, Denom.banana) == 1)
      check(balances(Addr.peaches, Denom.banana) == 1)
  val kingSendsPeachesBananas =
    find(peachesGetsBananas)
```

| Form | Meaning |
|---|---|
| `verify(claim)` | Holds on every path or step of the scenario |
| `find(claim)` | Some step the claim is about holds of it; that path is the witness |
| `... in env` | Replaces the claim's own environment |
| `.live(expected)` | Marks a live Case, with its expected Run ([21](#21-realizations)) |
| `.explore(space)` | Attaches finite variations (legacy direction) |

- **Targets.** `verify` and `find` take a `Unit` `val` of `invariants`, `postconditions` or
  `queries`, a scoped claim, or (verify only) a progress claim. Monitors and scenarios are refused.
- **Find targets** are usually postcondition-shaped blocks declared in `queries` itself. `queries`
  is not checked by default: a find target is no promise.
- **Witness detail:** `witness: <path>`; a pinned `find(t) in actions(...)` reports found with no
  detail.

Refusals:

| Text | Layer |
|---|---|
| `verify takes an invariant, a postcondition or a progress claim` (or `find takes ...`) | compile |
| `in takes a scenario: only(...), fair(...), starts(...) or a named one` | compile |
| `a bound is a promise (within) or a budget (the checker's configuration); a Model declares no Limits` | compile |
| `<name> runs on the model only: its scenario restricts or assumes the environment` (`.live`/`.explore` with `only`, `without` or `fair`) | runtime |
| `a progress claim is verified, not found` | compile |

Legacy: `witness(c, ...).records(f)`, `.excludes(g)`, `.recordsWhen(cond, f)`, `.ends(p)`, a pinned
path with a last-step fact.

---

## 12. Checking and results

Import `framework.checker.Checker` alongside the DSL. Its public entry points are:

```scala
Checker.explore(instance, starts = Seq.empty, budget = Checker.defaultBudget)
Checker.verifyAll(instance, starts = Seq.empty, budget = Checker.defaultBudget)
Checker.verifyAll(explorationResult)
Checker.monitor(instance, watch, starts, budget = Checker.defaultBudget)
Checker.progress(instance, claim, starts, budget = Checker.defaultBudget)
```

`explore` drains one bounded reachable search. Passing that Result to `verifyAll` reuses it.
The private NativeCheck query and negative-control helpers are internal test machinery.
Replay, horizon-bounded finding, sampling and feature-budget configuration remain separate work.

### What is checked by default

`Checker.verifyAll(machine, starts = Seq.empty, budget = Checker.defaultBudget)`:

- **Starts:** `machine.initialStates`, unless `starts` is given.
- **Unscoped claims:** `invariants` and `postconditions` share one exploration of every action.
- **Scoped claims and progress:** each in its own environment.
- **Monitors:** each on the product with the machine.
- **Not checked:** `queries`.
- **Order of results:** unscoped claims (invariants, then postconditions), scoped claims, progress,
  monitors.

Query execution and negative controls currently use internal helpers; they are not additional
public Checker operations.

### Budget

`defaultBudget` is `1 << 20` states. A cut exploration answers `unknown`, never `verified`; a
counterexample found before the cut stands. Refusal: `a budget admits at least one state`.

### `Verdict`

| Verdict | Shown | Meaning |
|---|---|---|
| `verified` | `verified` | Complete exploration, no violation |
| `found` | `found` | A `find` witness |
| `refuted` | `refuted` | A safety or progress counterexample |
| `notFound` | `notFound` | Reserved for the forthcoming bounded finding operation |
| `unknown` | `unknown` | A cut or a hole, and nothing decisive |


### `Result` and `Report`

A Result exposes `subject`, `verdict`, `budget`, `depth`, `complete`, `path`, `detail` and `longest`.
Its constructor is internal. Progress failures use `refuted`, with `reason: deadlock`, `reason: cycle`
or `reason: missed deadline` in detail.

- **`toString`:**
  `<subject>: <verdict> (budget <budget> states, <depth> steps; exploration <complete|cut>)`, for
  example `model.boils: verified (budget 1048576 states, 82 steps; exploration complete)`. `budget`
  is the cap, `depth` the deepest breadth-first level reached.
- **`subject`:** `<machine>.<member>`.
- **`longest`:** set only for a verified progress claim.
- **`path`:** the initial state and ordered PathStep entries: action, source, destination, outcome
  and facts. These are stored exploration edges, not a reconstructed guess.

`Report(results, warnings)`; `verdicts` keys by member name; `result(member)` throws `no check named
<member>`.

Detail lines:

| Kind | Lines |
|---|---|
| invariant | `counterexample: <a>, <b>`, `in <state>`, `fails at line <n>: <source>` |
| postcondition | `counterexample: <path>`, `from <s> to <t> answering <outcome>`, `fails at line <n>: <source>` |
| find | `witness: <a>, <b>` |
| pinned | `counterexample at step <i>, <class>` or `counterexample at initial state` |
| refinement | `reads <m> through a refinement, which no native check reads` |
| hole | `unknown: <subject> ... reaches <hole>` |

A class is shown as `action` or `action(v1, v2)`.

### Vacuity warnings

Emitted only on a complete exploration:

| Warning | Meaning |
|---|---|
| `<machine>: <action> never fires` | A bound action never taken |
| `<machine>.<claim>: <action> never fires` | An admitted action never taken in a scoped claim's environment |
| `<machine>.<postcondition>: <target> never fires, the postcondition is vacuous` | An `after(target)` never triggered |
| `<machine>.<postcondition>: no steps fire, the postcondition is vacuous` | No step at all |

None is emitted for a progress claim with an empty region, a monitor, or an `if` claim whose
condition never holds.

---

## 13. Holes and gaps

| Form | Meaning |
|---|---|
| `val missing = hole` | A declared unknown, named after its `val`, declared in an object of the feature |
| `missing.reached` | Reads the hole, in a `check`, a guard, an effect, a progress `from`/`to` or a monitor |
| `@gap(subjects*)(because, until = "")` | An annotation the exporter reads; the native checker ignores it |

A step or predicate that reaches a hole is neither disabled nor an error: its result is `unknown`,
naming the hole.

- Independent claims keep their verdicts.
- A `oneOf` branch that holds with no hole establishes the claim; `false` with a hole is `unknown`.
- A pinned counterexample before the hole stands, with `complete = false`.
- In progress, a hole in every outgoing row does not establish a deadlock, and a known cycle
  survives an unrelated hole.

`@gap` subjects: `silent`, `byDefault`, `neverEnabled`, `unconstrained`, `unproduced`,
`unperformed`, `unevidenced`, `uncovered`, `unmapped`, `unrealized`, `unread`, `witnessOnly`,
`untaken`, `unasked`, `unfired`, `unreachable`, `stuck`, `unreadObservation`, `explicitWait`,
`unreachableBinding`, `mustNotPinned`.

---

## 14. Refinement

```scala
object refinement extends Refinement(Consensus):
  def toProduct(state: State) =
    Consensus.init.copy(chosen = chosen(current))
  val visibleOutcomes =
    (_: Voting.Outcome) => false
  val unobservable =
    List(voter.increaseMaxBal)
```

| Member | Meaning |
|---|---|
| `Refinement(product)` | The machine this one refines. One `object refinement` per Machine |
| `def toProduct(state: State)` | Maps a state, read through the vars, to the product's state |
| `image(s)` | `toProduct` with `s` restored into the vars |
| `visible`, `visibleOutcomes` | The facts and outcomes the product sees |
| `unobservable` | Steps that record nothing a Run can read |

- **Checking:** no native refinement check exists. A Query that reads a product's claim through the
  refinement answers `unknown`: `reads <P> through a refinement, which no native check reads`. The
  Go engine checks refinement on the IR.
- **Typing:** every Machine's state type is `State`, so the compiler no longer stops a refinement
  from mapping into the wrong machine.

Refusals (export, old form): `<n> is a member of <name>'s refinement: declare it in \`object
refinement extends Refinement(product)\`...`;
`<name>'s refinement names the machine it refines, \`Refinement(product)\``.

---

## 15. Derivations and markers

```scala
object Generous extends Derived(
  Game.rebind(
    on(player.toss):
      if Game.score < 2 then Game.effects.toss()
    ,
    on(player.pay): amount =>
      Game.effects.payTwice(amount)))
```

| Operation | Meaning |
|---|---|
| `Derived(source.op(...))` | A machine that changes one thing of `source` |
| `rebind(on(a): ...)` | Replaces the whole binding of each named action |
| `extend(on(a): ...)` | Adds blocks for actions the source does not bind |
| `unmonitored` | Drops monitors and the refinement |
| `refining(product)(map)` | Replaces the refinement (no plain-form example) |

The `on` blocks follow the [transitions](#5-transitions) rules.

- **Inherited and final:** `init`, `initialStates`, `end`, `transitions` and the vars. A `var` in a
  `Derived` object adds no state.
- **Own sections:** a derived object may add sections, such as `object invariants`.
- **Default checks:** these read its own sections only, not the source's.
- **Where:** negative controls are derivations in test files.

Refusals:

| Text | Layer |
|---|---|
| `a transition says when; write the state change in effects`; `a transition ends in an effect call, a rejection or ()` | compile |
| `a block of plain control flow binds a Machine that declares its state as vars` | compile |
| `a block of plain control flow takes its accepted outcome from a given Ok[<O>]` | compile |
| `on(<a>) names its action bare: ...`; `on(<a>) reads an input it does not declare` | runtime |
| `Machine derivation Rules: rebind action <id> which the source does not bind`; `... extend action <id> which the source binds already`; `... binds an action twice` | export (natively, a rebind of an unbound action is ignored) |

Retired: `restrict`, `assuming`, `rebind(a ~> e)`.

---

## 16. Facets

```scala
trait Pressing extends Facet[Said, Note]:
  var light: Lit
  object pressing:
    object transitions extends Transitions:
      on(hand.press):
        effects.toggle()
    object effects:
      def toggle() =
        light = if light == Lit.on then Lit.off else Lit.on

object Faceted extends Machine[Said, Note], Pressing, Breaking:
  var light = Lit.off
  object transitions extends Facets(pressing.transitions, breaking.transitions)
```

- **A facet** is a trait `extends Facet[O, F]`, in a file named after it. It declares the vars it
  uses abstractly (`var light: Lit`) and holds one namespace object with its `transitions`,
  `effects`, and optionally `capabilities` and claim sections.
- **Dependencies** are self-types: `this: Pressing =>`.
- **The Machine** mixes the facets in, defines their vars, and lists their transitions in
  `Facets(...)`, whose order is the facet order. The vars, `end`, the refinement, monitors and
  whole-feature claims stay on the Machine.
- **Claims:** a facet's claim sections are not checked natively; declare claims in the Machine
  object's own sections.

Refusals:

| Text | Layer |
|---|---|
| `<action> is bound by the facets <a> and <b>: an action belongs to one facet, which holds all its transitions` | runtime |
| `<section> is not a facet's transitions: Facets lists the transitions declared in its facets; ...` | runtime |
| `<v>: <Trait> defines a var, which a facet declares abstractly and the Machine defines: \`var v: T\`` | compile |
| A facet's file must be named after the facet | lint |

`FacetCapabilities(...)` lists the facets' capability sections (legacy).

---

## 17. Capabilities

Legacy; read only by the exporter.

```scala
object capabilities extends Capabilities:
  val closable: Capability = Closable(...)
  except(Closable.staysClosed, because = "...")
```

| Form | Meaning |
|---|---|
| `val x: Capability = Kind(...)` | Brings the Properties `Kind`'s companion (a `CapabilityKind`) defines |
| `except(Kind.p, because = ...)` | Waives one brought Property |
| `overriding(Kind.p -> substitute, because = ...)` | Replaces one |

A plain-form Model cannot write a capability kind today: its Properties are written with `property`,
which is retired, and the Limits-based capability bound form is removed. The native checker
ignores capabilities. See [Known gaps](#known-gaps).

---

## 18. Compositions

Members are vals naming Machines or compositions. Their var snapshots form the composed state;
there is no authored state case class or selector pairing. The type parameter is the composition's
fact type, with a case wrapping each fact-recording member's fact type.

```scala
enum JoinedFact:
  case left(value: Fact)
  case right(value: Foreign)

object Together extends Composition[JoinedFact]:
  val left = Subject
  val right = Other

  object syncs:
    val started =
      sync(left(hand.start), right(hand.start))

  object postconditions:
    val startedTogether =
      after(syncs.started):
        check(before.left.phase == Phase.open)
        records(JoinedFact.left(Fact.completed))
```

| Form | Meaning |
|---|---|
| `Composition[F]` | The Model of its member snapshots, recording facts of type `F`; use `Nothing` when none are recorded |
| `val member = M` | A Machine or composition member, in declaration order |
| `val started = sync(a, b)` in `object syncs` | Two member actions stepping together; the val names the sync |
| `member(action)` / `member(action(input))` | Qualifies an action or class when several members bind it |
| `after(syncs.started)`, `only(action)`, `fair(member(action))` | The same claim/scenario target forms as a Machine, resolved to composed steps |
| `before.member.field` | A member's pre-step snapshot; nested compositions retain nested paths |
| `records(JoinedFact.member(fact))` | Checks the member-tagged fact; an unwrapped member fact has the wrong type |
| `Composition(Source.withMember(member = Replacement))` | A derivation retaining syncs, end, member order and fact type |
| `override def end = predicate` | Reads member vars/helpers; the default is true |

A sync combines its members' alternatives in one step, with the first member's actual outcome and
both members' tagged facts. A paired action does not also step alone; an unpaired action steps its
member. Starts are the product of member starts. Claims, monitors, progress and witnesses use the
shared on-demand checker, with the same budgets, cut/unknown behavior and fairness semantics;
one sync counts as one composed step.

Refusals name a non-Model member, repeated/shared member vars, missing or mistyped fact cases,
an unbound or ambiguous sync endpoint, an action in multiple syncs, an unknown replacement member
or an unrelated replacement. A replacement must hold the original member's vars, as that member
or a derivation of it does. Sync refusals identify the authored location.

Composition IR lifting and Go lowering remain held; these native forms do not establish those
integration proofs.

---

## 19. Channels and inboxes

```scala
object wire:
  val network: Channel[Msg] =
    channel[Msg](capacity = 4, order = Order.unordered, loss = Loss.reliable)

object Model extends Machine[Outcome, Nothing]:
  var network = wire.network.empty
  inbox(network, wire.network)
  object transitions extends Transitions:
    on(wire.network.deliver): msg =>
      effects.receive(msg)
```

| Form | Meaning |
|---|---|
| `channel[M](capacity, order, loss, duplicates = 0)` | A channel of messages `M` (needs `Finite[M]`), named after its `val` |
| `Order.fifo`, `Order.unordered` | Delivery order |
| `Loss.reliable`, `Loss.lossy` | Whether messages may be lost |
| `ch.empty` | An empty `Inbox[M]` |
| `inbox(v, ch)` | Associates an exact owning var or rooted record/optional/typed sum-case field with the channel; the selector is inspected, not run |
| `v.send(m)`, `v.isEmpty`, `v.isFull` | Sending to a full inbox leaves it unchanged |
| `ch.deliver`, `ch.lose` | System actions `<name>Delivery`, `<name>Loss`, bound with `on(ch.deliver): m => ...` |

- **Delivery:** a delivery or loss is enabled where the inbox holds the message (the first entry for
  FIFO, any for unordered), removes it, then runs the bound effect. With `duplicates = d`, an entry
  delivered fewer than `d` times may be delivered again.
- **Binding:** a lossy channel's `lose` is bound by the author, never automatically.

Refusals:

| Text | Layer |
|---|---|
| `a channel holds at most <n> messages; a channel holds at least one` | runtime |
| `a channel delivers a message <d> more times; it cannot deliver fewer than once` | runtime |
| `an Inbox association requires a rooted state var Inbox field`; `an Inbox association requires this Machine's owning state var` | compile |
| `<x> holds no channel natively: declare the inbox as a var` | runtime |
| `channel <id>: messages are Int with no declared bound: write channel[UpTo[N]] or a named opaque range` | export |
| `<m> holds the lossy channel <c> and binds no <c>Loss step: ...` | export |

---

## 20. Faults, crashes and durability

```scala
object fault extends Actor:
  val crash = action.crash

object Model extends Machine[Outcome, Nothing], FailureModel:
  durability(inMemory(origin)(...), durable(received), durable(network))
```

| Form | Meaning |
|---|---|
| `object fault extends Actor` with `.crash`, `.responseLoss`, `.storageLoss` | A fault action: its actor is named `fault`, and it has exactly one kind |
| `durability(fields*)` | Classifies every var, in the Machine body |
| `durable(v)` | Kept across a crash |
| `ephemeral(v, resetTo = x)` | Reset to `x` |
| `inMemory(v)(k -> fallback, ...)` | Listed values replaced; fallbacks must be durable values |
| `crashes(fault.a, outcome, facts*)` | In `transitions`: a generated crash that applies the durability classification in every state |
| `on(fault.a): ...` | An authored crash: still needs a full `durability(...)` |

Refusals (runtime unless marked):

| Text | Trigger |
|---|---|
| `Machine <m> crash <a> at <loc> requires a durability classification` | a crash binding with no `durability` |
| `Machine <m> durability classification <f> at <loc> requires a crash binding` | `durability` with no crash |
| `crashes(<a>) at <loc>: duplicate crash binding` | two crash bindings |
| `crashes at <loc> requires an action without inputs`; `... requires a crash fault` | wrong action |
| `durability field <f> at <loc>: unclassified state field` | a var left out |
| `... reset value outside the field domain`; `duplicate in-memory key`; `in-memory fallback must be durable`; `duplicate or overlapping classification` | durability |
| `a state field names a state var of its Machine` | compile: a non-var argument |

Bind a native fault budget with `fault.a.budgetedBy(remaining)`, using an owning Boolean, Int or
bounded Int var or rooted immutable record field. The target works in native `on` blocks, including
record/scalar input blocks, and in `crashes`. Foreign, computed and wrongly typed fields refuse;
two budgets on one binding refuse. Authored effects control their own budget writes. A generated
crash disables before reset at zero/false and refuses a negative Int; otherwise it consumes once,
rebuilding nested reset fields and its own budget atomically per root var. Other budgets remain
unchanged. Selected reset validity evidence is retained.

Storage-loss assumptions are generated privately per bound fault, shared by declaration copies
and derivations. There is no public authored `Assumption` or fairness-copy path.

---

## 21. Realizations

A realization binds a Machine's classes to commands against a real server.

Status: no current-form realization exists outside framework tests.

- **The Temporal kit:** every Temporal realization and the kit (`Realizes`, `Controller`, `await`,
  ...) are `.disabled`.
- **The generic `Realization`:** usable only with empty `roles`, `requiredSettings`, `behavior` and
  `serverSteps`, and it selects no Queries (`runs` is empty).
- **Pending:** the design in [runs-in-tests](../design/runs-in-tests.md) (fn-195) replaces Query
  selection with runs declared in test files.

### Declaring

| Form | Meaning |
|---|---|
| `Realization(machine, producer, producerVersion, roles, correlation, scripts, ...)` | A generic realization of one machine |
| `ConstructedRealization` | The trait a kit implements; `toIR` converts it |
| `RejectionCode(rejection, grpcCode)` | Maps a shared rejection to a gRPC code |
| `Correlation(...)` | How evidence is keyed, and the check window |

Generic-conversion refusals (runtime): `<decl>: roles require their owning kit's constructed
conversion`, and likewise for settings, behavior and server steps.

### Scripts and commands

| Form | Meaning |
|---|---|
| `script(id, activation)(items*)` | One ordered script |
| `everyCase(command)` | A command every Case carries |
| `onPath(classes*)(command)` | A command for Cases whose path takes one of the classes |
| `perform(class -> command, ...)` | Where path steps of those classes land, in path order |
| `command(instruction, after, timeout, regardless, closes)` | A command with options, named after its `val` |
| `aliasOf(target)(instruction)` | Takes `target`'s command name |
| `rpc(role, method)(assign)` | A unary call, typed by the protobuf method |
| `readUntil(evidence, role, until, interval)(assign)` | Polls an evidence read until `until` holds: the one wait form |
| `.withFields`, `.extended` | The same call with extra fields; a new command extending it |
| `statusTable(fact -> value, ...)` | A fact-to-value lookup |

`Performance(step, command).choosing(choices*)` binds a class to named choice alternatives.

Refusals:

| Text | Layer |
|---|---|
| `onPath requires at least one class`; `perform requires at least one class` | runtime |
| `performance of <id> at <loc> is modelOnly: <reason>` | runtime |
| `... choosing requires at least one choice`; `... choosing requires distinct choice tokens` | runtime |
| `a command is named after the val that declares it: declare it as a val and refer to it by value` | plugin |

### Evidence and conditions

| Form | Meaning |
|---|---|
| `Evidence.read(...)`, `Evidence.keyed(...)`, `Evidence.runEvent(...)` | Evidence read from a response, a kit record, or the Run's own record |
| `Condition.present`, `equal`, `greater`, `not`, `all` | Poll `until` and run-event guards |
| `Field[Root, V](_.path)` | A captured protobuf field path |
| `Elapsed.milliseconds(n)` | A duration |
| `Learned(id, kind)`, `Observed[M](id)`, `Actuator(id, kind, role)` | A learned value, a recorded message, a control |

### Run expectations

Import `framework.realize.*` for these authored expectations. They state what a Run should
support; constructing one establishes no observed Run or server verdict.

| Field | Path `satisfied` | Sample `conformant` | Property-only `inconclusive(reason, because)` |
| --- | --- | --- | --- |
| conformance | conformant | conformant | conformant |
| property | satisfied | inconclusive | inconclusive |
| contract | satisfied | satisfied | satisfied |
| disposition | completed | completed | completed |
| cleanup | succeeded | succeeded | succeeded |
| reason | None | Some(neverEvaluated) | Some(supplied reason) |
| conformanceReason | None | None | None |
| monitors | empty | empty | empty |
| because | empty | `A sampled path selects no individual Property` | supplied nonblank text |

The Property-only `inconclusive` shorthand accepts `Reason.explanationsDisagree`,
`Reason.neverEvaluated` and `Reason.unreadable`. Other reasons require a full
`RunExpectation(conformance, property, contract, disposition, cleanup, reason,
conformanceReason, monitors, because)` with independently chosen fields. Unsupported shorthand
reasons throw `IllegalArgumentException` directing the author to that full form; empty or
whitespace-only shorthand explanations throw `IllegalArgumentException` naming `because`.
Nonblank explanations retain the supplied text, including surrounding whitespace.

The framework's shared conformance validation boundary checks full and retained/copied
expectations as well as convenience outputs. The shared `reason` is absent exactly when both
Property and Contract are satisfied. In that pair `because` is empty; in each of the other eight
pairs it is nonblank after trimming. `conformanceReason` is absent exactly when conformance is
conformant. Each `MonitorExpectation(name, outcome, reason, because)` follows its own outcome:
satisfied requires absent reason and empty text; violated or inconclusive requires a reason and
nonblank text. Invalid metadata throws `IllegalArgumentException` identifying the failing field.
Disposition, cleanup and all other independently supplied fields are preserved.

Raw `RunExpectation` and `MonitorExpectation` construction retains its existing type restrictions:
satisfied assessments refuse explanation text, constructors are private, and `copy` permits only
`because = ""`. Raw non-satisfied values can still omit text or retain blank or copied-empty text;
the conformance boundary refuses those values when validated. Monitor names, declaration
references, order and duplicate entries remain intact here. Actual conformance binding must
resolve their identities, owners and duplicates separately.

### Binding a Query

| Form | Meaning | Refusal |
|---|---|---|
| `query.live(RunExpectation(...))` | Marks a live Case and its expected Run | runtime `<name> runs on the model only: its scenario restricts or assumes the environment` when the scenario says `only`, `without` or `fair` |
| `RunExpectation(conformance, property, contract, disposition, cleanup, reason, conformanceReason, monitors, because)` | The independent expected Run fields described above | export `<q>'s live expectation is short of satisfied and says no reason why: state it, \`because = "..."\`` |
| `action.modelOnly(because)` | A fault no realization performs | runtime, at `perform` |

---

## 22. The IR boundary

### What is exported

The exporter (`export/`) lifts roots: machine and composition objects, `queries` and `capabilities`
sections, and `Query`, `Witness`, `Progress` and `Property` values, plus constructed realizations.
It writes one `generated/ir.json` per feature package. It reads the record form; the plain-form
sections (`transitions`/`effects` with vars, `invariants`, `postconditions`, `progress` with
`whenever`/`leadsTo`) are checked natively and not exported yet ([Known gaps](#known-gaps)).

Root refusals (export): `root <r>: names no declaration of the lifted sources`;
`<r> is a <T>; a root is a machine, a composition, a Property, a Query, a list of Queries, a progress claim, a claim bundle, a realization, or a machine's \`properties\`, \`capabilities\` or \`queries\` section`.

### Expressions the reader evaluates

The IR holds functions as expression trees
(`proto/internal/temporal/server/api/umpire/v1/expression.proto`):

| Kind | Members |
|---|---|
| expressions | literal, var, field, call, construct, copy, unary, binary, if, match, let, list, lambda, hole, inbox |
| unary | `OP_NOT`, `OP_NEG` |
| binary | `OP_EQ`, `OP_NE` (structural), `OP_AND`, `OP_OR` (short-circuit), `OP_LT`, `OP_LE`, `OP_GT`, `OP_GE`, `OP_ADD`, `OP_SUB`, `OP_CONCAT`, `OP_CONTAINS` |
| inbox | `OP_SEND`, `OP_IS_EMPTY`, `OP_IS_FULL` |
| patterns | wildcard, bind, literal, case, alternatives |

A hole propagates; a branch not taken is not reached.

### Liftable subset

A shared vocabulary declares `object vocabulary extends Lifted` once.
The marker covers its member definitions and members of nested containers, stopping at local
method scopes. `Machine` supplies the marker itself, so an ordinary `object effects:` needs no
additional declaration. Local defs remain part of their enclosing body's ordinary lifting.
There is no per-definition lifting annotation.

The plugin stores one `liftedIr` body carrier for a covered member, a definition returning
`List[Step[S, O, F]]`, or an override of the Machine endpoint contract. Same-file captures prefer
the tree. A reached helper in another file without stored IR is refused at its capture:
`<fullName> is reached from another file: define it in an object that extends Lifted`.
That refusal also applies in a full compile and to a helper compiled without the plugin.
Realization-mode sample compilation remains the native checkpoint: plain Machine var/Unit body
lifting and the original six-sample full-mode equivalence proofs remain held for fn-190.9.

The author lint and the lifter accept the library operations below.
A step, guard or predicate is a pure expression in these rows. Other Scala tree
forms and library operations are refused at the author's line, naming the construct and this
table. Declaration computation is a different context: a library operation in a declaration
does not thereby become legal in its captured function. The table records the currently supported
source export, including the temporary declaration readers that constructed-value export retires.

| Source tree or operation | Context and IR lowering |
| --- | --- |
| Boolean literal | Body: Boolean `Literal` |
| Integer literal | Body: integer `Literal` |
| String literal | Body: text `Literal` |
| Parameter or local reference | Body: `Var`, or its bound argument |
| Field selection | Body: `Field` |
| Source function application | Body: `Call` of the indexed function |
| Case-class or enum constructor | Body: `Construct`, fields in declaration order |
| Typed/type-applied/named/inlined wrappers | Body: underlying expression; type and argument binding retained |
| Local immutable `val` block | Body: ordered `Let` |
| Structural lambda | Callback: `Function`; expression: `Lambda` |
| Empty draft block | `effect` capture only: unchanged state with the declared ok outcome; Unit is not a body value |
| `if` | Body: `If`, only the selected branch is read |
| `match` and literal, case, bind, wildcard, alternative patterns | Body: `Match` and matching IR patterns; enum wildcard restrictions still apply |
| `List(...)` / `Nil` / repeated arguments | Body: `List` in written order |
| `Some(...)` / `None` | Body: synthetic finite Option enum |
| `Step(...)` | Step body only: core step `Construct` |
| `==` | Body: structural `OP_EQ` |
| `!=` | Body: structural `OP_NE` |
| `&&` | Body: short-circuit `OP_AND` |
| `||` | Body: short-circuit `OP_OR` |
| `!` | Body: `OP_NOT` |
| `<` | Body: integer `OP_LT` |
| `<=` | Body: integer `OP_LE` |
| `>` | Body: integer `OP_GT` |
| `>=` | Body: integer `OP_GE` |
| `+` | Body: integer `OP_ADD` |
| `-` | Body: integer `OP_SUB` |
| `contains` | Body, Scala sequence receiver: list membership `OP_CONTAINS`; String and Java collection receivers are refused |
| `copy` | Body, case-class receiver: `Copy`; default getters preserve unchanged fields |
| `min` | Body, integer receiver: `If` with `OP_LE` |
| `max` | Body, integer receiver: `If` with `OP_GE` |
| `++` | Body, Scala sequence receiver: `OP_CONCAT`; unnamed step branching remains refused |
| `+:` | Body, Scala sequence receiver: prepend a literal item or `OP_CONCAT` |
| `toList` | Body, Scala sequence receiver: identity of the already lifted list |
| `forall` | Capability's bound fact-family parameter only: ordered conjunction of the recorded-fact guards; no dynamic collection lowering |
| `map` | Declaration sequence reader: substitute each written item into a structural mapper; no body lowering |
| `toVector` | Declaration sequence reader: underlying written items; no body lowering |
| `find` | Query declaration: `FORM_FIND`; collection `find` has no body lowering |
| `require` | Function contract: precondition |
| `ensuring` | Function contract: body; postcondition is not an IR assertion |
| Builder `name`/`when`, optional/sequence `isEmpty` | Declaration precondition only: read already-bound core metadata, then check the definition's `require`; no Option body lowering |
| String interpolation | Constant values only: text `Literal` |
| Channel `empty` | Channel literal |
| Channel `send` | `Inbox` send |
| Channel `isEmpty` | `Inbox` empty test |
| Channel `isFull` | `Inbox` full test |
| Hole `reached` | Body: declared `Hole` read |
| Role test through a marked definition | Body: membership in the enum's role cases |
| `choose` | Core choice naming |
| `because` | Core explanation |
| `UpTo` | Bounded integer |
| `through` (retired from the DSL; lifter arm until fn-190.12) | Projected source `Call` |

`var`, assignment, loops, `return`, exceptions and comprehensions inside these bodies are outside
the subset. A declaration may compute its declarations; the functions it hands them remain held
to this table.

---

## 23. Lints

`make lint-model` runs these; the layout rules apply to Temporal features (packages under
`features`, `foundations` or `actors`), not to `samples`.

| Rule | Text (abridged) |
|---|---|
| declaration layout | `R4.declaration-multiline: <name> starts its body on the definition line: put the body on the next line` |
| effect result | `R4.effect-return-type: <name> declares a return type: let its Unit result be inferred` |
| block form | `block-form rule: use a colon block for \`<name>\`, not optional braces`; the `in` scoping rule |
| comments | `//` comments only |
| sugar | public sugar definitions document their `Core form:`; core definitions do not depend on sugar |
| conventions | `R4.dotted-in`, `R4.enum-wildcard`, `R9.phase-import` |
| markers | negative control and failure model rules ([4](#markers)) |
| levels | A feature with refinement has two levels, each in its own folder; a feature without refinement has one level and no level subfolder |
| level names | A Product machine is named `<Feature>Product` and refines nothing |
| files | one machine, composition or derivation per file, named after it; `object exports sits in <path>: a feature's IR file is collected from its package, so nothing names its roots` |
| facets | A facet's file must be named after the facet |

Level folders are `product`, `protocol` (formerly `system`) and `realization`; form folders are
`workflow`, `standalone` and `mini`.

---

## Retired forms

These authoring paths are removed; the right column is what to write instead. Private metadata
containers retained for existing consumers do not restore the old record-state authoring engine.

| Retired | Write | Retired by |
|---|---|---|
| `RecordMachine[S, O, F]` with a `State` case class, `val init`, `def end(s)` | `Machine[O, F]` with `var`s, `override def end =` | fn-190 R1, R2 |
| `Phased`, the phase projection, `Phasing` | a phase var; role traits on its enum | R5 |
| `object rules extends Rules`, `on(a) { when(g) ~> e }`, `where`, `always`, `~>`, `enters`, `rejects`, `enter` | `object transitions extends Transitions` with `if`/`match` | R12 |
| `effect:` blocks with `View`/`Draft`, `is:`, `stay(s)`, `reject(o, s)`, step-list `choose` | `object effects` assigning vars; `reject(o)`; `choose(t -> {...})` | R3, R12 |
| `object properties` with `safety`/`liveness`, `property.once/never/stays`, `holds`, `holdsAcross`, `expect`, `across`, `invariant(...)`, `transition(...)` | `object invariants` / `object postconditions` with `check`, `after`, `before` | R11 |
| `Machine.leadsTo("name")(from, to, within, assumptions*)`, `Progress`, `assume`, `Assumption`, `restrict` for progress | `whenever(p): eventually(q)` in `progress`, with `in only/without/fair(...)` | R13 |
| `query`, `query(name)`, `limits`, `Limits`, `total`, `capabilities.bound` | `verify`/`find` in `queries`; the budget is the checker's | R14 |
| `action(this)`, `internal`, `action("name", Actor("x"))` | `action` in an `Actor`, `step` in an `Internal` | R16 |
| `.schema[M]` | the realization's binding of the class | — |
| `through(_.member, read)`, `records(_.member -> M, fact)`, `<field>_<fact>` keys | a refinement map; member fact cases | R17 |
| `Composition[S]` with a state case class and selectors, `Syncs`, `own`, `synced`, `DerivedComposition` | `Composition[F]` with member vals, sync vals, action references and named `withMember` | fn-193 |
| `restrict`, `assuming` derivations | a scenario's `only`/`without`/`fair` | R13 |
| `@sugar` and the lifter's expansion engine | `inline` sugar definitions | — |
| `irFile(...)`, `object exports` | IR files collected by package | — |

---

## Known gaps

Where the code and the spec (fn-190) or this reference's intent differ today:

- **Export.** Plain-form Machines are not lifted to the IR (fn-190.9 onward); their claims, progress
  and transitions are native-only.
- **Refusal texts.** The `in` refusal on a block claim and on `verify`/`find` does not list
  `without(...)`; on a progress claim it does.
- **Parametrised claims.** R11 allows a `def` claim "when parametrised"; the checker skips it and
  `verify` refuses it.
- **Spec refusals not implemented.** `<v> is state: it is assigned in an effect only` and `<v> is
  state: read it in an effect, guard, expectation, monitor or refinement`.
- **Outcome and fact types** are type parameters, `Machine[O, F]`, not the abstract type members the
  spec prototyped; `UpTo` is opaque, not an erasing alias.
- **Budget configuration.** R14's per-feature gate budget does not exist; checks take
  `defaultBudget` or an explicit `budget`.
- **Section order.** No lint holds the plain-form order; the structure and order rules still
  list the old sections (`states`, `rules`, `properties`, ...).
- **Capabilities** cannot be authored in the plain form (no public way to write a kind's Properties;
  the Limits-based bound form is removed).
- **Vacuity.** A claim triggered by an outcome (`if outcome != accepted then ...`) is silently
  vacuous when no step answers that outcome.
- **`starts(x = v)`** collapses the other vars' `oneOf` starts to their first values.
- **Collection operations beyond the builtins.** The samples call `forall`, `exists`, `map`,
  `filter`, `sum`, `values`, `toSet`, `head` and `nonEmpty`, none of which fn-192's initial builtin
  table holds ([Collection operations](#collection-operations)).

---

Native choice selection retains immutable tokens per action/class without calling guards or
effects. Compile native author and effect-library declarations with `-Yretain-trees`; the pinned
native settings include it. Static transitive and parameterized effect bodies can contribute tokens
across separate builds. Missing or dynamic inventories refuse explicit selection rather than
silently claiming no choices. A native rebind replaces the affected inventory; plain effects accept
no foreign or nonbranch tokens. Step-list and record-callback choice overloads are removed.

## Native checker

Import `framework.checker.Checker` explicitly alongside the DSL's declarations. `Checker.explore`
returns a `Result`; `Checker.verifyAll` accepts it and returns a `Report` with claim results and
warnings. `Checker.monitor` and `Checker.progress` return the same `Result` shape. Its `Verdict` is
`verified`, `refuted`, `found`, `notFound` or `unknown`, with the caller's budget and completion flag.
Budgets below one are refused. An incomplete search answers unknown unless a counterexample was
already found.

A result's `Path` retains its initial state and ordered `PathStep` values: action, source state,
destination state, outcome and facts. This is the path the checker took. Progress refutations keep
deadlock, cycle and deadline explanations in the result's detail. The DSL holds declarations and
callable references; the product checker evaluates them.

### Captured native bodies

`RunBody.capture(model): ...` retains a finite authored sequence for one exact `Machine` instance.
`RunBody.capture(model, from = start): ...` supplies an explicit start snapshot; otherwise execution
requires the Model to have exactly one initial state. A missing `from` with several starts is
refused at the captured body's position.

| Form inside capture | Meaning |
|---|---|
| `actor.send(field = value)` | One named-input action call; inputs read the restored state at the actual replay step |
| `actor.tick` | One stable, fieldless whole action |
| `oneOf(step1, step2, ...)` | One step retaining at least two finite alternatives |
| `repeat(count)(step)` | One action call or fieldless whole action repeated a positive number of times |
| `expect: ...` | Bare Boolean assertion statements, including literals, retained in source order |

Capture and declaration loading require no checker and evaluate no action-input or expectation
callbacks. Expectations run when replay reaches them, against the owning Model's restored state.
The first false assertion stops later assertions and retains its original file, line, text and
concrete state; Boolean operands retain ordinary Scala short-circuit behavior. Empty or non-Boolean
expectations and unsupported sequence statements refuse at compile time. This `expect` belongs to
captured bodies; the retired `properties.expect` engine remains retired.

Capture supplies the exact-instance context: `before` has the Model's typed vars, `outcome` its
outcome type, and `records(fact)` accepts only its admitted fact type. These read the actual preceding
step, including a rejected step with unchanged state. Foreign owner-bound state accesses refuse,
even for another instance of the same class; compatible shared fact values remain legal. Before
any step, state-only expectations read the start, `records` sees no facts, and `before` or `outcome`
refuses rather than inventing a preceding step.

`framework.checker.SuiteExecution.run(body, budget)` executes the captured body through native
replay and returns the existing `Result`, retaining replay evidence and caller budget cuts. Execution
requires the checker and a budget of at least one. Compile declarations before authored tests, or
consume a separately compiled declaration library; joint compilation of macro definitions and
authored callers as main sources is unsupported.

### ModelSuite

`class ModelTests extends ModelSuite(instance)` binds an ordinary MUnit suite to one exact
`Machine` instance. Import it through `framework.dsl.*`. Shared helpers belong at the suite top
level; tests belong inside these optional objects, registered in this order:

| Section | Test body |
|---|---|
| `object examples` | The captured native-body grammar above, replayed against the bound instance |
| `object explorations` | Reusable `explore` space declarations; exploration tests currently refuse at compile time |
| `object conformance` | The same captured native-body grammar and native replay as examples |
| `object unit` | An ordinary MUnit body, with MUnit assertions, return values and asynchronous behavior |

Write `test("name"): ...` or `test(munit.TestOptions("name")): ...` in a section. Both forms keep
MUnit filtering, ignore/only options, transforms and setup ordering. For an example or conformance
body, `test("name", from = start): ...` supplies the optional start snapshot; without it the
instance must have exactly one initial state. Action calls, `oneOf`, `repeat`, `expect`, `before`,
`outcome` and `records` retain the captured-body rules and exact-instance checks above. Unit bodies
are not captured as replay requests.

Registration happens once, at the first `munitTests` enumeration, after concrete eager helpers
initialize and before MUnit snapshots the test list or runs setup. Unrelated lazy helpers stay
lazy. Empty or absent sections and unit-only suites are supported. Tests outside sections,
cross-section early registration and arbitrary non-test section members refuse with a source
location; explorations also accepts immutable space vals. Names must yield a
nonempty slug, unique across all sections and both registration forms. Registered names have
`examples:`, `conformance:` or `unit:` prefixes. Recursive enumeration during loading refuses;
a failed section initializer remains failed on subsequent enumeration.

Use the framework source gate for source admission. Before suite selection it rejects erased inline
section members on every discovered runnable `ModelSuite` and its indexed source ancestors,
including suites with no authored tests. Plain MUnit enumeration cannot inspect erased declarations.

Every suite also registers `claims: <fully qualified suite name>`, including suites with only
examples or unit tests. It checks the bound instance's ordinary claims, retaining native results,
counterexample paths, completion flags and vacuity warnings in the diagnostics. Refuted results
fail the MUnit test. Unknown or cut checks also fail, explicitly labelled `inconclusive native
check`; they are never skipped or labelled verified. A counterexample found before an unrelated
cut retains its native refutation.

Native execution resolves the caller's validated budget policy lazily, once for the exact fully
qualified suite name, and shares that immutable selection between automatic claims and replay.
Neither a Model field, a horizon nor a sample count replaces this budget. Construction,
registration and DSL-only loading do not read environment policy or load the checker. Executing
a native test requires the product checker on the class path and fails explicitly if it is absent.

Conformance bodies here establish native replay evidence only. Typed conformance metadata (C),
realization/capability binding and Case production remain pending; the section name does not
establish any of them.

### Exploration space declarations

Inside a ModelSuite's `explorations`, declare a reusable bounded space:

```scala
val sends = explore(steps = 4):
  model.client.send
```

Each listed selector constrains that position, starting at the first step. A whole action searches
its existing finite input catalog. A named call can use `any` independently at each input while
retaining the other named values; omitted inputs keep their declared defaults. A standalone `any`
and positions after the listed selectors leave the search unconstrained. The positive `steps`
bound includes every listed position. Catalog order and whole-record correlations are preserved.
`any` is available only inside this capture, with a `SpaceContext`; examples and conformance
authored bodies cannot use it. Fixed-input authored runs retain their ordinary rules.

An `Explore` declaration retains its exact Model instance without executing effects, guards,
claims or input expressions. Its first native acquisition evaluates the named constraints once,
in written order, and retains those immutable values across reuse. Constraints are not reevaluated
at each reached state. Another instance's roots, actions, classes or starts are refused.

`space.find(steps = n): expect: ...` retains every positioned condition in a `SpaceFind` carrier;
`space.sample(count, seed)` retains a `SpaceSample` with the space's horizon, count and seed.
These are declaration signatures for the pending question adapter. They do not yet register or
execute ModelSuite questions, provide conformance binding or produce Cases.

The realize expectation forms and shared metadata validator are available independently.
Connecting that validator to actual retained-value admission remains part of typed conformance
integration. The proposed authored-path default is `satisfied`; the sampled default is `conformant`,
whose singular Property is inconclusive because no individual Property is selected. These
defaults do not register or admit conformance declarations.

## Public surface not in this reference

Every public top-level definition of `dsl/` and `realize/` (and each public extension method) is
either written somewhere above as code or listed here, with why it is public. Automated coverage
checks both directions: a new public definition needs a place in this reference, and a name listed
here must still exist. Making a listed definition
`private[framework]` removes it from this list.

### Types behind current forms

Named in signatures and error messages; an author writes the form, not the type.

`ActionDecl` `RecordAction` `ScalarAction` `ClassExample` `FaultKind` `InputBinding` `InputField` `InputFields`
`Choice` `Alternative` `Subject` `Scenario` `ScenarioDecl` `QueryForm` `ClaimQuery` `Monitor`
`CapabilityOf` `FieldDurability` `Accepting`
`Refusing` `Recording` `StepTarget` `StepRecords`
`ClaimAlternatives` `InitialAlternatives` `Branching` `ClaimBranch` `Declares`
`DerivedFrom` `Rule` `RuleGroup` `Rebinding` `RuleBook` `MemberStep` `CompositionTarget` `MemberReplacement` `Sync`
`Family`

Captured-body signature carriers retain source positions, ordered declarations and typed context.
Their constructors are internal except `RunPosition`, which is source metadata; authors use the
capture forms above rather than constructing an execution request from these pieces.

`RunAssertion` `RunPosition` `RunOperation` `RunRecords` `RunAlternatives` `RunAssertions`
`RunStep` `RunTarget` `RunContext`

### Retained metadata containers

Existing metadata consumers still use these containers. Their constructors and bounds are internal;
they do not provide the removed record-state authoring vocabulary. Broader Query/Witness retirement
is separate work.

`PropertyDecl` `PropertyBuilder` `SameStepProperty` `TransitionProperty` `ScenarioBuilder`
`Checker` `WitnessBuilder` `WitnessFacts` `witnessFacts` `Recorded`
`StepBinding` `BudgetedAction` `Exploration` `Variation`

### Checker execution inputs and evidence

Checker consumers pass a `Space` with an explicit finite step bound and positional action selection.
A `ReplayScript` orders `ReplayOperation` values, using `ReplayStep` for finite action alternatives
and `ReplayExpectation` for a positioned `Expectation` with its source line, text and predicate.

`ExpectationFailure` retains the failing position, line, text and state. `SampleEvidence` retains
ordered walks, seed, failing walk index, step bound and finite experiment completion. Successful
finite walks establish no exhaustive safety or unbounded progress conclusion.

The gate validates a `BudgetPolicy` against discovered suite identities. Its default and overrides
resolve into an immutable `Execution` boundary that passes the selected budget into every question.
ModelSuite supplies registration and discovery under each concrete suite's exact fully qualified
name; its native checks share that suite's immutable execution selection.

### Realization building blocks

The records, enums and scopes behind the forms of [Realizations](#21-realizations), for kits.

`Setting` `Behavior` `SystemStep` `Addressee` `Activation` `Script` `Item` `PerformanceBinding`
`After` `Command` `Instruction` `Operand` `Name` `Cardinality` `Target` `Assignment` `ResponseRead`
`Proto` `ProtoField` `ProtoEntry` `ProtoValue` `LearnedKind` `AttemptOf` `EventKind` `Commitment`
`FieldRole` `EvidenceField` `Taking` `ControlKind` `Conformance` `PropertyOutcome` `Disposition`
`Cleanup` `Reason` `MonitorExpectation` `RequestScope` `StatusTable` `RequestField` `ProtoScope`
`LiteralField` `ProtoValueOf` `FieldScope` `ResponseReadLine` `TypedProto` `TypedProtoField`
`TypedProtoValue` `TypedProtoEntry` `RecordedRef` `EvidenceRef` `TypedEvidence` `KeyedRef`
`RunEventRef` `TypedOperand` `ProjectedOrigin` `ProjectedPath` `RunKey` `TypedAssignment`
`TypedResponseRead` `TypedEvidenceField` `ConditionOperator`

### Framework plumbing

Public for the plugin, the exporter or the checker; no author use.

`FacetSource` `FiniteSchema` `IrFile` `IrRoot` `liftedIr` `Reify` `NativeInput` `RuleInput`
`Vars` `VarSchema` `RuleTable` `ClaimWords` `CompositionSchema` `CompositionFact` `IllustrationKind` `Illustration` `Reference`
`Named`

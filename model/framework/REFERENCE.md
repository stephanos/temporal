# Umpire DSL reference

A lookup of the DSL's API: each form, where it may be written, what it means, and what refuses it.
[SEMANTICS.md](SEMANTICS.md) walks through the same language as a guide; this file lists it. Paths
are relative to `model/framework`. Entries cite the file that defines a form, not a line: the DSL
moves faster than line numbers.

## Scope and status

- **Current form.** A Model is a plain-form `Machine`: state as `var`s, `object transitions extends
  Transitions`, `object effects`, and the claim sections (fn-190). The samples in `samples/` are the
  canonical examples; every example here is current unless marked otherwise.
- **Checked natively.** Today the plain form is checked by the Scala checker
  (`dsl/NativeCheck.scala`). Lifting a plain-form Machine to the IR is phase 2 of fn-190 (fn-190.9
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
| Layout | In `framework/samples` and `temporal/features`, scalafmt keeps the author's line breaks (`newlines.source = keep`, `model/.scalafmt.conf`), so a long `if` may break after `then` | `make fmt-model` |

Section order inside a Machine, as the samples write it: vars, `refinement`, `transitions`,
`effects`, `invariants`, `postconditions`, `monitors`, `progress`, `queries`. A named scenario is a
Machine-level `val` just before the section that uses it. No lint holds the plain-form order yet
([Known gaps](#known-gaps)).

---

## 2. Domains and data

Defined in `dsl/Domain.scala` and `dsl/DomainChoices.scala`.

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
| `contains` | yes | `List` (`OP_CONTAINS`) | `list`, `set`, `map` | NativeInputs, ReliableBroadcast, TwoPhaseCommit, Voting |
| `isEmpty` | yes | — | `list`, `set`, `map`, `option` | Bank, ReliableBroadcast, Voting |
| `size` | yes | — | `list`, `set`, `map` | ReliableBroadcast |
| `incl`, `excl` | yes | — | `set` | NativeInputs, TwoPhaseCommit, Voting (`incl`) |
| `get`, `getOrElse` | yes | — | `map`; `option` (`getOrElse`) | Bank, ReliableBroadcast |
| `updated`, `removed` | yes | — | `map` | Bank, ReliableBroadcast, TwoPhaseCommit, Voting |
| `++` | yes | `List` (`OP_CONCAT`) | `list.concat` | — |
| `+:` | yes | `List` | `list.prepend` | — |
| `isDefined` | yes | — | `option` | — |
| `min`, `max` (on `Int`) | yes | yes | `int` | — |
| `toList` | yes | `List` (identity) | — | ReliableBroadcast |
| `forall`, `exists` | yes | `forall` only for a capability's fact family | — | Bank, ReliableBroadcast, TwoPhaseCommit, Voting |
| `map`, `filter`, `sum`, `values`, `toSet`, `head`, `nonEmpty` | yes | — | — | Bank, ReliableBroadcast, NativeInputs, Voting |

The last two rows are used by the samples, but no body lifts them today and fn-192's initial table
does not hold them: once plain-form Models are lifted, a sample that keeps them is refused unless
the table grows or the sample is rewritten ([Known gaps](#known-gaps)).

---

## 3. The signature

Defined in `dsl/Action.scala`, `dsl/ActionInputs.scala`, `dsl/InputField.scala`,
`dsl/InputSource.scala` and `dsl/Syntax.scala`.

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

Example (`samples/Kettle.scala`):

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

Example (`samples/Bank.scala`):

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

Role traits mix into enum cases as metadata and add no state (`dsl/Roles.scala`): `Live`, `Waiting`,
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

Defined in `dsl/Machine.scala`, `dsl/Vars.scala` and `dsl/Markers.scala`.

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

Lint refusals (`lint/Markers.scala`):
`<n> binds the fault <a> and is marked neither a FailureModel ... nor a NegativeControl ...: mix in the one it is`;
`<n> is a failure model and binds no fault: bind an action of the actor \`fault\` that some state enables`;
`<n> is marked a failure model and a negative control: it is one`;
`<n> is a negative control that nothing the run checks can refute: ...`.

---

## 5. Transitions

Defined in `dsl/Syntax.scala` (`Transitions`), `dsl/TransitionMacros.scala` and
`dsl/RuleTable.scala`.

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

Defined in `dsl/Syntax.scala`, `dsl/Vars.scala` and `dsl/Machine.scala` (`choice`, `choose`).

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

Defined in `dsl/Syntax.scala` (`ClaimMacros`), `dsl/ClaimWords.scala`, `dsl/NativeClaims.scala`
and `dsl/Claims.scala`.

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

Defined in `dsl/Machine.scala` and `dsl/Monitor.scala`.

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

Defined in `dsl/Assume.scala` and `dsl/Syntax.scala` (`QueryMacros`); checked by
`NativeCheck.leading`.

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

Defined in `dsl/Claims.scala` (`Scenario`, `ScenarioDecl`) and `dsl/Syntax.scala`.

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

Defined in `dsl/Claims.scala` (`Query`, `ClaimQuery`, `Witness`) and `dsl/Syntax.scala`.

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

Defined in `dsl/NativeCheck.scala`. Its entry points are `private[framework]`: tests in package
`framework.dsl` call them.

### What is checked by default

`NativeCheck.verifyAll(machine, starts = Seq.empty, budget = defaultBudget)`:

- **Starts:** `machine.initialStates`, unless `starts` is given.
- **Unscoped claims:** `invariants` and `postconditions` share one exploration of every action.
- **Scoped claims and progress:** each in its own environment.
- **Monitors:** each on the product with the machine.
- **Not checked:** `queries`.
- **Order of results:** unscoped claims (invariants, then postconditions), scoped claims, progress,
  monitors.

Other entry points: `query(q, subject)` checks one Query; `queries(machine)` checks every Query of
`queries`. Test helpers (`dsl/NativeControls.test.scala`): `assertRefuted(machine, claim)` and
`invariant(body)`, the negative-control idiom.

### Budget

`defaultBudget` is `1 << 20` states. A cut exploration answers `unknown`, never `verified`; a
counterexample found before the cut stands. Refusal: `a budget admits at least one state`.

### `Verdict`

| Verdict | Shown | Meaning |
|---|---|---|
| `verified` | `verified` | Complete exploration, no violation |
| `found` | `found` | A `find` witness |
| `refuted` | `refuted` | A counterexample; for `find`, a complete search found none |
| `unknown` | `unknown` | A cut or a hole, and nothing decisive |
| `deadlock`, `cycle`, `missedDeadline` | `deadlock`, `cycle`, `missed deadline` | Progress only ([9](#9-progress)) |

### `Result` and `Report`

`Result(subject, verdict, budget, depth, complete, detail = Vector.empty, longest = None)`.

- **`line`:**
  `<subject>: <verdict> (budget <budget> states, <depth> steps; exploration <complete|cut>)`, for
  example `model.boils: verified (budget 1048576 states, 82 steps; exploration complete)`. `budget`
  is the cap, `depth` the deepest breadth-first level reached.
- **`subject`:** `<machine>.<member>`.
- **`longest`:** set only for a verified progress claim.

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

Defined in `dsl/Assume.scala` (`Hole`) and `dsl/Gaps.scala`.

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

Defined in `dsl/Refine.scala`.

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

Defined in `dsl/Machine.scala` (`Derived`) and `dsl/DerivationBindings.scala`.

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

Defined in `dsl/Facet.scala`.

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
| `<F> is a facet in <file>: a facet's file is named after it, <F>.scala` | lint |

`FacetCapabilities(...)` lists the facets' capability sections (legacy).

---

## 17. Capabilities

Legacy. Defined in `dsl/Capabilities.scala`; read only by the exporter
(`export/Capabilities.scala`).

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
| `bound(limits, ...)` | Bounds the generated Queries |

A plain-form Model cannot write a capability kind today: its Properties are written with `property`,
which is retired, and `bound` needs `Limits`, which a Model no longer declares. The native checker
ignores capabilities. See [Known gaps](#known-gaps).

---

## 18. Compositions

Legacy. Defined in `dsl/Compose.scala`. No plain-form example exists, and no native check explores
a composition: `<subject> runs on <c>, which no native check explores`.

```scala
object Twins extends Composition[Pair](_.left -> Switch, _.right -> Unfelt):
  def end(s: Pair) = Switch.end(s.left)
  object syncs extends Syncs:
    sync(_.left -> hand.press, _.right -> hand.press)
```

| Form | Meaning |
|---|---|
| `Composition[S](_.member -> M, ...)` | Several machines as fields of one state `S` |
| `object syncs extends Syncs: sync(a, b)` | A pair of member actions that step together; `sync(name, a, b)` names it |
| `c.synced(_.m -> a)`, `c.own(_.m, a)` | A composed step selector, for `only`, `fair`, `actions` |
| `c.withMember(_.m -> M2)` | The composition with one member replaced |
| `replaces(_.m, opaque)` | A member that stands in for an opaque machine |
| `type Fact` / `enum Fact` | One fact case per member |

Refusals: compile `a member selector must name exactly one direct state field`; runtime
`<c>: duplicate member fields`, `<c>: no member <field>`; export
`<c> declares no end: a composition object declares \`def end(s: S) =
...\``, `<c> declares no syncs: ...`.

---

## 19. Channels and inboxes

Defined in `dsl/Channel.scala`.

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
| `inbox(v, ch)` | Associates the var `v` with the channel, in the Machine body |
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
| `an Inbox association names the state var that holds the Inbox` | compile |
| `<x> holds no channel natively: declare the inbox as a var` | runtime |
| `channel <id>: messages are Int with no declared bound: write channel[UpTo[N]] or a named opaque range` | export |
| `<m> holds the lossy channel <c> and binds no <c>Loss step: ...` | export |

---

## 20. Faults, crashes and durability

Defined in `dsl/Action.scala`, `dsl/Machine.scala`, `dsl/Durability.scala` and `dsl/Crash.scala`.

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

Legacy: fault budgets (`budgetedBy`) and selector forms of `durable`/`ephemeral`/`inMemory` work
only on the record form.

---

## 21. Realizations

A realization binds a Machine's classes to commands against a real server. Defined in
`realize/`.

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

### Scripts and commands (`realize/Scripts.scala`)

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

### Binding a Query

| Form | Meaning | Refusal |
|---|---|---|
| `query.live(RunExpectation(...))` | Marks a live Case and its expected Run | runtime `<name> runs on the model only: its scenario restricts or assumes the environment` when the scenario says `only`, `without` or `fair` |
| `RunExpectation(conformance, property, contract, disposition, cleanup, reason, because)` | The expected Run | export `<q>'s live expectation is short of satisfied and says no reason why: state it, \`because = "..."\`` |
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
| `through` (retired from the DSL; lifter arm until fn-190.12) | Projected source `Call` | `lifts/Members.scala` |

`var`, assignment, loops, `return`, exceptions and comprehensions inside these bodies are outside
the subset. A declaration may compute its declarations; the functions it hands them remain held
to this table. The author-time refusal controls are `framework.lint.LiftableSubsetFixtures`.

---

## 23. Lints

`make lint-model` runs these; the layout rules apply to Temporal features (packages under
`features`, `foundations` or `actors`), not to `samples`.

| Rule | Text (abridged) | File |
|---|---|---|
| declaration layout | `R4.declaration-multiline: <name> starts its body on the definition line: put the body on the next line` | `lint/DeclarationStyle.scala` |
| effect result | `R4.effect-return-type: <name> declares a return type: let its Unit result be inferred` | `lint/EffectDefinitions.scala` |
| block form | `block-form rule: use a colon block for \`<name>\`, not optional braces`; the `in` scoping rule | `check/lint/BlockFormRule.scala` |
| comments | `//` comments only | `check/lint/CommentRule.scala` |
| sugar | every public definition of a `Syntax.scala` documents its `Core form:`; no core file imports sugar | `check/lint/SyntaxRule.scala` |
| conventions | `R4.dotted-in`, `R4.enum-wildcard`, `R9.phase-import` | `lint/Conventions.scala` |
| markers | negative control and failure model rules ([4](#markers)) | `lint/Markers.scala` |
| levels | `<M> refines <P>, so <feature> has two levels, each in its folder: the Product in product/Product.scala, ...`; `<f> has no machine that refines another of its own, so it has one level, ...: it has no <sub>/ folder` | `lint/Structure.scala` |
| level names | `<X> is the Product machine in <f>'s product/Product.scala: name it <Feature>Product, ...`; `... is <f>'s Product machine and refines another machine: a Product refines nothing` | `lint/Structure.scala` |
| files | one machine, composition or derivation per file, named after it; `object exports sits in <path>: a feature's IR file is collected from its package, so nothing names its roots` | `lint/Structure.scala` |
| facets | `<F> is a facet in <file>: a facet's file is named after it, <F>.scala` | `lint/Structure.scala` |

Level folders are `product`, `protocol` (formerly `system`) and `realization`; form folders are
`workflow`, `standalone` and `mini`.

---

## Retired forms

Each was removed or made `private[framework]`; the right column is what to write instead.

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
- **Section order.** No lint holds the plain-form order; `lint/Structure.scala` and
  `lint/Order.scala` still list the old sections (`states`, `rules`, `properties`, ...).
- **Capabilities** cannot be authored in the plain form (no public way to write a kind's Properties;
  `bound` needs `Limits`).
- **Compositions** have no plain-form example and no native check.
- **Vacuity.** A claim triggered by an outcome (`if outcome != accepted then ...`) is silently
  vacuous when no step answers that outcome.
- **`starts(x = v)`** collapses the other vars' `oneOf` starts to their first values.
- **Collection operations beyond the builtins.** The samples call `forall`, `exists`, `map`,
  `filter`, `sum`, `values`, `toSet`, `head` and `nonEmpty`, none of which fn-192's initial builtin
  table holds ([Collection operations](#collection-operations)).

---

## Public surface not in this reference

Every public top-level definition of `dsl/` and `realize/` (and each public extension method) is
either written somewhere above as code or listed here, with why it is public. `ReferenceSurfaceTest`
(`check/lint/ReferenceSurface.test.scala`) checks both directions: a new public definition needs a
place in this file, and a name listed here must still exist. Making a listed definition
`private[framework]` removes it from this list.

### Types behind current forms

Named in signatures and error messages; an author writes the form, not the type.

`ActionDecl` `RecordAction` `ScalarAction` `ClassExample` `FaultKind` `InputBinding` `InputFields`
`Choice` `Alternative` `Subject` `QueryForm` `CapabilityOf` `FieldDurability` `Accepting`
`Refusing` `Recording` `Before` `StepTarget` `StepRecords` `RequirementFacts` `Alternatives`
`ClaimAlternatives` `InitialAlternatives` `Branching` `ClaimBranch` `Owner` `Declaring` `Declares`
`DerivedFrom` `Rule` `RuleGroup` `Rebinding` `RuleBook` `MemberSelection` `Composed` `Move`
`Composer` `DerivedComposition` `Family`

### Legacy

The record form's API, kept public until fn-190.16 deletes it; do not write it in a new Model.

`PropertyDecl` `PropertyBuilder` `SameStepProperty` `TransitionProperty` `ScenarioBuilder`
`QueryDecl` `QueryOn` `QueryIn` `Checker` `WitnessBuilder` `WitnessFacts` `witnessFacts`
`Expectation` `Expecting` `expectationBuilder` `ExpectedAlternatives` `ExpectedBranch` `Across`
`afterAction` `Rejects` `Recorded` `keeps` `Once` `Never` `Stays` `Case` `Firing` `GuardCases`
`Bindings` `StepBinding` `BudgetedAction` `BudgetedScalarAction` `Exploration` `Variation`

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
`VarSchema` `StateChoiceProgram` `InputChoiceProgram` `IllustrationKind` `Illustration` `Reference`
`Named`

# Framework authoring

`framework.dsl` declares machines and their promises; `framework.realize` supplies generic
realization values. The framework has no Temporal Model dependency. The wider model pipeline is
described in [model/README.md](../README.md); the native expectation loop below is narrower.

## Inputs

Declare an action's record with `.inputs[In]`. The action keeps that actual input type:

```scala
final case class Start(enabled: Boolean = false, retry: Boolean = true)
val start = action(Actor("caller")).inputs[Start].creates(Entity("job"))
val defaults = start()
val selected = start(retry = false, enabled = true)
val value = start(Start(true, false))
```

Named calls use Scala's checked case-class constructor, including parameter defaults and
argument evaluation order. Unknown or duplicate labels, wrong types and omitted required fields
fail compilation. A one-field record also accepts its field value; its label comes from that
declared field. Catalog order follows record field order, with the last field varying fastest.
Selected finite witnesses, including restricted or empty catalogs, remain the action's witnesses;
metadata construction does not enumerate them. Defaults never come from catalog order, and a
default outside the selected domain is refused when its class is requested.

Scalar actions retain `In = A`, including when `A` is itself a case class. Supply the exact input
label explicitly; the declaration returns an action and cannot be appended as an input ladder:

```scala
val finish = action(Actor("worker")).input[Boolean](name = "result", default = false)
val required = action(Actor("worker")).input[Boolean](name = "answer")
val supplied = finish(true)
val defaulted = finish()
```

Omitting `default` makes a scalar required: requesting its empty class throws an error naming the
input. An explicit `None` is a default value, distinct from having no default. Examples take actual
input values with `.example(value, text)`.

Value helpers can take `Action[In]` directly. Named/default helpers preserve the explicit shape
evidence, `RecordAction[In]` or `ScalarAction[In]`; erasing it to `Action[In]` deliberately refuses
named and empty native calls rather than guessing from whether `In` is a product:

```scala
def supplied[In](action: Action[In], in: In): Class = action(in)
def preset(action: RecordAction[Start]): Class = action(retry = false)
inline def defaults[In <: Product](inline action: RecordAction[In]): Class = action()
```

`RecordAction[In <: Product]` declares a case-class record; `ScalarAction[A]` declares one labeled
value. Neither exposes standalone input tokens, appended input declarations, assignment-based class
calls or positional field-arity overloads. Use one complete record for any number of fields.
Even a scalar `EmptyTuple` remains one labeled input: declaration and binding checks use the
declared shape. Reopening its inputs or binding a state-only direct function refuses at construction.

Native declaration modifiers preserve this evidence and selected witnesses. Class metadata copies
retain the supplied input carrier while their values and declaration input shape remain identical;
changing either reconstructs the target input. Native callbacks receive `(state, in)`, where `in`
is the declared record or scalar,
including scalar products and tuples. Rules, direct bindings, facets and derivations share that
adapter; catalog enumeration retains each actual selected input value. Input-reading whole-action
callbacks take the state and one input, never a separate parameter per record field. State-only
callbacks can ignore the input. Zero-input actions, class-specific rules and heterogeneous
`on(a, b)` blocks bind state-only callbacks.

Capture shared-input field evidence at a concrete record boundary and forward that evidence into
the structural helper:

```scala
def enabled[In <: { def enabled: Boolean }](in: In, field: InputField[In, Boolean]): Boolean =
  field.get(in)

val enabledField = start.field(_.enabled)
val result = enabled(Start(), enabledField)
```

`inputField[Start](_.enabled)` captures the same descriptor without an action. The descriptor
retains its exact owner, field type, label and compiled getter; a generic helper needs no reflection
or name lookup. A selector must directly read its own lambda parameter's case-class field.
Computed reads, another record's field and opaque selector functions refuse capture. An abstract
structural bound alone supplies no concrete selector evidence: capture the descriptor first, then
forward it. Only a declared `RecordAction` exposes `.field`; a product-shaped scalar remains scalar.

Product and tuple scalars remain one labeled value. Their shape does not grant record-named calls
or record field descriptors. Widening to `Action[In]` retains supplied-value calls, while empty and
named calls require the precise declaration evidence above; zero-input actions retain `action()`.

### Explicit input domains

Supply field domains at declaration time, before finite evidence is searched for. Different fields
of the same type can have independent domains; other fields use their selected `Finite` witness:

```scala
final case class Request(amount: Int = 0, ceiling: Int = 2, enabled: Boolean = false)
val submit = action(Actor("caller")).inputs[Request](
  inputField[Request](_.amount).in(domains.oneOf(2, 0)),
  inputField[Request](_.ceiling).in(domains.oneOf(1 to 2))
)
val required = action(Actor("caller")).input[Int](name = "amount", domain = domains.oneOf(2, 0))
```

No global `Finite[Int]` is needed. Enumeration follows record declaration order, with the last
field fastest. `domains.oneOf` preserves authored order and expands a `Range` in its iteration order,
including descending and exclusive ranges. Lists, sets and maps supplied as arguments remain whole
values. Empty and duplicate explicit domains refuse immediately. Domain installation evaluates no
record constructor or default getter; materializing an omitted default validates it against the
selected domain. An explicit scalar default is already a value and is validated at installation.

Use `.inputs[Request](selectedWholeRecordWitness)` to select an authoritative joint domain. It
retains the witness's correlations, order and actual objects; field metadata cannot add Cartesian
combinations. Supplying that witness together with field bindings refuses because no intersection
or ordering contract is defined. The ordinary `.inputs[Request]` path still retains its selected
whole-record witness, including lazy empty or throwing catalogs.

### Guards that read inputs

Whole-action rules can read the exact declared input in a binary guard:

```scala
on(submit):
  where((s, in) => in.amount <= in.ceiling && !s.seen.contains(in.amount)) ~>
    ((s: AcceptedState, in: Request) => List(Step(true, s.copy(seen = s.seen.incl(in.amount)))))
```

The guard and effect receive the same input object. State and input guards chained with `.where`
short circuit in written order, after exact class filtering. A false guard skips the effect.
Phase conditions, facets and derivations use that same dispatch. Zero-input, exact-class and
heterogeneous-target contexts retain state-only guards; unsupported binary guard shapes refuse
compilation. A scalar `EmptyTuple` still carries an input and supports a binary guard.

Finite-value `domains.oneOf(2, 0)` and expectation `oneOf: branch: ...` have distinct forms; malformed
expectation branches cannot become finite domains. [NativeInputs.scala](samples/NativeInputs.scala)
is a small native example. These callbacks prove native authoring and dispatch only. Supported
operation lifting, interpreted numeric parity and the six samples' reached-graph equivalence remain
held; successful Scala guards or `Data` evidence establish none of them.

## Native state

State uses `Data[S]` for structural comparison and `StateData[S]` for admission; neither
enumerates states. Enums and case classes can contain `Boolean`, `Int`, `UpTo`, `Option`,
`Set`, partial `Map`, `List` and `Inbox` with supported leaves:

```scala
final case class State(seen: Set[Int], pending: Map[Int, List[Int]])
val data = summon[Data[State]]
val admitted = summon[StateData[State]]
```

Remove a hand-written whole-state `Finite[S]` when migrating collection state. Keep selected
`Finite` witnesses for inputs, outcomes, facts and channel message actions, and retain exact
channel ownership on each `Inbox` field. `Data[String]` supports text comparison, but direct or
nested String state, functions and unsupported leaves are refused at native construction even
when a custom `Finite` exists. Native admission does not depend on running the exporter.
Recursive state types and `Inbox` inside a `List`, `Set` or `Map`, including inside a
record stored in a collection, are unsupported. Direct and nested record fields may hold an Inbox.
Text comparison orders UTF-8 bytes; a UTF-16 tie-break distinguishes ill-formed surrogate strings
that encode to identical replacement bytes. This native edge behavior establishes no IR transport parity.

Finite catalog order and structural type order are independent. With `Finite.of(2, 0)`, action
classes remain `[2, 0]`, while numeric comparison puts `0` before `2`. An unordered channel
therefore canonicalizes those messages to contents `[0, 2]`; FIFO keeps send order. This is an
explicit representation change from catalog-ordered unordered contents, with IR spelling, keys
and backend parity still unproved. Capacity, duplicate counts and exact channel ownership remain
constraints.

Constructing a machine or registering declared or derived rules does not enumerate states or run
guards and effects. This supplies no overlap-safety verdict. Native callbacks use ordinary Scala:
`m(k)` throws for a missing partial-map key; use `m.get(k)` or `m.getOrElse(k, default)` for safe
reads. A missing-key exception is not a native checking result or an interpreted hole.

Author numeric state with Scala `Int`. Direct callbacks have signed 32-bit bounds, wrapping
arithmetic and Scala/JVM division behavior: division by zero throws, while minimum divided by
`-1` wraps. Native boundary tests are smoke evidence only. The interpreted contract is checked
signed 64-bit arithmetic with overflow holes; fn-173 .4/.5 still owe endpoint, overflow,
division-by-zero and minimum-divided-by-`-1` proofs, plus preservation or explicit refusal of
host constant folding and initialized numeric values. Native `Data[Int]` closes neither R5 nor
cross-interpreter parity; full R5/R6 remain open.

The native `StepTables` producer and `writeStepTables` command, `MachineLaws` catalog walks and
`Refinement.unclosed` are retired. Typed `Refinement` declarations and owner discovery remain;
native tests assert actual binding order/identity and selected callback results without walking
machine state catalogs. Construction and metadata inspection run no guards, effects or refinement
maps. Explicit callback examples establish only their supplied inputs.

Original fn-173 .3/R3/R4 retains universal refinement totality and target membership, role
closedness, ordered repeatability, state/outcome/fact membership and declared-hole/unknown-row
handling. The old source-state tables and text seal are unavailable. Retained pure comparison,
rename-map and lift-comparison utilities supply no replacement seal; ChangeCheck/Makefile consumer
reconciliation and Go static/reached-graph proofs remain held. The private Query total bridge and
exporter carriers remain held integration work.

Bank uses sparse `Map[Addr, Map[Denom, Int]]` balances: absent denominations read zero;
zero writes remove the denomination and then an empty account. Its Send record retains
the finite amount domain 0..3 through a field binding, including accepted funded zero sends
and refused empty zero sends. TwoPhaseCommit uses an exact three-key manager map, a set of
received managers and a separate ever-sent message set; messages are never consumed as an Inbox.
Required manager lookups remain partial rather than defaulting a missing manager to working.

Voting keeps exact acceptor keys in its vote-set and max-ballot maps, with -1 for no
ballot and explicit 0..1 action inputs. Its invariant remains a Property; normal effects
no longer use it as an enabling condition, and still return authored refusal Steps.
Chosen sets retain both values on inconsistent supplied states, while decides tests
v1 membership. ReliableBroadcast keeps its direct Inbox and declares the exact
network field owner with `inbox(_.network, wire.network)`, inherited by Naive. Its
receive binding is a callback; interpreter-owned delivery removal is a separate step.

Kettle stores signed Celsius directly: old readings 0/60/140 map to -40/20/100.
Its heat<100 and failover>=100 guards remain; native Int adds no global bound.
Each Cache monitor stores its own immutable ordered List of original Event and observed
after-L1 Option[Int] payloads. Writes take priority, silent expiry appends nothing and
repeated observations remain. Read staleness stays sticky and successive writes must
strictly increase. History-to-old-Seen projection is many-to-one, so this representation
map supplies no monitor-product graph bijection; original .6 still owes that reconciliation.

The [retained six-sample inputs](samples/testdata/native-migration/README.md) keep complete
pre-rewrite source texts at one committed baseline, with explicit representation maps and
intended Query answers. Native sample tests exercise selected ordinary dispatch and Property
cases. The full sample lift-and-Go check, complete Query answers and R7 graph proof remain held.

## Native faults, budgets and choices

Keep the existing `fault` actor and attach one kind to each action. Declaration modifiers retain
the actor, action identity and record or scalar input shape:

```scala
object fault extends Actor:
  val lost = action(this).responseLoss
  val restart = action(this).crash.modelOnly(because = "no restart control")
  val storage = action(this).storageLoss
```

Completed native binding acquisition rejects a missing or repeated kind, metadata on another
actor, and an empty or repeated `modelOnly` reason. `Performance` also rejects a model-only action,
including direct construction and copies.

Attach a budget at its binding inside the owning machine, `fault.lost.budgetedBy(_.remaining)`.
Use that target in `on(...) { ... }` or `target ~> callback`. The selector captures a writable
case-class field path of that machine's state, with Boolean, plain Int or an existing bounded
Int carrier such as `UpTo[2]`. Computed, foreign and wrongly typed selectors refuse. Each binding
has its own budget, so two machines can bind the same fault to different fields. A second budget
on one binding refuses at completed acquisition. Restriction keeps its budget; rebinding an effect
keeps it unless the replacement declares another budget. Authored guards and effects decide
enabledness and budget writes. Declaring a budget evaluates no state, guard, effect or default
and adds no hidden counter. Generated crashes consume their own budget as described below.

When a native performance selects named alternatives, bind a deferred program that retains its
exact tokens and the callbacks that execute them:

```scala
val committed = choice
val ignored = choice
val alternatives = choose[State, Outcome, Fact](
  committed -> ((s: State) => effects.commit(s)),
  ignored -> ((_: State) => Nil)
)
// Inside the machine's rules:
on(fault.lost.budgetedBy(_.remaining)) {
  where(_.remaining > 0) ~> alternatives
}
// In the realization's script:
val selected = Performance(fault.lost, lossCommand).choosing(committed)
val item = Item(performs = Vector(selected))
```

The input-reading overload is `choose[S, In, O, F]`, with `(s: S, in: In)` callbacks. Both forms
require two distinct tokens, preserve arm order and retain Nil-producing arms as declarations.
Execution checks that each arm returns at most one step. Existing eager `choose(token -> steps,
...)` and ordinary callbacks keep their behavior.

An omitted `choosing` means all alternatives. Explicit subsets must be nonempty, distinct and
belong by token identity to truthful programs for the performed action and class in that machine.
Completed `Realization` construction checks membership without running callbacks. An opaque
callback cannot justify an explicit subset. A native kit supplies protected `declaredPerformances`
and `declaredDependencies` data. Its final `nativePerformances` and `dependencies` accessors validate
the initialized values before exposing them. Replacing a
program with an unrelated callback removes the old inventory. Native membership does not establish
witness lowering, manifest enforcement, global reached-state budget rules or IR transport.

### Durability and crashes

Classify every state field on a machine that binds a crash. A whole record field or all its nested
fields may be classified; duplicate and overlapping paths refuse. For example, inside a machine:

```scala
durability(
  durable(_.stored),
  ephemeral(_.pending, resetTo = false),
  inMemory(_.phase)(Phase.buffered -> Phase.idle),
  durable(_.remaining)
)
// Inside its Rules section:
crashes(fault.restart.budgetedBy(_.remaining), Outcome.internal, Fact.restarted)
```

`durable` preserves the field, `ephemeral` writes its explicit reset value, and `inMemory` maps
only the listed values while preserving the rest. Every mapped fallback must itself be durable;
duplicate keys, missing fields, unsupported reset evidence and out-of-domain values refuse with
the field and declaration location. Typed selectors and selected finite leaf evidence validate
reset values without enumerating whole states. Plain Int and structural collections need no state
catalog. An Inbox reset needs adequate selected channel evidence; comparison alone cannot validate it.

`crashes` gives one native step with the supplied outcome and ordered facts and an empty `because`.
Core `Bindings` can receive the same `crashes(...)` binding. A budgeted generated crash first reads
its own field: false or zero disables before any reset; a negative Int refuses before reset. An
available Boolean becomes false, or a positive Int decreases by one, in the same step as the reset.
Reset values and the own-budget change are computed from the original state; each affected nested
record and the root are reconstructed once with their final fields.
Every budget field must be classified durable, and other budgets are preserved. An active generated
crash's budget path must not alias another active fault binding's budget. Rebinding the crash
to an authored effect retains the classification and budget metadata but executes only that effect;
the author controls its consumption. An unbudgeted crash performs only the declared reset.

Completed rule-table, binding and realization acquisition validates dormant declarations before
exposing callbacks. Construction does not force uninitialized subclass sections or inspect starts,
guards or effects. Classifying a machine with no crash, or binding a crash without classification,
refuses. Restriction may retain the validated classification after removing the crash; it does not
permit a descendant to declare new classification without an active crash. Authored fault budgets on
crash-free copies retain their existing semantics.

`assumes(a, b)` in a machine and `.assuming(c, a)` retain authored assumptions as `a, b, c, a`,
including fairness duplicates and empty fairness segments. The public `assumptions` list appends one
generated entry per bound storage-loss fault, shared across machines and declaration copies. Only
those generated entries coalesce; unrelated same-spelled faults keep separate identities. An
authored assumption colliding with a generated entry refuses. Restriction removes the generated
entries for removed faults and preserves every authored occurrence. Native reset and assumption
behavior does not establish the held IR, Go, exporter or provider migration proofs.

## Native reach and query configuration

Machines and compositions expose `reach: Limits`. The immutable `frameworkReach` default has
`steps = 256`, `actions = 0` and `search = 65536`; override `reach` on an owner to select another
bound. Derived, built and replaced owners retain source reach unless explicitly overridden.
A Query reads reach from its Scenario's owner, including when its Property belongs to a refined
owner. Native consumption requires nonnegative steps/actions and positive search. Zero steps
admits only the initial state; actions is retained for Limits compatibility, while steps/search
govern reach. Ordinary Query Limits keep their existing validation rules.

Configure a completed `find`, `verify` or Witness declaration with typed checker metadata:

```scala
val selected = (query("completion") find completed in finished limits bounds)
  .open(because = "the selected region is intentionally bounded")
  .checkedBy(Checker.apalache)
  .confirm(Checker.tlc, Checker.umpire)
```

The default primary is `Checker.umpire`, with no confirmations or open acceptance. The other
identifiers are `Checker.apalache` and `Checker.tlc`; native authoring accepts all three regardless
of backend availability. `.open` requires a nonblank reason and retains its exact text. Open and
primary checker are each assigned once; repetition rejects even when the value is unchanged.
`.confirm` takes one nonempty, ordered, duplicate-free list excluding the effective primary and
may be called only once. It fixes that primary, including the default: select a nondefault primary
with `.checkedBy` before `.confirm`. Calling `.checkedBy` after `.confirm` rejects.

Public `reach`, `openReason`, `checker` and `confirmations` retain this configuration across
`.live`, `.explore` and Witness refinements such as `.ends`, `.limits`, `.recordsWhen` and
`.excludes`; Witness modifiers retain specialized fluent types. Constructing, copying or inspecting
these declarations evaluates no model callbacks or search. These values are metadata, not a checked
answer or evidence of backend execution, coverage admission, trace replay or reach adequacy.

Public Query/Witness `total` getters and writers are retired without aliases; remove authored
total clauses. Internal carrier readers and whole-table engine retirement remain held with fn-173
.2/.3; backend and replay proofs remain held with .7–.9.

## Expectations

Finish a Property with a colon block. Each `check` or `records` statement is a requirement;
requirements are conjoined in written order. These examples assume an existing machine `Subject`
with state `Job`, outcome `Outcome`, fact `Fact`, and action `worker.finish`.

```scala
val completed = Subject.property.after(worker.finish()).expect:
  records(Fact.completed)
  check(after.state.done)

val becameDone = Subject.property.afterAction("finish").across.expect:
  check(!before.done)
  check(after.state.done)
```

`after(classRef)` selects one exact action class, including its inputs. `afterAction(name)` selects
the whole action; compositions also accept a composed action reference. Omitting both selects
all steps. `.across` enables `before`, the previous state. `after` is always the resulting typed
`Step`; bare state accessors, where declared, read that step's state. A same-step block cannot read
`before`. A bare Step does not carry action identity: restrictions belong to the Property.

Explicit contextual names retain the authored bindings. They name contexts, not raw states or
steps: `Expectation[S, O, F].step` gives the Step and `Before[S].state` gives the previous state.

```scala
val named: Step[Job, Outcome, Fact] => Boolean = expect:
  (result: Expectation[Job, Outcome, Fact]) ?=>
    check(result.step.state.done)
    records(Fact.completed)

val changed: (Job, Step[Job, Outcome, Fact]) => Boolean = expect.across:
  (old: Before[Job], result: Expectation[Job, Outcome, Fact]) ?=>
    check(!old.state.done)
    check(result.step.state.done)

val reused = Subject.property.expect:
  check(named(after))

val reusedTransition = Subject.property.across.expect:
  check(changed(before, after))
```

Standalone `expect` values are ordinary Boolean functions and can be called directly or reused
through `check` as above. Public `holds`/`holdsAcross` finishers and compatibility aliases are
retired; direct core constructors belong to framework implementation and independent test oracles.
Source-inspected method wrappers retain accessible expectation dependencies. Predicate parameters
or local predicate values confined to a helper are refused when their ownership cannot be retained.

`records(fact)` requires a fact and returns Unit. `after.records(fact)` is a Boolean test;
`after.facts` is the recorded fact list. Use the Boolean form inside a condition or `check`,
for example `if after.records(Fact.completed) then check(after.state.done)`. Do not discard a
Boolean expression in a block: `check(condition)` states the requirement explicitly.

Alternatives require explicit branches; each branch is itself a conjunction:

```scala
val acceptable = Subject.property.expect:
  oneOf:
    branch:
      records(Fact.completed)
      check(after.state.done)
    branch:
      check(!after.state.done)
```

At least two nonempty branches are required. A failed requirement skips the rest of its block;
`oneOf` tries the next branch after failure and skips later branches after success. Nested
alternatives work the same way. An `if` without an `else` imposes no requirement when its condition
is false. Immutable helper vals evaluate where written. Reached holes and unrelated exceptions
propagate; they are never converted into false or retried as alternative branches.

Expectation blocks state requirements, rather than actions: assignment, step construction,
path/Query declarations, empty blocks and discarded Boolean expressions are refused, including
when hidden in locally bound expressions or inspected source helpers. Write optional DSL blocks
with colon indentation; retain braces only where Scala requires them or in deliberate refusal
fixtures.

## Composed facts

Select both the exact state field and its machine:

```scala
val memberCompleted: Step[Joined, String, String] => Boolean = expect:
  records(_.left -> LeftMachine, LeftFact.completed)

val selected = JoinedMachine.property.expect:
  check(memberCompleted(after))
```

Use a direct field selector rooted in the supplied state, such as `_.left`, rather than a computed
selector or a selector val. The fact type belongs to the selected machine. The field/machine pair
must match the declaring composition; constructor order does not replace field identity, and two
fields containing the same machine remain distinct. Reusable predicates retain this evidence
through `check(named(after))` and the transition equivalent. Wrong composition attachment is
rejected even when an invalid member read lies in an unreachable branch. Witness selectors follow
the same exact-member rule.

## Property illustrations

A completed `property.expect` retains `SameStepProperty[S, O, F]`; `property.across.expect`
retains `TransitionProperty[S, O, F]`. Both remain ordinary `Property[S]` values. Attach optional
illustrations after the requirement block:

```scala
val completed = Subject.property.after(worker.finish()).expect:
  records(Fact.completed)

val illustrated = completed.illustrate(
  "completion recorded", "The resulting step records completion", IllustrationKind.positive,
  worker.finish(), Step(Outcome.accepted, done, List(Fact.completed))
).illustrate(
  "completion absent", "This hypothetical step omits completion", IllustrationKind.counterexample,
  worker.finish(), Step(Outcome.accepted, done)
)
```

Transition illustrations additionally supply the before-state immediately before the `Step`.
The explicit kind has only `positive` and `counterexample` forms. Labels and explanations must be
nonblank; labels are unique within the accumulated Property. Attachments append immutably, keep
exact text and fact repetitions, and retain each call's file and one-based line/column in `origin`.
Queries retain the illustrated declaration, including when the Property was widened to `Property[S]`.
Existing pattern finishers and ordinary Properties need no illustrations or new evidence.

Attachment admits the supplied values using the initialized individual owner's exact action
binding and selected input/outcome/fact domains. Record input correlations and supplied objects are
preserved. It checks one state's supported structure and known bounds without reading whole-state,
nested-record or Inbox-content catalogs. Opaque state leaves need explicit validity evidence.
Plain integers and collections gain no invented bound. Refining and derived individual machines
remain their own Property owners; compositions are unsupported for illustrations.

For Inbox state, declare the exact field-to-channel association inside the machine:

```scala
inbox(_.incoming, wire)
inbox(_.storage.pending, acknowledgments)
inbox(_.optionalStorage.get.pending, optionalWire)
// For a root enum case with fields:
inboxCase[State.waiting, Message](_.pending, waitingWire)
```

These rooted structural selectors retain coordinates; optional access does not execute `.get`.
Associations retain their channel references and origin through state-preserving derivations.
Repeated/conflicting fields, duplicate channel storage, foreign or missing ownership refuse.
An absent optional field or inactive case supplies no Inbox to validate. A supplied Inbox must use
the exact channel, respect capacity/redelivery/message bounds and already have canonical unordered
delivery order; equal repeated entries are valid. Admission never sorts or repairs values.

An illustration may describe an unreachable state or an unselected action bound by that owner.
Attachment executes no Property predicate, selector, guard, effect, init/end condition or search.
It establishes neither satisfaction nor violation nor reachability: those classification and
transport checks remain held. A hypothetical counterexample is not a discovered model failure or
a replayable witness. Keep independent predicate tests and ordinary witness declarations for those
different claims.

## Native checks

Run from the repository root, serially, using the pinned tools and existing
`model/build/ir-scalapb.jar`. These roots exclude capture/export sources, Temporal sources and
`model/project.scala`:

```sh
mise exec -- scala-cli compile --server=false model/framework/dsl model/framework/realize model/framework/samples
mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests
mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'
```

Require nonzero executed tests and retain the output and exit status. Format touched Scala sources
with `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf <paths>`; repeat the formatter
and require unchanged bytes. The block-form lint must scan a nonzero real native corpus.

For separate-library public-client checks, use a new Scala CLI `--workspace` directory for each
complete recipe, across its native package, author package, runtime and refusal stages. Overwriting
a local dependency jar at the same path can reuse cached author macro expansions; `package -f`
alone does not establish fresh compilation. Preserve failed attempts and require the full recipe's
strict flags, intended diagnostics and terminal success.

Native results establish only the framework boundary. They do not establish capture/exporter
correctness, exact IR equivalence or Temporal caller migration. Those fn-164 .3/.7–.9 obligations
and all Temporal migration remain held. This loop does not run `make check-framework`, schema
generation, Go checks or regeneration. See the
[migration guidance](migrate/README.md) before the later system sweep.

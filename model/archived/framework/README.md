# Framework authoring

[VISION.svg](VISION.svg) (source [VISION.d2](VISION.d2), render with `d2 --layout tala VISION.d2 VISION.svg`) is the north star: the
state after the one-language series fn-199 through fn-209, where Scala declares, checks, lints, emits
and produces, and Go only executes Cases against a real system and observes.

`framework.dsl` declares machines and their promises; `framework.checker` executes those declarations
as the Scala oracle, and `framework.realize` supplies generic realization values. The framework has no Temporal Model dependency. The wider model pipeline is
described in [model/README.md](../README.md); the native expectation loop below is narrower.

On 2026-10-17 this reversed the 2026-10-13 Go-oracle direction: use the existing simple compiled
Scala checker directly instead of routing native questions through IR and Go. The IR remains the
export and lowering format. The [measurement](../design/scala-checker-measurement.md) keeps
immediate Go checker retirement at **no-go**: native refinement/replacement, rule-overlap,
recorded-Run/backend replay and existing Case consumers still need migration. Future Scala
production of the Testpilot Case proto is selected, not implemented. Testpilot's runtime, Driver
and canary remain in Go; ModelSuite wiring is still fn-195 work. No speedup follows from this choice.

## Parametric Models in tests

Give a Model the domains and constants its behavior needs through a class constructor.
`class Model[D: Finite](initial: D, limit: Int) extends Machine[Outcome, Nothing]` takes a finite
type domain and two value constants. A value constant has a liftable kind, a `Finite` type,
`Int`, `Boolean`, or recursively a `Set` or `Map` of those. The Model needs only the `Finite`
bound on a generic domain. Its schema and snapshot state come from its vars. Keep
`object Model extends Machine[Outcome, Nothing]` for a Model with no parameters.

The test owns the concrete world. Declare its enums there and explicitly construct an instance
in a stable val, supplying every constant. The [standalone Voting example](SEMANTICS.md#parameters)
uses the actual [Voting constructor](samples/Voting.scala#L41) with the acceptors, values and
pairwise quorums from [Voting's test world](samples/Voting.test.scala#L7). Voting takes
`values: Set[Value]` and `quorums: Set[Set[Acceptor]]`; both type parameters have `Finite` bounds.
No global world, implicit/default instance or suite-header API supplies those choices.

Run `Checker.explore(smallVoting)` and pass its result to `Checker.verifyAll` to check the
instance's promises. The first checked stable holder supplies its name; aliases and helper
checks retain it. Bind a fresh constructor or factory result to a val before checking it.
Two instances have independent vars, schemas, starts, selected input catalogs and reachable
graphs. Var initializers and `oneOf` starts may read the constructor constants. Empty enumerated
start domains refuse rather than inventing a start.

Runtime snapshots and discovery metadata follow their live owners. Discarded Models can be
collected after initialization and checking once no Model, State, Result, action or other legitimate
owner remains. Retained States, and Results retaining States, path evidence or an explored graph,
keep their exact owner and stable diagnostics; retained scanned actions keep their channel behavior.
No disposal or cache-clearing call is needed.
Classloader/static metadata caches are outside this instance-lifetime guarantee. The
[separate retention rerun](../design/scala-checker-measurement.md#after-the-instance-lifetime-repairs)
released all 64 constructed, initialized and checked Voting owners in each batch after one observed
collection; the original retention receipt remains historical evidence. This establishes reclamation,
not reduced live-graph memory or improved timing, and does not change the no-go decision.

Nested action input records retain the selected instance's exact generic types and defaults.
For example, `smallVoting.voter.voteFor(a = Acceptor.a2, b = 0, v = Value.v1)` constructs its
own `VoteFor` input. A native `Realization` also binds that actual instance, through its
`machine = smallVoting` constructor argument. Instance lifting, parameter values in the IR and Case
identity remain the IR phase. See [Parameters](SEMANTICS.md#parameters) for the constructor,
selected-state, holder and progress-bound refusal contracts.

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

Native declaration modifiers preserve this evidence and selected witnesses. Class copies retain the
supplied input object while their values and declaration shape remain identical; changing either
reconstructs the input. Native transition blocks receive the actual record or scalar as their one
parameter, including scalar products and tuples. Facets and derivations share that adapter.
Zero-input actions, exact classes and heterogeneous target blocks read vars without an input parameter.
A widened scalar `EmptyTuple` also uses the state-only block form; its class still requires that
one explicitly supplied value.

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

Whole-action blocks receive the declared input directly:

```scala
on(client.submit): in =>
  if in.amount <= in.ceiling && !seen.contains(in.amount) then effects.accept(in)
```

Scala conditions short-circuit in written order after exact class filtering. A false condition skips
the effect. Facets and native rebind/extend blocks use the same dispatch. Wrong input types, flattened
parameter lists and input-reading class-specific or heterogeneous blocks refuse compilation.

Finite-value `domains.oneOf(2, 0)` and a claim's `oneOf: branch: ...` have distinct forms.
[Inputs.scala](samples/Inputs.scala) is a native example. Successful Scala execution does not
establish operation lifting, interpreted numeric parity or reached-graph equivalence.

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
reads. A missing-key exception in a checked effect produces a refuted Result; it is not an
interpreted hole. Independent monitor/progress predicate errors propagate.

Author numeric state with Scala `Int`. Direct callbacks have signed 32-bit bounds, wrapping
arithmetic and Scala/JVM division behavior: division by zero throws, while minimum divided by
`-1` wraps. These are the native oracle's host semantics. The held interpreted contract is checked
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
the finite amount domain -1..supply+1 through a field binding, including accepted funded zero sends
and refused empty zero sends. Its holder, supply and address domain come from its test.
TwoPhaseCommit's test supplies its manager set; the Model uses exactly those keys in its manager map, a set of
received managers and a separate ever-sent message set; messages are never consumed as an Inbox.
Required manager lookups remain partial rather than defaulting a missing manager to working.

Voting keeps exact acceptor keys in its vote-set and max-ballot maps, with -1 for no
ballot and explicit 0..1 action inputs. Its native invariant is checked independently of
transition enabling conditions; refused effects use native rejection statements.
Chosen sets retain both values on inconsistent supplied states, while decides tests
v1 membership. ReliableBroadcast keeps its direct Inbox and declares the exact
network field owner with `inbox(network, wire.network)`, inherited by Naive. Its
receive binding uses a native input block; interpreter-owned delivery removal is a separate step.

Kettle stores signed Celsius directly: old readings 0/60/140 map to -40/20/100.
Its heat<100 and failover>=100 guards remain; native Int adds no global bound.
Each Cache monitor stores its own immutable ordered List of original Event and observed
after-L1 Option[Int] payloads. Writes take priority, silent expiry appends nothing and
repeated observations remain. Read staleness stays sticky and successive writes must
strictly increase. Projection onto the earlier observation summary is many-to-one, so this representation
map supplies no monitor-product graph bijection; original .6 still owes that reconciliation.

The [retained six-sample inputs](samples/testdata/native-migration/README.md) keep complete
pre-rewrite source texts at one committed baseline, with explicit representation maps and
intended Query answers. Native sample tests exercise selected ordinary dispatch and claim
cases. The full sample lift-and-Go check, IR Query-answer parity and R7 graph proof remain held;
the supported native questions below are available now.

## Native faults, budgets and choices

Keep the existing `fault` actor and attach one kind to each action. Declaration modifiers retain
the actor, action identity and record or scalar input shape:

```scala
object fault extends Actor:
  val lost = action.responseLoss
  val restart = action.crash.modelOnly(because = "no restart control")
  val storage = action.storageLoss
```

Completed native binding acquisition rejects a missing or repeated kind, metadata on another
actor, and an empty or repeated `modelOnly` reason. `Performance` also rejects a model-only action,
including direct construction and copies.

Attach a budget at its binding inside the owning machine, `fault.lost.budgetedBy(remaining)`.
Boolean, Int and bounded Int vars or rooted immutable record fields are supported. Computed,
foreign and wrongly typed selectors refuse. Each binding owns its budget; a second budget refuses.
Rebinding preserves the source budget unless the replacement declares another. Authored effects
control consumption; only a generated crash consumes automatically. Metadata acquisition runs no
guard, effect, state initializer or reset.

Declare choices as immutable tokens and write Unit blocks in an effect:

```scala
object effects:
  val committed = choice
  val ignored = choice
  def lose(): Unit =
    choose(committed -> { pending = false }, ignored -> { () })

object transitions extends Transitions:
  on(fault.lost.budgetedBy(remaining)):
    if remaining > 0 then effects.lose()

// In the realization's script:
val selected = Performance(fault.lost, lossCommand).choosing(effects.committed)
```

The checker enumerates branches in written order, restoring the original state before each run.
Tokens cannot name two branches of one choice. Omitted `choosing` means all alternatives. Explicit
subsets must be nonempty, distinct and match tokens retained for that exact action and class.
A plain effect cannot justify a foreign or nonbranch token. Native rebinds retain their replacement
inventory, not the source's.

Compile native declarations and separate effect libraries with `-Yretain-trees`; the pinned DSL and
checker settings include it. Acquisition follows available static effect bodies and immutable token
references without calling effects or guards, including transitive parameterized helpers. Missing,
recursive or dynamic inventories refuse explicit selection with a diagnostic, never an empty
invented inventory. Separately compiled Models retain that metadata for clients.

A native kit supplies protected `declaredPerformances` and `declaredDependencies` data. Its final
accessors validate completed values before exposing dependencies. These checks establish no witness
lowering, replay, manifest enforcement or IR transport.

### Durability and crashes

Classify every state field on a machine that binds a crash. A whole record field or all its nested
fields may be classified; duplicate and overlapping paths refuse. For example, inside a machine:

```scala
durability(
  durable(stored),
  ephemeral(pending, resetTo = false),
  inMemory(phase)(Phase.buffered -> Phase.idle),
  durable(remaining)
)
// Inside transitions:
crashes(fault.restart.budgetedBy(remaining), Outcome.internal, Fact.restarted)
```

`durable` preserves the field, `ephemeral` writes its explicit reset value, and `inMemory` maps
only the listed values while preserving the rest. Every mapped fallback must itself be durable;
duplicate keys, missing fields, unsupported reset evidence and out-of-domain values refuse with
the field and declaration location. Typed selectors and selected finite leaf evidence validate
reset values without enumerating whole states. Plain Int and structural collections need no state
catalog. An Inbox reset needs adequate selected channel evidence; comparison alone cannot validate it.

`crashes` gives one native step with the supplied outcome and ordered facts and an empty `because`.
A budgeted generated crash first reads
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
refuses. A crash-free descendant cannot introduce new classification. Storage-loss assumptions
are generated privately per bound fault and retain exact identity across modifier copies, repeated
bindings and derivations; unrelated same-spelled faults remain distinct. Authored `assume`,
`assuming` and action-filtering `restrict` are removed. These native checks establish none of the
held IR, Go, exporter or provider migration proofs.

## Native queries and configuration

Write `verify(Model.invariants.bounded)` or `find(Model.queries.completed)`, optionally `in` a
scenario. Query/Witness containers still retain checker, confirmation, open-acceptance, live and
exploration metadata; their broader retirement is separate work. Construction and copying execute
no model callbacks or search.

`query`, `Limits`, owner `reach` overrides and Query/Witness bounds mutators are removed.
Budgets belong to the checker caller. No renamed author configuration surface replaces them.
A completed declaration can still select `.checkedBy(Checker.apalache)`, ordered
`.confirm(Checker.tlc, Checker.umpire)`, or `.open(because = "...")`. Each is assigned once;
confirmation fixes the primary, and blank reasons, duplicates or a primary in its own confirmations
refuse. These are metadata, not evidence that another backend ran.

## Claims

Write state claims in `invariants` and step claims in `postconditions`, directly over vars:

```scala
object invariants:
  val countNonnegative = check(count >= 0)

object postconditions:
  val completed = after(worker.finish):
    records(Fact.completed)
    check(done)
  val becameDone = check(!before.done implies done)
```

Each check or fact requirement is conjoined in written order. A failed requirement skips the rest
of that member. `oneOf:` requires at least two explicit `branch:` blocks; a successful branch skips
later branches, and alternatives can nest. An `if` without an `else` imposes no requirement when
its condition is false. Helpers evaluate where written. Reached holes remain uncertain; unrelated
exceptions are not converted into false. `before` is available only for step claims and monitors,
`outcome` is the answer, and `recorded(fact)` reads the step's facts.

The old `property`, `expect`, `across`, `is` and `effect` authoring paths and their record contexts
are deleted. Internal Property containers remain for existing metadata consumers, not as an
alternative authoring API.

## Compositions

A composition names its members as vals. Its state is their var snapshots, so claims read a
member's vars and ordinary helpers directly. There is no authored state record or field selector.
`Composition[F]` takes the composition's fact type; use `Nothing` when its members record no facts.

For Machines `Subject` and `Other` that both bind `hand.start`:

```scala
enum JoinedFact:
  case left(value: Fact)
  case right(value: Foreign)

object Together extends Composition[JoinedFact]:
  val left = Subject
  val right = Other

  override def end =
    left.isClosed && right.phase == Phase.closed

  object syncs:
    val started =
      sync(left(hand.start), right(hand.start))

  object invariants:
    val rightClosedWithLeft =
      if left.isClosed then check(right.phase == Phase.closed)

  object postconditions:
    val startedTogether =
      after(syncs.started):
        check(before.left.phase == Phase.open)
        records(JoinedFact.left(Fact.completed))
```

If an action belongs to only one member, name it directly in `sync`; `left(action)` chooses the
member when the same action is bound by more than one. A sync takes both effects in one composed
step, combining their alternatives and wrapping each recorded fact in its member case. Its first
member supplies the outcome. An action paired by a sync does not also step alone; an unpaired
action steps only its member. Each composed step counts once toward a progress bound.

Claims, monitors, progress and scenarios use the same forms as on a Machine. Name a sync by its val
or an action directly in `after`, `only`, `without`, `fair` and a witness; member qualification
resolves ambiguity. `before.left.phase` reads the corresponding member's pre-step snapshot,
including through nested compositions. Exploration starts from the product of member starts and
uses the same on-demand checker and budget/cut rules. Without an override, `end` is true.

Replace a member with that Machine or a derivation holding its vars:

```scala
object LateTogether extends Composition(Together.withMember(right = LateOther)):
  object invariants:
    val rightStaysOpen =
      check(Together.right.phase == Phase.open)
```

The replacement keeps the source's member order, syncs, end and fact type. A misspelt member or an
unrelated replacement is refused. Each member must hold its own vars: the same Machine, or two
derivations sharing its vars, cannot fill two member slots. A member may itself be a composition.
Missing or mistyped fact cases and an unbound or ambiguous sync endpoint are refused.

These are native authoring and checking rules. Composition IR lifting and Go lowering remain
held for the final-DSL integration work; native success does not establish those proofs.

## Property illustrations

Retained internal `SameStepProperty` and `TransitionProperty` containers preserve ordered,
immutable illustration metadata for existing consumers. There is no replacement public illustration
builder on Unit-valued native claims in this task. Their later retirement or native authoring seam
is separate work.

Labels and explanations are nonblank, labels are unique, and kinds are explicitly `positive` or
`counterexample`. Exact text, repeated facts, call origin and hypothetical before-state are retained.
Those metadata checks do not decide the ordinary predicate.

Attachment admits the supplied values using the initialized individual owner's exact action
binding and selected input/outcome/fact domains. Record input correlations and supplied objects are
preserved. It checks one state's supported structure and known bounds without reading whole-state,
nested-record or Inbox-content catalogs. Opaque state leaves need explicit validity evidence.
Plain integers and collections gain no invented bound. Refining and derived individual machines
remain their own Property owners; compositions are unsupported for illustrations.

For Inbox state, declare the exact field-to-channel association inside the machine:

```scala
inbox(incoming, wire)
inbox(storage.pending, acknowledgments)
inbox(optionalStorage.get.pending, optionalWire)
inbox(phase match
  case waiting: Phase.waiting => waiting.pending
  case _ => throw IllegalStateException("inactive")
, waitingWire)
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

## Native elapsed-time values

Use ScalaPB `com.google.protobuf.duration.Duration` for `Command.timeout` and
`Instruction.TypedPoll.interval`. Both default to `None`; `command(..., timeout = ...)`,
`Instruction.readUntil(...)(..., interval = ...)` and `readUntil(..., interval)(...)` forward
the supplied option. Copies and naming retain it. `None` and `Some(Duration())` remain distinct;
native construction neither substitutes a default nor discards an explicit zero.

`Elapsed.milliseconds(value: Long)` constructs seconds and nanos exactly, including negative
values and both Long endpoints. For example, `Elapsed.milliseconds(-1234)` produces
`Duration(seconds = -1, nanos = -234000000)`. It does not round or saturate. A constructed
value is not proof that it fits protobuf Duration's range or an owning field's contract.

Native constructors also retain directly supplied Duration values without normalizing or
validating them. Lifting/admission must check the owning field's sign, range, whole-millisecond
precision and presence requirements, including whether zero is allowed and what omission means.
Those checks, defaults and refusals remain deferred integration proof; native construction
establishes no IR conversion, runtime polling or elapsed-coordinate monotonicity behavior.

The Duration controls in [RealizationConstructionTest](realize/Construction.test.scala) cover
exact signed millisecond construction, optional/default/explicit-zero retention, copies and
public helper forwarding, alongside the existing positive interval, timeout and 1234ms fixtures.

## Shared lifted helpers

Declare a vocabulary shared with captures in other files as a `Lifted` container:

```scala
object temperatures extends Lifted:
  def boiling(reading: Int): Boolean = reading >= 100
```

Member definitions in nested containers are covered too. `Machine` supplies this marker, so its
ordinary `object effects:` stays as written. Local defs belong to their enclosing body; a marker
does not make them independent stored functions. A step-list result type or an override of the
Machine endpoint contract also qualifies a member for storage. No per-def annotation is needed.

The plugin stores one `liftedIr` body carrier, also when several rules cover the same definition.
Same-file captures use the tree. An uncovered helper reached from another file is refused at the
capture with `<fullName> is reached from another file: define it in an object that extends Lifted`,
both in a full compile and incrementally. A marker on a library compiled without the plugin
supplies no stored body and gets the same refusal.

The native checkpoint's realization-mode sample compilation leaves plain Machine bodies to
fn-190.9. Marker coverage does not establish var/Unit-effect lowering or the six samples' original
full-mode IR and exporter-equivalence proofs.

## Native checks

Run from the repository root, serially, using the pinned tools and existing
`model/build/ir-scalapb.jar`. These roots exclude capture/export sources, Temporal sources and
`model/project.scala`:

```sh
mise exec -- scala-cli compile --server=false model/framework/checker model/framework/samples
mise exec -- scala-cli test --server=false model/framework/checker model/framework/samples --require-tests
UMPIRE_FRAMEWORK_BUILD="$PWD/.flow/tmp/framework-native" mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'
```

Require nonzero executed tests and retain the output and exit status. Format touched Scala sources
with `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf <paths>`; repeat the formatter
and require unchanged bytes. The block-form lint must scan a nonzero real native corpus.

The checker is a production module. The DSL retains callable references and schemas; its runtime
seam refuses execution when the checker is absent. Import `framework.checker.Checker` explicitly
alongside DSL imports. `Checker.explore(instance)` returns a `Result` with verdict, budget and
completion; pass it to `Checker.verifyAll` for a `Report` of claim results and vacuity warnings.
`Checker.monitor(instance, watch, starts, budget)` and `Checker.progress(instance, claim, starts,
budget)` return the same result shape. Refutations retain a concrete `path` with the initial state
and each action's source state, destination state, outcome and recorded facts. Budgets below one
are refused. A cut answers unknown unless the check already found a counterexample.

`Checker.replay(instance, script, budget)` takes a typed `ReplayScript`, including positioned
expectations. A disabled step or thrown effect refutes the run; a failed expectation retains its
line, text and state. `Checker.space(instance, steps)(select)` supplies a bounded schedule space.
`Checker.find(space, horizon, budget)(target)` returns `found` with a witness, `notFound` only
after completing the requested horizon, or `unknown` on a cut. The target is an `Expectation`,
not a raw predicate. A missing target or horizon outside `1..space.steps` refuses.
`Checker.sample(space, count, seed, budget)` runs reproducible seeded walks and checks their claims; passing sampled evidence
is not exhaustive verification. Counts below one refuse. `Checker.refuted(instance, claim)`
returns the ordinary claim result for a negative control; it does not invert the verdict.

Budgets are positive Ints, defaulting to `1 << 20`, and every Result records its budget.
Exploration counts reachable states; find counts state-and-position identities, replay counts
retained execution nodes, and sampling shares distinct-state admission across its walks.
They belong to the caller/gate, not the Model. The gate parses `UMPIRE_CHECKER_BUDGETS` as
newline-separated `default=budget` and exact `suite.name=budget` entries and passes the resolved
policy to its suite process. `Checker.forSuite(policy, exactSuiteName)` resolves an immutable
`Execution` whose questions use that budget explicitly. The future ModelSuite adapter will own
automatic suite-boundary resolution; the direct native-test route does not validate this policy.
Empty/malformed policies, nonpositive or non-Int budgets, duplicate keys and unknown suite
overrides refuse. A search horizon, sample count and authored progress deadline are separate bounds.
`Report` includes claim results and vacuity warnings for unfired actions and untriggered
`after(action)` claims. Witnesses or counterexamples found before a budget cut stand.

For separate-library public-client checks, use a new Scala CLI `--workspace` directory for each
complete recipe, across its native package, author package, runtime and refusal stages. Overwriting
a local dependency jar at the same path can reuse cached author macro expansions; `package -f`
alone does not establish fresh compilation. Preserve failed attempts and require the full recipe's
strict flags, intended diagnostics and terminal success. Compile native declarations and separate
effect libraries with `-Yretain-trees` so static native choice inventories remain available to
client macros; missing bodies refuse explicit selection rather than inventing an empty inventory.

Native results establish only the framework boundary. They do not establish capture/exporter
correctness, exact IR equivalence or Temporal caller migration. Those fn-164 .3/.7–.9 obligations
and all Temporal migration remain held. This loop does not run `make check-framework`, schema
generation, Go checks or regeneration. See the
[migration guidance](migrate/README.md) before the later system sweep.

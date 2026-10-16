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
retain the supplied input carrier while their values remain identical; changing class values drops
the old carrier. Native callbacks receive `(state, in)`, where `in` is the declared record or scalar,
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

`StepTables`, `MachineLaws` and `Refinement.unclosed` remain finite-only utilities pending fn-173
.3. Their explicit finite evidence is no fallback for native construction, and whole-state table
runs are outside this checkpoint's gate. Native explorer/checker replacement is also held.

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

Native results establish only the framework boundary. They do not establish capture/exporter
correctness, exact IR equivalence or Temporal caller migration. Those fn-164 .3/.7–.9 obligations
and all Temporal migration remain held. This loop does not run `make check-framework`, schema
generation, Go checks or regeneration. See the
[migration guidance](migrate/README.md) before the later system sweep.

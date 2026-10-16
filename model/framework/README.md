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

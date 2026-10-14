# Feedback on the Umpire DSL

A review of the Scala DSL in `model/framework`, as the Models under `model/temporal` use it and as
`model/SEMANTICS.md` describes it, at commit `63cd2f2b2c` plus the working tree (the `derives
Finite` removal). The subject is the language, not the guide: what an author has to write, what
the words mean, where two words mean one thing or one word means two, and where the shape of a
construct leads to wrong Models.

The planned Scala batch (`MILESTONES.md`, "Open: Scala DSL work", fn-159 through fn-141) is taken
as decided. A point a planned spec already settles is listed under **Settled** with the spec and
not argued again. The rest of each section is what remains after the batch, or where a planned
spec should go one step further, or where two planned specs pull against each other.

Severities:

- **High**: the form leads authors to a wrong Model, or blocks a planned spec, or has to be
  resolved before fn-141 can make sugar definitions the only lowering.
- **Medium**: a cost paid in every Model, or a name that will keep causing mistakes in review.
- **Low**: polish; worth doing when the construct is next touched.

Each point: the problem, what is planned, and one recommendation.

---

## 0. What the batch already settles

| Concern | Settled by |
| --- | --- |
| Bare action names in rules (`on(poll)` inside `from(worker)`) | fn-159.1: every `on` and `disabled` target is qualified |
| Bare phase names (`when(scheduled)`) and phase-member imports | fn-159.3 |
| Hand-written phase-only effects | fn-159.7: `enters(Phase.x)` |
| Capability kinds named as nouns (`Retries`, `Deadline`) | fn-159.8: `Retryable`, `Expirable` |
| Bounds scattered through Models | fn-159.5: every `Limits` in `Bounds` |
| `Observation.read` free text | fn-159.6 |
| One machine object per level with unstructured effects | fn-161: action-owning facets |
| Derived machines that change nothing and only hold claims | fn-162: Properties declared once, realizations name their Queries |
| Hand-listed IR roots (`exports`, `irFile`) | fn-160 |
| "System" as a level name | fn-160: the level is Protocol |
| Distributed-systems claims written by hand per Model | fn-163: system capabilities |
| Three declarations for one path-and-outcome claim; `.live` hidden | fn-140: witness Queries |
| Safety and liveness mixed in `properties` | fn-149 |
| Property bodies as one Boolean expression; `implies`; `when` reused for Properties | fn-164: `expect` blocks, `after(<class>)` |
| Faults as ordinary actions with hand-written crash effects | fn-123 |
| Sugar lowered twice (Scala body and lifter) and drifting | fn-141 Part A: the definition is the lowering |
| Lifter refusing declaration-level Scala | fn-141 Part B: declarations export by running |
| Same-type `==` without `CanEqual`, section order, wildcard cases | fn-156 |
| Elapsed-time fields as integers | fn-147 (Scala part) |

---

## 1. Types and domains

### 1.1 Catalogs are now implicit; the one place inference cannot follow is role-narrowed phases

**Severity:** Medium

**Problem.** With `derives Finite` gone, every enum and case class is a domain and `Finite[T]` is
derived where it is first needed. That removes 76 clauses and no Model changes shape. The one cost
is in equality: `CanEqual` is granted through `Finite[T]`, and when both operands are role-narrowed
phases (`case q: Held => q != Phase.paused`, or `val landing = if c then Phase.paused else
Phase.scheduled`), the compiler joins them to `Phase & Live`, for which no catalog can be derived.
A companion instance used to pin the type; a generic given cannot. The author writes `(q: Phase)`.

**Planned.** Nothing. fn-156 (equality) is done but assumed companion instances.

**Recommendation.** Accept the ascription for now; it is rare (one fixture, one test, no Model)
and the lifter unwraps it. The real fix is in 2.2: roles should not be traits mixed into the case
types, after which no value ever has type `Phase & Live` and the problem disappears with it.

### 1.2 `import framework.Finite.given` plus a dummy `summon` is a wart every file carries

**Severity:** Medium

**Problem.** Strict equality needs the `CanEqual` given from `Finite`'s companion, so every Model
file imports it, and a file that only uses it in `==` keeps the import alive with
`val _ = () => summon[CanEqual[Phase, Phase]]`. That is a workaround for an unused-import warning
(scala/scala3#25755) that has become a convention, repeated in every file that compares a shared
outcome.

**Planned.** fn-156.7 documents the enforcement; nothing removes the workaround.

**Recommendation.** Put the `CanEqual` given where the compiler finds it without an import: in the
companion of a marker every domain already has (`Phase` and `Fact` enums could extend a framework
`Domain` trait whose companion holds `given [T <: Domain]: CanEqual[T, T]`), or export the givens
from the `framework` package object so `import framework.*` suffices. Then delete the `summon`
idiom and its lint.

### 1.3 Two spellings of a bounded integer

**Severity:** Low

**Problem.** `attempts: UpTo[2]` is the current form. Channels and older Models bound an `Int`
field with a local `given Finite[Int] = Finite.upTo(n)` beside the state, and fn-131 notes the
consequence: the lifter reads one range per state type, not per field, so two bounded `Int`
fields in one record cannot differ.

**Planned.** fn-131 records the gap; fn-123 and fn-125 add new bounded values (budgets, settings).

**Recommendation.** Retire `Finite.upTo` as a field bound once channels take `UpTo[N]`, and make
the lifter read ranges per field. Keep `UpTo[N]`: it states the bound at the field, reads as an
`Int`, and lifts to the IR range with no new vocabulary. `0..2` is not a Scala type; the literal
union `0 | 1 | 2` is, but it enumerates rather than bounds and `n + 1` comes back as `Int`. Do
not rename it `Attempts`: it also counts presses, boils, heartbeats and deliveries.

---

## 2. Phases, roles and status facts

### 2.1 "Role" is the wrong word

**Severity:** Medium

**Problem.** A role is what a phase says about the entity independently of the entity: alive or
over, waiting or held. "Role" reads as a part someone plays, and the signature has actors, which is
where a reader expects roles. The `Closed` sub-roles (`Succeeded`, `Failed`, `Canceled`,
`Terminated`, `TimedOut`) are exactly Temporal's `WorkflowExecutionStatus` values.

**Planned.** Nothing.

**Recommendation.** Rename role to **status**. Keep **phase** for where the entity is in its
lifecycle and **state** for everything the machine carries; renaming phase to state would leave
the record without its standard state-machine name. The rename touches `framework/dsl/Roles.scala`,
`irgen/Roles.scala`, the IR's `roles` field and the Go validator; batch it with 2.3.

### 2.2 Roles are traits mixed into case types, which leaks into Scala typing

**Severity:** High

**Problem.** `case paused extends Phase(Fact.statusPaused), Suspended` gives the case the Scala type
`Phase & Suspended`. The DSL then offers four spellings of one test: `case _: Closed`,
`p.isInstanceOf[Closed]`, `p.in[Closed]` and `when[Closed]`, all lowering to case membership (the
Models use the last three; the fixtures use `isInstanceOf` under a scalafix exemption). The
typing has two costs. A role-typed binding compares only with an ascription (1.1). And role
subtyping (`Retrying extends Waiting extends Live`) is Scala subtyping, so whether `when[Waiting]`
includes `Retrying` is left to the reader's knowledge of the language.

**Planned.** fn-159.3 qualifies phase references; fn-141 lifts `case _: R` by one generic rule.

**Recommendation.** Declare a phase's status as data on the case, the way the status fact already
is: `case paused extends Phase(Fact.statusPaused, Suspended)`, with `Suspended` a value of a
framework enum rather than a trait. One test, `p.in[Suspended]` (or `p.is(Suspended)`), lowers
to membership; the compiler never sees an intersection type; the lifter reads the role from the
constructor argument as it reads the status fact today; and sub-statuses are declared as a value
list, not as trait inheritance.

### 2.3 Status facts and roles are three parallel lists of one mapping

**Severity:** Medium

**Problem.** Each phase case declares its role (`Held`) and its status fact
(`Fact.statusStarted`, through `enum Phase(val status: Fact) extends Recorded[Fact]`), and the
`Fact` enum lists one `statusX` case per phase. Neither is the other: `started` and
`cancelRequested` share the role `Held` with different status facts, and the Protocol's
`backingOff` reports `statusQueued`, the same fact as `queued`. Both are projections of what the
API reports for the phase, and that mapping is written per phase, per Model, in three lists that
nothing checks against each other.

**Planned.** fn-155 B uses the `Recorded` setter more; fn-159.7's `enters(<phase>)` records the
status fact; fn-117 already types the API in the Models.

**Recommendation.** Make the API status a kit enum with its generic status declared once per
entity kind (`enum ActivityStatus(val status: Status)`, in `model/temporal/foundations`), let a
phase case name only what it reports (`case started extends Phase(Started)`), and replace the
`statusX` fact cases by one `reported(status)` case. `enters`, `in`/`when`, `records` and the
realization's evidence then all read the one declaration. Fact keys change
(`statusCanceled` to `reported-Canceled`), so this is a batch item with a rename map, not an
fn-159 task. A status fact with a field (`statusTimedOut(which)`) becomes `reported(TimedOut)`
beside a fact of its own, or the phase takes the field.

---

## 3. The signature

### 3.1 `on` means three things, and the batch fixes one

**Severity:** Medium

**Problem.** `on(client.submit) { ... }` opens a rule; `action(this).on(job)` names the entity an
action targets; `Observation(on = job)` names the entity a read is of.

**Planned.** fn-159 qualifies the rule form's target; the other two keep the word.

**Recommendation.** Keep `on` for rules only. The entity target becomes `.of(job)` and the
observation parameter `of = job`.

### 3.2 `action(this)` is a self-reference an author writes on every action

**Severity:** Low

**Problem.** `val submit = action(this).on(job)` names the declaring actor by `this` inside the
actor object, so the lifter can attribute the action. The author writes it on every line and it
carries no information.

**Planned.** fn-141 R7: a declaration records its name and owner through a context parameter the
framework supplies.

**Recommendation.** Once fn-141's capture exists, `val submit = action.on(job)` (or
`action(job)`) takes its actor from the enclosing object. Retire the `this` form in the same step.

### 3.3 Three spellings of a class

**Severity:** Low

**Problem.** A class of an action is written positionally (`finish(Result.ok)`), by name
(`finish(result := Result.ok)`) and empty (`start()`), and the positional and empty forms leave
other inputs at the domain's first value. The `:=` symbol is the third symbol an author learns
(`~>`, `->`, `:=`), and it exists only for this.

**Planned.** fn-141 R4 retires the hand-written lowering of named inputs.

**Recommendation.** Keep positional and named, drop the empty parenthesised form in favour of the
bare action name meaning "every class" and `a(first)` meaning the first class, so "first value of
every input" is never a silent default.

### 3.4 `given Ok[Outcome]` is an invisible default

**Severity:** Medium

**Problem.** A step that names no outcome takes it from a `given Ok[Outcome]` in scope, usually in
the outcome's companion, sometimes imported per file. The guide has to explain it as "the outcome
a step means when it says nothing". A reader of an effect cannot see which outcome a bare
`enter(s)` produces without finding the given.

**Planned.** Nothing.

**Recommendation.** Make the default outcome a declaration of the machine (`val accepted =
Outcome.accepted` or `Machine[S, O, F](ok = Outcome.accepted)`), read by the sugar, so it is one
line in the machine header rather than an implicit resolution.

---

## 4. Effects

### 4.1 `enter`/`enters` and `reject`/`rejects` differ by one letter and by kind

**Severity:** High

**Problem.** `enter(state, facts*)` is a step value written in an effect; `enters(Phase.p)` is a
rule form after `~>` that also records the phase's status fact. `reject(outcome, s)` is a step
value; `rejects(rejection)` is a rule form over shared rejections. The pairs differ in what they
are, their arity and what they record, and the names differ by an `s`.

**Planned.** fn-159.7 makes `enters` the common form for trivial effects, which makes the pair
more visible, not less.

**Recommendation.** Give the rule forms a different shape: `~> Phase.p` for "enter the phase and
record its status" (the phase alone is unambiguous after `~>`), and `~> Rejection.notFound` for a
shared rejection. Then `enter`, `stay` and `reject` are the only step words and they appear only
in effects.

### 4.2 `disabled` is a value and a function, and fn-141 needs it to be one

**Severity:** High

**Problem.** `val disabled: List[Step]` (`Nil`) is returned from an effect to give no step, and
`def disabled(actions*)` binds actions a machine refuses. `Syntax.scala` defines both; the guide
documents them inconsistently. fn-141 R4 makes each sugar's definition its only lowering, which
needs one definition per word.

**Planned.** fn-159.4 removes one use (`disabled(process.stop)` in the activity product).

**Recommendation.** The effect form is `Nil`; call it `none` or `noStep`, or let an effect return
`Nil` directly. Keep `disabled(a)` for the rule form. Resolve this before fn-141 Part A.

### 4.3 `effect { }` and `is { }` blocks change meaning silently

**Severity:** High

**Problem.** fn-155's planning found that a block form "appends the assigned status after
recorded facts, cannot assign a computed phase, and carries no `because`". A `because` passed as a
parameter does not lift; one written directly in the effect does. The block looks like a shorter
spelling of the `def` form and is not: fact order in the IR differs, so the step's identity
differs.

**Planned.** fn-155 works within these limits and defers the lifter change.

**Recommendation.** A block form must lower to exactly what the equivalent `def` lowers to, or be
refused where it cannot. Fix the status-ordering difference (either both forms append the status
last, or both put it first) before fn-155 collapses effects through the block form, since the
equivalence seal would otherwise mask a fact-order change as a migration.

### 4.4 `require` inside a step is an error, not a disabled row, and nothing in the spelling says so

**Severity:** Medium

**Problem.** A `require(cond)` at the head of an effect is a precondition: a rule whose guard admits
a state the effect `require`s against is a declaration error at lift, not a silent disabled row.
Authors reading Scala expect `require` to throw at run time; the DSL reads it as a static claim.

**Planned.** Nothing.

**Recommendation.** Spell the precondition as DSL: `expects(cond)` or make guards the only place
a condition is written, and refuse `require` in function bodies (it is in the liftable subset only
for this).

### 4.5 Named choices carry names the reader never sees

**Severity:** Low

**Problem.** `choose(enqueueCommits -> steps, enqueueFails -> steps)` pairs each alternative with
a declared `choice` token, and the token's name is inert metadata: the IR strips it before
identity hashing (fn-131). Authors declare and name things that nothing reads.

**Planned.** fn-131 Part B marks inert fields once.

**Recommendation.** Make the names reach the decision trace and the lint's explanation, or drop
`choice` and take the names from the branch's `because`. Inert text the author must write is the
worst of both.

---

## 5. Rules

### 5.1 `when` means three things; fn-164 removes one

**Severity:** Medium

**Problem.** `when(Phase.x)` and `when[Role]` are rule headings; `property when <class> holds` is
a Property selector; `whenAction` is a third form.

**Planned.** fn-164 moves Properties to `after(<class>)` and `afterAction`.

**Recommendation.** After fn-164, delete the Property `when` outright rather than keeping it as a
deprecated spelling (fn-141 Part D retires unused spellings; put it on that list).

### 5.2 `from(actor) { }` becomes a header with no content

**Severity:** Medium

**Problem.** Today `from(worker) { import worker.*; on(poll) ... }` brings an actor's actions into
scope. After fn-159.1 every `on` target is qualified and `from` is "a grouping header with an
ownership check", so a reader sees `from(worker) { on(worker.poll) ... }`: the actor twice, the
second time doing the work.

**Planned.** fn-159.1 keeps `from` as a grouping; fn-161 regroups rules by facet, where the actor
grouping stops being the primary structure.

**Recommendation.** Drop `from` in fn-161's stage 2. The per-actor view of "who can do what" is
what the lint's table and fn-130's views are for, and fn-161's edge cases already say so.

### 5.3 Silence is expressed by absence

**Severity:** Medium

**Problem.** A system action with no row is "MUST NOT"; an actor action with no row is silence,
and only the `silent-rejection` lint finding tells an author the Model says nothing about a
request. Two different modalities are written the same way, by not writing anything. `disabled(a)`
exists for the explicit refusal but is optional.

**Planned.** fn-161's ownership check requires every bound action to belong to a facet, which
tightens "bound" but not "unbound".

**Recommendation.** Make the actor case explicit: a machine declares the actor actions it
`ignores(...)` (silence by design) and the lint refuses an actor action that is neither bound,
disabled nor ignored. Then the table's modality is always read off a declaration.

### 5.4 Guards read the state only, by convention

**Severity:** Low

**Problem.** `where(g)` takes a predicate of the state; an effect that must branch on an input does
so in its body. Nothing in the spelling says that a guard may not read inputs, and the
disjointness check runs over states only, so an input-dependent guard would be a silent
narrowing if it ever compiled.

**Planned.** Nothing.

**Recommendation.** Type the guard as `S => Boolean` in the heading's signature (it may already
be) and say so once in the guide. If class-level guards are wanted, give them a form,
`on(a) { where((s, i) => ...) }`, and extend the overlap check to classes.

---

## 6. Properties and claims

### 6.1 `never(to).from(before)` reads backwards

**Severity:** Medium

**Problem.** The core form is `!before(b) || !to(a)`: precondition first, prohibition second.
The sugar states the prohibition first. `once(over).keeps(x)` has the related problem: "once"
reads as "exactly once" where it means "from the point `over` holds".

**Planned.** fn-164 keeps the claim patterns as the temporal layer above `expect` blocks and
documents each with a core form.

**Recommendation.** Fix the order in fn-164, since it rewrites the Property surface anyway:
`from(before).never(to)` and `after(over).keeps(x)`. The `after` word then means the same thing
in `property after(<class>)` and in `after(over)`: "from this point on".

### 6.2 `Declares[S]` is the type of a Model and does not say so

**Severity:** Low

**Problem.** Shared Properties and capability laws are written over `Declares[S]`, the supertype of
a machine and a composition, with `def` rather than lambda parameters so the lifter can read them.
The word says nothing; the guide calls the thing a Model.

**Planned.** fn-164 keeps `Declares` for generic constructors.

**Recommendation.** Rename to `Model[S]`.

### 6.3 Progress claims live in `properties` with no distinguishing word

**Severity:** Low

**Problem.** `m.leadsTo(name)(from, to, within, under*)` is written in `object properties` beside
same-step and transition Properties; only its arity tells the reader it is liveness, and it takes a
string name where every other claim is named by its `val`.

**Planned.** fn-149 groups it under `properties.liveness`; fn-150 extends it to compositions.

**Recommendation.** In fn-149, drop the string name (the `val` names it, as fn-141 R7 captures) and
make the bound and assumptions named parameters (`within = eight, under = assume.fair(worker)`),
since fn-150 adds composed assumptions and positional arguments will not scale.

---

## 7. Scenarios, Queries and bounds

### 7.1 Bounds are named by their numbers

**Severity:** Medium

**Problem.** `Limits` values are `three`, `eight`, `twelve`, `fourWide`, `fiveWide`, `twelveWide`
in `temporal/Bounds.scala`. A Query says `limits eight`; a reader learns the number and not why
eight is enough for that Query, and two Queries with the same number for different reasons share a
name.

**Planned.** fn-159.5 moves every bound there; fn-140 derives a witness's bound from its path, so
only `verify` Queries keep an explicit bound.

**Recommendation.** Name a bound by what it covers (`oneRetryRound`, `closeAfterTwoAttempts`),
with its numbers in the declaration and a `because`. A Query then reads as a claim about a
situation, and a bound change is reviewed against its reason.

### 7.2 "witness" will mean the declaration and the result

**Severity:** Medium

**Problem.** A witness is already the path a `find` reports and the counterexample a failed
`verify` reports. fn-140 adds `witness(...)` as the one-sentence declaration of a path-and-outcome
Query. The same word then names what the author asks for and what the tool answers with.

**Planned.** fn-140 (in progress, 6/10 tasks done).

**Recommendation.** If the declaration keeps the word, call the result a *trace* in results and
reports, and say in the guide that a witness Query is answered by a trace. Renaming the result is
cheaper than renaming the declaration at this point.

### 7.3 `through` is a modifier only a `verify` may take

**Severity:** Low

**Problem.** `query verify P in s through refinement` reads a Property through a refinement map,
and only `verify` may. A `find ... through` is refused at the Query, which the reader discovers
by trying it.

**Planned.** Nothing.

**Recommendation.** Make it a property of the Property, not of the Query: a Property declared on
the Product that a Protocol machine `verify`s is read through the refinement by construction.
Then `through` disappears.

---

## 8. Monitors, assumptions and holes

### 8.1 A monitor is a curried triple

**Severity:** Low

**Problem.** `monitor(initial)(next)(violated)` is three argument lists with no names, with a
second overload `monitor(name, initial)(...)` that takes a string name; `sticky` and
`stickyAcross` are sugar over it; `readAfter` and `readAtEnds` say when it reads. The shape is a
fold, which is right, but a reader has to know the position of each list.

**Planned.** fn-141 R4 retires the hand-written lowering of `sticky`/`stickyAcross`; fn-149 places
safety monitors under `properties.safety`.

**Recommendation.** Named parameters: `monitor(start = ..., step = ..., violated = ...)`. Keep the
sugar.

### 8.2 `assume` takes a string

**Severity:** Low

**Problem.** `assume("storageLoss")` names an assumption by string, which the derivation
`assuming(...)` and fn-123's fault binding then repeat. Everything else is named by its `val`.

**Planned.** fn-123 derives the storage-loss assumption from the fault's name; fn-141 R7 captures
`val` names.

**Recommendation.** `val storageLoss = assume` after fn-141's capture; refuse the string form.

---

## 9. Refinement and derivations

### 9.1 Seven words for "the same, except"

**Severity:** Medium

**Problem.** `restrict`, `rebind`, `extend`, `refining`, `assuming` and `unmonitored` derive a
machine, wrapped in `Derived(...)`; `DerivedComposition` and `withMember` derive a composition;
`DerivesFrom(run, machine, runs = ...)` derives a realization. The marker traits `NegativeControl` and
`FailureModel` say why a derivation exists.

**Planned.** fn-162 removes the derivations that change nothing, which were most of them; the
behaviour-changing six stay.

**Recommendation.** One `Derived(source)` for all three kinds, with the change methods on the
source's type, and the `FailureModel`/`NegativeControl` markers as a required `because` on the
derivation rather than a mixin. Do this after fn-162, when the surviving uses are visible.

### 9.2 The refinement map is a method with four optional parts

**Severity:** Low

**Problem.** `object refinement extends Refinement(Product)` with `toProduct`, `visible`,
`visibleOutcomes` and `unobservable`, and the closedness check lives in the Model's tests rather
than in the gate.

**Planned.** Nothing.

**Recommendation.** Move the closedness check into the lifter's refusals (fn-141 R12 gives each
refusal one home), so a refinement that is not closed is refused where it is written.

---

## 10. Facets and capabilities

### 10.1 Today's facets and fn-161's facets are two constructs with one name

**Severity:** High

**Problem.** The framework has `trait X extends Facet[...], Phased[...]` with an inner object
holding `effects`, `rules`, `capabilities`, `properties` and `queries`, assembled by `Facets(...)`
and `FacetCapabilities(...)`, with the lifter switches `facets-only` and `facet-vocabulary`.
fn-161 defines a facet as a unit that *owns* actions, with the ownership check and cross-facet
dependencies as self-types. The two agree on the name and on little else; fn-161 R1 asks for "a
facet construct", which already exists in a different shape.

**Planned.** fn-161 (0/4 done, next in order after fn-159).

**Recommendation.** fn-161's first task should state which of today's facet forms survive, and
the assembly should be one word (`Facets(a, b, c)` for every section, not `Facets` for some and
`FacetCapabilities` for others). The lifter switches are configuration of a migration and should
go with it.

### 10.2 Capabilities are entity-level; fn-163 needs action-level

**Severity:** High

**Problem.** A capability is declared on a machine (`CapabilityOf`, `Capabilities` with `except`,
`overriding(... of = ...)`, `claim`, `bound`) and its laws read a `Phasing[S, P]`. fn-163 wants
`Idempotent(client.start, by = requestId)`, a promise about an action, and `Fenced`, a promise
over history. Neither fits the kind-per-entity mechanism, and the spec says so.

**Planned.** fn-163 is scheduled fifth; its R4 allows a law to "stay hand-written with a recorded
reason".

**Recommendation.** Design the action-targeted and history-scoped capability forms before fn-163
starts, as a small spec of their own, so fn-163 is a migration and not a design. The alternative,
most system capabilities staying hand-written with a reason, is what fn-163 exists to end.

### 10.3 `except` and `overriding` are both waivers

**Severity:** Low

**Problem.** Both are recorded in `.waivers.json`; the verbs do not say so, and `overriding` is a
participle beside the imperative `except`.

**Planned.** fn-155 C shares one `because` across six `overriding` calls.

**Recommendation.** `waive(P, because = ...)` and `replace(P -> own, because = ...)`.

### 10.4 `Phased` and `Phasing`

**Severity:** Low

**Problem.** The machine mixin is `Phased[S, P](projection, writer)`; capability laws take a given
`Phasing[S, P]` and call `phasing.roleCases` and `phasing.phase`. One concept, two names, one of
them never introduced.

**Planned.** Nothing.

**Recommendation.** One name. `Phased` is the typeclass; the mixin provides it.

---

## 11. Compositions and channels

### 11.1 A composition's outcomes and facts are strings

**Severity:** Medium

**Problem.** `Composition[S](_.a -> A, _.b -> B)` keys members by field, and its outcome and fact
types are `String`: the composed keys `<field>_<fact>`. A composed Property reads facts by string,
and fn-164's contract has to add `.records(_.member -> MemberMachine, fact)` to get the member's
fact type back.

**Planned.** fn-164 (the typed `records` form); fn-150 adds progress claims on compositions.

**Recommendation.** Make the composed fact type a sum of the members' fact types (`A.Fact |
B.Fact`, or a framework `Composed[A.Fact, B.Fact]`), keyed in the IR exactly as today. The string
is a lowering artefact that should not be the author's type.

### 11.2 Channels are the model for declared mechanisms, and still need a hand-written catalog

**Severity:** Low

**Problem.** `channel[M](capacity, order, loss, duplicates)` with `Inbox[M]` in the state is the
pattern fn-123 follows for faults: declared once, rows derived. But a state that holds an `Inbox`
is the one remaining place an author writes `given Finite[S] = { given Finite[Inbox[M]] =
wire.contents; Finite.derived }`, and `loss = Loss.reliable` says the loss is reliable.

**Planned.** fn-123 reuses the channel pattern; nothing touches the catalog block.

**Recommendation.** Let `Inbox[M]` carry its channel in its type (`Inbox[wire.type]`), so its
catalog derives like everything else, and rename the parameter `delivery = Delivery.reliable |
.lossy`. Do it before fn-123 copies the shape for budgets.

---

## 12. Realizations

### 12.1 The realization vocabulary is the largest in the DSL and the least Temporal-agnostic

**Severity:** Medium

**Problem.** `Realizes(machine, learned, observations, runs)` with `controller`, `workers`,
`evidence`, `serverSteps`, `controls`; `DerivesFrom` with `changes`; `everyCase`, `onPath`,
`perform`; `rpc`, `await`, `withFields`, `extended`, `statusTable`; `Evidences`. Most live under
`model/temporal/realize`, which is right, but the shape is a second language an author learns
after the first, and fn-141 notes its lifter alone is 1,373 lines.

**Planned.** fn-141 Part B exports realizations first, by running them; fn-162 R8 adds `runs =
Seq(...)`; fn-144, fn-125 and fn-146 add tags, settings and CEL operands to the same surface.

**Recommendation.** Treat fn-141's realization step as the moment to cut the vocabulary: every
realization helper that exists only to be recognised by the lifter (the `Typed*` constructors
fn-141 R2 lists) goes, and what remains is plain data a kit builds. Before fn-144 and fn-125 add
to the realization surface, agree the one shape they add to.

### 12.2 Two `evidence` declarations, joined by name

**Severity:** Medium

**Problem.** A machine declares `val evidence: PartialFunction[Fact, String]`, "only the
exceptions", and a realization declares `object evidence extends Evidences(answered(Fact.x,
call), ...)`. The lowering joins them by the string. An author writing a new fact has to know
both exist.

**Planned.** fn-148 consolidates Testpilot evidence schemas on the Go side.

**Recommendation.** One declaration, on the realization, keyed by fact: the machine never names
evidence kinds. Where a machine must say "this fact is confirmed by nothing", that is a hole, not
an evidence line.

---

## 13. Cross-cutting

### 13.1 "level" and "class" collide with other meanings

**Severity:** Medium

**Problem.** *Level* is Product versus Protocol in every Model section, and the five expression
strata (value, state reading, step, step reading, claim) in the reader. *Class* is one assignment
of an action's inputs, in a language whose every example is a `case class`.

**Planned.** fn-160 renames the System level to Protocol; fn-131 Part A makes the reader check the
five expression levels, which makes the second meaning more prominent.

**Recommendation.** Rename the expression strata (*tier* or *place*) in fn-131, since it names them
in the reader's messages. Keep *class* but say once in the guide's conventions that a class of an
action is not a Scala class; the IR key `name-key(i1)` makes a rename expensive.

### 13.2 Sugar's core form is a comment until fn-141

**Severity:** High

**Problem.** Every sugar definition documents its `Core form:` in a comment, and the lifter has a
hand-written lowering for each. The two can disagree, and the lint only checks that the comment is
present. fn-141 R4 makes the definition the lowering, which is the fix, but fn-141 is last in the
batch and every spec before it adds sugar (`enters`, witnesses, `expect` blocks, groups, fault
declarations).

**Planned.** fn-141 Part A, scheduled thirteenth.

**Recommendation.** Pull fn-141 Part A (sugar once) forward to before fn-140, or require every
sugar the batch adds to be written in core forms from the start so that Part A finds nothing to
rewrite. Otherwise each new sugar is written three times and migrated once.

### 13.3 Declaration order is a correctness condition

**Severity:** Medium

**Problem.** Machines are `object`s; a `val` read before it is declared is `null`, and the
framework's overlap check and fn-141's export both depend on construction order. fn-156 adds
`-Wsafe-init` and the declaration-order lint runs before export (fn-141 R14), but the author still
learns the rule by hitting it.

**Planned.** fn-156 (done except documentation), fn-141 R14.

**Recommendation.** Make the sections `lazy` by construction (the framework's `Rules`, `Properties`
and `Queries` bases can defer their bodies until the machine is complete), so order is a style
rule the lint enforces and never a semantic one.

### 13.4 Naming by `val` is the right rule, with two exceptions that should go

**Severity:** Low

**Problem.** Every declaration is named after its `val` or object, except assumptions (8.2),
progress claims (6.3) and anything written as a string "where no `val` declares it". fn-141 R7
captures the `val` name at the declaration, which makes the exceptions unnecessary.

**Planned.** fn-141 R7.

**Recommendation.** After fn-141, refuse string names everywhere a `val` can name the thing.

---

## 14. Order of work

If the batch runs as `MILESTONES.md` lists it, the points above that have to land *before* a
given spec are:

| Before | Point |
| --- | --- |
| fn-159 (now) | 4.2 `disabled` as one definition; 4.1 the rule-form shape, since fn-159.7 spreads `enters` |
| fn-161 | 10.1 which facet form survives; 5.2 whether `from` survives |
| fn-163 | 10.2 action-targeted and history-scoped capabilities |
| fn-140 | 7.2 the name of the result; 7.1 named bounds |
| fn-164 | 6.1 pattern word order; 6.2 `Model[S]` |
| fn-123 | 11.2 `Inbox` catalog and `delivery`; 1.3 per-field ranges |
| fn-155 | 4.3 block-form equivalence |
| fn-141 | 13.2 sugar in core forms from the start; 2.2 statuses as data, so one lifting rule covers role tests; 2.3 one status mapping per kind |

Everything else can follow the batch.

# The DSL's concepts against Quint's: which earn their keep, which to simplify

Status: assessment with recommendations, 2026-10-17. No source was changed. Paths are relative to
the repository root; a bare `dsl/`, `reify/` or `samples/` path is under `model/framework/`. The
Temporal snippets are from the disabled sources under `model/temporal` (`.scala.disabled`), which
still spell the pre-fn-190 forms (`where`/`~>`, `s.copy`); the point of each snippet survives the
fn-190 rewrite, and each "proposed" block is written in the fn-190 form.

The question (owner, 2026-10-17): do we have too many concepts, and which of them are
over-engineered or overfitted, measured against Quint? This note inventories every author-facing
concept of the framework DSL, gives each its Quint analogue, its cost in framework and plugin code,
its use in the six samples and the 168 Temporal Model sources, and a verdict, with the code that
shows the point. It continues `ir-simplification.md` (the IR against Quint's) and
`samples-vs-quint.md` (the six samples written both ways); where those notes already made the point,
this one cites them.

## 0. Answer in brief

- **Quint has about a dozen concepts.** Module, `const`, `var`, `val`, `def`, `pure def`, `action`,
  `temporal`, `run`, `nondet` with `oneOf`, `any`/`all`, and the types. Everything else is a builtin
  operator or a command-line flag.
- **We carry about forty.** Roughly a third exist because we do something Quint does not: generate
  and judge tests against a real server. Those are kept. Another third are already being removed or
  reshaped by fn-189, fn-190, fn-191 and fn-192. The remaining third are the subject of this note:
  eight concepts where one mechanism would do the work of two or three, or where the framework
  carries a Temporal vocabulary it should not.
- **Recommended, in order of code saved per use lost:** the outcome as a plain type member and value
  instead of a given and a shared enum (§2.1); explicit `records` instead of status-recording phase
  enums (§2.2); roles as predicates in the Temporal kit instead of eleven framework traits (§2.3);
  choice names as strings instead of token vals (§2.4); one way to give an input its domain (§2.5);
  `because` as an argument, not an operator (§2.6); one mechanism for relating machines (§2.7);
  compositions reduced to members and syncs (§2.8, decided as fn-190 R17).

## 1. Inventory

Counts are occurrences of the concept's spelling in `samples/*.scala` (non-test) and in the Temporal
sources, taken with fixed-string grep on 2026-10-17; they are indicative, not exact. "Code" is the
framework and plugin code the concept owns, by file where a file is its own, otherwise an estimate
from reading.

| Concept | Quint analogue | Code (lines) | Samples / Temporal | Verdict |
| --- | --- | --- | --- | --- |
| `Machine` with vars, `init`, `end` | module with `var`s and `init` | `dsl/Machine.scala` 1,337 (shrinks under fn-190) | all | keep |
| `Actor` / `Internal`, `action` / `step` | none: every action is an `action` | `dsl/Action.scala` 418 | all / all | keep: decides who drives a Case |
| Action inputs as records, with domains | `nondet x = oneOf(S)` inside the action | `dsl/ActionInputs` 111, `InputSource` 252, `Domain` 308 | 16 / many | simplify (§2.5) |
| Transitions (`if`/`match`), effects | `action` bodies | `dsl/RuleBindings` 571, `CoreBindings` 422 | all | keep; fn-190 reshapes |
| Outcome: `O` type, `given Ok[O]`, shared `Outcome` | none; a disabled action | spread over `Machine`, `Syntax`; plugin outcome resolvers and carrier ~150 | `Ok[` 6 / 2; `rejects(` 0 / 49 | simplify (§2.1) |
| Facts (`F`), `records` | none; history is state the author writes | `dsl/MachineEvidence` 389 (part) | `records(` 3 / 79 | keep: what a Case checks in history |
| Status-recording phases (`Recorded`) | none | `declaredStatus`, `statusOf`, `recordedStatusIr` carrier ~200 | 2 / 21 | remove (§2.2) |
| Facets, phases | none; a module | `dsl/FacetBindings` 635 | 0 / 55 `Phased[` | keep as abstract vars (fn-190 R5) |
| Roles (`Live`, `Closed`, …) | a set or predicate | `dsl/Roles`, `reify/body/Roles` 180, role sets and patterns ~80 | 0 / 54 `Closed` | move to the kit as predicates (§2.3) |
| Named choices (`choice`, `choose`) | `any { … }`, unnamed | `choose`, `choiceToken`, `namedCopy` ~180 (shrinks under fn-189 R11) | 1 / 16 `choose(`, 0 / 31 `choice` | keep named; name by string (§2.4) |
| `because` explanations | none | ~30 lifter, a `Step` field | 0 / 39 | simplify (§2.6) |
| Holes | none | small | 0 / 52 `.reached` | keep: explicit unknowns |
| Channels, `Inbox`, capacity, order | a `Set`/`List` var | `dsl/Channel` 199, `InboxAssociation` 111, lifter ~90 | 1 / 13 | keep: bounded messaging the IR understands |
| Bounded ints (`UpTo`), erasing aliases | unbounded `int` | `dsl/Domain` part | 0 / 106 | keep (fn-190 R9) |
| Claims: invariants, postconditions | `val inv = …` checked by `--invariant` | `dsl/Claims` 348, `ExpectationBlock` 263 | all | keep; fn-190 R11 simplified |
| Monitors | `temporal` | `dsl/Monitor.scala` | 0 / 6 | keep |
| Progress (`leadsTo`), scenarios with `only`/`fair` | `temporal` with fairness in the spec | fn-190 R13 | few | keep |
| Witness / scenario / query | `run` and the CLI | `dsl/Syntax` part | 25 / 57 `query` | keep for witnesses and Cases only (fn-190 R15) |
| Assumptions, `Limits`, `Budget` | CLI flags | `dsl/Assume` 118, `Budget` 144 | 2 / 20 | gone (fn-190 R13, R14) |
| Negative controls, `FailureModel`, durability, `crash` | none | `dsl/Durability` 135 | 20 / 55 `crash` | keep crashes; controls become tests (fn-190 R15) |
| `Derived`, `rebind`, `extend` | module instance with overrides | `dsl/DerivationBindings` 579 | 6 / 31 `Derived` | simplify (§2.7) |
| `Refinement` maps | none | ~100 | 1 / 9 | simplify (§2.7) |
| Compositions, `Syncs`, member facts, `through` | module instances | `dsl/Compose` 272, `MemberPath` 164, `ComposedReference` 108 | 0 / 19 | reduce (§2.8, fn-190 R17) |
| Evidence (fact → history text) | none | `dsl/MachineEvidence` 389 | 0 / 116 | keep: how a Case judges history |
| Capabilities, gaps, waivers | none | `export/Capabilities` 1,313, `Gaps` 675 | 0 / 136 | keep in the kit; outside this note |
| Realizations, scripts, `Named`, commands | none | `realize/` 3,873 | 0 / 78 | keep: the test generator |
| Entities, observations | none | `dsl/Action` part, `NamesPhase` | 0 / 9 `Entity(`, 6 / 39 `observation` | keep; naming is out of scope |
| `Family` | module name | fn-189 R3 | all | gone |
| `Finite`, `Data`, `StateData`, native execution | none (Quint's simulator runs the IR) | `dsl/StructuralValue` 329, `StateField` 305, `NativeCheck` 784, `Vars` 377 | all | gone (fn-190 R6) |
| Sugar: `enter`, `stay`, `rejects`, `enters`, `@sugar` | none | `reify/body/Expansion` 583 | 0 / 68 `enters(` | gone (fn-189 R7, fn-190 R12) |

Of forty rows, fourteen are already decided by the four open specs, ten are the test-generation
layer Quint does not have and stay as they are, eight are modelling power the IR carries cheaply,
and eight are the candidates below.

## 2. The candidates

### 2.1 The outcome as a plain type member and value

**Today.** A machine names its outcome type, imports or declares a `given Ok[O]` that says which
value means "accepted", and the framework ships a shared `Outcome` enum. Bank, which never rejects:

```scala
// samples/Bank.scala:55-67
given Ok[Outcome] = Ok(Outcome.accepted)

object Model extends Machine[Outcome, Nothing]:
```

TaskQueue and the Nexus reject-after-close protocol have outcomes richer than accept and reject, and
each has to hide its given so that a `choose` across machines does not pick the wrong one:

```scala
// model/temporal/foundations/taskqueue/TaskQueue.scala:40-47
enum QueueOutcome:
  case committed, failed, delivered, acknowledged, lost, internal

object QueueOutcome:
  // A provider's step behind the interface is the one that answers `internal`. Kept in the
  // companion, where only a step known to answer a QueueOutcome finds it, so a `choose` of a feature's
  // own steps still finds its own `Ok` alone.
  given Ok[QueueOutcome] = Ok(QueueOutcome.internal)
```

```scala
// model/temporal/features/nexus/workflow/protocol/RejectAfterClose.scala:89-95
enum Answer:
  case accepted, retained, rejectedTransient, rejectedPermanent

object Answer:
  // Kept in the companion, and imported by this file alone: the Protocol level's other subjects, in
  // this package, answer the feature's Outcome.
  given Ok[Answer] = Ok(Answer.accepted)
```

The plugin resolves the given (`outcomeOf`, `declaredOutcome`, `knownOutcome`, `sharedOutcome` in
`reify/body/Expansion.scala`), checks its shape ("answers the outcome a `given Ok[O] = Ok(o)` of the
lifted sources names"), and carries it across files (`declaredOutcomeIr`, `saveOutcome` in
`reify/plugin/ReifyPlugin.scala`). The comments above are the cost in the Models: implicit
resolution is the wrong tool for "which value is my default".

**Proposed.** The outcome is a type member with a default value, both plain Scala, both members of
the machine, resolved by the compiler's ordinary override rules and read by the plugin as members
(the same rule fn-189 R4 applies to `init`, `end` and `evidence`):

```scala
object Model extends Machine:                 // Outcome defaults to the framework's accepted | rejected(r)
  ...

object TaskQueueProduct extends Machine:
  type Outcome = QueueOutcome
  val ok = QueueOutcome.internal              // what a step answers when its body says nothing

  object effects:
    def enqueueCommits() =
      outstanding = Outstanding.committed
      answer(QueueOutcome.committed)          // an explicit answer, where the step has one
```

What goes: `Ok`, the given, the shared `Outcome` enum's special status (it becomes the default
`Outcome` type the framework provides), the four resolvers and the carrier. What stays: rich outcomes
(`QueueOutcome`, `Answer`), `reject(r)` as the spelling for the default type's rejected case, and the
IR's `Step` outcome field unchanged. `samples-vs-quint.md` findings 5 and 11 describe the same
boilerplate from the sample side.

### 2.2 Explicit `records` instead of status-recording phases

**Today.** A phase enum can declare, per case, the fact a step that enters the case records; the
step then records it as a side effect of the phase assignment:

```scala
// model/temporal/features/activity/standalone/product/Product.scala:19-25
enum Phase(val status: Fact) extends Recorded[Fact]:
  case scheduled extends Phase(Fact.statusScheduled), Waiting
  case started extends Phase(Fact.statusStarted), Held
  case paused extends Phase(Fact.statusPaused), Suspended
  case cancelRequested extends Phase(Fact.statusCancelRequested), Held
  case completed extends Phase(Fact.statusCompleted), Succeeded
  case failed extends Phase(Fact.statusFailed), Failed
```

```scala
// model/temporal/actors/worker/Serving.scala:19-21
on(worker.stop) {
  when(Phase.polling) ~> enters(Phase.stopped)     // records Phase.stopped.status, invisibly
}
```

Behind it: `declaredStatus` reads the constructor argument out of each case's tree, `statusOf`
lifts it, a carrier (`recordedStatusIr`) stores it per case for other files, `effectSteps` refuses
a `records` of the same fact ("which its assignment of phase already records: drop the record"),
and `setterShape` distinguishes a setter that hands the draft the value from one that does not.
About 200 lines for 21 uses, all in Temporal.

**Proposed.** Say it where it happens. Under fn-190 an effect is assignments and `records` calls,
so the fact sits one line under the phase it belongs to:

```scala
enum Phase:
  case scheduled, started, paused, cancelRequested, completed, failed

object effects:
  def pause() =
    phase = Phase.paused
    records(Fact.statusPaused)
```

What goes: `Recorded`, the status constructor parameter, `declaredStatus`, `statusOf`, the carrier,
the duplicate refusal and the two setter shapes. If the Temporal Models want the discipline that
every phase assignment is followed by its status fact, that is a kit lint over effect bodies, not a
framework mechanism.

### 2.3 Roles as predicates in the kit

**Today.** The framework declares eleven roles as traits with an implication order, and a phase
case mixes them in (see the `Phase` enum above: `Waiting`, `Held`, `Suspended`, `Succeeded`,
`Failed`):

```scala
// dsl/Roles.scala:35-65
trait Live
trait Waiting extends Live
trait Retrying extends Waiting
trait Held extends Live
trait Suspended extends Live
trait Closed
trait Succeeded extends Closed
trait Failed extends Closed
trait Canceled extends Closed
trait Terminated extends Closed
trait TimedOut extends Closed
```

A Model tests a role three ways, each a special form in the plugin:

```scala
// model/temporal/features/nexus/workflow/protocol/Operation.scala:112
when[Closed] ~> rejects(Rejection.notFound)
// model/temporal/features/nexus/workflow/protocol/ForgedControl.scala:27
if s.phase.in[Closed] then reject(Outcome.rejected(Rejection.notFound), s)
// model/temporal/features/nexus/workflow/protocol/Protocol.scala:132
def created(p: Phase) = p.in[Live] || p.in[Closed]
```

The plugin computes each enum's role closure and its conflicts ("a phase is live or closed, never
both"; `reify/body/Roles.scala`, 180 lines), lifts every role test as membership in the unrolled list
of cases (`roleSet`, `roleTest`, `rolePattern`; `ir-simplification.md` §2.4 counts 167 such literals,
25% of the activity functions' bytes), and `typedRemainder` exists only so that a final
`case _: Closed` becomes a wildcard. The eleven words are Temporal's, in the Temporal-free framework.

**Proposed.** A role is a set the kit declares, in the kit:

```scala
// model/temporal/.../Product.scala
enum Phase:
  case scheduled, started, paused, cancelRequested, completed, failed

object roles:
  val closed = Set(Phase.completed, Phase.failed)
  val live = Phase.values.toSet -- closed
  val held = Set(Phase.started, Phase.cancelRequested)
```

```scala
on(handler.complete):
  if roles.closed(phase) then reject(Rejection.notFound)

def created(p: Phase) = roles.live(p) || roles.closed(p)
```

`roles.closed(phase)` is `set.contains`, a builtin under fn-192; the plugin lifts a call and keeps
nothing about roles. The exclusivity rules become a kit lint over the sets ("`held` and `suspended`
are disjoint"), where they can also be feature-specific. Cost: `case _: Closed` in a phase table
becomes `case p if roles.closed(p)`, and fn-190 R12's `case _: Role` form is re-cut to that guard.

### 2.4 Choice names as strings

**Today.** A branching is named by a token the author declares as a val, then used as the key of
each alternative:

```scala
// model/temporal/foundations/taskqueue/TaskQueue.scala:83-84
val enqueueCommits = choice
val enqueueFails = choice

// model/temporal/foundations/taskqueue/protocol/Queueing.scala:19-29
def enqueueDetail(s: State) =
  choose(
    enqueueCommits -> List(Step(QueueOutcome.committed, QueueDetail(...))),
    enqueueFails -> List(Step(QueueOutcome.failed, s, List(QueueFact.enqueueFailed)))
      .because("the durable write fails and no message is outstanding")
  )
```

The lifter resolves the token through aliases (`resolveSymbol`), checks that its val's right-hand
side is the `choice` def (`choiceToken`), refuses compiler-made names (`capturedName`), and refuses
two alternatives on one token. Thirty-one token vals name sixteen `choose` calls in Temporal.

**Proposed.** The name is a string, which is already a constant the lifter reads:

```scala
def enqueueDetail() =
  choose(
    "enqueueCommits" -> { custody = Custody.history; answer(QueueOutcome.committed) },
    "enqueueFails"   -> { records(QueueFact.enqueueFailed); answer(QueueOutcome.failed) }
  )
```

Named branching stays (SEMANTICS, Named choices: a witness and a Case say which way a step went).
What goes: `choice`, the token vals, three recognisers; the duplicate-name check is one line.

### 2.5 One way to give an input its domain

**Today.** An input record's field gets its domain from a field binding, or from a whole-record
`Finite` witness, or from its type's `Finite`, with a page of `framework/README.md` on precedence:

```scala
// samples/Bank.scala:61
val send = action.inputs[Send](inputField[Send](_.amount).in(domains.oneOf(0 to 3)))
// samples/NativeInputs.scala:12-15
val submit = action.inputs[Request](
  inputField[Request](_.amount).in(domains.oneOf(2, 0)),
  inputField[Request](_.ceiling).in(domains.oneOf(1 to 2))
)
```

**Proposed.** The field's type says it, as a state var's type does under fn-190 R9:

```scala
final case class Send(fromAddr: Addr, toAddr: Addr, amount: UpTo[3])
val send = action[Send]
```

A domain that is not a range or a type's full set of values (`oneOf(2, 0)`) becomes a two-case
enum. What goes: `domains.oneOf`, `inputField(...).in(...)`, the `InputField` descriptors and the
whole-record witness, with `Finite` enumeration (fn-190 R6). Five uses in the samples, none in
Temporal.

### 2.6 `because` as an argument

**Today.** An explanation is a post-fix operator on a step list, which the lifter applies by
rewriting the explanation field of each step the list holds, refusing when a step already has one:

```scala
// model/temporal/features/activity/standalone/product/Pausing.scala:22-24
pausePending -> enter(s, Fact.statusPaused),
pauseAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
  .because(pauseAlreadyRequestedReason)
// model/temporal/features/nexus/standalone/protocol/Operation.scala:77
rejects(Rejection.failedPrecondition).because("operation already completed")
```

**Proposed.** An argument of the form that makes the step:

```scala
reject(Rejection.failedPrecondition, because = "operation already completed")
```

and, on an accepting path, a statement beside `records`: `explain("a message not yet acknowledged
may be delivered again")`. The IR's explanation field stays, since it flows into Case reports; the
rewrite and its refusal go. Thirty-nine uses in Temporal, none in the samples.

### 2.7 One mechanism for relating machines

**Today.** Three exist. `Derived` with `rebind` swaps an effect, mostly for negative controls:

```scala
// samples/Bank.scala:136-138
object Leaky
    extends Derived(Model.rebind(client.send ~> Model.effects.sendFromNowhere)),
      NegativeControl:
```

`Derived` with `extend` adds a rule under an assumption, for fault models:

```scala
// model/temporal/foundations/taskqueue/product/TaskQueueProductUnderStorageLoss.scala:11-21
object TaskQueueProductUnderStorageLoss
    extends Derived(
      TaskQueueProduct
        .extend(on(fault.storageLoss) {
          where(TaskQueueProduct.states.holding) ~> TaskQueueProduct.queueing.effects.storageLossView
        })
        .assuming(storageLossAssumed)
    ),
      FailureModel
```

`Refinement` maps a Product machine's states onto a Protocol machine's:

```scala
// samples/Voting.scala:145-150
object refinement extends Refinement(Consensus):
  def toProduct(state: State): State =
    Consensus.init.copy(chosen = Model.at(state)(chosen(current)))
  val visibleOutcomes = (_: Voting.Outcome) => false
  val unobservable = List(voter.increaseMaxBal)
```

`DerivationBindings.scala` alone is 579 lines, and the plugin carries derivation operations
(`capturedRebind`, `capturedExtend`, `derivation-operation-v1`).

**Proposed.** fn-190 R15 already turns negative controls into tests (`assertRefuted(Model, ...)`),
which removes the `rebind` uses in the samples, and R13 retires `restrict` for progress. What is
left is the fault-model `extend` and `assuming` (gone with assumptions, R13). A second machine that
reuses the first's effects by name is plain Scala:

```scala
object TaskQueueProductUnderStorageLoss extends Machine:
  export TaskQueueProduct.effects
  object transitions extends Transitions:
    export TaskQueueProduct.transitions.*
    on(fault.storageLoss):
      if roles.holding(phase) then effects.storageLossView()
```

whether `export` of the effects and transitions reads well enough is for the fn-190.7 sample
rewrite to find out; the recommendation here is to list the remaining `Derived` uses in the
Temporal sources before the plugin diet decides what to carry. `Refinement` (one sample, nine
Temporal uses) stays only if the Protocol-to-Product proof is still wanted; otherwise it is a
postcondition written by hand.

### 2.8 Compositions reduced to members and syncs

**Today.** A composed state's member facts and member helpers are read through two special forms:

```scala
// model/temporal/features/activity/standalone/protocol/RecordOverQueue.scala:28-36
val failedCommitKeepsTheMessage = c.property holds (after =>
  after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
    (after.state.queue.outstanding != Outstanding.empty &&
      !after.records(_.activity, AdmissionFact.attemptAdmitted))
)
...
Exclusive.atMostOneActive(c)(through(_.activity, ActivityRecord.states.twoActive))
```

Each costs selector extraction, field paths, `MemberPath`, a composed `<field>_<fact>` text key and
a synthesised `State.through.activity.twoActive` function, about 150 lines of the body lifter for
two uses.

**Decided (owner, 2026-10-17, fn-190 R17).** Member facts are a case of the composition's own fact
enum, member reads are plain calls:

```scala
enum Fact:
  case activity(f: AdmissionFact)
  case queue(f: QueueFact)

object postconditions:
  val failedCommitKeepsTheMessage =
    if records(Fact.activity(AdmissionFact.admissionCommitFailed)) then
      check(queue.outstanding != Outstanding.empty)
      check(!records(Fact.activity(AdmissionFact.attemptAdmitted)))

Exclusive.atMostOneActive(c)(ActivityRecord.states.twoActive(activity))
```

What remains of compositions is the member list and the sync table, which is Quint's instances
with shared variables and is the minimum.

## 3. What is kept, and why

- **Actors versus internal steps, inputs, outcomes, facts, evidence, realizations.** The bridge from
  a model to a test against a real server: who performs a step, with what input, what the server
  answered, what the history must show, and how to drive it. Quint has none because it does not
  generate tests. `object user extends Actor: val pressButton = action` (`samples/Kettle.scala:19-21`)
  is the whole price on the author.
- **Channels with capacity and order.** `where(originSending(current) && !network.isFull)` and
  `network = network.send(Msg(next, payload))` (`samples/ReliableBroadcast.scala:149,169`) are one
  var and two builtins for the author; Quint spells the queue, its bound and its loss by hand in every
  model, and `samples-vs-quint.md` finding 21 shows the drift that causes.
- **Holes.** `AnswerDispatch.monitors.cancelDuringBackoff.reached` (`activity_mini/protocol/Answering.scala:54`)
  says "the model does not say what happens here" and the checker reports it as such. Quint has no
  way to say it.
- **Bounded ints, monitors, progress with `only`/`fair`.** Quint's `int` is unbounded and its
  `temporal` carries fairness in the spec; fn-190 R9 and R13 put ours in the same places.
- **Capabilities, gaps, waivers.** Temporal's vocabulary, in the kit. Large, and outside this note.

## 4. Order of decisions

Cheapest and least visible first, matching the owner's phasing (DSL changes in phase 1, lifter and
IR changes in phase 2):

1. §2.1 outcome and §2.2 explicit `records`: DSL changes with a direct plugin saving; both simplify
   the fn-190.4 lifter rules before they are written.
2. §2.4 choice names and §2.6 `because`: small DSL changes that remove recognisers.
3. §2.3 roles: a kit change plus a lint; re-cuts one line of fn-190 R12 and makes `typedRemainder`'s
   removal (fn-189.10) final.
4. §2.5 input domains: falls out of fn-190 R9 once bounded aliases exist.
5. §2.7 relating machines: needs the use list from the Temporal sources first.

Each the owner accepts becomes an R-ID on fn-190 (DSL) with its plugin half in fn-190.9 or fn-189,
not a spec of its own.

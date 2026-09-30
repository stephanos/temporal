# Walkthrough: the standalone activity Model in Scala 3

## What this is

Umpire is a model-based testing layer. A Model is the rulebook: it says which moves exist and
which are legal in each state. A test run is a playthrough of that rulebook against a real
Temporal server. The referee is the framework, which checks the rulebook is consistent before
any playthrough and later checks each playthrough against the rulebook.

A standalone activity is a Temporal activity started directly with `StartActivityExecution`,
with no workflow around it. It writes no history events, so the only way to watch it is to read
its status through `DescribeActivityExecution` or its result through `PollActivityExecution`.

`StandaloneActivity.scala` is the rulebook for one such activity. It declares the vocabulary, two
machines that describe the activity at two levels of detail, the claims the machines promise,
the paths a test should walk, and the sets that group those paths into test suites. The
framework it binds to is `Umpire.scala`; the worker it composes with is `Worker.scala`.

## The language in five minutes

Scala 3 code in this sample uses indentation instead of braces. A colon at the end of a line
opens a block, like Python.

An `enum` is a closed list of named values. A case can carry fields, in which case it is a small
record and each combination of field values is a distinct member.

```scala
enum AttemptResult derives Finite:
  case completed
  case failed(retryable: Boolean)
  case canceled
```

A `case class` is an immutable record with structural equality and a `copy` method that returns
a changed duplicate. `derives X` asks the compiler to generate an instance of the type class `X`
for it. A type class is an interface implemented next to the type rather than by it.

```scala
final case class ProductState(phase: ProductPhase) derives Finite, CanEqual
```

`match` is a switch that can destructure. The compiler checks that the arms cover every case of
an enum, and warns when they do not. With the `-Werror` flag that warning stops the build.

```scala
def attemptStartStep(state: ProductState): List[ProductStep] = state.phase match
  case ProductPhase.scheduled => productStep(ProductPhase.started, ProductFact.statusStarted)
  case _ => Nil
```

An `infix` method can be called without a dot or parentheses, so declarations read as sentences.
A `given` is a value the compiler passes automatically wherever a parameter marked `using` asks
for its type. A context-function block, written `Scope ?=> Unit`, is a block that receives such
a given implicitly, which is how `starts(...)` inside `machine(...) { ... }` finds its machine.

```scala
val completion = query("completion") find completes in completed limits three
```

`inline` marks code the compiler expands at the call site and can evaluate while compiling. The
sample uses it to reject an out-of-range literal or a wrong state count before any test runs.

## Vocabulary: entities, parties, actions, inputs

An entity is what a machine is about. The activity is named by the id the caller chose.

`StandaloneActivity.scala:28`
```scala
val activity: Entity = entity("activity") key "activityId"
```

A party is who performs an action. The framework fixes the list as an enum at
`Umpire.scala:157`; this Model uses `caller` and `worker`, and `system` is reserved for timers.

An action is a named side effect of a party, optionally creating or acting on an entity, with
typed finite inputs. Each `.input[...]("name")` line extends the action's tuple type by one
element, so the type of `start` says it takes three timeouts.

`StandaloneActivity.scala:56-63`
```scala
val start: Action[(Timeout, Timeout, Timeout)] =
  action("start")
    .party(caller)
    .creates(activity)
    .schema("temporal.api.workflowservice.v1.StartActivityExecutionRequest")
    .input[Timeout]("scheduleToClose")
    .input[Timeout]("scheduleToStart")
    .input[Timeout]("startToClose")
```

`attemptStart` has no input, so its type is `Action[EmptyTuple]`. `attemptResult`
(`StandaloneActivity.scala:72-81`) takes one input of type `AttemptResult` and names two
examples, one per member of `failed`, so a realization knows what to send.

An action class is one action with one assignment of its inputs. `AttemptResult` has four
members (`completed`, `failed(false)`, `failed(true)`, `canceled`), so `attemptResult` has four
classes. `Control` has four plain members, so `control` has four classes. `start` has eight,
one per assignment of three two-valued timeouts. The framework counts classes by enumerating the
input tuple type: `Action` carries a `Finite[I]` and `classes` maps over its values
(`Umpire.scala:173-174`).

A fault is not a separate kind of thing. `workerStop` is an ordinary action of the `worker`
party that names no entity. It is declared once in the worker module and imported here, so the
composition at the end of the file can synchronize the two copies.

`Worker.scala:43`
```scala
val workerStop: Action[EmptyTuple] = action("workerStop") party Party.worker
```

A finite input domain with a payload is an enum case with fields. `failed(retryable: Boolean)` is
one constructor and two classes because `Boolean` has two values. The `Finite` derivation walks
into the case and multiplies out its fields.

The action bodies in `Umpire.scala` are sketched with `???`, which in Scala means "not
implemented, throws if called". A real version would copy the action with one field set.

## State

The product state is one field, the phase. The protocol state adds the attempt count and the
three deadlines, because whether a timer fires is a question about the activity and not about
the request that started it.

`StandaloneActivity.scala:242-248`
```scala
final case class ProtocolState(
    phase: Phase,
    attempts: Attempts,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
) derives Finite, CanEqual
```

Every field is finite because the framework builds a table of every state and every action
class, and checks the table rather than sampling it. `Timeout` has two members. The attempt count
is bounded by a constant, and the type carries the bound.

`StandaloneActivity.scala:237-240`
```scala
inline val attemptBound = 2
type Attempts = Bounded[attemptBound.type]
object Attempts:
  inline def apply(inline n: Int): Attempts = Bounded[attemptBound.type](n)
```

`Bounded[N]` is an opaque type over `Int`, meaning an `Int` at runtime that the compiler treats
as a distinct type. Writing `Attempts(3)` is a compile error because `apply` is `inline` and
checks the literal against the bound. The successor saturates, so a retry past the bound stays
at it.

`Umpire.scala:98-101`
```scala
  inline def apply[N <: Int](inline i: Int): Bounded[N] =
    inline if i < 0 || i > constValue[N] then
      error("value out of 0.." + constValue[N])
    else i
```

`saturatingSucc` at `Umpire.scala:108` is `if b < constValue[N] then b + 1 else b`.

Enumeration is a type class, `Finite[T]`, with one method that lists every value. `derives
Finite` on an enum or a case class hands the compiler's `Mirror` (a compile-time description of
the type's cases and fields) to `Finite.derived`, which concatenates the cases of a sum and
multiplies the fields of a product.

`Umpire.scala:52-57`
```scala
  inline def derived[T](using m: Mirror.Of[T]): Finite[T] =
    inline m match
      case s: Mirror.SumOf[T] =>
        val cases = summonAll[s.MirroredElemTypes, s.MirroredLabel]
        new Finite[T]:
          def values = cases.flatMap(_.values).asInstanceOf[IndexedSeq[T]]
```

The product arm (lines 58-61) summons a `Finite` for the field tuple and maps the constructor
over it.

That is how the product machine has 9 states (nine phases, one field) and the protocol machine
288 (twelve phases, three attempt counts, three two-valued deadlines: 12 x 3 x 2 x 2 x 2). The
order of `values` is declaration order, so the table order is stable across runs.

## Step functions

A step function takes the state and the action's inputs and returns a list of rows. A row is an
outcome, the next state, and the facts an observer can read back (`Step` at `Umpire.scala:23`).
An empty list means the action is not enabled in that state.

The product `attemptResultStep` dispatches on the phase first, then on the result.

`StandaloneActivity.scala:144-156`
```scala
def attemptResultStep(state: ProductState, result: AttemptResult): List[ProductStep] =
  state.phase match
    case ProductPhase.started => result match
      case AttemptResult.completed => productStep(ProductPhase.completed, ProductFact.statusCompleted)
      case AttemptResult.failed(false) => productStep(ProductPhase.failed, ProductFact.statusFailed)
      case AttemptResult.failed(true) => productStep(ProductPhase.scheduled, ProductFact.statusScheduled)
      case AttemptResult.canceled => Nil
    case ProductPhase.cancelRequested => result match
      case AttemptResult.completed => productStep(ProductPhase.completed, ProductFact.statusCompleted)
      case AttemptResult.failed(false) => productStep(ProductPhase.failed, ProductFact.statusFailed)
      case AttemptResult.failed(true) => productStep(ProductPhase.canceled, ProductFact.statusCanceled)
      case AttemptResult.canceled => productStep(ProductPhase.canceled, ProductFact.statusCanceled)
    case _ => Nil
```

Arm by arm, from `started`: a completed result finishes the activity and Describe reads
COMPLETED. A non-retryable failure fails it. A retryable failure puts it back to `scheduled`,
and Describe reads SCHEDULED again; unlike the Nexus Model, the retry is visible at this level.
A cancel result with no cancel requested is not a row. From `cancelRequested` the first two arms
are the same; a retryable failure now settles as canceled because the server does not retry an
attempt whose cancel was requested; a cancel result is honored. Every other phase has no row,
because no attempt is in flight.

The protocol version dispatches on three source phases because they react differently to the
same retryable failure.

`StandaloneActivity.scala:309-326`
```scala
def protocolAttemptResultStep(state: ProtocolState, result: AttemptResult): List[ProtocolStep] =
  state.phase match
    case Phase.started => result match
      case AttemptResult.completed => moves(state, Phase.completed, List(ProtocolFact.statusCompleted))
      case AttemptResult.failed(false) => moves(state, Phase.failed, List(ProtocolFact.statusFailed))
      case AttemptResult.failed(true) => moves(state, Phase.backingOff, List(ProtocolFact.attemptCount))
      case AttemptResult.canceled => Nil
    case Phase.cancelRequested => result match
      case AttemptResult.completed => moves(state, Phase.completed, List(ProtocolFact.statusCompleted))
      case AttemptResult.failed(false) => moves(state, Phase.failed, List(ProtocolFact.statusFailed))
      case AttemptResult.failed(true) => moves(state, Phase.canceled, List(ProtocolFact.statusCanceled))
      case AttemptResult.canceled => moves(state, Phase.canceled, List(ProtocolFact.statusCanceled))
    case Phase.pauseRequested => result match
      case AttemptResult.completed => moves(state, Phase.completed, List(ProtocolFact.statusCompleted))
      case AttemptResult.failed(false) => moves(state, Phase.failed, List(ProtocolFact.statusFailed))
      case AttemptResult.failed(true) => moves(state, Phase.paused, List(ProtocolFact.statusPaused))
      case AttemptResult.canceled => Nil
    case _ => Nil
```

From `started`, a retryable failure enters `backingOff`, a phase the product cannot see, and the
only fact is the attempt count read through Describe. This mirrors `TransitionRescheduled` in
the CHASM state machine. From `pauseRequested`, the same failure lands in `paused`, mirroring
`TransitionAttemptFailedWhilePauseRequested`: the pause the caller asked for during the attempt
takes effect when the attempt ends. From `cancelRequested` it settles as canceled. Three source
phases, three destinations, one action class.

`moves` (`StandaloneActivity.scala:273-274`) is the helper that keeps every other field and
changes the phase, through `state.copy(phase = phase)`.

Exhaustiveness is enforced by the compiler. Every inner `match` names all four results, and the
compiler's checker knows that `failed(true)` and `failed(false)` together cover `failed`. Drop an
arm and the build reports the missing pattern. The outer `match` uses a wildcard for the phases
that have no rows, which is deliberate: a new phase added later is not a row here until someone
says so.

## The machine and its table

The machine declaration is a block that receives a `MachineScope` implicitly. Every line inside
is a top-level function that takes that scope as a `using` parameter.

`StandaloneActivity.scala:405-434`
```scala
val activityProtocol: Machine[ProtocolState, ProtocolOutcome, ProtocolFact] =
  machine[ProtocolState, ProtocolOutcome, ProtocolFact]("activityProtocol"):
    forEntity(activity)
    refines(activityProduct)
    starts(Phase.unstarted)
    ends(Phase.completed, Phase.failed, Phase.canceled, Phase.terminated, Phase.timedOut)
    timers(backoff, scheduleToClose, scheduleToStart, startToClose)
    unobservable(backoff)
    evidence:
      case ProtocolFact.statusScheduled => "statusScheduled"
      case ProtocolFact.statusStarted => "statusStarted"
      case ProtocolFact.statusPaused => "statusPaused"
      case ProtocolFact.statusCancelRequested => "statusCancelRequested"
      case ProtocolFact.statusCompleted => "statusCompleted"
      case ProtocolFact.statusFailed => "statusFailed"
      case ProtocolFact.statusCanceled => "statusCanceled"
      case ProtocolFact.statusTerminated => "statusTerminated"
      case ProtocolFact.statusTimedOut(_) => "statusTimedOut"
      case ProtocolFact.attemptCount => attemptCount
    steps(
      start ~> startStep,
      attemptStart ~> protocolAttemptStartStep,
      attemptResult ~> protocolAttemptResultStep,
      control ~> protocolControlStep,
      workerStop ~> protocolWorkerStopStep,
      backoff ~> backoffStep,
      scheduleToClose ~> scheduleToCloseStep,
      scheduleToStart ~> scheduleToStartStep,
      startToClose ~> startToCloseStep,
    )
```

`starts` and `ends` name states by phase. The framework finds the field called `phase` through
the `Phased` type class and expands over every other field, so `starts(Phase.unstarted)` is 24
states (one phase, three counts, eight deadline assignments) and the five end phases are 120.

`timers` lists the `system` actions the machine owns; `unobservable` says the backoff timer
records nothing, so a path through it is confirmed by the step after it.

`evidence` is a total function from facts to what an observer reads. For a system with no
history events, every string here names a status a `DescribeActivityExecution` call returns, and
`attemptCount` is an observation, a derived read with no event behind it. Because the function
must be total, a fact with no evidence line is a non-exhaustive match the compiler reports.

`steps` binds each action to its step function with `~>` (`Umpire.scala:221-225`). The
binding asks for a `TupledFunction` between the function's type and the action's tuple, which
checks arity: `start` takes three inputs, so its function must take a state and three timeouts.

The table is built when the machine's `table` field is first read (`Umpire.scala:250`, a
`lazy val`). It is every state crossed with every action class, each step function called once
per pair. `Table.build` is sketched as `???`. In this sample the table exists at test time: the
first munit test that touches the machine builds it. Nothing about the table is computed while
compiling.

## Two levels and the refinement

The product machine says what the caller sees through Describe, with no account of how. The
protocol machine says how the server gets there: the backoff the product cannot see, the pause
request a running attempt has to finish before it takes effect, the three timers, the attempt
count. Properties are written once, on whichever machine can state them, and a product claim is
carried to the protocol machine by the refinement.

`productOf` reads a protocol state as a product state.

`StandaloneActivity.scala:390-399`
```scala
def productOf(state: ProtocolState): ProductState = ProductState(state.phase match
  case Phase.unstarted | Phase.scheduled | Phase.backingOff => ProductPhase.scheduled
  case Phase.started | Phase.pauseRequested => ProductPhase.started
  case Phase.paused => ProductPhase.paused
  case Phase.cancelRequested => ProductPhase.cancelRequested
  case Phase.completed => ProductPhase.completed
  case Phase.failed => ProductPhase.failed
  case Phase.canceled => ProductPhase.canceled
  case Phase.terminated => ProductPhase.terminated
  case Phase.timedOut => ProductPhase.timedOut)
```

`pauseRequested` maps to `started`, not `paused`, because the worker still holds the attempt.
Every answer the worker can give from `pauseRequested` is a product row from `started`, and the
pause request itself becomes a product stutter. The map is declared once more as a `given`, so
both the machine and a `verify` Query can find it.

`StandaloneActivity.scala:403`
```scala
given productView: Refines[ProtocolState, ProductState] = Refines(productOf)
```

The refinement rule: a protocol row from `s` to `s'` is fine if `productOf(s) == productOf(s')`
(a stutter) or the product has any row from `productOf(s)` to `productOf(s')`, under any action
class. The Lean version derives a mapping per row and is stricter about which product step a row
corresponds to; the Scala sketch specifies the looser rule.

`Refinement.rejected` at `Umpire.scala:328` is `None` when every row was accounted for and
otherwise names the first row that was not. Its body is sketched; the check would run at test
time, when a pin reads `rejected`.

One row by hand. Protocol `started` with `attemptResult(failed(true))` goes to `backingOff`.
`productOf(started)` is `started`, `productOf(backingOff)` is `scheduled`. Does the product have
a row from `started` to `scheduled`? Yes: `attemptResultStep` at line 149 returns exactly that
for `failed(true)`. The row is accounted for. Now protocol `started` with `control(pause)` goes
to `pauseRequested`. Both map to `started`, so it is a stutter. Accounted for as well.

## Properties

A same-step claim names an action class under `when` and holds of the step that action
produces. The lambda takes one step.

`StandaloneActivity.scala:449-451`
```scala
val completes: Property[ProtocolState] =
  property("completes")(activityProtocol) when attemptResult(AttemptResult.completed) holds: step =>
    step.state.phase == Phase.completed && step.facts.contains(ProtocolFact.statusCompleted)
```

`retryCompletes` fixes the whole state, so every field is named in a helper value.

`StandaloneActivity.scala:464-466`
```scala
val retryCompletes: Property[ProtocolState] =
  property("retryCompletes")(activityProtocol) when attemptResult(AttemptResult.completed) holds: step =>
    step.state == completedOnRetry && step.facts.contains(ProtocolFact.statusCompleted)
```

`completedOnRetry` (line 460) is `ProtocolState(Phase.completed, Attempts(2), unset, unset, unset)`.

A transition claim has no `when` and its lambda takes two steps, before and after. The two
`holds` overloads are told apart by the lambda's arity. Both transition claims are declared on
the product machine.

`StandaloneActivity.scala:444-446`
```scala
val terminalIsFinal: Property[ProductState] =
  property("terminalIsFinal")(activityProduct) holds: (before, after) =>
    !productTerminal(before.state) || after.state.phase == before.state.phase
```

`StandaloneActivity.scala:484-486`
```scala
val pausedIsNotDispatched: Property[ProductState] =
  property("pausedIsNotDispatched")(activityProduct) holds: (before, after) =>
    before.state.phase != ProductPhase.paused || after.state.phase != ProductPhase.started
```

A Property is typed by its machine's state type, which is what lets the Query below check that
Property and Scenario belong together.

## Scenarios and limits

A Scenario names a start by phase and a list of classed actions in order. A classed action is
spelled by applying the action to its inputs; a timer or a no-input action is listed bare.

`StandaloneActivity.scala:518-521`
```scala
val retriedThenCompleted: Scenario[ProtocolState] =
  scenario("retriedThenCompleted")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.failed(true)), backoff,
    attemptStart, attemptResult(AttemptResult.completed))
```

`start(unset, unset, unset)` works because the framework adds an `apply` with three parameters
to any `Action[(A, B, C)]`. `backoff` is an `Action[EmptyTuple]` and the `actions` parameter
accepts either shape through a union type, `Classed | Action[EmptyTuple]`.

`StandaloneActivity.scala:533-536`
```scala
val pausedThenCompleted: Scenario[ProtocolState] =
  scenario("pausedThenCompleted")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), control(Control.pause), control(Control.unpause), attemptStart,
    attemptResult(AttemptResult.completed))
```

Limits bound the search: how many steps a trace may have, how many actions, and how many
candidate traces to examine. The declaration is `inline` and rejects `actions < steps` at
compile time.

`StandaloneActivity.scala:550-552`
```scala
val three: Limits = limits("three")(steps = 3, actions = 3, search = 4096)
val four: Limits = limits("four")(steps = 4, actions = 4, search = 32768)
val six: Limits = limits("six")(steps = 6, actions = 6, search = 262144)
```

## Queries

A `find` Query asks whether the Scenario reaches a step where the Property holds, and a set can
later realize the found trace as a test. A `verify` Query asks whether the Property holds on
every trace of the Scenario within the limits, and is never realized.

`StandaloneActivity.scala:563`
```scala
val retry = query("retry") find retryCompletes in retriedThenCompleted limits six
```

`StandaloneActivity.scala:572`
```scala
val pauseHolds = query("pauseHolds") verify pausedIsNotDispatched in pausedThenCompleted limits six
```

`pauseHolds` pairs a product Property with a protocol Scenario. The `in` method asks for a
`Refines[S, P]` given, which is the identity when the machines match and `productView` when the
Scenario's machine refines the Property's. Any other pairing has no given and does not compile.

`Umpire.scala:406`
```scala
  infix def in[S](s: Scenario[S])(using via: Refines[S, P]): QueryLimits[S, P] = QueryLimits(name, p, s, via, find)
```

The search itself is `Query.run`, sketched as `???`: a breadth-first walk over table rows along
the Scenario, cut at the search budget. It runs in two places. As a munit test, every find Query
in the functional set must return `Found`. And in the test module's compilation, through a
macro.

`Umpire.scala:418`
```scala
  inline def pinned[S](inline q: Query[S]): Query[S] = ${ pinnedImpl('q) }
```

A macro in Scala 3 is a function that runs inside the compiler; `'q` quotes the argument as a
syntax tree and `${ ... }` splices the result back. This one loads the Query from the already
compiled Model module, runs it, and reports a failure at the author's expression. A failed
`find` looks like this in the compiler output:

```
-- Error: Pins.scala:44:38
44 |  val activityRetry = Query.pinned(Activity.retry)
   |                                   ^^^^^^^^^^^^^^
   |retry: `retryCompletes` is not reached on `retriedThenCompleted`
   |within six (261 044 candidate traces searched)
```

The catch is that a macro cannot evaluate code from its own compilation run, so the Query must
live upstream. That is why the compile-time pin sits in `Pins.scala`, a separate module, and not
next to the Query.

## Sets

A set groups Queries into a suite and says which parties the test harness drives and which it
only observes. The functional set drives both the caller and the worker.

`StandaloneActivity.scala:581-585`
```scala
val standaloneActivityTests: Set = set("standaloneActivityTests"):
  purpose(functional)
  bind(caller -> driven, worker -> driven)
  queries(completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
    scheduleToStartTimeout, startToCloseTimeout)
```

There is no `repeat(implementation)` line. The Nexus Model repeats each Query under both server
implementations, HSM and CHASM. Standalone activities exist only in CHASM, so there is nothing
to repeat over.

The canary set observes the worker: a deployment runs its own worker, and the verifier reads
which result occurred and checks the machine allows it. Only paths where every step records
evidence qualify.

`StandaloneActivity.scala:592-595`
```scala
val standaloneActivityCanary: Set = set("standaloneActivityCanary"):
  purpose(canary)
  bind(caller -> driven, worker -> observed)
  queries(completion, cancel)
```

The exploratory set names no Queries. It covers the protocol machine: the rows within the
budget's steps of a start, the results they reach, and the members of the classes they claim.

`StandaloneActivity.scala:602-607`
```scala
val standaloneActivityExploration: Set = set("standaloneActivityExploration"):
  purpose(exploratory)
  bind(caller -> driven, worker -> driven)
  machine(activityProtocol)
  cover(rows | results | classMembers)
  budget(four)
```

## Composition with the worker

The protocol machine treats `workerStop` as a stutter: the activity cannot see its worker, so
the step keeps the state and records nothing. The worker module gives the worker its own
machine.

`Worker.scala:67-71`
```scala
val polling: Machine[WorkerState, WorkerOutcome, WorkerFact] =
  machine[WorkerState, WorkerOutcome, WorkerFact]("polling"):
    forEntity(worker)
    starts(Phase.polling)
    ends(Phase.polling, Phase.stopped)
```

Its three steps (`workerStop`, `workerResume`, `serve`) move between `polling` and `stopped`; a
stopped worker has no `serve` row.

The activity's view of the worker keeps only stop and serve. A resume would stay executable on
its own and admit a stop, a resume and then a start, which is not what the claim is about.

`StandaloneActivity.scala:619`
```scala
val activityWorker: Machine[WorkerState, ?, ?] = Worker.polling restrict (workerStop, serve)
```

The composition is an `object` extending `Compose`, so its members are named fields.

`StandaloneActivity.scala:621-630`
```scala
final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState) derives Finite, CanEqual

object standaloneActivity extends Compose[StandaloneActivityState]("standaloneActivity"):
  val activity = member("activity", activityProtocol)(_.activity)
  val worker = member("worker", activityWorker)(_.worker)
  sync(workerStop, activity(workerStop) || worker(workerStop))
  sync(attemptStart, activity(attemptStart) || worker(serve))
  starts(activity at Phase.unstarted, worker at Worker.Phase.polling)
  ends(activity at Phase.completed, activity at Phase.failed, activity at Phase.canceled,
    activity at Phase.terminated, activity at Phase.timedOut)
```

Each `sync` line says two member actions fire as one step. The activity's `workerStop` stutter
and the worker's real phase change become one row, and every `attemptStart` is also the worker
serving. A stopped worker has no `serve` row, so a stopped worker has no `attemptStart` row.

`StandaloneActivity.scala:633-635`
```scala
val startedByPollingWorker: Property[StandaloneActivityState] =
  property("startedByPollingWorker")(standaloneActivity) when attemptStart holds: step =>
    step.state.worker.phase == Worker.Phase.polling
```

The Query `stoppedWorkerStartsNothing` verifies this on `stoppedBeforeRetry`
(`StandaloneActivity.scala:641-645`): a polling worker picks the first attempt up and fails it
retryably, the backoff fires, the worker stops, and the schedule-to-start deadline settles the
retry nobody picked up. The path performs `attemptStart` once while the worker polls, so the
claim is exercised on it rather than holding vacuously; an earlier version of the Scenario
stopped the worker before any dispatch and never fired the claim. What the composed claim proves
is that the ordering convention in the single-machine Scenarios is not hiding a bug: nothing in
the composed table lets a stopped worker pick a task up.

`Compose.machine`, the product of member machines, is sketched as `???`. The `sync`, `starts`
and `ends` calls record into buffers that a real build would read.

## Pins

Pins are tests that guard the table against a step function that stops saying what it says.

`Pins.scala:206-209`
```scala
  test("protocol: twelve phases, three counts, three deadlines; the five ends") {
    assertEquals(activityProtocol.table.states.size, 12 * (attemptBound + 1) * 2 * 2 * 2)
    assertEquals(activityProtocol.ends.size, 5 * (attemptBound + 1) * 2 * 2 * 2)
  }
```

This guards the state space: add a field to `ProtocolState` and the count changes. The same
count is also pinned at compile time in the `CompileTimePins` object through `pinStates`, which
folds the size to a literal and fails the build with the actual number.

`Pins.scala:211-213`
```scala
  test("protocol: a cancel result with no cancel requested is not a row") {
    assertEquals(protocolAttemptResultStep(at(Phase.started, Attempts(1)), AttemptResult.canceled), Nil)
  }
```

This guards one arm: a cancel result is honored only after a cancel request.

`Pins.scala:247-250`
```scala
  test("refinement: every protocol row is a product step or a product stutter") {
    val r = activityProtocol.refinement.getOrElse(fail("activityProtocol declares no refinement"))
    assertEquals(r.rejected, None)
  }
```

This guards the relationship between the two machines: a protocol row the product cannot
account for names itself in `rejected`.

## From model to running test

After this file, a found trace is lowered to a Case: the sequence of classed actions, the
evidence expected after each, and the Contract, which is the Property's clause triggered by the
action the Case performs. A realization maps each action class to a concrete request against a
Temporal server, using the `schema` and `examples` lines. No realization exists yet for
standalone activities; the Nexus Model has one, this Model does not. The Go Testpilot runtime
would then drive the parties the set marks `driven`, read the evidence through Describe and
Poll, and issue a verdict per Case. None of that layer is in this sample; the sample stops where
the checked Model, its Properties, Scenarios, Queries and Sets exist.

## Gaps and gradual growth

The file says what it leaves out. Reset is not modeled, like cancellation in the Nexus Model.
Heartbeat timeout is not modeled. The worker stop is a stutter on the protocol machine and
invisible on the product machine, and the composition is the place where it becomes real.

Empty steps are the growth mechanism. A `Nil` arm says "not a row yet", and a wildcard `case _`
at the outer level says "no other phase has rows for this action yet". Adding a new action means
declaring it, writing a step function with `Nil` everywhere it does not apply, and binding it in
`steps`. The exhaustiveness checker then lists every inner `match` the new action's domain
touches, the pins report the new class and row counts, and the refinement check says whether
the product machine needs a row too. A new phase works the same way: add it to the enum, and the
compiler names every total `match` that now has a hole.

## Mental model recap

- A Model is enums, case classes and step functions; the framework turns them into a finite
  table of states by action classes.
- A step function returns rows; an empty list is "not enabled", and a row that keeps the state
  with no facts is a stutter.
- Two machines describe one activity: the product says what Describe shows, the protocol says
  how the server gets there, and `productOf` plus the refinement tie them.
- Facts are what an observer reads; with no history events, every fact here is a status or an
  observation.
- Properties are same-step claims on an action class or transition claims on consecutive steps.
- Scenarios are paths, limits bound the search, and Queries either find a witness or verify every
  trace.
- Sets group Queries into functional, canary and exploratory suites and say who is driven.
- The compiler checks exhaustiveness, arity, bounded literals and state counts; the table, the
  refinement and the search run at test time, or at compile time of a downstream module.

## Where this implementation is weak

- Nothing has been compiled or run. The review at `cmp/.eval/scala.md` found several framework
  signatures that could not type-check as first written: `Phased` projections in `starts` and
  `ends`, the `evidence` parameter order, an `Entity` field and method sharing a name, `derives
  Finite` for enum singleton cases, and `verify` Queries whose Property and Scenario had
  different state types. All five were reworked after the review (`Phased.Aux`, a leading
  `using` clause, `keyField`, in-place derivation of singleton cases, the `Refines` witness), but
  the reworked code has not been compiled either.
- The bodies that decide anything are sketches: `Table.build`, `Refinement.rejected`,
  `Query.run`, `restrict` and `Compose.machine` are `???`. The compile-time and test-time claims
  above describe where a check would run, not a check that has run.
- `Phased.derived` and `Finite.sizeOf` rely on `inline` matches folding to literals in exactly
  the way the sketch assumes. If they do not fold, `pinStates` becomes a runtime check and
  `starts(Phase.unstarted)` needs an explicit type argument.
- The review also noted visible ceremony: fully qualified enum cases everywhere, a type
  ascription on every top-level `val`, and three explicit type arguments on each `machine[...]`
  call. None of it is wrong, but the Lean original is terser.

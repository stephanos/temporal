# Walkthrough: the standalone activity Model in Kotlin

## What this is

Umpire is model-based testing for Temporal features. A Model is the rulebook: it says which moves
each party may make in each state and what the server records when they do. A test is a playthrough
of the rulebook against a real server, and the referee checks that what the server recorded is a
move the rulebook allows. A standalone activity is a Temporal activity started directly with
`StartActivityExecution`, with no workflow around it; it writes no history events, so everything a
test can observe is a status read through `DescribeActivityExecution` or a result read through
`PollActivityExecution`. `StandaloneActivity.kt` is the rulebook for one such activity, written
against the framework surface in `Umpire.kt`, with the worker it depends on in `Worker.kt`.

## The language in five minutes

Kotlin is a statically typed language on the JVM. Everything below is a plain declaration; there
are no macros and no custom syntax. Six features carry the whole file.

**Top-level values.** A `val` outside any class is a global constant, initialized when the file's
class loads. Every Model declaration is one, such as the `val activity = entity("activity") { ... }`
quoted in the next section.

**Trailing lambdas with a receiver.** A function's last argument can be a `{ ... }` block. If its
type is `Scope.() -> Unit`, the block runs with a `Scope` object as `this`, so `key = "activityId"`
inside `entity(...) { }` sets a property of that scope. That is the entire DSL mechanism.

**Enums and sealed interfaces.** An `enum class` lists its values. A `sealed interface` lists its
implementations in the same file, so the compiler knows all of them; `data object` is a singleton
implementation and `data class` one with fields:

```kotlin
enum class Control { Pause, Unpause, RequestCancel, Terminate }
data class Failed(val retryable: Boolean) : AttemptResult
```

**Exhaustive `when`.** A `when` used as a value must cover every case of an enum or sealed type, or
it does not compile. `is Type ->` matches a subtype; a bare value matches by equality.

**Infix functions and function references.** A one-argument function marked `infix` is called
without dot or parentheses, and `::name` refers to a function as a value, so a step binding reads
`attemptResult runs ::attemptResultStep`.

**Data classes.** A `data class` gets equality by field, a `copy(field = ...)` method, and default
arguments, so `ProtocolState(Phase.Unstarted)` fills the other four fields.

## Vocabulary: entities, parties, actions, inputs

An entity is what a machine is about. The activity is named by its id, since there is no workflow
history to name it (`StandaloneActivity.kt:43`):

```kotlin
val activity = entity("activity") { key = "activityId" }
```

Parties are who acts. The framework predefines `caller`, `handler`, `worker`, `network` and
`operator` as values, and reserves `system` for the timers a machine owns. An action is a named
side effect of one party, on or creating an entity, with typed inputs. The caller starts the
activity (`StandaloneActivity.kt:65-70`):

```kotlin
val start = action<Timeout, Timeout, Timeout>("start") {
    party = caller
    creates = activity
    schema = "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    input("scheduleToClose", "scheduleToStart", "startToClose")
}
```

The type arguments `<Timeout, Timeout, Timeout>` fix the action's arity: `start` is an
`Action3<Timeout, Timeout, Timeout>` and later can only be bound to a step function taking three
`Timeout`s. `input(...)` names the fields. The worker's poll receives the task
(`attemptStart`, `StandaloneActivity.kt:73-77`, no inputs) and its response resolves the attempt
(`StandaloneActivity.kt:79-90`; `party = worker`, `on = activity` and the schema lines trimmed):

```kotlin
val attemptResult = action<AttemptResult>("attemptResult") {
    input("result")
    examples {
        AttemptResult.Failed(retryable = false) realizedAs "ApplicationFailure nonRetryable"
        AttemptResult.Failed(retryable = true) realizedAs "ApplicationFailure retryable"
    }
}
```

The caller's four control requests share one action, `control`, with a `Control` input and a
`Delivery` result declared by `results<Delivery>()`, because a request against a finished activity is
answered `NotFound` rather than refused (`StandaloneActivity.kt:92-101`). The last action
(`StandaloneActivity.kt:103`):

```kotlin
val workerStop = action("workerStop") { party = worker }
```

`workerStop` is a fault: the worker process goes away. It is an ordinary action of the `worker`
party that names no entity, so nothing the server records points at it. Faults are not a separate
kind of thing in this framework.

An input domain must be finite so the framework can list every way an action can be performed.
One such way is an action class: one action with one assignment of its inputs.
`AttemptResult` has a constructor with a field, and the field counts (`StandaloneActivity.kt:50-58`):

```kotlin
sealed interface AttemptResult {
    data object Completed : AttemptResult
    data class Failed(val retryable: Boolean) : AttemptResult
    data object Canceled : AttemptResult
}

enum class Delivery { Accepted, NotFound }

enum class Control { Pause, Unpause, RequestCancel, Terminate }
```

`attemptResult` therefore has four classes: `Completed`, `Failed(false)`, `Failed(true)`,
`Canceled`. `control` has four, one per enum value. `start` has eight, one per assignment of three
`Timeout`s, and `attemptStart` and `workerStop` have one each. A domain without fields is an
`enum class`; a domain with a field has to be a `sealed interface`, because Kotlin enum entries
cannot carry per-entry parameters. The `examples` block pins one concrete realization value to
each `Failed` class, which is the granularity a protobuf `oneof` has.

## State

There are two machines and two state types. The product state is one field
(`StandaloneActivity.kt:120-124`):

```kotlin
enum class ProductPhase {
    Scheduled, Started, Paused, CancelRequested, Completed, Failed, Canceled, Terminated, TimedOut,
}

data class ProductState(val phase: ProductPhase)
```

The protocol state adds the attempt count and the three deadlines, each a state field because
whether a timer fires is a question about the activity, not about the request that created it
`Phase` (`StandaloneActivity.kt:250-253`) has twelve values: the product's nine plus `Unstarted`,
`BackingOff` and `PauseRequested`. The rest (`StandaloneActivity.kt:258-280`):

```kotlin
const val attemptBound = 2

/** `0..attemptBound` with a saturating successor; declared here as the Lean file declares its own. */
@JvmInline
value class Attempts(val count: Int) {
    init {
        require(count in 0..attemptBound) { "attempts $count outside 0..$attemptBound" }
    }

    fun saturatingSucc() = Attempts(minOf(count + 1, attemptBound))

    companion object : Finite<Attempts> {
        override val values = (0..attemptBound).map(::Attempts)
    }
}

data class ProtocolState(
    val phase: Phase,
    val attempts: Attempts = Attempts(0),
    val scheduleToClose: Timeout = Timeout.Unset,
    val scheduleToStart: Timeout = Timeout.Unset,
    val startToClose: Timeout = Timeout.Unset,
)
```

Every field is finite because the framework builds a table by listing every state. An `Int` would
not do, so the attempt count is a `value class` (a wrapper the compiler erases at runtime) whose
constructor rejects values outside `0..2` and whose `saturatingSucc` stops at the bound rather than
wrapping. The bound is written here because nothing wires the search Limits into a state. The
`companion object : Finite<Attempts>` is how a type declares its own value list explicitly.

The framework lists states by reflection over the type (`Umpire.kt:54-59`):

```kotlin
                type == Boolean::class -> listOf(false, true).asFinite()
                type.companionObjectInstance is Finite<*> -> type.companionObjectInstance as Finite<*>
                type.java.isEnum -> type.java.enumConstants.toList().asFinite()
                type.isSealed -> type.sealedSubclasses.flatMap { of(it).values }.asFinite()
                type.objectInstance != null -> listOf(type.objectInstance!!).asFinite()
                type.isData -> product(type.primaryConstructor!!).asFinite()
```

These are the arms of `Finite.of(type: KClass<T>)`; a `KClass` is Kotlin's runtime description of
a type. An enum lists its constants; a sealed type
lists each implementation, recursing into a `data class` implementation so `Failed(retryable)`
becomes two values; a `data class` state is the cartesian product of its constructor parameters
(`Umpire.kt:70-73`). So `ProductState` has 9 values and `ProtocolState` has 12 phases times 3
attempt counts times 2 times 2 times 2 deadlines, 288 values. This runs once per type when the
machine is first built and needs the `kotlin-reflect` library; a code generator could emit the same
lists at build time.

## Step functions

A step is what one action does in one state (`Umpire.kt:22`):

```kotlin
data class Step<S, O, F>(val outcome: O, val state: S, val facts: List<F>)
```

A step function takes the state and the action's inputs and returns a list of steps. The empty list
means the action is not enabled there. `ProductSteps` is a `typealias` for that list type, and
`productStep(phase, fact)` builds a one-element list with outcome `Accepted`
(`StandaloneActivity.kt:140-143`). The product's response to a worker result, arm by arm (`StandaloneActivity.kt:163-176`):

```kotlin
fun attemptResultStep(state: ProductState, result: AttemptResult): ProductSteps {
    if (state.phase != ProductPhase.Started && state.phase != ProductPhase.CancelRequested) return emptyList()
    return when (result) {
        AttemptResult.Completed -> productStep(ProductPhase.Completed, ProductFact.StatusCompleted)
        is AttemptResult.Failed -> when {
            !result.retryable -> productStep(ProductPhase.Failed, ProductFact.StatusFailed)
            state.phase == ProductPhase.CancelRequested -> productStep(ProductPhase.Canceled, ProductFact.StatusCanceled)
            else -> productStep(ProductPhase.Scheduled, ProductFact.StatusScheduled)
        }
        AttemptResult.Canceled ->
            if (state.phase == ProductPhase.CancelRequested) productStep(ProductPhase.Canceled, ProductFact.StatusCanceled)
            else emptyList()
    }
}
```

The first line refuses any phase in which no worker holds an attempt. A completed result completes
the activity. A non-retryable failure fails it. A retryable failure returns it to scheduled, since
Describe reads `SCHEDULED` again after the server reschedules; unlike the Nexus Model, the retry is
visible at this level, and only the backoff between the two attempts is hidden. A retryable failure
while a cancel is requested lands in canceled. A canceled result is honored only when a cancel was
requested; from `Started` it is refused with the empty list.

The protocol version dispatches on the phase first (`StandaloneActivity.kt:356-379`):

```kotlin
fun protocolAttemptResultStep(state: ProtocolState, result: AttemptResult): ProtocolSteps = when (state.phase) {
    Phase.Started -> when (result) {
        AttemptResult.Completed -> moves(state, Phase.Completed, listOf(ProtocolFact.StatusCompleted))
        is AttemptResult.Failed ->
            if (result.retryable) moves(state, Phase.BackingOff, listOf(ProtocolFact.AttemptCount))
            else moves(state, Phase.Failed, listOf(ProtocolFact.StatusFailed))
        AttemptResult.Canceled -> emptyList()
    }
    Phase.CancelRequested -> when (result) {
        AttemptResult.Completed -> moves(state, Phase.Completed, listOf(ProtocolFact.StatusCompleted))
        is AttemptResult.Failed ->
            if (result.retryable) moves(state, Phase.Canceled, listOf(ProtocolFact.StatusCanceled))
            else moves(state, Phase.Failed, listOf(ProtocolFact.StatusFailed))
        AttemptResult.Canceled -> moves(state, Phase.Canceled, listOf(ProtocolFact.StatusCanceled))
    }
    Phase.PauseRequested -> when (result) {
        AttemptResult.Completed -> moves(state, Phase.Completed, listOf(ProtocolFact.StatusCompleted))
        is AttemptResult.Failed ->
            if (result.retryable) moves(state, Phase.Paused, listOf(ProtocolFact.StatusPaused))
            else moves(state, Phase.Failed, listOf(ProtocolFact.StatusFailed))
        AttemptResult.Canceled -> emptyList()
    }
    else -> emptyList()
}
```

It dispatches on phase because the three phases in which a worker holds an attempt react differently
to the same retryable failure. From `Started` the server reschedules into a backoff
(`TransitionRescheduled` in CHASM's `statemachine.go`). From `CancelRequested` a retryable failure
is not retried; the activity is canceled. From `PauseRequested` the failed attempt lands in
`Paused` (`TransitionAttemptFailedWhilePauseRequested`). `moves` is a helper that copies the state
with a new phase and wraps it in an accepted step.

Exhaustiveness is enforced by the compiler wherever a `when` has no `else`. The inner `when (result)`
blocks cover the three `AttemptResult` implementations, so adding a fourth is a compile error at each
of them. The outer `when (state.phase)` ends in `else -> emptyList()`, so adding a phase is silent
here; the file pays the price of spelling out every phase only in `terminalPhase`, `productOf` and
the `evidence` blocks, where every arm differs.

## The machine and its table

A machine ties a state type, an entity, and the step functions together
(`StandaloneActivity.kt:463-495`):

```kotlin
val activityProtocol = machine<ProtocolState, ProtocolOutcome, ProtocolFact>("activityProtocol") {
    entity = activity
    refines(activityProduct) via ::productOf
    starts(ProtocolState(Phase.Unstarted))
    ends { terminalPhase(phase) }
    timers(backoff, scheduleToClose, scheduleToStart, startToClose)
    unobservable(backoff)
    evidence { fact ->
        when (fact) {
            ProtocolFact.StatusScheduled -> status("statusScheduled")
            /* one arm per status fact, lines 473-479 */
            is ProtocolFact.StatusTimedOut -> status("statusTimedOut")
            ProtocolFact.AttemptCount -> read(attemptCount)
        }
    }
    steps {
        start runs ::startStep
        attemptResult runs ::protocolAttemptResultStep
        backoff runs ::backoffStep
        /* one line per action and timer, lines 486-493 */
    }
}
```

`starts` is the one state a run begins in; the other four fields take their defaults. `ends` is a
predicate written with the state as `this`, so `phase` means `this.phase`. `timers` are `system`
actions the machine owns, declared as values beside it (`val backoff = timer("backoff")`), and
`unobservable(backoff)` says the backoff records nothing, which the test layer must know. Each
`runs` line is typed by the action: `start runs ::startStep` accepts only a function of a
`ProtocolState` and three `Timeout`s, because `runs` is overloaded once per arity
(`Umpire.kt:419-429`):

```kotlin
    infix fun <A, B, C> Action3<A, B, C>.runs(fn: (S, A, B, C) -> List<Step<S, O, F>>) =
        register(this) { s, i -> fn(s, i[0] as A, i[1] as B, i[2] as C) }
```

The `evidence` block says how each fact is read back. For a workflow, a fact would name a history
event. A standalone activity has no history, so every status fact names a value of the `status`
field of `DescribeActivityExecution`, resolved against the realization's catalog, and the attempt
count is a `read` of the `attemptCount` observation declared earlier (`StandaloneActivity.kt:110-113`).
Because this `when` is a value with no `else`, adding a fact without deciding its evidence does
not compile.

The table is every state crossed with every action class, run through the step functions
(`Umpire.kt:319-321`; `computeTransitions` at `331` is a `TODO`):

```kotlin
    val table: Table<S, O, F> by lazy { Table(stateType.values, computeTransitions()) }
    val transitions: List<Transition<S, O, F>> get() = table.transitions
    val ends: List<S> get() = stateType.values.filter(endPredicate)
```

`by lazy` means the table is computed the first time something asks, which here is the pins.
`computeTransitions` is sketched: it would call each bound step function on each of the 288 states
with each input assignment and keep every returned step as a row. The `machine { }` builder runs when
the `val` initializes, so what it can detect (a timer with no step, a step bound twice) throws then,
with the machine's name in the message. None of this is compile time.

## Two levels and the refinement

The product machine says what the caller sees: nine phases, one timer, no backoff. The protocol
machine says how the server gets there: the backoff, the three deadlines, the attempt count, and the
`PauseRequested` phase. Properties are written against the product where possible and read on the
protocol through a map from protocol states to product states (`StandaloneActivity.kt:449-461`):

```kotlin
fun productOf(state: ProtocolState): ProductState = ProductState(
    when (state.phase) {
        Phase.Unstarted, Phase.Scheduled, Phase.BackingOff -> ProductPhase.Scheduled
        Phase.Started, Phase.PauseRequested -> ProductPhase.Started
        Phase.Paused -> ProductPhase.Paused
        Phase.CancelRequested -> ProductPhase.CancelRequested
        Phase.Completed -> ProductPhase.Completed
        Phase.Failed -> ProductPhase.Failed
        Phase.Canceled -> ProductPhase.Canceled
        Phase.Terminated -> ProductPhase.Terminated
        Phase.TimedOut -> ProductPhase.TimedOut
    },
)
```

`PauseRequested` maps to `Started`, not `Paused`, because the worker still holds the attempt and its
result lands where a started attempt's would. The other fields are dropped, which is the map saying
the product cannot see them.

The refinement rule: for every protocol row from `s` to `s'`, either `productOf(s) == productOf(s')`
(a stutter, the product sees nothing) or the product table has some row from `productOf(s)` to
`productOf(s')`, of any action class. The Lean checker is stricter and also matches outcome and
facts; under that rule the retryable-failure row from `Started` would need `StatusScheduled` among its
facts. This sample implements the mapped-states rule as SPEC.md states it.

One row by hand. Protocol: `Started` with one attempt, on `attemptResult(Failed(true))`, moves to
`BackingOff` recording `AttemptCount` (`StandaloneActivity.kt:359-360`). Mapped: `Started` to
`Scheduled`. Product: `attemptResultStep` from `Started` on `Failed(true)` goes to `Scheduled`
(`StandaloneActivity.kt:170`). A product row exists, so the protocol row is fine. A second row:
protocol `PauseRequested` on `control(Unpause)` moves to `Started`; both map to `Started`, so it is
a stutter.

The check is declared by `refines(activityProduct) via ::productOf` and runs inside the builder,
before the machine object exists (`Umpire.kt:407-408`):

```kotlin
    private fun <P : Any> deriveRefinement(target: Machine<P, *, *>, map: (S) -> P): Refinement<S, P> =
        TODO("enumerate stateType x steps, map each row, look it up in target.transitions")
```

The body is elided. When implemented, a rejected row makes the builder throw
`machine activityProtocol does not refine activityProduct: row ...` (`Umpire.kt:395-397`) the first
time the file loads, and the result is kept on the machine as `refinement.rows` and
`refinement.rejected` for the pins.

## Properties

A same-step claim names an action class and holds of the step that action produces
(`StandaloneActivity.kt:505-509`):

```kotlin
val completes = property("completes", activityProtocol) {
    attemptResult(AttemptResult.Completed) holds { step ->
        step.state.phase == Phase.Completed && ProtocolFact.StatusCompleted in step.facts
    }
}
```

`attemptResult(AttemptResult.Completed)` calls the action's `invoke` operator to make an action
class; `holds` is an infix function on it taking a predicate over one step. The machine is a
parameter of `property` so that `step` is typed `Step<ProtocolState, ProtocolOutcome, ProtocolFact>`
and `step.state.phase` compiles. `retryCompletes` fixes the whole state with data-class equality against `completedOnRetry`, a
`ProtocolState` with phase `Completed`, two attempts and every deadline unset
(`StandaloneActivity.kt:519-532`):

```kotlin
val retryCompletes = property("retryCompletes", activityProtocol) {
    attemptResult(AttemptResult.Completed) holds { step ->
        step.state == completedOnRetry && ProtocolFact.StatusCompleted in step.facts
    }
}
```

A transition claim names no action and holds of two consecutive steps. Both are on the product
machine and are read on the protocol through `productOf` (`StandaloneActivity.kt:500-502` and
`556-558`):

```kotlin
val terminalIsFinal = property("terminalIsFinal", activityProduct) {
    holds { before, after -> !productTerminal(before.state) || after.state.phase == before.state.phase }
}
/* ... */
val pausedIsNotDispatched = property("pausedIsNotDispatched", activityProduct) {
    holds { before, after -> before.state.phase != ProductPhase.Paused || after.state.phase != ProductPhase.Started }
}
```

The two forms are told apart by the lambda's arity: a bare `holds` with two parameters is the
transition claim (`Umpire.kt:473-482`). The same-step `holds` checks at init that the machine has a
step for the named action and throws `property completes: activityProtocol has no action '...'`
otherwise.

## Scenarios and limits

A scenario is a path: a start state and classed actions in order. `unset`, `expires` and
`unstarted` are private shorthands for `Timeout.Unset`, `Timeout.Expires` and
`ProtocolState(Phase.Unstarted)` (`StandaloneActivity.kt:578-580`). The retry path
(`StandaloneActivity.kt:593-603`):

```kotlin
val retriedThenCompleted = scenario("retriedThenCompleted", activityProtocol) {
    starts(unstarted)
    actions(
        start(unset, unset, unset),
        attemptStart,
        attemptResult(AttemptResult.Failed(retryable = true)),
        backoff,
        attemptStart,
        attemptResult(AttemptResult.Completed),
    )
}
```

A classed action is spelled by calling the action with its inputs; an action with no inputs, like
`attemptStart`, and a timer, like `backoff`, are listed bare. The pause path uses the caller's control
requests (`StandaloneActivity.kt:616-625`):

```kotlin
val pausedThenCompleted = scenario("pausedThenCompleted", activityProtocol) {
    starts(unstarted)
    actions(
        start(unset, unset, unset),
        control(Control.Pause),
        control(Control.Unpause),
        attemptStart,
        attemptResult(AttemptResult.Completed),
    )
}
```

`actions(...)` checks at init that each listed action has a step in the scenario's model
(`Umpire.kt:505-511`). Limits bound the search: how many steps a path may take, how many actions it
may list, and how many candidates to examine (`StandaloneActivity.kt:637-639`):

```kotlin
val three = limits("three", steps = 3, actions = 3, search = 4096)
val four = limits("four", steps = 4, actions = 4, search = 32768)
val six = limits("six", steps = 6, actions = 6, search = 262144)
```

## Queries

A query pairs a property with a scenario and limits. `find` asks whether the path reaches a step on
which a same-step claim holds; `verify` asks whether a claim holds on every trace the scenario admits
within the limits (`StandaloneActivity.kt:645` and `652`):

```kotlin
val retry = query("retry") { find(retryCompletes) on retriedThenCompleted within six }
```

```kotlin
val pauseHolds = query("pauseHolds") { verify(pausedIsNotDispatched) on pausedThenCompleted within six }
```

`on` checks at init that the property is about the scenario's machine or about the machine it
refines, and that the machine has the named action (`Umpire.kt:550-566`). A product property on a
protocol scenario passes because `activityProtocol` refines `activityProduct`. The search itself is
sketched; its result is a sealed `Outcome` with `Found`, `NotFound`, `VerifiedWithinLimits` and
`Violated` (`Umpire.kt:524-529`, `540-541`):

```kotlin
    /** Search over the scenario's automaton within the limits. In this sample it runs from the pins. */
    fun run(): Outcome = TODO("bounded search; a find returns the first witness, a verify the first counterexample")
```

When implemented, `run()` would walk the table from the start state following the scenario's
actions in order, stopping at the limits; a `find` returns the first path on which the claim holds,
a `verify` the first path on which it fails. It runs from the test, not at load time. An author whose
query fails sees a kotest assertion such as `Expected instance of umpire.Outcome$Found but was
NotFound`, with the query's name in the test title.

## Sets

A set groups find-queries for one purpose and says which parties the test drives and which it only
observes (`StandaloneActivity.kt:658-665`):

```kotlin
val standaloneActivityTests = set("standaloneActivityTests") {
    purpose = Purpose.Functional
    bind(caller to driven, worker to driven)
    queries(
        completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
        scheduleToStartTimeout, startToCloseTimeout,
    )
}
```

Driven means the test performs the party's actions; observed means a real deployment performs them
and the referee checks that what happened is a row the machine has. A functional set drives
everything. The canary (`standaloneActivityCanary`, `StandaloneActivity.kt:668-672`) runs against a
deployment with its own workers, so it binds `worker to observed` and lists `completion` and `cancel`. An
exploratory set (`standaloneActivityExploration`, `StandaloneActivity.kt:674-680`) names a machine,
`cover(Cover.Rows, Cover.Results, Cover.ClassMembers)` and a budget instead of queries. The Nexus Model's
functional set has `repeat = Repeat.Implementation` to run once per implementation switch; there is
no `repeat` here because standalone activities exist only in CHASM. `set` checks at init that a
functional set lists only find-queries and that an exploratory set names a machine and a budget.

## Composition with the worker

The protocol machine treats `workerStop` as a stutter: the activity cannot see its worker. To state
anything about the worker, the activity is composed with the worker machine from `Worker.kt`
(`Worker.kt:34-36`, `81-90`; `serveStep` at `76-78` returns a row only while polling):

```kotlin
    enum class Phase { Polling, Stopped }

    data class WorkerState(val phase: Phase)
```

```kotlin
    /** A worker has no natural end: it may be left polling or stopped. */
    val polling = machine<WorkerState, WorkerOutcome, WorkerFact>("polling") {
        entity = worker
        starts(WorkerState(Phase.Polling))
        ends { true }
        steps {
            workerStop runs ::stopStep
            workerResume runs ::resumeStep
            serve runs ::serveStep
        }
    }
```

`Worker` is a Kotlin `object`, a singleton used as a namespace, so the activity file writes
`Worker.polling`. The composition keeps only stop and serve, so a stopped worker cannot come back
and reply (`StandaloneActivity.kt:687-698`):

```kotlin
val activityWorker = Worker.polling.restrict("activityWorker", Worker.workerStop, Worker.serve)

data class StandaloneActivityState(val activity: ProtocolState, val worker: Worker.WorkerState)

val standaloneActivity = compose<StandaloneActivityState>("standaloneActivity") {
    val activity = member(StandaloneActivityState::activity, activityProtocol)
    val worker = member(StandaloneActivityState::worker, activityWorker)
    workerStop syncs activity[workerStop] with worker[Worker.workerStop]
    attemptStart syncs activity[attemptStart] with worker[Worker.serve]
    starts(StandaloneActivityState(unstarted, Worker.WorkerState(Worker.Phase.Polling)))
    ends { terminalPhase(this.activity.phase) }
}
```

`member` ties a field of the composite state to a machine; `StandaloneActivityState::activity` is a
property reference, so the field's type and the machine's state type must agree. A `syncs ... with`
line makes two member actions fire as one: the activity's `workerStop` is the worker's phase change,
and every dispatch is the worker serving. A composite step exists only when both member steps do, so
`attemptStart` has no row while the worker is stopped. The composite table is sketched
(`Umpire.kt:725-732`); actions no sync names are reached as `activity[start]`. The claim and the
path it is verified over (`StandaloneActivity.kt:701-703` and `710-721`):

```kotlin
val startedByPollingWorker = property("startedByPollingWorker", standaloneActivity) {
    attemptStart holds { step -> step.state.worker.phase == Worker.Phase.Polling }
}
/* ... */
val stoppedBeforeRetry = scenario("stoppedBeforeRetry", standaloneActivity) {
    val activity = standaloneActivity.member(StandaloneActivityState::activity)
    starts(StandaloneActivityState(unstarted, Worker.WorkerState(Worker.Phase.Polling)))
    actions(
        activity[start](unset, expires, unset),
        attemptStart,
        activity[attemptResult](AttemptResult.Failed(retryable = true)),
        activity[backoff],
        workerStop,
        activity[scheduleToStart],
    )
}
```

The first attempt is dispatched and fails retryably, the worker stops during the backoff, and the
schedule-to-start deadline fires on the retry that never comes. `stoppedWorkerStartsNothing`
(`StandaloneActivity.kt:723-725`) verifies the claim over that path within `six`: on every trace, an
attempt starts only while the worker polls, which the protocol machine alone cannot state. Because the path performs `attemptStart` once, the claim fires and is
checked rather than holding vacuously. No set names the composition; it exists to be verified over.

## Pins

The pins are kotest tests that fix numbers and rows the Model's identity rests on. State counts
(`Pins.kt:127-130`; the product's 9 and 5 are pinned the same way at `120-123`):

```kotlin
        test("twelve phases, three attempt counts and three deadlines, and the five terminal phases") {
            activityProtocol.table.states shouldHaveSize 12 * (activityAttemptBound + 1) * 2 * 2 * 2
            activityProtocol.ends shouldHaveSize 5 * (activityAttemptBound + 1) * 2 * 2 * 2
        }
```

This guards that `Finite.of` enumerates what the types say and that `ends` still names the five
terminal phases. One arm of one step function (`Pins.kt:132-134`):

```kotlin
        test("a canceled response is refused while no cancel was requested") {
            attemptResultStep(ActivityProductState(ActivityProductPhase.Started), AttemptResult.Canceled).shouldBeEmpty()
        }
```

The refinement (`Pins.kt:139-143`):

```kotlin
        test("the refinement onto the product machine is derived for every row and rejects none") {
            val refinement = checkNotNull(activityProtocol.refinement)
            refinement.rejected.shouldBeNull()
            refinement.rows shouldHaveSize activityProtocol.transitions.size
        }
```

The first test to touch any of these values loads the file's class, which runs every builder; a
builder failure surfaces as an `ExceptionInInitializerError` whose cause names the declaration.

## From model to running test

After this file, a checked Model would be lowered to Cases: one protobuf `Case` per query of a set,
carrying the path, the evidence to read at each step, and the claim as a Contract. A realization
binds each action class to a concrete call (a `StartActivityExecutionRequest`, a worker responding
with an `ApplicationFailure`), and the Go Testpilot runtime plays the Case against a server and
returns a verdict per Contract clause. No realization exists yet for standalone activities, and this
layer is not in the Kotlin sample: the file stops where the Lean file's `case` blocks begin, and
`README.md` says so.

## Gaps and gradual growth

The Model says "not modeled" in several places, and each is a place to grow. Reset and the heartbeat
timeout are named as absent in the file header. The protocol's `workerStop` is a stutter that keeps
the state and records nothing; the composition is where its effect lives. Many step-function arms
return the empty list: a canceled result from `Started`, any control from `Unstarted`, a pause from
`Paused`. Each empty list is a claim that the action is not enabled there, and a future query that
needs it turns the empty list into a row. Adding an action is: declare it as a `val` with its party
and inputs, write a step function, and add a `runs` line; the compiler then reports every `when` over
a domain that must grow, and the refinement check reports any row the product cannot account for.

## Mental model recap

- An action is a party's named side effect with finite inputs; one assignment of inputs is an action
  class, and faults and timers are actions too.
- A step function maps a state and inputs to zero or more steps; the empty list means not enabled.
- A machine is a state type plus step functions; the framework enumerates every state by reflection
  and builds the table of rows.
- The product machine says what the caller can see; the protocol machine says how the server gets
  there; `productOf` maps one onto the other and every protocol row must be a stutter or a product
  row.
- A property is a same-step claim about one action's step, or a transition claim about two
  consecutive steps.
- A scenario is a path; a query asks whether a property is found on it or verified over it, within
  limits; a set binds parties as driven or observed and groups queries for a purpose.
- A composition synchronizes two machines' actions so cross-entity claims can be stated.

## Where this implementation is weak

- **The framework core is sketched, not written.** `computeTransitions`, `deriveRefinement`,
  `Query.run` and the step functions of `composedSteps` are `TODO()`. The pins name what they would
  check but cannot run today, and the two composition queries have no table to search.
- **Nothing has been compiled.** The independent review found compile errors in `Umpire.kt` (a
  missing `override`, public `inline` functions reaching internal declarations, `lateinit` on a
  value-class property). They are fixed in the current files, but no toolchain has confirmed it, and
  a fluent reader should expect more of the same.
- **Most model-level checks run at load time, not compile time.** A property naming a missing
  action, a scenario listing one, a query pairing unrelated machines, a member action of the wrong
  machine: all are runtime `check` calls that fire when the first test loads the file. The compiler
  catches action arity, unresolved names, and missing `when` arms over domains and facts, and no
  more.
- **Eight `when`s over `Phase` end in `else -> emptyList()`.** Adding a phase is silent there. Only
  the `when`s whose arms all differ spell every phase out.

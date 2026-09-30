# Walkthrough: the standalone activity Model in Quint

## 1. What this is

Umpire is a model-based testing layer. A Model is the rulebook: what a Temporal feature may do,
written as small state machines. A test run against a real server is a playthrough, and the Go
runtime is the referee that reads the rulebook, drives the players (caller, worker) through a
chosen path, and checks that what the server recorded matches what the rulebook allows.

A standalone activity is a Temporal activity started directly with `StartActivityExecution`, with
no workflow around it. It writes no history events, so the only way to see what happened is to
read its status through `DescribeActivityExecution` or its result through `PollActivityExecution`.

This file, `standalone_activity.qnt`, is the rulebook for one such activity in Quint, a
specification language from Informal Systems with a simulator and a model checker. It defines
two machines (what the caller sees, and how the server gets there), the claims they make, the
paths the tests follow, and the data a Go runtime would need to realize them. It leans on
`umpire.qnt` for shared shapes and the worker machine.

## 2. The language in five minutes

Quint files hold modules. A module is a namespace with types, pure functions, state variables
and actions. Only the features this walkthrough uses are introduced here.

A sum type lists alternatives, each a constructor that may carry a payload. Constructors are
capitalized, and their names are shared across the whole module.

`standalone_activity.qnt:28-30`
```quint
  type AttemptResult = AttemptCompleted | AttemptFailed(bool) | AttemptCanceled

  type Control = Pause | Unpause | RequestCancel | Terminate
```

A `pure def` is a function of its arguments only. `match` branches on a constructor and binds
its payload; leaving out a constructor is a type error.

`standalone_activity.qnt:150-157`
```quint
  pure def steps(s: ProductState, a: Action): List[ProductStep] =
    match a {
      | AttemptStart => attemptStartStep(s)
      | AttemptResult(result) => attemptResultStep(s, result)
      | Control(control) => controlStep(s, control)
      | WorkerStop => workerStopStep(s)
      | TimeoutFired => timeoutStep(s)
    }
```

A record is a set of named fields, written `{ phase: Started, attempts: 1 }`. The spread form
`{ ...s, phase: Started }` copies `s` with one field changed, and `s.with("phase", Started)` is
the same thing as a method.

A `var` is a state variable. An `action` describes one transition: `x' = e` (read "x prime")
assigns the next value, `all { ... }` requires every line to hold, and a line that is plain
boolean is a guard. If any line is false the action is not enabled.

`standalone_activity.qnt:500-507`
```quint
  action fire(a: Action): bool = {
    val rows = steps(state, a)
    all {
      rows.length() > 0,
      state' = rows[0].state,
      last' = Some({ action: a, before: state, outcome: rows[0].outcome, facts: rows[0].facts }),
    }
  }
```

A `val` without `pure` reads state variables; it is an invariant when the checker is told so. A
`run` chains actions with `.then` and checks a boolean with `.expect`; `quint test` executes it.

`standalone_activity.qnt:639`
```quint
  run completion = completed.expect(found(completes))
```

Imports either flatten a module into this one (`import M.*`) or keep it behind a prefix
(`import M as p`, used as `p::name`). Sets are `Set(...)`, lists are `[...]`, and `tuples(A, B)`
is the set of all pairs.

## 3. Vocabulary: entities, parties, actions, inputs

An entity is the thing a machine is about. Here it is the activity, keyed by its id. A party is
who acts: the caller (the client that started the activity) and the worker. Timers belong to a
reserved party, the system.

`standalone_activity.qnt:26`
```quint
  pure val activity: Entity = { name: "activity", key: "activityId", refer: Set() }
```

There are five actions. `start` is the caller's request. `attemptStart` is the worker's poll
receiving the task. `attemptResult` is the worker reporting the attempt. `control` is one of the
four caller controls. `workerStop` is the worker going away. Quint has no action declaration
form, so the catalog is data: party, entity, protobuf schema, and examples that map an input
class to a concrete value.

`standalone_activity.qnt:44-47`
```quint
    "attemptResult" -> { party: Worker, creates: "", on: "activity",
      schema: "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest",
      examples: Set(("AttemptFailed(false)", "ApplicationFailure nonRetryable"),
                    ("AttemptFailed(true)", "ApplicationFailure retryable")) },
```

An action class is one action with one assignment of its finite inputs. `AttemptResult` has
three constructors but one carries a boolean, so it yields four classes; `Control` yields four.
The sets below are the class lists a machine enumerates over.

`standalone_activity.qnt:32-35`
```quint
  pure val attemptResults: Set[AttemptResult] =
    Set(AttemptCompleted, AttemptFailed(true), AttemptFailed(false), AttemptCanceled)

  pure val controls: Set[Control] = Set(Pause, Unpause, RequestCancel, Terminate)
```

A fault is not a special kind. `workerStop` is an ordinary action of the worker party with no
entity (`"workerStop" -> decl(Worker)` in the catalog), and the machines answer it like any
other action.

The payload `AttemptFailed(bool)` is how Quint spells the spec's `failed(retryable)`: one
constructor, two classes, and a `match` arm that binds the flag. The constructors carry an
`Attempt` prefix because the phases `Completed`, `Failed` and `Canceled` own the bare names in
the same module.

## 4. State

The product state is one field, the phase the caller reads through Describe: scheduled, started,
paused, cancel requested, and the five ends (completed, failed, canceled, terminated, timed out).
The protocol state adds the attempt count and the three deadlines the start request set, and its
phase type has three more members: unstarted, backing off, and pause requested.

`standalone_activity.qnt:76`
```quint
  type ProductState = { phase: ProductPhase }
```

`standalone_activity.qnt:265-271`
```quint
  type ProtocolState = {
    phase: Phase,
    attempts: int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
  }
```

Every field is finite so the machine has a finite table that can be enumerated, searched and
compared. A `Timeout` is `Unset | Expires`: the model does not care about durations, only
whether a deadline fires. The attempt count is an `int`, which is not finite, so the framework
bounds it and the successor saturates instead of wrapping.

`umpire.qnt:190-194`
```quint
  pure val attemptBound: int = 2

  pure def saturatingSucc(n: int): int = if (n < attemptBound) n + 1 else n

  pure val attemptCounts: Set[int] = 0.to(attemptBound)
```

Quint has no way to derive the set of values of a type, so each table lists its phases and
builds its state set by hand. Nine product states come from nine phases; 288 protocol states
come from 12 phases, three counts and three two-valued deadlines.

`standalone_activity.qnt:439-441`
```quint
  pure val states: Set[ProtocolState] =
    tuples(phases, attemptCounts, timeouts, timeouts, timeouts).map(((p, n, c, s, t)) =>
      { phase: p, attempts: n, scheduleToClose: c, scheduleToStart: s, startToClose: t })
```

The lambda `((p, n, c, s, t)) => ...` destructures each 5-tuple. A phase added to the `Phase`
type but not to `phases` is silently absent from every count, which is a real weakness noted in
the README.

## 5. Step functions

A step function takes a state and the action's inputs and returns a list of steps. Each step is
an outcome (`Accepted` or `NotFound`), the next state, and the facts the step records. An empty
list means the action is not enabled in that state.

`umpire.qnt:52-55`
```quint
  type Step[s, f] = { outcome: Delivery, state: s, facts: List[f] }

  pure def accepted(state: s, facts: List[f]): List[Step[s, f]] =
    [{ outcome: Accepted, state: state, facts: facts }]
```

`notFound(state)` is the other helper: outcome `NotFound`, the state kept, no facts.

The product's `attemptResultStep` first rejects every phase except `Started` and
`CancelRequested`. A completed attempt settles the activity. A failure splits on the flag: not
retryable settles it as failed; retryable is visible to the caller as a reschedule (Describe
reads SCHEDULED again), or as canceled if a cancel had been requested. A cancellation is honored
only where one was requested, so from `Started` it is not a row.

`standalone_activity.qnt:113-123`
```quint
  pure def attemptResultStep(s: ProductState, result: AttemptResult): List[ProductStep] =
    if (s.phase != Started and s.phase != CancelRequested) [] else
    match result {
      | AttemptCompleted => productStep(Completed, StatusCompleted)
      | AttemptFailed(retryable) =>
        if (not(retryable)) productStep(Failed, StatusFailed)
        else if (s.phase == Started) productStep(Scheduled, StatusScheduled)
        else productStep(Canceled, StatusCanceled)
      | AttemptCanceled =>
        if (s.phase == CancelRequested) productStep(Canceled, StatusCanceled) else []
    }
```

The protocol version dispatches on the phase first and the result second, because the three
source phases react differently to the same retryable failure. From `Started` it backs off,
recording only the attempt count. From `CancelRequested` it lands in `Canceled`, which is what
CHASM's state machine does. From `PauseRequested` it lands in `Paused`, mirroring
`TransitionAttemptFailedWhilePauseRequested`, where `TransitionRescheduled` is the plain case.

`standalone_activity.qnt:332-356`
```quint
  pure def protocolAttemptResultStep(s: ProtocolState, result: AttemptResult): List[ProtocolStep] =
    match s.phase {
      | Started => match result {
          | AttemptCompleted => moves(s, Completed, [StatusCompleted])
          | AttemptFailed(retryable) =>
            if (retryable) moves(s, BackingOff, [AttemptCount])
            else moves(s, Failed, [StatusFailed])
          | AttemptCanceled => []
        }
      | CancelRequested => match result {
          | AttemptCompleted => moves(s, Completed, [StatusCompleted])
          | AttemptFailed(retryable) =>
            if (retryable) moves(s, Canceled, [StatusCanceled])
            else moves(s, Failed, [StatusFailed])
          | AttemptCanceled => moves(s, Canceled, [StatusCanceled])
        }
      | PauseRequested => match result {
          | AttemptCompleted => moves(s, Completed, [StatusCompleted])
          | AttemptFailed(retryable) =>
            if (retryable) moves(s, Paused, [StatusPaused])
            else moves(s, Failed, [StatusFailed])
          | AttemptCanceled => []
        }
      | _ => []
    }
```

`moves` keeps every field but the phase. The outer `match` uses a wildcard `_` for the phases
that have no rows; the inner ones do not, so a new `AttemptResult` constructor is a type error
at each of the three inner matches until an arm is added. That is where exhaustiveness lives in
Quint: the typechecker, for any `match` without a wildcard.

## 6. The machine and its table

There is no `machine` keyword. The Lean block's fields become values in the table module: the
dispatch from action class to step function, the starts (`unstarted` is the record with phase
`Unstarted`, count 0 and every deadline `Unset`), the ends, the timers, and the timer that
records nothing.

`standalone_activity.qnt:447-450`
```quint
  pure val starts: Set[ProtocolState] = Set(unstarted)
  pure val ends: Set[ProtocolState] = states.filter(s => terminalPhase(s.phase))
  pure val timers: Set[Action] = Set(Backoff, ScheduleToClose, ScheduleToStart, StartToClose)
  pure val unobservable: Set[Action] = Set(Backoff)
```

The action class catalog is the union of every class of every action. With `steps` and `states`
it is the whole finite table: a row is a state, a class, and the step the class produces there.
Nothing builds the table ahead of time; a pin or a check walks `states` times `actionClasses`
and calls `steps` when it needs the rows, at `quint test` time.

`standalone_activity.qnt:456-461`
```quint
  pure val actionClasses: Set[Action] =
    deadlines.map(d => Start(d))
      .union(Set(AttemptStart, WorkerStop))
      .union(attemptResults.map(r => AttemptResult(r)))
      .union(controls.map(c => Control(c)))
      .union(timers)
```

The machine module gives the table state. `init` picks a start, `fire` takes the single row a
class has, and each spec action is `fire` of its class. `last` is the evidence of the last step:
which class fired, the state before, the outcome and the facts. Everything that later reads
"what just happened" reads `last`.

`standalone_activity.qnt:492-493`
```quint
  var state: ProtocolState
  var last: Option[Fired[Action, ProtocolState, ProtocolFact]]
```

`init` sets `state` to a member of `starts` and `last` to `None`. `step` is the exploratory
relation: any class, any state. `nondet a = oneOf(S)` picks any member of a set; the simulator
picks at random, the model checker considers all.

`standalone_activity.qnt:521-524`
```quint
  action step = {
    nondet a = oneOf(actionClasses)
    fire(a)
  }
```

Evidence lines map a fact to what the runtime reads. For a system with no history events every
fact is a Describe status read or the attempt count, so the mapping is a function from fact to a
name the Go side resolves against its observation catalog.

`standalone_activity.qnt:471-475`
```quint
      | StatusCanceled => "statusCanceled"
      | StatusTerminated => "statusTerminated"
      | StatusTimedOut(_) => "statusTimedOut"
      | AttemptCount => "attemptCount"
    }
```

## 7. Two levels and the refinement

The product machine says what an activity does as the caller sees it: nine phases, one timer,
no retries beyond a reschedule. The protocol machine says how the server gets there: the
backoff, the requested pause of a running attempt, the three deadlines, and the count. Claims
are written against the product where they are simple, and carried to the protocol by a map.

`productOf` reads a protocol state as a product state. Before the start, scheduled and backing
off all read as scheduled. A requested pause is still a running attempt, so it reads as started.

`standalone_activity.qnt:406-420`
```quint
  pure def productOf(s: ProtocolState): product::ProductState =
    { phase: match s.phase {
        | Unstarted => product::Scheduled
        | Scheduled => product::Scheduled
        | BackingOff => product::Scheduled
        | Started => product::Started
        | Paused => product::Paused
        | PauseRequested => product::Started
        | CancelRequested => product::CancelRequested
        | Completed => product::Completed
        | Failed => product::Failed
        | Canceled => product::Canceled
        | Terminated => product::Terminated
        | TimedOut => product::TimedOut
      } }
```

The refinement rule: for every protocol row from `s` to `s'`, either the mapped states are
equal (the product sees nothing, a stutter), or the product has some row, under any of its
classes, from `productOf(s)` to `productOf(s')`. The Lean checker is stricter and also matches
the outcome and requires the product row's facts to be among the protocol row's facts; under
that rule one protocol row here would need an extra `StatusScheduled` fact. This sample
implements the rule as the spec states it.

`umpire.qnt:163-167`
```quint
  pure def refinementRow(mappedBefore: p, mappedAfter: p, productRows: Set[Step[p, g]]): bool =
    or {
      mappedBefore == mappedAfter,
      productRows.exists(row => row.state == mappedAfter),
    }
```

`rejectedRows` (`umpire.qnt:171-177`) applies this to every state and class of the protocol
table, passing the product's rows from the mapped before-state, and returns the pairs that fail;
`refines` is that set being empty. Both take the step functions as arguments, which Quint allows
for pure definitions. The check runs twice. Over the whole table it is a pure value, evaluated by `quint test` in the pins.
Over a running trace it is the invariant `refinesProduct`, which `quint run` samples and `quint
verify` checks exhaustively to a depth.

`standalone_activity.qnt:529-535`
```quint
  val refinesProduct = match last {
    | None => true
    | Some(l) =>
      val before = productOf(l.before)
      refinementRow(before, productOf(state),
        rowsFrom(before, product::actionClasses, product::steps))
  }
```

One row by hand. Protocol `Started`, attempts 1, all deadlines unset, class
`AttemptResult(AttemptFailed(true))`. The protocol row moves to `BackingOff` with fact
`AttemptCount`. `productOf` reads the before-state as `Started` and the after-state as
`Scheduled`. They differ, so the product must have a row from `Started` to `Scheduled`: the
product's `attemptResultStep` with `AttemptFailed(true)` from `Started` is exactly that row
(section 5). Accepted. Before the spec revision the product had no such row and this was one of
three rejected rows.

## 8. Properties

A same-step claim names the action it is about and what holds of the step that action produced.
In Quint it is a record: `when` is "did that class fire last", `holds` is the claim. Read as an
invariant it is an implication; read by a find Query it is a conjunction, so a path that never
performs the action fails rather than passing vacuously.

`umpire.qnt:92-98`
```quint
  type Claim = { when: bool, holds: bool }

  pure def claim(when: bool, holds: bool): Claim = { when: when, holds: holds }

  pure def asInvariant(c: Claim): bool = c.when implies c.holds

  pure def found(c: Claim): bool = c.when and c.holds
```

`completes` fixes the phase and one fact. `retryCompletes` fixes the whole state, so every field
is named: it is only true on the second attempt.

`standalone_activity.qnt:551-565`
```quint
  val completes = claim(fired(AttemptResult(AttemptCompleted)),
    state.phase == Completed and recorded(StatusCompleted))

  // A non-retryable failure settles the activity as failed.
  val nonRetryableFails = claim(fired(AttemptResult(AttemptFailed(false))),
    state.phase == Failed and recorded(StatusFailed))

  /// Completed on the second attempt with no deadline set; a claim fixes one state.
  pure val completedOnRetry: ProtocolState =
    { phase: Completed, attempts: 2, scheduleToClose: Unset, scheduleToStart: Unset,
      startToClose: Unset }

  // The retried attempt completes: the count the retryable failure raised is two.
  val retryCompletes = claim(fired(AttemptResult(AttemptCompleted)),
    state == completedOnRetry and recorded(StatusCompleted))
```

A transition claim is about the step before and the step after. It is written once, on the
product, as a pure function of two product states, and read on each machine through `last.before`
(and on the protocol, through `productOf`).

`standalone_activity.qnt:187-192`
```quint
  pure def terminalIsFinalRow(before: ProductState, after: ProductState): bool =
    not(productTerminal(before)) or after.phase == before.phase

  // A paused activity is not dispatched: no step takes it straight to started.
  pure def pausedIsNotDispatchedRow(before: ProductState, after: ProductState): bool =
    before.phase == Paused implies after.phase != Started
```

The protocol's `val pausedIsNotDispatched` has the same shape as `refinesProduct` in section 7:
`None` before the first step, and otherwise the pure row function applied to `productOf` of
`last.before` and of `state`.

## 9. Scenarios and limits

A scenario is a `run`: `init`, then the classed actions in order. A classed action is the
action with its inputs spelled as constructor values, so `attemptResult(AttemptFailed(true))`
is one class and `control(Pause)` another.

`standalone_activity.qnt:611-614`
```quint
  run retriedThenCompleted =
    init.then(start(Unset, Unset, Unset)).then(attemptStart)
      .then(attemptResult(AttemptFailed(true))).then(backoff).then(attemptStart)
      .then(attemptResult(AttemptCompleted))
```

`standalone_activity.qnt:623-625`
```quint
  run pausedThenCompleted =
    init.then(start(Unset, Unset, Unset)).then(control(Pause)).then(control(Unpause))
      .then(attemptStart).then(attemptResult(AttemptCompleted))
```

Note that `backoff` appears in the list like any action. A timer is a system action the machine
owns, and a scenario says where it fires.

Limits bound a search: how many steps, how many actions, how many candidates. The file declares
`three`, `four` and `six` as `Limits` records, but Quint has no search over scenarios, so nothing
in the file consumes them. The exploratory `step` is bounded from the command line instead.

## 10. Queries

A find Query asks whether a scenario reaches a same-step claim. In Quint it is the scenario
followed by `.expect(found(claim))`: the claim's class fired last and the claim holds. A verify
Query asks whether a transition claim holds along the path, so the invariant is checked after
every step.

`standalone_activity.qnt:641`
```quint
  run retry = retriedThenCompleted.expect(found(retryCompletes))
```

`standalone_activity.qnt:655-660`
```quint
  run pauseHolds =
    init.then(start(Unset, Unset, Unset)).expect(pausedIsNotDispatched)
      .then(control(Pause)).expect(pausedIsNotDispatched)
      .then(control(Unpause)).expect(pausedIsNotDispatched)
      .then(attemptStart).expect(pausedIsNotDispatched)
      .then(attemptResult(AttemptCompleted)).expect(pausedIsNotDispatched)
```

There is no search. `quint test standalone_activity.qnt --main activityProtocol` executes each
`run` once and reports it. The Lean version searches all paths within the limits for one that
matches the scenario; here the scenario is the path. To check a claim over the whole machine
rather than one path, the author runs `quint run --invariant properties --max-steps 6` for
random exploration or `quint verify --invariant pausedIsNotDispatched --max-steps 6` for an
exhaustive bounded check with Apalache.

When a query fails, `quint test` prints the run's name, the file and line of the `expect` that
was false, and a seed to replay it. It does not say which clause failed; `--out-itf` writes the
trace so the author can read `last` in the failing state. The output shapes in the README are
illustrative, since nothing in this sample was executed.

## 11. Sets

A set groups queries into a test suite and binds parties. Driven means the Case performs that
party's actions; observed means it only reads what that party did and checks the machine allows
it. Quint cannot tie a query string to a `run`, so a set is data the Go side interprets.

`standalone_activity.qnt:664-676`
```quint
  pure val standaloneActivityTests: SetDecl =
    setOf(Functional, Set((Caller, Driven), (Worker, Driven)),
      Set("completion", "nonRetryableFailure", "retry", "cancel", "terminate", "pauseResume",
          "scheduleToStartTimeout", "startToCloseTimeout"))

  pure val standaloneActivityCanary: SetDecl =
    setOf(Canary, Set((Caller, Driven), (Worker, Observed)), Set("completion", "cancel"))

  pure val standaloneActivityExploration: SetDecl =
    setOf(Exploratory, Set((Caller, Driven), (Worker, Driven)), Set())
      .with("machine", "activityProtocol")
      .with("cover", Set(Rows, Results, ClassMembers))
      .with("budget", "four")
```

The functional set drives both parties through the eight find queries. The canary observes the
worker: it runs against a deployment whose own worker does the work. The exploratory set covers
the protocol table (rows, results, class members) under the `four` budget instead of naming
queries. There is no `repeat: implementation` because standalone activities exist only in the
CHASM implementation, so there is no switch to repeat over.

## 12. Composition with the worker

The worker is its own entity and machine, in `umpire.qnt`. It polls or is stopped; the worker
party stops and resumes it; a polling worker serves. It is a module with a constant, so each
composition instantiates its own copy keyed by task queue.

`umpire.qnt:252-254`
```quint
  /// A polling worker serves and keeps polling; a stopped one serves nothing.
  pure def serveStep(s: WorkerState): List[WorkerStep] =
    if (s.phase != Polling) [] else accepted(s, [])
```

The composition imports the protocol machine as `activity` and a worker instance as `worker`
(`import Worker(taskQueue = "activity") as worker from "./umpire"`), and writes each `sync` line
as one action that fires both members. `activityWorker` names the
restriction to stop and serve; the restriction itself is enforced only by which worker actions
`step` and the scenarios name, since `workerResume` never appears.

`standalone_activity.qnt:690-697`
```quint
  pure val activityWorker: Set[worker::Action] = Set(worker::WorkerStop, worker::Serve)

  action init = all { activity::init, worker::init }

  // sync: workerStop = activity.workerStop ∥ worker.workerStop
  action workerStop = all { activity::workerStop, worker::workerStop }
  // sync: attemptStart = activity.attemptStart ∥ worker.serve
  action attemptStart = all { activity::attemptStart, worker::serve }
```

The composed claim reads across entities: every dispatch leaves the worker polling. Because
`attemptStart` is synchronized with `serve`, and `serve` has no row while the worker is stopped,
no dispatch can happen after `workerStop`. The scenario dispatches once, fails the attempt
retryably, lets the backoff fire, then stops the worker before the retry and lets the
schedule-to-start deadline fire.

`standalone_activity.qnt:716-727`
```quint
  val startedByPollingWorker =
    claim(activity::fired(AttemptStart), worker::state.phase == worker::Polling)

  val pollingWorkerStarts = asInvariant(startedByPollingWorker)

  // The start request sets the schedule-to-start deadline; the first poll dispatches and the
  // attempt fails retryably; the backoff fires; the worker stops before the retry, so nothing
  // polls again and the deadline fires. The claim is exercised by the dispatch, not vacuous.
  run stoppedBeforeRetry =
    init.then(activity::start(Unset, Expires, Unset)).then(attemptStart)
      .then(activity::attemptResult(AttemptFailed(true))).then(activity::backoff)
      .then(workerStop).then(activity::scheduleToStart)
```

The verify query `stoppedWorkerStartsNothing` is this path with `.expect(pollingWorkerStarts)`
after each step, in the shape of `pauseHolds`, under limits `six`. An earlier version of the
scenario never performed `attemptStart`, so the claim's `when` was never true and the query
verified nothing; the dispatch on the second step now exercises it.

What it proves: on the protocol machine alone, `workerStop` is a stutter and the scenario orders
it by convention. Composed, the stop is a real phase change of another entity, and the claim
that no attempt starts while the worker is stopped is checked rather than assumed.

## 13. Pins

Pins are assertions about the tables, run by `quint test pins.qnt --main activityPins`. They
are the author's arithmetic, evaluated by the simulator; nothing here was run.

The state space pin guards the enumeration: if someone adds a phase and forgets `phases`, or
changes `attemptBound`, the count moves.

`pins.qnt:181-183`
```quint
    assert(states.size() == 12 * (attemptBound + 1) * 2 * 2 * 2),
    assert(ends.size() == 5 * (attemptBound + 1) * 2 * 2 * 2),
    assert(actionClasses.size() == 8 + 1 + 4 + 4 + 1 + 4),
```

A row pin guards one semantic decision. Here: a cancellation nobody requested is not a row, and a
retryable failure records only the attempt count.

`pins.qnt:193-196`
```quint
    assert(protocolAttemptResultStep(at(Started, 1), AttemptCanceled) == []),
    // A retryable failure backs off and only the attempt count records it.
    assert(protocolAttemptResultStep(at(Started, 1), AttemptFailed(true)) ==
      [{ outcome: Accepted, state: at(BackingOff, 1), facts: [AttemptCount] }]),
```

The refinement pin is the whole-table check; the same block also pins the mapping of a requested
pause to `Started`, the row the spec revision turned on.

`pins.qnt:215-217`
```quint
    assert(refinement),
    assert(rejectedRows(states, actionClasses, steps, productOf, product::actionClasses,
      product::steps) == Set()),
```

## 14. From model to running test

This sample stops at the trace. After it, the Umpire pipeline lowers each query into a Case: a
Program of instructions per party (start the activity, poll, respond, describe) and the evidence
declarations read off the machine's `evidence` lines along the path. A realization binds those
instructions to real API calls; none exists yet for standalone activities, in any sample or in
the Lean source. The Go Testpilot runtime would execute the Case against a server, drive the
parties the set marks as driven, observe the rest, and compare what Describe returned against
the facts each step recorded. The verdict is per query: found, verified within limits, or a named
gap. None of that layer is in this sample. What Quint contributes is the ITF JSON trace `quint
test --out-itf` writes per `run`, with `state` and `last` in every step; the Go adapter to read it
does not exist, and the README lists what does.

## 15. Gaps and gradual growth

The Model says "not modeled" in several places on purpose. Reset is deferred. Heartbeat timeout
is absent. The `workerStop` step on the protocol machine is a stutter: it keeps the state and
records nothing, and the Run confirms it by the evidence of the step after it. The product does
not see `workerStop` at all, because a kept state with no facts would be indistinguishable from a
stutter and the refinement would read every stutter as that step.

`standalone_activity.qnt:141-142`
```quint
  /// Invisible, as in the Nexus Model: a kept state with no facts would read as a stutter.
  pure def workerStopStep(s: ProductState): List[ProductStep] = []
```

Empty steps are the growth mechanism. An action that is not enabled in a phase returns `[]`,
which costs nothing and asserts nothing. Adding heartbeat later means: one constructor in
`Action`, one step function, one arm in `steps`, one class in `actionClasses`, and the counts in
the pins move. Every `match` without a wildcard fails to typecheck until the new arm exists,
which is the language's checklist for the author. The catalog in `activityVocabulary` and the
evidence function grow by one line each.

## 16. Mental model recap

- Two machines: the product is what the caller sees through Describe, the protocol is how the
  server gets there. Claims are written on the simpler one and read on the other via `productOf`.
- A step function is `(state, inputs) -> list of steps`; empty means not enabled. Steps carry an
  outcome, a next state and facts.
- The finite table is `states` times `actionClasses` fed through `steps`; Quint computes it on
  demand as pure values.
- `last` is the evidence of the last step and is what properties, the refinement invariant and a
  future Go adapter all read.
- A same-step claim is a `when`/`holds` pair; a transition claim is a pure function of before and
  after.
- A scenario is a `run`; a find query is the run plus `expect(found(...))`; a verify query is
  `expect(invariant)` after every step.
- The refinement is a pure check over the table and the same predicate as a runtime invariant.
- Sets, limits, schemas and parties are data Quint types but does not interpret.

## 17. Where this implementation is weak

- The independent review found three name collisions that `quint parse` rejects: timer labels
  reused between `TimeoutType` and `Action`, the product timer constructor `Timeout` beside the
  `Timeout` type, and `val terminalIsFinal` shadowing a star-imported pure def of the same name.
  All three are fixed in the files this walkthrough quotes (`ByScheduleToStart`, `TimeoutFired`,
  `terminalIsFinalRow`), but the fix was made by reading, not by running the tool.
- Nothing has been executed. The 288 states, 120 ends and 22 classes are hand arithmetic, and
  the reviewer recomputed the Nexus numbers by hand and found them right; the Activity ones were
  not independently checked.
- The refinement rule implemented is the spec's mapped-states rule. The real Lean checker also
  compares outcomes and facts, and under it one protocol row here lacks a `StatusScheduled` fact.
- `activityWorker` and the `Limits` values are declared but consumed by nothing; restriction
  and bounds are convention and command-line flags, not something the file enforces.

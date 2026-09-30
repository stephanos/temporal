# Walkthrough: the standalone activity Model in Nim

## What this is

Umpire is a model-based testing framework. A Model is a rulebook: it says which moves each player
may make from each position and what the referee should observe afterwards. A test is one
playthrough of that rulebook against a real Temporal server, and the referee checks every move
against the table the rulebook generates.

A standalone activity is a Temporal activity started directly with `StartActivityExecution`, with
no workflow around it. It writes no history events, so the only way to see it is to read its
status through `DescribeActivityExecution` or its result through `PollActivityExecution`.

`standalone_activity.nim` is the rulebook for one such activity. It declares the vocabulary, two
machines (what the caller sees and how the server gets there), the claims the machines make, the
paths the tests run, and the sets that group them. `umpire.nim` is the framework it binds to, and
`worker.nim` is the worker entity it composes with.

## The language in five minutes

Nim is a compiled language with Python-like indentation and a C-like type system. These are the
features this file relies on.

**Enums and object variants.** An `enum` lists names. An object with `case kind` is a tagged union:
the fields after each `of` exist only when `kind` has that value. `AttemptResult` below is one.

**`case` statements are exhaustive.** A `case` over an enum must list every member or have an
`else`; the compiler refuses otherwise. The step functions lean on this.

```nim
  case state.phase
  of started, cancelRequested:
```

**`seq[T]`, `@[]`, `proc`, `*`, `{.pure.}`.** A `seq` is a growable list, `@[x, y]` builds one and
`@[]` is empty. `proc` declares a function; a trailing `*` exports a name from the module.
`{.pure.}` on an enum keeps its members out of the top-level scope, so they are written
`AttemptResultKind.failed`.

**Colon blocks.** A call followed by an indented block of `key: value` lines is ordinary Nim syntax.
The framework's macros receive that block as a parse tree, which is why the Model reads like
configuration.

```nim
machine activityProduct:
  `for`: activity
```

**Backticks.** `for`, `in`, `from`, `when` and `bind` are Nim keywords; backticks let a keyword be
used as a name, which is how the spec's field names survive.

**`const` and compile-time evaluation.** A `const` is computed while the program compiles, by the
compiler's built-in interpreter (the VM). Every machine, property and query in this file is a
`const`, so the tables are built and searched at compile time.

**Macros.** A `macro` is a proc that runs at compile time, receives code as a tree, and returns
code. `machine`, `property`, `finite` and the rest are macros in `umpire.nim`.

## Vocabulary: entities, parties, actions, inputs

An entity is what a machine is about. `entity activity:` with `key: activityId`
(`standalone_activity.nim:18-19`) says the recorded `activityId` field identifies one instance.

A party is who performs an action. The framework fixes the list as an enum in `umpire.nim`
(`caller`, `handler`, `worker`, `network`, `operator`); this Model uses `caller` and `worker`, and
`system` is reserved for timers.

An action is a named side effect of a party, with typed finite inputs. The `start` action is the
caller's request; its three inputs say whether each deadline will fire.

`standalone_activity.nim:61-68`

```nim
action start:
  party: caller
  creates: activity
  schema: temporal.api.workflowservice.v1.StartActivityExecutionRequest
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout
```

`attemptResult` is the worker's response to a task. Its one input is an `AttemptResult`, which is
where payloads come in.

`standalone_activity.nim:76-78` and `:82-86` (the three-line `schema:` naming the Respond
request messages is elided)

```nim
action attemptResult:
  party: worker
  on: activity
  input:
    result: AttemptResult
  examples:
    failed(retryable = false) -> ApplicationFailure nonRetryable
    failed(retryable = true) -> ApplicationFailure retryable
```

The `action` macro emits no Nim symbol. It records the action's inputs in a compile-time registry
(`std/macrocache`, a table that survives across modules), and the `machine` macro reads that
registry later to know which inputs to enumerate.

An action class is one action with one assignment of its inputs. `AttemptResult` has three
constructors, and `failed` carries a boolean, so the action has four classes: `completed`,
`failed(false)`, `failed(true)`, `canceled`. The `Control` enum has four members, so `control` has
four classes too. `start` has two choices per deadline, eight classes.

`standalone_activity.nim:26-35`

```nim
type
  AttemptResultKind* {.pure.} = enum
    completed, failed, canceled

  AttemptResult* = object
    case kind*: AttemptResultKind
    of AttemptResultKind.failed:
      retryable*: bool
    else:
      discard
```

The constructors are plain values and one proc, so a class is spelled the way Lean spells it:
`attemptResult(failed(true))`.

`standalone_activity.nim:49-54`

```nim
const
  completed* = AttemptResult(kind: AttemptResultKind.completed)
  canceled* = AttemptResult(kind: AttemptResultKind.canceled)

proc failed*(retryable: bool): AttemptResult =
  AttemptResult(kind: AttemptResultKind.failed, retryable: retryable)
```

Reviewer's note on this snippet: `completed` and `canceled` are also members of the non-pure
`ProductPhase` and `Phase` enums declared later in the same module. Nim puts non-pure enum members
and consts in one flat scope, so this is a redefinition error as written. See the last section.

A fault is just an action. The worker stopping is declared like any other, with a party and no
entity, because nothing recorded names the activity when it happens.

`standalone_activity.nim:99-100`

```nim
action workerStop:
  party: worker
```

## State

Each machine has a state type. The product state is one field; the protocol state adds the
attempt count and the three deadlines.

`standalone_activity.nim:129-130`

```nim
  ProductState* = object
    phase*: ProductPhase
```

`standalone_activity.nim:262-272`

```nim
const attemptBound* = 2

type
  Attempts* = range[0 .. attemptBound]

  ProtocolState* = object
    phase*: Phase
    attempts*: Attempts
    scheduleToClose*: Timeout
    scheduleToStart*: Timeout
    startToClose*: Timeout
```

Every field is finite because the framework builds a table by trying every state. An enum is
finite by definition. `range[0 .. 2]` is a Nim integer type restricted to three values, so it is
finite too. The count saturates at the bound rather than growing without limit.

`umpire.nim:145-147`

```nim
proc saturatingSucc*[T: Ordinal](x: T): T =
  if x == high(T): x else: succ(x)
```

Enumeration of an enum or range is `low..high`, one generic proc in `umpire.nim`. Enumeration of
an object is derived by the `finite` macro, which reads the object's field list and generates
nested loops, one per field, so the result is the cartesian product of the fields' enumerations.

The Model invokes it as `finite ProtocolState` (`standalone_activity.nim:291`). For
`ProtocolState` that is 12 phases times 3 counts times 2 times 2 times 2 deadlines, 288
states. For `ProductState` it is 9. For a variant like `AttemptResult`, the macro emits a `case`
over the kind and loops over each branch's fields, which yields four members.

## Step functions

A step function takes a state and the action's inputs and returns a list of steps. Each step
(`Step[S, O, F]` in `umpire.nim:38-43`) is an outcome, a next state, and the facts the referee
should observe. An empty list means the action is not enabled in that state.

The product `attemptResultStep`, arm by arm.

`standalone_activity.nim:166-180`

```nim
proc attemptResultStep*(state: ProductState, result: AttemptResult): seq[ProductStep] =
  case state.phase
  of started, cancelRequested:
    case result.kind
    of AttemptResultKind.completed: productStep(ProductPhase.completed, statusCompleted)
    of AttemptResultKind.failed:
      if not result.retryable: productStep(ProductPhase.failed, statusFailed)
      elif state.phase == started: productStep(scheduled, statusScheduled)
      else: productStep(ProductPhase.canceled, statusCanceled)
    of AttemptResultKind.canceled:
      if state.phase == cancelRequested: productStep(ProductPhase.canceled, statusCanceled)
      else: @[]
  of scheduled, paused, ProductPhase.completed, ProductPhase.failed, ProductPhase.canceled,
      terminated, timedOut:
    @[]
```

Only a started or cancel-requested activity has a running attempt, so only those two phases
react. A completed result completes it. A non-retryable failure fails it. A retryable failure from
`started` puts it back to `scheduled`, because Describe reads SCHEDULED again; from
`cancelRequested` it cancels, matching the CHASM state machine where CANCEL_REQUESTED is a source
of Canceled. A canceled result is honored only where cancellation was requested. Every other phase
returns the empty list. The `of` on the last arm lists all remaining phases; leaving one out is a
compile error.

The protocol version dispatches on phase first because three source phases react differently to
the same retryable failure.

`standalone_activity.nim:345-370`

```nim
proc protocolAttemptResultStep*(state: ProtocolState, result: AttemptResult): seq[ProtocolStep] =
  case state.phase
  of Phase.started:
    case result.kind
    of AttemptResultKind.completed: moves(state, Phase.completed, @[statusCompleted])
    of AttemptResultKind.failed:
      if result.retryable: moves(state, backingOff, @[attemptCount])
      else: moves(state, Phase.failed, @[statusFailed])
    of AttemptResultKind.canceled: @[]
  of Phase.cancelRequested:
    case result.kind
    of AttemptResultKind.completed: moves(state, Phase.completed, @[statusCompleted])
    of AttemptResultKind.failed:
      if result.retryable: moves(state, Phase.canceled, @[statusCanceled])
      else: moves(state, Phase.failed, @[statusFailed])
    of AttemptResultKind.canceled: moves(state, Phase.canceled, @[statusCanceled])
  of pauseRequested:
    case result.kind
    of AttemptResultKind.completed: moves(state, Phase.completed, @[statusCompleted])
    of AttemptResultKind.failed:
      if result.retryable: moves(state, Phase.paused, @[statusPaused])
      else: moves(state, Phase.failed, @[statusFailed])
    of AttemptResultKind.canceled: @[]
  of unstarted, Phase.scheduled, backingOff, Phase.paused, Phase.completed, Phase.failed,
      Phase.canceled, Phase.terminated, Phase.timedOut:
    @[]
```

From `started`, a retryable failure goes to `backingOff` and records only the attempt count. That
is `TransitionRescheduled` in `statemachine.go`. From `cancelRequested` it cancels. From
`pauseRequested` it pauses, which is `TransitionAttemptFailedWhilePauseRequested`. `moves` is a
helper that copies the state, changes the phase, and wraps it in a single accepted step.

Exhaustiveness is enforced by the compiler: both the outer `case` over `Phase` and each inner
`case` over `AttemptResultKind` must cover every member. Adding a phase to `Phase` and forgetting
this proc is a build error naming the missing member.

## The machine and its table

The `machine` block binds the pieces together.

`standalone_activity.nim:440-448` and `:460-469` (the ten `evidence:` lines between them are
elided)

```nim
machine activityProtocol:
  `for`: activity
  state: ProtocolState
  refines: activityProduct
  map: productOf
  starts: [unstarted]
  ends: [completed, failed, canceled, terminated, timedOut]
  timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
  unobservable: [backoff]
  steps:
    start: startStep
    attemptStart: protocolAttemptStartStep
    attemptResult: protocolAttemptResultStep
    control: protocolControlStep
    workerStop: protocolWorkerStopStep
    backoff: backoffStep
    scheduleToClose: scheduleToCloseStep
    scheduleToStart: scheduleToStartStep
    startToClose: startToCloseStep
```

`starts` names a phase; the framework picks the state at that phase with every other field at its
first value. `ends` names phases; every state at one of them is an end, so the protocol has 120
ends (5 phases times 24 field combinations). `timers` are system actions this machine owns; they
take no input and need no `action` declaration. `unobservable` marks the backoff timer as one that
records nothing.

`evidence` maps a fact name to the recorded thing the referee reads, one line per fact such as
`statusPaused: statusPaused`. For an activity nothing is a history event; every line names a
Describe status or the `attemptCount` observation, and the names match on both sides.

The macro turns the `steps` lines into a table at compile time. For each `action: stepFn` line it
looks up the action's inputs in the registry, then emits a loop over every state and every input
assignment, calling the step function and adding a row for each step returned
(`umpire.nim:455-470`). A step naming an action that is neither declared nor one of this machine's
timers is an error reported at that line:

```
standalone_activity.nim(463, 5) Error: step names an undeclared action `attemptResutl`; declare it with `action`, or list it under `timers:`
```

The emitted code is a single `const activityProtocol = block: ...`, so the table exists when
compilation ends and nothing is computed at test time.

## Two levels and the refinement

The product machine says what the caller sees: nine phases, one field. The protocol machine says
how the server gets there: the backoff between attempts, the pause that is only requested while an
attempt runs, the attempt count, and which deadline fired. Claims are written against whichever
level can state them, and a product claim is carried to the protocol by a map.

`standalone_activity.nim:428-438`

```nim
proc productOf*(state: ProtocolState): ProductState =
  ProductState(phase: (case state.phase
    of unstarted, Phase.scheduled, backingOff: ProductPhase.scheduled
    of Phase.started, pauseRequested: ProductPhase.started
    of Phase.paused: ProductPhase.paused
    of Phase.cancelRequested: ProductPhase.cancelRequested
    of Phase.completed: ProductPhase.completed
    of Phase.failed: ProductPhase.failed
    of Phase.canceled: ProductPhase.canceled
    of Phase.terminated: ProductPhase.terminated
    of Phase.timedOut: ProductPhase.timedOut))
```

`pauseRequested` maps to `started`, not `paused`. The worker still holds the attempt, so every
answer it can give is a product row from `started`, and the request itself is a product stutter.
The first version of the spec mapped it to `paused`, and three protocol rows then had no product
counterpart.

The refinement rule: for every protocol row from `s` to `s'`, either `productOf(s)` equals
`productOf(s')` (a stutter, the product did not move), or the product table has some row from
`productOf(s)` to `productOf(s')`, under any action class.

`umpire.nim:184-196` (inside a loop over `protocol.table.rows`, with `source` and `target` the
mapped ends of the row)

```nim
    if source == target:
      result.rows.add (rowKey, none(ActionKey))
      continue
    let candidates = product.table.rows.filterIt(it.source == source and it.step.state == target)
    let same = candidates.filterIt(it.action == row.action)
    if same.len > 0:
      result.rows.add (rowKey, some(same[0].action))
    elif candidates.len > 0:
      result.rows.add (rowKey, some(candidates[0].action))
    else:
      result.rejected = some("protocol row " & rowKey & " maps " & key(source) & " -> " &
        key(target) & ", which is no product step and no stutter")
      return
```

This runs inside the `const` while the module compiles. The result is then handed to `admit`, a
macro with a `static` parameter: the VM evaluates the value first, and the macro reports a
rejection at the `refines:` line the author wrote.

One row by hand. Protocol state `started, attempts 1, all unset`, action
`attemptResult-failed-true`. The step function returns `backingOff, attempts 1`. The map sends
`started` to product `started` and `backingOff` to product `scheduled`. Those differ, so it is not
a stutter. The product table has a row `started -> scheduled` under `attemptResult-failed-true`
(the retry is visible in the product for activities). The class matches, so the row is recorded as
that step and the refinement continues.

The real Lean checker is stricter: the matching product row must also have the same outcome, and
its facts must be among the protocol row's facts. Under that rule the row above would need the
protocol to record `statusScheduled` as well as `attemptCount`. This sample implements the
mapped-states rule as the spec states it, so the row passes here.

## Properties

A same-step claim names the action it is about with `when:` and a predicate over the step that
action produced.

`standalone_activity.nim:484-487`

```nim
property completes:
  machine: activityProtocol
  `when`: attemptResult(completed)
  holds: step => step.state.phase == Phase.completed and statusCompleted in step.facts
```

`step => ...` is Nim's arrow syntax for a lambda. The `property` macro rewrites it into a proc
typed by the machine's step type, so `step.state.phase` type-checks without an annotation.
`retryCompletes` fixes the whole state, including the count.

`standalone_activity.nim:501-504` (`completedOnRetry` is a const naming the state
`completed, attempts 2, all deadlines unset`)

```nim
property retryCompletes:
  machine: activityProtocol
  `when`: attemptResult(completed)
  holds: step => step.state == completedOnRetry and statusCompleted in step.facts
```

A transition claim has no `when:` and takes two steps, before and after.

`standalone_activity.nim:528-531`

```nim
property pausedIsNotDispatched:
  machine: activityProduct
  holds: (before, after) =>
    before.state.phase != ProductPhase.paused or after.state.phase != ProductPhase.started
```

Both transition claims are on the product machine. `terminalIsFinal` says no step leaves a
terminal phase; `pausedIsNotDispatched` says nothing goes from paused straight to started.

## Scenarios and limits

A scenario is a path: a start phase and a list of classed actions in order.

`standalone_activity.nim:564-568`

```nim
scenario retriedThenCompleted:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), attemptStart, attemptResult(failed(true)), backoff,
    attemptStart, attemptResult(completed)]
```

`standalone_activity.nim:583-587`

```nim
scenario pausedThenCompleted:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), control(pause), control(unpause), attemptStart,
    attemptResult(completed)]
```

A classed action is spelled as a call. The macro turns `attemptResult(failed(true))` into the key
`"attemptResult-failed-true"` by emitting `key(failed(true))`, so the argument is type-checked by
the compiler. A bare name like `backoff` is a timer or an input-free action and becomes its own
key; the reviewer notes that bare names are not checked against the registry, so a misspelled
timer surfaces only when the search finds no row.

Limits bound the search: steps on the path, distinct actions, and candidates visited. `limits six:`
(`standalone_activity.nim:611-614`) is six steps, six actions and a budget of 262144 candidates,
sized for the two six-step scenarios above.

## Queries

A `find` query asks whether a same-step claim holds somewhere on a scenario; the path found is
what a set realizes as a test. A `verify` query asks whether a transition claim holds on every
consecutive pair of steps along the scenario; it is never realized.

`standalone_activity.nim:632-635`

```nim
query retry:
  find: retryCompletes
  `in`: retriedThenCompleted
  limits: six
```

`standalone_activity.nim:668-671`

```nim
query pauseHolds:
  verify: pausedIsNotDispatched
  `in`: pausedThenCompleted
  limits: six
```

The `query` macro emits `const retry = search(...)` and then `admit(retry, node)`. `search` follows
the scenario's actions from the start through the table, branching where an action has more than
one row, then checks the claim on the resulting paths. It runs in the VM, so a query that fails
stops the build with a message at the `find:` line:

```
standalone_activity.nim(633, 3) Error: query `retry`: retryCompletes never holds on retriedThenCompleted
```

Two reviewer findings apply here. First, `search` is a path walk, not a search: it never explores
actions the scenario did not list, and `limits.actions` is unused. Second, `pauseHolds` and
`terminalHolds` bind a product claim to a protocol scenario, and the sample passes the property's
machine to `search` with no map-through, so these two queries would not type-check as written. A
working version would walk the protocol path and apply `productOf` to each step before evaluating
the claim. The README's compile-time claim for `verify` is therefore not delivered by this code.

## Sets

A set groups queries and says which parties the test drives and which it only observes.

`standalone_activity.nim:678-684`

```nim
set standaloneActivityTests:
  purpose: functional
  `bind`:
    caller: driven
    worker: driven
  queries: [completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
    scheduleToStartTimeout, startToCloseTimeout]
```

The functional set drives both the caller and the worker. There is no `repeat: implementation`
line, unlike the Nexus set, because standalone activities exist only under CHASM and there is no
second implementation to run against.

The canary set (`standalone_activity.nim:691-696`) binds the worker as `observed` and lists
`completion` and `cancel`: a deployment runs its own worker, and the referee checks that whatever
result occurred is one the machine allows.

The exploratory set names a machine and a coverage target instead of queries: within budget
`four`, cover the rows, the results those rows reach, and the members of the classes.

## Composition with the worker

The protocol machine's `workerStop` is a stutter: the activity cannot see its own worker. To state
a claim across both, the Model composes with a worker machine from `worker.nim`.

`worker.nim:83-86`

```nim
  steps:
    workerStop: stopStep
    workerResume: resumeStep
    serve: serveStep
```

The `polling` machine has two phases, `polling` and `stopped`, and no natural end. `serveStep`
returns a row only from `polling`.

The activity's view of that worker, `activityWorker` (`standalone_activity.nim:717-719`), is
`polling` restricted to `workerStop` and `serve`. A resume would stay executable on its own and
admit stop, resume, dispatch, which is not the case being asked about.

The composition names members and synchronizes actions. A `sync` line says two member actions fire
as one row: the activity's `workerStop` and the worker's `workerStop` are one event, and the
activity's `attemptStart` happens only as the worker's `serve`.

`standalone_activity.nim:727-738`

```nim
compose standaloneActivity:
  `for`: [activity, Worker.worker]
  state: StandaloneActivityState
  members:
    activity: activityProtocol
    worker: activityWorker
  sync:
    workerStop: activity.workerStop || worker.workerStop
    attemptStart: activity.attemptStart || worker.serve
  starts: [activity.unstarted, worker.polling]
  ends: [activity.completed, activity.failed, activity.canceled, activity.terminated,
    activity.timedOut]
```

`standalone_activity.nim:741-744`

```nim
property startedByPollingWorker:
  machine: standaloneActivity
  `when`: attemptStart
  holds: step => step.state.worker.phase == Worker.polling
```

The claim: every dispatch leaves the worker polling. Because `serve` has no row from `stopped` and
`attemptStart` fires only with `serve`, no attempt can start after the worker stops. The query
`stoppedWorkerStartsNothing` verifies it on `stoppedBeforeRetry`
(`standalone_activity.nim:749-754`): the first attempt is dispatched and fails retryably, the
worker stops during the backoff, and the schedule-to-start deadline fires. The path performs one
`attemptStart`, so the claim is exercised on a real step and not vacuously true. An earlier version
of the spec used a path with no dispatch at all.

The `product` proc that would build the composed table is a signature and a comment in
`umpire.nim`, and the composition macro has parsing slips on dotted names such as
`Worker.polling`. The composition is an honest sketch: it shows what is declared, not code that
runs.

## Pins

`pins.nim` restates what the tables say, once at compile time and once as unit tests.

`pins.nim:25-26`

```nim
  doAssert Activity.activityProduct.table.states.len == 9
  doAssert Activity.activityProtocol.table.states.len == 12 * 3 * 8
```

Inside a `static:` block, so a change to `Phase` that alters the state count fails the build.

`pins.nim:118-120`

```nim
  test "a cancellation nobody requested has no row":
    check protocolAttemptResultStep(
      Activity.ProtocolState(phase: Activity.Phase.started), Activity.canceled) == @[]
```

Guards the arm that says a worker cannot cancel an attempt nobody asked to cancel.

`pins.nim:121-122`

```nim
  test "the refinement passes":
    check activityProtocol.refinement.get.rejected.isNone
```

Guards the whole product-protocol relationship. If a protocol step is added that the product cannot
account for, this and the `refines:` line both report it.

## From model to running test

After this file, the pipeline would lower each `find` query's witness path into a Case: a protobuf
document listing the actions to perform, the party that performs each, and the evidence to read
after each step. A realization maps each action class to a concrete client call. No realization
exists yet for standalone activities. The Go Testpilot runtime would execute the Case against a
server, read Describe after each step, and produce a verdict per step. None of that layer is in
this sample; the sample stops where the tables, refinement and queries exist.

## Gaps and gradual growth

The Model says "not modeled" in several places, and each is a deliberate seam.

- Reset is deferred and has no action. Heartbeat timeout has no timer.
- `workerStop` is a stutter on the protocol machine and an empty step on the product machine. The
  composition is where it gains meaning.
- A canceled result from `started` returns the empty list. An unrequested cancel is not a row the
  server admits, so the Model does not either.
- The backoff timer is `unobservable`: it records nothing, and a test confirms it by the step after.

Adding an action is mechanical: declare it with `action`, write one step function per machine
with an exhaustive `case`, add the `steps:` line, and let the compiler and the refinement tell you
what else has to change. A new phase forces every `case` over `Phase` to be revisited, which is
the point.

## Mental model recap

- An action class is one action with one input assignment; the table has a row per state per
  class that the step function enables.
- A step function returns a list; empty means not enabled. Outcome, next state, facts.
- Finite state types are enumerated by the compiler's knowledge of enums and ranges, and by
  `finite` for objects.
- The product machine is what the caller sees; the protocol machine is how the server gets there;
  `productOf` connects them and the refinement checks every protocol row against it.
- Same-step claims are found and realized; transition claims are verified and never realized.
- Everything is a `const`, so tables, refinement and searches run while the module compiles, and
  errors land on the author's line.
- Facts for an activity are Describe reads, not history events.

## Where this implementation is weak

From the independent review at `cmp/.eval/nim.md`, before trusting the code:

- **Neither Model module compiles as written.** Nim's flat module scope makes roughly twenty
  same-module name collisions: the `completed` and `canceled` constructor consts against the
  `Phase` and `ProductPhase` members, the eight `status*` fact consts against the non-pure
  `ProductFact` members, and the spec's `terminated`, `completed`, `terminate` declarations. The
  README lists only four. Fixes are `{.pure.}` on the product enums or a module split.
- **The two `verify` queries have no implementation.** `terminalHolds` and `pauseHolds` bind a
  product claim to a protocol scenario, and `search` has no map-through, so they would not
  type-check. The pins asserting they pass therefore pin nothing yet.
- **`search` walks the scenario path only.** It does not explore other actions, `limits.actions`
  is unread, and the README's talk of candidate counts describes an engine this code is not.
- **Composition is a sketch.** `product` is elided, `memberPhases` returns an empty list, and the
  `compose` macro's handling of dotted names would fail; `stoppedWorkerStartsNothing` pins nothing.

# Walkthrough: the standalone activity Model in Lean

This page walks through `StandaloneActivity.lean` from scratch. Code blocks are quoted verbatim
with their file and line range. Paths under `temporal/model/` are the real Umpire framework; paths
without a directory are files in `cmp/lean/`.

The framework is real and compiles. `StandaloneActivity.lean` itself has never been compiled;
sections 14 and 17 name what would not elaborate today.

## 1. What this is

Umpire tests a Temporal feature in three layers: a rulebook that says what may happen, a
playthrough that performs one path through it against a real server, and a referee that checks the
recorded run against the rulebook. This file is the rulebook for a standalone activity. A
standalone activity is started directly with `StartActivityExecution`, with no workflow around it;
it writes no history events, and a caller watches it through `DescribeActivityExecution`. The file
declares the activity's vocabulary, two state machines over it, the claims they make, and the
paths the tests should run.

## 2. The language in five minutes

Lean 4 is a functional language and proof assistant. You only need a few features here.

**Namespaces.** `namespace Temporal.Feature.Activity.Standalone` (`StandaloneActivity.lean:21`)
scopes every name until `end`; `open Umpire.Command` makes that namespace's names usable
unqualified.

**Comments.** `/-! -/` is a section comment, `/-- -/` documents the next declaration, `--` is a
line comment.

**Custom commands.** Lean lets a library add new top-level keywords. `entity`, `enum`, `action`,
`machine`, `property`, `scenario`, `query`, `set`, `compose` and `case` are not Lean; they are
commands defined in `temporal/model/Umpire/Command/Syntax.lean`. Each one expands into ordinary
Lean definitions and runs checks while the file compiles.

**Structures.** A `structure` is a record. `deriving` asks the compiler to generate instances,
here equality (`BEq`, `DecidableEq`), printing (`Repr`) and Umpire's `Finite` (section 4).

**Dot constructors.** `.started` means "the constructor `started` of whatever type is expected
here". `{ state with phase }` copies a record and replaces one field; `phase` alone is shorthand
for `phase := phase`.

StandaloneActivity.lean:359-361
```lean
private def moves (state : ProtocolState) (phase : Phase) (recorded : List ProtocolFact) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  [{ outcome := .accepted, state := { state with phase }, facts := recorded }]
```

**Match.** `match x with | .a => ... | .b => ...` branches on a constructor. The compiler rejects
a `match` that misses a case.

**Bounded numbers.** `Fin n` is a natural number below `n`. `abbrev` defines a constant that
unfolds freely.

**Lambdas and guards.** `fun step => ...` is an anonymous function. `#guard e` evaluates a Boolean
at compile time and fails the build if it is false; the test file is made of them.

## 3. Vocabulary: entities, parties, actions, inputs

**The entity.** An entity is what a machine is about. The activity is named by the id the caller
chose, so that is its key.

StandaloneActivity.lean:33-34
```lean
entity activity
  key: activityId
```

**The parties.** A party is who performs an action. This Model has three: `caller` starts and
controls the activity, `worker` runs its attempts, and `system` owns the timers. `system` is
reserved; the `action` command refuses it (`reservedPartyMessage` in `Syntax.lean`).

**Finite input domains.** Every input is an `enum`. A constructor may carry finite fields, and
then each assignment of those fields is its own member.

StandaloneActivity.lean:48-51
```lean
enum AttemptResult
  | completed
  | failed (retryable : Bool)
  | canceled
```

`failed (retryable : Bool)` is two members, `failed false` and `failed true`, mirroring the
retryable flag of an `ApplicationFailure`. So `AttemptResult` has four members. `Control` is
`pause | unpause | requestCancel | terminate`, and `Delivery` is `accepted | notFound`
(`StandaloneActivity.lean:53-61`).

**Actions and action classes.** An action is a side effect of a party. An action class is one
action with one assignment of its inputs. `attemptResult` therefore yields four classes and
`control` four. `start` has three `Timeout` inputs of two values each, so eight classes.

StandaloneActivity.lean:85-95
```lean
action attemptResult
  party: worker
  on: activity
  schema: temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest |
    temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest |
    temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest
  input:
    result: AttemptResult
  examples:
    failed (retryable := false) → ApplicationFailureNonRetryable
    failed (retryable := true) → ApplicationFailureRetryable
```

`on:` says which entity the action touches (`creates:` for `start`). `schema:` names the protobuf
messages that carry it; the elaborator checks those names against the generated Temporal API.
`examples:` names the concrete value a test should use for a class.

The four caller controls are one action, `control`, with a `Control` input and `results: Delivery`
(`StandaloneActivity.lean:97-108`). They share one result: a control on an activity that is over
is `notFound`.

**A fault is just an action.** Stopping the worker is not special machinery. `workerStop` is an
ordinary action of the `worker` party that names no entity (`StandaloneActivity.lean:110-113`).

**Observations.** Because nothing is written to history, facts are read back through Describe. An
`observation` declares such a read: `attemptCount` reads the field `attempt` of the activity
(`StandaloneActivity.lean:122-124`). The file also declares one observation per status, all
reading the same `status` field (`StandaloneActivity.lean:131-165`). Section 6 explains why they
are needed, and section 17 why they are a guess.

## 4. State

Each machine has a state type. The product machine's state is one phase out of nine
(`StandaloneActivity.lean:175-188`). The protocol machine adds an attempt counter and the three
deadlines.

StandaloneActivity.lean:319-327
```lean
abbrev attemptBound : Nat := 2

structure ProtocolState where
  phase : Phase
  attempts : Fin (attemptBound + 1)
  scheduleToClose : Timeout
  scheduleToStart : Timeout
  startToClose : Timeout
  deriving BEq, DecidableEq, Repr, Finite
```

**Every field is finite.** A deadline is `unset` or `expires`, not a duration. The attempt count
is `Fin 3`, so 0, 1 or 2. This is what lets the framework list every state: 12 phases, times 3
counts, times 2 for each deadline, gives 288 states. The product machine has 9.

**How the states are enumerated.** `Finite` is a type class in
`temporal/model/Umpire/Command/Finite.lean:37-39`. A type class is an interface the compiler
resolves by type. `Finite` has one member, `members`, the ordered list of all values.

The `enum` command turns into an `inductive` type (Lean's sum type) and derives `Finite` for it.

temporal/model/Umpire/Command/Syntax.lean:156-158
```lean
  elabCommand (← `(command| $[$doc?:docComment]? inductive $name where
      $declared:ctor*
      deriving BEq, DecidableEq, Repr, Umpire.Command.Finite))
```

The deriving handler (`mkFiniteInstanceHandler`, same file) builds the list. An enum lists its
constructors, one member per field assignment. A structure takes the product of its fields. Any
other field type, say `Nat`, is refused at that field by name.

**The saturating counter.** Incrementing the attempt count past 2 must not wrap to 0, or a Model
that retried twice would look as if it never counted. The framework provides a helper.

temporal/model/Umpire/Command/Finite.lean:62-63
```lean
def saturatingSucc {n : Nat} (count : Fin (n + 1)) : Fin (n + 1) :=
  if h : count.val + 1 < n + 1 then ⟨count.val + 1, h⟩ else ⟨n, Nat.lt_succ_self n⟩
```

At the bound the count stays at 2, which reads as "the limit was reached".

## 5. Step functions

A step function takes a state and the action's inputs and returns a list of steps. A `Step` is
`{ outcome, state, facts }`: the answer the caller gets, the next state, and what gets recorded. The
empty list means the action is not enabled in that state. A list with several entries would mean
nondeterminism; this Model never needs it.

A helper, `productStep` (`StandaloneActivity.lean:205-207`), builds the common single step with
outcome `accepted`, a phase and one fact.

**The product `attemptResultStep`.** This is the worker's answer as the caller sees it.

StandaloneActivity.lean:224-234
```lean
def attemptResultStep (state : ProductState) (result : AttemptResult) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .started && state.phase != .cancelRequested then [] else
  match result with
  | .completed => productStep .completed .statusCompleted
  | .failed false => productStep .failed .statusFailed
  | .failed true =>
      if state.phase == .cancelRequested then productStep .canceled .statusCanceled
      else productStep .scheduled .statusScheduled
  | .canceled =>
      if state.phase == .cancelRequested then productStep .canceled .statusCanceled else []
```

- **The guard.** Only `started` or `cancelRequested` has an attempt in flight; other phases return
  `[]`.
- **Completed and non-retryable failure** settle the activity.
- **Retryable failure.** From `started` the caller reads `SCHEDULED` again, so unlike the Nexus
  caller Model the retry is visible. From `cancelRequested` it settles as canceled.
- **Canceled** is accepted only after a cancel request; from `started` it returns `[]`.

**The protocol `protocolAttemptResultStep`.** The protocol machine knows more phases. Three of
them, `started`, `pauseRequested` and `cancelRequested`, mean "a worker holds the attempt"; the
helper `attemptHeld` tests for them (`StandaloneActivity.lean:356-357`).

StandaloneActivity.lean:383-395
```lean
def protocolAttemptResultStep (state : ProtocolState) (result : AttemptResult) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if !attemptHeld state.phase then [] else
  match result with
  | .completed => moves state .completed [.statusCompleted]
  | .failed false => moves state .failed [.statusFailed]
  | .failed true =>
      match state.phase with
      | .cancelRequested => moves state .canceled [.statusCanceled]
      | .pauseRequested => moves state .paused [.statusPaused]
      | _ => moves state .backingOff [.statusScheduled, .attemptCount]
  | .canceled =>
      if state.phase == .cancelRequested then moves state .canceled [.statusCanceled] else []
```

It dispatches on the phase as well as the result because the three source phases react differently
to a retryable failure. This mirrors CHASM's `chasm/lib/activity/statemachine.go`:

- **From `started`.** The attempt backs off (`TransitionRescheduled`). The caller reads
  `SCHEDULED` again with a higher attempt count, so both facts are recorded.
- **From `cancelRequested`.** The cancel request wins and the activity is canceled.
- **From `pauseRequested`.** The activity lands in `paused` rather than backing off
  (`TransitionAttemptFailedWhilePauseRequested`).

The `_` arm is a wildcard; because of the guard it only ever sees `started`.

**Exhaustiveness.** Lean enforces it: a `match` that misses a constructor is a compile error. The
`if` guards are not checked; a phase the author forgot simply returns `[]`. The `machine` command
adds its own checks for undeclared actions and stuck states (section 6).

## 6. The machine and its table

The `machine` command ties the pieces together.

StandaloneActivity.lean:465-473, 485-494
```lean
machine activityProtocol
  for: activity
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

- **`starts` and `ends`** name phases; other fields take their first value. Every state with an
  end phase is an end, so there are 5 × 24 = 120 end states.
- **`timers`** are `system` actions the machine owns and need no `action` command.
- **`unobservable`** lists timers that record nothing. The command refuses a silent timer not
  listed here, and a listed one that records something.
- **`evidence`** (lines 474-484, elided). Each line maps a fact the steps return to the recorded
  data that confirms it. A line may name a constructor and cover all its members, so
  `statusTimedOut` covers the three typed `statusTimedOut (timeoutType := ...)` facts.
- **`steps`.** Each action or timer is paired with its step function. The input fields of the
  action become the function's extra arguments, in order.

**When the table is built.** At compile time, inside the `machine` command. It evaluates the step
function once for every pair of state and action class, and refuses the machine if that walk is
too large.

temporal/model/Umpire/Command/Syntax.lean:2549-2558
```lean
  let walked := stateMembers.length * actionMembers.length
  if walked > enumerationBound then
    throwErrorAt stateType
      (machineTooLargeMessage stateMembers.length actionMembers.length enumerationBound)
  let transitionsName := mkIdentFrom name (name.getId ++ `transitions)
  elabGenerated (← `(command|
    def $transitionsName :
        List (Umpire.FiniteTransitionRow $stateType $actionType
          $(mkIdent outcomeType) $(mkIdent factType)) :=
      Umpire.Command.enumerateOver $actionsName $rowKeyName $stepName))
```

`enumerationBound` is 16384 (`Finite.lean:123`). This machine has 288 states and 22 action classes
(8 `start`, 1 `attemptStart`, 4 `attemptResult`, 4 `control`, 1 `workerStop`, 4 timers), so the
walk is 6336 evaluations. `enumerateOver` (`Finite.lean`) keeps a row only when the step function
returns a non-empty list. Each row gets a key such as
`started-1-unset-unset-unset-attemptResult-failed-true`: the state's fields, then the action
class.

Generated definitions this large need raised compiler limits. The command wraps its own output in
`maxRecDepth 65536` and `maxHeartbeats 1000000` (`elabGenerated`, `Syntax.lean:2001-2008`).

After the table exists the command also computes `activityProtocol.stuck`, the first reachable
state that is not an end and has no row, and fails at the `steps:` block if there is one.

**Evidence without history events.** Nexus evidence names history events. A standalone activity
writes none, so the command must accept something else. It checks declared observations first,
then the platform catalog.

temporal/model/Umpire/Command/Syntax.lean:2639-2645
```lean
  let declaredObservations := (Registry.observations (← getEnv)).map fun entry => entry.name
  for (_, observedRef) in evidenceRefs do
    let spelling := observedRef.getId.getString!
    unless declaredObservations.contains spelling do
      match ← (Umpire.Command.checkCatalog spelling : IO _) with
      | .ok () => pure ()
      | .error reason => throwErrorAt observedRef reason
```

The Temporal catalog (`temporal/model/Temporal/Case/Catalog.lean:69-81`) admits only history
event kinds, Run Event kinds and a few catalog reads. So the file declares nine `status*`
observations, one per Describe status. Nothing rejects nine observations over one field; whether a
realization can tell them apart is another matter (section 17).

## 7. Two levels and the refinement

**Why two machines.** The product machine says what the caller can see through Describe, and
nothing else. The protocol machine says how the server gets there: the backoff, the pause request,
the deadlines, the attempt count. Claims about what the user observes belong on the product. The
protocol is what the tests actually run.

**The map.** `productOf` reads a protocol state as a product state.

StandaloneActivity.lean:453-463
```lean
def productOf (state : ProtocolState) : ProductState :=
  { phase := match state.phase with
    | .unstarted | .scheduled | .backingOff => .scheduled
    | .started | .pauseRequested => .started
    | .paused => .paused
    | .cancelRequested => .cancelRequested
    | .completed => .completed
    | .failed => .failed
    | .canceled => .canceled
    | .terminated => .terminated
    | .timedOut => .timedOut }
```

`pauseRequested` reads as `started`. The worker still holds the attempt, so every answer it can
give is a product row from `started`. The pause request itself becomes a stutter. The first
version of the spec mapped it to `paused` and left three protocol rows with no product counterpart
(SPEC.md, first revision note).

**The rule.** For every protocol row, the checker maps its source and target. The row passes if
the product has a row from the mapped source with the same target, the same outcome, and facts
that are all among the protocol row's facts. Failing that, it passes as a stutter if the mapped
source and target are equal. Anything else rejects the refinement.

temporal/model/Umpire/Command/Refinement.lean:110-113
```lean
    let carriers := destination.table.transitions.filter fun carrier =>
      carrier.source == mappedFrom && carrier.results.any fun carried =>
        carried.state == mappedTo && some carried.outcome == mappedOutcome &&
          carried.facts.all mappedFacts.contains
```

The stutter fallback follows at `Refinement.lean:122-124`. This is stricter than the "mapped
states only" rule in SPEC.md. A protocol fact the product does not declare, like `attemptCount`,
is invisible to it. The action name is only a tie-breaker. Every protocol start must also map to a
product start, and `unstarted` maps to `scheduled`.

**Where and when it runs.** At compile time, inside the `machine` command. It first derives a
report and fails at the `map:` line with an English message if a row is rejected. Then it emits a
theorem over the two tables and has the Lean kernel decide it.

temporal/model/Umpire/Command/Syntax.lean:2872-2875
```lean
    elabGenerated (← `(command|
      theorem $witnessName :
          Umpire.TableRefinement ($name).table ($refined).table $abstractionName :=
        Umpire.TableRefinement.ofChecked (by decide +kernel)))
```

`decide +kernel` means the kernel evaluates the finite check itself. The command then reads the
theorem's axioms and refuses it if it rests on `sorryAx`, Lean's placeholder for a missing proof.

**One row by hand.** Take the protocol row
`started-1-unset-unset-unset-attemptResult-failed-true`.

1. The protocol steps from `{started, 1, unset, unset, unset}` to `backingOff` with outcome
   `accepted` and facts `[statusScheduled, attemptCount]`.
2. `productOf` maps the source to `started` and the target to `scheduled`.
3. The facts map by name to `[statusScheduled]`; `attemptCount` has no product namesake and drops
   out.
4. The product row `attemptResult (failed true)` from `started` goes to `scheduled`, `accepted`,
   `[statusScheduled]`. Its facts are all present, so it carries the protocol row.

The first draft of this file recorded only `[attemptCount]` on that row, and the real checker
would have rejected all 24 such rows. The current file carries both facts (SPEC.md, second
revision note).

## 8. Properties

A property is a claim. There are two shapes.

**Same-step claims** have `when:`. The predicate sees each single step taken by that action class.

StandaloneActivity.lean:511-515
```lean
property completes
  machine: activityProtocol
  when: attemptResult (completed)
  holds: fun step =>
    step.state.phase == .completed && step.facts.contains .statusCompleted
```

`retryCompletes` has the same shape with a sharper predicate: the resulting state must equal
`completedOnRetry`, a completed state with two attempts and no deadline
(`StandaloneActivity.lean:523-531`).

**Transition claims** have no `when:`. The predicate sees the step before and the step after and
must hold for every transition.

StandaloneActivity.lean:500-503
```lean
property terminalIsFinal
  machine: activityProduct
  holds: fun before after =>
    !(productTerminal before.state) || after.state.phase == before.state.phase
```

`pausedIsNotDispatched` has the same shape: `before.state.phase != .paused ||
after.state.phase != .started` (`StandaloneActivity.lean:505-509`).

Both are stated on the product, because they are about what the caller sees. A query can still run
them over a protocol scenario: `Refinement.lean` lifts a product property onto the refining
machine (`refinedProperty`). The `property` command enumerates a claim over the table at compile
time. It rejects a `when:` on a transition claim and a same-step claim without one.

## 9. Scenarios and limits

A scenario is one path: a start phase and the classed actions in order. An action class is spelled
as the action name with its inputs in parentheses, positionally.

StandaloneActivity.lean:577-581
```lean
scenario retriedThenCompleted
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, unset), attemptStart, attemptResult (failed true), backoff,
    attemptStart, attemptResult (completed)]
```

Traced by hand: `start` gives `scheduled` with 0 attempts; `attemptStart` gives `started` with 1;
the retryable failure gives `backingOff`; the silent `backoff` timer gives `scheduled`; the second
`attemptStart` gives 2; the completion lands on exactly `completedOnRetry`.

`pausedThenCompleted` is `start`, `control (pause)`, `control (unpause)`, `attemptStart`,
`attemptResult (completed)` (`StandaloneActivity.lean:595-599`).

A scenario admits exactly that schedule and nothing interleaved. The command lowers it through
`Scenario.exactly` (`temporal/model/Umpire/Command/Authoring.lean:505`).

Limits bound the search. `limits six` is 6 steps, 6 actions and a search budget of 262144
(`StandaloneActivity.lean:621-624`). `steps` and `actions` bound a trace's length. `search` bounds
how much the search may explore; what it counts depends on the backend (`AUTHORING.md`, section
8). The values here were copied from the Nexus Model and not derived for this machine (section
17).

## 10. Queries

A query pairs a property with a scenario and limits. The form picks the planner policy
(`temporal/model/Umpire/Command/Authoring.lean:608-615`).

- **`find`** searches for the shortest trace where the claim holds; a functional set turns it
  into a test.
- **`verify`** searches exhaustively for a counterexample and is never turned into a test.

`query retry` finds `retryCompletes` in `retriedThenCompleted` under `six`
(`StandaloneActivity.lean:638-641`). The verify form looks like this.

StandaloneActivity.lean:673-676
```lean
query pauseHolds
  verify: pausedIsNotDispatched
  in: pausedThenCompleted
  limits: six
```

**Where it runs.** While the file compiles. The `query` command evaluates a diagnostic and reports
a failure on the part of the block it belongs to, a wrong answer at the `find:` or `verify:`
keyword. A witness counts only after replay against the checked table; an absence answer is a
bounded search, not a proof.

**What a failure looks like.** The message separates "no such trace" from "the search was cut
short".

temporal/model/Umpire/Command/Authoring.lean:850-856
```lean
  | .noneFound | .limitReached =>
      if boundWasHit explored limits then
        s!"the search stopped at its declared bound after {explored.traces} traces; raise " ++
          "`limits` if the trace you mean is longer"
      else
        s!"no trace the Scenario admits satisfies the Property; the search explored " ++
          s!"{explored.traces} traces within the declared limits and no bound stopped it"
```

Because scenarios are exact, `pauseHolds` checks one path. The global claim is pinned in the test
file (section 13).

## 11. Sets

A set groups queries by purpose and binds every party except `system`.

StandaloneActivity.lean:685-691
```lean
set standaloneActivityTests
  purpose: functional
  bind:
    caller: driven
    worker: driven
  queries: [completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
    scheduleToStartTimeout, startToCloseTimeout]
```

- **Functional.** Each `find` query becomes one test. `driven` means the test's own program
  performs that party's actions.
- **Canary.** `standaloneActivityCanary` runs `completion` and `cancel` against a deployment with
  the worker `observed`: the real worker answers and the checker reads which class occurred.
- **Exploratory.** It names a machine, coverage goals and a budget instead of queries
  (`StandaloneActivity.lean:700-707`).
- **No `repeat`.** Standalone activities exist only under CHASM, so there is no HSM/CHASM switch
  to repeat over.

## 12. Composition with the worker

The worker is its own entity with its own machine, `polling`, in `Worker.lean:84-92`. A polling
worker can serve; a stopped one returns `[]` for `serve` (`Worker.lean:78-81`).

The activity only needs the stop and the serve, so it derives a restricted machine.

StandaloneActivity.lean:739-741
```lean
machine activityWorker
  from: Worker.polling
  restrict: [workerStop, serve]
```

`compose` builds the product of the two machines. Unsynchronized actions interleave. A `sync` line
fires two member actions as one step, enabled only when both are.

StandaloneActivity.lean:748-759
```lean
compose standaloneActivity
  for: [activity, Worker.worker]
  state: StandaloneActivityState
  members:
    activity: activityProtocol
    worker: activityWorker
  sync:
    workerStop: activity.workerStop ∥ worker.workerStop
    attemptStart: activity.attemptStart ∥ worker.serve
  starts: [activity.unstarted, worker.polling]
  ends: [activity.completed, activity.failed, activity.canceled, activity.terminated,
    activity.timedOut]
```

The first sync makes the stop move the worker to `stopped`. The second makes every attempt start a
worker serve, which has no row from `stopped`. So the composed table has no `attemptStart` row at
all from a state whose worker is stopped. The `compose` command proves its composed table agrees
with its members by `decide +kernel`, the same way as the refinement (`elabComposedAgreement`,
`temporal/model/Umpire/Command/Syntax.lean:3034-3068`).

The property `startedByPollingWorker` claims that on every `attemptStart` the worker is polling
(`StandaloneActivity.lean:761-764`).

StandaloneActivity.lean:770-779
```lean
scenario stoppedBeforeRetry
  model: standaloneActivity
  starts: activity.unstarted
  actions: [activity.start (unset, expires, unset), attemptStart,
    activity.attemptResult (failed true), activity.backoff, workerStop, activity.scheduleToStart]

query stoppedWorkerStartsNothing
  verify: startedByPollingWorker
  in: stoppedBeforeRetry
  limits: six
```

The first attempt starts while the worker polls, so the claim fires on the path. The attempt
fails retryably and backs off. Then the worker stops, so the retry is never dispatched and the
schedule-to-start deadline fires. Synchronized actions are spelled bare (`attemptStart`,
`workerStop`), and member actions carry their member's prefix (`activity.backoff`).

The attempt start matters. Scenarios are exact, and the query does not set `requireFiring`, which
defaults to `false` (`temporal/model/Umpire/Query.lean:246`). A path without an attempt start would
verify vacuously, and the first version of this scenario did (SPEC.md, third revision note). The
test file now pins that the claim is exercised (section 13). The general guarantee is still the
missing row in the composed table.

## 13. Pins

`ActivityPins.lean` holds the spec's pins as `#guard` lines. Like the Model, it has not been
compiled.

ActivityPins.lean:34-36
```lean
/- A paused activity is dispatched to no worker: no product row leaves paused for started. -/
#guard (activityProduct.transitions.filter fun row =>
  row.source.phase == .paused && row.results.any (·.state.phase == .started)).isEmpty
```

This pin guards `pausedIsNotDispatched` over the whole product table, not just one path. A future
`unpause` that went straight to `started` would fail it. A row has a `source` state and a list of
`results` (`temporal/model/Umpire/Model/Table.lean:53-57`), hence the `any`.

A second pin (`ActivityPins.lean:59-60`) fixes the facts of the retryable-failure row from
`started`, the one row the real checker is sensitive to.

ActivityPins.lean:99-100, 106-107
```lean
#guard activityProtocol.refinement.rows.lookup
  "started-1-unset-unset-unset-attemptResult-failed-true" == some (some "attemptResult-failed-true")
#guard activityProtocol.refinement.rows.lookup
  "pauseRequested-1-unset-unset-unset-attemptResult-failed-true" == some (some "control-pause")
```

These guard how two rows were classified. The visible retry is carried by the product's own
retryable-failure row. A retryable failure under a pause request is carried by the product's
`control (pause)` row, the product row from `started` to `paused`. `some none` would mean a stutter.

ActivityPins.lean:121-124
```lean
#guard stoppedWorkerStartsNothing.answer.verified
/- Not vacuous: the scenario performs an attempt start while the worker polls, so the claim is
exercised, not merely never contradicted. -/
#guard stoppedWorkerStartsNothing.answer.coverage == .exercised
```

The last pin guards the composed query against passing vacuously again.

## 14. From model to running test

After a file like this compiles, the real pipeline has three more steps.

1. **Case lowering.** The `case` command names a set and a realization. A realization is the
   Temporal-owned value that binds each action to an RPC or worker instruction, each observation
   to where it is read, and each timer to a duration. The command produces one Case per `find`
   query: a Program to perform and a Contract to check. It is checked in as a JSON fixture.
2. **The Go Testpilot runtime.** `testpilot.Prepare(case, Profile)` admits the Case, and
   `PreparedCase.Run(ctx, Driver)` runs it against a server and records a Run
   (`temporal/model/README.md`, "Runtime ownership").
3. **Verdicts.** The runtime checks the Run against the Contract and returns a Verdict: satisfied,
   violated or inconclusive.

For standalone activities, none of this exists yet. The real realizations are `asyncNexus`,
`workflowStart`, `workflowOutage` and `unaryRpc`, and the `case` elaborator is Nexus specific. The
three `case` blocks (`StandaloneActivity.lean:719-729`) call
`Temporal.Case.Realization.standaloneActivity`, which does not exist, so they would not elaborate.

## 15. Gaps and gradual growth

The Model leaves several things out on purpose.

- **Reset** is deferred, like cancellation in the Nexus caller Model. No action exists for it.
- **Heartbeat timeout** is not modeled. The pause request is described as reaching the worker on
  its next heartbeat, but heartbeats themselves have no action.
- **Stutter rows.** The protocol's worker stop keeps the state and records nothing
  (`StandaloneActivity.lean:419-423`); a Case would carry a Known Gap for it.
- **Empty steps.** The product's `workerStop` returns `[]` everywhere.
- **Silent timer.** `backoff` is `unobservable`, which becomes a Known Gap on paths that use it.

This supports gradual growth. Adding a behavior is local: an enum member or action, a step arm, a
`steps:` line, and a product row if the caller can see it. The compiler re-checks the rest:
matches, refinement, stuck states and every query. Heartbeat timeout, for example, would add a
timer, a `TimeoutType` member and a `heartbeat : Timeout` field, doubling the states to 576.

## 16. Mental model recap

- The file is a rulebook: finite vocabulary, two machines, claims, and paths.
- Everything is finite, so the whole transition table is built at compile time.
- A step function returns the steps an action can take; `[]` means "not enabled".
- The product is what Describe shows; the protocol is how the server gets there, and it refines
  the product row by row, matching states, outcome and facts, or stuttering.
- Properties are same-step or transition claims; queries `find` or `verify` them on an exact path
  while the file compiles.
- Sets decide what becomes a test; nothing past the set exists for standalone activities yet.

## 17. Where this implementation is weak

- **It has never been compiled.** The `case` blocks name a realization that does not exist, so
  they would fail today. There may be other errors nobody has seen, and a one-line edit costs about
  three minutes of rebuild to find out.
- **The status observations are a guess about the realization layer.** Nine observations all read
  the field `status` and none says which value it expects. From reading the elaborator they should
  pass the machine command. A realization would still need to know that `statusScheduled` means
  `status == SCHEDULED`, and the `observation` command has no way to say so.
- **Search budgets are copied, not derived.** The review of this sample (`cmp/.eval/lean.md`)
  notes that `three`, `four` and `six` reuse the Nexus numbers. The activity has more enabled
  actions per state, so whether `4096` covers three-step paths is unverified.

Two weaknesses found while writing this walkthrough are now fixed:

- **The table pin used the wrong field names.** `ActivityPins.lean:35-36` now uses `row.source` and
  `row.results`, the real field names.
- **The composed query passed vacuously.** `stoppedBeforeRetry` replaces `stoppedBeforeDispatch`
  with an attempt start before the stop, and `ActivityPins.lean:124` pins that the claim is
  exercised.

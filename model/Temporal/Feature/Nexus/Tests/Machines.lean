import Temporal.Feature.Nexus.Tests.Commands

/-!
# The design's own machines, written as step functions

`DESIGN.md` section 3 writes the Nexus caller-side operation as two machines: a product machine that
says what an operation does, and a protocol machine that says how the server gets there. It writes
them as step functions, and the Caller Model (`Temporal.Feature.Nexus.Caller`) is where both live
since fn-85 .10. This module keeps the product machine, written over the vocabulary specimen of
`Tests.Commands`, as the small machine the `machine` command's rejections are pinned against, and
the loops below as the specimens of what a Search makes of a timer and what a refinement rejects.

Not here: the `requestCancel` and `cancelReply` rows, and the `cancel` state field they move. The
design marks them `fn-79`, which resumes only on an explicit request, so writing them here would
deliver that task early.
-/

namespace Temporal.Feature.Nexus.Tests.Machines

open Umpire
open Umpire.Command
open Temporal.Feature.Nexus.Tests.Commands

/-! ### The product machine

What an operation does, with no account of how. Every Property written against it is carried to the
protocol machine by the refinement (`DESIGN.md` section 2.5), which is declared on the protocol
machine below. -/

enum ProductPhase
  | scheduled
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

structure ProductState where
  phase : ProductPhase
  deriving BEq, DecidableEq, Repr, Finite

enum ProductOutcome
  | accepted
  | notFound

enum ProductFact
  | nexusOperationScheduled
  | nexusOperationStarted
  | nexusOperationCompleted
  | nexusOperationFailed
  | nexusOperationCanceled
  | nexusOperationTimedOut
  | faultInjected

private def productStep (phase : ProductPhase) (recorded : ProductFact) :
    List (Step ProductState ProductOutcome ProductFact) :=
  [{ outcome := .accepted, state := { phase }, facts := [recorded] }]

/-- The handler's reply to the server's start request. An operation that has not started yet is the
only one a reply can move. -/
def handlerReplyStep (state : ProductState) (reply : Reply) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => productStep .succeeded .nexusOperationCompleted
  | .async => productStep .started .nexusOperationStarted
  | .operationFailed => productStep .failed .nexusOperationFailed
  | .operationCanceled => productStep .canceled .nexusOperationCanceled
  -- A retryable handler error leaves the operation where it is: the product machine does not know
  -- about backing off, which is the whole of what the protocol machine adds.
  | .handlerError true => []
  | .handlerError false => productStep .failed .nexusOperationFailed

/-- An asynchronous completion. A completion that arrives after the operation is over is not found,
and changes nothing. -/
def completeStep (state : ProductState) (resolution : Resolution) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase == .succeeded || state.phase == .failed || state.phase == .canceled ||
      state.phase == .timedOut then
    [{ outcome := .notFound, state, facts := [] }]
  else
    match resolution with
    | .succeeded => productStep .succeeded .nexusOperationCompleted
    | .failed => productStep .failed .nexusOperationFailed
    | .canceled => productStep .canceled .nexusOperationCanceled

/-- A transport fault is an ordinary action of the network. The product machine cannot see one:
whether a delivery was retried is the protocol's account of how, not what. -/
def transportFaultStep (_state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) := []

/-- The handler's worker stopping is a fault the run records and the operation does not feel. -/
def workerStopStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  [{ outcome := .accepted, state, facts := [.faultInjected] }]

/-- One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
product machine has one timer, and it fires while the operation runs. -/
def timeoutStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase == .scheduled || state.phase == .started then
    productStep .timedOut .nexusOperationTimedOut
  else []

machine nexusProduct
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [timeout]
  evidence:
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
    nexusOperationTimedOut: nexusOperationTimedOut
    faultInjected: faultInjected
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep
    timeout: timeoutStep

/-! ### What the product machine says

The claims below are about the table, because the table is what Search, the Behavior Fingerprint and
Contract lowering read. A `match` arm that stopped saying what it says would fail here. -/

/- Six phases, and the four the design ends on. -/
#guard nexusProduct.table.states.length == 6
#guard nexusProduct.ends.length == 4

/- Every action class the machine steps on: six replies, three resolutions, the two faults, and the
one timer. -/
#guard nexusProduct.actionKeys.size == 12

/- An async reply starts the operation and records that it started. -/
#guard (handlerReplyStep { phase := .scheduled } .async).map (·.state.phase) == [.started]
#guard (handlerReplyStep { phase := .scheduled } .async).flatMap (·.facts) ==
  [.nexusOperationStarted]

/- A retryable handler error is invisible here: it is the protocol machine that backs off. -/
#guard handlerReplyStep { phase := .scheduled } (.handlerError (retryable := true)) == []

/- A completion after the operation is over is accepted and found nothing. -/
#guard (completeStep { phase := .succeeded } .succeeded).map (·.outcome) == [.notFound]
#guard (completeStep { phase := .succeeded } .succeeded).map (·.state.phase) == [.succeeded]

/- What the Model actually reaches: every phase. `timedOut` is reached by the one timer, because a
refinement carries every protocol step to a product step or a stutter, and a deadline firing is
neither a stutter nor anything a product without a timer could take. -/
#guard (Umpire.Command.reachableFrom nexusProduct.starts
    nexusProduct.transitions).map nexusProduct.stateKeyFor ==
  ["scheduled", "canceled", "failed", "succeeded", "started", "timedOut"]

/- Nothing the command declares rests on an unchecked proof. The canonical-table law is what the
Behavior Fingerprint, Search and Contract lowering all read, and `elabCommand` logs a failure rather
than throwing it -- so a machine too large to prove would otherwise be declared carrying `sorryAx`
and read as complete. -/
/-- info: 'Temporal.Feature.Nexus.Tests.Machines.nexusProduct' depends on axioms: [propext] -/
#guard_msgs in
#print axioms nexusProduct

/- Nothing here is stuck. Weak on its own -- `workerStop` steps from every state, so no state of this
machine could be stuck -- which is why the rejection is pinned below on a machine that can be. -/
#guard nexusProduct.stuck == none

/-! ### The protocol machine

The protocol machine that refines this product machine, its map, the refinement and the Queries over
it are the Caller Model's (`Temporal.Feature.Nexus.Caller`), and what they say is pinned in
`Temporal.Feature.Nexus.Caller.Tests`; this module keeps the product machine as the small specimen
the rejections below are written against. -/

/-! ### What a Search makes of a timer

A timer is an ordinary member of the machine's Action domain, so a Search selects it the way it
selects anything else and the Limits count it the way they count anything else. Neither is a rule
the command enforces -- there is no place where a timer could have been treated differently -- and
this is the machine that shows it, small enough for a Query to be written over it by name.

`DESIGN.md` section 3's protocol machine cannot be: a Scenario names its start state and its Actions
with identifiers, and that machine's keys are punctuated -- `schedule-unset-unset-expires`, and a
state key naming all five fields. Which surface a Scenario should use over a structured state is
task `.5`'s question, and this module says only that the answer is not "the machine is special". -/

enum AttemptPhase
  | trying
  | waiting
  | finished

structure AttemptState where
  phase : AttemptPhase
  deriving BEq, DecidableEq, Repr, Finite

enum AttemptOutcome
  | accepted

enum AttemptFact
  | pendingAttempts
  | faultInjected

/-- A dropped delivery puts the operation into its backoff, and the attempt count it raises is read
back through a call rather than off the history. -/
def attemptFaultStep (state : AttemptState) :
    List (Step AttemptState AttemptOutcome AttemptFact) :=
  if state.phase != .trying then [] else
  [{ outcome := .accepted, state := { phase := .waiting }, facts := [.pendingAttempts] }]

/-- The backoff timer, which fires only out of the phase it backs off in. Nothing records that it
fired, which is what `unobservable:` says. -/
def attemptRetryStep (state : AttemptState) :
    List (Step AttemptState AttemptOutcome AttemptFact) :=
  if state.phase != .waiting then [] else
  [{ outcome := .accepted, state := { phase := .trying }, facts := [] }]

/-- Stopping the handler's worker ends the attempt, and the harness records that it injected the
fault. -/
def attemptStopStep (state : AttemptState) :
    List (Step AttemptState AttemptOutcome AttemptFact) :=
  if state.phase != .trying then [] else
  [{ outcome := .accepted, state := { phase := .finished }, facts := [.faultInjected] }]

machine attemptLoop
  for: operation
  state: AttemptState
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: attemptFaultStep
    workerStop: attemptStopStep
    retry: attemptRetryStep

/- The timer has a row where its step returns a successor and nowhere else. A Search reads the
table, so this is the whole of "a timer fires only while it is enabled". -/
#guard attemptLoop.transitions.filterMap (fun row =>
  if attemptLoop.actionKeyFor row.action == "retry" then some (attemptLoop.stateKeyFor row.source)
  else none) == ["waiting"]

/- Three action classes, and the timer is one of them: nothing tells a Search that one of these is
a timer, which is why nothing has to tell the Limits either. -/
#guard attemptLoop.actionKeys.toList == ["retry", "transportFault", "workerStop"]

property attemptEnds
  machine: attemptLoop
  when: workerStop
  holds: fun step => step.state.phase == .finished

/- The operation is dropped once, backs off, and is stopped: three occurrences, one of them the
timer's and one of them the fault's. -/
scenario faultThenRetry
  model: attemptLoop
  starts: trying
  actions: [transportFault, retry, workerStop]

limits threeOccurrences
  steps: 3
  actions: 3
  search: 16

query attemptCompletes
  find: attemptEnds
  in: faultThenRetry
  limits: threeOccurrences

/- The predicate fixes the one state the keyed form named, so the fingerprint was the keyed form's
when the form changed (pinned at fcbc068); the value here is the one after fn-85 `.4` gave a
machine's state fields a meaning. -/
#guard (attemptCompletes.toOption.map fun checked =>
  checked.property.behaviorFingerprint.render) ==
  some "sha256:c610cd4a976af184563203c4150073ab75fee68895f48c7b08222240a8bf755d"

/- The Search finds the trace: three occurrences within a budget of three, one of them the timer's
firing and one of them the fault. -/
#guard (match attemptCompletes with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"

/- The same Query with one candidate of search budget stops at the limit. A timer firing and a fault
action are what it spent that budget on: nothing in the planner knows that one of these Actions is a
timer, so there is no place where they could have been counted differently. -/
limits oneCandidate
  steps: 3
  actions: 3
  search: 1

/--
error: the search stopped at its declared bound after 1 traces; raise `limits` if the trace you mean is longer
-/
#guard_msgs in
query attemptOutOfBudget
  find: attemptEnds
  in: faultThenRetry
  limits: oneCandidate

/- The count a machine keeps of its own attempts is bounded by the Limits the same way, and reaching
that bound is what `limitReached` says of a count. Saturating rather than wrapping is what makes the
claim readable: a count that rolled over would say the operation had never been retried. -/
#guard (Umpire.Command.members (α := Fin 3)).map Umpire.Command.limitReached ==
  [false, false, true]
#guard Umpire.Command.limitReached (Umpire.Command.saturatingSucc (Fin.last 2)) == true

/-! ### What a refinement is, and what it rejects

The rejections are pinned on a small pair, because each is about the declaration rather than about
the size of the machine. `retryLoop` refines `attemptLoop` exactly: the loop's phase is the
attempt's phase, and whether it has retried is hidden. -/

structure RetryState where
  phase : AttemptPhase
  retried : Bool
  deriving BEq, DecidableEq, Repr, Finite

def retryFaultStep (state : RetryState) : List (Step RetryState AttemptOutcome AttemptFact) :=
  if state.phase != .trying then [] else
  [{ outcome := .accepted, state := { state with phase := .waiting }, facts := [.pendingAttempts] }]

def retryRetryStep (state : RetryState) : List (Step RetryState AttemptOutcome AttemptFact) :=
  if state.phase != .waiting then [] else
  [{ outcome := .accepted, state := { phase := .trying, retried := true }, facts := [] }]

def retryStopStep (state : RetryState) : List (Step RetryState AttemptOutcome AttemptFact) :=
  if state.phase != .trying then [] else
  [{ outcome := .accepted, state := { state with phase := .finished }, facts := [.faultInjected] }]

/-- The attempt loop's phase is the retry loop's phase; whether it retried is hidden. -/
def attemptOf (state : RetryState) : AttemptState := { phase := state.phase }

machine retryLoop
  for: operation
  state: RetryState
  refines: attemptLoop
  map: attemptOf
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: retryFaultStep
    workerStop: retryStopStep
    retry: retryRetryStep

/- Every row is the attempt loop's step of the same name; nothing stutters. -/
#guard retryLoop.refinement.rejected == none
#guard retryLoop.refinement.rows.all (·.2.isSome)
#guard retryLoop.refinement.rows.lookup "trying-false-transportFault" == some (some "transportFault")
#guard retryLoop.refinement.rows.lookup "waiting-true-retry" == some (some "retry")

/- The attempt loop's state is a field of the retry loop's, named after the attempt loop. -/
#guard retryLoop.stateFieldIds.map (·.1) == ["phase", "retried", "attemptLoop"]

/- A Property on the attempt loop is found on the retry loop's paths: `attemptEnds` is about
`workerStop`, which the retry loop names too. -/
scenario retryThenStop
  model: retryLoop
  starts: trying
  actions: [transportFault, retry, workerStop]

query retryCompletes
  find: attemptEnds
  in: retryThenStop
  limits: threeOccurrences

#guard (match retryCompletes with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"

/- A map says how this machine's state reads as another's, so it names that machine. -/
/--
error: `map:` says how this machine's state reads as another machine's, so `refines:` names that machine; a `map:` without `refines:` maps to nothing
-/
#guard_msgs in
machine mapAlone
  for: operation
  state: RetryState
  map: attemptOf
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: retryFaultStep
    workerStop: retryStopStep
    retry: retryRetryStep

/--
error: `refines:` names the machine this one refines, and `map:` names the function that reads this machine's state as its state; a refinement needs both
-/
#guard_msgs in
machine refinesAlone
  for: operation
  state: RetryState
  refines: attemptLoop
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: retryFaultStep
    workerStop: retryStopStep
    retry: retryRetryStep

/--
error: 'notAMachine' is not a machine declared by a `machine` command; `refines:` names the product machine this one refines
-/
#guard_msgs in
machine refinesNothing
  for: operation
  state: RetryState
  refines: notAMachine
  map: attemptOf
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: retryFaultStep
    workerStop: retryStopStep
    retry: retryRetryStep

/-- A map into the wrong machine's state: every value it produces is one the refined machine never
declared. -/
def productOfRetry (_state : RetryState) : ProductState := { phase := .scheduled }

/--
error: 'Temporal.Feature.Nexus.Tests.Machines.productOfRetry' is not a map from this machine's state to the refined machine's; `map:` names a function `Temporal.Feature.Nexus.Tests.Machines.RetryState → Temporal.Feature.Nexus.Tests.Machines.AttemptState`
-/
#guard_msgs in
machine mappedElsewhere
  for: operation
  state: RetryState
  refines: attemptLoop
  map: productOfRetry
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: retryFaultStep
    workerStop: retryStopStep
    retry: retryRetryStep

/-- An outcome domain with a value the attempt loop has no name for. -/
enum RetryOutcome
  | accepted
  | rejected

def loudFaultStep (state : RetryState) : List (Step RetryState RetryOutcome AttemptFact) :=
  if state.phase != .trying then [] else
  [{ outcome := .rejected, state := { state with phase := .waiting }, facts := [.pendingAttempts] }]

def loudRetryStep (state : RetryState) : List (Step RetryState RetryOutcome AttemptFact) :=
  if state.phase != .waiting then [] else
  [{ outcome := .accepted, state := { phase := .trying, retried := true }, facts := [] }]

def loudStopStep (state : RetryState) : List (Step RetryState RetryOutcome AttemptFact) :=
  if state.phase != .trying then [] else
  [{ outcome := .accepted, state := { state with phase := .finished }, facts := [.faultInjected] }]

/- An outcome reads as the refined machine's outcome of the same name, and one with no such name
reads as nothing. -/
/--
error: 'rejected' is an outcome of loudRetryLoop and no outcome of attemptLoop has that name; an outcome reads as the one of its name, so the refined machine declares it
-/
#guard_msgs in
machine loudRetryLoop
  for: operation
  state: RetryState
  refines: attemptLoop
  map: attemptOf
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: loudFaultStep
    workerStop: loudStopStep
    retry: loudRetryStep

/-- A map under which a dropped delivery finishes the attempt: the attempt loop finishes only by the
worker stopping, which records a different fact. -/
def finishedEarly (state : RetryState) : AttemptState :=
  { phase := if state.phase == .waiting then .finished else state.phase }

/- A row whose mapped states are neither a step of the refined machine nor equal is the refinement's
own rejection, reported with the row, both readings and what the refined machine lacks. -/
/--
error: the row 'trying-false-transportFault' steps from 'trying-false' to 'waiting-false', which read as 'trying' and 'finished' in attemptLoop; attemptLoop has no step from 'trying' reaching 'finished' with outcome 'accepted' and the facts [pendingAttempts], and the two are not equal, so the row is neither a step of attemptLoop nor a stutter
-/
#guard_msgs in
machine finishesEarly
  for: operation
  state: RetryState
  refines: attemptLoop
  map: finishedEarly
  starts: [trying]
  ends: [finished]
  timers: [retry]
  unobservable: [retry]
  evidence:
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    transportFault: retryFaultStep
    workerStop: retryStopStep
    retry: retryRetryStep

/-! ### What the machine command rejects

Each at the line that made it, because a Model file is read and corrected one line at a time. -/

/- A machine tracks one entity's instances, so `for:` names one the file declared. -/
/--
error: 'notAnEntity' is not an entity declared by an `entity` command; a machine tracks one entity's instances, so `for:` names one
-/
#guard_msgs in
machine untrackedEntity
  for: notAnEntity
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  steps:
    handlerReply: handlerReplyStep

/--
error: 'notAnAction' is not an action declared by an `action` command; a `steps:` line names the action its function steps on
-/
#guard_msgs in
machine unknownStep
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  steps:
    notAnAction: handlerReplyStep

/- A machine's states are the members of a structure, so its fields can be enumerated. -/
/--
error: 'ProductPhase' is not a finite state structure; a machine's `state:` names a `structure` whose fields are all finite, so its members can be enumerated
-/
#guard_msgs in
machine phaseAsState
  for: operation
  state: ProductPhase
  starts: [scheduled]
  ends: [succeeded]
  steps:
    handlerReply: handlerReplyStep

/- One action has one step function; two would not say which applies. -/
/--
error: the machine steps on 'handlerReply' twice; one action has one step function, and two would not say which one applies
-/
#guard_msgs in
machine twiceStepped
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  steps:
    handlerReply: handlerReplyStep
    handlerReply: handlerReplyStep

/-- A step function takes the state and the action's inputs, and returns what may happen. This one
returns phases, so it says what the state became and not what may happen -- no outcome, no evidence,
and nothing a row could carry. -/
def wrongShape (_state : ProductState) : List ProductPhase := []

/--
error: 'Temporal.Feature.Nexus.Tests.Machines.wrongShape' is not a step function; a `steps:` line names one of the shape `State -> <the action's input domains, curried> -> List (Step State Outcome Fact)`
-/
#guard_msgs in
machine wronglyShaped
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  steps:
    transportFault: wrongShape

/- A timer no `steps:` line names never fires. -/
/--
error: no `steps:` line names the timer 'neverFires'; a timer is `system` behaviour written as a step function, and one that never fires is a timer the machine does not have
-/
#guard_msgs in
machine idleTimer
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  timers: [neverFires]
  steps:
    handlerReply: handlerReplyStep

/- A state the machine reaches, does not end in, and can take no step from is where a Search stops
without having finished. Only `handlerReply` steps here, so `started` is such a state. -/
/--
error: the machine reaches 'started', does not end there, and can take no step from it; either a step is missing or 'started' belongs under `ends:`
-/
#guard_msgs in
machine stuckInStarted
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  steps:
    handlerReply: handlerReplyStep

/- A step function's arguments after the state are the action's own input domains, in order. Two
actions of the same arity over different domains are an ordinary slip, and it is caught at the line
that named the function rather than inside the code the command generates. -/
/--
error: 'Temporal.Feature.Nexus.Tests.Machines.completeStep' takes Resolution where this action's input 1 is 'Temporal.Feature.Nexus.Tests.Commands.Reply'; a step function's arguments after the state are the action's own input domains, in order
-/
#guard_msgs in
machine wrongInputDomain
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  steps:
    handlerReply: completeStep

/- Evidence names a fact the steps return; one that names another confirms nothing that happens. -/
/--
error: no step of this machine returns the fact 'nothingRecordsThis', so nothing it confirms ever happens; the steps return nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut, faultInjected
-/
#guard_msgs in
machine unreturnedEvidence
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded]
  evidence:
    nothingRecordsThis: nexusOperationScheduled
  steps:
    handlerReply: handlerReplyStep

/-- A timer that fires and records what it did. -/
def loudTimerStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .started then [] else
  [{ outcome := .accepted, state := { phase := .timedOut },
     facts := [.nexusOperationTimedOut] }]

/- A machine says where it ends. Without `ends:` nothing is terminal, so a Search runs to its limit
on every path and a Property that requires an instance to finish holds by never being reached. -/
/--
error: the machine declares no 'ends:'; it is required
-/
#guard_msgs in
machine endlessly
  for: operation
  state: ProductState
  starts: [scheduled]
  steps:
    handlerReply: handlerReplyStep

/-- A timer that fires and records nothing. Nothing drives a timer, so its evidence is the only way
a Case can tell it fired. -/
def quietTimerStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .started then [] else
  [{ outcome := .accepted, state := { phase := .succeeded }, facts := [] }]

/--
error: the timer 'quiet' fires and records nothing an `evidence:` line names, so no Contract can tell it fired; give it evidence, or declare it `unobservable:` and every Case whose path uses it carries a Known Gap
-/
#guard_msgs in
machine silentTimer
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [quiet]
  evidence:
    nexusOperationStarted: nexusOperationStarted
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep
    quiet: quietTimerStep

/- Declaring it `unobservable:` is what admits the same machine, and what puts a Known Gap in every
Case whose path fires the timer. -/
machine gappedTimer
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [quiet]
  unobservable: [quiet]
  evidence:
    nexusOperationStarted: nexusOperationStarted
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep
    quiet: quietTimerStep

/- The machine is declared, and the timer is one of its actions. What `unobservable:` records lives
on the machine's registry entry rather than on the value, because a Case reads it while it is being
built and nothing downstream of the table needs it. -/
#guard gappedTimer.actionKeys.contains "quiet"
#guard gappedTimer.stuck == none

/- A timer whose firing the realization does record is observable, and declaring it unobservable
would put a Known Gap in every Case that does not need one. -/
/--
error: the timer 'loud' records evidence, so a Contract can tell it fired; `unobservable:` is for a firing nothing records, and declaring an observable one would put a Known Gap in every Case that does not need it
-/
#guard_msgs in
machine gappedObservable
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [loud]
  unobservable: [loud]
  evidence:
    nexusOperationTimedOut: nexusOperationTimedOut
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep
    loud: loudTimerStep

/- `unobservable:` is about a timer: an action a party takes is driven, and what a Case does with it
is decided by the binding rather than by the machine. -/
/--
error: 'handlerReply' is not a timer of this machine; `unobservable:` names a timer whose firing the realization records nowhere, and an action a party takes is driven rather than observed
-/
#guard_msgs in
machine unobservableAction
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  unobservable: [handlerReply]
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep

/-- A timer whose step function is enabled in no state. -/
def neverEnabledStep (_state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) := []

/--
error: the timer 'asleep' is never enabled: its step function returns nothing in every state, so the timer never fires and the machine does not have it
-/
#guard_msgs in
machine idleStep
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [asleep]
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep
    asleep: neverEnabledStep

/- The other side of an evidence line is recorded data the realization carries or an `observation`
this Model declares. A name in neither confirms nothing that can be read back. -/
/--
error: 'nexusOperationDreamt' is neither a recorded event kind the realization carries, a read observation its catalog binds, nor an observation this Model declares; evidence names recorded data, so it is a generated history event kind, a Testpilot Run Event kind, a catalog read such as `pendingAttempts`, or a derived `observation`
-/
#guard_msgs in
machine strayObservation
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  evidence:
    nexusOperationStarted: nexusOperationDreamt
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep

/- A derived observation the Model declares is admitted beside the catalog: the attempt count is
read back through a call because no history event records it. -/
machine readObservation
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  evidence:
    nexusOperationStarted: pendingAttempts
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep

#guard readObservation.actionKeys.size == 11

/- A `match` arm no input reaches is the design's "shadowed row", and nothing in the command has to
say so: the step function is ordinary Lean, so Lean reports it at the arm that cannot be taken --
before the machine is declared, and pointing at the arm rather than at the `steps:` line. -/
/--
error: Redundant alternative: Any expression matching
  Reply.handlerError true
will match one of the preceding alternatives
-/
#guard_msgs in
def shadowedArm (_state : ProductState) (reply : Reply) :
    List (Step ProductState ProductOutcome ProductFact) :=
  match reply with
  | .handlerError _ => []
  | .handlerError true => []
  | _ => []


end Temporal.Feature.Nexus.Tests.Machines

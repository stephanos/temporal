import Temporal.Feature.Nexus.Tests.Commands

/-!
# The design's own machines, written as step functions

`DESIGN.md` section 3 writes the Nexus caller-side operation as two machines: a product machine that
says what an operation does, and a protocol machine that says how the server gets there. It writes
them as rows, in the grammar the user's 2026-09-12 decision replaced with ordinary Lean. This module
is that decision applied to the design's own specimen -- the same behaviour, written as `match` a Go
reader would recognise, enumerated by the `machine` command into the same table the rows produced.

Not here: the `requestCancel` and `cancelReply` rows, and the `cancel` state field they move. The
design marks them `fn-79`, which resumes only on an explicit request, so writing them here would
deliver that task early. The protocol machine's state is therefore its phase, its attempt count and
the three timeouts the schedule command sets.
-/

namespace Temporal.Feature.Nexus.Tests.Machines

open Umpire
open Umpire.Command
open Temporal.Feature.Nexus.Tests.Commands

/-! ### The product machine

What an operation does, with no account of how. Every Property written against it is carried to the
protocol machine by the refinement (`DESIGN.md` section 2.5, delivered by task `.6`). -/

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

machine nexusProduct
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  evidence:
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
    faultInjected: faultInjected
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep

/-! ### What the product machine says

The claims below are about the table, because the table is what Search, the Behavior Fingerprint and
Contract lowering read. A `match` arm that stopped saying what it says would fail here. -/

/- Six phases, and the four the design ends on. -/
#guard nexusProduct.table.states.length == 6
#guard nexusProduct.ends.length == 4

/- Every action class the machine steps on: six replies, three resolutions, and the two faults. -/
#guard nexusProduct.actionKeys.size == 11

/- An async reply starts the operation and records that it started. -/
#guard (handlerReplyStep { phase := .scheduled } .async).map (·.state.phase) == [.started]
#guard (handlerReplyStep { phase := .scheduled } .async).flatMap (·.facts) ==
  [.nexusOperationStarted]

/- A retryable handler error is invisible here: it is the protocol machine that backs off. -/
#guard handlerReplyStep { phase := .scheduled } (.handlerError (retryable := true)) == []

/- A completion after the operation is over is accepted and found nothing. -/
#guard (completeStep { phase := .succeeded } .succeeded).map (·.outcome) == [.notFound]
#guard (completeStep { phase := .succeeded } .succeeded).map (·.state.phase) == [.succeeded]

/- What the Model actually reaches. `timedOut` is not among them: the product machine has no timer,
because when an operation times out is the protocol's account of how and not what, so the state
exists for the refinement to map onto and nothing here reaches it. -/
#guard (Umpire.Command.reachableFrom nexusProduct.starts
    nexusProduct.transitions).map nexusProduct.stateKeyFor ==
  ["scheduled", "canceled", "failed", "succeeded", "started"]

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

How the server gets there: the retry the product machine cannot see, the three timers the schedule
command sets, and the attempt count a retryable failure raises. Written against the same actions, so
a Property proved on the product machine is carried here by the refinement.

Two parts of `DESIGN.md` section 3's machine are not here, for reasons recorded rather than silent.
The `cancel` field and its two rows are fn-79's. The `atConcurrencyLimit: true + schedule → reject`
row needs the table to vary with the setup, which is task `.5`; `setup:` is declared below because
the machine has that parameter, and the one-setup table this command builds is the table without
the rejection row.

The design's `none + schedule` row is a step from a phase rather than from no instance: a machine's
state structure has no "no instance yet" member, so `unscheduled` is that member. It is what makes
the three timeout fields reachable at anything but their first value -- the schedule command is what
sets them, so a machine that started already scheduled could never observe a timer firing. -/

enum Phase
  | unscheduled
  | scheduled
  | backingOff
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

/-- Which timer fired. The history event records it, so a Contract that did not check it would pass
a run that timed out on the wrong deadline. -/
enum TimeoutType
  | scheduleToClose
  | scheduleToStart
  | startToClose

/-- The attempt count is `attempts: count` in the design, bounded by the Limits. Nothing wires the
Limits into a machine's state yet (that is task `.4`'s Limits accounting), so the bound is written
here and the saturating successor `Umpire.Command.saturatingSucc` is what keeps a retry inside it. -/
abbrev attemptBound : Nat := 2

structure ProtocolState where
  phase : Phase
  attempts : Fin (attemptBound + 1)
  scheduleToClose : Timeout
  scheduleToStart : Timeout
  startToClose : Timeout
  deriving BEq, DecidableEq, Repr, Finite

enum ProtocolOutcome
  | accepted
  | notFound

enum ProtocolFact
  | nexusOperationScheduled
  | nexusOperationStarted
  | nexusOperationCompleted
  | nexusOperationFailed
  | nexusOperationCanceled
  | nexusOperationTimedOut (timeoutType : TimeoutType)
  | pendingAttempts
  | faultInjected

/-- The four phases the design ends on. A completion that arrives after one of them is not found. -/
private def terminalPhase (phase : Phase) : Bool :=
  phase == .succeeded || phase == .failed || phase == .canceled || phase == .timedOut

/-- Scheduled and not yet over: the phases a completion resolves and a timer can fire in. -/
private def running (phase : Phase) : Bool :=
  phase == .scheduled || phase == .backingOff || phase == .started

private def moves (state : ProtocolState) (phase : Phase) (recorded : List ProtocolFact) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  [{ outcome := .accepted, state := { state with phase }, facts := recorded }]

/-- The caller's schedule command. It names the operation's three deadlines, and every one of them
is a state field because whether a timer fires is a question about the operation and not about the
command that started it. -/
def scheduleStep (state : ProtocolState)
    (scheduleToClose scheduleToStart startToClose : Timeout) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .unscheduled then [] else
  [{ outcome := .accepted
     state := { phase := .scheduled, attempts := 0, scheduleToClose, scheduleToStart, startToClose }
     facts := [.nexusOperationScheduled] }]

/-- The handler's reply to the server's start request. What the product machine cannot see is the
last arm: a retryable failure backs the operation off and raises its attempt count, and the count is
read back through the `pendingAttempts` observation because no history event records it. -/
def protocolHandlerReplyStep (state : ProtocolState) (reply : Reply) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => moves state .succeeded [.nexusOperationCompleted]
  | .async => moves state .started [.nexusOperationStarted]
  | .operationFailed => moves state .failed [.nexusOperationFailed]
  | .operationCanceled => moves state .canceled [.nexusOperationCanceled]
  | .handlerError false => moves state .failed [.nexusOperationFailed]
  | .handlerError true =>
      [{ outcome := .accepted
         state := { state with phase := .backingOff, attempts := saturatingSucc state.attempts }
         facts := [.pendingAttempts] }]

/-- A transport fault is the same failure arriving as a dropped delivery rather than as a reply. -/
def protocolTransportFaultStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .scheduled then [] else
  [{ outcome := .accepted
     state := { state with phase := .backingOff, attempts := saturatingSucc state.attempts }
     facts := [.pendingAttempts] }]

/-- The handler's worker stopping is a fault the run records and the operation does not feel. -/
def protocolWorkerStopStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  [{ outcome := .accepted, state, facts := [.faultInjected] }]

/-- An asynchronous completion. Before a start, the server records a Started event first, which is
why the evidence is two facts and not one -- and why the product machine, which has no `backingOff`
phase to have skipped, could write the completion alone. -/
def protocolCompleteStep (state : ProtocolState) (resolution : Resolution) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if terminalPhase state.phase then
    [{ outcome := .notFound, state, facts := [] }]
  else if state.phase == .unscheduled then []
  else
    let startedFirst : List ProtocolFact :=
      if state.phase == .started then [] else [.nexusOperationStarted]
    match resolution with
    | .succeeded => moves state .succeeded (startedFirst ++ [.nexusOperationCompleted])
    | .failed => moves state .failed (startedFirst ++ [.nexusOperationFailed])
    | .canceled => moves state .canceled (startedFirst ++ [.nexusOperationCanceled])

/-- The backoff timer. It is what makes `backingOff` a phase the operation leaves rather than a
state it is stuck in, and it records nothing: a retry writes no history event. -/
def backoffStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .backingOff then [] else moves state .scheduled []

/-- The schedule-to-close deadline covers the whole operation, so it fires in every running phase --
and only when the schedule command set it. -/
def scheduleToCloseStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if running state.phase && state.scheduleToClose == .expires then
    moves state .timedOut [.nexusOperationTimedOut (timeoutType := .scheduleToClose)]
  else []

/-- The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
start. -/
def scheduleToStartStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if (state.phase == .scheduled || state.phase == .backingOff) &&
      state.scheduleToStart == .expires then
    moves state .timedOut [.nexusOperationTimedOut (timeoutType := .scheduleToStart)]
  else []

/-- The start-to-close deadline covers the handler's own work, so it begins at the start. -/
def startToCloseStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase == .started && state.startToClose == .expires then
    moves state .timedOut [.nexusOperationTimedOut (timeoutType := .startToClose)]
  else []

machine nexusProtocol
  for: operation
  state: ProtocolState
  starts: [unscheduled]
  ends: [succeeded, failed, canceled, timedOut]
  setup:
    atConcurrencyLimit: Bool
  timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
  unobservable: [backoff]
  evidence:
    nexusOperationScheduled: nexusOperationScheduled
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
    nexusOperationTimedOut: nexusOperationTimedOut
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    schedule: scheduleStep
    handlerReply: protocolHandlerReplyStep
    complete: protocolCompleteStep
    transportFault: protocolTransportFaultStep
    workerStop: protocolWorkerStopStep
    backoff: backoffStep
    scheduleToClose: scheduleToCloseStep
    scheduleToStart: scheduleToStartStep
    startToClose: startToCloseStep

/-! ### What the protocol machine says

Elaborating this machine costs about forty seconds on a four-core box: 192 states over 23 action
classes, enumerated into 1152 rows and checked against the canonical-table law. The cost is the law
rather than the walk -- the short circuit in `Umpire.Command.satisfiesTransitionRequirement` is what
made a machine this size provable at all. -/

/-- A state written the way a reader names one: the phase, and whichever fields are not at the value
the operation begins with. -/
private def at' (phase : Phase) (attempts : Fin (attemptBound + 1) := 0)
    (scheduleToClose : Timeout := .unset) (scheduleToStart : Timeout := .unset)
    (startToClose : Timeout := .unset) : ProtocolState :=
  { phase, attempts, scheduleToClose, scheduleToStart, startToClose }

/- Eight phases, three attempt counts and three deadlines, and the four phases the design ends on. -/
#guard nexusProtocol.table.states.length == 8 * (attemptBound + 1) * 2 * 2 * 2
#guard nexusProtocol.ends.length == 4 * (attemptBound + 1) * 2 * 2 * 2

/- Every action class the machine steps on. `schedule` contributes eight, one per assignment of the
three deadlines, because which timers an operation has is settled by the command that scheduled it
and an example is written at the class.

The catalog is in canonical order -- sorted by the member key, which is the order a Search admits --
rather than in the order the `steps:` lines are written, so it opens on the backoff timer and not on
`schedule`. -/
#guard nexusProtocol.actionKeys.size == 8 + 6 + 3 + 1 + 1 + 4
#guard nexusProtocol.actionKeys.toList.take 2 == ["backoff", "complete-canceled"]

/- The machine begins before the operation exists, with every deadline at its first value: the
schedule command is what sets them, so a machine that began already scheduled could never reach a
state a timer fires in. -/
#guard nexusProtocol.starts == [at' .unscheduled]

/- A retryable handler error backs the operation off and raises the attempt count. No history event
records that, which is why its evidence is the derived `pendingAttempts` observation. -/
#guard protocolHandlerReplyStep (at' .scheduled) (.handlerError (retryable := true)) ==
  [{ outcome := .accepted, state := at' .backingOff 1, facts := [.pendingAttempts] }]

/- The count saturates rather than wrapping: a machine whose attempt count rolled over to zero would
claim the operation had never been retried. -/
#guard (protocolHandlerReplyStep (at' .scheduled (attempts := Fin.last attemptBound))
  (.handlerError (retryable := true))).map (·.state.attempts) == [Fin.last attemptBound]

/- A completion that arrives before the start records the Started event first, and one that arrives
after it does not. Both record the completion. -/
#guard (protocolCompleteStep (at' .backingOff 1) .succeeded).flatMap (·.facts) ==
  [.nexusOperationStarted, .nexusOperationCompleted]
#guard (protocolCompleteStep (at' .started) .succeeded).flatMap (·.facts) ==
  [.nexusOperationCompleted]

/- A completion after the operation is over is not found and changes nothing. -/
#guard protocolCompleteStep (at' .timedOut) .succeeded ==
  [{ outcome := .notFound, state := at' .timedOut, facts := [] }]

/- A timer fires only when the schedule command set it, and each covers its own span: the
start-to-close deadline begins at the start, so it cannot fire before one. -/
#guard startToCloseStep (at' .scheduled (startToClose := .expires)) == []
#guard (startToCloseStep (at' .started (startToClose := .expires))).map (·.state.phase) == [.timedOut]
#guard scheduleToCloseStep (at' .started) == []

/- Which timer fired is recorded, because the history event records it and a Contract that did not
check it would pass a run that timed out on the wrong deadline. -/
#guard (scheduleToStartStep (at' .scheduled (scheduleToStart := .expires))).flatMap (·.facts) ==
  [.nexusOperationTimedOut (timeoutType := .scheduleToStart)]

/- Nothing is stuck. Unlike the product machine's, this claim can fail: `workerStop` steps from every
state there, and here the operation's own phase decides which steps are enabled. -/
#guard nexusProtocol.stuck == none

/- Not every state is reachable, and that is what enumerating the structure rather than the walk
costs: the count and the deadlines are fields of the state, so states that disagree about them exist
in the type and no run produces them. The Behavior Fingerprint reads the table, so this number is
part of the Model's identity. -/
#guard (Umpire.Command.reachableFrom nexusProtocol.starts nexusProtocol.transitions).length == 158

/- The canonical-table law holds, and holds by a proof rather than by a logged failure. -/
/-- info: 'Temporal.Feature.Nexus.Tests.Machines.nexusProtocol' depends on axioms: [propext] -/
#guard_msgs in
#print axioms nexusProtocol

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
  model: attemptLoop
  when: workerStop
  require:
    state: finished

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
#guard (Umpire.Command.members (α := Fin (attemptBound + 1))).map Umpire.Command.limitReached ==
  [false, false, true]
#guard Umpire.Command.limitReached
  (Umpire.Command.saturatingSucc (Fin.last attemptBound)) == true

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
error: 'nexusOperationDreamt' is neither a recorded event kind the realization carries nor an observation this Model declares; evidence names recorded data, so it is a generated history event kind, a Testpilot Run Event kind, or a derived `observation`
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

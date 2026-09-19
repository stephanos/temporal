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
  ["scheduled", "succeeded", "started", "failed", "canceled"]

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

end Temporal.Feature.Nexus.Tests.Machines

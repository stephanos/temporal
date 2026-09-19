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

/- The Model reaches every one of its end states. Nothing here is stuck, which is what the command
checked before it wrote the table. -/
#guard nexusProduct.stuck == none

end Temporal.Feature.Nexus.Tests.Machines

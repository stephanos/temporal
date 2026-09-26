import Temporal.Feature.Nexus.Caller.Model

/-!
# Two Nexus operations of one workflow

One workflow schedules two Nexus operations and awaits both, each answered asynchronously by its
own handler and completed by the caller. The Model is the caller Model's `operation` entity and
action classes over a machine that keeps only the asynchronous success path, run over two instances
of the entity so their steps interleave; what it adds is one claim the typed Nexus example (fn-83)
wrote by hand as a captured field Property: a completion references the scheduled event its own
operation was scheduled at. That is a field relation between two recorded events of one
operation -- the completed event's `scheduled_event_id` and the scheduled event's own `event_id`
-- and the rule the Case carries captures each instance's scheduled event by the operation name it
records, then matches the completion's reference against it.
-/

namespace Temporal.Feature.Nexus.Pair

open Umpire
open Umpire.Command
open Temporal.Feature.Nexus.Caller

/-! ### The machine

The operation without its deadlines and retries: scheduled, then settled by the handler's reply --
at once, or asynchronously by the caller's completion. Every class of the caller Model's actions the
machine steps on has a row, because admission requires one; the path the pair runs is the
asynchronous success. -/

enum PairPhase
  | unscheduled
  | scheduled
  | started
  | succeeded
  | failed
  | canceled

structure PairState where
  phase : PairPhase
  deriving BEq, DecidableEq, Repr, Finite

enum PairOutcome
  | accepted
  | notFound

enum PairFact
  | nexusOperationScheduled
  | nexusOperationStarted
  | nexusOperationCompleted
  | nexusOperationFailed
  | nexusOperationCanceled

private def moves (phase : PairPhase) (recorded : List PairFact) :
    List (Step PairState PairOutcome PairFact) :=
  [{ outcome := .accepted, state := { phase }, facts := recorded }]

/-- The caller's schedule command. The deadlines it sets are not modeled here: no timer is. -/
def pairScheduleStep (state : PairState)
    (_scheduleToClose _scheduleToStart _startToClose : Timeout) :
    List (Step PairState PairOutcome PairFact) :=
  if state.phase != .unscheduled then [] else moves .scheduled [.nexusOperationScheduled]

/-- The handler's reply. A retryable error leaves the operation scheduled and records nothing:
the retry is the caller Model's account, not this one's. -/
def pairHandlerReplyStep (state : PairState) (reply : Reply) :
    List (Step PairState PairOutcome PairFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => moves .succeeded [.nexusOperationCompleted]
  | .async => moves .started [.nexusOperationStarted]
  | .operationFailed => moves .failed [.nexusOperationFailed]
  | .operationCanceled => moves .canceled [.nexusOperationCanceled]
  | .handlerError false => moves .failed [.nexusOperationFailed]
  | .handlerError true => [{ outcome := .accepted, state, facts := [] }]

/-- The caller's completion of a started operation. One that arrives after the operation is over
is not found. -/
def pairCompleteStep (state : PairState) (resolution : Resolution) :
    List (Step PairState PairOutcome PairFact) :=
  if state.phase == .succeeded || state.phase == .failed || state.phase == .canceled then
    [{ outcome := .notFound, state, facts := [] }]
  else if state.phase != .started then [] else
  match resolution with
  | .succeeded => moves .succeeded [.nexusOperationCompleted]
  | .failed => moves .failed [.nexusOperationFailed]
  | .canceled => moves .canceled [.nexusOperationCanceled]

machine pair
  for: operation
  state: PairState
  starts: [unscheduled]
  ends: [succeeded, failed, canceled]
  evidence:
    nexusOperationScheduled: nexusOperationScheduled
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
  steps:
    schedule: pairScheduleStep
    handlerReply: pairHandlerReplyStep
    complete: pairCompleteStep

/-! ### What the machine promises -/

/- The completion settles the operation and the completed event records it. -/
property completed
  machine: pair
  when: complete (succeeded)
  holds: fun step =>
    step.state.phase == .succeeded && step.facts.contains .nexusOperationCompleted

/- A completion references the scheduled event its own operation was scheduled at. The scheduled
event is an earlier step's, so the rule captures each instance's own -- selected by the operation
name the schedule command assigned -- and matches the completion's reference against its id. -/
property completionReferencesSchedule
  machine: pair
  when: complete (succeeded)
  relates: nexusOperationCompleted.scheduled_event_id = nexusOperationScheduled.event_id

/-! ### The path, the Query and the set

Two instances of the operation, each scheduled, started and completed, their steps interleaved as
one workflow performs them: both schedules, both replies, both completions. -/

scenario twoAsync
  model: pair
  instances: 2
  starts: unscheduled
  actions: [schedule (unset, unset, unset) 1, schedule (unset, unset, unset) 2,
    handlerReply (async) 1, handlerReply (async) 2, complete (succeeded) 1,
    complete (succeeded) 2]

limits six
  steps: 6
  actions: 6
  search: 64

/- The rule the relation lowers to reads what history records, and a completed event records the
scheduled event it references but not the operation that was scheduled: which the two Known Gaps
below say, as the hand-written Case said them. -/
query bothComplete
  find: completed
  in: twoAsync
  limits: six
  gap: interpretation
    code: "completion-identity-is-unrecorded"
    subject: "completed"
    detail: "the bounded-response window runs online from lifted history evidence; a completed event records no operation identity, so the operation key is the scheduled event it references and the projection releases one representative completed step"
  gap: interpretation
    code: "crossed-completion-is-inconclusive"
    subject: "completionReferencesSchedule"
    detail: "a completed event carries no operation identity, so a completion referencing another scheduled event leaves the rule pending rather than violated; the model Property still distinguishes the two"

set nexusPairTests
  purpose: functional
  bind:
    caller: driven
    handler: driven
  queries: [bothComplete]

/- The caller realization serves the pair: each instance's schedule, handler and completion carry
the instance, so the two operations never share an instruction, a slot or a handler. -/
case nexusPairCases
  realizes nexusPairTests
  as (Temporal.Case.Realization.asyncNexus "umpire.case.service" "complete")

end Temporal.Feature.Nexus.Pair

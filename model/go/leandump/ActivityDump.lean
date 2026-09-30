-- Copied from .plans/archive/cmp/lean/StandaloneActivity.lean for the Go parity harness, with the three
-- `case` blocks, the `schema:` lines and everything from the protocol machine on removed, and a
-- dump `main` appended. The schemas are removed
-- because no activity RPC message is reachable from the roots in model/Temporal/Case/Schema.lean,
-- so the checked `schema:` line rejects every action that names one; the tables, Queries and
-- targets this harness compares do not read schemas. Run by activity-dump.sh; not part of model/.
-- authoring: header
import Temporal.Case.Syntax
import Temporal.Feature.Worker.Model
import Lean.Data.Json

/-!
# The standalone activity Model

One activity started directly through `StartActivityExecution`, with no workflow around it: the
product machine says what an activity does as `DescribeActivityExecution` reports it, the protocol
machine says how the server gets there and refines it, and the functional set runs one Query per
side effect that settles the activity. Grounded in `chasm/lib/activity/statemachine.go`; reset is
deferred, like cancellation in the Nexus caller Model, and the heartbeat timeout is not modeled.

A standalone activity writes no history event. Every fact below is a status read through
`DescribeActivityExecution`, or a result read through `PollActivityExecution`, so the evidence lines
of the machines name observations rather than events.

Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
-/

namespace Temporal.Feature.Activity.Standalone

open Umpire
open Umpire.Command

-- authoring: entities

/-! ### Entities

An activity is named by the id the caller chose for it: every status read and every result read
carries it, and no run id or event id is needed to tell two apart. -/

entity activity
  key: activityId

-- authoring: domains

/-! ### The input domains

As in the caller Model, a constructor with a finite field contributes one class per assignment:
`failed (retryable : Bool)` is two classes, which mirrors the retryable flag of an
`ApplicationFailure`. -/

enum Timeout
  | unset
  | expires

enum AttemptResult
  | completed
  | failed (retryable : Bool)
  | canceled

enum Delivery
  | accepted
  | notFound

enum Control
  | pause
  | unpause
  | requestCancel
  | terminate

-- authoring: actions

/-! ### Actions

Parties: `caller` starts and controls the activity, `worker` runs its attempts, `system` owns the
timers. The worker's stop is an ordinary action of the `worker` party, as in every other Model. -/

action start
  party: caller
  creates: activity
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout

/-- The worker's poll receives the task for the current attempt. -/
action attemptStart
  party: worker
  on: activity

action attemptResult
  party: worker
  on: activity
  input:
    result: AttemptResult
  examples:
    failed (retryable := false) → ApplicationFailureNonRetryable
    failed (retryable := true) → ApplicationFailureRetryable

/-- The four caller-side controls are one action with a finite input, because they share a result:
a control on an activity that is over is not found. -/
action control
  party: caller
  on: activity
  input:
    control: Control
  results: Delivery

/-- The worker stops polling. Nothing recorded names the activity, so the machines keep their state
and record nothing at it. -/
action workerStop
  party: worker

-- authoring: observation

/-! ### The derived observation

A retried attempt writes nothing the caller can see except the attempt count that
`DescribeActivityExecution` reports, so it is the one derived observation. -/

observation attemptCount
  on: activity
  read: attempt

/-- The status reads. The Temporal evidence catalog admits history event kinds, Run Event kinds and
declared observations; a Describe status is none of the first two, so each status a machine records
is declared as an observation of the `status` field. Whether the catalog accepts several
observations over one field is not verified; this is the one place the file departs from checked
grammar into a guess about the realization layer. -/
observation statusScheduled
  on: activity
  read: status

observation statusStarted
  on: activity
  read: status

observation statusPaused
  on: activity
  read: status

observation statusCancelRequested
  on: activity
  read: status

observation statusCompleted
  on: activity
  read: status

observation statusFailed
  on: activity
  read: status

observation statusCanceled
  on: activity
  read: status

observation statusTerminated
  on: activity
  read: status

observation statusTimedOut
  on: activity
  read: status

-- authoring: product

/-! ### The product machine

What the caller sees through `DescribeActivityExecution`, with no account of how: a retry reads
as scheduled again, and a pause requested of a running attempt reads as started until the worker
yields. -/

enum ProductPhase
  | scheduled
  | started
  | paused
  | cancelRequested
  | completed
  | failed
  | canceled
  | terminated
  | timedOut

structure ProductState where
  phase : ProductPhase
  deriving BEq, DecidableEq, Repr, Finite

enum ProductOutcome
  | accepted
  | notFound

enum ProductFact
  | statusScheduled
  | statusStarted
  | statusPaused
  | statusCancelRequested
  | statusCompleted
  | statusFailed
  | statusCanceled
  | statusTerminated
  | statusTimedOut

private def productStep (phase : ProductPhase) (recorded : ProductFact) :
    List (Step ProductState ProductOutcome ProductFact) :=
  [{ outcome := .accepted, state := { phase }, facts := [recorded] }]

/-- The phases the product machine ends on. -/
def productTerminal (state : ProductState) : Bool :=
  state.phase == .completed || state.phase == .failed || state.phase == .canceled ||
    state.phase == .terminated || state.phase == .timedOut

/-- A worker takes the attempt of a scheduled activity. -/
def attemptStartStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .scheduled then [] else productStep .started .statusStarted

/-- The worker's answer to the attempt. Unlike the Nexus caller, a retryable failure is visible
here: `DescribeActivityExecution` reads `SCHEDULED` again with a higher attempt count
(`TransitionRescheduled`), and under a cancel request it settles the activity as canceled. The
backoff between the two is what the protocol machine adds. A canceled answer settles only an
activity whose cancellation was requested. -/
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

/-- A control on an activity that is over is not found and changes nothing. -/
def controlStep (state : ProductState) (control : Control) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if productTerminal state then
    [{ outcome := .notFound, state, facts := [] }]
  else
    match control with
    | .pause =>
        if state.phase == .scheduled || state.phase == .started then
          productStep .paused .statusPaused
        else []
    | .unpause =>
        if state.phase == .paused then productStep .scheduled .statusScheduled else []
    | .requestCancel =>
        if state.phase == .scheduled || state.phase == .started || state.phase == .paused ||
            state.phase == .cancelRequested then
          productStep .cancelRequested .statusCancelRequested
        else []
    | .terminate => productStep .terminated .statusTerminated

/-- The worker stopping is a fault the Run records and the activity does not feel. -/
def workerStopStep (_state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) := []

/-- One of the activity's deadlines firing. Which deadline is the protocol's account of how. -/
def timeoutStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase == .scheduled || state.phase == .started || state.phase == .cancelRequested ||
      state.phase == .paused then
    productStep .timedOut .statusTimedOut
  else []

machine activityProduct
  for: activity
  state: ProductState
  starts: [scheduled]
  ends: [completed, failed, canceled, terminated, timedOut]
  timers: [timeout]
  evidence:
    statusScheduled: statusScheduled
    statusStarted: statusStarted
    statusPaused: statusPaused
    statusCancelRequested: statusCancelRequested
    statusCompleted: statusCompleted
    statusFailed: statusFailed
    statusCanceled: statusCanceled
    statusTerminated: statusTerminated
    statusTimedOut: statusTimedOut
  steps:
    attemptStart: attemptStartStep
    attemptResult: attemptResultStep
    control: controlStep
    workerStop: workerStopStep
    timeout: timeoutStep

-- authoring: protocol (left out)

/- Everything from the protocol machine on is left out. Lean's `machine` command refuses the protocol
state: `ProtocolState` has 12 phases x 3 attempt counts x 2 x 2 x 2 deadlines = 288 members, and
`Umpire.Command.elaborationBound` is 256, a constant with no option to raise it. The Queries, the
sets and the composition all name the protocol machine, so none of them elaborate either. -/

end Temporal.Feature.Activity.Standalone

namespace UmpireGo.ActivityDump

open Lean (Json)
open Umpire
open Umpire.Command
open Temporal.Feature.Activity.Standalone

def str (s : String) : Json := Json.str s
def strs (xs : List String) : Json := Json.arr (xs.map str).toArray

def catalogKey {α : Type} [BEq α] (catalog : FiniteCatalog α) (value : α) : String :=
  ((catalog.find? (·.value == value)).map (·.key)).getD "?"

partial def reach {St A O F : Type} [BEq St] (rows : List (FiniteTransitionRow St A O F))
    (seen : List St) : List St :=
  let grown := rows.foldl (init := seen) fun seen row =>
    if seen.contains row.source then
      row.results.foldl (init := seen) fun seen result =>
        if seen.contains result.state then seen else seen ++ [result.state]
    else seen
  if grown.length == seen.length then seen else reach rows grown

def tableJson {S St A O F : Type} [BEq S] [BEq St] [BEq A] [BEq O] [BEq F]
    (name : String) (model : DeclaredModel S St A O F) : Json :=
  let t := model.table
  let sk := catalogKey t.states
  let ak := catalogKey t.actions
  let ok := catalogKey t.outcomes
  let fk := catalogKey t.facts
  Json.mkObj [
    ("machine", str name),
    ("states", strs (t.states.map (·.key))),
    ("actions", strs (t.actions.map (·.key))),
    ("outcomes", strs (t.outcomes.map (·.key))),
    ("facts", strs (t.facts.map (·.key))),
    ("starts", strs (model.initial.map sk)),
    ("ends", strs (model.terminal.map sk)),
    ("reachable", strs ((reach t.transitions model.initial).map sk)),
    ("transitions", Json.arr (t.transitions.map fun row => Json.mkObj [
      ("key", str row.key),
      ("source", str (sk row.source)),
      ("action", str (ak row.action)),
      ("results", Json.arr (row.results.map fun step => Json.mkObj [
        ("outcome", str (ok step.outcome)),
        ("state", str (sk step.state)),
        ("facts", strs (step.facts.map fk))]).toArray)]).toArray)]

def idsJson {S St A O F : Type} [BEq S] [BEq St] [BEq A] [BEq O] [BEq F]
    (model : DeclaredModel S St A O F) : Json :=
  Json.mkObj [
    ("target", str model.targetId.value),
    ("states", strs (model.stateIds.map (·.value))),
    ("stateFields", Json.arr (model.stateFieldIds.map fun (n, id) =>
      Json.arr #[str n, str id.value]).toArray),
    ("actions", strs (model.actionIds.map (·.value))),
    ("outcomes", strs (model.outcomeIds.map (·.value))),
    ("facts", strs (model.factIds.map (·.value)))]

def atomJson (a : ModelValue) : Json :=
  Json.mkObj [("id", str a.definitionId.value), ("value", str a.value)]

def traceJson (t : Scenario.Trace) : Json :=
  Json.mkObj [
    ("initial", atomJson t.trace.initialState),
    ("steps", Json.arr (t.trace.steps.map fun s => Json.mkObj [
      ("action", atomJson s.selectedAction),
      ("outcome", atomJson s.outcome),
      ("state", atomJson s.state),
      ("facts", Json.arr (s.facts.map atomJson).toArray)]).toArray)]

def write (dir name : String) (j : Json) : IO Unit :=
  IO.FS.writeFile (dir ++ "/" ++ name) (j.pretty ++ "\n")

end UmpireGo.ActivityDump

open UmpireGo.ActivityDump in
open Temporal.Feature.Activity.Standalone in
def main (args : List String) : IO UInt32 := do
  let dir := args.headD "."
  write dir "activity-table-activityProduct.json" (tableJson "activityProduct" activityProduct)
  write dir "activity-ids-activityProduct.json" (idsJson activityProduct)
  return 0

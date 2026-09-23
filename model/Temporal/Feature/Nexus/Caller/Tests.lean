import Temporal.Feature.Nexus.Caller.Model

/-!
# What the Caller Model says

The claims below are about the tables, because the tables are what Search, the Behavior
Fingerprint and Contract lowering read. A `match` arm that stopped saying what it says would fail
here. The pins on the protocol machine, its refinement and the product Property read on protocol
paths moved here from `Temporal.Feature.Nexus.Tests.Machines` with the machines (fn-85 .10).
-/

namespace Temporal.Feature.Nexus.Caller.Tests

open Umpire
open Umpire.Case
open Umpire.Command
open Temporal.Feature.Nexus.Caller

/-! ### The product machine -/

/- Six phases, and the four the design ends on. -/
#guard nexusProduct.table.states.length == 6
#guard nexusProduct.ends.length == 4

/- Every action class the machine steps on: six replies, three resolutions, the two faults it cannot
see, and the one timer. -/
#guard nexusProduct.actionKeys.size == 12

/- A retryable handler error is invisible here: it is the protocol machine that backs off. -/
#guard handlerReplyStep { phase := .scheduled } (.handlerError (retryable := true)) == []

/- What the Model actually reaches: every phase. -/
#guard (Umpire.Command.reachableFrom nexusProduct.starts
    nexusProduct.transitions).map nexusProduct.stateKeyFor ==
  ["scheduled", "canceled", "failed", "succeeded", "started", "timedOut"]

/-- info: 'Temporal.Feature.Nexus.Caller.nexusProduct' depends on axioms: [propext] -/
#guard_msgs in
#print axioms nexusProduct

#guard nexusProduct.stuck == none

/-! ### The protocol machine -/

/-- A state written the way a reader names one: the phase, and whichever fields are not at the value
the operation begins with. -/
private def at' (phase : Phase) (attempts : Fin (attemptBound + 1) := 0)
    (scheduleToClose : Timeout := .unset) (scheduleToStart : Timeout := .unset)
    (startToClose : Timeout := .unset) : ProtocolState :=
  { phase, attempts, scheduleToClose, scheduleToStart, startToClose }

/- Eight phases, three attempt counts and three deadlines, and the four phases the design ends on. -/
#guard nexusProtocol.table.states.length == 8 * (attemptBound + 1) * 2 * 2 * 2
#guard nexusProtocol.ends.length == 4 * (attemptBound + 1) * 2 * 2 * 2

/- Every action class the machine steps on: eight schedule commands, one per assignment of the three
deadlines, six replies, three resolutions, the two faults and the four timers. The catalog is in
canonical order, so it opens on the backoff timer rather than on `schedule`. -/
#guard nexusProtocol.actionKeys.size == 8 + 6 + 3 + 1 + 1 + 4
#guard nexusProtocol.actionKeys.toList.take 2 == ["backoff", "complete-canceled"]

/- The machine begins before the operation exists, with every deadline at its first value. -/
#guard nexusProtocol.starts == [at' .unscheduled]

/- The machine carries no setup parameter: the concurrency-limit rejection is not modeled. -/
#guard nexusProtocol.setupParameters == []

/- A retryable handler error backs the operation off and raises the attempt count. No history event
records that, which is why its evidence is the derived `pendingAttempts` observation. -/
#guard protocolHandlerReplyStep (at' .scheduled) (.handlerError (retryable := true)) ==
  [{ outcome := .accepted, state := at' .backingOff 1, facts := [.pendingAttempts] }]

/- The count saturates rather than wrapping. -/
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

/- A timer fires only when the schedule command set it, and each covers its own span. -/
#guard startToCloseStep (at' .scheduled (startToClose := .expires)) == []
#guard (startToCloseStep (at' .started (startToClose := .expires))).map (·.state.phase) == [.timedOut]
#guard scheduleToCloseStep (at' .started) == []

/- Which timer fired is recorded. -/
#guard (scheduleToStartStep (at' .scheduled (scheduleToStart := .expires))).flatMap (·.facts) ==
  [.nexusOperationTimedOut (timeoutType := .scheduleToStart)]

/- The handler's worker stopping keeps the state and records nothing: the Run records the fault, but
nothing recorded names the operation. The product machine does not see it at all. -/
#guard protocolWorkerStopStep (at' .scheduled (scheduleToStart := .expires)) ==
  [{ outcome := .accepted, state := at' .scheduled (scheduleToStart := .expires), facts := [] }]
#guard workerStopStep { phase := .scheduled } == []

/- Nothing is stuck: the operation's own phase decides which steps are enabled. -/
#guard nexusProtocol.stuck == none

/- Not every state is reachable: the count and the deadlines are fields of the state, so states that
disagree about them exist in the type and no run produces them. The Behavior Fingerprint reads the
table, so this number is part of the Model's identity. -/
#guard (Umpire.Command.reachableFrom nexusProtocol.starts nexusProtocol.transitions).length == 158

/-- info: 'Temporal.Feature.Nexus.Caller.nexusProtocol' depends on axioms: [propext] -/
#guard_msgs in
#print axioms nexusProtocol

/-! ### The refinement

`refines: nexusProduct` with `map: productOf` walked every protocol row through the map and derived
the step mapping: a row whose mapped states are a product step is that step, a row whose mapped
states are equal is a stutter. -/

#guard nexusProtocol.refinement.rejected == none
#guard nexusProtocol.refinement.rows.length == nexusProtocol.transitions.length

/- A reply the product machine sees is that reply's step. A retry it cannot see is a stutter, and so
are the schedule command and the backoff timer. -/
#guard nexusProtocol.refinement.rows.lookup "scheduled-0-unset-unset-unset-handlerReply-async" ==
  some (some "handlerReply-async")
#guard nexusProtocol.refinement.rows.lookup
  "scheduled-0-unset-unset-unset-handlerReply-handlerError-true" == some none
#guard nexusProtocol.refinement.rows.lookup
  "unscheduled-0-unset-unset-unset-schedule-unset-unset-expires" == some none
#guard nexusProtocol.refinement.rows.lookup "backingOff-1-unset-unset-unset-backoff" == some none

/- A deadline firing is the product's one timer, whichever deadline it was. -/
#guard nexusProtocol.refinement.rows.lookup "started-0-unset-unset-expires-startToClose" ==
  some (some "timeout")

/- A completion before the start records the Started event first and still carries the step. -/
#guard nexusProtocol.refinement.rows.lookup "backingOff-1-unset-unset-unset-complete-succeeded" ==
  some (some "complete-succeeded")

/- The rows the product machine does not see: every schedule command, every retry, every backoff
and every worker stop. -/
#guard (nexusProtocol.refinement.rows.filter (·.2.isNone)).length == 24 * 8 + 24 + 24 + 24 + 192

/-- The phase a refinement row leaves from: the first segment of its key. -/
private def rowPhase (row : String × Option String) : String :=
  ((row.1.splitOn "-").head?).getD ""

/- **Stutter invariance.** A product Property read on the protocol machine is checked on every row,
stutters included, and on a stutter the mapped state before and after are equal. `terminalIsFinal`
triggers only at a terminal prior state, and the only stutters that leave one are the worker stop's,
which keep the phase: every other row the product machine does not see leaves from a phase that
reads as `scheduled`. That is the fact the invariance argument rests on
(`UMPIRE4_RESEARCH_NEXUS_MODEL.md` section 2), pinned rather than assumed. -/
#guard (nexusProtocol.refinement.rows.filter (·.2.isNone)).all fun row =>
  ["unscheduled", "scheduled", "backingOff"].contains (rowPhase row) ||
    row.1.endsWith "-workerStop"

/- The product state a protocol state reads as is a field of the protocol state, named after the
product machine. -/
#guard nexusProtocol.stateFieldIds.map (·.1) ==
  ["phase", "attempts", "scheduleToClose", "scheduleToStart", "startToClose", "nexusProduct"]

/--
info: 'Temporal.Feature.Nexus.Caller.nexusProtocol.refines' depends on axioms: [propext, Quot.sound]
-/
#guard_msgs in
#print axioms nexusProtocol.refines

/-! ### The Properties and the Queries -/

/- One group per terminal phase, each fixing the phase it leaves from. -/
#guard terminalIsFinal.names.groups.length == 4

/- A protocol Scenario names its classed actions with their inputs, and its start by its phase. -/
#guard asyncThenSucceeded.names.setupState == "unscheduled-0-unset-unset-unset"
#guard asyncThenSucceeded.names.occurrences.map (·.action) ==
  ["schedule-unset-unset-unset", "handlerReply-async", "complete-succeeded"]

/- Each functional Query finds its claim on its path. -/
#guard (match syncCompletion with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"
#guard (match asyncCompletion with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"
#guard (match asyncFailure with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"
#guard (match handlerError with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"
#guard (match retry with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"
#guard (match scheduleToStartTimeout with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"
#guard (match startToCloseTimeout with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"

/- A timer is named under `when:` like any action, and a Scenario lists it where it fires. -/
#guard retriedThenSucceeded.names.occurrences.map (·.action) ==
  ["schedule-unset-unset-unset", "handlerReply-handlerError-true", "backoff",
    "handlerReply-syncSuccess"]
#guard scheduleToStartExpires.names.occurrences.map (·.action) ==
  ["schedule-unset-expires-unset", "workerStop", "scheduleToStart"]

/- The product claim is verified over every trace of the asynchronous path, and keeps its own
identity: it is the product Property and no other. -/
#guard (match terminalHolds with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "verified-within-limits"
#guard (terminalHolds.toOption.map fun checked => checked.property.id.value) ==
  some "temporal.nexus.caller.property.terminalIsFinal"

/- A product Property about an action the protocol machine does not have cannot be read there. -/
property timesOut
  machine: nexusProduct
  when: timeout
  holds: fun step => step.state.phase == .timedOut

/--
error: the Property names the action 'timeout' of 'Temporal.Feature.Nexus.Caller.nexusProduct', and 'Temporal.Feature.Nexus.Caller.nexusProtocol' has no action of that name; a Property on the refined machine is read on the refining one through the values of the same name, and a state through its `map:`
-/
#guard_msgs in
query timesOutOnProtocol
  find: timesOut
  in: asyncThenSucceeded
  limits: three

/-! ### The sets

A canary is admitted when a deployment can close every gap its Cases carry: the handler is
`observed`, and every step of the sync and async completion paths records evidence. The Cases are
produced under the functional set's realization and registered apart from the functional fixtures,
rendered only on request. -/

#guard nexusCallerCanary.purpose == .canary
#guard nexusCallerCanary.bindings ==
  [("caller", .driven), ("handler", .observed), ("network", .observed), ("worker", .driven)]
#guard [nexusCallerCanaryCases.syncCompletion, nexusCallerCanaryCases.asyncCompletion].all
  fun produced => Temporal.Case.whiteBoxGaps produced == []

/- A Query whose path takes a silent step carries a capability gap no deployment closes, so a
canary naming it is rejected at the block, naming the Query and the gap. -/
set canaryRetry
  purpose: canary
  bind:
    caller: driven
    handler: observed
    network: observed
    worker: driven
  queries: [retry]

/--
error: Query 'Temporal.Feature.Nexus.Caller.retry' cannot be a canary: its Case carries the white-box Known Gap 'temporal.nexus.caller.action.nexusProtocol.backoff.unobserved' (capability), a step of its path that no observation confirms; a canary runs against a deployment the Case does not drive, so leave the Query out or give the step evidence
-/
#guard_msgs in
case canaryRetryCases
  realizes canaryRetry
  as (Temporal.Case.Realization.asyncNexus "umpire.case.service" "complete")

/- The exploration covers the protocol machine under `four`: the rows within three steps of a
start, in table order, then the results those rows reach and the claims their actions make, well
under the search count. The golden `Fixtures/CallerExploratoryCoverage.json` pins the list. -/
#guard nexusCallerExploration.purpose == .exploratory
#guard nexusCallerExploration.machine.map (·.value) ==
  some "temporal.nexus.caller.machine.nexusProtocol"
#guard nexusCallerExploration.cover == [.rows, .results, .classMembers]
#guard nexusCallerExploration.budget == some "four"
#guard nexusProtocol.table.transitions.length == 1152
#guard nexusCallerExploration.targets.length == 885 + 2 + 2
#guard (nexusCallerExploration.targets.map CoverageTarget.kind).eraseDups ==
  ["row", "result", "classMember"]
#guard (nexusCallerExploration.targets.filter (·.kind == "row")).length == 885
#guard nexusCallerExploration.targets.length ≤ four.search.value
#guard (nexusCallerExploration.targets.filterMap fun target => match target with
    | .result outcome => some outcome.value
    | _ => none) ==
  ["temporal.nexus.caller.outcome.nexusProtocol.accepted",
    "temporal.nexus.caller.outcome.nexusProtocol.notFound"]
#guard (nexusCallerExploration.targets.filterMap fun target => match target with
    | .classMember _ action field className exampleValue =>
        some (action, field, className, exampleValue)
    | _ => none) ==
  [("temporal.nexus.caller.action.handlerReply", "reply", "handlerError (retryable := false)",
      "BadRequest"),
    ("temporal.nexus.caller.action.handlerReply", "reply", "handlerError (retryable := true)",
      "Internal")]

/-! ### The Cases

One realization serves the seven Queries, and each Case's Program is the path's: the completion
classes land on the controller between the wait for the authority and the close-event read, and
that wait is emitted only where a completion is on the path; the attempt-count poll only where a
retryable failure is; the worker stop, before the workflow starts, only where the path stops the
worker. Every Case reads the scheduled event as soon as it exists, so the evidence that opens the
operation is lifted before any poll that follows it. -/

private def instructionIds (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case)
    (entrypointId : String) : List String :=
  match produced with
  | .ok output =>
      ((output.program.bind fun (program : temporal.server.api.testpilot.v1.Program) =>
        program.entrypoints.find? (·.entrypoint_id == entrypointId)).map fun entrypoint =>
          entrypoint.instructions.toList.map (·.instruction_id)).getD []
  | .error _ => []

#guard instructionIds nexusCallerCases.syncCompletion "controller" ==
  ["start-workflow", "await-scheduled", "await-close", "history"]
#guard instructionIds nexusCallerCases.syncCompletion "handler" == ["respond-sync"]
#guard instructionIds nexusCallerCases.asyncCompletion "controller" ==
  ["start-workflow", "await-scheduled", "await-completion-authority", "complete-nexus-operation",
    "await-close", "history"]
#guard instructionIds nexusCallerCases.asyncFailure "controller" ==
  ["start-workflow", "await-scheduled", "await-completion-authority", "fail-nexus-operation",
    "await-close", "history"]
#guard instructionIds nexusCallerCases.handlerError "handler" == ["respond-error"]
#guard instructionIds nexusCallerCases.retry "controller" ==
  ["start-workflow", "await-scheduled", "pending-attempts", "await-close", "history"]
#guard instructionIds nexusCallerCases.retry "handler" == ["respond-error-retryable", "respond-sync"]
#guard instructionIds nexusCallerCases.scheduleToStartTimeout "controller" ==
  ["stop-handler-worker", "start-workflow", "await-scheduled", "await-close", "history"]
#guard instructionIds nexusCallerCases.scheduleToStartTimeout "handler" == []
#guard instructionIds nexusCallerCases.startToCloseTimeout "controller" ==
  ["start-workflow", "await-scheduled", "await-close", "history"]
#guard instructionIds nexusCallerCases.startToCloseTimeout "handler" == ["respond-async"]

/-- The evidence kinds a Case declares, in the order its witness records them. -/
private def declaredKinds
    (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) : List String :=
  match produced with
  | .ok output =>
      (output.program.map fun (program : temporal.server.api.testpilot.v1.Program) =>
        program.evidence.toList.map (·.evidence_id)).getD []
  | .error _ => []

/- The evidence each Case lifts is read off the machine's `evidence:` lines along its witness: one
declaration per recorded kind, the scheduled event first. A kind read by a poll keeps its
`evidence.` prefix under the Case's local names, a history kind is named by its event. -/
#guard declaredKinds nexusCallerCases.syncCompletion == ["evidence.scheduled", "completed"]
#guard declaredKinds nexusCallerCases.asyncCompletion ==
  ["evidence.scheduled", "started", "completed"]
#guard declaredKinds nexusCallerCases.asyncFailure == ["evidence.scheduled", "started", "failed"]
#guard declaredKinds nexusCallerCases.handlerError == ["evidence.scheduled", "failed"]
#guard declaredKinds nexusCallerCases.retry ==
  ["evidence.scheduled", "evidence.pendingAttempts", "completed"]
#guard declaredKinds nexusCallerCases.scheduleToStartTimeout == ["evidence.scheduled", "timedOut"]
#guard declaredKinds nexusCallerCases.startToCloseTimeout ==
  ["evidence.scheduled", "started", "timedOut"]

/- Every Case is named by the set and the Query. -/
#guard (match nexusCallerCases.asyncCompletion with
  | .ok output => output.case_id
  | .error _ => "") == "temporal.case.nexusCallerTests.asyncCompletion"
#guard nexusCallerCases.asyncCompletion.identity.fixture == "nexusCallerTests-asyncCompletion"

/-- The Known Gaps a Case carries, as (kind, code). -/
private def knownGaps (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) :
    List (temporal.server.api.testpilot.v1.KnownGapKind × String) :=
  match produced with
  | .ok output =>
      (output.provenance.map fun provenance =>
        provenance.known_gaps.toList.map fun gap => (gap.kind, gap.code)).getD []
  | .error _ => []

/- A path with no silent step carries no Known Gap: the machine has no setup parameter left
unbound, and every step it takes records what confirms it. -/
#guard [nexusCallerCases.syncCompletion, nexusCallerCases.asyncCompletion,
    nexusCallerCases.asyncFailure, nexusCallerCases.handlerError,
    nexusCallerCases.startToCloseTimeout].all fun produced => knownGaps produced == []

/- A silent step on the path -- the unobservable backoff, the worker stop that records nothing -- is
a capability Known Gap coded after the step: the Contract infers it from the evidence of the step
after it. -/
#guard ((knownGaps nexusCallerCases.retry).map fun (kind, code) =>
  (kind == temporal.server.api.testpilot.v1.KnownGapKind.KNOWN_GAP_KIND_CAPABILITY,
    code.endsWith ".backoff.unobserved")) == [(true, true)]
#guard ((knownGaps nexusCallerCases.scheduleToStartTimeout).map fun (kind, code) =>
  (kind == temporal.server.api.testpilot.v1.KnownGapKind.KNOWN_GAP_KIND_CAPABILITY,
    code.endsWith ".workerStop.unobserved")) == [(true, true)]

/-- The confirmed steps each projection rule of a Case carries, by kind in the Contract's order: a
silent step is confirmed with the step after it. -/
private def confirmedSteps
    (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) :
    List (String × Nat) :=
  match produced with
  | .ok output =>
      ((output.contract.bind (·.«correlated»)).map fun capability =>
        capability.projection_rules.toList.map fun rule =>
          (rule.kind, rule.outputs.size)).getD []
  | .error _ => []

#guard confirmedSteps nexusCallerCases.retry ==
  [("completed", 2), ("evidence.pendingAttempts", 1), ("evidence.scheduled", 1)]
#guard confirmedSteps nexusCallerCases.scheduleToStartTimeout ==
  [("evidence.scheduled", 1), ("timedOut", 2)]
#guard confirmedSteps nexusCallerCases.asyncCompletion ==
  [("completed", 1), ("evidence.scheduled", 1), ("started", 1)]

/- A derived fixture is registered once: a second `case` over the same set would name the same
fixture and Case ID. -/
/--
error: fixture 'nexusCallerTests-syncCompletion' is already registered by Case 'temporal.case.nexusCallerTests.syncCompletion'
-/
#guard_msgs in
case nexusCallerAgain
  realizes nexusCallerTests
  as (Temporal.Case.Realization.asyncNexus "umpire.case.service" "complete")

end Temporal.Feature.Nexus.Caller.Tests

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

/- Every action class the machine steps on: six replies, three resolutions, the two faults, and the
one timer. -/
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

/- The rows the product machine does not see: every schedule command, every retry and every
backoff. -/
#guard (nexusProtocol.refinement.rows.filter (·.2.isNone)).length == 24 * 8 + 24 + 24 + 24

/-- The phase a refinement row leaves from: the first segment of its key. -/
private def rowPhase (row : String × Option String) : String :=
  ((row.1.splitOn "-").head?).getD ""

/- **Stutter invariance.** A product Property read on the protocol machine is checked on every row,
stutters included, and on a stutter the mapped state before and after are equal. `terminalIsFinal`
triggers only at a terminal prior state, and no stutter has one: every row the product machine does
not see leaves from a phase that reads as `scheduled`. That is the fact the invariance argument
rests on (`UMPIRE4_RESEARCH_NEXUS_MODEL.md` section 2), pinned rather than assumed. -/
#guard (nexusProtocol.refinement.rows.filter (·.2.isNone)).all fun row =>
  ["unscheduled", "scheduled", "backingOff"].contains (rowPhase row)

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

/-! ### The Cases

One realization serves the four Queries, and each Case's Program is the path's: the completion
classes land on the controller between the wait for the authority and the close-event read, and
that wait is emitted only where a completion is on the path. -/

private def instructionIds (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case)
    (entrypointId : String) : List String :=
  match produced with
  | .ok output =>
      ((output.program.bind fun (program : temporal.server.api.testpilot.v1.Program) =>
        program.entrypoints.find? (·.entrypoint_id == entrypointId)).map fun entrypoint =>
          entrypoint.instructions.toList.map (·.instruction_id)).getD []
  | .error _ => []

#guard instructionIds nexusCallerCases.syncCompletion "controller" ==
  ["start-workflow", "await-close", "history"]
#guard instructionIds nexusCallerCases.syncCompletion "handler" == ["respond-sync"]
#guard instructionIds nexusCallerCases.asyncCompletion "controller" ==
  ["start-workflow", "await-completion-authority", "complete-nexus-operation", "await-close",
    "history"]
#guard instructionIds nexusCallerCases.asyncFailure "controller" ==
  ["start-workflow", "await-completion-authority", "fail-nexus-operation", "await-close", "history"]
#guard instructionIds nexusCallerCases.handlerError "handler" == ["respond-error"]

/-- The evidence kinds a Case declares, in the order its witness records them. -/
private def declaredKinds
    (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) : List String :=
  match produced with
  | .ok output =>
      (output.program.map fun (program : temporal.server.api.testpilot.v1.Program) =>
        program.evidence.toList.map (·.evidence_id)).getD []
  | .error _ => []

/- The evidence each Case lifts is read off the machine's `evidence:` lines along its witness: one
declaration per recorded kind, the scheduled event first. -/
#guard declaredKinds nexusCallerCases.syncCompletion == ["scheduled", "completed"]
#guard declaredKinds nexusCallerCases.asyncCompletion == ["scheduled", "started", "completed"]
#guard declaredKinds nexusCallerCases.asyncFailure == ["scheduled", "started", "failed"]
#guard declaredKinds nexusCallerCases.handlerError == ["scheduled", "failed"]

/- Every Case is named by the set and the Query, and none carries a Known Gap: no path uses an
unobservable timer, and the machine has no setup parameter left unbound. -/
#guard (match nexusCallerCases.asyncCompletion with
  | .ok output => output.case_id
  | .error _ => "") == "temporal.case.nexusCallerTests.asyncCompletion"
#guard nexusCallerCases.asyncCompletion.identity.fixture == "nexusCallerTests-asyncCompletion"
#guard [nexusCallerCases.syncCompletion, nexusCallerCases.asyncCompletion,
    nexusCallerCases.asyncFailure, nexusCallerCases.handlerError].all fun produced =>
  match produced with
  | .ok output => (output.provenance.map (·.known_gaps.isEmpty)) == some true
  | .error _ => false

end Temporal.Feature.Nexus.Caller.Tests

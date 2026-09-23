import Temporal.Feature.Nexus.Control.Model

/-!
# What the control says

The control's machine keeps the platform's rows and adds the forged one; its Case declares the
evidence of both results of the witnessed row, projected each to its own row, so a Run that takes
the real row is read as a violation rather than as nothing. These pins are the check the replay's
early proof point requires before any live Run.
-/

namespace Temporal.Feature.Nexus.Control.Tests

open Umpire
open Umpire.Command
open Temporal.Feature.Nexus.Caller
open Temporal.Feature.Nexus.Control
open temporal.server.api.testpilot.v1

/-! ### The machine -/

/- The non-retryable error from `scheduled` has two rows: the platform's, failing the operation
with the failed event, and the forged one, completing it with the completed event. -/
#guard (controlHandlerReplyStep { phase := .scheduled } (.handlerError (retryable := false))).map
    (fun step => (step.state.phase, step.facts)) ==
  [(.failed, [.nexusOperationFailed]), (.succeeded, [.nexusOperationCompleted])]

/- Every other reply has the platform's one row, so the machine is the pair Model's plus the
forged row. -/
#guard (controlHandlerReplyStep { phase := .scheduled } .syncSuccess).length == 1
#guard (controlHandlerReplyStep { phase := .scheduled } (.handlerError (retryable := true))).length == 1
/- Every class the machine steps on: the eight schedules, six replies and three resolutions. -/
#guard nexusControl.actionKeys.size == 17

/-! ### The Case -/

#guard nexusCallerControlCases.forgedCompletion.identity.caseId ==
  "temporal.case.nexusCallerControl.forgedCompletion"
#guard nexusCallerControlCases.forgedCompletion.identity.fixture == "nexusCallerControl-forgedCompletion"

private def produced : Option Case := nexusCallerControlCases.forgedCompletion.toOption

#guard produced.isSome

/-- The evidence kinds the Program declares, by their Case-local ids: the scheduled read, the
witness's completed event and the platform's failed event. -/
private def declaredEvidence : List String :=
  ((produced.bind fun output => output.program).map fun program =>
    program.evidence.toList.map (·.evidence_id)).getD []

/- The witness's kind and the platform's real kind are both declared: the Program lifts the failed
event the platform writes, so the Contract reads it rather than nothing. -/
#guard declaredEvidence == ["evidence.scheduled", "completed", "evidence.failed"]

/-- The projection rule of one kind: the rows it confirms, as (action class, resulting state). -/
private def projected (kind : String) : Option (List (String × String)) := do
  let output ← produced
  let contract ← output.contract
  let correlated ← contract.correlated
  let rule ← correlated.projection_rules.toList.find? (·.kind == kind)
  pure (rule.outputs.toList.map fun row =>
    ((row.action.map (·.value)).getD "", (row.state.map (·.value)).getD ""))

/- The completed kind confirms the forged row and the failed kind the real one: each result of the
witnessed row is projected to its own row. -/
#guard projected "completed" == some [("handlerReply-handlerError-false", "succeeded")]
#guard projected "evidence.failed" == some [("handlerReply-handlerError-false", "failed")]
#guard projected "evidence.scheduled" == some [("schedule-unset-unset-unset", "scheduled")]

/- The Case carries no Known Gap: both steps of the path record what confirms them. -/
#guard (produced.bind fun output => output.provenance.map fun provenance =>
  provenance.known_gaps.size) == some 0

end Temporal.Feature.Nexus.Control.Tests

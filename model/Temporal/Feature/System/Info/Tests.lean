import Temporal.Feature.System.Info.Model

/-!
# What the system-info Model says

The one-step machine, its one Query, and the produced Case: one controller instruction invoking
`GetSystemInfo` and reading the server version into an observation, no workflow, a Contract that
is the correlated capability alone, and the instruction's completion as the one evidence kind.
-/

namespace Temporal.Feature.System.Info.Tests

open Umpire
open Umpire.Command
open Temporal.Feature.System.Info
open temporal.server.api.testpilot.v1 hiding ModelValue SourceLocation

/-! ### The machine -/

#guard systemInfo.table.states.length == 2
#guard systemInfo.actionKeys == #["getSystemInfo"]
#guard systemInfo.stuck == none

/-- info: 'Temporal.Feature.System.Info.systemInfo' depends on axioms: [propext] -/
#guard_msgs in
#print axioms systemInfo

/-! ### The Query and the Case -/

#guard (match answered with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"

#guard systemInfoCases.answered.identity.fixture == "systemInfoTests-answered"
#guard systemInfoCases.answered.identity.caseId == "temporal.case.systemInfoTests.answered"

private def produced : Option Case := systemInfoCases.answered.toOption

#guard produced.isSome

/- One controller, one instruction, no workflow. -/
#guard ((produced.bind (·.program)).map fun program =>
  program.entrypoints.toList.map fun entrypoint =>
    (entrypoint.entrypoint_id, entrypoint.instructions.toList.map (·.instruction_id))) ==
  some [("controller", ["get-system-info"])]

/- The instruction invokes `GetSystemInfo` on the workflow-service role with no request assignment
and reads `server_version` into the observation. -/
#guard ((produced.bind (·.program)).bind fun program =>
  (program.entrypoints[0]?.bind (·.instructions[0]?)).bind fun node =>
    node.instruction.bind (·.instruction) |>.map fun instruction =>
      match instruction with
      | .invoke_rpc request =>
          (request.endpoint_role_id, request.method, request.request_assignments.size,
            request.response_reads.toList.map (·.path))
      | _ => ("", "", 0, [])) ==
  some (Temporal.Case.Support.workflowServiceRole, Temporal.Case.Support.getSystemInfoMethod, 0,
    ["server_version"])

/- The Contract is the correlated capability alone; the call's completion confirms the one step. -/
#guard ((produced.bind (·.contract)).map fun contract => contract.rules.size) == some 0
#guard ((produced.bind fun output => output.contract.bind (·.«correlated»)).map fun capability =>
  capability.projection_rules.toList.map fun rule => (rule.kind, rule.outputs.size)) ==
  some [("instructionCompleted", 1)]

/- The evidence is the instruction-completed Run Event, keyed by its protocol code. -/
#guard ((produced.bind (·.program)).map fun program =>
  program.evidence.toList.map fun declaration =>
    (declaration.evidence_id, declaration.operation,
      match declaration.source with
      | some (.run_event source) => some source.kind
      | _ => none)) ==
  some [("instructionCompleted", "protocol_code", some .RUN_EVENT_KIND_INSTRUCTION_COMPLETED)]

/- No step is silent, so the Case carries no Known Gap. -/
#guard ((produced.bind (·.provenance)).map fun provenance => provenance.known_gaps.size) == some 0

end Temporal.Feature.System.Info.Tests

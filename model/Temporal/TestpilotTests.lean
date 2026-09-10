import Temporal.Testpilot
import Temporal.Feature.Nexus3.Testpilot
import Umpire.Variations.Tests.Lowering

namespace Temporal.TestpilotTests

open temporal.server.api.testpilot.v1

private def activationKind : EntrypointDefinition → Option Nat
  | { activation := some (.controller _), .. } => some 0
  | { activation := some (.workflow _), .. } => some 1
  | { activation := some (.nexus_handler _), .. } => some 2
  | _ => none

#guard match Temporal.Testpilot.getSystemInfoCase with
  | .ok output =>
      match output.program, output.contract with
      | some program, some contract =>
        match program.entrypoints.toList, contract.rules.toList with
        | [entrypoint], [rule] =>
          match entrypoint.instructions.toList with
          | [node] =>
            match node.instruction.bind (·.instruction) with
            | some (.invoke_rpc request) =>
                request.endpoint_role_id == Temporal.Testpilot.workflowServiceRole &&
                request.method == Temporal.Testpilot.getSystemInfoMethod &&
                request.request_assignments.isEmpty &&
                request.response_projections.map (·.kind) == #[.PROJECTION_KIND_ONE] &&
                rule.rule_id == "server-version-present"
            | _ => false
          | _ => false
        | _, _ => false
      | _, _ => false
  | .error _ => false

-- The async-nexus Case carries no monitor rule: everything its Contract says is the scoped
-- capability the checked model lowered into, reading the evidence this Program's history read
-- lifts.
#guard match Temporal.Feature.Nexus3.Testpilot.completionCase with
  | .ok output =>
      output.program.map (fun program => program.entrypoints.map activationKind) ==
        some #[some 0, some 1, some 2] &&
      output.contract.map (·.rules.isEmpty) == some true &&
      (match output.contract.bind (·.«scoped») with
        | some capability =>
            capability.evidence_observation_id == Temporal.Feature.Nexus3.Testpilot.scopedObservation &&
            capability.projection_id == Temporal.Feature.Nexus3.Testpilot.projectionId.value &&
            capability.clauses.size == 3 &&
            -- The Behavior places the required Action one semantic transition after the
            -- operation's opening one, so that is the window each clause carries.
            capability.clauses.all (fun clause =>
              clause.clock == .SCOPED_CLOCK_OPERATION_TRANSITIONS && clause.bound == 1 &&
              clause.endpoint == .SCOPED_ENDPOINT_RUNTIME_PREFIX) &&
            capability.projection_rules.map (·.kind) == #[
              Temporal.Feature.Nexus3.Testpilot.completedEvidenceKindId.value,
              Temporal.Feature.Nexus3.Testpilot.startedEvidenceKindId.value]
        | none => false)
  | .error _ => false

-- The worker-outage Case's stop instruction is the lowered fault intent itself, not a hand-written
-- copy of one: the Program the fixture renders carries exactly what `lower` produced.
#guard match Temporal.Testpilot.stopIntent.lower Temporal.Testpilot.stopRealization,
    Temporal.Testpilot.workerOutageCase with
  | .ok stop, .ok artifact =>
      (artifact.program.bind fun program =>
        (program.entrypoints.find? fun entrypoint => entrypoint.entrypoint_id == "controller").bind
          fun controller => controller.instructions[0]?).any
        (Umpire.VariationsLoweringTests.sameFaultNode · stop)
  | _, _ => false

-- The outage rule is the checked-in `rule_events` horizon: no elapsed-time bound participates.
#guard match Temporal.Testpilot.workerOutageCase with
  | .ok artifact =>
      (artifact.contract.bind fun contract =>
        (contract.rules.find? fun rule => rule.rule_id == "worker-outage-order").bind
          (·.horizon)).any fun horizon =>
            horizon.rule_events == Temporal.Testpilot.workerOutageHorizon &&
              horizon.elapsed_milliseconds == 0 && horizon.violation_state_id == "expired"
  | .error _ => false

end Temporal.TestpilotTests

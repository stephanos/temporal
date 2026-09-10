import Umpire.Space.Lowering

/-! Exact fault-intent lowering. The identity between what this produces and the instruction the
shipped worker-outage Case carries is pinned in `Temporal.TestpilotTests`, which is the side of the
boundary allowed to name both. -/

namespace Umpire.SpaceLoweringTests

open Umpire
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def source : SourceLocation := { path := "Umpire/Space/Tests/Lowering.lean" }

private def intent (capability : DefinitionId) : FaultIntentDeclaration :=
  FaultIntentDeclaration.atOccurrence (.of "test.fault") source
    (.of "test.occurrence") (.of "test.action") capability

private def limits : InstructionLimits := Program.instructionLimits 1000 1 1 1024

private def realization : FaultRealization :=
  { instructionId := "stop", roleId := "queue", limits }

/-- Everything a lowered fault instruction carries, compared field for field: the generated
protocol types derive no equality, so the comparison is written out rather than assumed. -/
def sameFaultNode (left right : InstructionDefinition) : Bool :=
  let faultOf := fun (node : InstructionDefinition) =>
    node.instruction.bind fun instruction => match instruction.instruction with
      | some (.inject_fault fault) => some (fault.role_id, fault.kind)
      | _ => none
  let bounds := fun (node : InstructionDefinition) =>
    node.limits.map fun value => (value.timeout_milliseconds, value.max_attempts,
      value.max_emitted_events, value.max_response_bytes)
  let references := fun (node : InstructionDefinition) =>
    node.dependencies.map fun reference => (reference.entrypoint_id, reference.instruction_id)
  left.instruction_id == right.instruction_id && faultOf left == faultOf right &&
    bounds left == bounds right && references left == references right &&
    left.guard.isSome == right.guard.isSome && left.outcome.isSome == right.outcome.isSome

private def lowered (capability : DefinitionId) : Option InstructionDefinition :=
  ((intent capability).lower realization).toOption

-- The declaration decides the outage; the realization decides only where it lands.
#guard (lowered workerStopCapabilityId).any fun node =>
  sameFaultNode node
    (Program.node "stop" (Program.injectFault "queue" .FAULT_KIND_WORKER_STOP) limits)

#guard (lowered workerResumeCapabilityId).any fun node =>
  sameFaultNode node
    (Program.node "stop" (Program.injectFault "queue" .FAULT_KIND_WORKER_RESUME) limits)

-- A capability outside the version-one vocabulary has no outage to realize, and it rejects by name
-- rather than defaulting to one.
#guard match (intent (.of "test.capability.unknown")).lower realization with
  | .error failure =>
      failure.sourceDefinitionId == "test.fault" &&
        failure.construct == "fault.unknown-capability/test.capability.unknown"
  | .ok _ => false

#guard [{ realization with instructionId := "" }, { realization with roleId := "" }].all
  fun incomplete => match (intent workerStopCapabilityId).lower incomplete with
    | .error failure => failure.construct == "fault.incomplete-realization"
    | .ok _ => false

end Umpire.SpaceLoweringTests

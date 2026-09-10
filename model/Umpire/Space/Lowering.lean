import Umpire.Space.Language
import Umpire.Case.Compiler

/-!
Lowering one checked fault intent to the generated instruction a Driver realizes.

A `FaultIntentDeclaration` says *what* outage is requested and *where in the model* it is requested:
the occurrence it attaches to, the Action that occurrence selects, and the capability it targets. It
says nothing about a Program, because the Space language has no Program: it has no instruction
identity, no role, no bounds and no outcome schema. Those are realization, and this module takes
them as a separate argument rather than inventing them.

What the declaration does decide is the outage itself. The capability a fault intent targets is the
one the Driver realizes, so the version-one vocabulary below maps that capability onto a `FaultKind`
and a capability outside it rejects by name. That keeps the requested outage a property of the
declaration rather than a value the caller could pick freely beside it.

`Umpire.Exploration.Coverage` keeps its wording: a requested fault is intent until the Run carries a
`FAULT_INJECTED` event for it. Lowering produces the instruction that can realize one; it does not
claim the realization.
-/

namespace Umpire

open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-- The capability a worker-stop intent targets. -/
def workerStopCapabilityId : DefinitionId := .of "temporal.testpilot.fault.worker-stop"

/-- The capability a worker-resume intent targets. -/
def workerResumeCapabilityId : DefinitionId := .of "temporal.testpilot.fault.worker-resume"

/-- The version-one fault vocabulary. Every value is a worker-lifecycle transition on one activation
queue, which is exactly what the closed `FaultKind` enum admits. -/
def faultKindOf (capability : DefinitionId) : Option FaultKind :=
  if capability == workerStopCapabilityId then some .FAULT_KIND_WORKER_STOP
  else if capability == workerResumeCapabilityId then some .FAULT_KIND_WORKER_RESUME
  else none

/-- Where a lowered fault intent lands in a Program. The declaration cannot carry any of this: the
Space language has no Program to name an instruction, a role or a bound in. -/
structure FaultRealization where
  instructionId : String
  /-- The `ROLE_KIND_TASK_QUEUE` role whose resource binding identifies the affected queue. -/
  roleId : String
  limits : InstructionLimits
  dependencies : Array InstructionRef := #[]
  guard : Option ProgramExpression := none
  outcome : Option InstructionOutcomeDefinition := none

/-- Lower one fault intent to the instruction definition a Driver realizes it through. The outage
comes from the declaration's capability; the placement comes from the realization. A capability
outside the version-one vocabulary, or a placement missing an instruction identity or a role,
rejects rather than producing an instruction no Driver could dispatch. -/
def FaultIntentDeclaration.lower
    (declaration : FaultIntentDeclaration)
    (realization : FaultRealization) :
    Except Umpire.Case.Compiler.LoweringError InstructionDefinition :=
  let failed := fun construct =>
    Umpire.Case.Compiler.LoweringError.mk declaration.id.value declaration.source construct
  match faultKindOf declaration.capability with
  | none => .error (failed ("fault.unknown-capability/" ++ declaration.capability.value))
  | some kind =>
      if realization.instructionId.isEmpty || realization.roleId.isEmpty then
        .error (failed "fault.incomplete-realization")
      else
        .ok (Program.node realization.instructionId (Program.injectFault realization.roleId kind)
          realization.limits realization.dependencies realization.guard realization.outcome)

end Umpire

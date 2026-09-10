import Umpire.Space.Language
import Umpire.Case.Compiler

/-!
Lowering one checked fault intent to the generated instruction a Driver realizes.

What a `FaultIntentDeclaration` decides here is the outage itself: the capability a fault intent
targets is the one the Driver realizes, so the version-one vocabulary below maps that capability onto
a `FaultKind`, and a capability outside it rejects by name. That keeps the requested outage a
property of the declaration rather than a value the caller could pick freely beside it.

Everything else is realization. The declaration's occurrence and Action name a model coordinate, and
nothing in the Space language turns a model coordinate into an instruction identity, a role, a bound
or a dependency edge -- there is no Program there to name one in. So placement arrives as a separate
argument, and this lowering does not read the occurrence. Deriving the dependency edge from the
occurrence would need an occurrence-to-instruction map the caller owns; no caller has one yet.

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

/-- Where a lowered fault intent lands in a Program. The declaration carries none of it: the Space
language has no Program to name an instruction, a role or a bound in. -/
structure FaultRealization where
  instructionId : String
  /-- The `ROLE_KIND_TASK_QUEUE` role whose resource binding identifies the affected queue. -/
  roleId : String
  limits : InstructionLimits
  dependencies : Array InstructionRef := #[]
  guard : Option ProgramExpression := none
  outcome : Option InstructionOutcomeDefinition := none

/-- Lower one fault intent to the instruction definition a Driver realizes it through. The outage
comes from the declaration's capability; the placement comes entirely from the realization. A
capability outside the version-one vocabulary, or a placement missing an instruction identity or a
role, rejects rather than producing an instruction no Driver could dispatch. -/
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

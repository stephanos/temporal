import Temporal.Shared

/-!
# Nexus lifecycle semantics

Start here with the ordinary Nexus states, events, and complete transition relation. Read
`Temporal.Feature.Nexus.Lifecycle.Target` next to see this meaning encoded as a checked Umpire
target.
-/

namespace Temporal.Feature.Nexus.Lifecycle

open Umpire

private def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Lifecycle.lean"

def targetId : DefinitionId := id "temporal.nexus.basic-lifecycle.target"
def kernelId : DefinitionId := id "temporal.nexus.basic-lifecycle.kernel"
def lifecycleCapabilityId : DefinitionId := id "temporal.nexus.basic-lifecycle.capability"
def lifecycleProviderId : DefinitionId := id "temporal.nexus.basic-lifecycle.provider"
def lifecycleLawId : DefinitionId := id "temporal.nexus.basic-lifecycle.law.authoritative-step"
def operationStateId : DefinitionId := id "temporal.nexus.basic-lifecycle.state.operation"
def startActionId : DefinitionId := id "temporal.nexus.basic-lifecycle.action.start"
def cancelActionId : DefinitionId := id "temporal.nexus.basic-lifecycle.action.cancel"
def reportSuccessActionId : DefinitionId := id "temporal.nexus.basic-lifecycle.action.succeed"
def transitionOutcomeId : DefinitionId := id "temporal.nexus.basic-lifecycle.outcome.transition"
def lifecycleObservationId : DefinitionId := id "temporal.nexus.basic-lifecycle.observation.state"
def operationRoleId : DefinitionId := id "temporal.nexus.basic-lifecycle.role.operation"

/-- The four states exposed by the ordinary Nexus operation lifecycle. -/
inductive OperationState where
  | scheduled
  | started
  | canceled
  | succeeded
  deriving DecidableEq, Repr

/-- The three events exposed by the ordinary Nexus operation lifecycle. -/
inductive OperationEvent where
  | start
  | cancel
  | succeed
  deriving DecidableEq, Repr

/-- The complete focused Nexus transition relation. -/
def step : OperationState → OperationEvent → Option OperationState
  | .scheduled, .start => some .started
  | .started, .cancel => some .canceled
  | .started, .succeed => some .succeeded
  | _, _ => none

/-- The provider law ties the teaching surface to the authoritative Nexus lifecycle. -/
def LawStatement (law : LawDefinition) : Prop :=
  law.id = lifecycleLawId ∧
    law.body = "temporal-nexus-basic-lifecycle-authoritative-step/v2" ∧
    step .scheduled .start = some .started ∧
    step .started .cancel = some .canceled ∧
    step .started .succeed = some .succeeded

def lifecycleLaw : LawDefinition := {
  id := lifecycleLawId
  body := "temporal-nexus-basic-lifecycle-authoritative-step/v2"
}

theorem lifecycleLawProof : LawStatement lifecycleLaw := by
  exact ⟨rfl, rfl, rfl, rfl, rfl⟩

end Temporal.Feature.Nexus.Lifecycle

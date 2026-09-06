import Temporal.Feature.Nexus.Lifecycle.Semantics

namespace Temporal.Feature.Nexus.LifecycleTests

open Temporal.Feature.Nexus.Lifecycle

theorem lifecycleDefinitionIdsRetainTheirEstablishedValues :
    targetId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.target" ∧
    kernelId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.kernel" ∧
    lifecycleCapabilityId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.capability" ∧
    lifecycleProviderId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.provider" ∧
    lifecycleLawId =
      Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.law.authoritative-step" ∧
    operationStateId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.state.operation" ∧
    startActionId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.action.start" ∧
    cancelActionId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.action.cancel" ∧
    reportSuccessActionId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.action.succeed" ∧
    transitionOutcomeId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.outcome.transition" ∧
    lifecycleObservationId =
      Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.observation.state" ∧
    operationRoleId = Umpire.DefinitionId.of "temporal.nexus.basic-lifecycle.role.operation" := by
  exact ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl⟩

theorem supportedTransitionsReachTheirNextStates : step .scheduled .start = some .started ∧
    step .started .cancel = some .canceled ∧
    step .started .succeed = some .succeeded := by
  exact ⟨rfl, rfl, rfl⟩

theorem scheduledAndStartedRejectUnsupportedEvents : step .scheduled .cancel = none ∧
    step .scheduled .succeed = none ∧
    step .started .start = none := by
  exact ⟨rfl, rfl, rfl⟩

theorem terminalStatesRejectEveryEvent : step .canceled .start = none ∧
    step .canceled .cancel = none ∧
    step .canceled .succeed = none ∧
    step .succeeded .start = none ∧
    step .succeeded .cancel = none ∧
    step .succeeded .succeed = none := by
  exact ⟨rfl, rfl, rfl, rfl, rfl, rfl⟩

end Temporal.Feature.Nexus.LifecycleTests

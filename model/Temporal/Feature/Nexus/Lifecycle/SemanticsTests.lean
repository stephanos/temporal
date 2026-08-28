import Temporal.Feature.Nexus.Lifecycle.Semantics

namespace Temporal.Feature.Nexus.LifecycleTests

open Temporal.Feature.Nexus.Lifecycle

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

import Temporal.System.Nexus.Tests

namespace Temporal.System.Nexus.ImplementationLinkTests

open Umpire

/-!
This focused System test root stays below Feature. The checked correspondence and its forward
witness are exercised from `Temporal.ImplementationLinkTests.Nexus`, the exact composed-test root.
-/

example : Temporal.System.Nexus.target.kernel.steps
    Temporal.System.Nexus.queuedState
    Temporal.System.Nexus.dispatchAction =
    [Temporal.System.Nexus.dispatchedResult] := by
  native_decide

theorem named_system_authority_remains_available_for_correspondence :
    Temporal.System.Nexus.target.kernel.authoritativeInitial
      Temporal.System.Nexus.queuedSetup
      Temporal.System.Nexus.queuedState ∧
    Temporal.System.Nexus.target.kernel.authoritativeStep
      Temporal.System.Nexus.queuedState
      Temporal.System.Nexus.dispatchAction
      Temporal.System.Nexus.dispatchedResult ∧
    Temporal.System.Nexus.target.kernel.authoritativeStep
      Temporal.System.Nexus.runningState
      Temporal.System.Nexus.recordCancellationAction
      Temporal.System.Nexus.cancellationRecordedResult ∧
    Temporal.System.Nexus.target.kernel.authoritativeStep
      Temporal.System.Nexus.runningState
      Temporal.System.Nexus.recordCompletionAction
      Temporal.System.Nexus.completionRecordedResult := by
  exact ⟨Temporal.System.Nexus.target_queued_initial_authoritative,
    Temporal.System.Nexus.target_queued_dispatch_authoritative,
    Temporal.System.Nexus.target_running_cancellation_authoritative,
    Temporal.System.Nexus.target_running_completion_authoritative⟩

theorem callerClosureKernelDomainsRetainEqualityProofShapes
    (setup : List RoleBinding) (state action outcome observation : ModelValue)
    (setupAdmitted : Temporal.System.Nexus.CallerClosure.transitionKernel.setupDomain setup)
    (stateAdmitted : Temporal.System.Nexus.CallerClosure.transitionKernel.stateDomain state)
    (actionAdmitted : Temporal.System.Nexus.CallerClosure.transitionKernel.actionDomain action)
    (outcomeAdmitted : Temporal.System.Nexus.CallerClosure.transitionKernel.outcomeDomain outcome)
    (observationAdmitted :
      Temporal.System.Nexus.CallerClosure.transitionKernel.observationDomain observation) :
    setup = Temporal.System.Nexus.CallerClosure.setup ∧
    (state = Temporal.System.Nexus.CallerClosure.openState ∨
      state = Temporal.System.Nexus.CallerClosure.closedState) ∧
    action = Temporal.System.Nexus.CallerClosure.forceCloseAction ∧
    outcome = Temporal.System.Nexus.CallerClosure.cancellationUpgradedOutcome ∧
    (observation = Temporal.System.Nexus.CallerClosure.deliveryObservation ∨
      observation = Temporal.System.Nexus.CallerClosure.cancellationCountObservation ∨
      observation = Temporal.System.Nexus.CallerClosure.ownershipObservation) := by
  change setup = Temporal.System.Nexus.CallerClosure.setup at setupAdmitted
  change state = Temporal.System.Nexus.CallerClosure.openState ∨
    state = Temporal.System.Nexus.CallerClosure.closedState at stateAdmitted
  change action = Temporal.System.Nexus.CallerClosure.forceCloseAction at actionAdmitted
  change outcome = Temporal.System.Nexus.CallerClosure.cancellationUpgradedOutcome at outcomeAdmitted
  change observation = Temporal.System.Nexus.CallerClosure.deliveryObservation ∨
    observation = Temporal.System.Nexus.CallerClosure.cancellationCountObservation ∨
    observation = Temporal.System.Nexus.CallerClosure.ownershipObservation at observationAdmitted
  exact ⟨setupAdmitted, stateAdmitted, actionAdmitted, outcomeAdmitted, observationAdmitted⟩

theorem callerClosureAuthoritiesRetainConjunctionProofShapes
    (setup : List RoleBinding) (initialState state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (initialAdmitted :
      Temporal.System.Nexus.CallerClosure.transitionKernel.authoritativeInitial
        setup initialState)
    (stepAdmitted : Temporal.System.Nexus.CallerClosure.transitionKernel.authoritativeStep
      state action result) :
    setup = Temporal.System.Nexus.CallerClosure.setup ∧
    initialState = Temporal.System.Nexus.CallerClosure.openState ∧
    state = Temporal.System.Nexus.CallerClosure.openState ∧
    action = Temporal.System.Nexus.CallerClosure.forceCloseAction ∧
    result = Temporal.System.Nexus.CallerClosure.closeResult := by
  change Temporal.System.Nexus.CallerClosure.authoritativeInitial
    setup initialState at initialAdmitted
  rcases initialAdmitted with ⟨rfl, rfl⟩
  change Temporal.System.Nexus.CallerClosure.authoritativeStep
    state action result at stepAdmitted
  rcases stepAdmitted with ⟨rfl, rfl, rfl⟩
  exact ⟨rfl, rfl, rfl, rfl, rfl⟩

end Temporal.System.Nexus.ImplementationLinkTests

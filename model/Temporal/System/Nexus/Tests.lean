import Temporal.System.Nexus.Core
import Temporal.System.Nexus.CallerClosure

namespace Temporal.System.Nexus.Tests

open Umpire
open Temporal.System.Nexus

#check (Temporal.System.Nexus.finiteMachine : FiniteMachine
  ExecutionSetup ModelValue ModelValue ModelValue ModelValue)
#check (Temporal.System.Nexus.authoritativeInitial : ExecutionSetup → ModelValue → Prop)
#check (Temporal.System.Nexus.authoritativeStep : ModelValue → ModelValue →
  TransitionResult ModelValue ModelValue ModelValue → Prop)
#check (Temporal.System.Nexus.target : CheckedTarget LawStatement
  ExecutionSetup ModelValue ModelValue ModelValue ModelValue)

example : step .queued .dispatch = some .running ∧
    step .running .recordCancellation = some .cancellationRecorded ∧
    step .running .recordCompletion = some .completionRecorded := by
  exact ⟨rfl, rfl, rfl⟩

example : step .queued .recordCancellation = none ∧
    step .queued .recordCompletion = none ∧
    step .running .dispatch = none ∧
    step .cancellationRecorded .dispatch = none ∧
    step .cancellationRecorded .recordCancellation = none ∧
    step .cancellationRecorded .recordCompletion = none ∧
    step .completionRecorded .dispatch = none ∧
    step .completionRecorded .recordCancellation = none ∧
    step .completionRecorded .recordCompletion = none := by
  exact ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl⟩

example : (checkTarget targetAuthoring).isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map CapabilityProvider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
  native_decide

example : target.kernel.initialStates queuedSetup = [queuedState] ∧
    target.kernel.initialStates runningSetup = [runningState] ∧
    target.kernel.steps queuedState dispatchAction = [dispatchedResult] ∧
    target.kernel.steps runningState recordCancellationAction = [cancellationRecordedResult] ∧
    target.kernel.steps runningState recordCompletionAction = [completionRecordedResult] ∧
    target.kernel.steps queuedState recordCancellationAction = [] ∧
    target.kernel.steps cancellationRecordedState dispatchAction = [] := by
  native_decide

example : (checkTarget targetAuthoring).toOption.map (fun checked =>
    (checked.id, checked.source, canonicalCheckedTargetJson checked, checked.behaviorFingerprint)) =
    some (targetId, source, canonicalCheckedTargetJson target, target.behaviorFingerprint) := by
  native_decide

example : transitionKernel.metadata = finiteMachine.kernel.metadata ∧
    transitionKernel.initialStates = finiteMachine.kernel.initialStates ∧
    transitionKernel.steps = finiteMachine.kernel.steps ∧
    finitePlanning = finiteMachine.planning ∧
    targetDefinition.kernel = .checked transitionKernel ∧
    finitePlanning.actions = actions ∧
    actions = [dispatchAction, recordCancellationAction, recordCompletionAction] := by
  exact ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl⟩

example : transitionKernel.behaviorDescription? =
    finiteMachine.kernel.behaviorDescription? := by
  native_decide

example : target.behaviorFingerprint.render =
    "sha256:54d0b7ed28698c0db28e7c9de00f3c0c0998db889a50de917d100173b37cf374" := by
  native_decide

example : target.kernel.authoritativeInitial queuedSetup queuedState ∧
    target.kernel.authoritativeStep queuedState dispatchAction dispatchedResult ∧
    target.kernel.authoritativeStep runningState recordCancellationAction
      cancellationRecordedResult ∧
    target.kernel.authoritativeStep runningState recordCompletionAction
      completionRecordedResult := by
  exact ⟨target_queued_initial_authoritative, target_queued_dispatch_authoritative,
    target_running_cancellation_authoritative, target_running_completion_authoritative⟩

#check (Temporal.System.Nexus.CallerClosure.finiteMachine : FiniteMachine
  (List RoleBinding) ModelValue ModelValue ModelValue ModelValue)

theorem callerClosureFiniteMachineRetainsOrderedSemantics :
    Temporal.System.Nexus.CallerClosure.finiteMachine.setups =
      [Temporal.System.Nexus.CallerClosure.setup] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.states =
      [Temporal.System.Nexus.CallerClosure.openState,
        Temporal.System.Nexus.CallerClosure.closedState] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.actions =
      [Temporal.System.Nexus.CallerClosure.forceCloseAction] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.outcomes =
      [Temporal.System.Nexus.CallerClosure.cancellationUpgradedOutcome] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.observations =
      [Temporal.System.Nexus.CallerClosure.deliveryObservation,
        Temporal.System.Nexus.CallerClosure.cancellationCountObservation,
        Temporal.System.Nexus.CallerClosure.ownershipObservation] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.initialStates
      Temporal.System.Nexus.CallerClosure.setup =
        [Temporal.System.Nexus.CallerClosure.openState] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.steps
      Temporal.System.Nexus.CallerClosure.openState
      Temporal.System.Nexus.CallerClosure.forceCloseAction =
        [Temporal.System.Nexus.CallerClosure.closeResult] := by
  native_decide

theorem callerClosureFiniteMachineRetainsEncoders :
    Temporal.System.Nexus.CallerClosure.finiteMachine.encodeSetup =
      (fun (_ : List RoleBinding) => "caller-closure") ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.encodeState = reprStr ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.encodeAction = reprStr ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.encodeOutcome = reprStr ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.encodeObservation = reprStr := by
  exact ⟨rfl, rfl, rfl, rfl, rfl⟩

theorem callerClosureFiniteMachineRejectsInvalidInputs :
    Temporal.System.Nexus.CallerClosure.finiteMachine.initialStates [] = [] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.steps
      Temporal.System.Nexus.CallerClosure.closedState
      Temporal.System.Nexus.CallerClosure.forceCloseAction = [] ∧
    Temporal.System.Nexus.CallerClosure.finiteMachine.steps
      Temporal.System.Nexus.CallerClosure.openState
      Temporal.System.Nexus.CallerClosure.openState = [] := by
  native_decide

theorem callerClosureTargetMachineryUsesFiniteMachineCapabilities :
    Temporal.System.Nexus.CallerClosure.transitionKernel.metadata =
      Temporal.System.Nexus.CallerClosure.finiteMachine.kernel.metadata ∧
    Temporal.System.Nexus.CallerClosure.transitionKernel.initialStates =
      Temporal.System.Nexus.CallerClosure.finiteMachine.kernel.initialStates ∧
    Temporal.System.Nexus.CallerClosure.transitionKernel.steps =
      Temporal.System.Nexus.CallerClosure.finiteMachine.kernel.steps ∧
    Temporal.System.Nexus.CallerClosure.finitePlanning.actions =
      Temporal.System.Nexus.CallerClosure.finiteMachine.planning.actions := by
  exact ⟨rfl, rfl, rfl, rfl⟩

end Temporal.System.Nexus.Tests

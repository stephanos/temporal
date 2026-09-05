import Temporal.System.Nexus.Core

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

end Temporal.System.Nexus.Tests

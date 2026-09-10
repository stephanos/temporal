import Temporal.System.Nexus.Core

namespace Temporal.System.Nexus.Tests

open Umpire
open Temporal.System.Nexus

#check (Temporal.System.Nexus.finiteMachine : FiniteMachine
  ExecutionSetup ModelValue ModelValue ModelValue ModelValue)
#check (Temporal.System.Nexus.authoritativeInitial : ExecutionSetup → ModelValue → Prop)
#check (Temporal.System.Nexus.authoritativeStep : ModelValue → ModelValue →
  Step ModelValue ModelValue ModelValue → Prop)
#check (Temporal.System.Nexus.target : CheckedModel LawStatement
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

example : (checkModel targetAuthoring).isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map Provider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
  native_decide

example : target.machine.initialStates queuedSetup = [queuedState] ∧
    target.machine.initialStates runningSetup = [runningState] ∧
    target.machine.steps queuedState dispatchAction = [dispatchedResult] ∧
    target.machine.steps runningState recordCancellationAction = [cancellationRecordedResult] ∧
    target.machine.steps runningState recordCompletionAction = [completionRecordedResult] ∧
    target.machine.steps queuedState recordCancellationAction = [] ∧
    target.machine.steps cancellationRecordedState dispatchAction = [] := by
  native_decide

example : (checkModel targetAuthoring).toOption.map (fun checked =>
    (checked.id, checked.source, canonicalCheckedModelJson checked, checked.behaviorFingerprint)) =
    some (targetId, source, canonicalCheckedModelJson target, target.behaviorFingerprint) := by
  native_decide

example : machine.metadata = finiteMachine.kernel.metadata ∧
    machine.initialStates = finiteMachine.kernel.initialStates ∧
    machine.steps = finiteMachine.kernel.steps ∧
    finitePlanning = finiteMachine.planning ∧
    modelSpec.machine = .checked machine ∧
    finitePlanning.actions = actions ∧
    actions = [dispatchAction, recordCancellationAction, recordCompletionAction] := by
  exact ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl⟩

example : machine.behaviorTable? =
    finiteMachine.kernel.behaviorTable? := by
  native_decide

example : target.behaviorFingerprint.render =
    "sha256:9a131c48af0f15669b5f414754389046129da313f07774abbab76eaab25372b4" := by
  native_decide

example : target.machine.authoritativeInitial queuedSetup queuedState ∧
    target.machine.authoritativeStep queuedState dispatchAction dispatchedResult ∧
    target.machine.authoritativeStep runningState recordCancellationAction
      cancellationRecordedResult ∧
    target.machine.authoritativeStep runningState recordCompletionAction
      completionRecordedResult := by
  exact ⟨target_queued_initial_authoritative, target_queued_dispatch_authoritative,
    target_running_cancellation_authoritative, target_running_completion_authoritative⟩

end Temporal.System.Nexus.Tests

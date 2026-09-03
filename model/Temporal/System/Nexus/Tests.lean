import Temporal.System.Nexus.Core
import Temporal.System.Nexus.CallerClosure
import Temporal.System.Nexus.Observation

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

/-- Field specifications retain the authored Observation profile and checked identities. -/
example :
    Observation.Profile.declaration.kinds = [
      { id := Observation.Profile.cleanupKind, fields := [
          Observation.Profile.cleanupCommandKindFieldSpec.declaration,
          Observation.Profile.cleanupEndpointIdentityFieldSpec.declaration,
          Observation.Profile.cleanupErrorCodeFieldSpec.declaration,
          Observation.Profile.cleanupNamespaceIdentityFieldSpec.declaration,
          Observation.Profile.cleanupOpenHandleCountFieldSpec.declaration,
          Observation.Profile.cleanupOperationCorrelationFieldSpec.declaration,
          Observation.Profile.cleanupRunCorrelationFieldSpec.declaration,
          Observation.Profile.cleanupStatusFieldSpec.declaration,
          Observation.Profile.cleanupTaskQueueIdentityFieldSpec.declaration,
          Observation.Profile.cleanupWorkflowCorrelationFieldSpec.declaration
        ] },
      { id := Observation.Profile.controlReceiptKind, fields := [
          Observation.Profile.controlReceiptActionFieldSpec.declaration,
          Observation.Profile.controlReceiptAttemptFieldSpec.declaration,
          Observation.Profile.controlReceiptOccurrenceFieldSpec.declaration,
          Observation.Profile.controlReceiptStatusFieldSpec.declaration
        ] },
      { id := Observation.Profile.historyKind, fields := [
          Observation.Profile.historyEventIdFieldSpec.declaration,
          Observation.Profile.historyEventTypeFieldSpec.declaration,
          Observation.Profile.historyOperationCorrelationFieldSpec.declaration,
          Observation.Profile.historyRunCorrelationFieldSpec.declaration,
          Observation.Profile.historyWorkflowCorrelationFieldSpec.declaration
        ] },
      { id := Observation.Profile.participantKind, fields := [
          Observation.Profile.participantCancellationCountFieldSpec.declaration,
          Observation.Profile.participantCommandKindFieldSpec.declaration,
          Observation.Profile.participantEndpointIdentityFieldSpec.declaration,
          Observation.Profile.participantErrorCodeFieldSpec.declaration,
          Observation.Profile.participantNamespaceIdentityFieldSpec.declaration,
          Observation.Profile.participantOperationCorrelationFieldSpec.declaration,
          Observation.Profile.participantRunCorrelationFieldSpec.declaration,
          Observation.Profile.participantStatusFieldSpec.declaration,
          Observation.Profile.participantTaskQueueIdentityFieldSpec.declaration,
          Observation.Profile.participantWorkflowCorrelationFieldSpec.declaration
        ] }
    ] ∧
    Observation.checkedPlan.source = Observation.source ∧
    Observation.checkedPlan.behaviorFingerprint.render =
      "sha256:150c75ffcdd8b8e6e2ca8807c2c6ac7d924407b3291a0bc1f10ea04469a7df9b" ∧
    Observation.DuplicateDelivery.checkedPlan.behaviorFingerprint.render =
      "sha256:cc5910e77e3d43f4cad56de88a68f099eea8b25bbbe0fde451a02b2afda01438" := by
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

theorem callerClosureDefinitionsRetainCanonicalMetadata :
    Temporal.System.Nexus.CallerClosure.definitions = [
      { id := Temporal.System.Nexus.CallerClosure.targetId, kind := .target,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-target/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.kernelId, kind := .kernel,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-kernel/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.capabilityId, kind := .capability,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure/v1", documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.providerId, kind := .provider,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-provider/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.lifecycleCapabilityId, kind := .capability,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-lifecycle/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.ownershipCapabilityId, kind := .capability,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-ownership/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.lifecycleProviderId, kind := .provider,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-lifecycle-provider/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.ownershipProviderId, kind := .provider,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-ownership-provider/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.lawId, kind := .law,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := Temporal.System.Nexus.CallerClosure.law.body,
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.stateId, kind := .state,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-state/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.actionId, kind := .action,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-action/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.outcomeId, kind := .outcome,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-outcome/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.deliveryObservationId, kind := .observation,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-delivery/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.cancellationCountObservationId,
        kind := .observation, source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-count/v1",
        documentation := "" },
      { id := Temporal.System.Nexus.CallerClosure.ownershipObservationId, kind := .observation,
        source := Temporal.System.Nexus.CallerClosure.source, version := 1,
        canonicalBehavior := "temporal-system-nexus-caller-closure-ownership/v1",
        documentation := "" }
    ] := by
  native_decide

theorem callerClosureTargetAuthoringRetainsCompositionAndPlannerOrder :
    (checkTarget Temporal.System.Nexus.CallerClosure.targetAuthoring).isOk = true ∧
    Temporal.System.Nexus.CallerClosure.target.requiredCapabilities = [
      Temporal.System.Nexus.CallerClosure.capabilityId,
      Temporal.System.Nexus.CallerClosure.lifecycleCapabilityId,
      Temporal.System.Nexus.CallerClosure.ownershipCapabilityId
    ] ∧
    Temporal.System.Nexus.CallerClosure.target.providers.map CapabilityProvider.id = [
      Temporal.System.Nexus.CallerClosure.providerId,
      Temporal.System.Nexus.CallerClosure.lifecycleProviderId,
      Temporal.System.Nexus.CallerClosure.ownershipProviderId
    ] ∧
    Temporal.System.Nexus.CallerClosure.target.connectors = [] ∧
    Temporal.System.Nexus.CallerClosure.finitePlanning.actions = [
      Temporal.System.Nexus.CallerClosure.forceCloseAction
    ] ∧
    (match checkTarget Temporal.System.Nexus.CallerClosure.targetAuthoring with
      | .error _ => none
      | .ok checked =>
          some <| match checked.planning with
            | .unavailable => none
            | .available capability => some capability.actions) =
      some (some [Temporal.System.Nexus.CallerClosure.forceCloseAction]) := by
  native_decide

theorem callerClosureCheckedTargetRetainsCanonicalIdentity :
    (checkTarget Temporal.System.Nexus.CallerClosure.targetAuthoring).toOption.map (fun checked =>
      (checked.id, checked.source, canonicalCheckedTargetJson checked,
        checked.behaviorFingerprint)) =
      some (Temporal.System.Nexus.CallerClosure.targetId,
        Temporal.System.Nexus.CallerClosure.source,
        canonicalCheckedTargetJson Temporal.System.Nexus.CallerClosure.target,
        Temporal.System.Nexus.CallerClosure.target.behaviorFingerprint) ∧
    Temporal.System.Nexus.CallerClosure.target.behaviorFingerprint.render =
      "sha256:6729e790d336a96173ffd0ebe0b2b2d2406e6c5444596924f0c06c4ba9652bf8" ∧
    (behaviorFingerprintOf <|
      canonicalCheckedTargetJson Temporal.System.Nexus.CallerClosure.target).render =
      "sha256:dac443c8cb5c3ddb60391f746cac3d296f721014051c47d5a5fb3e4df5817744" := by
  native_decide

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

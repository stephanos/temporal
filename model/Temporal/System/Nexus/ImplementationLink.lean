import Temporal.Feature.Nexus.Caller.Model
import Temporal.System.Nexus.Evidence
import Temporal.System.Nexus.Core
import Umpire.ImplementationLink
import Umpire.Property.Elab
import Umpire.Property.Evaluate
import Umpire.Property.Correlated

/-!
# Nexus lifecycle Implementation Link

This is the sole production leaf that imports both the independently authored Nexus System
mechanism and Feature product meaning. It declares and proves the bounded forward correspondence;
neither base module imports or redefines the other.

The Feature side is the caller Model's product machine, `nexusProduct`: what an operation does,
with no account of how. Its own table is a Search view rather than a Target -- the two classes the
product machine cannot see, the transport fault and the worker stop, have no row, and it starts
only at `scheduled` -- so the destination here is a Target over the product machine's own rows: the
same states, outcomes and recorded facts, every class with a row, `started` admitted as a start
beside `scheduled` (the System's running setup begins with an operation already started), and the
machine's `ends:` as the terminal closure the cancellation projection closes on.
-/

namespace Temporal.System.Nexus.ImplementationLink

open Umpire
open Temporal.Feature.Nexus.Caller

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/System/Nexus/ImplementationLink.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def implementationLinkId : DefinitionId :=
  id "temporal.system.nexus.lifecycle.implementation-link"

/-! ### The product machine as a Target -/

/-- The product machine's rows as a Target's table: the classes with a row, `started` a start
beside `scheduled`, and `ends:` the terminal closure. -/
def productTable : FiniteTable nexusProduct.Setup ProductState nexusProduct.Action ProductOutcome
    ProductFact := {
  nexusProduct.table with
  actions := nexusProduct.table.actions.filter fun entry =>
    nexusProduct.table.transitions.any (·.action == entry.value)
  initial := [⟨nexusProduct.setupValue, [{ phase := .scheduled }, { phase := .started }]⟩]
  terminalConditions := [nexusProduct.ends] }

/-- The rows are the product machine's own, none added and none dropped. -/
theorem productTable_transitions : productTable.transitions = nexusProduct.table.transitions := rfl

def productTargetId : DefinitionId := id "temporal.system.nexus.lifecycle.product-target"
def productKernelId : DefinitionId := id "temporal.system.nexus.lifecycle.product-kernel"

/-- The Target's own identity over the product machine's definitions: a Target over more starts
than the machine declares is a new identity, and the machine's own is left to its Queries. -/
def productSpec : TableModelSpec := {
  nexusProduct.modelSpec with
  id := productTargetId
  source
  metadata := { id := productKernelId, source }
  definitions := nexusProduct.modelSpec.definitions.map fun definition =>
    if definition.id == nexusProduct.targetId then
      { definition with
        id := productTargetId
        source
        behaviorVersion := "temporal-system-nexus-lifecycle-product-target/v1" }
    else if definition.id == nexusProduct.kernelId then
      { definition with
        id := productKernelId
        source
        behaviorVersion := "temporal-system-nexus-lifecycle-product-kernel/v1" }
    else definition }

def productTargetResult : Except TableAdmissionError (QueryModel nexusProduct.lawStatement) :=
  productTable.checkModel nexusProduct.identity productSpec nexusProduct.composition

private theorem productTargetResult_isSome : productTargetResult.toOption.isSome = true := by
  native_decide

/-- The destination of the link: the product machine's rows, checked. Irreducible so that a goal
over its machine is never unfolded into the admission itself; every fact about it is decided over
the compiled admission. -/
@[irreducible] def productTarget : QueryModel nexusProduct.lawStatement :=
  productTargetResult.toOption.get productTargetResult_isSome

private def productDomainOption := match productTarget.machine.vocabulary with
  | .complete domain => some domain
  | _ => none

private theorem productDomainOption_isSome : productDomainOption.isSome = true := by native_decide

/-- The checked Target's own values, which is what every mapping below names. -/
def productDomain := productDomainOption.get productDomainOption_isSome

private def named (values : List ModelValue) (spelling : String)
    (present : (values.find? (·.value == spelling)).isSome = true := by native_decide) :
    ModelValue :=
  (values.find? (·.value == spelling)).get present

def scheduledState : ModelValue := named productDomain.states "scheduled"
def startedState : ModelValue := named productDomain.states "started"
def succeededState : ModelValue := named productDomain.states "succeeded"
def canceledState : ModelValue := named productDomain.states "canceled"

/-- The handler's asynchronous reply, which starts the operation. -/
def asyncReplyAction : ModelValue := named productDomain.actions "handlerReply-async"
/-- The caller's completion recording a cancellation. -/
def cancelCompletionAction : ModelValue := named productDomain.actions "complete-canceled"
/-- The caller's completion recording a success. -/
def successCompletionAction : ModelValue := named productDomain.actions "complete-succeeded"

def acceptedOutcome : ModelValue := named productDomain.outcomes "accepted"

def startedFact : ModelValue := named productDomain.observations "nexusOperationStarted"
def canceledFact : ModelValue := named productDomain.observations "nexusOperationCanceled"
def completedFact : ModelValue := named productDomain.observations "nexusOperationCompleted"

/-- The one setup the Target has: the operation role bound to the first start. Both System setups
map to it; which start an operation begins at is the state, not the setup. -/
def productSetup : List RoleBinding := [{ «role» := nexusProduct.operationRoleId, value := scheduledState }]

def startedResult : Step ModelValue ModelValue ModelValue :=
  { outcome := acceptedOutcome, state := startedState, facts := [startedFact] }
def canceledResult : Step ModelValue ModelValue ModelValue :=
  { outcome := acceptedOutcome, state := canceledState, facts := [canceledFact] }
def succeededResult : Step ModelValue ModelValue ModelValue :=
  { outcome := acceptedOutcome, state := succeededState, facts := [completedFact] }

/-! ### The maps -/

def mapSetup : Temporal.System.Nexus.ExecutionSetup → List RoleBinding
  | .queued => productSetup
  | .running => productSetup

def mapState (state : ModelValue) : ModelValue :=
  if state = Temporal.System.Nexus.queuedState then scheduledState
  else if state = Temporal.System.Nexus.runningState then startedState
  else if state = Temporal.System.Nexus.cancellationRecordedState then canceledState
  else succeededState

def mapAction (action : ModelValue) : ModelValue :=
  if action = Temporal.System.Nexus.dispatchAction then asyncReplyAction
  else if action = Temporal.System.Nexus.recordCancellationAction then cancelCompletionAction
  else successCompletionAction

/-- Every System transition is accepted by the product machine: the three transition outcomes map
to its one accepting outcome. -/
def mapOutcome (_outcome : ModelValue) : ModelValue := acceptedOutcome

def mapObservation (observation : ModelValue) : ModelValue :=
  if observation = Temporal.System.Nexus.runningObservation then startedFact
  else if observation = Temporal.System.Nexus.cancellationRecordedObservation then canceledFact
  else completedFact

@[simp] theorem mapSetup_queued : mapSetup Temporal.System.Nexus.queuedSetup = productSetup := rfl
@[simp] theorem mapSetup_running : mapSetup Temporal.System.Nexus.runningSetup = productSetup := rfl

@[simp] theorem mapState_queued :
    mapState Temporal.System.Nexus.queuedState = scheduledState := by native_decide
@[simp] theorem mapState_running :
    mapState Temporal.System.Nexus.runningState = startedState := by native_decide
@[simp] theorem mapState_cancellationRecorded :
    mapState Temporal.System.Nexus.cancellationRecordedState = canceledState := by native_decide
@[simp] theorem mapState_completionRecorded :
    mapState Temporal.System.Nexus.completionRecordedState = succeededState := by native_decide

@[simp] theorem mapAction_dispatch :
    mapAction Temporal.System.Nexus.dispatchAction = asyncReplyAction := by native_decide
@[simp] theorem mapAction_recordCancellation :
    mapAction Temporal.System.Nexus.recordCancellationAction = cancelCompletionAction := by
  native_decide
@[simp] theorem mapAction_recordCompletion :
    mapAction Temporal.System.Nexus.recordCompletionAction = successCompletionAction := by
  native_decide

@[simp] theorem mapOutcome_dispatched :
    mapOutcome Temporal.System.Nexus.dispatchedOutcome = acceptedOutcome := rfl
@[simp] theorem mapOutcome_cancellationRecorded :
    mapOutcome Temporal.System.Nexus.cancellationRecordedOutcome = acceptedOutcome := rfl
@[simp] theorem mapOutcome_completionRecorded :
    mapOutcome Temporal.System.Nexus.completionRecordedOutcome = acceptedOutcome := rfl

@[simp] theorem mapObservation_running :
    mapObservation Temporal.System.Nexus.runningObservation = startedFact := by native_decide
@[simp] theorem mapObservation_cancellationRecorded :
    mapObservation Temporal.System.Nexus.cancellationRecordedObservation = canceledFact := by
  native_decide
@[simp] theorem mapObservation_completionRecorded :
    mapObservation Temporal.System.Nexus.completionRecordedObservation = completedFact := by
  native_decide

def sourceCapabilityReference : ImplementationSemanticReference :=
  (implementationSemanticReference? Temporal.System.Nexus.target
    Temporal.System.Nexus.lifecycleCapabilityId .capability).get (by native_decide)

def destinationCapabilityReference : ImplementationSemanticReference :=
  (implementationSemanticReference? productTarget nexusProduct.capabilityId .capability).get
    (by native_decide)

def lifecycleCapabilityMapping : ImplementationSemanticMapping :=
  .forward sourceCapabilityReference destinationCapabilityReference

def declaration : ImplementationLinkDeclaration
    Temporal.System.Nexus.ExecutionSetup ModelValue ModelValue ModelValue ModelValue
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := implementationLinkId
  source
  sourceTarget := .ofTarget Temporal.System.Nexus.target
  destinationTarget := .ofTarget productTarget
  setupMappings := [
    .forward Temporal.System.Nexus.queuedSetup productSetup,
    .forward Temporal.System.Nexus.runningSetup productSetup
  ]
  stateMappings := [
    .forward Temporal.System.Nexus.queuedState scheduledState,
    .forward Temporal.System.Nexus.runningState startedState,
    .forward Temporal.System.Nexus.cancellationRecordedState canceledState,
    .forward Temporal.System.Nexus.completionRecordedState succeededState
  ]
  actionMappings := [
    .forward Temporal.System.Nexus.dispatchAction asyncReplyAction,
    .forward Temporal.System.Nexus.recordCancellationAction cancelCompletionAction,
    .forward Temporal.System.Nexus.recordCompletionAction successCompletionAction
  ]
  outcomeMappings := [
    .forward Temporal.System.Nexus.dispatchedOutcome acceptedOutcome,
    .forward Temporal.System.Nexus.cancellationRecordedOutcome acceptedOutcome,
    .forward Temporal.System.Nexus.completionRecordedOutcome acceptedOutcome
  ]
  observationMappings := [
    .forward Temporal.System.Nexus.runningObservation startedFact,
    .forward Temporal.System.Nexus.cancellationRecordedObservation canceledFact,
    .forward Temporal.System.Nexus.completionRecordedObservation completedFact
  ]
  relationMappings := []
  capabilityMappings := [lifecycleCapabilityMapping]
  applicationLimit := { value := 3, unit := .steps }
  documentation := "The pure Nexus System lifecycle forward-simulates the caller Model's product machine."
}

theorem requiredCoverage : ImplementationLinkRequiredCoverage declaration
    Temporal.System.Nexus.target mapSetup mapState mapAction mapOutcome mapObservation := {
  setup := by
    intro value admitted
    change value = Temporal.System.Nexus.queuedSetup ∨
      value = Temporal.System.Nexus.runningSetup at admitted
    rcases admitted with rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.queuedSetup
        (mapSetup Temporal.System.Nexus.queuedSetup) ∈ declaration.setupMappings
      rw [mapSetup_queued]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.runningSetup
        (mapSetup Temporal.System.Nexus.runningSetup) ∈ declaration.setupMappings
      rw [mapSetup_running]
      exact List.mem_cons.mpr (.inr (List.mem_singleton.mpr rfl))
  state := by
    intro value admitted
    change value = Temporal.System.Nexus.queuedState ∨
      value = Temporal.System.Nexus.runningState ∨
      value = Temporal.System.Nexus.cancellationRecordedState ∨
      value = Temporal.System.Nexus.completionRecordedState at admitted
    rcases admitted with rfl | rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.queuedState
        (mapState Temporal.System.Nexus.queuedState) ∈ declaration.stateMappings
      rw [mapState_queued]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.runningState
        (mapState Temporal.System.Nexus.runningState) ∈ declaration.stateMappings
      rw [mapState_running]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.cancellationRecordedState
        (mapState Temporal.System.Nexus.cancellationRecordedState) ∈ declaration.stateMappings
      rw [mapState_cancellationRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_cons.mpr (.inl rfl)))))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.completionRecordedState
        (mapState Temporal.System.Nexus.completionRecordedState) ∈ declaration.stateMappings
      rw [mapState_completionRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_cons.mpr (.inr (List.mem_singleton.mpr rfl))))))
  action := by
    intro value admitted
    change value = Temporal.System.Nexus.dispatchAction ∨
      value = Temporal.System.Nexus.recordCancellationAction ∨
      value = Temporal.System.Nexus.recordCompletionAction at admitted
    rcases admitted with rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.dispatchAction
        (mapAction Temporal.System.Nexus.dispatchAction) ∈ declaration.actionMappings
      rw [mapAction_dispatch]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.recordCancellationAction
        (mapAction Temporal.System.Nexus.recordCancellationAction) ∈ declaration.actionMappings
      rw [mapAction_recordCancellation]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.recordCompletionAction
        (mapAction Temporal.System.Nexus.recordCompletionAction) ∈ declaration.actionMappings
      rw [mapAction_recordCompletion]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_singleton.mpr rfl))))
  outcome := by
    intro value admitted
    change value = Temporal.System.Nexus.dispatchedOutcome ∨
      value = Temporal.System.Nexus.cancellationRecordedOutcome ∨
      value = Temporal.System.Nexus.completionRecordedOutcome at admitted
    rcases admitted with rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.dispatchedOutcome
        (mapOutcome Temporal.System.Nexus.dispatchedOutcome) ∈ declaration.outcomeMappings
      rw [mapOutcome_dispatched]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.cancellationRecordedOutcome
        (mapOutcome Temporal.System.Nexus.cancellationRecordedOutcome) ∈ declaration.outcomeMappings
      rw [mapOutcome_cancellationRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.completionRecordedOutcome
        (mapOutcome Temporal.System.Nexus.completionRecordedOutcome) ∈ declaration.outcomeMappings
      rw [mapOutcome_completionRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_singleton.mpr rfl))))
  observation := by
    intro value admitted
    change value = Temporal.System.Nexus.runningObservation ∨
      value = Temporal.System.Nexus.cancellationRecordedObservation ∨
      value = Temporal.System.Nexus.completionRecordedObservation at admitted
    rcases admitted with rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.runningObservation
        (mapObservation Temporal.System.Nexus.runningObservation) ∈ declaration.observationMappings
      rw [mapObservation_running]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.cancellationRecordedObservation
        (mapObservation Temporal.System.Nexus.cancellationRecordedObservation) ∈
          declaration.observationMappings
      rw [mapObservation_cancellationRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.completionRecordedObservation
        (mapObservation Temporal.System.Nexus.completionRecordedObservation) ∈
          declaration.observationMappings
      rw [mapObservation_completionRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_singleton.mpr rfl))))
  relation := by native_decide
  capability := by native_decide
}

/-- Membership in a checked list, decided by its own search rather than a lawful `BEq`. -/
private theorem mem_of_found {α : Type} [BEq α] {value : α} {values : List α}
    (found : values.find? (· == value) = some value) : value ∈ values :=
  List.mem_of_find?_eq_some found

/-- The value translation of the link, from the System's kernel values to the product Target's. -/
def morphism : ValueTranslation Temporal.System.Nexus.ExecutionSetup ModelValue ModelValue
    ModelValue ModelValue (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  { mapSetup, mapState, mapAction, mapOutcome, mapObservation }

/-- Each System start lands on a start of the checked product Target, whose own soundness law turns
the found start into the authoritative relation: decided over the Target, not proved start by start. -/
theorem initialForward (setup : Temporal.System.Nexus.ExecutionSetup) (state : ModelValue)
    (admitted : Temporal.System.Nexus.target.machine.authoritativeInitial setup state) :
    productTarget.machine.authoritativeInitial (morphism.mapSetup setup)
      (morphism.mapState state) := by
  change Temporal.System.Nexus.authoritativeInitial setup state at admitted
  rcases Temporal.System.Nexus.authoritativeInitial_cases setup state admitted with
    ⟨rfl, rfl⟩ | ⟨rfl, rfl⟩
  · refine productTarget.machine.initialSound _ _ ?_
    exact mem_of_found (by native_decide)
  · refine productTarget.machine.initialSound _ _ ?_
    exact mem_of_found (by native_decide)

/-- Each System step lands on a row of the checked product Target, the same way. -/
theorem stepForward (state action : ModelValue) (result : Step ModelValue ModelValue ModelValue)
    (admitted : Temporal.System.Nexus.target.machine.authoritativeStep state action result) :
    productTarget.machine.authoritativeStep (morphism.mapState state) (morphism.mapAction action)
      (morphism.mapStep result) := by
  change Temporal.System.Nexus.authoritativeStep state action result at admitted
  rcases Temporal.System.Nexus.authoritativeStep_cases state action result admitted with
    ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩
  · refine productTarget.machine.stepSound _ _ _ ?_
    exact mem_of_found (by native_decide)
  · refine productTarget.machine.stepSound _ _ _ ?_
    exact mem_of_found (by native_decide)
  · refine productTarget.machine.stepSound _ _ _ ?_
    exact mem_of_found (by native_decide)

/-- The forward simulation: the value translation with its two decided laws. -/
def witness : ImplementationLinkWitness declaration Temporal.System.Nexus.target productTarget := {
  index := implementationLinkWitnessIndex declaration Temporal.System.Nexus.target productTarget
  stepPreservation := { morphism, initialForward, stepForward }
  requiredCoverage
}

def checkedResult := checkImplementationLink declaration Temporal.System.Nexus.target productTarget
  witness

private theorem checkedResult_isSome : checkedResult.toOption.isSome = true := by
  native_decide

/-- The checked first Temporal System-to-Feature correspondence. -/
def checked := checkedResult.toOption.get checkedResult_isSome

/-- The semantic layer responsible for one System-to-Feature Property result. -/
inductive FeaturePropertyLayer where
  | observation
  | implementationLink
  | featureProperty
  deriving BEq, DecidableEq, Repr

/-- Successful composition retains both the complete Implementation Link Evidence Links and the
unchanged Feature Property evaluation. -/
structure EvaluatedFeatureProperty where
  application : AppliedImplementationLink checked
  evaluation : PropertyEvaluation

/-- Observation, Implementation Link, and Feature Property outcomes remain disjoint. -/
inductive FeaturePropertyResult where
  | observationFailure (diagnostic : ObservationDiagnostic)
  | implementationLinkFailure (diagnostic : ImplementationLinkDiagnostic)
  | propertyFailure (diagnostic : PropertyError)
  | evaluated (result : EvaluatedFeatureProperty)

def FeaturePropertyResult.layer : FeaturePropertyResult → FeaturePropertyLayer
  | .observationFailure _ => .observation
  | .implementationLinkFailure _ => .implementationLink
  | .propertyFailure _ => .featureProperty
  | .evaluated _ => .featureProperty

def FeaturePropertyResult.observationDiagnostic? :
    FeaturePropertyResult → Option ObservationDiagnostic
  | .observationFailure diagnostic => some diagnostic
  | _ => none

def FeaturePropertyResult.implementationLinkDiagnostic? :
    FeaturePropertyResult → Option ImplementationLinkDiagnostic
  | .implementationLinkFailure diagnostic => some diagnostic
  | _ => none

def FeaturePropertyResult.evaluated? :
    FeaturePropertyResult → Option EvaluatedFeatureProperty
  | .evaluated result => some result
  | _ => none

def FeaturePropertyResult.propertyDiagnostic? :
    FeaturePropertyResult → Option PropertyError
  | .propertyFailure diagnostic => some diagnostic
  | _ => none

/-- Compose an upstream Observation result through the checked Nexus Implementation Link. Property
evaluation runs only after the source trace is re-admitted and translated successfully. -/
def evaluateFeatureProperty
    (sourceSetup : Temporal.System.Nexus.ExecutionSetup)
    (checkedProperty : CheckedProperty)
    (observation : ObservationResult) : FeaturePropertyResult :=
  match observation with
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic =>
      .observationFailure diagnostic
  | .accepted trace =>
      match applyImplementationLink checked sourceSetup trace with
      | .applied application =>
          match evaluatePropertyOnTrace checkedProperty application.trace with
          | .ok evaluation => .evaluated { application, evaluation }
          | .error diagnostic => .propertyFailure diagnostic
      | .invalid diagnostic
      | .unknown diagnostic
      | .conflict diagnostic
      | .unsupported diagnostic => .implementationLinkFailure diagnostic

end Temporal.System.Nexus.ImplementationLink


namespace Temporal.System.Nexus.ImplementationLink.Cancellation

open Umpire Case.Projection
open Temporal.Feature.Nexus.Caller

/-- Cancellation admission preserves which owner rejected the declaration or source evidence. -/
inductive Error where
  | correlation (error : Evidence.Error)
  | projection (error : Case.Projection.Error)

/-- Target-bound cancellation mapping. Only `check` constructs this checked declaration. -/
structure Checked where
  private mk ::
  target : QueryModel nexusProduct.lawStatement
  private plan : Case.Projection.Checked target
  private maxOperations : Nat

/-- The product machine records no cancellation request: `nexusProduct` resolves a started operation
straight to `canceled` on the handler's completion, so the request and its confirmation carry no
step of their own (fn-79 keeps the request row deferred). -/
private def declaration (bounds : Case.Projection.Limits) :
    Declaration ModelValue ModelValue ModelValue ModelValue := {
  id := Evidence.field "cancellation-projection"
  scopeFields := [Evidence.field "execution", Evidence.field "namespace",
    Evidence.field "workflow", Evidence.field "run"]
  operationField := Evidence.field "scheduled-operation-request"
  sources := [Evidence.Source.sdk.id, Evidence.Source.history.id]
  rules := [
    { kind := Evidence.Kind.cancellationSubmitted.id, meaning := .irrelevant },
    { kind := Evidence.Kind.cancellationConfirmed.id, meaning := .irrelevant },
    { kind := Evidence.Kind.canceled.id,
      meaning := .confirmed none [(cancelCompletionAction, canceledResult)] },
    { kind := Evidence.Kind.completed.id,
      meaning := .confirmed none [(successCompletionAction, succeededResult)] },
    { kind := Evidence.Kind.unrelated.id, meaning := .irrelevant },
    { kind := Evidence.Kind.workflowCancellation.id, meaning := .irrelevant },
    { kind := Evidence.Kind.activationShutdown.id, meaning := .irrelevant }]
  «limits» := bounds
}

/-- Check the evidence mapping against the product Target; no Query witness is an input. -/
def check (bounds : Case.Projection.Limits) : Except Error Checked := do
  let plan ← Case.Projection.check productTarget (declaration bounds) productSetup startedState
    |>.mapError .projection
  pure ⟨productTarget, plan, bounds.keys⟩

/-- Canonical mapping provenance includes Target terminal semantics and independent evidence limits. -/
def Checked.behaviorFingerprint (checked : Checked) : BehaviorFingerprint :=
  checked.plan.behaviorFingerprint

/-- One immutable Run, containing its frozen correlation authority and generic projection state. -/
structure Run (checked : Checked) where
  private mk ::
  private binding : Evidence.Binding
  private projection : Case.Projection.Run checked.plan

/-- Allocate fresh state after validating the entire operation binding. -/
def Checked.start (checked : Checked) (binding : Evidence.Binding) : Except Error (Run checked) := do
  binding.validate checked.maxOperations |>.mapError .correlation
  let projection ← checked.plan.start binding.scope.fields |>.mapError .projection
  pure ⟨binding, projection⟩

variable {checked : Checked}

/-- Confirmed steps retain checked product authority and exact source/Run support. -/
def Run.steps (run : Run checked) : List (Step checked.target) := run.projection.steps

/-- Accepted closed evidence is immutable even when a later append fails. -/
def Run.accepted (run : Run checked) : List Event := run.projection.accepted

/-- Missing causal support stays distinct from a justified stutter. -/
def Run.pending (run : Run checked) : List Identity := run.projection.pending

/-- Admit correlation first, then atomically release every causally supported semantic step. -/
def Run.admit (run : Run checked) (record : Evidence.Record) :
    Except Error (Run checked × Progress (Step checked.target)) := do
  let event ← record.toEvent run.binding |>.mapError .correlation
  let (projection, progress) ← run.projection.admit event |>.mapError .projection
  pure (⟨run.binding, projection⟩, progress)

/-- Close only when every bound operation has reached one of the product machine's `ends:`. -/
def Run.close (run : Run checked) : Except Error (Run checked) := do
  let projection ← run.projection.close |>.mapError .projection
  let unresolved := run.binding.operations.filter fun operation =>
    !checked.target.isTerminal (projection.state operation.key)
  unless unresolved.isEmpty do
    throw (.projection (.nonterminal (unresolved.map Evidence.Operation.key)))
  pure ⟨run.binding, projection⟩

end Temporal.System.Nexus.ImplementationLink.Cancellation

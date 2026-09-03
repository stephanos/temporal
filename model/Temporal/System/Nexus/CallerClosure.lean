import Umpire.Planning

/-!
# Temporal Nexus caller-closure mechanism

Pure System meaning for the single closed caller-closure transition established from runtime
evidence. Feature product meaning remains outside this module and is related only by the checked
Implementation Link leaf.
-/

namespace Temporal.System.Nexus.CallerClosure

open Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/System/Nexus/CallerClosure.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def targetId : DefinitionId := id "temporal.system.nexus.caller-closure.target"
def kernelId : DefinitionId := id "temporal.system.nexus.caller-closure.kernel"
def capabilityId : DefinitionId := id "temporal.system.nexus.caller-closure.capability"
def providerId : DefinitionId := id "temporal.system.nexus.caller-closure.provider"
def lifecycleCapabilityId : DefinitionId :=
  id "temporal.system.nexus.caller-closure.capability.lifecycle"
def ownershipCapabilityId : DefinitionId :=
  id "temporal.system.nexus.caller-closure.capability.ownership"
def lifecycleProviderId : DefinitionId :=
  id "temporal.system.nexus.caller-closure.provider.lifecycle"
def ownershipProviderId : DefinitionId :=
  id "temporal.system.nexus.caller-closure.provider.ownership"
def lawId : DefinitionId := id "temporal.system.nexus.caller-closure.law"
def stateId : DefinitionId := id "temporal.system.nexus.caller-closure.state"
def actionId : DefinitionId := id "temporal.system.nexus.caller-closure.action"
def outcomeId : DefinitionId := id "temporal.system.nexus.caller-closure.outcome"
def deliveryObservationId : DefinitionId :=
  id "temporal.system.nexus.caller-closure.observation.delivery"
def cancellationCountObservationId : DefinitionId :=
  id "temporal.system.nexus.caller-closure.observation.cancellation-count"
def ownershipObservationId : DefinitionId :=
  id "temporal.system.nexus.caller-closure.observation.ownership"
def operationRoleId : DefinitionId := id "temporal.system.nexus.caller-closure.role.operation"

def openState : ModelValue :=
  ModelValue.named stateId "temporal.history.WorkflowExecutionStarted"
def closedState : ModelValue :=
  ModelValue.named stateId "temporal.history.WorkflowExecutionCanceled"
def forceCloseAction : ModelValue := ModelValue.named actionId "force-close"
def cancellationUpgradedOutcome : ModelValue := ModelValue.named outcomeId "upgrade"
def deliveryObservation : ModelValue := ModelValue.named deliveryObservationId "true"
def cancellationCountObservation : ModelValue :=
  ModelValue.named cancellationCountObservationId "1"
def ownershipObservation : ModelValue := ModelValue.named ownershipObservationId "true"

def setup : List RoleBinding := [{ role := operationRoleId, value := openState }]

def closeResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := cancellationUpgradedOutcome
  resultingState := closedState
  observations := [deliveryObservation, cancellationCountObservation, ownershipObservation]
}

def law : LawDefinition := { id := lawId, body := "temporal-system-nexus-caller-closure/v1" }

def LawStatement (candidate : LawDefinition) : Prop := candidate = law

theorem lawProof : LawStatement law := rfl

def authoritativeInitial (candidateSetup : List RoleBinding) (state : ModelValue) : Prop :=
  candidateSetup = setup ∧ state = openState

def authoritativeStep
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) : Prop :=
  state = openState ∧ action = forceCloseAction ∧ result = closeResult

/-- Ordered finite authority for caller closure: the sole setup admits the open state, and only
force-close produces `closeResult`. The public kernel and planning declarations below are
compatibility projections of this machine. -/
def finiteMachine : FiniteMachine
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := { id := kernelId, source }
  setups := [setup]
  states := [openState, closedState]
  actions := [forceCloseAction]
  outcomes := [cancellationUpgradedOutcome]
  observations := [deliveryObservation, cancellationCountObservation, ownershipObservation]
  encodeSetup := fun _ => "caller-closure"
  encodeState := reprStr
  encodeAction := reprStr
  encodeOutcome := reprStr
  encodeObservation := reprStr
  initialStates := fun candidate => if candidate = setup then [openState] else []
  steps := fun state action =>
    if state = openState ∧ action = forceCloseAction then [closeResult] else []
  setupCoverage := by intro candidate state member; simp_all
  initialStateCoverage := by intro candidate state member; simp_all
  transitionSourceCoverage := by
    intro state action result member
    by_cases selected : state = openState ∧ action = forceCloseAction
    · simp [selected.1]
    · simp [selected] at member
  actionCoverage := by
    intro state action result member
    by_cases selected : state = openState ∧ action = forceCloseAction
    · simp [selected.2]
    · simp [selected] at member
  resultingStateCoverage := by
    intro state action result member
    by_cases selected : state = openState ∧ action = forceCloseAction
    · rw [if_pos selected] at member
      simp [List.mem_singleton.mp member, closeResult]
    · rw [if_neg selected] at member
      exact (List.not_mem_nil member).elim
  outcomeCoverage := by
    intro state action result member
    by_cases selected : state = openState ∧ action = forceCloseAction
    · rw [if_pos selected] at member
      simp [List.mem_singleton.mp member, closeResult]
    · rw [if_neg selected] at member
      exact (List.not_mem_nil member).elim
  observationCoverage := by
    intro state action result observation member observationMember
    by_cases selected : state = openState ∧ action = forceCloseAction
    · rw [if_pos selected] at member
      simpa [List.mem_singleton.mp member, closeResult] using observationMember
    · rw [if_neg selected] at member
      exact (List.not_mem_nil member).elim
  actionExecutable := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨openState, closeResult, by simp⟩
}

private theorem setupDomain_eq :
    (fun candidate => candidate = setup) = finiteMachine.kernel.setupDomain := by
  funext candidate
  apply propext
  simp [finiteMachine]

private theorem stateDomain_eq :
    (fun candidate => candidate = openState ∨ candidate = closedState) =
      finiteMachine.kernel.stateDomain := by
  funext candidate
  apply propext
  simp [finiteMachine]

private theorem actionDomain_eq :
    (fun candidate => candidate = forceCloseAction) = finiteMachine.kernel.actionDomain := by
  funext candidate
  apply propext
  simp [finiteMachine]

private theorem outcomeDomain_eq :
    (fun candidate => candidate = cancellationUpgradedOutcome) =
      finiteMachine.kernel.outcomeDomain := by
  funext candidate
  apply propext
  simp [finiteMachine]

private theorem observationDomain_eq :
    (fun candidate => candidate = deliveryObservation ∨
      candidate = cancellationCountObservation ∨ candidate = ownershipObservation) =
      finiteMachine.kernel.observationDomain := by
  funext candidate
  apply propext
  simp [finiteMachine]

private theorem authoritativeInitial_iff
    (candidateSetup : List RoleBinding) (state : ModelValue) :
    finiteMachine.kernel.authoritativeInitial candidateSetup state ↔
      authoritativeInitial candidateSetup state := by
  simp [finiteMachine, authoritativeInitial]

private theorem authoritativeStep_iff
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) :
    finiteMachine.kernel.authoritativeStep state action result ↔
      authoritativeStep state action result := by
  simp [finiteMachine, authoritativeStep, and_assoc]

def transitionKernel : TransitionKernel
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := finiteMachine.kernel.metadata
  setupDomain := fun candidate => candidate = setup
  stateDomain := fun candidate => candidate = openState ∨ candidate = closedState
  actionDomain := fun candidate => candidate = forceCloseAction
  outcomeDomain := fun candidate => candidate = cancellationUpgradedOutcome
  observationDomain := fun candidate => candidate = deliveryObservation ∨
    candidate = cancellationCountObservation ∨ candidate = ownershipObservation
  initialStates := finiteMachine.kernel.initialStates
  authoritativeInitial
  initialSound := by
    intro candidate state member
    exact (authoritativeInitial_iff candidate state).mp
      (finiteMachine.kernel.initialSound candidate state member)
  initialComplete := by
    intro candidate state admitted
    exact finiteMachine.kernel.initialComplete candidate state
      ((authoritativeInitial_iff candidate state).mpr admitted)
  steps := finiteMachine.kernel.steps
  authoritativeStep
  stepSound := by
    intro state action result member
    exact (authoritativeStep_iff state action result).mp
      (finiteMachine.kernel.stepSound state action result member)
  stepComplete := by
    intro state action result admitted
    exact finiteMachine.kernel.stepComplete state action result
      ((authoritativeStep_iff state action result).mpr admitted)
  behaviorDomain := by
    rw [setupDomain_eq, stateDomain_eq, actionDomain_eq, outcomeDomain_eq,
      observationDomain_eq]
    exact finiteMachine.kernel.behaviorDomain
}

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata := {
  id := definitionId
  kind
  source
  canonicalBehavior
}

private def witness : LawWitness LawStatement := { definition := law, proof := lawProof }

def provider : CapabilityProvider LawStatement := {
  id := providerId
  source
  contract := {
    id := capabilityId
    canonicalBehavior := "temporal-system-nexus-caller-closure/v1"
    requiredLaws := [law]
  }
  meanings := [
    { definitionId := stateId, kind := .state,
      canonicalBehavior := "temporal-system-nexus-caller-closure-state/v1" },
    { definitionId := actionId, kind := .action,
      canonicalBehavior := "temporal-system-nexus-caller-closure-action/v1" },
    { definitionId := outcomeId, kind := .outcome,
      canonicalBehavior := "temporal-system-nexus-caller-closure-outcome/v1" },
    { definitionId := deliveryObservationId, kind := .observation,
      canonicalBehavior := "temporal-system-nexus-caller-closure-delivery/v1" },
    { definitionId := cancellationCountObservationId, kind := .observation,
      canonicalBehavior := "temporal-system-nexus-caller-closure-count/v1" },
    { definitionId := ownershipObservationId, kind := .observation,
      canonicalBehavior := "temporal-system-nexus-caller-closure-ownership/v1" }
  ]
  lawWitnesses := [witness]
}

private def auxiliaryProvider
    (definitionId capability : DefinitionId)
    (canonicalBehavior : String) : CapabilityProvider LawStatement := {
  id := definitionId
  source
  contract := {
    id := capability
    canonicalBehavior
    requiredLaws := [law]
  }
  meanings := []
  lawWitnesses := [witness]
}

def lifecycleProvider : CapabilityProvider LawStatement :=
  auxiliaryProvider lifecycleProviderId lifecycleCapabilityId
    "temporal-system-nexus-caller-closure-lifecycle/v1"

def ownershipProvider : CapabilityProvider LawStatement :=
  auxiliaryProvider ownershipProviderId ownershipCapabilityId
    "temporal-system-nexus-caller-closure-ownership/v1"

def definitions : List DefinitionMetadata := [
  metadata targetId .target "temporal-system-nexus-caller-closure-target/v1",
  metadata kernelId .kernel "temporal-system-nexus-caller-closure-kernel/v1",
  metadata capabilityId .capability "temporal-system-nexus-caller-closure/v1",
  metadata providerId .provider "temporal-system-nexus-caller-closure-provider/v1",
  metadata lifecycleCapabilityId .capability
    "temporal-system-nexus-caller-closure-lifecycle/v1",
  metadata ownershipCapabilityId .capability
    "temporal-system-nexus-caller-closure-ownership/v1",
  metadata lifecycleProviderId .provider
    "temporal-system-nexus-caller-closure-lifecycle-provider/v1",
  metadata ownershipProviderId .provider
    "temporal-system-nexus-caller-closure-ownership-provider/v1",
  metadata lawId .law law.body,
  metadata stateId .state "temporal-system-nexus-caller-closure-state/v1",
  metadata actionId .action "temporal-system-nexus-caller-closure-action/v1",
  metadata outcomeId .outcome "temporal-system-nexus-caller-closure-outcome/v1",
  metadata deliveryObservationId .observation
    "temporal-system-nexus-caller-closure-delivery/v1",
  metadata cancellationCountObservationId .observation
    "temporal-system-nexus-caller-closure-count/v1",
  metadata ownershipObservationId .observation
    "temporal-system-nexus-caller-closure-ownership/v1"
]

def targetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [
    capabilityId,
    lifecycleCapabilityId,
    ownershipCapabilityId
  ]
  resolvedSetups := [setup]
  kernel := .checked transitionKernel
}

def finitePlanning : FinitePlanningCapability transitionKernel.authoritativeStep := {
  actions := finiteMachine.planning.actions
  actionSound := by
    intro action member
    rcases finiteMachine.planning.actionSound action member with ⟨state, result, admitted⟩
    exact ⟨state, result, (authoritativeStep_iff state action result).mp admitted⟩
  actionComplete := by
    intro state action result admitted
    exact finiteMachine.planning.actionComplete state action result
      ((authoritativeStep_iff state action result).mpr admitted)
}

def targetAuthoring : AuthoredTarget LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make targetDefinition
    (TargetComposition.empty
      |>.provide provider
      |>.provide lifecycleProvider
      |>.provide ownershipProvider)
    (.available transitionKernel rfl finitePlanning)

def target : QueryTarget LawStatement := checkedTarget targetAuthoring

end Temporal.System.Nexus.CallerClosure

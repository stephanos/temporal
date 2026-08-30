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

def openState : ModelValue := {
  definitionId := stateId
  value := "temporal.history.WorkflowExecutionStarted"
}
def closedState : ModelValue := {
  definitionId := stateId
  value := "temporal.history.WorkflowExecutionCanceled"
}
def forceCloseAction : ModelValue := { definitionId := actionId, value := "force-close" }
def cancellationUpgradedOutcome : ModelValue := {
  definitionId := outcomeId
  value := "upgrade"
}
def deliveryObservation : ModelValue := { definitionId := deliveryObservationId, value := "true" }
def cancellationCountObservation : ModelValue := {
  definitionId := cancellationCountObservationId
  value := "1"
}
def ownershipObservation : ModelValue := { definitionId := ownershipObservationId, value := "true" }

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

def transitionKernel : TransitionKernel
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := { id := kernelId, source }
  setupDomain := fun candidate => candidate = setup
  stateDomain := fun candidate => candidate = openState ∨ candidate = closedState
  actionDomain := fun candidate => candidate = forceCloseAction
  outcomeDomain := fun candidate => candidate = cancellationUpgradedOutcome
  observationDomain := fun candidate => candidate = deliveryObservation ∨
    candidate = cancellationCountObservation ∨ candidate = ownershipObservation
  initialStates := fun candidate => if candidate = setup then [openState] else []
  authoritativeInitial
  initialSound := by
    intro candidate state member
    by_cases selected : candidate = setup
    · simp [selected] at member
      exact ⟨selected, member⟩
    · simp [selected] at member
  initialComplete := by
    intro candidate state admitted
    rcases admitted with ⟨rfl, rfl⟩
    simp
  steps := fun state action =>
    if state = openState ∧ action = forceCloseAction then [closeResult] else []
  authoritativeStep
  stepSound := by
    intro state action result member
    by_cases selected : state = openState ∧ action = forceCloseAction
    · simp [selected] at member
      exact ⟨selected.1, selected.2, member⟩
    · simp [selected] at member
  stepComplete := by
    intro state action result admitted
    rcases admitted with ⟨rfl, rfl, rfl⟩
    simp
  behaviorDomain := .complete {
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
    setupSound := by simp
    setupComplete := by intro candidate admitted; simpa using admitted
    stateSound := by simp
    stateComplete := by intro candidate admitted; simpa using admitted
    actionSound := by simp
    actionComplete := by intro candidate admitted; simpa using admitted
    outcomeSound := by simp
    outcomeComplete := by intro candidate admitted; simpa using admitted
    observationSound := by simp
    observationComplete := by intro candidate admitted; simpa using admitted
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
  }
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
  actions := [forceCloseAction]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨openState, closeResult, ⟨rfl, rfl, rfl⟩⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.2.1]
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

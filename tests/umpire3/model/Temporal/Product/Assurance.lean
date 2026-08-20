import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.Assurance

structure State where
  workflowTaskExists : Bool
  speculativeTask : Bool
  nexusOpen : Bool
  nexusTerminal : Bool
  nexusActivityForwardLink : Bool
  nexusActivityBackwardLink : Bool
  nexusTimedOut : Bool
  callbackRegistered : Bool
  callbackReferenceValid : Bool
  callbackResponseRecorded : Bool
  workflowTaskPending : Bool
  workerAvailable : Bool
  entityProgressed : Bool
  continuationLineageValid : Bool
  resetLineageValid : Bool
  workflowRoutingIsolated : Bool
  workflowOwnerFenced : Bool
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | createSpeculativeTask
  | commitSpeculativeTask
  | closeNexusOperation
  | linkNexusActivity
  | timeoutNexusOperation
  | registerCallback
  | recordCallbackResponse
  | dispatchWorkflowTask
  | progressEntity
  | continueWorkflow
  | resetWorkflow
  | routeWorkflowTask
  | fenceWorkflowOwner
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  workflowTaskExists := false
  speculativeTask := false
  nexusOpen := true
  nexusTerminal := false
  nexusActivityForwardLink := false
  nexusActivityBackwardLink := false
  nexusTimedOut := false
  callbackRegistered := false
  callbackReferenceValid := false
  callbackResponseRecorded := false
  workflowTaskPending := false
  workerAvailable := true
  entityProgressed := true
  continuationLineageValid := true
  resetLineageValid := true
  workflowRoutingIsolated := true
  workflowOwnerFenced := true

def next (state : State) : Action → List State
  | .createSpeculativeTask => [{ state with
      workflowTaskExists := true
      speculativeTask := true
      workflowTaskPending := true
      entityProgressed := true }]
  | .commitSpeculativeTask => [{ state with
      workflowTaskExists := true
      speculativeTask := false
      entityProgressed := true }]
  | .closeNexusOperation => [{ state with
      nexusOpen := false
      nexusTerminal := true
      entityProgressed := true }]
  | .linkNexusActivity => [{ state with
      nexusActivityForwardLink := true
      nexusActivityBackwardLink := true
      entityProgressed := true }]
  | .timeoutNexusOperation => [{ state with
      nexusOpen := false
      nexusTerminal := true
      nexusTimedOut := true
      entityProgressed := true }]
  | .registerCallback => [{ state with
      callbackRegistered := true
      callbackReferenceValid := true
      entityProgressed := true }]
  | .recordCallbackResponse => [{ state with
      callbackRegistered := true
      callbackReferenceValid := true
      callbackResponseRecorded := true
      entityProgressed := true }]
  | .dispatchWorkflowTask => [{ state with
      workflowTaskExists := true
      workflowTaskPending := false
      entityProgressed := true }]
  | .progressEntity => [{ state with entityProgressed := true }]
  | .continueWorkflow => [{ state with
      continuationLineageValid := true
      entityProgressed := true }]
  | .resetWorkflow => [{ state with
      resetLineageValid := true
      entityProgressed := true }]
  | .routeWorkflowTask => [{ state with
      workflowRoutingIsolated := true
      entityProgressed := true }]
  | .fenceWorkflowOwner => [{ state with
      workflowOwnerFenced := true
      entityProgressed := true }]

abbrev model : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := fun state action nextState => nextState ∈ next state action

def executable : ExecutableModel model where
  next := next
  next_iff := by intros; rfl

def bounded : BoundedModel model where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := [.createSpeculativeTask, .commitSpeculativeTask, .closeNexusOperation,
    .linkNexusActivity, .timeoutNexusOperation, .registerCallback,
    .recordCallbackResponse, .dispatchWorkflowTask, .progressEntity,
    .continueWorkflow, .resetWorkflow, .routeWorkflowTask, .fenceWorkflowOwner]
  action_complete := by
    intro state action nextState step
    cases action <;> simp

def SpeculativeTaskCreation (state : State) : Prop :=
  state.speculativeTask = true → state.workflowTaskExists = true

def NexusOperationClosure (state : State) : Prop :=
  state.nexusOpen = false → state.nexusTerminal = true

def NexusActivityLinkConsistency (state : State) : Prop :=
  state.nexusActivityForwardLink = state.nexusActivityBackwardLink

def NexusOperationTimeoutSemantics (state : State) : Prop :=
  state.nexusTimedOut = true → state.nexusTerminal = true

def CallbackReferenceConsistency (state : State) : Prop :=
  state.callbackRegistered = true → state.callbackReferenceValid = true

def CallbackResponseConsistency (state : State) : Prop :=
  state.callbackResponseRecorded = true → state.callbackReferenceValid = true

def WorkflowTaskStarvation (state : State) : Prop :=
  state.workflowTaskPending = true → state.workerAvailable = true → state.entityProgressed = true

def EntityProgress (state : State) : Prop := state.entityProgressed = true

def ContinuationLineage (state : State) : Prop := state.continuationLineageValid = true

def ResetLineage (state : State) : Prop := state.resetLineageValid = true

def WorkflowRoutingIsolation (state : State) : Prop := state.workflowRoutingIsolated = true

def WorkflowOwnershipFencing (state : State) : Prop := state.workflowOwnerFenced = true

def AllProperties (state : State) : Prop :=
  SpeculativeTaskCreation state ∧ NexusOperationClosure state ∧
    NexusActivityLinkConsistency state ∧ NexusOperationTimeoutSemantics state ∧
    CallbackReferenceConsistency state ∧ CallbackResponseConsistency state ∧
    WorkflowTaskStarvation state ∧ EntityProgress state ∧
    ContinuationLineage state ∧ ResetLineage state ∧
    WorkflowRoutingIsolation state ∧ WorkflowOwnershipFencing state

theorem initialProperties : AllProperties initial := by
  simp [AllProperties, SpeculativeTaskCreation, NexusOperationClosure,
    NexusActivityLinkConsistency, NexusOperationTimeoutSemantics,
    CallbackReferenceConsistency, CallbackResponseConsistency,
    WorkflowTaskStarvation, EntityProgress, ContinuationLineage, ResetLineage,
    WorkflowRoutingIsolation, WorkflowOwnershipFencing, initial]

theorem stepPreservesProperties {state action nextState}
    (properties : AllProperties state) (step : model.Step state action nextState) :
    AllProperties nextState := by
  cases action <;> simp [next] at step <;> subst nextState <;>
    simp_all [AllProperties, SpeculativeTaskCreation, NexusOperationClosure,
      NexusActivityLinkConsistency, NexusOperationTimeoutSemantics,
      CallbackReferenceConsistency, CallbackResponseConsistency,
      WorkflowTaskStarvation, EntityProgress, ContinuationLineage, ResetLineage,
      WorkflowRoutingIsolation, WorkflowOwnershipFencing]

theorem runsPreserveProperties {start actions final}
    (run : Runs model start actions final) (properties : AllProperties start) :
    AllProperties final := by
  induction run with
  | nil => exact properties
  | cons step _ induction => exact induction (stepPreservesProperties properties step)

theorem allPropertiesSafe : Safety model AllProperties := by
  intro state reachable
  rcases reachable with ⟨start, actions, initialState, run⟩
  subst start
  exact runsPreserveProperties run initialProperties

def invalidSpeculative : State := { initial with speculativeTask := true }
def invalidClosure : State := { initial with nexusOpen := false }
def invalidLink : State := { initial with nexusActivityForwardLink := true }
def invalidTimeout : State := { initial with nexusTimedOut := true }
def invalidCallbackReference : State := { initial with callbackRegistered := true }
def invalidCallbackResponse : State := { initial with callbackResponseRecorded := true }
def invalidStarvation : State := { initial with workflowTaskPending := true, entityProgressed := false }
def invalidProgress : State := { initial with entityProgressed := false }
def invalidContinuationLineage : State := { initial with continuationLineageValid := false }
def invalidResetLineage : State := { initial with resetLineageValid := false }
def invalidRoutingIsolation : State := { initial with workflowRoutingIsolated := false }
def invalidOwnershipFencing : State := { initial with workflowOwnerFenced := false }

theorem speculativeNegativeControl : ¬SpeculativeTaskCreation invalidSpeculative := by
  simp [SpeculativeTaskCreation, invalidSpeculative, initial]

theorem closureNegativeControl : ¬NexusOperationClosure invalidClosure := by
  simp [NexusOperationClosure, invalidClosure, initial]

theorem linkNegativeControl : ¬NexusActivityLinkConsistency invalidLink := by
  simp [NexusActivityLinkConsistency, invalidLink, initial]

theorem timeoutNegativeControl : ¬NexusOperationTimeoutSemantics invalidTimeout := by
  simp [NexusOperationTimeoutSemantics, invalidTimeout, initial]

theorem callbackReferenceNegativeControl :
    ¬CallbackReferenceConsistency invalidCallbackReference := by
  simp [CallbackReferenceConsistency, invalidCallbackReference, initial]

theorem callbackResponseNegativeControl :
    ¬CallbackResponseConsistency invalidCallbackResponse := by
  simp [CallbackResponseConsistency, invalidCallbackResponse, initial]

theorem starvationNegativeControl : ¬WorkflowTaskStarvation invalidStarvation := by
  simp [WorkflowTaskStarvation, invalidStarvation, initial]

theorem progressNegativeControl : ¬EntityProgress invalidProgress := by
  simp [EntityProgress, invalidProgress, initial]

theorem continuationLineageNegativeControl :
    ¬ContinuationLineage invalidContinuationLineage := by
  simp [ContinuationLineage, invalidContinuationLineage, initial]

theorem resetLineageNegativeControl : ¬ResetLineage invalidResetLineage := by
  simp [ResetLineage, invalidResetLineage, initial]

theorem routingIsolationNegativeControl :
    ¬WorkflowRoutingIsolation invalidRoutingIsolation := by
  simp [WorkflowRoutingIsolation, invalidRoutingIsolation, initial]

theorem ownershipFencingNegativeControl :
    ¬WorkflowOwnershipFencing invalidOwnershipFencing := by
  simp [WorkflowOwnershipFencing, invalidOwnershipFencing, initial]

theorem allNegativeControls :
    ¬SpeculativeTaskCreation invalidSpeculative ∧
    ¬NexusOperationClosure invalidClosure ∧
    ¬NexusActivityLinkConsistency invalidLink ∧
    ¬NexusOperationTimeoutSemantics invalidTimeout ∧
    ¬CallbackReferenceConsistency invalidCallbackReference ∧
    ¬CallbackResponseConsistency invalidCallbackResponse ∧
    ¬WorkflowTaskStarvation invalidStarvation ∧
    ¬EntityProgress invalidProgress ∧
    ¬ContinuationLineage invalidContinuationLineage ∧
    ¬ResetLineage invalidResetLineage ∧
    ¬WorkflowRoutingIsolation invalidRoutingIsolation ∧
    ¬WorkflowOwnershipFencing invalidOwnershipFencing :=
  ⟨speculativeNegativeControl, closureNegativeControl, linkNegativeControl,
    timeoutNegativeControl, callbackReferenceNegativeControl,
    callbackResponseNegativeControl, starvationNegativeControl, progressNegativeControl,
    continuationLineageNegativeControl, resetLineageNegativeControl,
    routingIsolationNegativeControl, ownershipFencingNegativeControl⟩

end Umpire3.Temporal.Product.Assurance

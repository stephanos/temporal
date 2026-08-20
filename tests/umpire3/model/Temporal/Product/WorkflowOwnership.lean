import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.WorkflowOwnership

inductive TaskID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive AttemptID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive Epoch where
  | first
  | second
  deriving BEq, DecidableEq, Inhabited, Repr

inductive AttemptState where
  | unused
  | dispatched
  | failed
  | rejected
  | completed
  | invalidCompletion
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  primaryTaskEpoch : Option Epoch
  secondaryTaskEpoch : Option Epoch
  primaryAttemptTask : Option TaskID
  secondaryAttemptTask : Option TaskID
  primaryAttemptEpoch : Option Epoch
  secondaryAttemptEpoch : Option Epoch
  primaryAttemptState : AttemptState
  secondaryAttemptState : AttemptState
  deriving DecidableEq, Inhabited, Repr

def State.taskEpoch (state : State) : TaskID → Option Epoch
  | .primary => state.primaryTaskEpoch
  | .secondary => state.secondaryTaskEpoch

def State.attemptTask (state : State) : AttemptID → Option TaskID
  | .primary => state.primaryAttemptTask
  | .secondary => state.secondaryAttemptTask

def State.attemptEpoch (state : State) : AttemptID → Option Epoch
  | .primary => state.primaryAttemptEpoch
  | .secondary => state.secondaryAttemptEpoch

def State.attemptState (state : State) : AttemptID → AttemptState
  | .primary => state.primaryAttemptState
  | .secondary => state.secondaryAttemptState

def State.setTaskEpoch (state : State) : TaskID → Epoch → State
  | .primary, epoch => { state with primaryTaskEpoch := some epoch }
  | .secondary, epoch => { state with secondaryTaskEpoch := some epoch }

def State.setAttempt (state : State) : AttemptID → TaskID → Epoch → AttemptState → State
  | .primary, task, epoch, attemptState => {
      state with
      primaryAttemptTask := some task
      primaryAttemptEpoch := some epoch
      primaryAttemptState := attemptState
    }
  | .secondary, task, epoch, attemptState => {
      state with
      secondaryAttemptTask := some task
      secondaryAttemptEpoch := some epoch
      secondaryAttemptState := attemptState
    }

def State.setAttemptState (state : State) : AttemptID → AttemptState → State
  | .primary, attemptState => { state with primaryAttemptState := attemptState }
  | .secondary, attemptState => { state with secondaryAttemptState := attemptState }

def taskIDs : List TaskID := [.primary, .secondary]
def attemptIDs : List AttemptID := [.primary, .secondary]
def epochs : List Epoch := [.first, .second]

theorem task_mem (task : TaskID) : task ∈ taskIDs := by cases task <;> simp [taskIDs]
theorem attempt_mem (attempt : AttemptID) : attempt ∈ attemptIDs := by cases attempt <;> simp [attemptIDs]
theorem epoch_mem (epoch : Epoch) : epoch ∈ epochs := by cases epoch <;> simp [epochs]

def attemptFencedB (state : State) (attempt : AttemptID) : Bool :=
  state.attemptState attempt != .invalidCompletion &&
    if state.attemptState attempt != .dispatched then true
    else match state.attemptTask attempt, state.attemptEpoch attempt with
      | some task, some epoch => state.taskEpoch task == some epoch
      | _, _ => false

def ownershipFencingB (state : State) : Bool := attemptIDs.all (attemptFencedB state)
def ownershipReadyB (state : State) : Bool :=
  attemptIDs.any (fun attempt => state.attemptState attempt == .rejected) &&
    attemptIDs.any (fun attempt => state.attemptState attempt == .completed)

def OwnershipFencing (state : State) : Prop := ownershipFencingB state = true
def OwnershipQualified (state : State) : Prop :=
  ownershipReadyB state = true ∧ OwnershipFencing state

instance (state : State) : Decidable (OwnershipFencing state) := by
  unfold OwnershipFencing
  infer_instance

instance (state : State) : Decidable (OwnershipQualified state) := by
  unfold OwnershipQualified
  infer_instance

inductive Action where
  | bootstrap (task : TaskID) (epoch : Epoch)
  | dispatch (attempt : AttemptID) (task : TaskID) (epoch : Epoch)
  | fail (attempt : AttemptID)
  | rotate (task : TaskID) (previous : Epoch) (current : Epoch)
  | rejectStale (attempt : AttemptID)
  | complete (attempt : AttemptID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryTaskEpoch := none
  secondaryTaskEpoch := none
  primaryAttemptTask := none
  secondaryAttemptTask := none
  primaryAttemptEpoch := none
  secondaryAttemptEpoch := none
  primaryAttemptState := .unused
  secondaryAttemptState := .unused

def rawNext (state : State) : Action → List State
  | .bootstrap task epoch =>
      if (state.taskEpoch task).isSome then [] else [state.setTaskEpoch task epoch]
  | .dispatch attempt task epoch =>
      if state.attemptState attempt != .unused || state.taskEpoch task != some epoch then []
      else [state.setAttempt attempt task epoch .dispatched]
  | .fail attempt =>
      if state.attemptState attempt != .dispatched then []
      else [state.setAttemptState attempt .failed]
  | .rotate task previous current =>
      if previous == current || state.taskEpoch task != some previous then []
      else [state.setTaskEpoch task current]
  | .rejectStale attempt =>
      match state.attemptTask attempt, state.attemptEpoch attempt with
        | some task, some epoch =>
            if state.attemptState attempt == .failed && state.taskEpoch task != some epoch then
              [state.setAttemptState attempt .rejected]
            else []
        | _, _ => []
  | .complete attempt =>
      match state.attemptTask attempt, state.attemptEpoch attempt with
        | some task, some epoch =>
            if state.attemptState attempt == .dispatched && state.taskEpoch task == some epoch then
              [state.setAttemptState attempt .completed]
            else if state.attemptState attempt == .failed && state.taskEpoch task != some epoch then
              [state.setAttemptState attempt .invalidCompletion]
            else []
        | _, _ => []

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter ownershipFencingB

def step (state : State) (action : Action) (nextState : State) : Prop :=
  nextState ∈ next state action

abbrev model : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := step

def executable : ExecutableModel model where
  next := next
  next_iff := by intros; rfl

def bootstrapActions : List Action := taskIDs.flatMap fun task =>
  epochs.map fun epoch => .bootstrap task epoch

def dispatchActions : List Action := attemptIDs.flatMap fun attempt =>
  taskIDs.flatMap fun task => epochs.map fun epoch => .dispatch attempt task epoch

def failActions : List Action := attemptIDs.map Action.fail

def rotateActions : List Action := taskIDs.flatMap fun task => epochs.flatMap fun previous =>
  epochs.map fun current => .rotate task previous current

def rejectActions : List Action := attemptIDs.map Action.rejectStale
def completeActions : List Action := attemptIDs.map Action.complete

def actions : List Action := bootstrapActions ++ dispatchActions ++ failActions ++ rotateActions ++
  rejectActions ++ completeActions

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | bootstrap task epoch =>
      simp only [actions, List.mem_append]
      left; left; left; left; left
      apply List.mem_flatMap.mpr
      exact ⟨task, task_mem task, List.mem_map.mpr ⟨epoch, epoch_mem epoch, rfl⟩⟩
  | dispatch attempt task epoch =>
      simp only [actions, List.mem_append]
      left; left; left; left; right
      apply List.mem_flatMap.mpr
      refine ⟨attempt, attempt_mem attempt, ?_⟩
      apply List.mem_flatMap.mpr
      exact ⟨task, task_mem task, List.mem_map.mpr ⟨epoch, epoch_mem epoch, rfl⟩⟩
  | fail attempt =>
      simp only [actions, List.mem_append]
      left; left; left; right
      exact List.mem_map.mpr ⟨attempt, attempt_mem attempt, rfl⟩
  | rotate task previous current =>
      simp only [actions, List.mem_append]
      left; left; right
      apply List.mem_flatMap.mpr
      refine ⟨task, task_mem task, ?_⟩
      apply List.mem_flatMap.mpr
      exact ⟨previous, epoch_mem previous, List.mem_map.mpr ⟨current, epoch_mem current, rfl⟩⟩
  | rejectStale attempt =>
      simp only [actions, List.mem_append]
      left; right
      exact List.mem_map.mpr ⟨attempt, attempt_mem attempt, rfl⟩
  | complete attempt =>
      simp only [actions, List.mem_append]
      right
      exact List.mem_map.mpr ⟨attempt, attempt_mem attempt, rfl⟩

def bounded : BoundedModel model where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    exact action_mem action

abbrev weakenedModel : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := fun state action nextState => nextState ∈ rawNext state action

def weakenedExecutable : ExecutableModel weakenedModel where
  next := rawNext
  next_iff := by intros; rfl

def beforeRotation : State :=
  ((initial.setTaskEpoch .primary .first).setAttempt .primary .primary .first .failed).setTaskEpoch
    .primary .second

def fencedFinal : State :=
  ((beforeRotation.setAttempt .secondary .primary .second .completed).setAttemptState
    .primary .rejected)

def staleCompletionFinal : State := beforeRotation.setAttemptState .primary .invalidCompletion

theorem initialOwnershipFencing : OwnershipFencing initial := by decide

theorem successorOwnershipFencing {state action nextState}
    (transition : model.Step state action nextState) : OwnershipFencing nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveOwnershipFencing {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : OwnershipFencing start) : OwnershipFencing final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorOwnershipFencing transition)

theorem ownershipFencingSafe : Safety model OwnershipFencing := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveOwnershipFencing run initialOwnershipFencing

theorem staleCompletionMutationNegativeControl : ¬OwnershipFencing staleCompletionFinal := by decide

end Umpire3.Temporal.Product.WorkflowOwnership

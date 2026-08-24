import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Feature.SpeculativeTask

inductive TaskID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive UpdateID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive UpdateState where
  | absent
  | pending
  | admitted
  deriving BEq, DecidableEq, Inhabited, Repr

inductive TaskState where
  | absent
  | speculative
  | committed
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  primaryUpdateState : UpdateState
  secondaryUpdateState : UpdateState
  primaryTaskState : TaskState
  secondaryTaskState : TaskState
  primaryTaskUpdate : Option UpdateID
  secondaryTaskUpdate : Option UpdateID
  deriving DecidableEq, Inhabited, Repr

def State.updateState (state : State) : UpdateID → UpdateState
  | .primary => state.primaryUpdateState
  | .secondary => state.secondaryUpdateState

def State.taskState (state : State) : TaskID → TaskState
  | .primary => state.primaryTaskState
  | .secondary => state.secondaryTaskState

def State.taskUpdate (state : State) : TaskID → Option UpdateID
  | .primary => state.primaryTaskUpdate
  | .secondary => state.secondaryTaskUpdate

def State.setUpdateState (state : State) : UpdateID → UpdateState → State
  | .primary, updateState => { state with primaryUpdateState := updateState }
  | .secondary, updateState => { state with secondaryUpdateState := updateState }

def State.setTask (state : State) : TaskID → UpdateID → TaskState → State
  | .primary, update, taskState => {
      state with primaryTaskState := taskState, primaryTaskUpdate := some update
    }
  | .secondary, update, taskState => {
      state with secondaryTaskState := taskState, secondaryTaskUpdate := some update
    }

def State.setTaskState (state : State) : TaskID → TaskState → State
  | .primary, taskState => { state with primaryTaskState := taskState }
  | .secondary, taskState => { state with secondaryTaskState := taskState }

def taskIDs : List TaskID := [.primary, .secondary]
def updateIDs : List UpdateID := [.primary, .secondary]

theorem task_mem (task : TaskID) : task ∈ taskIDs := by cases task <;> simp [taskIDs]
theorem update_mem (update : UpdateID) : update ∈ updateIDs := by cases update <;> simp [updateIDs]

def taskCreationValidB (state : State) (task : TaskID) : Bool :=
  match state.taskState task, state.taskUpdate task with
  | .absent, none => true
  | .speculative, some update => state.updateState update == .pending
  | .committed, some update => state.updateState update == .admitted
  | _, _ => false

def speculativeCreationB (state : State) : Bool := taskIDs.all (taskCreationValidB state)
def speculativeReadyB (state : State) : Bool :=
  taskIDs.any fun task => state.taskState task == .committed

def SpeculativeTaskCreation (state : State) : Prop := speculativeCreationB state = true
def SpeculativeQualified (state : State) : Prop :=
  speculativeReadyB state = true ∧ SpeculativeTaskCreation state

instance (state : State) : Decidable (SpeculativeTaskCreation state) := by
  unfold SpeculativeTaskCreation
  infer_instance

instance (state : State) : Decidable (SpeculativeQualified state) := by
  unfold SpeculativeQualified
  infer_instance

inductive Action where
  | requestUpdate (update : UpdateID)
  | create (task : TaskID) (update : UpdateID)
  | commit (task : TaskID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryUpdateState := .absent
  secondaryUpdateState := .absent
  primaryTaskState := .absent
  secondaryTaskState := .absent
  primaryTaskUpdate := none
  secondaryTaskUpdate := none

def rawNext (state : State) : Action → List State
  | .requestUpdate update =>
      if state.updateState update != .absent then []
      else [state.setUpdateState update .pending]
  | .create task update =>
      if state.taskState task != .absent then []
      else [state.setTask task update .speculative]
  | .commit task =>
      match state.taskUpdate task with
      | some update =>
          if state.taskState task == .speculative && state.updateState update == .pending then
            [(state.setUpdateState update .admitted).setTaskState task .committed]
          else []
      | none => []

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter speculativeCreationB

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

def requestActions : List Action := updateIDs.map Action.requestUpdate
def createActions : List Action := taskIDs.flatMap fun task => updateIDs.map fun update => .create task update
def commitActions : List Action := taskIDs.map Action.commit
def actions : List Action := requestActions ++ createActions ++ commitActions

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | requestUpdate update =>
      apply List.mem_append_left
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨update, update_mem update, rfl⟩
  | create task update =>
      apply List.mem_append_left
      apply List.mem_append_right
      apply List.mem_flatMap.mpr
      exact ⟨task, task_mem task, List.mem_map.mpr ⟨update, update_mem update, rfl⟩⟩
  | commit task =>
      apply List.mem_append_right
      exact List.mem_map.mpr ⟨task, task_mem task, rfl⟩

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

def requested : State := initial.setUpdateState .primary .pending
def committedFinal : State :=
  (requested.setUpdateState .primary .admitted).setTask .primary .primary .committed
def orphanedFinal : State := initial.setTask .primary .secondary .speculative

theorem initialSpeculativeCreation : SpeculativeTaskCreation initial := by decide

theorem successorSpeculativeCreation {state action nextState}
    (transition : model.Step state action nextState) : SpeculativeTaskCreation nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveSpeculativeCreation {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : SpeculativeTaskCreation start) : SpeculativeTaskCreation final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorSpeculativeCreation transition)

theorem speculativeCreationSafe : Safety model SpeculativeTaskCreation := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveSpeculativeCreation run initialSpeculativeCreation

theorem orphanedTaskMutationNegativeControl : ¬SpeculativeTaskCreation orphanedFinal := by decide

end Umpire3.Temporal.Feature.SpeculativeTask

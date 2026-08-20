import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.WorkflowProgress

inductive TaskID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive EntityID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive WorkerID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive TaskState where
  | absent
  | queued
  | dispatched
  | completed
  deriving BEq, DecidableEq, Inhabited, Repr

inductive EntityState where
  | idle
  | pending
  | progressed
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  primaryTaskState : TaskState
  secondaryTaskState : TaskState
  primaryTaskEntity : Option EntityID
  secondaryTaskEntity : Option EntityID
  primaryTaskWorker : Option WorkerID
  secondaryTaskWorker : Option WorkerID
  primaryTaskWaitAge : Nat
  secondaryTaskWaitAge : Nat
  primaryEntityState : EntityState
  secondaryEntityState : EntityState
  primaryWorkerAvailable : Bool
  secondaryWorkerAvailable : Bool
  deriving DecidableEq, Inhabited, Repr

def State.taskState (state : State) : TaskID → TaskState
  | .primary => state.primaryTaskState
  | .secondary => state.secondaryTaskState

def State.taskEntity (state : State) : TaskID → Option EntityID
  | .primary => state.primaryTaskEntity
  | .secondary => state.secondaryTaskEntity

def State.taskWorker (state : State) : TaskID → Option WorkerID
  | .primary => state.primaryTaskWorker
  | .secondary => state.secondaryTaskWorker

def State.taskWaitAge (state : State) : TaskID → Nat
  | .primary => state.primaryTaskWaitAge
  | .secondary => state.secondaryTaskWaitAge

def State.entityState (state : State) : EntityID → EntityState
  | .primary => state.primaryEntityState
  | .secondary => state.secondaryEntityState

def State.workerAvailable (state : State) : WorkerID → Bool
  | .primary => state.primaryWorkerAvailable
  | .secondary => state.secondaryWorkerAvailable

def State.setWorkerAvailable (state : State) : WorkerID → State
  | .primary => { state with primaryWorkerAvailable := true }
  | .secondary => { state with secondaryWorkerAvailable := true }

def State.setEntityState (state : State) : EntityID → EntityState → State
  | .primary, entityState => { state with primaryEntityState := entityState }
  | .secondary, entityState => { state with secondaryEntityState := entityState }

def State.enqueue (state : State) : TaskID → EntityID → State
  | .primary, entity =>
      { state with primaryTaskState := .queued, primaryTaskEntity := some entity, primaryTaskWaitAge := 0 }
  | .secondary, entity =>
      { state with secondaryTaskState := .queued, secondaryTaskEntity := some entity, secondaryTaskWaitAge := 0 }

def State.dispatch (state : State) : TaskID → WorkerID → State
  | .primary, worker => {
      state with primaryTaskState := .dispatched, primaryTaskWorker := some worker
    }
  | .secondary, worker => {
      state with secondaryTaskState := .dispatched, secondaryTaskWorker := some worker
    }

def State.incrementWait (state : State) : TaskID → State
  | .primary => { state with primaryTaskWaitAge := state.primaryTaskWaitAge + 1 }
  | .secondary => { state with secondaryTaskWaitAge := state.secondaryTaskWaitAge + 1 }

def State.complete (state : State) : TaskID → EntityID → State
  | .primary, entity => {
      (state.setEntityState entity .progressed) with primaryTaskState := .completed
    }
  | .secondary, entity => {
      (state.setEntityState entity .progressed) with secondaryTaskState := .completed
    }

def taskIDs : List TaskID := [.primary, .secondary]
def entityIDs : List EntityID := [.primary, .secondary]
def workerIDs : List WorkerID := [.primary, .secondary]

theorem task_mem (task : TaskID) : task ∈ taskIDs := by cases task <;> simp [taskIDs]
theorem entity_mem (entity : EntityID) : entity ∈ entityIDs := by cases entity <;> simp [entityIDs]
theorem worker_mem (worker : WorkerID) : worker ∈ workerIDs := by cases worker <;> simp [workerIDs]

def anyWorkerAvailableB (state : State) : Bool := workerIDs.any state.workerAvailable

def taskNotStarvedB (state : State) (task : TaskID) : Bool :=
  if state.taskState task == .queued && anyWorkerAvailableB state then
    state.taskWaitAge task ≤ 1
  else true

def workflowTaskStarvationB (state : State) : Bool := taskIDs.all (taskNotStarvedB state)

def taskProgressValidB (state : State) (task : TaskID) : Bool :=
  if state.taskState task != .completed then true
  else match state.taskEntity task with
    | some entity => state.entityState entity == .progressed
    | none => false

def entityProgressB (state : State) : Bool := taskIDs.all (taskProgressValidB state)
def progressReadyB (state : State) : Bool :=
  entityIDs.any fun entity => state.entityState entity == .progressed
def starvationReadyB (state : State) : Bool :=
  taskIDs.any fun task => state.taskState task == .completed && state.taskWaitAge task > 0

def WorkflowTaskStarvation (state : State) : Prop := workflowTaskStarvationB state = true
def EntityProgress (state : State) : Prop := entityProgressB state = true
def StarvationQualified (state : State) : Prop :=
  starvationReadyB state = true ∧ WorkflowTaskStarvation state
def ProgressQualified (state : State) : Prop :=
  progressReadyB state = true ∧ EntityProgress state

instance (state : State) : Decidable (WorkflowTaskStarvation state) := by
  unfold WorkflowTaskStarvation
  infer_instance

instance (state : State) : Decidable (EntityProgress state) := by
  unfold EntityProgress
  infer_instance

instance (state : State) : Decidable (StarvationQualified state) := by
  unfold StarvationQualified
  infer_instance

instance (state : State) : Decidable (ProgressQualified state) := by
  unfold ProgressQualified
  infer_instance

inductive Action where
  | enqueue (task : TaskID) (entity : EntityID)
  | makeWorkerAvailable (worker : WorkerID)
  | wait (task : TaskID)
  | dispatch (task : TaskID) (worker : WorkerID)
  | complete (task : TaskID) (entity : EntityID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryTaskState := .absent
  secondaryTaskState := .absent
  primaryTaskEntity := none
  secondaryTaskEntity := none
  primaryTaskWorker := none
  secondaryTaskWorker := none
  primaryTaskWaitAge := 0
  secondaryTaskWaitAge := 0
  primaryEntityState := .idle
  secondaryEntityState := .idle
  primaryWorkerAvailable := false
  secondaryWorkerAvailable := false

def rawNext (state : State) : Action → List State
  | .enqueue task entity =>
      if state.taskState task != .absent || state.entityState entity != .idle then []
      else [(state.enqueue task entity).setEntityState entity .pending]
  | .makeWorkerAvailable worker =>
      if state.workerAvailable worker then [] else [state.setWorkerAvailable worker]
  | .wait task =>
      if state.taskState task != .queued || !anyWorkerAvailableB state then []
      else [state.incrementWait task]
  | .dispatch task worker =>
      if state.taskState task != .queued || !state.workerAvailable worker then []
      else [state.dispatch task worker]
  | .complete task entity =>
      if state.taskState task != .dispatched then [] else [state.complete task entity]

def invariantB (state : State) : Bool := workflowTaskStarvationB state && entityProgressB state
def next (state : State) (action : Action) : List State := (rawNext state action).filter invariantB

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

def enqueueActions : List Action := taskIDs.flatMap fun task => entityIDs.map fun entity => .enqueue task entity
def workerActions : List Action := workerIDs.map Action.makeWorkerAvailable
def waitActions : List Action := taskIDs.map Action.wait
def dispatchActions : List Action := taskIDs.flatMap fun task => workerIDs.map fun worker => .dispatch task worker
def completeActions : List Action := taskIDs.flatMap fun task => entityIDs.map fun entity => .complete task entity
def actions : List Action := enqueueActions ++ workerActions ++ waitActions ++ dispatchActions ++ completeActions

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | enqueue task entity =>
      simp only [actions, List.mem_append]
      left; left; left; left
      apply List.mem_flatMap.mpr
      exact ⟨task, task_mem task, List.mem_map.mpr ⟨entity, entity_mem entity, rfl⟩⟩
  | makeWorkerAvailable worker =>
      simp only [actions, List.mem_append]
      left; left; left; right
      exact List.mem_map.mpr ⟨worker, worker_mem worker, rfl⟩
  | wait task =>
      simp only [actions, List.mem_append]
      left; left; right
      exact List.mem_map.mpr ⟨task, task_mem task, rfl⟩
  | dispatch task worker =>
      simp only [actions, List.mem_append]
      left; right
      apply List.mem_flatMap.mpr
      exact ⟨task, task_mem task, List.mem_map.mpr ⟨worker, worker_mem worker, rfl⟩⟩
  | complete task entity =>
      simp only [actions, List.mem_append]
      right
      apply List.mem_flatMap.mpr
      exact ⟨task, task_mem task, List.mem_map.mpr ⟨entity, entity_mem entity, rfl⟩⟩

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

def queuedAvailable : State :=
  ((initial.enqueue .primary .primary).setEntityState .primary .pending).setWorkerAvailable .primary
def waited : State := queuedAvailable.incrementWait .primary
def progressedFinal : State :=
  (waited.dispatch .primary .primary).complete .primary .primary
def starvedFinal : State := waited.incrementWait .primary
def wrongEntityFinal : State :=
  (queuedAvailable.dispatch .primary .primary).complete .primary .secondary

theorem initialWorkflowTaskStarvation : WorkflowTaskStarvation initial := by decide
theorem initialEntityProgress : EntityProgress initial := by decide

theorem successorInvariant {state action nextState}
    (transition : model.Step state action nextState) :
    WorkflowTaskStarvation nextState ∧ EntityProgress nextState := by
  have property := (List.mem_filter.mp transition).2
  simpa [invariantB, WorkflowTaskStarvation, EntityProgress] using property

theorem runsPreserveInvariant {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : WorkflowTaskStarvation start ∧ EntityProgress start) :
    WorkflowTaskStarvation final ∧ EntityProgress final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorInvariant transition)

theorem workflowTaskStarvationSafe : Safety model WorkflowTaskStarvation := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact (runsPreserveInvariant run ⟨initialWorkflowTaskStarvation, initialEntityProgress⟩).1

theorem entityProgressSafe : Safety model EntityProgress := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact (runsPreserveInvariant run ⟨initialWorkflowTaskStarvation, initialEntityProgress⟩).2

theorem starvationMutationNegativeControl : ¬WorkflowTaskStarvation starvedFinal := by decide
theorem progressMutationNegativeControl : ¬EntityProgress wrongEntityFinal := by decide

end Umpire3.Temporal.Product.WorkflowProgress

import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.WorkflowRouting

inductive TaskID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive PollerID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive RouteID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive AttemptID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  primaryTaskRoute : Option RouteID
  secondaryTaskRoute : Option RouteID
  primaryPollerRoute : Option RouteID
  secondaryPollerRoute : Option RouteID
  primaryAttemptObserved : Bool
  secondaryAttemptObserved : Bool
  primaryAttemptTask : Option TaskID
  secondaryAttemptTask : Option TaskID
  primaryAttemptPoller : Option PollerID
  secondaryAttemptPoller : Option PollerID
  deriving DecidableEq, Inhabited, Repr

def State.taskRoute (state : State) : TaskID → Option RouteID
  | .primary => state.primaryTaskRoute
  | .secondary => state.secondaryTaskRoute

def State.pollerRoute (state : State) : PollerID → Option RouteID
  | .primary => state.primaryPollerRoute
  | .secondary => state.secondaryPollerRoute

def State.attemptObserved (state : State) : AttemptID → Bool
  | .primary => state.primaryAttemptObserved
  | .secondary => state.secondaryAttemptObserved

def State.attemptTask (state : State) : AttemptID → Option TaskID
  | .primary => state.primaryAttemptTask
  | .secondary => state.secondaryAttemptTask

def State.attemptPoller (state : State) : AttemptID → Option PollerID
  | .primary => state.primaryAttemptPoller
  | .secondary => state.secondaryAttemptPoller

def State.setTaskRoute (state : State) : TaskID → RouteID → State
  | .primary, route => { state with primaryTaskRoute := some route }
  | .secondary, route => { state with secondaryTaskRoute := some route }

def State.setPollerRoute (state : State) : PollerID → RouteID → State
  | .primary, route => { state with primaryPollerRoute := some route }
  | .secondary, route => { state with secondaryPollerRoute := some route }

def State.setAttempt (state : State) : AttemptID → TaskID → PollerID → State
  | .primary, task, poller => {
      state with
      primaryAttemptObserved := true
      primaryAttemptTask := some task
      primaryAttemptPoller := some poller
    }
  | .secondary, task, poller => {
      state with
      secondaryAttemptObserved := true
      secondaryAttemptTask := some task
      secondaryAttemptPoller := some poller
    }

def taskIDs : List TaskID := [.primary, .secondary]
def pollerIDs : List PollerID := [.primary, .secondary]
def routeIDs : List RouteID := [.primary, .secondary]
def attemptIDs : List AttemptID := [.primary, .secondary]

theorem task_mem (task : TaskID) : task ∈ taskIDs := by cases task <;> simp [taskIDs]
theorem poller_mem (poller : PollerID) : poller ∈ pollerIDs := by cases poller <;> simp [pollerIDs]
theorem route_mem (route : RouteID) : route ∈ routeIDs := by cases route <;> simp [routeIDs]
theorem attempt_mem (attempt : AttemptID) : attempt ∈ attemptIDs := by cases attempt <;> simp [attemptIDs]

def routingConsistentForB (state : State) (attempt : AttemptID) : Bool :=
  if !state.attemptObserved attempt then true
  else match state.attemptTask attempt, state.attemptPoller attempt with
    | some task, some poller => match state.taskRoute task, state.pollerRoute poller with
      | some taskRoute, some pollerRoute => taskRoute == pollerRoute
      | _, _ => false
    | _, _ => false

def routingIsolationB (state : State) : Bool := attemptIDs.all (routingConsistentForB state)
def routingReadyB (state : State) : Bool := attemptIDs.any (state.attemptObserved ·)

def RoutingIsolation (state : State) : Prop := routingIsolationB state = true
def RoutingQualified (state : State) : Prop := routingReadyB state = true ∧ RoutingIsolation state

instance (state : State) : Decidable (RoutingIsolation state) := by
  unfold RoutingIsolation
  infer_instance

instance (state : State) : Decidable (RoutingQualified state) := by
  unfold RoutingQualified
  infer_instance

inductive Action where
  | assignTask (task : TaskID) (route : RouteID)
  | registerPoller (poller : PollerID) (route : RouteID)
  | reserve (attempt : AttemptID) (task : TaskID) (poller : PollerID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryTaskRoute := none
  secondaryTaskRoute := none
  primaryPollerRoute := none
  secondaryPollerRoute := none
  primaryAttemptObserved := false
  secondaryAttemptObserved := false
  primaryAttemptTask := none
  secondaryAttemptTask := none
  primaryAttemptPoller := none
  secondaryAttemptPoller := none

def rawNext (state : State) : Action → List State
  | .assignTask task route =>
      if (state.taskRoute task).isSome then [] else [state.setTaskRoute task route]
  | .registerPoller poller route =>
      if (state.pollerRoute poller).isSome then [] else [state.setPollerRoute poller route]
  | .reserve attempt task poller =>
      if state.attemptObserved attempt || (state.taskRoute task).isNone ||
          (state.pollerRoute poller).isNone then []
      else [state.setAttempt attempt task poller]

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter routingIsolationB

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

def taskActions : List Action := taskIDs.flatMap fun task =>
  routeIDs.map fun route => .assignTask task route

def pollerActions : List Action := pollerIDs.flatMap fun poller =>
  routeIDs.map fun route => .registerPoller poller route

def attemptActions : List Action := attemptIDs.flatMap fun attempt =>
  taskIDs.flatMap fun task => pollerIDs.map fun poller => .reserve attempt task poller

def actions : List Action := taskActions ++ pollerActions ++ attemptActions

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | assignTask task route =>
      apply List.mem_append_left
      apply List.mem_append_left
      apply List.mem_flatMap.mpr
      refine ⟨task, task_mem task, ?_⟩
      exact List.mem_map.mpr ⟨route, route_mem route, rfl⟩
  | registerPoller poller route =>
      apply List.mem_append_left
      apply List.mem_append_right
      apply List.mem_flatMap.mpr
      refine ⟨poller, poller_mem poller, ?_⟩
      exact List.mem_map.mpr ⟨route, route_mem route, rfl⟩
  | reserve attempt task poller =>
      apply List.mem_append_right
      apply List.mem_flatMap.mpr
      refine ⟨attempt, attempt_mem attempt, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨task, task_mem task, ?_⟩
      exact List.mem_map.mpr ⟨poller, poller_mem poller, rfl⟩

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

def matchingFinal : State :=
  ((initial.setTaskRoute .primary .primary).setPollerRoute .primary .primary).setAttempt
    .primary .primary .primary

def crossingFinal : State :=
  ((initial.setTaskRoute .primary .primary).setPollerRoute .primary .secondary).setAttempt
    .primary .primary .primary

theorem initialRoutingIsolation : RoutingIsolation initial := by decide

theorem successorRoutingIsolation {state action nextState}
    (transition : model.Step state action nextState) : RoutingIsolation nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveRoutingIsolation {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : RoutingIsolation start) : RoutingIsolation final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorRoutingIsolation transition)

theorem routingIsolationSafe : Safety model RoutingIsolation := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveRoutingIsolation run initialRoutingIsolation

theorem crossingRouteMutationNegativeControl : ¬RoutingIsolation crossingFinal := by decide

end Umpire3.Temporal.Product.WorkflowRouting

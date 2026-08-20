import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.NexusActivityLink

inductive OperationID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive ActivityID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  primaryOperationObserved : Bool
  secondaryOperationObserved : Bool
  primaryActivityObserved : Bool
  secondaryActivityObserved : Bool
  primaryForward : Option ActivityID
  secondaryForward : Option ActivityID
  primaryReverse : Option OperationID
  secondaryReverse : Option OperationID
  deriving DecidableEq, Inhabited, Repr

def State.operationObserved (state : State) : OperationID → Bool
  | .primary => state.primaryOperationObserved
  | .secondary => state.secondaryOperationObserved

def State.setOperationObserved (state : State) : OperationID → Bool → State
  | .primary, observed => { state with primaryOperationObserved := observed }
  | .secondary, observed => { state with secondaryOperationObserved := observed }

def State.activityObserved (state : State) : ActivityID → Bool
  | .primary => state.primaryActivityObserved
  | .secondary => state.secondaryActivityObserved

def State.setActivityObserved (state : State) : ActivityID → Bool → State
  | .primary, observed => { state with primaryActivityObserved := observed }
  | .secondary, observed => { state with secondaryActivityObserved := observed }

def State.forward (state : State) : OperationID → Option ActivityID
  | .primary => state.primaryForward
  | .secondary => state.secondaryForward

def State.setForward (state : State) : OperationID → Option ActivityID → State
  | .primary, activity => { state with primaryForward := activity }
  | .secondary, activity => { state with secondaryForward := activity }

def State.reverse (state : State) : ActivityID → Option OperationID
  | .primary => state.primaryReverse
  | .secondary => state.secondaryReverse

def State.setReverse (state : State) : ActivityID → Option OperationID → State
  | .primary, operation => { state with primaryReverse := operation }
  | .secondary, operation => { state with secondaryReverse := operation }

def operationIDs : List OperationID := [.primary, .secondary]

def activityIDs : List ActivityID := [.primary, .secondary]

def linkConsistencyB (state : State) : Bool :=
  operationIDs.all fun operation =>
    activityIDs.all fun activity =>
      !(state.operationObserved operation && state.activityObserved activity) ||
        ((state.forward operation == some activity) == (state.reverse activity == some operation))

def LinkConsistency (state : State) : Prop := linkConsistencyB state = true

instance (state : State) : Decidable (LinkConsistency state) := by
  unfold LinkConsistency
  infer_instance

inductive Action where
  | observeOperation (operation : OperationID) (activity : Option ActivityID)
  | observeActivity (activity : ActivityID) (operation : Option OperationID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryOperationObserved := false
  secondaryOperationObserved := false
  primaryActivityObserved := false
  secondaryActivityObserved := false
  primaryForward := none
  secondaryForward := none
  primaryReverse := none
  secondaryReverse := none

def rawNext (state : State) : Action → List State
  | .observeOperation operation activity =>
      if state.operationObserved operation then []
      else [(state.setOperationObserved operation true).setForward operation activity]
  | .observeActivity activity operation =>
      if state.activityObserved activity then []
      else [(state.setActivityObserved activity true).setReverse activity operation]

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter linkConsistencyB

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

def activities : List (Option ActivityID) := [none, some .primary, some .secondary]

def operations : List (Option OperationID) := [none, some .primary, some .secondary]

def actions : List Action :=
  (operationIDs.flatMap fun operation =>
    activities.map fun activity => Action.observeOperation operation activity) ++
  (activityIDs.flatMap fun activity =>
    operations.map fun operation => Action.observeActivity activity operation)

def bounded : BoundedModel model where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    cases action with
    | observeOperation operation activity =>
        cases operation <;> cases activity with
        | none => simp [actions, operationIDs, activities]
        | some activity => cases activity <;> simp [actions, operationIDs, activities]
    | observeActivity activity operation =>
        cases activity <;> cases operation with
        | none => simp [actions, activityIDs, operations]
        | some operation => cases operation <;> simp [actions, activityIDs, operations]

abbrev unsafeModel : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := fun state action nextState => nextState ∈ rawNext state action

def unsafeExecutable : ExecutableModel unsafeModel where
  next := rawNext
  next_iff := by intros; rfl

def matchingFinal : State := {
  initial with
  primaryOperationObserved := true
  primaryActivityObserved := true
  primaryForward := some .primary
  primaryReverse := some .primary
}

def oneSidedFinal : State := {
  initial with
  primaryOperationObserved := true
  primaryActivityObserved := true
  primaryForward := some .primary
}

theorem initialLinkConsistency : LinkConsistency initial := by decide

theorem successorLinkConsistency {state action nextState}
    (transition : model.Step state action nextState) : LinkConsistency nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveLinkConsistency {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : LinkConsistency start) : LinkConsistency final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorLinkConsistency transition)

theorem linkConsistencySafe : Safety model LinkConsistency := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveLinkConsistency run initialLinkConsistency

theorem missingReverseMutationNegativeControl : ¬LinkConsistency oneSidedFinal := by decide

end Umpire3.Temporal.Product.NexusActivityLink

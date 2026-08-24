import Umpire3.Transition

namespace Umpire3

structure ExecutableModel (model : TransitionSystem) where
  next : model.State → model.Action → List model.State
  next_iff : ∀ state action nextState,
    nextState ∈ next state action ↔ model.Step state action nextState

structure BoundedModel (model : TransitionSystem) extends ExecutableModel model where
  initials : List model.State
  initial_iff : ∀ state, state ∈ initials ↔ model.Initial state
  actions : List model.Action
  action_complete : ∀ state action nextState,
    model.Step state action nextState → action ∈ actions

def BoundedModel.frontier (model : BoundedModel transition) :
    Nat → List (List transition.Action × transition.State)
  | 0 => model.initials.map fun state => ([], state)
  | depth + 1 =>
      (model.frontier depth).flatMap fun execution =>
        model.actions.flatMap fun action =>
          (model.next execution.2 action).map fun state =>
            (execution.1 ++ [action], state)

structure ExplorationBound where
  maxDepth : Nat
  maxResults : Nat
  deriving DecidableEq, Repr

structure Assumption where
  identifier : String
  statementHash : String
  deriving DecidableEq, Repr

structure ExplorationScope where
  bound : ExplorationBound
  assumptions : List Assumption
  deriving DecidableEq, Repr

def BoundedModel.explore (model : BoundedModel transition) (bound : ExplorationBound) :
    List (List transition.Action × transition.State) :=
  ((List.range (bound.maxDepth + 1)).flatMap model.frontier).take bound.maxResults

def ExecutableModel.follow (model : ExecutableModel transition) :
    List transition.State → List transition.Action → List transition.State
  | states, [] => states
  | states, action :: actions =>
      model.follow (states.flatMap fun state => model.next state action) actions

theorem BoundedModel.frontier_sound (model : BoundedModel transition)
    {history : List transition.Action} {state : transition.State} {depth : Nat}
    (member : (history, state) ∈ model.frontier depth) :
    ∃ initial, transition.Initial initial ∧ Runs transition initial history state := by
  induction depth generalizing history state with
  | zero =>
      rcases List.mem_map.mp member with ⟨initial, initialMember, equality⟩
      cases equality
      exact ⟨state, (model.initial_iff state).mp initialMember, Runs.nil state⟩
  | succ depth inductionHypothesis =>
      rcases List.mem_flatMap.mp member with ⟨execution, executionMember, actionMember⟩
      rcases List.mem_flatMap.mp actionMember with ⟨action, _, nextMember⟩
      rcases List.mem_map.mp nextMember with ⟨nextState, transitionMember, equality⟩
      cases equality
      rcases inductionHypothesis executionMember with ⟨initial, initialState, run⟩
      have step := (model.next_iff execution.2 action state).mp transitionMember
      exact ⟨initial, initialState,
        run.append (Runs.cons step (Runs.nil (model := transition) state))⟩

end Umpire3

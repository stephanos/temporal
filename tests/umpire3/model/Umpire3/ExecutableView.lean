import Umpire3.Behavior
import Umpire3.Executable

namespace Umpire3

universe u v

structure ExecutableView {World : Type u} (model : Behavior.{u, v} World) where
  initials : (world : World) → List (model.State world)
  successors : (world : World) → model.State world →
    List (model.Action world × model.State world)
  initials_exact : ∀ world state,
    state ∈ initials world ↔ model.Initial world state
  successors_exact : ∀ world state action nextState,
    (action, nextState) ∈ successors world state ↔
      model.Step world state action nextState

def ExecutableView.ofBoundedModel (bounded : BoundedModel transition) :
    ExecutableView (Behavior.ofTransitionSystem transition) where
  initials := fun _ => bounded.initials
  successors := fun _ state =>
    bounded.actions.flatMap fun action =>
      (bounded.next state action).map fun nextState => (action, nextState)
  initials_exact := by
    intro world state
    cases world
    exact bounded.initial_iff state
  successors_exact := by
    intro _ state action nextState
    constructor
    · intro member
      rcases List.mem_flatMap.mp member with ⟨candidate, _, successorMember⟩
      rcases List.mem_map.mp successorMember with ⟨candidateNext, nextMember, equality⟩
      cases equality
      exact (bounded.next_iff state action nextState).mp nextMember
    · intro step
      apply List.mem_flatMap.mpr
      refine ⟨action, bounded.action_complete state action nextState step, ?_⟩
      apply List.mem_map.mpr
      exact ⟨nextState, (bounded.next_iff state action nextState).mpr step, rfl⟩

def ExecutableView.follow {World : Type u} {model : Behavior.{u, v} World}
    (view : ExecutableView model)
    (world : World) [DecidableEq (model.Action world)] :
    List (model.State world) → List (model.Action world) → List (model.State world)
  | states, [] => states
  | states, action :: actions =>
      view.follow world
        (states.flatMap fun state =>
          (view.successors world state).filterMap fun successor =>
            if successor.1 = action then some successor.2 else none)
        actions

end Umpire3

import Umpire3.Behavior
import Umpire3.Execution
import Umpire3.Property

namespace Umpire3

structure ReplicatedWorld (World : Type) where
  core : World
  replicas : Nat

def replicatedBehavior (model : Behavior World) : Behavior (ReplicatedWorld World) where
  State := fun world => Fin 10 × model.State world.core
  Action := fun world => model.Action world.core
  Initial := fun world state => state.1.val < world.replicas ∧ model.Initial world.core state.2
  Step := fun world state action nextState =>
    state.1 = nextState.1 ∧ model.Step world.core state.2 action nextState.2

theorem replicatedRunsProject {model : Behavior World} {world : ReplicatedWorld World}
    {initial final : (replicatedBehavior model).State world}
    {actions : List ((replicatedBehavior model).Action world)}
    (run : Runs ((replicatedBehavior model).at world) initial actions final) :
    Runs (model.at world.core) initial.2 actions final.2 := by
  induction actions generalizing initial with
  | nil =>
      have equality := Runs.empty run
      subst final
      exact Runs.nil (model := model.at world.core) initial.2
  | cons action actions inductionHypothesis =>
      rcases Runs.uncons run with ⟨nextState, step, tail⟩
      exact Runs.cons step.2 (inductionHypothesis tail)

theorem replicatedReachableProject {model : Behavior World} {world : ReplicatedWorld World}
    {state : (replicatedBehavior model).State world}
    (reachable : (replicatedBehavior model).Reachable world state) :
    model.Reachable world.core state.2 := by
  rcases reachable with ⟨initial, actions, initialState, run⟩
  exact ⟨initial.2, actions, initialState.2, replicatedRunsProject run⟩

theorem replicatedSafety {model : Behavior World} {world : ReplicatedWorld World}
    {property : model.State world.core → Prop}
    (safe : Safety (model.at world.core) property) :
    Safety ((replicatedBehavior model).at world) (fun state => property state.2) := by
  intro state reachable
  exact safe state.2 (replicatedReachableProject reachable)

end Umpire3

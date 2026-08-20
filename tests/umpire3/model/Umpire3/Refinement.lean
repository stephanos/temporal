import Umpire3.Execution

namespace Umpire3

structure Refinement (system product : TransitionSystem) where
  Relates : system.State → product.State → Prop
  initial : ∀ systemState,
    system.Initial systemState →
    ∃ productState, product.Initial productState ∧ Relates systemState productState
  step : ∀ systemState productState action nextSystemState,
    Relates systemState productState →
    system.Step systemState action nextSystemState →
    ∃ nextProductState,
      product.StepStar productState nextProductState ∧
      Relates nextSystemState nextProductState

def Refinement.Stutters (refinement : Refinement system product)
    (systemState : system.State) (productState : product.State)
    (action : system.Action) (nextSystemState : system.State) : Prop :=
  refinement.Relates systemState productState ∧
    system.Step systemState action nextSystemState ∧
    refinement.Relates nextSystemState productState

def Refinement.identity (model : TransitionSystem) : Refinement model model where
  Relates := Eq
  initial := by
    intro state initial
    exact ⟨state, initial, rfl⟩
  step := by
    intro state productState action nextState related step
    subst productState
    exact ⟨nextState, model.stepStarSingle step, rfl⟩

inductive ActionEmission (Action : Type u) where
  | stutter
  | one (action : Action)
  | many (actions : List Action)

def ActionEmission.actions : ActionEmission Action → List Action
  | .stutter => []
  | .one action => [action]
  | .many actions => actions

def StepSimulation (system feature : Behavior World)
    (Relates : {world : World} → system.State world → feature.State world → Prop)
    (mapAction : {world : World} →
      system.Action world → ActionEmission (feature.Action world)) : Prop :=
  ∀ {world} systemState featureState action nextSystemState,
    Relates systemState featureState →
    system.Step world systemState action nextSystemState →
    ∃ nextFeatureState,
      Runs (feature.at world) featureState (mapAction action).actions nextFeatureState ∧
      Relates nextSystemState nextFeatureState

structure SafetySimulation (system feature : Behavior World) where
  Relates : {world : World} → system.State world → feature.State world → Prop
  mapAction : {world : World} →
    system.Action world → ActionEmission (feature.Action world)
  initial : ∀ {world} systemState,
    system.Initial world systemState →
    ∃ featureState, feature.Initial world featureState ∧ Relates systemState featureState
  step : StepSimulation system feature Relates mapAction

end Umpire3

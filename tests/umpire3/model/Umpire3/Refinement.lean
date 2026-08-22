import Umpire3.Execution

namespace Umpire3

structure Refinement (system feature : TransitionSystem) where
  Relates : system.State → feature.State → Prop
  initial : ∀ systemState,
    system.Initial systemState →
    ∃ featureState, feature.Initial featureState ∧ Relates systemState featureState
  step : ∀ systemState featureState action nextSystemState,
    Relates systemState featureState →
    system.Step systemState action nextSystemState →
    ∃ nextFeatureState,
      feature.StepStar featureState nextFeatureState ∧
      Relates nextSystemState nextFeatureState

def Refinement.Stutters (refinement : Refinement system feature)
    (systemState : system.State) (featureState : feature.State)
    (action : system.Action) (nextSystemState : system.State) : Prop :=
  refinement.Relates systemState featureState ∧
    system.Step systemState action nextSystemState ∧
    refinement.Relates nextSystemState featureState

def Refinement.identity (model : TransitionSystem) : Refinement model model where
  Relates := Eq
  initial := by
    intro state initial
    exact ⟨state, initial, rfl⟩
  step := by
    intro state featureState action nextState related step
    subst featureState
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

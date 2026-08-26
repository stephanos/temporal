namespace Shared

universe u

structure TransitionSystem where
  State : Type u
  Action : Type u
  Initial : State → Prop
  Step : State → Action → State → Prop

inductive Runs (model : TransitionSystem) :
    model.State → List model.Action → model.State → Prop where
  | nil (state) : Runs model state [] state
  | cons : model.Step state action next →
      Runs model next actions final →
      Runs model state (action :: actions) final

def TransitionSystem.Reachable (model : TransitionSystem) (state : model.State) : Prop :=
  ∃ initial actions, model.Initial initial ∧ Runs model initial actions state

def TransitionSystem.StepStar (model : TransitionSystem)
    (state final : model.State) : Prop :=
  ∃ actions, Runs model state actions final

theorem TransitionSystem.stepStarRefl (model : TransitionSystem) (state : model.State) :
    model.StepStar state state :=
  ⟨[], Runs.nil state⟩

theorem TransitionSystem.stepStarSingle (model : TransitionSystem)
    {state nextState : model.State} {action : model.Action}
    (step : model.Step state action nextState) : model.StepStar state nextState :=
  ⟨[action], Runs.cons step (Runs.nil nextState)⟩

theorem Runs.append {model : TransitionSystem}
    {start middle final : model.State} {actions more : List model.Action}
    (firstRun : Runs model start actions middle)
    (secondRun : Runs model middle more final) :
    Runs model start (actions ++ more) final := by
  induction firstRun generalizing final with
  | nil => exact secondRun
  | cons step _ ih => exact Runs.cons step (ih secondRun)

theorem Runs.empty {model : TransitionSystem} {start final : model.State}
    (run : Runs model start [] final) : start = final := by
  cases run
  rfl

theorem Runs.firstStep {model : TransitionSystem} {start final : model.State}
    {action : model.Action} {actions : List model.Action}
    (run : Runs model start (action :: actions) final) :
    ∃ nextState, model.Step start action nextState := by
  cases run with
  | cons step _ => exact ⟨_, step⟩

theorem Runs.uncons {model : TransitionSystem} {start final : model.State}
    {action : model.Action} {actions : List model.Action}
    (run : Runs model start (action :: actions) final) :
    ∃ nextState, model.Step start action nextState ∧
      Runs model nextState actions final := by
  cases run with
  | cons step tail => exact ⟨_, step, tail⟩

structure Observation (Action Delta : Type) where
  action : Action
  preStateIdentity : String
  postStateIdentity : String
  observableDelta : Delta

structure TraceStep (Action Delta : Type) extends Observation Action Delta where
  ordinal : Nat

end Shared

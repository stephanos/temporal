import Shared.Transition

namespace Umpire3

abbrev TransitionSystem := Shared.TransitionSystem

abbrev Runs := Shared.Runs

namespace Runs

abbrev nil := @Shared.Runs.nil

abbrev cons := @Shared.Runs.cons

theorem append {model : TransitionSystem}
    {start middle final : model.State} {actions more : List model.Action}
    (firstRun : Runs model start actions middle)
    (secondRun : Runs model middle more final) :
    Runs model start (actions ++ more) final :=
  Shared.Runs.append firstRun secondRun

theorem empty {model : TransitionSystem} {start final : model.State}
    (run : Runs model start [] final) : start = final :=
  Shared.Runs.empty run

theorem firstStep {model : TransitionSystem} {start final : model.State}
    {action : model.Action} {actions : List model.Action}
    (run : Runs model start (action :: actions) final) :
    ∃ nextState, model.Step start action nextState :=
  Shared.Runs.firstStep run

theorem uncons {model : TransitionSystem} {start final : model.State}
    {action : model.Action} {actions : List model.Action}
    (run : Runs model start (action :: actions) final) :
    ∃ nextState, model.Step start action nextState ∧
      Runs model nextState actions final :=
  Shared.Runs.uncons run

end Runs

namespace TransitionSystem

def Reachable (model : TransitionSystem) (state : model.State) : Prop :=
  Shared.TransitionSystem.Reachable model state

def StepStar (model : TransitionSystem) (state final : model.State) : Prop :=
  Shared.TransitionSystem.StepStar model state final

theorem stepStarRefl (model : TransitionSystem) (state : model.State) :
    StepStar model state state :=
  Shared.TransitionSystem.stepStarRefl model state

theorem stepStarSingle (model : TransitionSystem)
    {state nextState : model.State} {action : model.Action}
    (step : model.Step state action nextState) : StepStar model state nextState :=
  Shared.TransitionSystem.stepStarSingle model step

end TransitionSystem

end Umpire3

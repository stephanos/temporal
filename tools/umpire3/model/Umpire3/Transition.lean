import SharedModel.Transition

namespace Umpire3

abbrev TransitionSystem := SharedModel.TransitionSystem

abbrev Runs := SharedModel.Runs

namespace Runs

abbrev nil := @SharedModel.Runs.nil

abbrev cons := @SharedModel.Runs.cons

theorem append {model : TransitionSystem}
    {start middle final : model.State} {actions more : List model.Action}
    (firstRun : Runs model start actions middle)
    (secondRun : Runs model middle more final) :
    Runs model start (actions ++ more) final :=
  SharedModel.Runs.append firstRun secondRun

theorem empty {model : TransitionSystem} {start final : model.State}
    (run : Runs model start [] final) : start = final :=
  SharedModel.Runs.empty run

theorem firstStep {model : TransitionSystem} {start final : model.State}
    {action : model.Action} {actions : List model.Action}
    (run : Runs model start (action :: actions) final) :
    ∃ nextState, model.Step start action nextState :=
  SharedModel.Runs.firstStep run

theorem uncons {model : TransitionSystem} {start final : model.State}
    {action : model.Action} {actions : List model.Action}
    (run : Runs model start (action :: actions) final) :
    ∃ nextState, model.Step start action nextState ∧
      Runs model nextState actions final :=
  SharedModel.Runs.uncons run

end Runs

namespace TransitionSystem

def Reachable (model : TransitionSystem) (state : model.State) : Prop :=
  SharedModel.TransitionSystem.Reachable model state

def StepStar (model : TransitionSystem) (state final : model.State) : Prop :=
  SharedModel.TransitionSystem.StepStar model state final

theorem stepStarRefl (model : TransitionSystem) (state : model.State) :
    StepStar model state state :=
  SharedModel.TransitionSystem.stepStarRefl model state

theorem stepStarSingle (model : TransitionSystem)
    {state nextState : model.State} {action : model.Action}
    (step : model.Step state action nextState) : StepStar model state nextState :=
  SharedModel.TransitionSystem.stepStarSingle model step

end TransitionSystem

end Umpire3

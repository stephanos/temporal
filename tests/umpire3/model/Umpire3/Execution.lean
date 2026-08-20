import Umpire3.Behavior

namespace Umpire3

abbrev Behavior.Execution (model : Behavior World) (world : World) :=
  Runs (model.at world)

def Behavior.Reachable (model : Behavior World) (world : World)
    (state : model.State world) : Prop :=
  ∃ initial actions, model.Initial world initial ∧
    model.Execution world initial actions state

end Umpire3

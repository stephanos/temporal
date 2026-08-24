import Umpire3.Transition

namespace Umpire3

universe u v

structure Behavior (World : Type u) where
  State : World → Type v
  Action : World → Type v
  Initial : (world : World) → State world → Prop
  Step : (world : World) → State world → Action world → State world → Prop

def Behavior.at (model : Behavior World) (world : World) : TransitionSystem where
  State := model.State world
  Action := model.Action world
  Initial := model.Initial world
  Step := model.Step world

def Behavior.ofTransitionSystem (model : TransitionSystem) : Behavior Unit where
  State := fun _ => model.State
  Action := fun _ => model.Action
  Initial := fun _ => model.Initial
  Step := fun _ => model.Step

end Umpire3

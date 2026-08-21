import Umpire3.Behavior

namespace Umpire3

universe u v

structure InfiniteExecution {World : Type u} (model : Behavior World) (world : World) where
  state : Nat → model.State world
  action : Nat → Option (model.Action world)
  initial : model.Initial world (state 0)
  step : ∀ index, match action index with
    | none => state (index + 1) = state index
    | some selected => model.Step world (state index) selected (state (index + 1))

def Behavior.Enabled {World : Type u} (model : Behavior World) (world : World)
    (state : model.State world) (action : model.Action world) : Prop :=
  ∃ nextState, model.Step world state action nextState

def EventuallyFrom {State : Type u} (predicate : State → Prop)
    (states : Nat → State) (start : Nat) : Prop :=
  ∃ index, start ≤ index ∧ predicate (states index)

def AlwaysFrom {State : Type u} (predicate : State → Prop)
    (states : Nat → State) (start : Nat) : Prop :=
  ∀ index, start ≤ index → predicate (states index)

def LeadsTo {State : Type u} (trigger goal : State → Prop) (states : Nat → State) : Prop :=
  ∀ start, trigger (states start) → EventuallyFrom goal states start

def InfiniteExecution.Occurs {World : Type u} {model : Behavior World} {world : World}
    (execution : InfiniteExecution model world) (action : model.Action world) (index : Nat) : Prop :=
  execution.action index = some action

def InfiniteExecution.WeakFair {World : Type u} {model : Behavior World} {world : World}
    (execution : InfiniteExecution model world) (action : model.Action world) : Prop :=
  ∀ start,
    AlwaysFrom (fun state => model.Enabled world state action) execution.state start →
    ∃ index, start ≤ index ∧ execution.Occurs action index

def InfiniteExecution.StrongFair {World : Type u} {model : Behavior World} {world : World}
    (execution : InfiniteExecution model world) (action : model.Action world) : Prop :=
  ∀ start,
    (∀ lowerBound, start ≤ lowerBound →
      ∃ index, lowerBound ≤ index ∧ model.Enabled world (execution.state index) action) →
    ∃ index, start ≤ index ∧ execution.Occurs action index

def InfiniteExecution.ResponsiveFair {World : Type u} {model : Behavior World} {world : World}
    (execution : InfiniteExecution model world) (action : model.Action world) : Prop :=
  ∀ start, model.Enabled world (execution.state start) action →
    ∃ index, start ≤ index ∧ execution.Occurs action index

end Umpire3

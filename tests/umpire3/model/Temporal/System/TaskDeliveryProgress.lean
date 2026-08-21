import Umpire3.TemporalLogic

namespace Umpire3.Temporal.System.TaskDeliveryProgress

inductive Phase where
  | unavailable
  | ready
  | completed
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | recover
  | deliver
  deriving DecidableEq, Inhabited, Repr

def Step : Phase → Action → Phase → Prop
  | .unavailable, .recover, .ready => True
  | .ready, .deliver, .completed => True
  | _, _, _ => False

abbrev behavior : Behavior Unit where
  State := fun _ => Phase
  Action := fun _ => Action
  Initial := fun _ state => state = .unavailable
  Step := fun _ => Step

def Unfinished (state : Phase) : Prop := state ≠ .completed

def Completed (state : Phase) : Prop := state = .completed

structure Fairness (execution : InfiniteExecution behavior ()) where
  delivery : execution.ResponsiveFair .deliver
  recovery : execution.ResponsiveFair .recover

def deliveryFairnessIdentifier : String := "task-delivery.delivery-responsive"

def recoveryFairnessIdentifier : String := "task-delivery.recovery-responsive"

private theorem scheduledDeliveryCompletes (execution : InfiniteExecution behavior ())
    {index : Nat} (scheduled : execution.Occurs .deliver index) :
    execution.state (index + 1) = .completed := by
  have transition := execution.step index
  rw [scheduled] at transition
  generalize sourceEquality : execution.state index = source at transition
  generalize targetEquality : execution.state (index + 1) = target at transition
  cases source <;> cases target <;> simp [Step] at transition ⊢

private theorem scheduledRecoveryReadies (execution : InfiniteExecution behavior ())
    {index : Nat} (scheduled : execution.Occurs .recover index) :
    execution.state (index + 1) = .ready := by
  have transition := execution.step index
  rw [scheduled] at transition
  generalize sourceEquality : execution.state index = source at transition
  generalize targetEquality : execution.state (index + 1) = target at transition
  cases source <;> cases target <;> simp [Step] at transition ⊢

theorem progressUnderFairness (execution : InfiniteExecution behavior ())
    (fairness : Fairness execution) :
    LeadsTo Unfinished Completed execution.state := by
  intro start unfinished
  cases current : execution.state start with
  | unavailable =>
      have enabled : behavior.Enabled () (execution.state start) .recover := by
        rw [current]
        exact ⟨.ready, by trivial⟩
      rcases fairness.recovery start enabled with ⟨recoveryIndex, afterStart, recoveryScheduled⟩
      have ready := scheduledRecoveryReadies execution recoveryScheduled
      have deliveryEnabled : behavior.Enabled () (execution.state (recoveryIndex + 1)) .deliver := by
        rw [ready]
        exact ⟨.completed, by trivial⟩
      rcases fairness.delivery (recoveryIndex + 1) deliveryEnabled with
        ⟨deliveryIndex, afterRecovery, deliveryScheduled⟩
      exact ⟨deliveryIndex + 1,
        Nat.le_trans afterStart (Nat.le_trans (Nat.le_succ recoveryIndex)
          (Nat.le_trans afterRecovery (Nat.le_succ deliveryIndex))),
        scheduledDeliveryCompletes execution deliveryScheduled⟩
  | ready =>
      have enabled : behavior.Enabled () (execution.state start) .deliver := by
        rw [current]
        exact ⟨.completed, by trivial⟩
      rcases fairness.delivery start enabled with ⟨index, afterStart, scheduled⟩
      exact ⟨index + 1, Nat.le_trans afterStart (Nat.le_succ index),
        scheduledDeliveryCompletes execution scheduled⟩
  | completed => exact False.elim (unfinished current)

def MutatedStep : Phase → Action → Phase → Prop
  | .unavailable, .recover, .ready => True
  | _, _, _ => False

abbrev mutatedBehavior : Behavior Unit where
  State := fun _ => Phase
  Action := fun _ => Action
  Initial := fun _ state => state = .unavailable
  Step := fun _ => MutatedStep

def mutatedLassoState : Nat → Phase
  | 0 => .unavailable
  | _ + 1 => .ready

def mutatedLassoAction : Nat → Option Action
  | 0 => some .recover
  | _ + 1 => none

def mutatedLasso : InfiniteExecution mutatedBehavior () where
  state := mutatedLassoState
  action := mutatedLassoAction
  initial := rfl
  step := by
    intro index
    cases index with
    | zero => trivial
    | succ index => rfl

theorem mutatedLassoViolatesProgress :
    ¬LeadsTo Unfinished Completed mutatedLasso.state := by
  intro progresses
  rcases progresses 0 (by simp [Unfinished, mutatedLasso, mutatedLassoState]) with
    ⟨index, _, completed⟩
  cases index <;> simp [mutatedLasso, mutatedLassoState, Completed] at completed

end Umpire3.Temporal.System.TaskDeliveryProgress

import Temporal.Product.Update
import Temporal.System.TaskDelivery

namespace Umpire3.Temporal.System.UpdateTasks

structure State where
  visible : Temporal.Product.Update.State
  taskDispatched : Bool
  historyRecorded : Bool
  ownerEpoch : Nat
  completionEpoch : Option Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | StartUpdate
  | DispatchWorkflowTask
  | AcceptUpdate
  | RecordUpdateHistory
  | CompleteWorkflowTask
  | CompleteUpdate
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := .idle
  taskDispatched := false
  historyRecorded := false
  ownerEpoch := 0
  completionEpoch := none

structure TransitionResult (state : State) where
  nextState : State
  productActions : List Temporal.Product.Update.Command
  productRun : Runs Temporal.Product.Update.product state.visible productActions nextState.visible

def stutterResult (state nextState : State)
    (sameVisible : nextState.visible = state.visible) : TransitionResult state where
  nextState := nextState
  productActions := []
  productRun := by
    rw [sameVisible]
    exact Runs.nil (model := Temporal.Product.Update.product) state.visible

def productResult (state nextState : State) (action : Temporal.Product.Update.Command)
    (productStep : Temporal.Product.Update.product.Step state.visible action nextState.visible) :
    TransitionResult state where
  nextState := nextState
  productActions := [action]
  productRun := Runs.cons productStep
    (Runs.nil (model := Temporal.Product.Update.product) nextState.visible)

def transitions (state : State) : Action → List (TransitionResult state)
  | .StartUpdate =>
      if starts : state.visible = .idle then
        [productResult state { state with visible := .requested } .request (by
          cases visible : state.visible <;> simp_all [Temporal.Product.Update.step])]
      else []
  | .DispatchWorkflowTask =>
      if state.visible = .requested ∧ state.taskDispatched = false then
        [stutterResult state { state with taskDispatched := true } rfl]
      else []
  | .AcceptUpdate =>
      if accepts : state.visible = .requested ∧ state.taskDispatched = true then
        [productResult state { state with visible := .accepted } .accept (by
          rw [accepts.1]
          simp [Temporal.Product.Update.step])]
      else []
  | .RecordUpdateHistory =>
      if state.visible = .accepted then
        [stutterResult state { state with historyRecorded := true } rfl]
      else []
  | .CompleteWorkflowTask =>
      if state.taskDispatched = true then
        [stutterResult state { state with completionEpoch := some state.ownerEpoch } rfl]
      else []
  | .CompleteUpdate =>
      if completes : state.visible = .accepted ∧ state.historyRecorded = true ∧
          Temporal.System.TaskDelivery.CurrentCompletion state.completionEpoch state.ownerEpoch then
        [productResult state { state with visible := .completed } .complete (by
          rw [completes.1]
          simp [Temporal.Product.Update.step])]
      else []

def next (state : State) (action : Action) : List State :=
  (transitions state action).map TransitionResult.nextState

def step (state : State) (action : Action) (nextState : State) : Prop :=
  ∃ result, result ∈ transitions state action ∧ result.nextState = nextState

abbrev system : TransitionSystem where
  State := State
  Action := Action
  Initial := fun state => state = initial
  Step := step

theorem next_iff (state action nextState) :
    nextState ∈ next state action ↔ system.Step state action nextState := by
  constructor
  · intro member
    rcases List.mem_map.mp member with ⟨result, resultMember, equality⟩
    exact ⟨result, resultMember, equality⟩
  · rintro ⟨result, resultMember, rfl⟩
    exact List.mem_map.mpr ⟨result, resultMember, rfl⟩

def executable : ExecutableModel system where
  next := next
  next_iff := next_iff

def updateDeliveryRequirement : Temporal.System.TaskDelivery.Requirement where
  provider := Temporal.System.TaskDelivery.guarantee.identifier
  statementHash := Temporal.System.TaskDelivery.guarantee.statementHash

end Umpire3.Temporal.System.UpdateTasks

import Umpire3.Declaration
import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Feature.TaskAck

inductive State where
  | empty
  | queued
  | delivered
  | acknowledged
  | acknowledgedWithBacklog
  deriving DecidableEq, Inhabited, Repr

inductive Command where
  | enqueue
  | deliver
  | acknowledge
  deriving DecidableEq, Inhabited, Repr

def step : State → Command → State → Prop
  | .empty, .enqueue, .queued => True
  | .queued, .deliver, .delivered => True
  | .delivered, .acknowledge, .acknowledged => True
  | _, _, _ => False

abbrev feature : TransitionSystem where
  State := State
  Action := Command
  Initial := (· = .empty)
  Step := step

def next : State → Command → List State
  | .empty, .enqueue => [.queued]
  | .queued, .deliver => [.delivered]
  | .delivered, .acknowledge => [.acknowledged]
  | _, _ => []

theorem next_iff (state action nextState) :
    nextState ∈ next state action ↔ feature.Step state action nextState := by
  cases state <;> cases action <;> cases nextState <;> simp [next, step]

def executable : ExecutableModel feature where
  next := next
  next_iff := next_iff

def bounded : BoundedModel feature where
  toExecutableModel := executable
  initials := [.empty]
  initial_iff := by intro state; cases state <;> simp
  actions := [.enqueue, .deliver, .acknowledge]
  action_complete := by intro state action nextState step; cases action <;> simp

def BacklogAbsent : State → Prop
  | .empty | .acknowledged => True
  | _ => False

def Acknowledged : State → Prop
  | .acknowledged | .acknowledgedWithBacklog => True
  | _ => False

def AcknowledgementConsistent (state : State) : Prop :=
  Acknowledged state → BacklogAbsent state

theorem successorAcknowledgementConsistent {state action nextState}
    (transition : feature.Step state action nextState) :
    AcknowledgementConsistent nextState := by
  cases state <;> cases action <;> cases nextState <;>
    simp [step, AcknowledgementConsistent, Acknowledged, BacklogAbsent] at transition ⊢

theorem runsPreserveAcknowledgementConsistency {start actions final}
    (run : Runs feature start actions final)
    (consistent : AcknowledgementConsistent start) :
    AcknowledgementConsistent final := by
  induction run with
  | nil => exact consistent
  | cons transition _ induction =>
      exact induction (successorAcknowledgementConsistent transition)

theorem acknowledged_removes_backlog : Safety feature AcknowledgementConsistent := by
  intro state reachable
  rcases reachable with ⟨start, actions, initialState, run⟩
  subst start
  exact runsPreserveAcknowledgementConsistency run (by
    simp [AcknowledgementConsistent, Acknowledged])

theorem acknowledgementMutationNegativeControl :
    ¬AcknowledgementConsistent .acknowledgedWithBacklog := by
  simp [AcknowledgementConsistent, Acknowledged, BacklogAbsent]

theorem acknowledged_is_stable {action nextState}
    (transition : feature.Step .acknowledged action nextState) : nextState = .acknowledged := by
  cases action <;> cases nextState <;> simp [step] at transition

def declaration : LifecycleDeclaration where
  entity := { identifier := "workflow-task", description := "Workflow Task delivery lifecycle" }
  actions := [
    { identifier := "enqueue-workflow-task", description := "Enqueue a Workflow Task",
      parameters := [], requiredCapabilities := ["workflow-task-control"] },
    { identifier := "deliver-workflow-task", description := "Deliver a Workflow Task",
      parameters := [], requiredCapabilities := ["workflow-task-control"] },
    { identifier := "acknowledge-workflow-task", description := "Acknowledge a Workflow Task",
      parameters := [], requiredCapabilities := ["workflow-task-control"] },
  ]
  observations := [{
    identifier := "workflow-task-acknowledged"
    description := "The delivered Workflow Task was acknowledged"
  }]
  properties := [
    (RegisteredProperty.mk
      "task-delivery.acknowledged-removes-backlog"
      "Acknowledging a delivered Workflow Task removes it from backlog"
      ["source-sequence", "identity-lineage"]
      (resolved_theorem% acknowledged_removes_backlog)).declaration
  ]
  module := {
    identifier := "Temporal.Feature.TaskAck"
    description := "Workflow Task acknowledgement feature contract"
  }
  target := {
    identifier := "foundation-backlog-ack"
    modules := ["Temporal.Feature.TaskAck", "Temporal.System.TaskAck", "Temporal.Refinement.TaskAck"]
    properties := ["task-delivery.acknowledged-removes-backlog"]
  }

theorem declarationWellFormed : declaration.WellFormed := by rfl

end Umpire3.Temporal.Feature.TaskAck

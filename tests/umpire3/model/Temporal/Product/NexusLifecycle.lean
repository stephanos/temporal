import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.NexusLifecycle

inductive State where
  | unspecified
  | scheduled
  | backingOff
  | started
  | succeeded
  | failed
  | canceled
  | timedOut
  | terminated
  | rejected
  deriving DecidableEq, Inhabited, Repr

inductive Command where
  | schedule
  | attemptFailed
  | start
  | succeed
  | fail
  | cancel
  | timeout
  | terminate
  | reject
  deriving DecidableEq, Inhabited, Repr

def initial : State := .unspecified

def step : State → Command → State → Prop
  | .unspecified, .schedule, .scheduled => True
  | .backingOff, .schedule, .scheduled => True
  | .scheduled, .attemptFailed, .backingOff => True
  | .scheduled, .start, .started => True
  | .scheduled, .succeed, .succeeded => True
  | .started, .succeed, .succeeded => True
  | .scheduled, .fail, .failed => True
  | .started, .fail, .failed => True
  | .scheduled, .cancel, .canceled => True
  | .started, .cancel, .canceled => True
  | .scheduled, .timeout, .timedOut => True
  | .backingOff, .timeout, .timedOut => True
  | .started, .timeout, .timedOut => True
  | .scheduled, .terminate, .terminated => True
  | .backingOff, .terminate, .terminated => True
  | .started, .terminate, .terminated => True
  | .unspecified, .reject, .rejected => True
  | _, _, _ => False

abbrev product : TransitionSystem where
  State := State
  Action := Command
  Initial := fun state => state = initial
  Step := step

def next : State → Command → List State
  | .unspecified, .schedule => [.scheduled]
  | .backingOff, .schedule => [.scheduled]
  | .scheduled, .attemptFailed => [.backingOff]
  | .scheduled, .start => [.started]
  | .scheduled, .succeed => [.succeeded]
  | .started, .succeed => [.succeeded]
  | .scheduled, .fail => [.failed]
  | .started, .fail => [.failed]
  | .scheduled, .cancel => [.canceled]
  | .started, .cancel => [.canceled]
  | .scheduled, .timeout => [.timedOut]
  | .backingOff, .timeout => [.timedOut]
  | .started, .timeout => [.timedOut]
  | .scheduled, .terminate => [.terminated]
  | .backingOff, .terminate => [.terminated]
  | .started, .terminate => [.terminated]
  | .unspecified, .reject => [.rejected]
  | _, _ => []

theorem next_iff (state action nextState) :
    nextState ∈ next state action ↔ product.Step state action nextState := by
  cases state <;> cases action <;> cases nextState <;> simp [next, step]

def executable : ExecutableModel product where
  next := next
  next_iff := next_iff

def bounded : BoundedModel product where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by
    intro state
    cases state <;> decide
  actions := [
    .schedule, .attemptFailed, .start, .succeed, .fail, .cancel, .timeout, .terminate, .reject
  ]
  action_complete := by
    intro state action nextState _
    cases action <;> simp

structure Edge where
  identifier : String
  fromState : String
  action : String
  toState : String
  requiresFault : Bool := false
  standaloneOnly : Bool := false
  deriving DecidableEq, Repr

def edges : List Edge := [
  ⟨"nexus-operation/unspecified/schedule/scheduled", "unspecified", "schedule", "scheduled", false, false⟩,
  ⟨"nexus-operation/backing-off/schedule/scheduled", "backing-off", "schedule", "scheduled", false, false⟩,
  ⟨"nexus-operation/scheduled/attempt-failed/backing-off", "scheduled", "attempt-failed", "backing-off", false, false⟩,
  ⟨"nexus-operation/scheduled/start/started", "scheduled", "start", "started", false, false⟩,
  ⟨"nexus-operation/scheduled/succeed/succeeded", "scheduled", "succeed", "succeeded", false, false⟩,
  ⟨"nexus-operation/started/succeed/succeeded", "started", "succeed", "succeeded", false, false⟩,
  ⟨"nexus-operation/scheduled/fail/failed", "scheduled", "fail", "failed", false, false⟩,
  ⟨"nexus-operation/started/fail/failed", "started", "fail", "failed", false, false⟩,
  ⟨"nexus-operation/scheduled/cancel/canceled", "scheduled", "cancel", "canceled", false, false⟩,
  ⟨"nexus-operation/started/cancel/canceled", "started", "cancel", "canceled", false, false⟩,
  ⟨"nexus-operation/scheduled/timeout/timed-out", "scheduled", "timeout", "timed-out", true, false⟩,
  ⟨"nexus-operation/backing-off/timeout/timed-out", "backing-off", "timeout", "timed-out", true, false⟩,
  ⟨"nexus-operation/started/timeout/timed-out", "started", "timeout", "timed-out", true, false⟩,
  ⟨"nexus-operation/scheduled/terminate/terminated", "scheduled", "terminate", "terminated", false, true⟩,
  ⟨"nexus-operation/backing-off/terminate/terminated", "backing-off", "terminate", "terminated", false, true⟩,
  ⟨"nexus-operation/started/terminate/terminated", "started", "terminate", "terminated", false, true⟩,
  ⟨"nexus-operation/unspecified/reject/rejected", "unspecified", "reject", "rejected", false, false⟩
]

theorem edge_count : edges.length = 17 := by decide

theorem edge_identifiers_unique : (edges.map (·.identifier)).Nodup := by decide

def Terminal : State → Prop
  | .succeeded | .failed | .canceled | .timedOut | .terminated | .rejected => True
  | _ => False

theorem terminal_stable {state action nextState}
    (terminal : Terminal state) (transition : product.Step state action nextState) :
    state = nextState := by
  cases state <;> cases action <;> cases nextState <;> simp [Terminal, step] at terminal transition ⊢

end Umpire3.Temporal.Product.NexusLifecycle

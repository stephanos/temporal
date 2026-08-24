import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Feature.NexusClosure

inductive OperationID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive WorkflowState where
  | running
  | completed
  | failed
  | cancelled
  | terminated
  | timedOut
  deriving BEq, DecidableEq, Inhabited, Repr

def WorkflowState.terminal : WorkflowState → Bool
  | .running => false
  | _ => true

inductive OperationState where
  | absent
  | scheduled
  | started
  | succeeded
  | failed
  | cancelled
  | timedOut
  | terminated
  | rejected
  deriving BEq, DecidableEq, Inhabited, Repr

def OperationState.terminal : OperationState → Bool
  | .succeeded | .failed | .cancelled | .timedOut | .terminated | .rejected => true
  | _ => false

inductive OperationOutcome where
  | succeeded
  | failed
  | cancelled
  | timedOut
  | terminated
  | rejected
  deriving BEq, DecidableEq, Inhabited, Repr

def OperationOutcome.state : OperationOutcome → OperationState
  | .succeeded => .succeeded
  | .failed => .failed
  | .cancelled => .cancelled
  | .timedOut => .timedOut
  | .terminated => .terminated
  | .rejected => .rejected

inductive WorkflowOutcome where
  | completed
  | failed
  | cancelled
  | terminated
  | timedOut
  deriving BEq, DecidableEq, Inhabited, Repr

def WorkflowOutcome.state : WorkflowOutcome → WorkflowState
  | .completed => .completed
  | .failed => .failed
  | .cancelled => .cancelled
  | .terminated => .terminated
  | .timedOut => .timedOut

structure State where
  workflow : WorkflowState
  primaryOperation : OperationState
  secondaryOperation : OperationState
  primaryCaller : Bool
  secondaryCaller : Bool
  deriving DecidableEq, Inhabited, Repr

def State.operation (state : State) : OperationID → OperationState
  | .primary => state.primaryOperation
  | .secondary => state.secondaryOperation

def State.setOperation (state : State) : OperationID → OperationState → State
  | .primary, operation => { state with primaryOperation := operation }
  | .secondary, operation => { state with secondaryOperation := operation }

def State.caller (state : State) : OperationID → Bool
  | .primary => state.primaryCaller
  | .secondary => state.secondaryCaller

def State.setCaller (state : State) : OperationID → Bool → State
  | .primary, caller => { state with primaryCaller := caller }
  | .secondary, caller => { state with secondaryCaller := caller }

def operationIDs : List OperationID := [.primary, .secondary]

def allRelatedTerminal (state : State) : Bool :=
  operationIDs.all fun operation =>
    !state.caller operation || (state.operation operation).terminal

def closureB (state : State) : Bool :=
  !state.workflow.terminal || allRelatedTerminal state

def Closure (state : State) : Prop := closureB state = true

instance (state : State) : Decidable (Closure state) := by
  unfold Closure
  infer_instance

inductive Action where
  | schedule (operation : OperationID)
  | start (operation : OperationID)
  | settle (operation : OperationID) (outcome : OperationOutcome)
  | closeWorkflow (outcome : WorkflowOutcome)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  workflow := .running
  primaryOperation := .absent
  secondaryOperation := .absent
  primaryCaller := false
  secondaryCaller := false

def next (state : State) : Action → List State
  | .schedule operation =>
      if state.workflow.terminal then []
      else if state.operation operation == .absent then
        [(state.setOperation operation .scheduled).setCaller operation true]
      else []
  | .start operation =>
      if state.workflow.terminal then []
      else if state.operation operation == .scheduled then
        [state.setOperation operation .started]
      else []
  | .settle operation outcome =>
      if state.workflow.terminal then []
      else if state.operation operation == .scheduled || state.operation operation == .started then
        [state.setOperation operation outcome.state]
      else []
  | .closeWorkflow outcome =>
      if state.workflow.terminal then []
      else if allRelatedTerminal state then [{ state with workflow := outcome.state }]
      else []

def step (state : State) (action : Action) (nextState : State) : Prop :=
  nextState ∈ next state action

abbrev model : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := step

def executable : ExecutableModel model where
  next := next
  next_iff := by intros; rfl

def actions : List Action := [
  .schedule .primary,
  .schedule .secondary,
  .start .primary,
  .start .secondary,
  .settle .primary .succeeded,
  .settle .primary .failed,
  .settle .primary .cancelled,
  .settle .primary .timedOut,
  .settle .primary .terminated,
  .settle .primary .rejected,
  .settle .secondary .succeeded,
  .settle .secondary .failed,
  .settle .secondary .cancelled,
  .settle .secondary .timedOut,
  .settle .secondary .terminated,
  .settle .secondary .rejected,
  .closeWorkflow .completed,
  .closeWorkflow .failed,
  .closeWorkflow .cancelled,
  .closeWorkflow .terminated,
  .closeWorkflow .timedOut,
]

def bounded : BoundedModel model where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    cases action with
    | schedule operation => cases operation <;> simp [actions]
    | start operation => cases operation <;> simp [actions]
    | settle operation outcome => cases operation <;> cases outcome <;> simp [actions]
    | closeWorkflow outcome => cases outcome <;> simp [actions]

def permittedFinal : State := {
  initial with
  workflow := .completed
  primaryOperation := .succeeded
  primaryCaller := true
}

def unsafeNext (state : State) : Action → List State
  | .closeWorkflow outcome =>
      if state.workflow.terminal then [] else [{ state with workflow := outcome.state }]
  | action => next state action

abbrev unsafeModel : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := fun state action nextState => nextState ∈ unsafeNext state action

def unsafeExecutable : ExecutableModel unsafeModel where
  next := unsafeNext
  next_iff := by intros; rfl

def unsafeFinal : State := {
  initial with
  workflow := .completed
  primaryOperation := .started
  primaryCaller := true
}

theorem initialClosure : Closure initial := by decide

theorem successorClosure {state action nextState}
    (transition : model.Step state action nextState) : Closure nextState := by
  cases action with
  | schedule operation =>
      cases workflow : state.workflow <;> cases operation <;>
        simp [step, next, workflow, Closure, closureB, allRelatedTerminal, operationIDs,
          WorkflowState.terminal, State.operation, State.caller, State.setOperation,
          State.setCaller] at transition ⊢ <;> simp_all
  | start operation =>
      cases workflow : state.workflow <;> cases operation <;>
        simp [step, next, workflow, Closure, closureB, allRelatedTerminal, operationIDs,
          WorkflowState.terminal, State.operation, State.caller, State.setOperation]
          at transition ⊢ <;> simp_all
  | settle operation outcome =>
      cases workflow : state.workflow <;> cases operation <;> cases outcome <;>
        simp [step, next, workflow, Closure, closureB, allRelatedTerminal, operationIDs,
          WorkflowState.terminal, OperationOutcome.state, State.operation, State.caller,
          State.setOperation] at transition ⊢ <;> simp_all
  | closeWorkflow outcome =>
      cases workflow : state.workflow <;> cases outcome <;>
        simp [step, next, workflow, Closure, closureB, allRelatedTerminal, operationIDs,
          WorkflowState.terminal, WorkflowOutcome.state, State.operation, State.caller]
          at transition ⊢ <;> simp_all

theorem runsPreserveClosure {start actionHistory final}
    (run : Runs model start actionHistory final) (property : Closure start) : Closure final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorClosure transition)

theorem closureSafe : Safety model Closure := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveClosure run initialClosure

theorem unsafeClosureMutation : ¬Closure unsafeFinal := by decide

end Umpire3.Temporal.Feature.NexusClosure

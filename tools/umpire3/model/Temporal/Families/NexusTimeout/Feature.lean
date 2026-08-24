import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Feature.NexusTimeout

inductive OperationID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive EvidenceID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive TimeoutKind where
  | startToClose
  | scheduleToClose
  | unspecified
  deriving BEq, DecidableEq, Inhabited, Repr

inductive TimeoutMessage where
  | operationTimedOut
  | unrelatedFailure
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  primaryConfigured : Bool
  secondaryConfigured : Bool
  primaryTimedOut : Bool
  secondaryTimedOut : Bool
  primaryEvidence : Option EvidenceID
  secondaryEvidence : Option EvidenceID
  primaryEvidenceValid : Bool
  secondaryEvidenceValid : Bool
  deriving DecidableEq, Inhabited, Repr

def State.configured (state : State) : OperationID → Bool
  | .primary => state.primaryConfigured
  | .secondary => state.secondaryConfigured

def State.setConfigured (state : State) : OperationID → Bool → State
  | .primary, configured => { state with primaryConfigured := configured }
  | .secondary, configured => { state with secondaryConfigured := configured }

def State.timedOut (state : State) : OperationID → Bool
  | .primary => state.primaryTimedOut
  | .secondary => state.secondaryTimedOut

def State.setTimedOut (state : State) : OperationID → Bool → State
  | .primary, timedOut => { state with primaryTimedOut := timedOut }
  | .secondary, timedOut => { state with secondaryTimedOut := timedOut }

def State.timeoutEvidence (state : State) : OperationID → Option EvidenceID
  | .primary => state.primaryEvidence
  | .secondary => state.secondaryEvidence

def State.setTimeoutEvidence (state : State) : OperationID → Option EvidenceID → State
  | .primary, evidence => { state with primaryEvidence := evidence }
  | .secondary, evidence => { state with secondaryEvidence := evidence }

def State.evidenceValid (state : State) : EvidenceID → Bool
  | .primary => state.primaryEvidenceValid
  | .secondary => state.secondaryEvidenceValid

def State.setEvidenceValid (state : State) : EvidenceID → Bool → State
  | .primary, valid => { state with primaryEvidenceValid := valid }
  | .secondary, valid => { state with secondaryEvidenceValid := valid }

def operationIDs : List OperationID := [.primary, .secondary]

def evidenceIDs : List EvidenceID := [.primary, .secondary]

def evidenceUsed (state : State) (evidence : EvidenceID) : Bool :=
  operationIDs.any fun operation => state.timeoutEvidence operation == some evidence

def timeoutSemanticsB (state : State) : Bool :=
  operationIDs.all fun operation =>
    !state.timedOut operation ||
      match state.timeoutEvidence operation with
      | some evidence => state.evidenceValid evidence
      | none => false

def TimeoutSemantics (state : State) : Prop := timeoutSemanticsB state = true

instance (state : State) : Decidable (TimeoutSemantics state) := by
  unfold TimeoutSemantics
  infer_instance

inductive Action where
  | configure (operation : OperationID)
  | recordTimeout (operation : OperationID) (evidence : EvidenceID)
      (kind : TimeoutKind) (message : TimeoutMessage)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryConfigured := false
  secondaryConfigured := false
  primaryTimedOut := false
  secondaryTimedOut := false
  primaryEvidence := none
  secondaryEvidence := none
  primaryEvidenceValid := false
  secondaryEvidenceValid := false

def rawNext (state : State) : Action → List State
  | .configure operation =>
      if state.configured operation then [] else [state.setConfigured operation true]
  | .recordTimeout operation evidence kind message =>
      if state.configured operation && !state.timedOut operation && !evidenceUsed state evidence then
        let valid := kind == .startToClose && message == .operationTimedOut
        [((state.setTimedOut operation true).setTimeoutEvidence operation (some evidence)).setEvidenceValid
          evidence valid]
      else []

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter timeoutSemanticsB

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

def actions : List Action :=
  [.configure .primary, .configure .secondary] ++
    [OperationID.primary, OperationID.secondary].flatMap fun operation =>
      [EvidenceID.primary, EvidenceID.secondary].flatMap fun evidence =>
        [TimeoutKind.startToClose, TimeoutKind.scheduleToClose, TimeoutKind.unspecified].flatMap fun kind =>
          [TimeoutMessage.operationTimedOut, TimeoutMessage.unrelatedFailure].map fun message =>
            .recordTimeout operation evidence kind message

def bounded : BoundedModel model where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    cases action with
    | configure operation => cases operation <;> simp [actions]
    | recordTimeout operation evidence kind message =>
        cases operation <;> cases evidence <;> cases kind <;> cases message <;> simp [actions]

abbrev unsafeModel : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := fun state action nextState => nextState ∈ rawNext state action

def unsafeExecutable : ExecutableModel unsafeModel where
  next := rawNext
  next_iff := by intros; rfl

def permittedFinal : State := {
  initial with
  primaryConfigured := true
  primaryTimedOut := true
  primaryEvidence := some .primary
  primaryEvidenceValid := true
}

def unsafeInvalidFinal : State := {
  initial with
  primaryConfigured := true
  primaryTimedOut := true
  primaryEvidence := some .primary
}

theorem initialTimeoutSemantics : TimeoutSemantics initial := by decide

theorem successorTimeoutSemantics {state action nextState}
    (transition : model.Step state action nextState) : TimeoutSemantics nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveTimeoutSemantics {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : TimeoutSemantics start) : TimeoutSemantics final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorTimeoutSemantics transition)

theorem timeoutSemanticsSafe : Safety model TimeoutSemantics := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveTimeoutSemantics run initialTimeoutSemantics

theorem timeoutMetadataMutationNegativeControl : ¬TimeoutSemantics unsafeInvalidFinal := by decide

end Umpire3.Temporal.Feature.NexusTimeout

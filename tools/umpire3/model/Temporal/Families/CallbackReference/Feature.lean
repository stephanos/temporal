import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Feature.CallbackReference

inductive CallbackID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive OperationID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive HandlerID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive ReferenceKind where
  | event
  | request
  deriving BEq, DecidableEq, Inhabited, Repr

inductive ReferenceValue where
  | workflowStarted
  | optionsUpdated
  | unrelated
  deriving BEq, DecidableEq, Inhabited, Repr

inductive Position where
  | first
  | second
  | third
  deriving BEq, DecidableEq, Inhabited, Repr

def Position.rank : Position → Nat
  | .first => 1
  | .second => 2
  | .third => 3

def atOrAfterB (later earlier : Position) : Bool := earlier.rank ≤ later.rank

structure State where
  primaryAttachmentObserved : Bool
  secondaryAttachmentObserved : Bool
  primaryOperationObserved : Bool
  secondaryOperationObserved : Bool
  primaryCallbackOperation : Option OperationID
  secondaryCallbackOperation : Option OperationID
  primaryCallbackHandler : Option HandlerID
  secondaryCallbackHandler : Option HandlerID
  primaryOperationHandler : Option HandlerID
  secondaryOperationHandler : Option HandlerID
  primaryAttachmentKind : Option ReferenceKind
  secondaryAttachmentKind : Option ReferenceKind
  primaryOperationKind : Option ReferenceKind
  secondaryOperationKind : Option ReferenceKind
  primaryAttachmentValue : Option ReferenceValue
  secondaryAttachmentValue : Option ReferenceValue
  primaryOperationValue : Option ReferenceValue
  secondaryOperationValue : Option ReferenceValue
  primaryAttachmentPosition : Option Position
  secondaryAttachmentPosition : Option Position
  primaryOperationPosition : Option Position
  secondaryOperationPosition : Option Position
  primaryAttachmentMalformed : Bool
  secondaryAttachmentMalformed : Bool
  primaryOperationMalformed : Bool
  secondaryOperationMalformed : Bool
  deriving DecidableEq, Inhabited, Repr

def State.attachmentObserved (state : State) : CallbackID → Bool
  | .primary => state.primaryAttachmentObserved
  | .secondary => state.secondaryAttachmentObserved

def State.operationObserved (state : State) : OperationID → Bool
  | .primary => state.primaryOperationObserved
  | .secondary => state.secondaryOperationObserved

def State.callbackOperation (state : State) : CallbackID → Option OperationID
  | .primary => state.primaryCallbackOperation
  | .secondary => state.secondaryCallbackOperation

def State.callbackHandler (state : State) : CallbackID → Option HandlerID
  | .primary => state.primaryCallbackHandler
  | .secondary => state.secondaryCallbackHandler

def State.operationHandler (state : State) : OperationID → Option HandlerID
  | .primary => state.primaryOperationHandler
  | .secondary => state.secondaryOperationHandler

def State.attachmentKind (state : State) : CallbackID → Option ReferenceKind
  | .primary => state.primaryAttachmentKind
  | .secondary => state.secondaryAttachmentKind

def State.operationKind (state : State) : OperationID → Option ReferenceKind
  | .primary => state.primaryOperationKind
  | .secondary => state.secondaryOperationKind

def State.attachmentValue (state : State) : CallbackID → Option ReferenceValue
  | .primary => state.primaryAttachmentValue
  | .secondary => state.secondaryAttachmentValue

def State.operationValue (state : State) : OperationID → Option ReferenceValue
  | .primary => state.primaryOperationValue
  | .secondary => state.secondaryOperationValue

def State.attachmentPosition (state : State) : CallbackID → Option Position
  | .primary => state.primaryAttachmentPosition
  | .secondary => state.secondaryAttachmentPosition

def State.operationPosition (state : State) : OperationID → Option Position
  | .primary => state.primaryOperationPosition
  | .secondary => state.secondaryOperationPosition

def State.attachmentMalformed (state : State) : CallbackID → Bool
  | .primary => state.primaryAttachmentMalformed
  | .secondary => state.secondaryAttachmentMalformed

def State.operationMalformed (state : State) : OperationID → Bool
  | .primary => state.primaryOperationMalformed
  | .secondary => state.secondaryOperationMalformed

def State.setAttachment (state : State) : CallbackID → HandlerID → ReferenceKind →
    ReferenceValue → Position → Bool → State
  | .primary, handler, kind, value, position, malformed => {
      state with
      primaryAttachmentObserved := true
      primaryCallbackHandler := some handler
      primaryAttachmentKind := some kind
      primaryAttachmentValue := some value
      primaryAttachmentPosition := some position
      primaryAttachmentMalformed := malformed
    }
  | .secondary, handler, kind, value, position, malformed => {
      state with
      secondaryAttachmentObserved := true
      secondaryCallbackHandler := some handler
      secondaryAttachmentKind := some kind
      secondaryAttachmentValue := some value
      secondaryAttachmentPosition := some position
      secondaryAttachmentMalformed := malformed
    }

def State.setOperationStart (state : State) : CallbackID → OperationID → HandlerID →
    ReferenceKind → ReferenceValue → Position → Bool → State
  | callback, .primary, handler, kind, value, position, malformed =>
      let state := match callback with
        | .primary => { state with primaryCallbackOperation := some .primary }
        | .secondary => { state with secondaryCallbackOperation := some .primary }
      {
        state with
        primaryOperationObserved := true
        primaryOperationHandler := some handler
        primaryOperationKind := some kind
        primaryOperationValue := some value
        primaryOperationPosition := some position
        primaryOperationMalformed := malformed
      }
  | callback, .secondary, handler, kind, value, position, malformed =>
      let state := match callback with
        | .primary => { state with primaryCallbackOperation := some .secondary }
        | .secondary => { state with secondaryCallbackOperation := some .secondary }
      {
        state with
        secondaryOperationObserved := true
        secondaryOperationHandler := some handler
        secondaryOperationKind := some kind
        secondaryOperationValue := some value
        secondaryOperationPosition := some position
        secondaryOperationMalformed := malformed
      }

def callbackIDs : List CallbackID := [.primary, .secondary]
def operationIDs : List OperationID := [.primary, .secondary]
def handlerIDs : List HandlerID := [.primary, .secondary]
def referenceKinds : List ReferenceKind := [.event, .request]
def referenceValues : List ReferenceValue := [.workflowStarted, .optionsUpdated, .unrelated]
def positions : List Position := [.first, .second, .third]
def malformedValues : List Bool := [false, true]

theorem callback_mem (callback : CallbackID) : callback ∈ callbackIDs := by cases callback <;> simp [callbackIDs]
theorem operation_mem (operation : OperationID) : operation ∈ operationIDs := by cases operation <;> simp [operationIDs]
theorem handler_mem (handler : HandlerID) : handler ∈ handlerIDs := by cases handler <;> simp [handlerIDs]
theorem referenceKind_mem (kind : ReferenceKind) : kind ∈ referenceKinds := by cases kind <;> simp [referenceKinds]
theorem referenceValue_mem (value : ReferenceValue) : value ∈ referenceValues := by cases value <;> simp [referenceValues]
theorem position_mem (position : Position) : position ∈ positions := by cases position <;> simp [positions]
theorem malformed_mem (malformed : Bool) : malformed ∈ malformedValues := by cases malformed <;> simp [malformedValues]

def referenceConsistentForB (state : State) (callback : CallbackID) : Bool :=
  if !state.attachmentObserved callback then true
  else match state.callbackOperation callback with
    | none => true
    | some operation =>
        if !state.operationObserved operation then true
        else
          !state.attachmentMalformed callback && !state.operationMalformed operation &&
          state.callbackHandler callback == state.operationHandler operation &&
          state.attachmentKind callback == state.operationKind operation &&
          state.attachmentValue callback == state.operationValue operation &&
          match state.attachmentPosition callback, state.operationPosition operation with
            | some attachment, some started => atOrAfterB started attachment
            | _, _ => false

def referenceConsistencyB (state : State) : Bool :=
  callbackIDs.all (referenceConsistentForB state)

def referenceReadyForB (state : State) (callback : CallbackID) : Bool :=
  state.attachmentObserved callback &&
    match state.callbackOperation callback with
      | some operation => state.operationObserved operation
      | none => false

def referenceReadyB (state : State) : Bool := callbackIDs.any (referenceReadyForB state)

def ReferenceConsistency (state : State) : Prop := referenceConsistencyB state = true
def ReferenceQualified (state : State) : Prop :=
  referenceReadyB state = true ∧ ReferenceConsistency state

instance (state : State) : Decidable (ReferenceConsistency state) := by
  unfold ReferenceConsistency
  infer_instance

instance (state : State) : Decidable (ReferenceQualified state) := by
  unfold ReferenceQualified
  infer_instance

inductive Action where
  | observeAttachment (callback : CallbackID) (handler : HandlerID) (kind : ReferenceKind)
      (value : ReferenceValue) (position : Position) (malformed : Bool)
  | observeOperationStart (callback : CallbackID) (operation : OperationID) (handler : HandlerID)
      (kind : ReferenceKind) (value : ReferenceValue) (position : Position) (malformed : Bool)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryAttachmentObserved := false
  secondaryAttachmentObserved := false
  primaryOperationObserved := false
  secondaryOperationObserved := false
  primaryCallbackOperation := none
  secondaryCallbackOperation := none
  primaryCallbackHandler := none
  secondaryCallbackHandler := none
  primaryOperationHandler := none
  secondaryOperationHandler := none
  primaryAttachmentKind := none
  secondaryAttachmentKind := none
  primaryOperationKind := none
  secondaryOperationKind := none
  primaryAttachmentValue := none
  secondaryAttachmentValue := none
  primaryOperationValue := none
  secondaryOperationValue := none
  primaryAttachmentPosition := none
  secondaryAttachmentPosition := none
  primaryOperationPosition := none
  secondaryOperationPosition := none
  primaryAttachmentMalformed := false
  secondaryAttachmentMalformed := false
  primaryOperationMalformed := false
  secondaryOperationMalformed := false

def rawNext (state : State) : Action → List State
  | .observeAttachment callback handler kind value position malformed =>
      if state.attachmentObserved callback then []
      else [state.setAttachment callback handler kind value position malformed]
  | .observeOperationStart callback operation handler kind value position malformed =>
      if state.callbackOperation callback != none || state.operationObserved operation then []
      else [state.setOperationStart callback operation handler kind value position malformed]

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter referenceConsistencyB

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

def attachmentActions : List Action := callbackIDs.flatMap fun callback =>
  handlerIDs.flatMap fun handler => referenceKinds.flatMap fun kind =>
    referenceValues.flatMap fun value => positions.flatMap fun position =>
      malformedValues.map fun malformed => .observeAttachment callback handler kind value position malformed

def operationActions : List Action := callbackIDs.flatMap fun callback =>
  operationIDs.flatMap fun operation => handlerIDs.flatMap fun handler =>
    referenceKinds.flatMap fun kind => referenceValues.flatMap fun value =>
      positions.flatMap fun position => malformedValues.map fun malformed =>
        .observeOperationStart callback operation handler kind value position malformed

def actions : List Action := attachmentActions ++ operationActions

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observeAttachment callback handler kind value position malformed =>
      apply List.mem_append_left
      apply List.mem_flatMap.mpr
      refine ⟨callback, callback_mem callback, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨handler, handler_mem handler, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨kind, referenceKind_mem kind, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨value, referenceValue_mem value, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨position, position_mem position, ?_⟩
      exact List.mem_map.mpr ⟨malformed, malformed_mem malformed, rfl⟩
  | observeOperationStart callback operation handler kind value position malformed =>
      apply List.mem_append_right
      apply List.mem_flatMap.mpr
      refine ⟨callback, callback_mem callback, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨operation, operation_mem operation, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨handler, handler_mem handler, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨kind, referenceKind_mem kind, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨value, referenceValue_mem value, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨position, position_mem position, ?_⟩
      exact List.mem_map.mpr ⟨malformed, malformed_mem malformed, rfl⟩

def bounded : BoundedModel model where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    exact action_mem action

abbrev weakenedModel : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := fun state action nextState => nextState ∈ rawNext state action

def weakenedExecutable : ExecutableModel weakenedModel where
  next := rawNext
  next_iff := by intros; rfl

def matchingFinal : State :=
  (initial.setAttachment .primary .primary .event .workflowStarted .first false).setOperationStart
    .primary .primary .primary .event .workflowStarted .second false

def wrongReferenceFinal : State :=
  (initial.setAttachment .primary .primary .event .workflowStarted .first false).setOperationStart
    .primary .primary .secondary .request .optionsUpdated .second false

theorem initialReferenceConsistency : ReferenceConsistency initial := by decide

theorem successorReferenceConsistency {state action nextState}
    (transition : model.Step state action nextState) : ReferenceConsistency nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveReferenceConsistency {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : ReferenceConsistency start) : ReferenceConsistency final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorReferenceConsistency transition)

theorem referenceConsistencySafe : Safety model ReferenceConsistency := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveReferenceConsistency run initialReferenceConsistency

theorem wrongReferenceMutationNegativeControl :
    ¬ReferenceConsistency wrongReferenceFinal := by decide

end Umpire3.Temporal.Feature.CallbackReference

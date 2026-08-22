import Temporal.Families.CallbackReference.Feature
import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Feature.CallbackResponse

inductive CallbackID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive OperationID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive DeliveryID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive ResponseKind where
  | asyncSuccess
  | failure
  deriving BEq, DecidableEq, Inhabited, Repr

inductive Fingerprint where
  | accepted
  | conflicting
  deriving BEq, DecidableEq, Inhabited, Repr

abbrev Position := Umpire3.Temporal.Feature.CallbackReference.Position

def afterB (later earlier : Position) : Bool := earlier.rank < later.rank

structure AcceptedResponse where
  kind : ResponseKind
  fingerprint : Fingerprint
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  primaryRegistrationObserved : Bool
  secondaryRegistrationObserved : Bool
  primaryDeliveryCallback : Option CallbackID
  secondaryDeliveryCallback : Option CallbackID
  primaryCallbackOperation : Option OperationID
  secondaryCallbackOperation : Option OperationID
  primarySettlementPosition : Option Position
  secondarySettlementPosition : Option Position
  primaryOperationTerminal : Bool
  secondaryOperationTerminal : Bool
  primaryAcceptedResponse : Option AcceptedResponse
  secondaryAcceptedResponse : Option AcceptedResponse
  primaryResponsePosition : Option Position
  secondaryResponsePosition : Option Position
  primaryConflictObserved : Bool
  secondaryConflictObserved : Bool
  deriving DecidableEq, Inhabited, Repr

def State.registrationObserved (state : State) : DeliveryID → Bool
  | .primary => state.primaryRegistrationObserved
  | .secondary => state.secondaryRegistrationObserved

def State.deliveryCallback (state : State) : DeliveryID → Option CallbackID
  | .primary => state.primaryDeliveryCallback
  | .secondary => state.secondaryDeliveryCallback

def State.callbackOperation (state : State) : CallbackID → Option OperationID
  | .primary => state.primaryCallbackOperation
  | .secondary => state.secondaryCallbackOperation

def State.settlementPosition (state : State) : OperationID → Option Position
  | .primary => state.primarySettlementPosition
  | .secondary => state.secondarySettlementPosition

def State.operationTerminal (state : State) : OperationID → Bool
  | .primary => state.primaryOperationTerminal
  | .secondary => state.secondaryOperationTerminal

def State.acceptedResponse (state : State) : DeliveryID → Option AcceptedResponse
  | .primary => state.primaryAcceptedResponse
  | .secondary => state.secondaryAcceptedResponse

def State.responsePosition (state : State) : DeliveryID → Option Position
  | .primary => state.primaryResponsePosition
  | .secondary => state.secondaryResponsePosition

def State.conflictObserved (state : State) : DeliveryID → Bool
  | .primary => state.primaryConflictObserved
  | .secondary => state.secondaryConflictObserved

def State.register (state : State) : DeliveryID → CallbackID → OperationID → State
  | .primary, callback, operation =>
      let state := match callback with
        | .primary => { state with primaryCallbackOperation := some operation }
        | .secondary => { state with secondaryCallbackOperation := some operation }
      { state with primaryRegistrationObserved := true, primaryDeliveryCallback := some callback }
  | .secondary, callback, operation =>
      let state := match callback with
        | .primary => { state with primaryCallbackOperation := some operation }
        | .secondary => { state with secondaryCallbackOperation := some operation }
      { state with secondaryRegistrationObserved := true, secondaryDeliveryCallback := some callback }

def State.settle (state : State) : OperationID → Position → Bool → State
  | .primary, position, terminal => {
      state with primarySettlementPosition := some position, primaryOperationTerminal := terminal
    }
  | .secondary, position, terminal => {
      state with secondarySettlementPosition := some position, secondaryOperationTerminal := terminal
    }

def State.accept (state : State) : DeliveryID → AcceptedResponse → Position → State
  | .primary, response, position => {
      state with primaryAcceptedResponse := some response, primaryResponsePosition := some position
    }
  | .secondary, response, position => {
      state with secondaryAcceptedResponse := some response, secondaryResponsePosition := some position
    }

def State.markConflict (state : State) : DeliveryID → State
  | .primary => { state with primaryConflictObserved := true }
  | .secondary => { state with secondaryConflictObserved := true }

def callbackIDs : List CallbackID := [.primary, .secondary]
def operationIDs : List OperationID := [.primary, .secondary]
def deliveryIDs : List DeliveryID := [.primary, .secondary]
def responseKinds : List ResponseKind := [.asyncSuccess, .failure]
def fingerprints : List Fingerprint := [.accepted, .conflicting]
def positions : List Position := [.first, .second, .third]
def terminalValues : List Bool := [false, true]

theorem callback_mem (callback : CallbackID) : callback ∈ callbackIDs := by cases callback <;> simp [callbackIDs]
theorem operation_mem (operation : OperationID) : operation ∈ operationIDs := by cases operation <;> simp [operationIDs]
theorem delivery_mem (delivery : DeliveryID) : delivery ∈ deliveryIDs := by cases delivery <;> simp [deliveryIDs]
theorem responseKind_mem (kind : ResponseKind) : kind ∈ responseKinds := by cases kind <;> simp [responseKinds]
theorem fingerprint_mem (fingerprint : Fingerprint) : fingerprint ∈ fingerprints := by cases fingerprint <;> simp [fingerprints]
theorem position_mem (position : Position) : position ∈ positions := by cases position <;> simp [positions]
theorem terminal_mem (terminal : Bool) : terminal ∈ terminalValues := by cases terminal <;> simp [terminalValues]

def responseConsistentForB (state : State) (delivery : DeliveryID) : Bool :=
  if !state.registrationObserved delivery then true
  else match state.acceptedResponse delivery with
    | none => true
    | some _ =>
        !state.conflictObserved delivery &&
        match state.deliveryCallback delivery with
          | none => false
          | some callback => match state.callbackOperation callback with
            | none => false
            | some operation =>
                match state.settlementPosition operation, state.responsePosition delivery with
                  | some settled, some responded =>
                      !afterB responded settled ||
                        state.operationTerminal operation
                  | none, some _ => true
                  | _, _ => false

def responseConsistencyB (state : State) : Bool :=
  deliveryIDs.all (responseConsistentForB state)

def responseReadyB (state : State) : Bool :=
  deliveryIDs.any fun delivery => state.registrationObserved delivery &&
    (state.acceptedResponse delivery).isSome

def ResponseConsistency (state : State) : Prop := responseConsistencyB state = true
def ResponseQualified (state : State) : Prop :=
  responseReadyB state = true ∧ ResponseConsistency state

instance (state : State) : Decidable (ResponseConsistency state) := by
  unfold ResponseConsistency
  infer_instance

instance (state : State) : Decidable (ResponseQualified state) := by
  unfold ResponseQualified
  infer_instance

inductive Action where
  | register (delivery : DeliveryID) (callback : CallbackID) (operation : OperationID)
  | settleOperation (operation : OperationID) (position : Position) (terminal : Bool)
  | recordResponse (delivery : DeliveryID) (kind : ResponseKind)
      (fingerprint : Fingerprint) (position : Position)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  primaryRegistrationObserved := false
  secondaryRegistrationObserved := false
  primaryDeliveryCallback := none
  secondaryDeliveryCallback := none
  primaryCallbackOperation := none
  secondaryCallbackOperation := none
  primarySettlementPosition := none
  secondarySettlementPosition := none
  primaryOperationTerminal := false
  secondaryOperationTerminal := false
  primaryAcceptedResponse := none
  secondaryAcceptedResponse := none
  primaryResponsePosition := none
  secondaryResponsePosition := none
  primaryConflictObserved := false
  secondaryConflictObserved := false

def rawNext (state : State) : Action → List State
  | .register delivery callback operation =>
      if state.registrationObserved delivery || state.callbackOperation callback != none then []
      else [state.register delivery callback operation]
  | .settleOperation operation position terminal =>
      if state.settlementPosition operation != none then []
      else [state.settle operation position terminal]
  | .recordResponse delivery kind fingerprint position =>
      if !state.registrationObserved delivery then []
      else
        let response : AcceptedResponse := { kind, fingerprint }
        match state.acceptedResponse delivery with
          | none => [state.accept delivery response position]
          | some accepted =>
              if accepted == response then [state]
              else [state.markConflict delivery]

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter responseConsistencyB

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

def registerActions : List Action := deliveryIDs.flatMap fun delivery =>
  callbackIDs.flatMap fun callback => operationIDs.map fun operation =>
    .register delivery callback operation

def settlementActions : List Action := operationIDs.flatMap fun operation =>
  positions.flatMap fun position => terminalValues.map fun terminal =>
    .settleOperation operation position terminal

def responseActions : List Action := deliveryIDs.flatMap fun delivery =>
  responseKinds.flatMap fun kind => fingerprints.flatMap fun fingerprint =>
    positions.map fun position => .recordResponse delivery kind fingerprint position

def actions : List Action := registerActions ++ settlementActions ++ responseActions

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | register delivery callback operation =>
      apply List.mem_append_left
      apply List.mem_append_left
      apply List.mem_flatMap.mpr
      refine ⟨delivery, delivery_mem delivery, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨callback, callback_mem callback, ?_⟩
      exact List.mem_map.mpr ⟨operation, operation_mem operation, rfl⟩
  | settleOperation operation position terminal =>
      apply List.mem_append_left
      apply List.mem_append_right
      apply List.mem_flatMap.mpr
      refine ⟨operation, operation_mem operation, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨position, position_mem position, ?_⟩
      exact List.mem_map.mpr ⟨terminal, terminal_mem terminal, rfl⟩
  | recordResponse delivery kind fingerprint position =>
      apply List.mem_append_right
      apply List.mem_flatMap.mpr
      refine ⟨delivery, delivery_mem delivery, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨kind, responseKind_mem kind, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨fingerprint, fingerprint_mem fingerprint, ?_⟩
      exact List.mem_map.mpr ⟨position, position_mem position, rfl⟩

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

def consistentFinal : State :=
  ((initial.register .primary .primary .primary).settle .primary .second true).accept
    .primary { kind := .asyncSuccess, fingerprint := .accepted } .third

def conflictingFinal : State :=
  ((initial.register .primary .primary .primary).accept
    .primary { kind := .asyncSuccess, fingerprint := .accepted } .second).markConflict .primary

def lateNonTerminalFinal : State :=
  ((initial.register .primary .primary .primary).settle .primary .second false).accept
    .primary { kind := .asyncSuccess, fingerprint := .accepted } .third

theorem initialResponseConsistency : ResponseConsistency initial := by decide

theorem successorResponseConsistency {state action nextState}
    (transition : model.Step state action nextState) : ResponseConsistency nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveResponseConsistency {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : ResponseConsistency start) : ResponseConsistency final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorResponseConsistency transition)

theorem responseConsistencySafe : Safety model ResponseConsistency := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveResponseConsistency run initialResponseConsistency

theorem conflictingResponseMutationNegativeControl :
    ¬ResponseConsistency conflictingFinal := by decide

theorem lateNonTerminalMutationNegativeControl :
    ¬ResponseConsistency lateNonTerminalFinal := by decide

end Umpire3.Temporal.Feature.CallbackResponse

import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.WorkflowLineage

inductive RunID where
  | primary
  | secondary
  deriving BEq, DecidableEq, Inhabited, Repr

inductive LineageKind where
  | continuation
  | reset
  deriving BEq, DecidableEq, Inhabited, Repr

structure RunEvidence where
  observed : Bool
  kind : Option LineageKind
  predecessor : Option RunID
  original : Option RunID
  first : Option RunID
  deriving DecidableEq, Inhabited, Repr

structure State where
  primary : RunEvidence
  secondary : RunEvidence
  deriving DecidableEq, Inhabited, Repr

def State.evidence (state : State) : RunID → RunEvidence
  | .primary => state.primary
  | .secondary => state.secondary

def State.setEvidence (state : State) : RunID → RunEvidence → State
  | .primary, evidence => { state with primary := evidence }
  | .secondary, evidence => { state with secondary := evidence }

def runIDs : List RunID := [.primary, .secondary]
def lineageKinds : List LineageKind := [.continuation, .reset]

theorem run_mem (run : RunID) : run ∈ runIDs := by cases run <;> simp [runIDs]
theorem lineageKind_mem (kind : LineageKind) : kind ∈ lineageKinds := by cases kind <;> simp [lineageKinds]

def continuationConsistentForB (state : State) (run : RunID) : Bool :=
  let evidence := state.evidence run
  if !evidence.observed || evidence.kind != some .continuation then true
  else match evidence.predecessor, evidence.original, evidence.first with
    | some predecessor, some original, some first =>
        predecessor != run && original == run && first == predecessor
    | _, _, _ => false

def resetConsistentForB (state : State) (run : RunID) : Bool :=
  let evidence := state.evidence run
  if !evidence.observed || evidence.kind != some .reset then true
  else match evidence.predecessor, evidence.original, evidence.first with
    | some predecessor, some original, some first =>
        predecessor != run && original == predecessor && first == predecessor
    | _, _, _ => false

def continuationConsistencyB (state : State) : Bool :=
  runIDs.all (continuationConsistentForB state)

def resetConsistencyB (state : State) : Bool := runIDs.all (resetConsistentForB state)

def lineageConsistencyB (state : State) : Bool :=
  continuationConsistencyB state && resetConsistencyB state

def continuationReadyB (state : State) : Bool :=
  runIDs.any fun run =>
    let evidence := state.evidence run
    evidence.observed && evidence.kind == some .continuation

def resetReadyB (state : State) : Bool :=
  runIDs.any fun run =>
    let evidence := state.evidence run
    evidence.observed && evidence.kind == some .reset

def ContinuationLineage (state : State) : Prop := continuationConsistencyB state = true
def ResetLineage (state : State) : Prop := resetConsistencyB state = true
def ContinuationQualified (state : State) : Prop :=
  continuationReadyB state = true ∧ ContinuationLineage state
def ResetQualified (state : State) : Prop := resetReadyB state = true ∧ ResetLineage state

instance (state : State) : Decidable (ContinuationLineage state) := by
  unfold ContinuationLineage
  infer_instance

instance (state : State) : Decidable (ResetLineage state) := by
  unfold ResetLineage
  infer_instance

instance (state : State) : Decidable (ContinuationQualified state) := by
  unfold ContinuationQualified
  infer_instance

instance (state : State) : Decidable (ResetQualified state) := by
  unfold ResetQualified
  infer_instance

inductive Action where
  | observe (run : RunID) (kind : LineageKind) (predecessor : RunID)
      (original : RunID) (first : RunID)
  deriving DecidableEq, Inhabited, Repr

def emptyEvidence : RunEvidence where
  observed := false
  kind := none
  predecessor := none
  original := none
  first := none

def initial : State where
  primary := emptyEvidence
  secondary := emptyEvidence

def rawNext (state : State) : Action → List State
  | .observe run kind predecessor original first =>
      if (state.evidence run).observed then []
      else [state.setEvidence run {
        observed := true
        kind := some kind
        predecessor := some predecessor
        original := some original
        first := some first
      }]

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter lineageConsistencyB

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

def actions : List Action := runIDs.flatMap fun run => lineageKinds.flatMap fun kind =>
  runIDs.flatMap fun predecessor => runIDs.flatMap fun original =>
    runIDs.map fun first => .observe run kind predecessor original first

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observe run kind predecessor original first =>
      apply List.mem_flatMap.mpr
      refine ⟨run, run_mem run, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨kind, lineageKind_mem kind, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨predecessor, run_mem predecessor, ?_⟩
      apply List.mem_flatMap.mpr
      refine ⟨original, run_mem original, ?_⟩
      exact List.mem_map.mpr ⟨first, run_mem first, rfl⟩

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

def continuationFinal : State := initial.setEvidence .secondary {
  observed := true
  kind := some .continuation
  predecessor := some .primary
  original := some .secondary
  first := some .primary
}

def resetFinal : State := initial.setEvidence .secondary {
  observed := true
  kind := some .reset
  predecessor := some .primary
  original := some .primary
  first := some .primary
}

def invalidContinuationFinal : State := initial.setEvidence .secondary {
  observed := true
  kind := some .continuation
  predecessor := some .primary
  original := some .primary
  first := some .primary
}

theorem initialLineageConsistency : lineageConsistencyB initial = true := by decide

theorem successorLineageConsistency {state action nextState}
    (transition : model.Step state action nextState) : lineageConsistencyB nextState = true := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveLineageConsistency {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : lineageConsistencyB start = true) : lineageConsistencyB final = true := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorLineageConsistency transition)

theorem continuationLineageSafe : Safety model ContinuationLineage := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  have consistency := runsPreserveLineageConsistency run initialLineageConsistency
  simp [lineageConsistencyB] at consistency
  exact consistency.1

theorem resetLineageSafe : Safety model ResetLineage := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  have consistency := runsPreserveLineageConsistency run initialLineageConsistency
  simp [lineageConsistencyB] at consistency
  exact consistency.2

theorem invalidContinuationMutationNegativeControl :
    ¬ContinuationLineage invalidContinuationFinal := by decide

end Umpire3.Temporal.Product.WorkflowLineage

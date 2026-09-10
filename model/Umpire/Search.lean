import Umpire.Artifact.Planning
import Umpire.OutcomeClassification

/-! Implementation behind the `Umpire.Search` public facade. -/

namespace Umpire

/-! Incremental, deterministic enumeration through a checked target's semantic relation. -/

def modelValueOrderKey (value : ModelValue) : String :=
  value.definitionId.value ++ "\u001f" ++ value.value

def stepOrderKey
    (result : Step ModelValue ModelValue ModelValue) : String :=
  modelValueOrderKey result.outcome ++ "\u001e" ++
    modelValueOrderKey result.state ++ "\u001e" ++
    String.intercalate "\u001d" (result.facts.map modelValueOrderKey)

/-- A backend exposes a single candidate and continuation per pull; Query owns policy and result
semantics, while implementations own only incremental enumeration state. -/
private inductive PlannerPull (State Candidate : Type) where
  | yield (candidate : Candidate) (nextState : State)
  | complete
  deriving BEq, DecidableEq, Repr

private structure PlannerBackend (Input State Candidate : Type) where
  start : Input → State
  pull : Input → State → PlannerPull State Candidate

/--
The planner-specific kernel view is indexed rather than List-valued. Its proof fields tie every
incremental value to the selected target relation, establish completeness independently of Query's
claim-bearing evidence, and require canonical identity order for unseeded traversal.
-/
structure SearchView (target : QueryModel LawStatement) where
  actionLimit : Nat
  actionAt : Nat → Option ModelValue
  initialLimit : List RoleBinding → Nat
  initialAt : List RoleBinding → Nat → Option ModelValue
  stepLimit : ModelValue → ModelValue → Nat
  stepAt : ModelValue → ModelValue → Nat →
    Option (Step ModelValue ModelValue ModelValue)
  actionSound : ∀ index action, index < actionLimit → actionAt index = some action →
    ∃ state result, target.machine.authoritativeStep state action result
  actionComplete : ∀ state action result,
    target.machine.authoritativeStep state action result →
      ∃ index, index < actionLimit ∧ actionAt index = some action
  initialSound : ∀ setup index state, index < initialLimit setup →
    initialAt setup index = some state → target.machine.authoritativeInitial setup state
  initialComplete : ∀ setup state, target.machine.authoritativeInitial setup state →
    ∃ index, index < initialLimit setup ∧ initialAt setup index = some state
  stepSound : ∀ state action index result, index < stepLimit state action →
    stepAt state action index = some result →
      target.machine.authoritativeStep state action result
  stepComplete : ∀ state action result, target.machine.authoritativeStep state action result →
    ∃ index, index < stepLimit state action ∧ stepAt state action index = some result
  actionOrdered : ∀ first second left right, first < second →
    actionAt first = some left → actionAt second = some right →
      modelValueOrderKey left ≤ modelValueOrderKey right
  initialOrdered : ∀ setup first second left right, first < second →
    initialAt setup first = some left → initialAt setup second = some right →
      modelValueOrderKey left ≤ modelValueOrderKey right
  stepOrdered : ∀ state action first second left right, first < second →
    stepAt state action first = some left → stepAt state action second = some right →
      stepOrderKey left ≤ stepOrderKey right

/-- Canonical ordering obligations for the finite lists already owned by a checked target. -/
structure FiniteKernelOrder
    (target : QueryModel LawStatement)
    (evidence : FiniteCompletenessEvidence LawStatement target) where
  action : evidence.actions.Pairwise fun left right =>
    modelValueOrderKey left ≤ modelValueOrderKey right
  initial : ∀ setup, (target.machine.initialStates setup).Pairwise fun left right =>
    modelValueOrderKey left ≤ modelValueOrderKey right
  step : ∀ state action, (target.machine.steps state action).Pairwise fun left right =>
    stepOrderKey left ≤ stepOrderKey right

/-- Derive indexed planning from the target's sound and complete finite list interface. -/
def SearchView.ofFinite
    (evidence : FiniteCompletenessEvidence LawStatement target)
    (order : FiniteKernelOrder target evidence) : SearchView target := {
  actionLimit := evidence.actions.length
  actionAt := fun index => evidence.actions[index]?
  initialLimit := fun setup => (target.machine.initialStates setup).length
  initialAt := fun setup index => (target.machine.initialStates setup)[index]?
  stepLimit := fun state action => (target.machine.steps state action).length
  stepAt := fun state action index => (target.machine.steps state action)[index]?
  actionSound := by
    intro index action _ emitted
    apply evidence.actionSound action
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    rw [List.mem_iff_getElem]
    exact ⟨index, inBounds, selected⟩
  actionComplete := by
    intro state action result admitted
    have member := evidence.actionComplete state action result admitted
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    exact ⟨index, inBounds, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  initialSound := by
    intro setup index state _ emitted
    apply target.machine.initialSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    rw [List.mem_iff_getElem]
    exact ⟨index, inBounds, selected⟩
  initialComplete := by
    intro setup state admitted
    have member := target.machine.initialComplete setup state admitted
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    exact ⟨index, inBounds, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  stepSound := by
    intro state action index result _ emitted
    apply target.machine.stepSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    rw [List.mem_iff_getElem]
    exact ⟨index, inBounds, selected⟩
  stepComplete := by
    intro state action result admitted
    have member := target.machine.stepComplete state action result admitted
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    exact ⟨index, inBounds, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  actionOrdered := by
    intro first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have ordered := List.pairwise_iff_getElem.mp order.action
      first second firstBound secondBound earlier
    simpa [selectedLeft, selectedRight] using ordered
  initialOrdered := by
    intro setup first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have ordered := List.pairwise_iff_getElem.mp (order.initial setup)
      first second firstBound secondBound earlier
    simpa [selectedLeft, selectedRight] using ordered
  stepOrdered := by
    intro state action first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have ordered := List.pairwise_iff_getElem.mp (order.step state action)
      first second firstBound secondBound earlier
    simpa [selectedLeft, selectedRight] using ordered
}

/-- Derive the indexed Planning view from admitted Query completeness. Callers state only the
canonical ordering obligations; the established finite-kernel implementation remains authoritative. -/
def SearchView.ofCheckedQuery?
    (query : CheckedQuery LawStatement)
    (actionOrdered : ∀ evidence, query.completeness = some evidence →
      evidence.actions.Pairwise fun left right =>
        modelValueOrderKey left ≤ modelValueOrderKey right)
    (initialOrdered : ∀ evidence, query.completeness = some evidence → ∀ setup,
      (query.target.machine.initialStates setup).Pairwise fun left right =>
        modelValueOrderKey left ≤ modelValueOrderKey right)
    (stepOrdered : ∀ evidence, query.completeness = some evidence → ∀ state action,
      (query.target.machine.steps state action).Pairwise fun left right =>
        stepOrderKey left ≤ stepOrderKey right) :
    Option (SearchView query.target) :=
  match evidenceEq : query.completeness with
  | none => none
  | some evidence =>
      some (.ofFinite evidence {
        action := actionOrdered evidence evidenceEq
        initial := initialOrdered evidence evidenceEq
        step := stepOrdered evidence evidenceEq
      })

inductive FinitePlannerAdmissionErrorKind where
  | targetMismatch
  | missingFiniteCompleteness
  | noncanonicalActionOrder
  | noncanonicalInitialOrder
  | noncanonicalStepOrder
  deriving BEq, DecidableEq, Repr

/-- Planner-kernel admission failures identify the selected Target and the offending finite rows. -/
structure FiniteSearchAdmissionError where
  kind : FinitePlannerAdmissionErrorKind
  expectedTarget : DefinitionId
  actualTarget : DefinitionId
  relatedDefinitionIds : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

private def modelValueLe (left right : ModelValue) : Bool :=
  decide (modelValueOrderKey left ≤ modelValueOrderKey right)

private def stepLe
    (left right : Step ModelValue ModelValue ModelValue) : Bool :=
  decide (stepOrderKey left ≤ stepOrderKey right)

private theorem modelValueLe_trans (a b c : ModelValue) :
    modelValueLe a b → modelValueLe b c → modelValueLe a c := by
  simp only [modelValueLe, decide_eq_true_eq]
  exact fun ab bc => String.le_trans ab bc

private theorem modelValueLe_total (a b : ModelValue) :
    modelValueLe a b || modelValueLe b a := by
  simp only [modelValueLe, Bool.or_eq_true, decide_eq_true_eq]
  exact String.le_total _ _

private theorem transitionResultLe_trans
    (a b c : Step ModelValue ModelValue ModelValue) :
    stepLe a b → stepLe b c → stepLe a c := by
  simp only [stepLe, decide_eq_true_eq]
  exact fun ab bc => String.le_trans ab bc

private theorem transitionResultLe_total
    (a b : Step ModelValue ModelValue ModelValue) :
    stepLe a b || stepLe b a := by
  simp only [stepLe, Bool.or_eq_true, decide_eq_true_eq]
  exact String.le_total _ _

private theorem modelValueBeqSelf (value : ModelValue) : (value == value) = true := by
  cases value with
  | mk definitionId text =>
    cases definitionId with
    | mk id =>
      change (id == id && text == text) = true
      simp only [beq_self_eq_true, Bool.and_self]

private theorem modelValueListBeqSelf (values : List ModelValue) :
    (values == values) = true := by
  induction values with
  | nil => rfl
  | cons head tail ih =>
    change (head == head && tail == tail) = true
    rw [modelValueBeqSelf, ih]
    rfl

private theorem stepBeqSelf
    (result : Step ModelValue ModelValue ModelValue) :
    (result == result) = true := by
  cases result with
  | mk outcome state observations =>
    change (outcome == outcome && (state == state && observations == observations)) = true
    rw [modelValueBeqSelf, modelValueBeqSelf, modelValueListBeqSelf]
    rfl

private theorem stepListBeqSelf
    (results : List (Step ModelValue ModelValue ModelValue)) :
    (results == results) = true := by
  induction results with
  | nil => rfl
  | cons head tail ih =>
    change (head == head && tail == tail) = true
    rw [stepBeqSelf, ih]
    rfl

/-- Derive the indexed kernel with Definition-ID ordering owned by Planning. The separate adapter
is required because finite completeness does not constrain arbitrary out-of-domain selectors. -/
private def SearchView.ofCanonicalFinite
    (evidence : FiniteCompletenessEvidence LawStatement target) :
    SearchView target :=
  let actions := evidence.actions.mergeSort modelValueLe
  {
  actionLimit := actions.length
  actionAt := fun index => actions[index]?
  initialLimit := fun setup => (target.machine.initialStates setup).length
  initialAt := fun setup index =>
    (target.machine.initialStates setup |>.mergeSort modelValueLe)[index]?
  stepLimit := fun state action => (target.machine.steps state action).length
  stepAt := fun state action index =>
    (target.machine.steps state action |>.mergeSort stepLe)[index]?
  actionSound := by
    intro index action _ emitted
    apply evidence.actionSound action
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    have member : action ∈ evidence.actions.mergeSort modelValueLe := by
      rw [List.mem_iff_getElem]
      exact ⟨index, inBounds, selected⟩
    simpa [List.mem_iff_getElem] using List.mem_mergeSort.mp member
  actionComplete := by
    intro state action result admitted
    have member : action ∈ evidence.actions.mergeSort modelValueLe :=
      List.mem_mergeSort.mpr (evidence.actionComplete state action result admitted)
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    exact ⟨index, inBounds, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  initialSound := by
    intro setup index state _ emitted
    apply target.machine.initialSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    have member : state ∈ (target.machine.initialStates setup |>.mergeSort modelValueLe) := by
      rw [List.mem_iff_getElem]
      exact ⟨index, inBounds, selected⟩
    exact List.mem_mergeSort.mp member
  initialComplete := by
    intro setup state admitted
    have member : state ∈ (target.machine.initialStates setup |>.mergeSort modelValueLe) :=
      List.mem_mergeSort.mpr (target.machine.initialComplete setup state admitted)
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    have originalBound : index < (target.machine.initialStates setup).length := by
      simpa using inBounds
    exact ⟨index, originalBound, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  stepSound := by
    intro state action index result _ emitted
    apply target.machine.stepSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    have member : result ∈
        (target.machine.steps state action |>.mergeSort stepLe) := by
      rw [List.mem_iff_getElem]
      exact ⟨index, inBounds, selected⟩
    exact List.mem_mergeSort.mp member
  stepComplete := by
    intro state action result admitted
    have member : result ∈
        (target.machine.steps state action |>.mergeSort stepLe) :=
      List.mem_mergeSort.mpr (target.machine.stepComplete state action result admitted)
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    have originalBound : index < (target.machine.steps state action).length := by
      simpa using inBounds
    exact ⟨index, originalBound, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  actionOrdered := by
    intro first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have ordered := List.pairwise_iff_getElem.mp
      (List.pairwise_mergeSort modelValueLe_trans modelValueLe_total evidence.actions)
      first second firstBound secondBound earlier
    simpa [actions, modelValueLe, selectedLeft, selectedRight] using ordered
  initialOrdered := by
    intro setup first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have ordered := List.pairwise_iff_getElem.mp
      (List.pairwise_mergeSort modelValueLe_trans modelValueLe_total
        (target.machine.initialStates setup))
      first second firstBound secondBound earlier
    simpa [modelValueLe, selectedLeft, selectedRight] using ordered
  stepOrdered := by
    intro state action first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have ordered := List.pairwise_iff_getElem.mp
      (List.pairwise_mergeSort transitionResultLe_trans transitionResultLe_total
        (target.machine.steps state action))
      first second firstBound secondBound earlier
    simpa [stepLe, selectedLeft, selectedRight] using ordered
}

private def finiteOrderError?
    (query : CheckedQuery LawStatement)
    (evidence : FiniteCompletenessEvidence LawStatement query.target) :
    Option FiniteSearchAdmissionError :=
  let targetId := query.target.id
  if evidence.actions.mergeSort modelValueLe != evidence.actions then
    some { kind := .noncanonicalActionOrder, expectedTarget := targetId, actualTarget := targetId }
  else
    match query.target.machine.vocabulary with
    | .missing | .incomplete _ => some {
        kind := .missingFiniteCompleteness
        expectedTarget := targetId
        actualTarget := targetId
      }
    | .complete domain =>
        match domain.setups.find? fun setup =>
            (query.target.machine.initialStates setup |>.mergeSort modelValueLe) !=
              query.target.machine.initialStates setup with
        | some setup => some {
            kind := .noncanonicalInitialOrder
            expectedTarget := targetId
            actualTarget := targetId
            relatedDefinitionIds := setup.map RoleBinding.role
          }
        | none =>
            let pairs := domain.states.flatMap fun state => domain.actions.map fun action =>
              (state, action)
            match pairs.find? fun pair =>
                (query.target.machine.steps pair.1 pair.2 |>.mergeSort stepLe) !=
                  query.target.machine.steps pair.1 pair.2 with
            | some (state, action) => some {
                kind := .noncanonicalStepOrder
                expectedTarget := targetId
                actualTarget := targetId
                relatedDefinitionIds := [state.definitionId, action.definitionId]
              }
            | none => none

/-- Admit the planner view from one checked Query, rejecting identity, completeness, or order drift. -/
def SearchView.ofCheckedQuery
    (expectedTarget : DefinitionId)
    (query : CheckedQuery LawStatement) :
    Except FiniteSearchAdmissionError (SearchView query.target) :=
  if expectedTarget != query.target.id then
    .error {
      kind := .targetMismatch
      expectedTarget
      actualTarget := query.target.id
      relatedDefinitionIds := [expectedTarget, query.target.id]
    }
  else
    match query.completeness with
    | none => .error {
        kind := .missingFiniteCompleteness
        expectedTarget
        actualTarget := query.target.id
        relatedDefinitionIds := [query.target.id, query.target.machine.metadata.id]
      }
    | some evidence =>
        match finiteOrderError? query evidence with
        | some error => .error error
        | none => .ok (.ofCanonicalFinite evidence)

/-- Prove checked-query planner admission from the same explicit finite-order evidence consumed by
the admission checker. This keeps successful extraction kernel-checked while the `Except` result
continues to expose target, completeness, and canonical-order failures to ordinary callers. -/
theorem SearchView.ofCheckedQuery_isSome
    (expectedTarget : DefinitionId)
    (query : CheckedQuery LawStatement)
    (evidence : FiniteCompletenessEvidence LawStatement query.target)
    (targetMatches : (expectedTarget != query.target.id) = false)
    (completeness : query.completeness = some evidence)
    (behaviorDomainComplete : ∃ domain,
      query.target.machine.vocabulary = .complete domain)
    (actionCanonical : evidence.actions.mergeSort (fun left right =>
      decide (modelValueOrderKey left ≤ modelValueOrderKey right)) = evidence.actions)
    (initialCanonical : ∀ setup,
      (query.target.machine.initialStates setup).mergeSort (fun left right =>
        decide (modelValueOrderKey left ≤ modelValueOrderKey right)) =
      query.target.machine.initialStates setup)
    (stepCanonical : ∀ state action,
      (query.target.machine.steps state action).mergeSort (fun left right =>
        decide (stepOrderKey left ≤ stepOrderKey right)) =
      query.target.machine.steps state action) :
    (SearchView.ofCheckedQuery expectedTarget query).toOption.isSome = true := by
  rcases behaviorDomainComplete with ⟨domain, vocabulary⟩
  change evidence.actions.mergeSort modelValueLe = evidence.actions at actionCanonical
  change ∀ setup, (query.target.machine.initialStates setup).mergeSort modelValueLe =
    query.target.machine.initialStates setup at initialCanonical
  change ∀ state action, (query.target.machine.steps state action).mergeSort stepLe =
    query.target.machine.steps state action at stepCanonical
  have actionCanonicalBool :
      (evidence.actions.mergeSort modelValueLe != evidence.actions) = false := by
    rw [actionCanonical]
    change (!(evidence.actions == evidence.actions)) = false
    rw [modelValueListBeqSelf]
    rfl
  have initialCanonicalBool : ∀ setup,
      ((query.target.machine.initialStates setup).mergeSort modelValueLe !=
        query.target.machine.initialStates setup) = false := by
    intro setup
    rw [initialCanonical]
    change (!(query.target.machine.initialStates setup ==
      query.target.machine.initialStates setup)) = false
    rw [modelValueListBeqSelf]
    rfl
  have stepCanonicalBool : ∀ state action,
      ((query.target.machine.steps state action).mergeSort stepLe !=
        query.target.machine.steps state action) = false := by
    intro state action
    rw [stepCanonical]
    change (!(query.target.machine.steps state action ==
      query.target.machine.steps state action)) = false
    rw [stepListBeqSelf]
    rfl
  have findFalse : ∀ {α : Type} (items : List α),
      items.find? (fun _ => false) = none := by
    intro α items
    induction items <;> simp_all
  simp [SearchView.ofCheckedQuery, targetMatches, completeness, finiteOrderError?,
    vocabulary, actionCanonicalBool, initialCanonicalBool, stepCanonicalBool, findFalse]
  rfl

private structure PlannerCursor where
  trace : Scenario.Trace
  nextAction : Nat := 0
  currentAction : Option ModelValue := none
  nextOutcome : Nat := 0
  deriving BEq, DecidableEq, Repr

private structure PurePlannerState where
  targetDepth : Nat := 0
  setupIndex : Nat := 0
  initialIndex : Nat := 0
  activePath : List PlannerCursor := []
  actionDomainPulls : Nat := 0
  initialKernelPulls : Nat := 0
  stepKernelPulls : Nat := 0
  deriving BEq, DecidableEq, Repr

structure SearchStats where
  backendPulls : Nat := 0
  generatedCandidates : Nat := 0
  retainedPendingCandidates : Nat := 0
  peakActiveFrontierDepth : Nat := 0
  actionDomainPulls : Nat := 0
  initialKernelPulls : Nat := 0
  stepKernelPulls : Nat := 0
  deriving BEq, DecidableEq, Repr

inductive PlanningOutcome where
  | found (trace : Scenario.Trace) (reason : SelectionReason)
  | verified
  | noneFound
  | limitReached
  | unsatisfiable
  | neverTriggered
  | stillPending
  | invalid (error : QueryError)
  deriving BEq, DecidableEq, Repr

def PlanningOutcome.name : PlanningOutcome → String
  | .found _ _ => "found"
  | .verified => "verified-within-limits"
  | .noneFound => "none-found"
  | .limitReached => "limit-reached"
  | .unsatisfiable => "unsatisfiable"
  | .neverTriggered => "never-triggered"
  | .stillPending => "still-pending"
  | .invalid _ => "invalid"

private def searchOutcomeConstructorIndex : PlanningOutcome → Nat
  | .found _ _ => 0
  | .verified => 1
  | .noneFound => 2
  | .limitReached => 3
  | .unsatisfiable => 4
  | .invalid _ => 5
  | .neverTriggered => 6
  | .stillPending => 7

/-- Canonical documentation and exact constructor matchers for Planning outcomes. -/
def PlanningOutcome.constructorClassifiers :
    List (OutcomeConstructorClassifier PlanningOutcome) := [
  {
    descriptor := { name := "found", description := "Planning selected one Model Trace." }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 0
  },
  {
    descriptor := {
      name := "verified-within-limits"
      description := "Planning verified the requested universal claim within complete Limits."
    }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 1
  },
  {
    descriptor := {
      name := "none-found"
      description := "Complete bounded search found no matching Model Trace."
    }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 2
  },
  {
    descriptor := {
      name := "limit-reached"
      description := "Planning reached its search Limit before completing the Query."
    }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 3
  },
  {
    descriptor := {
      name := "unsatisfiable"
      description := "The checked Behavior admits no Model Traces."
    }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 4
  },
  {
    descriptor := { name := "invalid", description := "Planning rejected the Query." }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 5
  },
  {
    descriptor := { name := "never-triggered", description := "Admissible traces leave requested triggers unexercised." }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 6
  },
  {
    descriptor := { name := "still-pending", description := "An admitted runtime prefix retains unresolved obligations." }
    accepts := fun outcome => searchOutcomeConstructorIndex outcome == 7
  }
]

/-- Every Planning outcome, including arbitrary payloads, matches exactly one descriptor. -/
theorem PlanningOutcome.constructorClassifiers_exactlyOne :
    OutcomeConstructorClassifiers.ExactlyOne PlanningOutcome.constructorClassifiers
  | .found _ _ => rfl
  | .verified => rfl
  | .noneFound => rfl
  | .limitReached => rfl
  | .unsatisfiable => rfl
  | .invalid _ => rfl
  | .neverTriggered => rfl
  | .stillPending => rfl

structure PlanningResult where
  private mk ::
  outcome : PlanningOutcome
  metadata : PlanningMetadata
  deriving BEq, DecidableEq, Repr

namespace PlanningResult

def isVerified (result : PlanningResult) : Bool :=
  match result.outcome with
  | .verified => result.metadata.completeness.established
  | _ => false

end PlanningResult

private def receiptValue (value : ModelValue) : Lean.Json :=
  .mkObj [("definitionId", .str value.definitionId.value), ("value", .str value.value)]

private def receiptTrace (trace : Scenario.Trace) : Lean.Json :=
  .mkObj [
    ("setup", .arr (trace.setup.map fun binding => .mkObj [
      ("role", .str binding.role.value), ("value", receiptValue binding.value)]).toArray),
    ("initialState", receiptValue trace.trace.initialState),
    ("steps", .arr (trace.trace.steps.map fun step => .mkObj [
      ("selectedAction", receiptValue step.selectedAction),
      ("outcome", receiptValue step.outcome),
      ("state", receiptValue step.state),
      ("facts", .arr (step.facts.map receiptValue).toArray)]).toArray)]

/-- Canonical endpoint receipt binds independent claims, all Query limits and policies, the
assurance method, and exact model paths supporting realized trigger coverage. -/
def canonicalPlanningReceiptJson (result : PlanningResult) : String :=
  let validity := result.metadata.validity
  let satisfiability := match validity.satisfiability with
    | .unknown => "unknown" | .nonempty => "nonempty" | .impossible => "impossible"
  let coverage := match validity.coverage with
    | .unknown => "unknown" | .exercised => "exercised" | .unexercised => "unexercised"
  let answer := match validity.answer with
    | .unknown => "unknown" | .witness => "witness" | .verified => "verified"
    | .counterexample => "counterexample" | .stillPending => "still-pending"
  let limits := result.metadata.completeness.limits
  Lean.Json.compress <| .mkObj [
    ("formatVersion", .str "umpire-planning-receipt/v1"),
    ("query", .str validity.queryMetadata),
    ("ending", .str validity.ending.name),
    ("requireFiring", .bool validity.requireFiring),
    ("satisfiability", .str satisfiability),
    ("coverage", .str coverage),
    ("answer", .str answer),
    ("outcome", .str result.outcome.name),
    ("selectedTrace", match result.outcome with
      | .found trace _ => receiptTrace trace
      | _ => .null),
    ("searchComplete", .bool validity.searchComplete),
    ("searchTermination", .str validity.searchTermination),
    ("requestedTriggers", .arr (validity.requestedTriggers.map fun (propertyId, clauseId) =>
      .mkObj [("propertyId", .str propertyId.value), ("clauseId", .str clauseId.value)]).toArray),
    ("assuranceMethod", .str validity.assuranceMethod),
    ("limits", .arr #[.str (canonicalLimitJson limits.steps),
      .str (canonicalLimitJson limits.actions),
      .str (canonicalLimitJson limits.search)]),
    ("explored", .mkObj [
      ("setups", Lean.toJson result.metadata.explored.setups),
      ("traces", Lean.toJson result.metadata.explored.traces),
      ("transitions", Lean.toJson result.metadata.explored.transitions),
      ("propertyEvaluations", Lean.toJson result.metadata.explored.propertyEvaluations)]),
    ("triggers", .arr (validity.triggers.map fun evidence => .mkObj [
      ("trace", receiptTrace evidence.trace),
      ("propertyId", .str evidence.trigger.propertyId.value),
      ("clauseId", .str evidence.trigger.clauseId.value),
      ("transitionPosition", Lean.toJson evidence.trigger.transitionPosition),
      ("field", .str evidence.trigger.occurrence.field.name),
      ("coordinateUnit", .str evidence.trigger.occurrence.coordinateUnit.name),
      ("coordinate", Lean.toJson evidence.trigger.occurrence.coordinate),
      ("value", evidence.trigger.occurrence.value.map receiptValue |>.getD .null)]).toArray)]

structure PlanResult where
  result : PlanningResult
  artifact : Option Plan
  instrumentation : SearchStats
  deriving BEq, DecidableEq, Repr

/-- Typed failures that can reject a complete planning and Artifact-intent request. -/
inductive PlanningRequestError where
  | knownGap (error : KnownGapError)
  | planRequest (error : ArtifactIntentError)
  deriving BEq, DecidableEq, Repr

private instance : Inhabited (PlannerPull State Candidate) := ⟨.complete⟩

private def evidenceFingerprints
    (query : CheckedQuery LawStatement) : List BehaviorFingerprint :=
  match query.completeness with
  | none => []
  | some evidence => [
      evidence.roleDomainFingerprint,
      evidence.actionDomainFingerprint
    ]

private def planningMetadata
    (query : CheckedQuery LawStatement)
    (explored : ExploredCounts)
    (established : Bool) : PlanningMetadata := {
  explored
  completeness := {
    established
    limits := query.limits
    finiteEvidenceFingerprints := evidenceFingerprints query
  }
}

inductive BoundedTraversalTermination where
  | stopped (trace : Scenario.Trace) (reason : SelectionReason)
  | complete (behaviorAdmitted : Bool)
  | limitReached
  | invalid (error : QueryError)
  deriving BEq, DecidableEq, Repr

def BoundedTraversalTermination.name : BoundedTraversalTermination → String
  | .stopped _ _ => "stopped"
  | .complete true => "exhaustive"
  | .complete false => "unsatisfiable"
  | .limitReached => "limit-reached"
  | .invalid _ => "invalid"

/-- One admitted-candidate fold decision. The traversal owns candidate order and continuation;
consumers can retain only the semantic state their analysis needs. -/
inductive BoundedTraversalStep (State : Type) where
  | continue (state : State)
  | stop (state : State) (trace : Scenario.Trace) (reason : SelectionReason)

/-- Bounded traversal evidence shared by planning and finite semantic analyses. -/
structure BoundedTraversalResult (State : Type) where
  state : State
  termination : BoundedTraversalTermination
  metadata : PlanningMetadata
  instrumentation : SearchStats

/-- The planner-private result finalizer enforces the query's claim strength. A backend completion
signal establishes completeness only for a finite exhaustive query that admitted at least one
behavior trace, and an empty behavior always wins over every attempted terminal claim. -/
private def finalizePlanning
    (query : CheckedQuery LawStatement)
    (explored : ExploredCounts)
    (termination : BoundedTraversalTermination) : PlanningResult :=
  let (outcome, established) :=
    if query.behavior.isUnsatisfiable then
      (PlanningOutcome.unsatisfiable, false)
    else
      match termination with
      | .stopped trace reason => (.found trace reason, false)
      | .limitReached => (.limitReached, false)
      | .invalid error => (.invalid error, false)
      | .complete false => (.unsatisfiable, false)
      | .complete true =>
          if query.policy.strategy != .exhaustive || query.completeness.isNone then
            (.limitReached, false)
          else
            match query.form with
            | .verify _ => (.verified, true)
            | .find _ | .findViolation _ | .pick _ => (.noneFound, true)
  PlanningResult.mk outcome (planningMetadata query explored established)

private def setupLe (left right : List RoleBinding) : Bool :=
  compare left right != .gt

private def rotate (offset : Nat) (items : List α) : List α :=
  if items.isEmpty then
    []
  else
    let pivot := offset % items.length
    items.drop pivot ++ items.take pivot

private def applySeed
    (query : CheckedQuery LawStatement)
    (items : List α) : List α :=
  if query.policy.strategy == .seeded then
    rotate query.policy.seed items
  else
    items

private def candidateSetups (query : CheckedQuery LawStatement) : List (List RoleBinding) :=
  let setups := match query.completeness with
    | some evidence => evidence.roleAssignments
    | none => query.target.resolvedSetups
  applySeed query (setups.mergeSort setupLe)

private def seededIndex
    (query : CheckedQuery LawStatement)
    (limit logicalIndex : Nat) : Nat :=
  if query.policy.strategy == .seeded && limit > 0 then
    (logicalIndex + query.policy.seed % limit) % limit
  else
    logicalIndex

private def maximumDepth (query : CheckedQuery LawStatement) : Nat :=
  Nat.min query.limits.steps.value query.limits.actions.value

private def rootTrace (setup : List RoleBinding) (initialState : ModelValue) : Scenario.Trace := {
  setup
  trace := { initialState, steps := [] }
}

private def appendStep
    (candidate : Scenario.Trace)
    (action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) : Scenario.Trace := {
  candidate with trace := {
    candidate.trace with
    steps := candidate.trace.steps ++ [{
      selectedAction := action
      outcome := result.outcome
      state := result.state
      facts := result.facts
    }]
  }
}

private def currentState (candidate : Scenario.Trace) : ModelValue :=
  match candidate.trace.steps.getLast? with
  | some step => step.state
  | none => candidate.trace.initialState

private partial def nextRoot?
    (query : CheckedQuery LawStatement)
    (kernel : SearchView query.target)
    (state : PurePlannerState) : Option (Scenario.Trace × PurePlannerState) :=
  match (candidateSetups query)[state.setupIndex]? with
  | none => none
  | some setup =>
      let limit := kernel.initialLimit setup
      if state.initialIndex < limit then
        let index := seededIndex query limit state.initialIndex
        let next := {
          state with
          initialIndex := state.initialIndex + 1
          initialKernelPulls := state.initialKernelPulls + 1
        }
        match kernel.initialAt setup index with
        | some initial => some (rootTrace setup initial, next)
        | none => nextRoot? query kernel next
      else
        nextRoot? query kernel {
          state with
          setupIndex := state.setupIndex + 1
          initialIndex := 0
        }

/--
Enumerate one trace at a time. The state retains cursor indexes for the active path, never a queue
of produced candidates or an unconsumed collection of kernel results.
-/
private partial def pullCandidate
    (query : CheckedQuery LawStatement)
    (kernel : SearchView query.target)
    (state : PurePlannerState) : PlannerPull PurePlannerState Scenario.Trace :=
  match state.activePath with
  | [] =>
      match nextRoot? query kernel state with
      | some (root, next) =>
          if state.targetDepth == 0 then
            .yield root next
          else
            pullCandidate query kernel { next with activePath := [{ trace := root }] }
      | none =>
          if state.targetDepth < maximumDepth query then
            pullCandidate query kernel {
              state with
              targetDepth := state.targetDepth + 1
              setupIndex := 0
              initialIndex := 0
              activePath := []
            }
          else
            .complete
  | cursor :: parents =>
      match cursor.currentAction with
      | none =>
          if cursor.nextAction < kernel.actionLimit then
            let index := seededIndex query kernel.actionLimit cursor.nextAction
            let advanced := { cursor with nextAction := cursor.nextAction + 1 }
            let next := {
              state with
              activePath := advanced :: parents
              actionDomainPulls := state.actionDomainPulls + 1
            }
            match kernel.actionAt index with
            | none => pullCandidate query kernel next
            | some action =>
                pullCandidate query kernel {
                  next with
                  activePath := { advanced with currentAction := some action } :: parents
                }
          else
            pullCandidate query kernel { state with activePath := parents }
      | some action =>
          let semanticState := currentState cursor.trace
          let limit := kernel.stepLimit semanticState action
          if cursor.nextOutcome < limit then
            let index := seededIndex query limit cursor.nextOutcome
            let advanced := { cursor with nextOutcome := cursor.nextOutcome + 1 }
            let next := {
              state with
              activePath := advanced :: parents
              stepKernelPulls := state.stepKernelPulls + 1
            }
            match kernel.stepAt semanticState action index with
            | none => pullCandidate query kernel next
            | some result =>
                let child := appendStep cursor.trace action result
                if child.trace.steps.length == state.targetDepth then
                  .yield child next
                else
                  pullCandidate query kernel {
                    next with activePath := { trace := child } :: next.activePath
                  }
          else
            pullCandidate query kernel {
              state with
              activePath := { cursor with currentAction := none, nextOutcome := 0 } :: parents
            }

private def pureSearchBackend
    (query : CheckedQuery LawStatement)
    (kernel : SearchView query.target) :
    PlannerBackend Unit PurePlannerState Scenario.Trace := {
  start := fun _ => {}
  pull := fun _ => pullCandidate query kernel
}

private structure PlanningObservations where
  nonempty : Bool := false
  unresolved : Bool := false
  required : List (DefinitionId × DefinitionId) := []
  triggers : List PlanningTriggerEvidence := []
  counterexample : Option Scenario.Trace := none

private def coverageMet
    (query : CheckedQuery LawStatement) (state : PlanningObservations) : Bool :=
  !query.requireFiring || state.required.all fun (propertyId, clauseId) =>
    state.triggers.any fun evidence =>
      evidence.trigger.propertyId == propertyId && evidence.trigger.clauseId == clauseId

private def observeCandidate
    (query : CheckedQuery LawStatement)
    (state : PlanningObservations)
    (candidate : Scenario.Trace) : Except QueryError (BoundedTraversalStep PlanningObservations) := do
  let mut answers := []
  let mut current : PlanningObservations := { nonempty := true }
  for property in query.form.properties.mergeSort (fun left right =>
      decide (left.id.value ≤ right.id.value)) do
    let input ← (checkPropertyEvaluationInput property candidate.trace).mapError fun error => {
      kind := QueryErrorKind.propertyEvaluationFailure
      definitionId := query.id
      sourcePath := error.sourcePath
      offendingValue := error.kind.name ++ ":" ++ error.offendingValue
      relatedDefinitionIds := DefinitionId.canonicalSet
        (property.id :: property.guardedClauseIds ++ error.relatedDefinitionIds)
    }
    let evaluation := evaluatePropertyEndpoint property input (query.ending == .«partial»)
    answers := answers ++ [evaluation.answer]
    current := { current with
      required := current.required ++ evaluation.requestedTriggers.map (property.id, ·)
      triggers := current.triggers ++ evaluation.realizedTriggers.map ({ trace := candidate, trigger := · }) }
  let violated := answers.contains .violated
  let unresolved := answers.contains .unresolved
  let next := { state with
    nonempty := true
    unresolved := state.unresolved || unresolved
    required := (state.required ++ current.required).eraseDups
    triggers := state.triggers ++ current.triggers
    counterexample := if violated && state.counterexample.isNone then some candidate else state.counterexample }
  match query.form with
  | .verify _ => pure (.continue next)
  | .findViolation _ =>
      if violated then pure (.stop next candidate .violatingCounterexample) else pure (.continue next)
  | .find _ =>
      if !violated && !unresolved && coverageMet query current then
        pure (.stop next candidate .satisfyingWitness)
      else pure (.continue next)
  | .pick _ =>
      if coverageMet query current then pure (.stop next candidate .behaviorSelection)
      else pure (.continue next)

private def noteCandidate
    (candidate : Scenario.Trace)
    (explored : ExploredCounts) : ExploredCounts := {
  explored with
  setups := explored.setups + if candidate.trace.steps.isEmpty then 1 else 0
  traces := explored.traces + 1
  transitions := explored.transitions + if candidate.trace.steps.isEmpty then 0 else 1
}

private def notePropertyEvaluations
    (query : CheckedQuery LawStatement)
    (explored : ExploredCounts) : ExploredCounts := {
  explored with
  propertyEvaluations := explored.propertyEvaluations + query.form.properties.length
}

private def notePull
    (candidate : Scenario.Trace)
    (next : PurePlannerState)
    (instrumentation : SearchStats) : SearchStats := {
  instrumentation with
  backendPulls := instrumentation.backendPulls + 1
  generatedCandidates := instrumentation.generatedCandidates + 1
  retainedPendingCandidates := 0
  peakActiveFrontierDepth := Nat.max instrumentation.peakActiveFrontierDepth
    (candidate.trace.steps.length + 1)
  actionDomainPulls := next.actionDomainPulls
  initialKernelPulls := next.initialKernelPulls
  stepKernelPulls := next.stepKernelPulls
}

private def finish
    (query : CheckedQuery LawStatement)
    (explored : ExploredCounts)
    (instrumentation : SearchStats)
    (termination : BoundedTraversalTermination)
    (knownGaps : KnownGapSet) : PlanResult :=
  let result := finalizePlanning query explored termination
  let artifact := match termination with
    | .stopped trace reason =>
        some (ArtifactPlanning.Internal.artifactOfSelectionWithKnownGaps
          query trace reason explored knownGaps)
    | _ => none
  { result, artifact, instrumentation }

private def traversalMetadata
    (query : CheckedQuery LawStatement)
    (explored : ExploredCounts)
    (termination : BoundedTraversalTermination) : PlanningMetadata :=
  let established := match termination with
    | .complete true => query.policy.strategy == .exhaustive && query.completeness.isSome
    | _ => false
  planningMetadata query explored established

private def traversalResult
    (query : CheckedQuery LawStatement)
    (state : State)
    (termination : BoundedTraversalTermination)
    (explored : ExploredCounts)
    (instrumentation : SearchStats) : BoundedTraversalResult State := {
  state
  termination
  metadata := traversalMetadata query explored termination
  instrumentation
}

private def traverseLoop
    (query : CheckedQuery LawStatement)
    (backend : PlannerBackend Unit PurePlannerState Scenario.Trace)
    (cursor : PurePlannerState)
    (consumerState : State)
    (visit : State → Scenario.Trace → Except QueryError (BoundedTraversalStep State))
    (remaining : Nat)
    (behaviorAdmitted : Bool)
    (explored : ExploredCounts)
    (instrumentation : SearchStats) : BoundedTraversalResult State :=
  match remaining with
  | 0 =>
      match backend.pull () cursor with
      | .complete => traversalResult query consumerState (.complete behaviorAdmitted) explored
          { instrumentation with backendPulls := instrumentation.backendPulls + 1 }
      | .yield _ _ => traversalResult query consumerState .limitReached explored instrumentation
  | remaining + 1 =>
      match backend.pull () cursor with
      | .complete =>
          traversalResult query consumerState (.complete behaviorAdmitted) explored
            { instrumentation with backendPulls := instrumentation.backendPulls + 1 }
      | .yield candidate next =>
          let explored := noteCandidate candidate explored
          let instrumentation := notePull candidate next instrumentation
          if query.behavior.admits candidate &&
              (query.ending != .terminal || query.target.isTerminal (currentState candidate)) then
            let explored := notePropertyEvaluations query explored
            match visit consumerState candidate with
            | .error error =>
                traversalResult query consumerState (.invalid error) explored instrumentation
            | .ok (.stop state trace reason) =>
                traversalResult query state (.stopped trace reason) explored instrumentation
            | .ok (.continue state) =>
                traverseLoop query backend next state visit remaining true explored instrumentation
          else
            traverseLoop query backend next consumerState visit remaining behaviorAdmitted explored
              instrumentation
termination_by remaining

/-- Fold admitted traces through the planner's bounded candidate stream. The private cursor and
backend stay hidden; candidate order, Behavior filtering, accounting, and completion are shared. -/
def traverseBoundedCandidates
    (query : CheckedQuery LawStatement)
    (kernel : SearchView query.target)
    (initial : State)
    (visit : State → Scenario.Trace → Except QueryError (BoundedTraversalStep State)) :
    BoundedTraversalResult State :=
  if query.behavior.isUnsatisfiable then
    traversalResult query initial (.complete false) {} {}
  else
    let backend := pureSearchBackend query kernel
    traverseLoop query backend (backend.start ()) initial visit query.limits.search.value false {} {}

/-- Search a checked Query without invoking runtime, readers, evidence, or promotion behavior. -/
def search
    (query : CheckedQuery LawStatement)
    (kernel : SearchView query.target) : Except KnownGapError PlanResult := do
  let knownGaps ← composeSearchKnownGaps query
  let traversed := traverseBoundedCandidates query kernel {} (observeCandidate query)
  let state := traversed.state
  let searchComplete := match traversed.termination with
    | .complete _ => query.policy.strategy == .exhaustive && query.completeness.isSome
    | _ => false
  let covered := coverageMet query state
  let termination := match state.counterexample, query.form with
    | some trace, .verify _ => .stopped trace .violatingCounterexample
    | _, _ => traversed.termination
  let run := finish query traversed.metadata.explored traversed.instrumentation termination knownGaps
  let outcome := match run.result.outcome with
    | .verified | .noneFound =>
        if state.unresolved then .stillPending
        else if !covered then .neverTriggered else run.result.outcome
    | other => other
  let answer := match outcome with
    | .found _ .violatingCounterexample => PlanningAnswer.counterexample
    | .found _ .satisfyingWitness => .witness
    | .verified => .verified
    | .stillPending => .stillPending
    | _ => .unknown
  let validity : PlanningValidity := {
    satisfiability := if state.nonempty then .nonempty else
      if searchComplete || query.behavior.isUnsatisfiable then .impossible else .unknown
    coverage := if state.nonempty && (coverageMet { query with requireFiring := true } state) then .exercised else
      if searchComplete then .unexercised else .unknown
    answer
    searchComplete
    searchTermination := traversed.termination.name
    requestedTriggers := state.required
    ending := query.ending
    requireFiring := query.requireFiring
    queryMetadata := query.canonicalMetadata
    triggers := state.triggers
  }
  let result := PlanningResult.mk outcome { traversed.metadata with validity }
  pure { run with result }

/--
Plan through the unchanged target kernel, then project checked Artifact intent if one is selected.
-/
def searchWithPlanRequest
    (query : CheckedQuery LawStatement)
    (kernel : SearchView query.target)
    (intent : PlanRequest) : Except PlanningRequestError PlanResult := do
  intent.validateFor query |>.mapError PlanningRequestError.planRequest
  let run ← search query kernel |>.mapError PlanningRequestError.knownGap
  let artifact ← match run.artifact with
    | none => pure none
    | some spec => some <$> (spec.withArtifactIntent query intent |>.mapError
        PlanningRequestError.planRequest)
  pure { run with artifact }

end Umpire

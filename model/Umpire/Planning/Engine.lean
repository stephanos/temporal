import Umpire.Artifact.Planning
import Umpire.SemanticInventory.Types

/-! Implementation behind the `Umpire.Planning` public facade. -/

namespace Umpire

/-! Incremental, deterministic enumeration through a checked target's semantic relation. -/

def modelValueOrderKey (value : ModelValue) : String :=
  value.definitionId.value ++ "\u001f" ++ value.value

def transitionResultOrderKey
    (result : TransitionResult ModelValue ModelValue ModelValue) : String :=
  modelValueOrderKey result.modelOutcome ++ "\u001e" ++
    modelValueOrderKey result.resultingState ++ "\u001e" ++
    String.intercalate "\u001d" (result.observations.map modelValueOrderKey)

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
structure IncrementalPlannerKernel (target : QueryTarget LawStatement) where
  actionLimit : Nat
  actionAt : Nat → Option ModelValue
  initialLimit : List RoleBinding → Nat
  initialAt : List RoleBinding → Nat → Option ModelValue
  stepLimit : ModelValue → ModelValue → Nat
  stepAt : ModelValue → ModelValue → Nat →
    Option (TransitionResult ModelValue ModelValue ModelValue)
  actionSound : ∀ index action, index < actionLimit → actionAt index = some action →
    ∃ state result, target.kernel.authoritativeStep state action result
  actionComplete : ∀ state action result,
    target.kernel.authoritativeStep state action result →
      ∃ index, index < actionLimit ∧ actionAt index = some action
  initialSound : ∀ setup index state, index < initialLimit setup →
    initialAt setup index = some state → target.kernel.authoritativeInitial setup state
  initialComplete : ∀ setup state, target.kernel.authoritativeInitial setup state →
    ∃ index, index < initialLimit setup ∧ initialAt setup index = some state
  stepSound : ∀ state action index result, index < stepLimit state action →
    stepAt state action index = some result →
      target.kernel.authoritativeStep state action result
  stepComplete : ∀ state action result, target.kernel.authoritativeStep state action result →
    ∃ index, index < stepLimit state action ∧ stepAt state action index = some result
  actionOrdered : ∀ first second left right, first < second →
    actionAt first = some left → actionAt second = some right →
      modelValueOrderKey left ≤ modelValueOrderKey right
  initialOrdered : ∀ setup first second left right, first < second →
    initialAt setup first = some left → initialAt setup second = some right →
      modelValueOrderKey left ≤ modelValueOrderKey right
  stepOrdered : ∀ state action first second left right, first < second →
    stepAt state action first = some left → stepAt state action second = some right →
      transitionResultOrderKey left ≤ transitionResultOrderKey right

/-- Canonical ordering obligations for the finite lists already owned by a checked target. -/
structure FiniteKernelOrder
    (target : QueryTarget LawStatement)
    (evidence : FiniteCompletenessEvidence LawStatement target) where
  action : evidence.actions.Pairwise fun left right =>
    modelValueOrderKey left ≤ modelValueOrderKey right
  initial : ∀ setup, (target.kernel.initialStates setup).Pairwise fun left right =>
    modelValueOrderKey left ≤ modelValueOrderKey right
  step : ∀ state action, (target.kernel.steps state action).Pairwise fun left right =>
    transitionResultOrderKey left ≤ transitionResultOrderKey right

/-- Derive indexed planning from the target's sound and complete finite list interface. -/
def IncrementalPlannerKernel.ofFinite
    (evidence : FiniteCompletenessEvidence LawStatement target)
    (order : FiniteKernelOrder target evidence) : IncrementalPlannerKernel target := {
  actionLimit := evidence.actions.length
  actionAt := fun index => evidence.actions[index]?
  initialLimit := fun setup => (target.kernel.initialStates setup).length
  initialAt := fun setup index => (target.kernel.initialStates setup)[index]?
  stepLimit := fun state action => (target.kernel.steps state action).length
  stepAt := fun state action index => (target.kernel.steps state action)[index]?
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
    apply target.kernel.initialSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    rw [List.mem_iff_getElem]
    exact ⟨index, inBounds, selected⟩
  initialComplete := by
    intro setup state admitted
    have member := target.kernel.initialComplete setup state admitted
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    exact ⟨index, inBounds, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  stepSound := by
    intro state action index result _ emitted
    apply target.kernel.stepSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    rw [List.mem_iff_getElem]
    exact ⟨index, inBounds, selected⟩
  stepComplete := by
    intro state action result admitted
    have member := target.kernel.stepComplete state action result admitted
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
def IncrementalPlannerKernel.ofCheckedQuery?
    (query : CheckedQuery LawStatement)
    (actionOrdered : ∀ evidence, query.completeness = some evidence →
      evidence.actions.Pairwise fun left right =>
        modelValueOrderKey left ≤ modelValueOrderKey right)
    (initialOrdered : ∀ evidence, query.completeness = some evidence → ∀ setup,
      (query.target.kernel.initialStates setup).Pairwise fun left right =>
        modelValueOrderKey left ≤ modelValueOrderKey right)
    (stepOrdered : ∀ evidence, query.completeness = some evidence → ∀ state action,
      (query.target.kernel.steps state action).Pairwise fun left right =>
        transitionResultOrderKey left ≤ transitionResultOrderKey right) :
    Option (IncrementalPlannerKernel query.target) :=
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
structure FinitePlannerAdmissionError where
  kind : FinitePlannerAdmissionErrorKind
  expectedTarget : DefinitionId
  actualTarget : DefinitionId
  relatedDefinitionIds : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

private def modelValueLe (left right : ModelValue) : Bool :=
  decide (modelValueOrderKey left ≤ modelValueOrderKey right)

private def transitionResultLe
    (left right : TransitionResult ModelValue ModelValue ModelValue) : Bool :=
  decide (transitionResultOrderKey left ≤ transitionResultOrderKey right)

private theorem modelValueLe_trans (a b c : ModelValue) :
    modelValueLe a b → modelValueLe b c → modelValueLe a c := by
  simp only [modelValueLe, decide_eq_true_eq]
  exact fun ab bc => String.le_trans ab bc

private theorem modelValueLe_total (a b : ModelValue) :
    modelValueLe a b || modelValueLe b a := by
  simp only [modelValueLe, Bool.or_eq_true, decide_eq_true_eq]
  exact String.le_total _ _

private theorem transitionResultLe_trans
    (a b c : TransitionResult ModelValue ModelValue ModelValue) :
    transitionResultLe a b → transitionResultLe b c → transitionResultLe a c := by
  simp only [transitionResultLe, decide_eq_true_eq]
  exact fun ab bc => String.le_trans ab bc

private theorem transitionResultLe_total
    (a b : TransitionResult ModelValue ModelValue ModelValue) :
    transitionResultLe a b || transitionResultLe b a := by
  simp only [transitionResultLe, Bool.or_eq_true, decide_eq_true_eq]
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

private theorem transitionResultBeqSelf
    (result : TransitionResult ModelValue ModelValue ModelValue) :
    (result == result) = true := by
  cases result with
  | mk outcome state observations =>
    change (outcome == outcome && (state == state && observations == observations)) = true
    rw [modelValueBeqSelf, modelValueBeqSelf, modelValueListBeqSelf]
    rfl

private theorem transitionResultListBeqSelf
    (results : List (TransitionResult ModelValue ModelValue ModelValue)) :
    (results == results) = true := by
  induction results with
  | nil => rfl
  | cons head tail ih =>
    change (head == head && tail == tail) = true
    rw [transitionResultBeqSelf, ih]
    rfl

/-- Derive the indexed kernel with Definition-ID ordering owned by Planning. The separate adapter
is required because finite completeness does not constrain arbitrary out-of-domain selectors. -/
private def IncrementalPlannerKernel.ofCanonicalFinite
    (evidence : FiniteCompletenessEvidence LawStatement target) :
    IncrementalPlannerKernel target :=
  let actions := evidence.actions.mergeSort modelValueLe
  {
  actionLimit := actions.length
  actionAt := fun index => actions[index]?
  initialLimit := fun setup => (target.kernel.initialStates setup).length
  initialAt := fun setup index =>
    (target.kernel.initialStates setup |>.mergeSort modelValueLe)[index]?
  stepLimit := fun state action => (target.kernel.steps state action).length
  stepAt := fun state action index =>
    (target.kernel.steps state action |>.mergeSort transitionResultLe)[index]?
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
    apply target.kernel.initialSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    have member : state ∈ (target.kernel.initialStates setup |>.mergeSort modelValueLe) := by
      rw [List.mem_iff_getElem]
      exact ⟨index, inBounds, selected⟩
    exact List.mem_mergeSort.mp member
  initialComplete := by
    intro setup state admitted
    have member : state ∈ (target.kernel.initialStates setup |>.mergeSort modelValueLe) :=
      List.mem_mergeSort.mpr (target.kernel.initialComplete setup state admitted)
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    have originalBound : index < (target.kernel.initialStates setup).length := by
      simpa using inBounds
    exact ⟨index, originalBound, List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  stepSound := by
    intro state action index result _ emitted
    apply target.kernel.stepSound
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    have member : result ∈
        (target.kernel.steps state action |>.mergeSort transitionResultLe) := by
      rw [List.mem_iff_getElem]
      exact ⟨index, inBounds, selected⟩
    exact List.mem_mergeSort.mp member
  stepComplete := by
    intro state action result admitted
    have member : result ∈
        (target.kernel.steps state action |>.mergeSort transitionResultLe) :=
      List.mem_mergeSort.mpr (target.kernel.stepComplete state action result admitted)
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    have originalBound : index < (target.kernel.steps state action).length := by
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
        (target.kernel.initialStates setup))
      first second firstBound secondBound earlier
    simpa [modelValueLe, selectedLeft, selectedRight] using ordered
  stepOrdered := by
    intro state action first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have ordered := List.pairwise_iff_getElem.mp
      (List.pairwise_mergeSort transitionResultLe_trans transitionResultLe_total
        (target.kernel.steps state action))
      first second firstBound secondBound earlier
    simpa [transitionResultLe, selectedLeft, selectedRight] using ordered
}

private def finiteOrderError?
    (query : CheckedQuery LawStatement)
    (evidence : FiniteCompletenessEvidence LawStatement query.target) :
    Option FinitePlannerAdmissionError :=
  let targetId := query.target.id
  if evidence.actions.mergeSort modelValueLe != evidence.actions then
    some { kind := .noncanonicalActionOrder, expectedTarget := targetId, actualTarget := targetId }
  else
    match query.target.kernel.behaviorDomain with
    | .missing | .incomplete _ => some {
        kind := .missingFiniteCompleteness
        expectedTarget := targetId
        actualTarget := targetId
      }
    | .complete domain =>
        match domain.setups.find? fun setup =>
            (query.target.kernel.initialStates setup |>.mergeSort modelValueLe) !=
              query.target.kernel.initialStates setup with
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
                (query.target.kernel.steps pair.1 pair.2 |>.mergeSort transitionResultLe) !=
                  query.target.kernel.steps pair.1 pair.2 with
            | some (state, action) => some {
                kind := .noncanonicalStepOrder
                expectedTarget := targetId
                actualTarget := targetId
                relatedDefinitionIds := [state.definitionId, action.definitionId]
              }
            | none => none

/-- Admit the planner view from one checked Query, rejecting identity, completeness, or order drift. -/
def IncrementalPlannerKernel.ofCheckedQuery
    (expectedTarget : DefinitionId)
    (query : CheckedQuery LawStatement) :
    Except FinitePlannerAdmissionError (IncrementalPlannerKernel query.target) :=
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
        relatedDefinitionIds := [query.target.id, query.target.kernel.metadata.id]
      }
    | some evidence =>
        match finiteOrderError? query evidence with
        | some error => .error error
        | none => .ok (.ofCanonicalFinite evidence)

/-- Prove checked-query planner admission from the same explicit finite-order evidence consumed by
the admission checker. This keeps successful extraction kernel-checked while the `Except` result
continues to expose target, completeness, and canonical-order failures to ordinary callers. -/
theorem IncrementalPlannerKernel.ofCheckedQuery_isSome
    (expectedTarget : DefinitionId)
    (query : CheckedQuery LawStatement)
    (evidence : FiniteCompletenessEvidence LawStatement query.target)
    (targetMatches : (expectedTarget != query.target.id) = false)
    (completeness : query.completeness = some evidence)
    (behaviorDomainComplete : ∃ domain,
      query.target.kernel.behaviorDomain = .complete domain)
    (actionCanonical : evidence.actions.mergeSort (fun left right =>
      decide (modelValueOrderKey left ≤ modelValueOrderKey right)) = evidence.actions)
    (initialCanonical : ∀ setup,
      (query.target.kernel.initialStates setup).mergeSort (fun left right =>
        decide (modelValueOrderKey left ≤ modelValueOrderKey right)) =
      query.target.kernel.initialStates setup)
    (stepCanonical : ∀ state action,
      (query.target.kernel.steps state action).mergeSort (fun left right =>
        decide (transitionResultOrderKey left ≤ transitionResultOrderKey right)) =
      query.target.kernel.steps state action) :
    (IncrementalPlannerKernel.ofCheckedQuery expectedTarget query).toOption.isSome = true := by
  rcases behaviorDomainComplete with ⟨domain, behaviorDomain⟩
  change evidence.actions.mergeSort modelValueLe = evidence.actions at actionCanonical
  change ∀ setup, (query.target.kernel.initialStates setup).mergeSort modelValueLe =
    query.target.kernel.initialStates setup at initialCanonical
  change ∀ state action, (query.target.kernel.steps state action).mergeSort transitionResultLe =
    query.target.kernel.steps state action at stepCanonical
  have actionCanonicalBool :
      (evidence.actions.mergeSort modelValueLe != evidence.actions) = false := by
    rw [actionCanonical]
    change (!(evidence.actions == evidence.actions)) = false
    rw [modelValueListBeqSelf]
    rfl
  have initialCanonicalBool : ∀ setup,
      ((query.target.kernel.initialStates setup).mergeSort modelValueLe !=
        query.target.kernel.initialStates setup) = false := by
    intro setup
    rw [initialCanonical]
    change (!(query.target.kernel.initialStates setup ==
      query.target.kernel.initialStates setup)) = false
    rw [modelValueListBeqSelf]
    rfl
  have stepCanonicalBool : ∀ state action,
      ((query.target.kernel.steps state action).mergeSort transitionResultLe !=
        query.target.kernel.steps state action) = false := by
    intro state action
    rw [stepCanonical]
    change (!(query.target.kernel.steps state action ==
      query.target.kernel.steps state action)) = false
    rw [transitionResultListBeqSelf]
    rfl
  have findFalse : ∀ {α : Type} (items : List α),
      items.find? (fun _ => false) = none := by
    intro α items
    induction items <;> simp_all
  simp [IncrementalPlannerKernel.ofCheckedQuery, targetMatches, completeness, finiteOrderError?,
    behaviorDomain, actionCanonicalBool, initialCanonicalBool, stepCanonicalBool, findFalse]
  rfl

private structure PlannerCursor where
  trace : BehaviorTrace
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

structure PlannerInstrumentation where
  backendPulls : Nat := 0
  generatedCandidates : Nat := 0
  retainedPendingCandidates : Nat := 0
  peakActiveFrontierDepth : Nat := 0
  actionDomainPulls : Nat := 0
  initialKernelPulls : Nat := 0
  stepKernelPulls : Nat := 0
  deriving BEq, DecidableEq, Repr

inductive PlanningOutcome where
  | found (trace : BehaviorTrace) (reason : SelectionReason)
  | verified
  | noSuchTraceWithinCompleteLimits
  | limitReached
  | unsatisfiable
  | invalid (error : QueryError)
  deriving BEq, DecidableEq, Repr

def PlanningOutcome.name : PlanningOutcome → String
  | .found _ _ => "found"
  | .verified => "verified-within-limits"
  | .noSuchTraceWithinCompleteLimits => "no-such-trace-within-complete-limits"
  | .limitReached => "limit-reached"
  | .unsatisfiable => "unsatisfiable"
  | .invalid _ => "invalid"

private def planningOutcomeConstructorIndex : PlanningOutcome → Nat
  | .found _ _ => 0
  | .verified => 1
  | .noSuchTraceWithinCompleteLimits => 2
  | .limitReached => 3
  | .unsatisfiable => 4
  | .invalid _ => 5

/-- Canonical documentation and exact constructor matchers for Planning outcomes. -/
def PlanningOutcome.constructorClassifiers :
    List (OutcomeConstructorClassifier PlanningOutcome) := [
  {
    descriptor := { name := "found", description := "Planning selected one Model Trace." }
    accepts := fun outcome => planningOutcomeConstructorIndex outcome == 0
  },
  {
    descriptor := {
      name := "verified-within-limits"
      description := "Planning verified the requested universal claim within complete Limits."
    }
    accepts := fun outcome => planningOutcomeConstructorIndex outcome == 1
  },
  {
    descriptor := {
      name := "no-such-trace-within-complete-limits"
      description := "Complete bounded search found no matching Model Trace."
    }
    accepts := fun outcome => planningOutcomeConstructorIndex outcome == 2
  },
  {
    descriptor := {
      name := "limit-reached"
      description := "Planning reached its search Limit before completing the Query."
    }
    accepts := fun outcome => planningOutcomeConstructorIndex outcome == 3
  },
  {
    descriptor := {
      name := "unsatisfiable"
      description := "The checked Behavior admits no Model Traces."
    }
    accepts := fun outcome => planningOutcomeConstructorIndex outcome == 4
  },
  {
    descriptor := { name := "invalid", description := "Planning rejected the Query." }
    accepts := fun outcome => planningOutcomeConstructorIndex outcome == 5
  }
]

/-- Every Planning outcome, including arbitrary payloads, matches exactly one descriptor. -/
theorem PlanningOutcome.constructorClassifiers_exactlyOne :
    OutcomeConstructorClassifiers.ExactlyOne PlanningOutcome.constructorClassifiers
  | .found _ _ => rfl
  | .verified => rfl
  | .noSuchTraceWithinCompleteLimits => rfl
  | .limitReached => rfl
  | .unsatisfiable => rfl
  | .invalid _ => rfl

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

structure PlannerRun where
  result : PlanningResult
  artifact : Option ExperimentSpec
  instrumentation : PlannerInstrumentation
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
  | stopped (trace : BehaviorTrace) (reason : SelectionReason)
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
  | stop (state : State) (trace : BehaviorTrace) (reason : SelectionReason)

/-- Bounded traversal evidence shared by planning and finite semantic analyses. -/
structure BoundedTraversalResult (State : Type) where
  state : State
  termination : BoundedTraversalTermination
  metadata : PlanningMetadata
  instrumentation : PlannerInstrumentation

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
            match query.claim with
            | .verifiedWithinLimits => (.verified, true)
            | .satisfyingWitness | .violatingCounterexample | .limitedSelection =>
                (.noSuchTraceWithinCompleteLimits, true)
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
  Nat.min query.limits.behavior.transitions.value query.limits.behavior.selectedActions.value

private def rootTrace (setup : List RoleBinding) (initialState : ModelValue) : BehaviorTrace := {
  setup
  trace := { initialState, steps := [] }
}

private def appendStep
    (candidate : BehaviorTrace)
    (action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) : BehaviorTrace := {
  candidate with trace := {
    candidate.trace with
    steps := candidate.trace.steps ++ [{
      selectedAction := action
      modelOutcome := result.modelOutcome
      resultingState := result.resultingState
      observations := result.observations
    }]
  }
}

private def currentState (candidate : BehaviorTrace) : ModelValue :=
  match candidate.trace.steps.getLast? with
  | some step => step.resultingState
  | none => candidate.trace.initialState

private partial def nextRoot?
    (query : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel query.target)
    (state : PurePlannerState) : Option (BehaviorTrace × PurePlannerState) :=
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
    (kernel : IncrementalPlannerKernel query.target)
    (state : PurePlannerState) : PlannerPull PurePlannerState BehaviorTrace :=
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

private def purePlannerBackend
    (query : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel query.target) :
    PlannerBackend Unit PurePlannerState BehaviorTrace := {
  start := fun _ => {}
  pull := fun _ => pullCandidate query kernel
}

private def evaluatesToSelection
    (query : CheckedQuery LawStatement)
    (candidate : BehaviorTrace) : Except QueryError (Option SelectionReason) := do
  let evaluate (property : CheckedProperty) : Except QueryError PropertyEvaluation := do
    match checkPropertyEvaluationInput property candidate.trace with
    | .ok input => pure (evaluateProperty property input)
    | .error error =>
        throw {
          kind := .propertyEvaluationFailure
          definitionId := query.id
          sourcePath := error.sourcePath
          offendingValue := error.kind.name ++ ":" ++ error.offendingValue
          relatedDefinitionIds := DefinitionId.canonicalSet
            (property.id :: property.guardedClauseIds ++ error.relatedDefinitionIds)
        }
  match query.form with
  | .verify property =>
      if (← evaluate property).satisfied then
        pure none
      else
        pure (some .violatingCounterexample)
  | .witness property =>
      if (← evaluate property).satisfied then
        pure (some .satisfyingWitness)
      else
        pure none
  | .counterexample property =>
      if (← evaluate property).satisfied then
        pure none
      else
        pure (some .violatingCounterexample)
  | .select properties =>
      for property in properties do
        let _ ← evaluate property
      pure (some .behaviorSelection)

private def noteCandidate
    (candidate : BehaviorTrace)
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
    (candidate : BehaviorTrace)
    (next : PurePlannerState)
    (instrumentation : PlannerInstrumentation) : PlannerInstrumentation := {
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
    (instrumentation : PlannerInstrumentation)
    (termination : BoundedTraversalTermination) : PlannerRun :=
  let result := finalizePlanning query explored termination
  let artifact := match termination with
    | .stopped trace reason => some (artifactOfSelection query trace reason explored)
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
    (instrumentation : PlannerInstrumentation) : BoundedTraversalResult State := {
  state
  termination
  metadata := traversalMetadata query explored termination
  instrumentation
}

private def traverseLoop
    (query : CheckedQuery LawStatement)
    (backend : PlannerBackend Unit PurePlannerState BehaviorTrace)
    (cursor : PurePlannerState)
    (consumerState : State)
    (visit : State → BehaviorTrace → Except QueryError (BoundedTraversalStep State))
    (remaining : Nat)
    (behaviorAdmitted : Bool)
    (explored : ExploredCounts)
    (instrumentation : PlannerInstrumentation) : BoundedTraversalResult State :=
  match remaining with
  | 0 => traversalResult query consumerState .limitReached explored instrumentation
  | remaining + 1 =>
      match backend.pull () cursor with
      | .complete =>
          traversalResult query consumerState (.complete behaviorAdmitted) explored
            { instrumentation with backendPulls := instrumentation.backendPulls + 1 }
      | .yield candidate next =>
          let explored := noteCandidate candidate explored
          let instrumentation := notePull candidate next instrumentation
          if query.behavior.admits candidate then
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
    (kernel : IncrementalPlannerKernel query.target)
    (initial : State)
    (visit : State → BehaviorTrace → Except QueryError (BoundedTraversalStep State)) :
    BoundedTraversalResult State :=
  if query.behavior.isUnsatisfiable then
    traversalResult query initial (.complete false) {} {}
  else
    let backend := purePlannerBackend query kernel
    traverseLoop query backend (backend.start ()) initial visit query.limits.search.value false {} {}

/-- Plan a checked Query without invoking runtime, readers, evidence, or promotion behavior. -/
def plan
    (query : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel query.target) : PlannerRun :=
  let traversed := traverseBoundedCandidates query kernel () fun _ candidate => do
    match ← evaluatesToSelection query candidate with
    | some reason => pure (.stop () candidate reason)
    | none => pure (.continue ())
  finish query traversed.metadata.explored traversed.instrumentation traversed.termination

/--
Plan through the unchanged target kernel, then project checked Artifact intent if one is selected.
-/
def planWithArtifactIntent
    (query : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel query.target)
    (intent : ArtifactIntent) : Except ArtifactIntentError PlannerRun := do
  intent.validateFor query
  let run := plan query kernel
  let artifact ← match run.artifact with
    | none => pure none
    | some spec => some <$> spec.withArtifactIntent query intent
  pure { run with artifact }

end Umpire

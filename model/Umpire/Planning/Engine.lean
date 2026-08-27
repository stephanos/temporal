import Umpire.Artifact

/-! Implementation behind the `Umpire.Planning` public facade. -/

namespace Umpire

/-! Incremental, deterministic enumeration through a checked target's semantic relation. -/

def semanticValueOrderKey (value : SemanticValue) : String :=
  value.identity.value ++ "\u001f" ++ value.value

def transitionResultOrderKey
    (result : TransitionResult SemanticValue SemanticValue SemanticValue) : String :=
  semanticValueOrderKey result.modelOutcome ++ "\u001e" ++
    semanticValueOrderKey result.resultingState ++ "\u001e" ++
    String.intercalate "\u001d" (result.observations.map semanticValueOrderKey)

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
  actionAt : Nat → Option SemanticValue
  initialLimit : List RoleBinding → Nat
  initialAt : List RoleBinding → Nat → Option SemanticValue
  stepLimit : SemanticValue → SemanticValue → Nat
  stepAt : SemanticValue → SemanticValue → Nat →
    Option (TransitionResult SemanticValue SemanticValue SemanticValue)
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
      semanticValueOrderKey left ≤ semanticValueOrderKey right
  initialOrdered : ∀ setup first second left right, first < second →
    initialAt setup first = some left → initialAt setup second = some right →
      semanticValueOrderKey left ≤ semanticValueOrderKey right
  stepOrdered : ∀ state action first second left right, first < second →
    stepAt state action first = some left → stepAt state action second = some right →
      transitionResultOrderKey left ≤ transitionResultOrderKey right

/-- Canonical ordering obligations for the finite lists already owned by a checked target. -/
structure FiniteKernelOrder
    (target : QueryTarget LawStatement)
    (evidence : FiniteCompletenessEvidence LawStatement target) where
  action : evidence.actions.Pairwise fun left right =>
    semanticValueOrderKey left ≤ semanticValueOrderKey right
  initial : ∀ setup, (target.kernel.initialStates setup).Pairwise fun left right =>
    semanticValueOrderKey left ≤ semanticValueOrderKey right
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
        semanticValueOrderKey left ≤ semanticValueOrderKey right)
    (initialOrdered : ∀ evidence, query.completeness = some evidence → ∀ setup,
      (query.target.kernel.initialStates setup).Pairwise fun left right =>
        semanticValueOrderKey left ≤ semanticValueOrderKey right)
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

private structure PlannerCursor where
  trace : BehaviorTrace
  nextAction : Nat := 0
  currentAction : Option SemanticValue := none
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
  | noSuchTraceWithinCompleteBounds
  | budgetExhausted
  | unsatisfiable
  | invalid (error : QueryError)
  deriving BEq, DecidableEq, Repr

def PlanningOutcome.name : PlanningOutcome → String
  | .found _ _ => "found"
  | .verified => "verified-within-bounds"
  | .noSuchTraceWithinCompleteBounds => "no-such-trace-within-complete-bounds"
  | .budgetExhausted => "budget-exhausted"
  | .unsatisfiable => "unsatisfiable"
  | .invalid _ => "invalid"

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

private def evidenceDigests (query : CheckedQuery LawStatement) : List String :=
  match query.completeness with
  | none => []
  | some evidence => [
      evidence.roleDomainDigest,
      evidence.actionDomainDigest
    ]

private def planningMetadata
    (query : CheckedQuery LawStatement)
    (explored : ExploredCounts)
    (established : Bool) : PlanningMetadata := {
  explored
  completeness := {
    established
    bounds := query.bounds
    finiteEvidenceDigests := evidenceDigests query
  }
}

private inductive PlanningTermination where
  | found (trace : BehaviorTrace) (reason : SelectionReason)
  | complete (behaviorAdmitted : Bool)
  | budgetExhausted
  | invalid (error : QueryError)
  deriving BEq, DecidableEq, Repr

/-- The planner-private result finalizer enforces the query's claim strength. A backend completion
signal establishes completeness only for a finite exhaustive query that admitted at least one
behavior trace, and an empty behavior always wins over every attempted terminal claim. -/
private def finalizePlanning
    (query : CheckedQuery LawStatement)
    (explored : ExploredCounts)
    (termination : PlanningTermination) : PlanningResult :=
  let (outcome, established) :=
    if query.behavior.isUnsatisfiable then
      (PlanningOutcome.unsatisfiable, false)
    else
      match termination with
      | .found trace reason => (.found trace reason, false)
      | .budgetExhausted => (.budgetExhausted, false)
      | .invalid error => (.invalid error, false)
      | .complete false => (.unsatisfiable, false)
      | .complete true =>
          if query.policy.strategy != .exhaustive || query.completeness.isNone then
            (.budgetExhausted, false)
          else
            match query.claim with
            | .verifiedWithinBounds => (.verified, true)
            | .satisfyingWitness | .violatingCounterexample | .boundedSelection =>
                (.noSuchTraceWithinCompleteBounds, true)
  PlanningResult.mk outcome (planningMetadata query explored established)

private def valueLe (left right : SemanticValue) : Bool :=
  decide (semanticValueOrderKey left ≤ semanticValueOrderKey right)

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value < right.role.value) ||
    (left.role == right.role && valueLe left.value right.value)

private def canonicalSetup (setup : List RoleBinding) : List RoleBinding :=
  setup.mergeSort bindingLe

private def setupKey (setup : List RoleBinding) : String :=
  String.intercalate "\u001f" ((canonicalSetup setup).map fun binding =>
    binding.role.value ++ "\u001e" ++ semanticValueOrderKey binding.value)

private def setupLe (left right : List RoleBinding) : Bool :=
  decide (setupKey left ≤ setupKey right)

private def rotate (offset : Nat) (items : List α) : List α :=
  if items.isEmpty then
    []
  else
    let pivot := offset % items.length
    items.drop pivot ++ items.take pivot

private def applySeed
    (query : CheckedQuery LawStatement)
    (items : List α) : List α :=
  if query.policy.strategy == .coverageGuided then
    rotate query.policy.seed items
  else
    items

private def candidateSetups (query : CheckedQuery LawStatement) : List (List RoleBinding) :=
  let setups := match query.completeness with
    | some evidence => evidence.roleAssignments
    | none => query.target.resolvedSetups
  applySeed query (setups.mergeSort setupLe |>.eraseDups)

private def seededIndex
    (query : CheckedQuery LawStatement)
    (limit logicalIndex : Nat) : Nat :=
  if query.policy.strategy == .coverageGuided && limit > 0 then
    (logicalIndex + query.policy.seed % limit) % limit
  else
    logicalIndex

private def maximumDepth (query : CheckedQuery LawStatement) : Nat :=
  Nat.min query.bounds.behavior.transitions.value query.bounds.behavior.selectedActions.value

private def rootTrace (setup : List RoleBinding) (initialState : SemanticValue) : BehaviorTrace := {
  setup
  trace := { initialState, steps := [] }
}

private def appendStep
    (candidate : BehaviorTrace)
    (action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue) : BehaviorTrace := {
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

private def currentState (candidate : BehaviorTrace) : SemanticValue :=
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
    (candidate : BehaviorTrace) : Option SelectionReason :=
  match query.form with
  | .verify property =>
      if (evaluateProperty property candidate.trace).satisfied then
        none
      else
        some .violatingCounterexample
  | .witness property =>
      if (evaluateProperty property candidate.trace).satisfied then
        some .satisfyingWitness
      else
        none
  | .counterexample property =>
      if (evaluateProperty property candidate.trace).satisfied then
        none
      else
        some .violatingCounterexample
  | .select properties =>
      let _ := properties.map fun property => evaluateProperty property candidate.trace
      some .behaviorSelection

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
    (termination : PlanningTermination) : PlannerRun :=
  let result := finalizePlanning query explored termination
  let artifact := match termination with
    | .found trace reason => some (artifactOfSelection query trace reason explored)
    | _ => none
  { result, artifact, instrumentation }

private def planLoop
    (query : CheckedQuery LawStatement)
    (backend : PlannerBackend Unit PurePlannerState BehaviorTrace)
    (state : PurePlannerState)
    (remaining : Nat)
    (behaviorAdmitted : Bool)
    (explored : ExploredCounts)
    (instrumentation : PlannerInstrumentation) : PlannerRun :=
  match remaining with
  | 0 => finish query explored instrumentation .budgetExhausted
  | remaining + 1 =>
      match backend.pull () state with
      | .complete =>
          finish query explored
            { instrumentation with backendPulls := instrumentation.backendPulls + 1 }
            (.complete behaviorAdmitted)
      | .yield candidate next =>
          let explored := noteCandidate candidate explored
          let instrumentation := notePull candidate next instrumentation
          if query.behavior.admits candidate then
            let explored := notePropertyEvaluations query explored
            match evaluatesToSelection query candidate with
            | some reason =>
                finish query explored instrumentation (.found candidate reason)
            | none =>
                planLoop query backend next remaining true explored instrumentation
          else
            planLoop query backend next remaining behaviorAdmitted explored instrumentation
termination_by remaining

/-- Plan a checked Query without invoking runtime, readers, evidence, or promotion behavior. -/
def plan
    (query : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel query.target) : PlannerRun :=
  if query.behavior.isUnsatisfiable then
    finish query {} {} (.complete false)
  else
    let backend := purePlannerBackend query kernel
    planLoop query backend (backend.start ()) query.bounds.search.value false {} {}

end Umpire

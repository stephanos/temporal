import Temporal.Experiment.Artifact

namespace Temporal.Experiment

/-! Incremental, deterministic enumeration through a checked target's semantic relation. -/

def semanticValueOrderKey (value : SemanticValue) : String :=
  value.identity.value ++ "\u001f" ++ value.value

def transitionResultOrderKey
    (result : TransitionResult SemanticValue SemanticValue SemanticValue) : String :=
  semanticValueOrderKey result.modelOutcome ++ "\u001e" ++
    semanticValueOrderKey result.resultingState ++ "\u001e" ++
    String.intercalate "\u001d" (result.observations.map semanticValueOrderKey)

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

structure PlannerCursor where
  trace : BehaviorTrace
  nextAction : Nat := 0
  currentAction : Option SemanticValue := none
  nextOutcome : Nat := 0
  deriving BEq, DecidableEq, Repr

structure PurePlannerState where
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

structure PlannerRun where
  result : PlanningResult
  artifact : Option ExperimentSpec
  instrumentation : PlannerInstrumentation
  deriving BEq, DecidableEq, Repr

private instance : Inhabited (PlannerPull State Candidate) := ⟨.complete⟩

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

def purePlannerBackend
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
    (explored : ExploredCounts)
    (instrumentation : PlannerInstrumentation) : PlannerRun :=
  match remaining with
  | 0 => finish query explored instrumentation .budgetExhausted
  | remaining + 1 =>
      match backend.pull () state with
      | .complete =>
          finish query explored
            { instrumentation with backendPulls := instrumentation.backendPulls + 1 }
            .complete
      | .yield candidate next =>
          let explored := noteCandidate candidate explored
          let instrumentation := notePull candidate next instrumentation
          if query.behavior.admits candidate then
            let explored := notePropertyEvaluations query explored
            match evaluatesToSelection query candidate with
            | some reason =>
                finish query explored instrumentation (.found candidate reason)
            | none =>
                planLoop query backend next remaining explored instrumentation
          else
            planLoop query backend next remaining explored instrumentation
termination_by remaining

/-- Plan a checked Query without invoking runtime, readers, evidence, or promotion behavior. -/
def plan
    (query : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel query.target) : PlannerRun :=
  if query.behavior.isUnsatisfiable then
    finish query {} {} .complete
  else
    let backend := purePlannerBackend query kernel
    planLoop query backend (backend.start ()) query.bounds.search.value {} {}

end Temporal.Experiment

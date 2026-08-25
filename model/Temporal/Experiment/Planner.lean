import Temporal.Experiment.Artifact

namespace Temporal.Experiment

/-! Incremental, deterministic enumeration through a checked target's semantic kernel. -/

structure PlannerCursor where
  trace : BehaviorTrace
  nextAction : Nat := 0
  nextOutcome : Nat := 0
  deriving BEq, DecidableEq, Repr

structure PurePlannerState where
  targetDepth : Nat := 0
  setupIndex : Nat := 0
  initialIndex : Nat := 0
  activePath : List PlannerCursor := []
  deriving BEq, DecidableEq, Repr

structure PlannerInstrumentation where
  backendPulls : Nat := 0
  generatedCandidates : Nat := 0
  retainedPendingCandidates : Nat := 0
  peakActiveFrontierDepth : Nat := 0
  deriving BEq, DecidableEq, Repr

structure PlannerRun where
  result : PlanningResult
  artifact : Option ExperimentSpec
  instrumentation : PlannerInstrumentation
  deriving BEq, DecidableEq, Repr

private instance : Inhabited (PlannerPull State Candidate) := ⟨.complete⟩

private def idLe (left right : DeclarationId) : Bool :=
  decide (left.value ≤ right.value)

private def valueLe (left right : SemanticValue) : Bool :=
  decide (left.identity.value < right.identity.value) ||
    (left.identity == right.identity && decide (left.value ≤ right.value))

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value < right.role.value) ||
    (left.role == right.role && valueLe left.value right.value)

private def canonicalSetup (setup : List RoleBinding) : List RoleBinding :=
  setup.mergeSort bindingLe

private def setupKey (setup : List RoleBinding) : String :=
  String.intercalate "\u001f" ((canonicalSetup setup).map fun binding =>
    binding.role.value ++ "\u001e" ++ binding.value.identity.value ++ "\u001e" ++
      binding.value.value)

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

private def exactTraceActions (behavior : CheckedBehavior) : List SemanticValue :=
  match behavior.traceExactly with
  | none => []
  | some exact => exact.trace.steps.map fun step => step.selectedAction

private def candidateActions (query : CheckedQuery LawStatement) : List SemanticValue :=
  let actions := match query.completeness with
    | some evidence => evidence.actions
    | none => exactTraceActions query.behavior
  applySeed query (actions.mergeSort valueLe |>.eraseDups)

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
    (state : PurePlannerState) : Option (BehaviorTrace × PurePlannerState) :=
  match (candidateSetups query)[state.setupIndex]? with
  | none => none
  | some setup =>
      match (query.target.kernel.initialStates setup)[state.initialIndex]? with
      | some initial =>
          some (rootTrace setup initial, { state with initialIndex := state.initialIndex + 1 })
      | none =>
          nextRoot? query {
            state with
            setupIndex := state.setupIndex + 1
            initialIndex := 0
          }

/--
Enumerate one trace at a time. The state retains cursor indexes for the active path, never a queue
of produced candidates or the unconsumed tail of a kernel result list.
-/
private partial def pullCandidate
    (query : CheckedQuery LawStatement)
    (state : PurePlannerState) : PlannerPull PurePlannerState BehaviorTrace :=
  match state.activePath with
  | [] =>
      match nextRoot? query state with
      | some (root, next) =>
          if state.targetDepth == 0 then
            .yield root next
          else
            pullCandidate query { next with activePath := [{ trace := root }] }
      | none =>
          if state.targetDepth < maximumDepth query then
            pullCandidate query {
              targetDepth := state.targetDepth + 1
              setupIndex := 0
              initialIndex := 0
              activePath := []
            }
          else
            .complete
  | cursor :: parents =>
      match (candidateActions query)[cursor.nextAction]? with
      | none => pullCandidate query { state with activePath := parents }
      | some action =>
          let results := query.target.kernel.steps (currentState cursor.trace) action
          match results[cursor.nextOutcome]? with
          | none =>
              let advanced := { cursor with
                nextAction := cursor.nextAction + 1
                nextOutcome := 0
              }
              pullCandidate query { state with activePath := advanced :: parents }
          | some result =>
              let advanced := { cursor with nextOutcome := cursor.nextOutcome + 1 }
              let child := appendStep cursor.trace action result
              let next := { state with activePath := advanced :: parents }
              if child.trace.steps.length == state.targetDepth then
                .yield child next
              else
                pullCandidate query { next with activePath := { trace := child } :: next.activePath }

def purePlannerBackend (LawStatement : DeclarationId → Prop) :
    PlannerBackend (CheckedQuery LawStatement) PurePlannerState BehaviorTrace := {
  start := fun _ => {}
  pull := pullCandidate
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
    (instrumentation : PlannerInstrumentation) : PlannerInstrumentation := {
  instrumentation with
  backendPulls := instrumentation.backendPulls + 1
  generatedCandidates := instrumentation.generatedCandidates + 1
  retainedPendingCandidates := 0
  peakActiveFrontierDepth := Nat.max instrumentation.peakActiveFrontierDepth
    (candidate.trace.steps.length + 1)
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
    (state : PurePlannerState)
    (remaining : Nat)
    (explored : ExploredCounts)
    (instrumentation : PlannerInstrumentation) : PlannerRun :=
  match remaining with
  | 0 => finish query explored instrumentation .budgetExhausted
  | remaining + 1 =>
    match (purePlannerBackend LawStatement).pull query state with
    | .complete =>
        finish query explored
          { instrumentation with backendPulls := instrumentation.backendPulls + 1 }
          .complete
    | .yield candidate next =>
        let explored := noteCandidate candidate explored
        let instrumentation := notePull candidate instrumentation
        if query.behavior.admits candidate then
          let explored := notePropertyEvaluations query explored
          match evaluatesToSelection query candidate with
          | some reason =>
              finish query explored instrumentation (.found candidate reason)
          | none =>
              planLoop query next remaining explored instrumentation
        else
          planLoop query next remaining explored instrumentation
termination_by remaining

/-- Plan a checked Query without invoking runtime, readers, evidence, or promotion behavior. -/
def plan (query : CheckedQuery LawStatement) : PlannerRun :=
  if query.behavior.isUnsatisfiable then
    finish query {} {} .complete
  else
    let backend := purePlannerBackend LawStatement
    planLoop query (backend.start query) query.bounds.search.value {} {}

end Temporal.Experiment

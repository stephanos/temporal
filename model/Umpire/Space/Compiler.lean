import Umpire.Planning
import Umpire.Space.Intent

/-! Exact checked-Space point lowering and atomic target-owned batch compilation. -/

namespace Umpire

inductive SpaceCompilationErrorKind where
  | missingChoice
  | extraChoice
  | duplicateChoice
  | unknownChoice
  | incompatibleFaultSelection
  | behaviorCheckFailed
  | queryCheckFailed
  | intentCheckFailed
  | duplicatePointIdentity
  | plannerInvalid
  | unsatisfiable
  | budgetExhausted
  | verifiedWithoutArtifact
  | noArtifact
  | duplicateExperimentSpecIdentity
  deriving BEq, DecidableEq, Ord, Repr

def SpaceCompilationErrorKind.name : SpaceCompilationErrorKind → String
  | .missingChoice => "missing-choice"
  | .extraChoice => "extra-choice"
  | .duplicateChoice => "duplicate-choice"
  | .unknownChoice => "unknown-choice"
  | .incompatibleFaultSelection => "incompatible-fault-selection"
  | .behaviorCheckFailed => "behavior-check-failed"
  | .queryCheckFailed => "query-check-failed"
  | .intentCheckFailed => "intent-check-failed"
  | .duplicatePointIdentity => "duplicate-point-identity"
  | .plannerInvalid => "planner-invalid"
  | .unsatisfiable => "unsatisfiable"
  | .budgetExhausted => "budget-exhausted"
  | .verifiedWithoutArtifact => "verified-without-artifact"
  | .noArtifact => "no-artifact"
  | .duplicateExperimentSpecIdentity => "duplicate-experiment-spec-identity"

structure SpaceCompilationError where
  kind : SpaceCompilationErrorKind
  pointId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- One exact assignment lowered through Behavior, Query, and Artifact-intent checking. -/
structure LoweredSpacePoint
    (space : CheckedExperimentSpace LawStatement) where
  id : DefinitionId
  assignment : List ModelValue
  query : CheckedQuery LawStatement
  targetEq : query.target = space.baseQuery.target
  intent : ArtifactIntent

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def assignmentLe (left right : ModelValue) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId && decide (left.value ≤ right.value))

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def canonicalAssignment (assignment : List ModelValue) : List ModelValue :=
  assignment.mergeSort assignmentLe

private def assignmentJson (assignment : List ModelValue) : String :=
  array (canonicalAssignment assignment |>.map fun selected =>
    "{\"axis\":" ++ quote selected.definitionId.value ++
      ",\"choice\":" ++ quote selected.value ++ "}")

private def pointIdentity
    (space : CheckedExperimentSpace LawStatement)
    (assignment : List ModelValue) : DefinitionId :=
  let digest := (behaviorFingerprintOf <|
    "umpire-space-point/v1\n" ++ space.id.value ++ "\n" ++ assignmentJson assignment).render
  DefinitionId.of (space.id.value ++ ".point." ++ (digest.drop 7).toString)

private def compilationError
    (space : CheckedExperimentSpace LawStatement)
    (kind : SpaceCompilationErrorKind)
    (pointId : DefinitionId)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : SpaceCompilationError := {
  kind
  pointId
  sourcePath := if space.source.path == "" then "<unknown>" else space.source.path
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

def canonicalSpaceCompilationErrorJson (error : SpaceCompilationError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"pointId\":" ++ quote error.pointId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (canonicalIds error.relatedDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def firstDuplicateAxis : List ModelValue → Option DefinitionId
  | first :: second :: rest =>
      if first.definitionId == second.definitionId then
        some first.definitionId
      else
        firstDuplicateAxis (second :: rest)
  | _ => none

private def selectedChoices
    (space : CheckedExperimentSpace LawStatement)
    (assignment : List ModelValue) :
    Except SpaceCompilationError (DefinitionId × List ModelValue ×
      List (CheckedVariationAxis × CheckedChoice)) := do
  let assignment := canonicalAssignment assignment
  let pointId := pointIdentity space assignment
  match firstDuplicateAxis assignment with
  | some duplicate =>
      throw (compilationError space .duplicateChoice pointId duplicate.value [duplicate])
  | none => pure ()
  for selected in assignment do
    if !(space.axes.any fun axis => axis.id == selected.definitionId) then
      throw (compilationError space .extraChoice pointId selected.definitionId.value
        [selected.definitionId, DefinitionId.of selected.value])
  for axis in space.axes do
    if !(assignment.any fun selected => selected.definitionId == axis.id) then
      throw (compilationError space .missingChoice pointId axis.id.value [axis.id])
  let mut result := []
  for axis in space.axes do
    let selected := assignment.find? fun candidate => candidate.definitionId == axis.id
    let selected := selected.getD { definitionId := axis.id, value := "" }
    let choiceId := DefinitionId.of selected.value
    let choice ← match axis.choices.find? fun candidate => candidate.id == choiceId with
      | some choice => pure choice
      | none => throw (compilationError space .unknownChoice pointId choiceId.value
          [axis.id, choiceId])
    result := result ++ [(axis, choice)]
  pure (pointId, assignment, result)

private def authoredExactTrace (trace : BehaviorTrace) : AuthoredExactTrace := {
  setup := trace.setup
  initialState := some trace.trace.initialState
  steps := trace.trace.steps.map fun step => {
    selectedAction := some step.selectedAction
    modelOutcome := some step.modelOutcome
    resultingState := some step.resultingState
    observations := some step.observations
  }
}

private def derivedBehaviorId (pointId : DefinitionId) : DefinitionId :=
  DefinitionId.of (pointId.value ++ ".behavior")

private def derivedQueryId (pointId : DefinitionId) : DefinitionId :=
  DefinitionId.of (pointId.value ++ ".query")

private def bindingConstraint
    (pointId : DefinitionId)
    (binding : RoleBinding) : SetupConstraint := {
  id := DefinitionId.of (pointId.value ++ ".binding." ++ binding.role.value)
  relation := .equal
  left := .role binding.role
  right := .value binding.value
}

private def behaviorDeclaration
    (space : CheckedExperimentSpace LawStatement)
    (pointId : DefinitionId)
    (bindings : List RoleBinding) : BehaviorDeclaration :=
  let base := space.baseQuery.behavior
  {
    id := derivedBehaviorId pointId
    source := base.source
    version := base.version
    requires := base.requires
    roles := base.roles
    setup := base.setup ++ bindings.map (bindingConstraint pointId)
    allowedActions := base.allowedActions
    requiredOccurrences := base.requiredOccurrences
    forbiddenActions := base.forbiddenActions
    occurrenceBounds := base.occurrenceBounds
    ordering := base.ordering
    sequences := base.sequences
    adjacencies := base.adjacencies
    actionsExactly := base.actionsExactly
    traceExactly := base.traceExactly.map authoredExactTrace
    documentation := base.documentation
  }

private def queryDeclaration
    (space : CheckedExperimentSpace LawStatement)
    (pointId : DefinitionId)
    (behavior : CheckedBehavior) : QueryDeclaration :=
  let base := space.baseQuery
  {
    id := derivedQueryId pointId
    source := base.source
    version := base.version
    target := base.target.id
    form := base.form
    behavior
    limits := base.limits
    policy := base.policy
    documentation := base.documentation
  }

private structure RecheckedQuery (target : QueryTarget LawStatement) where
  query : CheckedQuery LawStatement
  targetEq : query.target = target

private def materializeQuery
    (target : QueryTarget LawStatement)
    (query : CheckedQuery LawStatement) : RecheckedQuery target :=
  let checkedTarget := CheckedQueryTarget.ofTarget target
  {
    query := {
      query with
      target := checkedTarget.target
      completeness := checkedTarget.completeness
    }
    targetEq := by
      cases planningEq : target.planning <;>
        simp [checkedTarget, CheckedQueryTarget.ofTarget, planningEq]
  }

private def selectedFaults
    (space : CheckedExperimentSpace LawStatement)
    (choices : List (CheckedVariationAxis × CheckedChoice)) :
    Except SpaceCompilationError (List CheckedFaultIntent) := do
  let ids := choices.flatMap fun selected => selected.2.faults
  let pointId := pointIdentity space (choices.map fun selected => {
    definitionId := selected.1.id
    value := selected.2.id.value
  })
  for fault in space.faults do
    if ids.contains fault.id then
      for incompatible in fault.incompatibleWith do
        if ids.contains incompatible then
          throw (compilationError space .incompatibleFaultSelection pointId
            (fault.id.value ++ "<->" ++ incompatible.value) [fault.id, incompatible])
  pure (space.faults.filter fun fault => ids.contains fault.id)

private def intentDeclaration
    (assignment : List ModelValue)
    (choices : List (CheckedVariationAxis × CheckedChoice))
    (faults : List CheckedFaultIntent) : ArtifactIntentDeclaration := {
  selectedChoices := assignment
  selectedVariants := choices.filterMap fun selected => selected.2.binding
  requestedFaults := faults.map fun fault => {
    definitionId := fault.id
    occurrenceDefinitionId := fault.occurrence.id
    actionDefinitionId := fault.occurrence.action
    capabilityDefinitionId := fault.capability.id
  }
  additionalCapabilityRequirementDefinitionIds := faults.map fun fault => fault.capability.id
}

/-- Lower one complete exact assignment without planning or constructing a target kernel. -/
def lowerSpacePoint
    (space : CheckedExperimentSpace LawStatement)
    (assignment : List ModelValue) :
    Except SpaceCompilationError (LoweredSpacePoint space) := do
  let (pointId, assignment, choices) ← selectedChoices space assignment
  let bindings := choices.filterMap fun selected => selected.2.binding
  let behavior ← match checkBehavior (.ofTarget space.baseQuery.target)
      (behaviorDeclaration space pointId bindings) with
    | .ok behavior => pure behavior
    | .error error => throw (compilationError space .behaviorCheckFailed pointId
        error.offendingValue error.relatedDefinitionIds)
  let checkedQuery ← match checkQuery (.ofTarget space.baseQuery.target)
      (queryDeclaration space pointId behavior) with
    | .ok query => pure query
    | .error error => throw (compilationError space .queryCheckFailed pointId
        error.offendingValue error.relatedDefinitionIds)
  let materialized := materializeQuery space.baseQuery.target checkedQuery
  let query := materialized.query
  let faults ← selectedFaults space choices
  let intent ← match checkArtifactIntent query (intentDeclaration assignment choices faults) with
    | .ok intent => pure intent
    | .error error => throw (compilationError space .intentCheckFailed pointId error.kind.name
        error.relatedDefinitionIds)
  pure {
    id := pointId
    assignment
    query
    targetEq := materialized.targetEq
    intent
  }

private def cartesianAssignments :
    List CheckedVariationAxis → List (List ModelValue)
  | [] => [[]]
  | axis :: rest =>
      axis.choices.flatMap fun choice =>
        cartesianAssignments rest |>.map fun assignment =>
          { definitionId := axis.id, value := choice.id.value } :: assignment

private def planPoint
    (space : CheckedExperimentSpace LawStatement)
    (kernel : IncrementalPlannerKernel space.baseQuery.target)
    (point : LoweredSpacePoint space) : Except SpaceCompilationError PlannerRun := do
  let pointKernel : IncrementalPlannerKernel point.query.target :=
    Eq.mpr (congrArg IncrementalPlannerKernel point.targetEq) kernel
  match planWithArtifactIntent point.query pointKernel point.intent with
    | .ok run => pure run
    | .error error => throw (compilationError space .intentCheckFailed point.id error.kind.name
        error.relatedDefinitionIds)

namespace SpaceCompiler.Internal

/-- Append a canonical point identity only once; duplicate identity exposes no updated prefix. -/
def appendPointIdentity
    (space : CheckedExperimentSpace LawStatement)
    (pointIds : List DefinitionId)
    (pointId : DefinitionId) : Except SpaceCompilationError (List DefinitionId) :=
  if pointIds.contains pointId then
    .error (compilationError space .duplicatePointIdentity pointId pointId.value [pointId])
  else
    .ok (pointIds ++ [pointId])

/--
Append one planned point only after it selected a unique Artifact; failure exposes no prefix.
-/
def appendPlannerRun
    (space : CheckedExperimentSpace LawStatement)
    (pointId : DefinitionId)
    (specs : List ExperimentSpec)
    (run : PlannerRun) : Except SpaceCompilationError (List ExperimentSpec) := do
  match run.result.outcome with
  | .invalid error =>
      throw (compilationError space .plannerInvalid pointId error.offendingValue
        error.relatedDefinitionIds)
  | .unsatisfiable =>
      throw (compilationError space .unsatisfiable pointId run.result.outcome.name)
  | .limitReached =>
      throw (compilationError space .budgetExhausted pointId run.result.outcome.name)
  | .verified =>
      throw (compilationError space .verifiedWithoutArtifact pointId run.result.outcome.name)
  | .noSuchTraceWithinCompleteLimits =>
      throw (compilationError space .noArtifact pointId run.result.outcome.name)
  | .found _ _ =>
      match run.artifact with
      | none => throw (compilationError space .noArtifact pointId run.result.outcome.name)
      | some spec =>
          if specs.any fun existing => existing.artifactChecksum == spec.artifactChecksum then
            throw (compilationError space .duplicateExperimentSpecIdentity pointId
              spec.artifactChecksum.render [pointId])
          pure (specs ++ [spec])

end SpaceCompiler.Internal

/-- Compile every canonical point through one transported caller-owned kernel, or return no batch. -/
def compileBatch
    (space : CheckedExperimentSpace LawStatement)
    (kernel : IncrementalPlannerKernel space.baseQuery.target) :
    Except SpaceCompilationError (List ExperimentSpec) := do
  let assignments := cartesianAssignments space.axes
  let mut pointIds := []
  let mut specs := []
  for assignment in assignments do
    let point ← lowerSpacePoint space assignment
    pointIds ← SpaceCompiler.Internal.appendPointIdentity space pointIds point.id
    let run ← planPoint space kernel point
    specs ← SpaceCompiler.Internal.appendPlannerRun space point.id specs run
  pure specs

end Umpire

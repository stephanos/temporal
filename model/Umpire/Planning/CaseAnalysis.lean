import Umpire.Planning.Engine

/-!
# Finite guarded-case coverage

This module folds checked Property applicability over Planning's bounded candidate stream. Results
retain the exact Query scope and reachable witnesses while keeping completeness, case obligations,
and ordinary Property evaluation as separate evidence classes.
-/

namespace Umpire

/-- Stable identity of one selected checked Property in an analysis scope. -/
structure AnalyzedProperty where
  id : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- The exact checked Query inputs bounded by one case analysis. -/
structure CaseAnalysisScope where
  queryId : DefinitionId
  queryFingerprint : BehaviorFingerprint
  targetId : DefinitionId
  targetFingerprint : BehaviorFingerprint
  behaviorId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  properties : List AnalyzedProperty
  limits : QueryLimits
  endpoint : QueryEndpoint := .deliberatelyClosed
  exercise : QueryExercisePolicy := .allowVacuous
  deriving BEq, DecidableEq, Repr

inductive CaseAnalysisStatus where
  | exhaustive
  | limitReached
  | unsatisfiable
  | invalid (error : QueryError)
  deriving BEq, DecidableEq, Repr

inductive JointCompatibilityStatus where
  | compatibleWithinLimits
  | logicalConflict
  | modelIncompatible
  | unexercised
  | deadEnd
  | unsatisfiable
  | limitReached
  | unsupported
  | invalid (error : QueryError)
  deriving BEq, DecidableEq, Repr

def JointCompatibilityStatus.name : JointCompatibilityStatus → String
  | .compatibleWithinLimits => "compatible-within-limits"
  | .logicalConflict => "logical-conflict"
  | .modelIncompatible => "model-incompatible"
  | .unexercised => "unexercised"
  | .deadEnd => "dead-end"
  | .unsatisfiable => "unsatisfiable"
  | .limitReached => "limit-reached"
  | .unsupported => "unsupported"
  | .invalid _ => "invalid"

/-- Exact reachable trigger scope shared by jointly applicable obligations. The prefix ends before
the triggering transition; each temporal expectation separately retains its own coordinate. -/
structure JointTriggerScope where
  modeledPrefix : Scenario.Trace
  transitionPosition : Nat
  occurrence : JointTriggerOccurrence
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  limits : QueryLimits
  deriving BEq, DecidableEq, Repr

/-- One source-linked expectation selected at a common trigger. -/
structure JointExpectationEvidence where
  propertyId : DefinitionId
  parentId : DefinitionId
  caseId : Option DefinitionId
  clauseId : DefinitionId
  source : SourceLocation
  transitionPosition : Nat
  triggerCoordinate : Nat
  effectiveGuards : List (CheckedPropertyPredicate .guard)
  exceptions : List ResolvedPropertyException
  formula : JointObligationFormula
  deriving BEq, DecidableEq, Repr

/-- A finite logical contradiction over one typed field/reference intersection. -/
structure JointConflictEvidence where
  trigger : JointTriggerScope
  field : PropertyPredicateField
  reference : DefinitionId
  constraints : List PropertyAtomConstraint
  expectations : List JointExpectationEvidence
  deriving BEq, DecidableEq, Repr

/-- Exhaustive bounded evidence that the selected conjunction has modeled continuations but none
satisfies it. This is Target-relative evidence, not a logical contradiction. -/
structure JointModelIncompatibility where
  trigger : JointTriggerScope
  expectations : List JointExpectationEvidence
  admittedContinuations : List Scenario.Trace
  deriving BEq, DecidableEq, Repr

inductive JointUnsupportedFormulaClass where
  | disjunction
  | negation
  | propertyClause
  deriving BEq, DecidableEq, Ord, Repr

def JointUnsupportedFormulaClass.name : JointUnsupportedFormulaClass → String
  | .disjunction => "any"
  | .negation => "not"
  | .propertyClause => "unsupported-clause"

structure UnsupportedJointFormula where
  propertyId : DefinitionId
  clauseId : DefinitionId
  source : SourceLocation
  formulaClass : JointUnsupportedFormulaClass
  formula : Option PropertyPredicate
  deriving BEq, DecidableEq, Repr

structure JointTriggerAnalysis where
  trigger : JointTriggerScope
  expectations : List JointExpectationEvidence
  admittedContinuations : List Scenario.Trace
  satisfyingContinuations : List Scenario.Trace
  deriving BEq, DecidableEq, Repr

structure JointCompatibilityResult where
  status : JointCompatibilityStatus
  triggers : List JointTriggerAnalysis
  logicalConflicts : List JointConflictEvidence
  modelIncompatibilities : List JointModelIncompatibility
  unsupported : List UnsupportedJointFormula
  deriving BEq, DecidableEq, Repr

def CaseAnalysisStatus.name : CaseAnalysisStatus → String
  | .exhaustive => "exhaustive"
  | .limitReached => "limit-reached"
  | .unsatisfiable => "unsatisfiable"
  | .invalid _ => "invalid"

inductive CaseFindingKind where
  | caseExcluded
  | missingReplacement
  | uncoveredCompleteGroup
  | overlappingExclusiveCases
  | parentExcluded
  deriving BEq, DecidableEq, Ord, Repr

/-- One admitted trace and its exact evaluator-owned same-step applicability. -/
structure CaseObservation extends CaseGroupApplicability where
  trace : Scenario.Trace
  deriving BEq, DecidableEq, Repr

/-- A reachable failure or exclusion with source, identity, trigger, and effective-guard evidence. -/
structure CaseFinding where
  kind : CaseFindingKind
  propertyId : DefinitionId
  parentId : DefinitionId
  parentSource : SourceLocation
  caseIds : List DefinitionId
  caseSources : List SourceLocation
  clauses : List PropertyClauseIdentity
  trace : Scenario.Trace
  transitionPosition : Nat
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  effectiveGuards : List (CheckedPropertyPredicate .guard)
  exceptions : List ResolvedPropertyException
  deriving BEq, DecidableEq, Repr

/-- Exercise status for one declared completeness/exclusivity obligation. -/
structure CaseRequirement where
  propertyId : DefinitionId
  parentId : DefinitionId
  source : SourceLocation
  complete : Bool
  exclusive : Bool
  caseIds : List DefinitionId
  clauses : List PropertyClauseIdentity
  parentExercised : Bool
  exercisedCaseIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Ordinary Property truth remains separate from coverage and exclusivity obligations. -/
structure CasePropertyEvaluation where
  trace : Scenario.Trace
  evaluation : PropertyEvaluation
  endpointAnswer : PropertyEndpointAnswer := .satisfied
  deriving BEq, DecidableEq, Repr

structure CaseAnalysisResult where
  scope : CaseAnalysisScope
  status : CaseAnalysisStatus
  findings : List CaseFinding
  requirements : List CaseRequirement
  observations : List CaseObservation
  propertyEvaluations : List CasePropertyEvaluation
  joint : JointCompatibilityResult
  metadata : PlanningMetadata
  instrumentation : PlannerInstrumentation
  deriving BEq, DecidableEq, Repr

/-- The result is already normalized by stable Property, group, case, and finding identities. -/
def CaseAnalysisResult.canonicalView (result : CaseAnalysisResult) : CaseAnalysisResult := result

private def propertyLe (left right : CheckedProperty) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def caseLe (left right : CaseApplicability) : Bool :=
  decide (left.caseId.value ≤ right.caseId.value)

private def observationKey (observation : CaseObservation) : String :=
  observation.propertyId.value ++ "\u001f" ++ observation.parentId.value ++ "\u001f" ++
    toString observation.transitionPosition ++ "\u001f" ++ reprStr observation.trace

private def observationLe (left right : CaseObservation) : Bool :=
  decide (observationKey left ≤ observationKey right)

private def findingKindIndex : CaseFindingKind → Nat
  | .caseExcluded => 0
  | .missingReplacement => 1
  | .uncoveredCompleteGroup => 2
  | .overlappingExclusiveCases => 3
  | .parentExcluded => 4

private def findingKey (finding : CaseFinding) : String :=
  finding.propertyId.value ++ "\u001f" ++ finding.parentId.value ++ "\u001f" ++
    toString finding.transitionPosition ++ "\u001f" ++ toString (findingKindIndex finding.kind) ++
    "\u001f" ++ String.intercalate "\u001e" (finding.caseIds.map DefinitionId.value) ++
    "\u001f" ++ reprStr finding.trace

private def findingLe (left right : CaseFinding) : Bool :=
  decide (findingKey left ≤ findingKey right)

private def clauseKey (clause : PropertyClauseIdentity) : String :=
  clause.parentId.value ++ "\u001f" ++
    (clause.caseId.map DefinitionId.value).getD "" ++ "\u001f" ++ clause.clauseId.value

private def clauseLe (left right : PropertyClauseIdentity) : Bool :=
  decide (clauseKey left ≤ clauseKey right)

private def queryEvaluationError
    (query : CheckedQuery LawStatement)
    (property : CheckedProperty)
    (error : PropertyError) : QueryError := {
  kind := .propertyEvaluationFailure
  definitionId := query.id
  sourcePath := error.sourcePath
  offendingValue := error.kind.name ++ ":" ++ error.offendingValue
  relatedDefinitionIds := DefinitionId.canonicalSet
    (property.id :: property.guardedClauseIds ++ error.relatedDefinitionIds)
}

private structure AnalysisState where
  observations : List CaseObservation := []
  propertyEvaluations : List CasePropertyEvaluation := []
  jointObservations : List (Scenario.Trace × JointObligationObservation) := []
  admittedTraces : List Scenario.Trace := []
  unresolvedPrefixes : Bool := false

private def observeProperty
    (query : CheckedQuery LawStatement)
    (trace : Scenario.Trace)
    (state : AnalysisState)
    (property : CheckedProperty) : Except QueryError AnalysisState := do
  let input ← checkPropertyEvaluationInput property trace.trace
    |>.mapError (queryEvaluationError query property)
  let evaluation := evaluateProperty property input
  let endpoint := evaluatePropertyEndpoint property input (query.endpoint == .runtimePrefix)
  let observations := (analyzeCaseApplicability property input).map fun applicability =>
    { applicability with
      trace
      cases := applicability.cases.mergeSort caseLe
    }
  pure {
    observations := state.observations ++ observations
    propertyEvaluations := state.propertyEvaluations ++ [{ trace, evaluation, endpointAnswer := endpoint.answer }]
    jointObservations := state.jointObservations ++
      (analyzeJointObligations property input).map fun observation => (trace, observation)
    admittedTraces := state.admittedTraces
    unresolvedPrefixes := state.unresolvedPrefixes || endpoint.answer == .unresolved
  }

private def observeCandidate
    (query : CheckedQuery LawStatement)
    (properties : List CheckedProperty)
    (state : AnalysisState)
    (trace : Scenario.Trace) : Except QueryError (BoundedTraversalStep AnalysisState) := do
  let mut next := state
  for property in properties do
    next ← observeProperty query trace next property
  pure (.continue { next with admittedTraces := next.admittedTraces ++ [trace] })

private def findingBase
    (kind : CaseFindingKind)
    (observation : CaseObservation)
    (cases : List CaseApplicability) : CaseFinding := {
  kind
  propertyId := observation.propertyId
  parentId := observation.parentId
  parentSource := observation.source
  caseIds := cases.map CaseApplicability.caseId
  caseSources := cases.map CaseApplicability.source
  clauses := cases.flatMap CaseApplicability.clauses |>.mergeSort clauseLe
  trace := observation.trace
  transitionPosition := observation.transitionPosition
  priorState := observation.priorState
  selectedAction := observation.selectedAction
  effectiveGuards := if cases.isEmpty then observation.effectiveGuards else
    cases.flatMap CaseApplicability.effectiveGuards
  exceptions := if cases.isEmpty then observation.exceptions else
    cases.flatMap CaseApplicability.exceptions
}

private def findingsAt (observation : CaseObservation) : List CaseFinding :=
  let applicable := observation.cases.filter CaseApplicability.applies
  let excluded := observation.cases.filter CaseApplicability.excluded
  let caseExclusions := excluded.map fun item => findingBase .caseExcluded observation [item]
  let missingReplacement := if observation.parentApplies && applicable.isEmpty &&
      !excluded.isEmpty then
    [findingBase .missingReplacement observation excluded]
  else
    []
  let uncovered := if observation.complete && observation.parentApplies && applicable.isEmpty then
    [findingBase .uncoveredCompleteGroup observation observation.cases]
  else
    []
  let overlap := if observation.exclusive && applicable.length > 1 then
    [findingBase .overlappingExclusiveCases observation applicable]
  else
    []
  let parentExcluded := if observation.parentExcluded then
    [findingBase .parentExcluded observation []]
  else
    []
  caseExclusions ++ missingReplacement ++ uncovered ++ overlap ++ parentExcluded

private def requirementOf
    (property : CheckedProperty)
    (group : ResolvedPropertyCaseGroup)
    (observations : List CaseObservation) : CaseRequirement :=
  let matching := observations.filter fun observation =>
    observation.propertyId == property.id && observation.parentId == group.id
  let cases := matching.flatMap fun observation => observation.cases
  let clauseIdentities := group.cases.flatMap fun item => item.clauseIdentities group
  {
    propertyId := property.id
    parentId := group.id
    source := group.source
    complete := group.complete
    exclusive := group.exclusive
    caseIds := DefinitionId.canonicalSet (group.cases.map ResolvedPropertyCase.id)
    clauses := clauseIdentities.mergeSort clauseLe |>.eraseDups
    parentExercised := matching.any fun observation => observation.parentGuardMatched
    exercisedCaseIds := DefinitionId.canonicalSet
      (cases.filter CaseApplicability.applies |>.map CaseApplicability.caseId)
  }

private def requirementsOf
    (properties : List CheckedProperty)
    (observations : List CaseObservation) : List CaseRequirement :=
  properties.flatMap fun property => property.clauses.filterMap fun clause => match clause with
    | .sameStepCases group => some (requirementOf property group observations)
    | _ => none

private def statusOf
    (traversed : BoundedTraversalResult AnalysisState) : CaseAnalysisStatus :=
  match traversed.termination with
  | .complete false => .unsatisfiable
  | .complete true => if traversed.metadata.completeness.established then
      .exhaustive
    else
      .limitReached
  | .limitReached | .stopped _ _ => .limitReached
  | .invalid error => .invalid error

private def expectationOf
    (observation : JointObligationObservation) : JointExpectationEvidence := {
  propertyId := observation.propertyId
  parentId := observation.parentId
  caseId := observation.caseId
  clauseId := observation.clauseId
  source := observation.source
  transitionPosition := observation.transitionPosition
  triggerCoordinate := observation.triggerCoordinate
  effectiveGuards := observation.effectiveGuards
  exceptions := observation.exceptions
  formula := observation.formula
}

private def expectationKey (expectation : JointExpectationEvidence) : String :=
  expectation.propertyId.value ++ "\u001f" ++ expectation.parentId.value ++ "\u001f" ++
    (expectation.caseId.map DefinitionId.value).getD "" ++ "\u001f" ++
    expectation.clauseId.value ++ "\u001f" ++ toString expectation.transitionPosition ++
    "\u001f" ++ toString expectation.triggerCoordinate ++ "\u001f" ++ reprStr expectation.formula

private def expectationLe (left right : JointExpectationEvidence) : Bool :=
  decide (expectationKey left ≤ expectationKey right)

private def traceLe (left right : Scenario.Trace) : Bool :=
  decide (reprStr left ≤ reprStr right)

private def prefixAt (trace : Scenario.Trace) (transitionPosition : Nat) : Scenario.Trace := {
  trace with trace := {
    trace.trace with
    steps := trace.trace.steps.take (transitionPosition - 1)
  }
}

private def triggerScopeOf
    (limits : QueryLimits)
    (trace : Scenario.Trace)
    (observation : JointObligationObservation) : JointTriggerScope := {
  modeledPrefix := prefixAt trace observation.transitionPosition
  transitionPosition := observation.transitionPosition
  occurrence := observation.triggerOccurrence
  priorState := observation.priorState
  selectedAction := observation.selectedAction
  limits
}

private def triggerKey (trigger : JointTriggerScope) : String :=
  toString trigger.transitionPosition ++ "\u001f" ++ reprStr trigger.modeledPrefix ++ "\u001f" ++
    reprStr trigger.occurrence ++ "\u001f" ++ reprStr trigger.priorState ++ "\u001f" ++
    reprStr trigger.selectedAction

private def triggerLe (left right : JointTriggerScope) : Bool :=
  decide (triggerKey left ≤ triggerKey right)

private def observationAt
    (trigger : JointTriggerScope)
    (item : Scenario.Trace × JointObligationObservation) : Bool :=
  let (trace, observation) := item
  observation.transitionPosition == trigger.transitionPosition &&
    observation.triggerOccurrence == trigger.occurrence &&
    observation.priorState == trigger.priorState &&
    observation.selectedAction == trigger.selectedAction &&
    prefixAt trace observation.transitionPosition == trigger.modeledPrefix

private def traceContinuesTrigger
    (trigger : JointTriggerScope)
    (expectations : List JointExpectationEvidence)
    (observations : List (Scenario.Trace × JointObligationObservation))
    (trace : Scenario.Trace) : Bool :=
  expectations.all fun expectation => observations.any fun item =>
    item.1 == trace && observationAt trigger item && expectationOf item.2 == expectation

private def constraintValues : PropertyAtomConstraint → Option (List PropertyLiteral)
  | .fields _ => none
  | .present => none
  | .equals value => some [value]
  | .oneOf values => some values

private def intersectValues
    (left right : List PropertyLiteral) : List PropertyLiteral :=
  left.filter fun value => right.contains value

private def constraintIntersection
    (constraints : List PropertyAtomConstraint) : Option (List PropertyLiteral) :=
  constraints.foldl (fun current constraint =>
    match current, constraintValues constraint with
    | none, values => values
    | values, none => values
    | some left, some right => some (intersectValues left right)) none

private def atomsOfPredicate : PropertyPredicate → Except JointUnsupportedFormulaClass (List PropertyAtom)
  | .atom atom => if atom.fieldComparison.isSome then throw .propertyClause else pure [atom]
  | .all items => items.flatMapM atomsOfPredicate
  | .any _ => throw .disjunction
  | .not _ => throw .negation

private structure ExpectationAtom where
  expectation : JointExpectationEvidence
  atom : PropertyAtom

private def sameStepAtoms
    (expectations : List JointExpectationEvidence) : List ExpectationAtom :=
  expectations.flatMap fun expectation => match expectation.formula with
    | .sameStep predicate => match atomsOfPredicate predicate.expression with
        | .ok atoms => atoms.map fun atom => { expectation, atom }
        | .error _ => []
    | .guardedTemporal .. => []

private def atomDomainKey (atom : PropertyAtom) : String :=
  atom.field.name ++ "\u001f" ++ atom.reference.value

private def scalarExpectationField : PropertyPredicateField → Bool
  | .resultingState | .modelOutcome => true
  | .priorState | .selectedAction | .expectationFact => false

private def logicalConflictsAt
    (trigger : JointTriggerScope)
    (expectations : List JointExpectationEvidence) : List JointConflictEvidence :=
  let atoms := (sameStepAtoms expectations).filter fun item =>
    scalarExpectationField item.atom.field
  let domains := atoms.map (atomDomainKey ·.atom) |>.mergeSort (fun left right => decide (left ≤ right))
    |>.eraseDups
  domains.filterMap fun domain =>
    let selected := atoms.filter fun item => atomDomainKey item.atom == domain
    let constraints := selected.map (·.atom.constraint)
    match constraintIntersection constraints with
    | some [] => selected.head?.map fun first => {
        trigger
        field := first.atom.field
        reference := first.atom.reference
        constraints
        expectations := selected.map (·.expectation) |>.mergeSort expectationLe |>.eraseDups
      }
    | _ => none

private def unsupportedPredicateClasses
    (predicate : PropertyPredicate) : List JointUnsupportedFormulaClass :=
  match predicate with
  | .atom atom => if atom.fieldComparison.isSome then [.propertyClause] else []
  | .all items => items.flatMap unsupportedPredicateClasses
  | .any items => .disjunction :: items.flatMap unsupportedPredicateClasses
  | .not item => .negation :: unsupportedPredicateClasses item

private def unsupportedOfProperty (property : CheckedProperty) : List UnsupportedJointFormula :=
  property.clauses.flatMap fun clause => match clause with
  | .sameStepCases group => group.cases.flatMap fun item =>
      item.clauses.flatMap fun sameStep =>
        (unsupportedPredicateClasses sameStep.expectation.expression).map fun formulaClass => {
          propertyId := property.id
          clauseId := sameStep.id
          source := sameStep.source
          formulaClass
          formula := some sameStep.expectation.expression
        }
  | .guardedEventuallyWithin _ | .guardedQuiescentWithin _ => []
  | other => [{
      propertyId := property.id
      clauseId := other.id
      source := property.source
      formulaClass := .propertyClause
      formula := none
    }]

private def unsupportedKey (unsupported : UnsupportedJointFormula) : String :=
  unsupported.propertyId.value ++ "\u001f" ++ unsupported.clauseId.value ++ "\u001f" ++
    unsupported.formulaClass.name

private def unsupportedLe (left right : UnsupportedJointFormula) : Bool :=
  decide (unsupportedKey left ≤ unsupportedKey right)

private def expectationsAt
    (trigger : JointTriggerScope)
    (observations : List (Scenario.Trace × JointObligationObservation)) :
    List JointExpectationEvidence :=
  ((observations.filter (observationAt trigger)).map (expectationOf ·.2)).mergeSort expectationLe
    |>.eraseDups

private def continuationSatisfies
    (trigger : JointTriggerScope)
    (expectations : List JointExpectationEvidence)
    (observations : List (Scenario.Trace × JointObligationObservation))
    (trace : Scenario.Trace) : Bool :=
  expectations.all fun expectation => observations.any fun item =>
    item.1 == trace && observationAt trigger item &&
      expectationOf item.2 == expectation && item.2.satisfied

private def selectedAtTrigger (expectations : List JointExpectationEvidence) : Bool :=
  expectations.length > 1

private def jointResult
    (query : CheckedQuery LawStatement)
    (properties : List CheckedProperty)
    (traversed : BoundedTraversalResult AnalysisState) : JointCompatibilityResult :=
  let observations := traversed.state.jointObservations
  let unsupported := (properties.flatMap unsupportedOfProperty).mergeSort unsupportedLe |>.eraseDups
  let scopes := (observations.map fun item => triggerScopeOf query.limits item.1 item.2)
    |>.mergeSort triggerLe |>.eraseDups
  let triggers := scopes.filterMap fun trigger =>
    let expectations := expectationsAt trigger observations
    if selectedAtTrigger expectations then
      let continuations := (traversed.state.admittedTraces.filter
          (traceContinuesTrigger trigger expectations observations))
        |>.mergeSort traceLe |>.eraseDups
      let satisfying := continuations.filter
        (continuationSatisfies trigger expectations observations)
      some ({
        trigger := trigger
        expectations := expectations
        admittedContinuations := continuations
        satisfyingContinuations := satisfying
      } : JointTriggerAnalysis)
    else
      none
  let logicalConflicts := triggers.flatMap fun trigger =>
    logicalConflictsAt trigger.trigger trigger.expectations
  let modelIncompatibilities := if traversed.metadata.completeness.established &&
      !traversed.state.unresolvedPrefixes then
      triggers.filterMap fun trigger =>
        if trigger.admittedContinuations.isEmpty || !trigger.satisfyingContinuations.isEmpty ||
            !(logicalConflictsAt trigger.trigger trigger.expectations).isEmpty then
          none
        else
          some {
            trigger := trigger.trigger
            expectations := trigger.expectations
            admittedContinuations := trigger.admittedContinuations
          }
    else
      []
  let status := match traversed.termination with
    | .invalid error => JointCompatibilityStatus.invalid error
    | .limitReached | .stopped _ _ => .limitReached
    | .complete false => if query.behavior.isUnsatisfiable then .unsatisfiable else .deadEnd
    | .complete true =>
        if !traversed.metadata.completeness.established || traversed.state.unresolvedPrefixes then .limitReached
        else if !logicalConflicts.isEmpty then .logicalConflict
        else if !modelIncompatibilities.isEmpty then .modelIncompatible
        else if !unsupported.isEmpty then .unsupported
        else if triggers.isEmpty then .unexercised
        else .compatibleWithinLimits
  { status, triggers, logicalConflicts, modelIncompatibilities, unsupported }

/-- Analyze guarded cases over exactly one checked Query and its admitted finite planner kernel. -/
def analyzeCases
    (query : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel query.target) : CaseAnalysisResult :=
  let properties := query.form.properties.mergeSort propertyLe
  let traversed := traverseBoundedCandidates query kernel {} (observeCandidate query properties)
  let observations := traversed.state.observations.mergeSort observationLe
  {
    scope := {
      queryId := query.id
      queryFingerprint := query.behaviorFingerprint
      targetId := query.target.id
      targetFingerprint := query.target.behaviorFingerprint
      behaviorId := query.behavior.id
      behaviorFingerprint := query.behavior.behaviorFingerprint
      properties := properties.map fun property => {
        id := property.id
        behaviorFingerprint := property.behaviorFingerprint
      }
      limits := query.limits
      endpoint := query.endpoint
      exercise := query.exercise
    }
    status := statusOf traversed
    findings := observations.flatMap findingsAt |>.mergeSort findingLe
    requirements := requirementsOf properties observations
    observations
    propertyEvaluations := traversed.state.propertyEvaluations
    joint := jointResult query properties traversed
    metadata := traversed.metadata
    instrumentation := traversed.instrumentation
  }

end Umpire

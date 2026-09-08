import Umpire.Property.Trace

/-! Executable and denotational semantics for checked Properties over admitted trace views. -/

namespace Umpire

def ValueConstraint.denote (constraint : ValueConstraint) (value : String) : Prop :=
  match constraint with
  | .present => True
  | .equals expected => value = expected
  | .notEquals rejected => value ≠ rejected
  | .naturalAtMost maximum =>
      match value.toNat? with
      | some actual => actual ≤ maximum
      | none => False
  | .naturalAtLeast minimum =>
      match value.toNat? with
      | some actual => minimum ≤ actual
      | none => False

def ValueConstraint.evaluate (constraint : ValueConstraint) (value : String) : Bool :=
  match constraint with
  | .present => true
  | .equals expected => value == expected
  | .notEquals rejected => value != rejected
  | .naturalAtMost maximum => value.toNat?.any fun actual => decide (actual ≤ maximum)
  | .naturalAtLeast minimum => value.toNat?.any fun actual => decide (minimum ≤ actual)

theorem ValueConstraint.evaluate_agrees
    (constraint : ValueConstraint)
    (value : String) :
    constraint.evaluate value = true ↔ constraint.denote value := by
  cases constraint with
  | present => simp [ValueConstraint.evaluate, ValueConstraint.denote]
  | equals expected => simp [ValueConstraint.evaluate, ValueConstraint.denote]
  | notEquals rejected => simp [ValueConstraint.evaluate, ValueConstraint.denote]
  | naturalAtMost maximum =>
      cases parsed : value.toNat? <;>
        simp [ValueConstraint.evaluate, ValueConstraint.denote, parsed]
  | naturalAtLeast minimum =>
      cases parsed : value.toNat? <;>
        simp [ValueConstraint.evaluate, ValueConstraint.denote, parsed]

def PropertyPattern.denote (pattern : PropertyPattern) (value : ModelValue) : Prop :=
  value.definitionId = pattern.reference ∧ pattern.constraint.denote value.value

def PropertyPattern.evaluate (pattern : PropertyPattern) (value : ModelValue) : Bool :=
  decide (value.definitionId = pattern.reference) && pattern.constraint.evaluate value.value

theorem PropertyPattern.evaluate_agrees
    (pattern : PropertyPattern)
    (value : ModelValue) :
    pattern.evaluate value = true ↔ pattern.denote value := by
  simp [PropertyPattern.evaluate, PropertyPattern.denote, ValueConstraint.evaluate_agrees]

private def allHolds {α : Type} : List α → (α → Prop) → Prop
  | [], _ => True
  | item :: rest, predicate => predicate item ∧ allHolds rest predicate

private def anyHolds {α : Type} : List α → (α → Prop) → Prop
  | [], _ => False
  | item :: rest, predicate => predicate item ∨ anyHolds rest predicate

private theorem allHolds_agrees
    (items : List α)
    (evaluate : α → Bool)
    (denote : α → Prop)
    (agreement : ∀ item, evaluate item = true ↔ denote item) :
    items.all evaluate = true ↔ allHolds items denote := by
  induction items with
  | nil => simp [allHolds]
  | cons item rest inductionHypothesis =>
      simp [allHolds, agreement item, inductionHypothesis]

private theorem anyHolds_agrees
    (items : List α)
    (evaluate : α → Bool)
    (denote : α → Prop)
    (agreement : ∀ item, evaluate item = true ↔ denote item) :
    items.any evaluate = true ↔ anyHolds items denote := by
  induction items with
  | nil => simp [anyHolds]
  | cons item rest inductionHypothesis =>
      simp [anyHolds, agreement item, inductionHypothesis]

private theorem booleanNot_agrees
    (value : Bool)
    (proposition : Prop)
    (agreement : value = true ↔ proposition) :
    (!value) = true ↔ ¬proposition := by
  cases value <;> simp_all

/-- Propositional interpretation of one typed literal against its portable payload. -/
private def PropertyLiteral.denote (literal : PropertyLiteral) (payload : String) : Prop :=
  match literal with
  | .text expected => payload = expected
  | .natural expected => payload.toNat? = some expected
  | .boolean expected => payload = toString expected

/-- Executable interpretation of one typed literal against its portable payload. -/
private def PropertyLiteral.evaluate (literal : PropertyLiteral) (payload : String) : Bool :=
  match literal with
  | .text expected => payload == expected
  | .natural expected => payload.toNat? == some expected
  | .boolean expected => payload == toString expected

/-- Typed literal evaluation agrees with its denotation. -/
private theorem PropertyLiteral.evaluate_agrees
    (literal : PropertyLiteral)
    (payload : String) :
    literal.evaluate payload = true ↔ literal.denote payload := by
  cases literal <;> simp [PropertyLiteral.evaluate, PropertyLiteral.denote]

/-- Propositional interpretation of a closed atomic comparison. -/
private def PropertyAtomConstraint.denote
    (constraint : PropertyAtomConstraint)
    (payload : String) : Prop :=
  match constraint with
  | .present => True
  | .equals literal => literal.denote payload
  | .oneOf literals => anyHolds literals fun literal => literal.denote payload

/-- Executable interpretation of a closed atomic comparison. -/
private def PropertyAtomConstraint.evaluate
    (constraint : PropertyAtomConstraint)
    (payload : String) : Bool :=
  match constraint with
  | .present => true
  | .equals literal => literal.evaluate payload
  | .oneOf literals => literals.any fun literal => literal.evaluate payload

/-- Atomic comparison evaluation agrees with its denotation. -/
private theorem PropertyAtomConstraint.evaluate_agrees
    (constraint : PropertyAtomConstraint)
    (payload : String) :
    constraint.evaluate payload = true ↔ constraint.denote payload := by
  cases constraint with
  | present => simp [PropertyAtomConstraint.evaluate, PropertyAtomConstraint.denote]
  | equals literal =>
      exact literal.evaluate_agrees payload
  | oneOf literals =>
      exact anyHolds_agrees _ _ _ fun literal => literal.evaluate_agrees payload

/-- Propositional interpretation of an atom over one same-step field projection. -/
private def PropertyAtom.denote
    (atom : PropertyAtom)
    (values : PropertyPredicateField → List ModelValue) : Prop :=
  anyHolds (values atom.field) fun value =>
    value.definitionId = atom.reference ∧ atom.constraint.denote value.value

/-- Executable interpretation of an atom over one same-step field projection. -/
private def PropertyAtom.evaluate
    (atom : PropertyAtom)
    (values : PropertyPredicateField → List ModelValue) : Bool :=
  (values atom.field).any fun value =>
    decide (value.definitionId = atom.reference) && atom.constraint.evaluate value.value

/-- Atomic field evaluation agrees with its denotation. -/
private theorem PropertyAtom.evaluate_agrees
    (atom : PropertyAtom)
    (values : PropertyPredicateField → List ModelValue) :
    atom.evaluate values = true ↔ atom.denote values :=
  anyHolds_agrees _ _ _ fun value => by
    simp [PropertyAtomConstraint.evaluate_agrees]

mutual
  /-- Propositional interpretation of the portable Boolean predicate kernel. -/
  private def PropertyPredicate.denote
      (predicate : PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) : Prop :=
    match predicate with
    | .atom value => value.denote values
    | .all items => propertyPredicateAllDenote items values
    | .any items => propertyPredicateAnyDenote items values
    | .not item => ¬item.denote values

  private def propertyPredicateAllDenote
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) : Prop :=
    match items with
    | [] => True
    | item :: rest => item.denote values ∧ propertyPredicateAllDenote rest values

  private def propertyPredicateAnyDenote
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) : Prop :=
    match items with
    | [] => False
    | item :: rest => item.denote values ∨ propertyPredicateAnyDenote rest values
end

mutual
  /-- Executable interpretation of the portable Boolean predicate kernel. -/
  private def PropertyPredicate.evaluate
      (predicate : PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) : Bool :=
    match predicate with
    | .atom value => value.evaluate values
    | .all items => propertyPredicateAllEvaluate items values
    | .any items => propertyPredicateAnyEvaluate items values
    | .not item => !(item.evaluate values)

  private def propertyPredicateAllEvaluate
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) : Bool :=
    match items with
    | [] => true
    | item :: rest => item.evaluate values && propertyPredicateAllEvaluate rest values

  private def propertyPredicateAnyEvaluate
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) : Bool :=
    match items with
    | [] => false
    | item :: rest => item.evaluate values || propertyPredicateAnyEvaluate rest values
end

mutual
  /-- Structural evaluator/denotation agreement for every Boolean predicate constructor. -/
  private theorem PropertyPredicate.evaluate_agrees
      (predicate : PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) :
      predicate.evaluate values = true ↔ predicate.denote values := by
    cases predicate with
    | atom value => exact value.evaluate_agrees values
    | all items => exact propertyPredicateAllEvaluate_agrees items values
    | any items => exact propertyPredicateAnyEvaluate_agrees items values
    | not item =>
        simpa [PropertyPredicate.evaluate, PropertyPredicate.denote] using
          booleanNot_agrees (item.evaluate values) (item.denote values)
            (PropertyPredicate.evaluate_agrees item values)

  private theorem propertyPredicateAllEvaluate_agrees
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) :
      propertyPredicateAllEvaluate items values = true ↔
        propertyPredicateAllDenote items values := by
    cases items with
    | nil => simp [propertyPredicateAllEvaluate, propertyPredicateAllDenote]
    | cons item rest =>
        simp [propertyPredicateAllEvaluate, propertyPredicateAllDenote,
          PropertyPredicate.evaluate_agrees item values,
          propertyPredicateAllEvaluate_agrees rest values]

  private theorem propertyPredicateAnyEvaluate_agrees
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue) :
      propertyPredicateAnyEvaluate items values = true ↔
        propertyPredicateAnyDenote items values := by
    cases items with
    | nil => simp [propertyPredicateAnyEvaluate, propertyPredicateAnyDenote]
    | cons item rest =>
        simp [propertyPredicateAnyEvaluate, propertyPredicateAnyDenote,
          PropertyPredicate.evaluate_agrees item values,
          propertyPredicateAnyEvaluate_agrees rest values]
end

/-- Denotational meaning of a validated predicate over one validated same-step input. -/
def CheckedPropertyPredicate.denote
    {predicateContext : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate predicateContext)
    (input : CheckedPropertyPredicateInput predicate) : Prop :=
  predicate.expression.denote input.valuesAt

/-- Evaluate a predicate only after both its syntax and complete same-step input passed checking. -/
def evaluatePropertyPredicate
    {predicateContext : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate predicateContext)
    (input : CheckedPropertyPredicateInput predicate) : Bool :=
  predicate.expression.evaluate input.valuesAt

/-- Generic kernel-checked agreement for every admitted Boolean Property predicate. -/
theorem evaluatePropertyPredicate_agrees
    {predicateContext : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate predicateContext)
    (input : CheckedPropertyPredicateInput predicate) :
    evaluatePropertyPredicate predicate input = true ↔ predicate.denote input :=
  predicate.expression.evaluate_agrees input.valuesAt

/-- A trace view whose every guarded same-step input was validated for this exact Property. -/
structure CheckedPropertyEvaluationInput (property : CheckedProperty) where
  private view : PropertyTraceView
  deriving Repr

private def guardInput (step : PropertyTraceStep) : PropertyPredicateInput := {
  context := .guard
  priorState := step.priorState
  selectedAction := step.selectedAction
}

private def expectationInput (step : PropertyTraceStep) : PropertyPredicateInput := {
  context := .expectation
  resultingState := step.resultingState
  modelOutcome := step.modelOutcome
  facts := some step.observations
}

private def valuesInStep
    (field : PropertyTraceField)
    (step : PropertyTraceStep) : List ModelValue :=
  match field with
  | .state | .resultingState => step.resultingState.toList
  | .priorState => step.priorState.toList
  | .selectedAction => step.selectedAction.toList
  | .modelOutcome => step.modelOutcome.toList
  | .observation | .relation => step.observations

private def patternHoldsInStep
    (pattern : PropertyPattern)
    (step : PropertyTraceStep) : Bool :=
  (valuesInStep pattern.field step).any pattern.evaluate

private def validateResolvedExceptionInput
    (exception : Option ResolvedPropertyException)
    (input : PropertyPredicateInput) : Except PropertyError Unit := do
  match exception with
  | none => pure ()
  | some exception =>
      let _ ← checkPropertyPredicateInput exception.condition input
      pure ()

private def validateCaseGroupStep
    (group : ResolvedPropertyCaseGroup)
    (step : PropertyTraceStep) : Except PropertyError Unit := do
  let guardValues := guardInput step
  let expectationValues := expectationInput step
  let _ ← checkPropertyPredicateInput group.guard guardValues
  validateResolvedExceptionInput group.exception guardValues
  for item in group.cases do
    let _ ← checkPropertyPredicateInput item.guard guardValues
    validateResolvedExceptionInput item.exception guardValues
    for clause in item.clauses do
      let _ ← checkPropertyPredicateInput clause.expectation expectationValues
      pure ()

private def validateGuardedTemporalStep
    (clause : ResolvedGuardedTemporalClause)
    (step : PropertyTraceStep) : Except PropertyError Unit := do
  if patternHoldsInStep clause.trigger step then
    let input := guardInput step
    let _ ← checkPropertyPredicateInput clause.guard input
    validateResolvedExceptionInput clause.exception input
    for guard in clause.caseGuard do
      let _ ← checkPropertyPredicateInput guard input
      pure ()
    validateResolvedExceptionInput clause.caseException input

/-- Validate every same-step slot before guarded Boolean reduction begins. Known absent facts are
represented by `some []`; only unavailable fields produce an error. -/
def checkPropertyEvaluationInput
    (property : CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Except PropertyError (CheckedPropertyEvaluationInput property) := do
  let view := property.traceView trace
  for clause in property.clauses do
    match clause with
    | .sameStepCases group =>
        if group.complete || group.exclusive || group.cases.any (fun item => !item.clauses.isEmpty) then
          for step in view.steps do
            validateCaseGroupStep group step
        for item in group.cases do
          for temporal in item.temporalClauses do
            for step in view.steps do
              validateGuardedTemporalStep temporal step
    | .guardedEventuallyWithin guarded | .guardedQuiescentWithin guarded =>
        for step in view.steps do
          validateGuardedTemporalStep guarded step
    | _ => pure ()
  pure { view }

private def predicateValues
    (input : PropertyPredicateInput)
    (field : PropertyPredicateField) : List ModelValue :=
  match field with
  | .priorState => input.priorState.toList
  | .selectedAction => input.selectedAction.toList
  | .resultingState => input.resultingState.toList
  | .modelOutcome => input.modelOutcome.toList
  | .expectationFact => input.facts.getD []

private def evaluateCheckedPredicate
    (predicate : CheckedPropertyPredicate context)
    (input : PropertyPredicateInput) : Bool :=
  predicate.expression.evaluate (predicateValues input)

private def checkedPredicateDenotes
    (predicate : CheckedPropertyPredicate context)
    (input : PropertyPredicateInput) : Prop :=
  predicate.expression.denote (predicateValues input)

private theorem evaluateCheckedPredicate_agrees
    (predicate : CheckedPropertyPredicate context)
    (input : PropertyPredicateInput) :
    evaluateCheckedPredicate predicate input = true ↔ checkedPredicateDenotes predicate input :=
  predicate.expression.evaluate_agrees (predicateValues input)

private theorem booleanImplication_agrees
    (left right : Bool)
    (antecedent consequent : Prop)
    (leftAgreement : left = true ↔ antecedent)
    (rightAgreement : right = true ↔ consequent) :
    (!left || right) = true ↔ (antecedent → consequent) := by
  cases left <;> cases right <;> simp_all

structure PropertyOccurrence where
  value : ModelValue
  transitionPosition : Nat
  selectedActionPosition : Nat
  observationPosition : Nat
  logicalTime : Option Nat
  deriving BEq, DecidableEq, Repr

private def observationOccurrences
    (pattern : PropertyPattern)
    (transitionPosition selectedActionPosition observationOffset : Nat)
    (logicalTime : Option Nat) : List ModelValue → List PropertyOccurrence
  | [] => []
  | value :: rest =>
      let tail := observationOccurrences pattern transitionPosition selectedActionPosition
        (observationOffset + 1) logicalTime rest
      if pattern.evaluate value then
        {
          value
          transitionPosition
          selectedActionPosition
          observationPosition := observationOffset + 1
          logicalTime
        } :: tail
      else
        tail

private def optionalOccurrence
    (pattern : PropertyPattern)
    (transitionPosition selectedActionPosition observationPosition : Nat)
    (logicalTime : Option Nat)
    (value : Option ModelValue) : List PropertyOccurrence :=
  match value with
  | some value =>
      if pattern.evaluate value then [{
        value
        transitionPosition
        selectedActionPosition
        observationPosition
        logicalTime
      }] else []
  | none => []

private def stepOccurrences
    (pattern : PropertyPattern)
    (transitionPosition observationOffset : Nat)
    (step : PropertyTraceStep) : List PropertyOccurrence :=
  match pattern.field with
  | .state | .resultingState =>
      optionalOccurrence pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.resultingState
  | .priorState =>
      optionalOccurrence pattern (transitionPosition - 1) transitionPosition observationOffset
        step.logicalTime step.priorState
  | .selectedAction =>
      optionalOccurrence pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.selectedAction
  | .modelOutcome =>
      optionalOccurrence pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.modelOutcome
  | .observation | .relation =>
      observationOccurrences pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.observations

private def traceStepOccurrences
    (pattern : PropertyPattern)
    (transitionPosition observationOffset : Nat) :
    List PropertyTraceStep → List PropertyOccurrence
  | [] => []
  | step :: rest =>
      stepOccurrences pattern transitionPosition observationOffset step ++
        traceStepOccurrences pattern (transitionPosition + 1)
          (observationOffset + step.observations.length) rest

private def occurrences
    (pattern : PropertyPattern)
    (view : PropertyTraceView) : List PropertyOccurrence :=
  let initial := if pattern.field == .state then
    optionalOccurrence pattern 0 0 0 none view.initialState
  else
    []
  initial ++ traceStepOccurrences pattern 1 0 view.steps

private def positionOf
    (unit : LimitUnit)
    (occurrence : PropertyOccurrence) : Option Nat :=
  match unit with
  | .semanticTransitions => some occurrence.transitionPosition
  | .selectedActions => some occurrence.selectedActionPosition
  | .observationPositions => some occurrence.observationPosition
  | .logicalTime => occurrence.logicalTime
  | .candidateEvaluations => none
  | .experimentSpecs => none

private def collectPositions : List (Option Nat) → Option (List Nat)
  | [] => some []
  | none :: _ => none
  | some position :: rest =>
      (collectPositions rest).map fun positions => position :: positions

/-- Preserve the distinction between no matching occurrences and matching occurrences whose
requested coordinate is missing. In particular, logical-time evaluation must fail closed. -/
private def checkedPositions
    (pattern : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) : Option (List Nat) :=
  collectPositions ((occurrences pattern view).map (positionOf unit))

private def valuesAtField
    (field : PropertyTraceField)
    (view : PropertyTraceView) : List ModelValue :=
  let initial := match field with
    | .state => view.initialState.toList
    | _ => []
  let fromSteps := view.steps.flatMap fun step =>
    match field with
    | .state | .resultingState => step.resultingState.toList
    | .priorState => step.priorState.toList
    | .selectedAction => step.selectedAction.toList
    | .modelOutcome => step.modelOutcome.toList
    | .observation | .relation => step.observations
  initial ++ fromSteps

private def patternDenotesInStep
    (pattern : PropertyPattern)
    (step : PropertyTraceStep) : Prop :=
  anyHolds (valuesInStep pattern.field step) pattern.denote

private theorem patternHoldsInStep_agrees
    (pattern : PropertyPattern)
    (step : PropertyTraceStep) :
    patternHoldsInStep pattern step = true ↔ patternDenotesInStep pattern step :=
  anyHolds_agrees _ _ _ pattern.evaluate_agrees

private def evaluateStateInvariant
    (pattern : PropertyPattern)
    (view : PropertyTraceView) : Bool :=
  let matching := (valuesAtField .state view).filter fun value => value.definitionId == pattern.reference
  !matching.isEmpty && matching.all fun value => pattern.constraint.evaluate value.value

private def stateInvariantDenotes
    (pattern : PropertyPattern)
    (view : PropertyTraceView) : Prop :=
  let matching := (valuesAtField .state view).filter fun value => value.definitionId == pattern.reference
  matching ≠ [] ∧ allHolds matching fun value => pattern.constraint.denote value.value

private theorem evaluateStateInvariant_agrees
    (pattern : PropertyPattern)
    (view : PropertyTraceView) :
    evaluateStateInvariant pattern view = true ↔ stateInvariantDenotes pattern view := by
  let matching := (valuesAtField .state view).filter fun value =>
    value.definitionId == pattern.reference
  have constraintsAgree :
      matching.all (fun value => pattern.constraint.evaluate value.value) = true ↔
        allHolds matching (fun value => pattern.constraint.denote value.value) :=
    allHolds_agrees _ _ _ fun value => pattern.constraint.evaluate_agrees value.value
  change (!matching.isEmpty && matching.all
    (fun value => pattern.constraint.evaluate value.value)) = true ↔
      matching ≠ [] ∧ allHolds matching (fun value => pattern.constraint.denote value.value)
  simp [constraintsAgree]

private def evaluateTransitionContract
    (precondition postcondition : PropertyPattern)
    (view : PropertyTraceView) : Bool :=
  view.steps.all fun step =>
    !patternHoldsInStep precondition step || patternHoldsInStep postcondition step

private def transitionContractDenotes
    (precondition postcondition : PropertyPattern)
    (view : PropertyTraceView) : Prop :=
  allHolds view.steps fun step =>
    patternDenotesInStep precondition step → patternDenotesInStep postcondition step

private theorem evaluateTransitionContract_agrees
    (precondition postcondition : PropertyPattern)
    (view : PropertyTraceView) :
    evaluateTransitionContract precondition postcondition view = true ↔
      transitionContractDenotes precondition postcondition view :=
  allHolds_agrees _ _ _ fun step =>
    booleanImplication_agrees _ _ _ _
      (patternHoldsInStep_agrees precondition step)
      (patternHoldsInStep_agrees postcondition step)

private def evaluateIdentityRelation
    (relation : PropertyPattern)
    (view : PropertyTraceView) : Bool :=
  (valuesAtField relation.field view).any relation.evaluate

private def identityRelationDenotes
    (relation : PropertyPattern)
    (view : PropertyTraceView) : Prop :=
  anyHolds (valuesAtField relation.field view) relation.denote

private theorem evaluateIdentityRelation_agrees
    (relation : PropertyPattern)
    (view : PropertyTraceView) :
    evaluateIdentityRelation relation view = true ↔ identityRelationDenotes relation view :=
  anyHolds_agrees _ _ _ relation.evaluate_agrees

private def evaluateOrdered
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) : Bool :=
  match checkedPositions before unit view, checkedPositions after unit view with
  | some beforePositions, some afterPositions =>
      beforePositions.any fun first => afterPositions.any fun second => first < second
  | _, _ => false

private def orderedDenotes
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) : Prop :=
  match checkedPositions before unit view, checkedPositions after unit view with
  | some beforePositions, some afterPositions =>
      anyHolds beforePositions fun first => anyHolds afterPositions fun second => first < second
  | _, _ => False

private theorem evaluateOrdered_agrees
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) :
    evaluateOrdered before after unit view = true ↔ orderedDenotes before after unit view := by
  cases beforeResult : checkedPositions before unit view with
  | none => simp [evaluateOrdered, orderedDenotes, beforeResult]
  | some beforePositions =>
      cases afterResult : checkedPositions after unit view with
      | none => simp [evaluateOrdered, orderedDenotes, beforeResult, afterResult]
      | some afterPositions =>
          have afterAgreement (first : Nat) :
              afterPositions.any (fun second => first < second) = true ↔
                anyHolds afterPositions (fun second => first < second) :=
            anyHolds_agrees _ _ _ fun second => by simp
          have beforeAgreement :
              beforePositions.any (fun first =>
                afterPositions.any fun second => first < second) = true ↔
                anyHolds beforePositions (fun first =>
                  anyHolds afterPositions fun second => first < second) :=
            anyHolds_agrees _ _ _ afterAgreement
          simpa [evaluateOrdered, orderedDenotes, beforeResult, afterResult] using beforeAgreement

private def evaluateEventuallyWithin
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Bool :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggerPositions, some responsePositions =>
      triggerPositions.all fun first =>
        responsePositions.any fun second => first ≤ second && second - first ≤ limit.value
  | _, _ => false

private def eventuallyWithinDenotes
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Prop :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggerPositions, some responsePositions =>
      allHolds triggerPositions fun first =>
        anyHolds responsePositions fun second =>
          first ≤ second ∧ second - first ≤ limit.value
  | _, _ => False

private theorem evaluateEventuallyWithin_agrees
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) :
    evaluateEventuallyWithin trigger response limit view = true ↔
      eventuallyWithinDenotes trigger response limit view := by
  cases triggerResult : checkedPositions trigger limit.unit view with
  | none => simp [evaluateEventuallyWithin, eventuallyWithinDenotes, triggerResult]
  | some triggerPositions =>
      cases responseResult : checkedPositions response limit.unit view with
      | none =>
          simp [evaluateEventuallyWithin, eventuallyWithinDenotes, triggerResult, responseResult]
      | some responsePositions =>
          have responseAgreement (first : Nat) :
              responsePositions.any (fun second =>
                first ≤ second && second - first ≤ limit.value) = true ↔
                anyHolds responsePositions (fun second =>
                  first ≤ second ∧ second - first ≤ limit.value) :=
            anyHolds_agrees _ _ _ fun second => by simp
          have triggerAgreement :
              triggerPositions.all (fun first =>
                responsePositions.any fun second =>
                  first ≤ second && second - first ≤ limit.value) = true ↔
                allHolds triggerPositions (fun first =>
                  anyHolds responsePositions fun second =>
                    first ≤ second ∧ second - first ≤ limit.value) :=
            allHolds_agrees _ _ _ responseAgreement
          simpa [evaluateEventuallyWithin, eventuallyWithinDenotes,
            triggerResult, responseResult] using triggerAgreement

private def evaluateQuiescentWithin
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Bool :=
  match checkedPositions trigger limit.unit view, checkedPositions forbidden limit.unit view with
  | some triggerPositions, some forbiddenPositions =>
      triggerPositions.all fun first =>
        !(forbiddenPositions.any fun second => first ≤ second && second - first ≤ limit.value)
  | _, _ => false

private def quiescentWithinDenotes
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Prop :=
  match checkedPositions trigger limit.unit view, checkedPositions forbidden limit.unit view with
  | some triggerPositions, some forbiddenPositions =>
      allHolds triggerPositions fun first =>
        ¬anyHolds forbiddenPositions fun second =>
          first ≤ second ∧ second - first ≤ limit.value
  | _, _ => False

private theorem evaluateQuiescentWithin_agrees
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) :
    evaluateQuiescentWithin trigger forbidden limit view = true ↔
      quiescentWithinDenotes trigger forbidden limit view := by
  cases triggerResult : checkedPositions trigger limit.unit view with
  | none => simp [evaluateQuiescentWithin, quiescentWithinDenotes, triggerResult]
  | some triggerPositions =>
      cases forbiddenResult : checkedPositions forbidden limit.unit view with
      | none =>
          simp [evaluateQuiescentWithin, quiescentWithinDenotes, triggerResult, forbiddenResult]
      | some forbiddenPositions =>
          have forbiddenAgreement (first : Nat) :
              forbiddenPositions.any (fun second =>
                first ≤ second && second - first ≤ limit.value) = true ↔
                anyHolds forbiddenPositions (fun second =>
                  first ≤ second ∧ second - first ≤ limit.value) :=
            anyHolds_agrees _ _ _ fun second => by simp
          have absenceAgreement (first : Nat) :
              Bool.not (forbiddenPositions.any fun second =>
                first ≤ second && second - first ≤ limit.value) = true ↔
                ¬anyHolds forbiddenPositions (fun second =>
                  first ≤ second ∧ second - first ≤ limit.value) :=
            booleanNot_agrees _ _ (forbiddenAgreement first)
          have triggerAgreement :
              triggerPositions.all (fun first =>
                !(forbiddenPositions.any fun second =>
                  first ≤ second && second - first ≤ limit.value)) = true ↔
                allHolds triggerPositions (fun first =>
                  ¬anyHolds forbiddenPositions fun second =>
                    first ≤ second ∧ second - first ≤ limit.value) :=
            allHolds_agrees _ _ _ absenceAgreement
          simpa [evaluateQuiescentWithin, quiescentWithinDenotes,
            triggerResult, forbiddenResult] using triggerAgreement

private def exceptionAllowsEvaluate
    (exception : Option ResolvedPropertyException)
    (input : PropertyPredicateInput) : Bool :=
  match exception with
  | none => true
  | some exception => !(evaluateCheckedPredicate exception.condition input)

private def exceptionAllowsDenote
    (exception : Option ResolvedPropertyException)
    (input : PropertyPredicateInput) : Prop :=
  match exception with
  | none => True
  | some exception => ¬checkedPredicateDenotes exception.condition input

private theorem exceptionAllows_agrees
    (exception : Option ResolvedPropertyException)
    (input : PropertyPredicateInput) :
    exceptionAllowsEvaluate exception input = true ↔ exceptionAllowsDenote exception input := by
  cases exception with
  | none => simp [exceptionAllowsEvaluate, exceptionAllowsDenote]
  | some exception =>
      exact booleanNot_agrees _ _ (evaluateCheckedPredicate_agrees exception.condition input)

private def guardedTemporalAppliesEvaluate
    (clause : ResolvedGuardedTemporalClause)
    (input : PropertyPredicateInput) : Bool :=
  evaluateCheckedPredicate clause.guard input &&
    exceptionAllowsEvaluate clause.exception input &&
    clause.caseGuard.all (fun guard => evaluateCheckedPredicate guard input) &&
    exceptionAllowsEvaluate clause.caseException input

private def guardedTemporalAppliesDenote
    (clause : ResolvedGuardedTemporalClause)
    (input : PropertyPredicateInput) : Prop :=
  checkedPredicateDenotes clause.guard input ∧
    exceptionAllowsDenote clause.exception input ∧
    (∀ guard ∈ clause.caseGuard, checkedPredicateDenotes guard input) ∧
    exceptionAllowsDenote clause.caseException input

private theorem guardedTemporalApplies_agrees
    (clause : ResolvedGuardedTemporalClause)
    (input : PropertyPredicateInput) :
    guardedTemporalAppliesEvaluate clause input = true ↔
      guardedTemporalAppliesDenote clause input := by
  cases caseGuard : clause.caseGuard <;>
    simp [guardedTemporalAppliesEvaluate, guardedTemporalAppliesDenote,
      caseGuard, evaluateCheckedPredicate_agrees, exceptionAllows_agrees, and_assoc]

private def triggerPositionsInStep
    (clause : ResolvedGuardedTemporalClause)
    (transitionPosition observationOffset : Nat)
    (step : PropertyTraceStep) : Option (List Nat) :=
  collectPositions ((stepOccurrences clause.trigger transitionPosition observationOffset step).map
    (positionOf clause.limit.unit))

private def responseWithinEvaluate
    (forbidden : Bool)
    (first : Nat)
    (responsePositions : List Nat)
    (limit : Nat) : Bool :=
  let found := responsePositions.any fun second =>
    first ≤ second && second - first ≤ limit
  if forbidden then !found else found

private def responseWithinDenote
    (forbidden : Bool)
    (first : Nat)
    (responsePositions : List Nat)
    (limit : Nat) : Prop :=
  let found := anyHolds responsePositions fun second =>
    first ≤ second ∧ second - first ≤ limit
  if forbidden then ¬found else found

private theorem responseWithin_agrees
    (forbidden : Bool)
    (first : Nat)
    (responsePositions : List Nat)
    (limit : Nat) :
    responseWithinEvaluate forbidden first responsePositions limit = true ↔
      responseWithinDenote forbidden first responsePositions limit := by
  have foundAgreement :
      responsePositions.any (fun second =>
        first ≤ second && second - first ≤ limit) = true ↔
      anyHolds responsePositions (fun second =>
        first ≤ second ∧ second - first ≤ limit) :=
    anyHolds_agrees _ _ _ fun second => by simp
  cases forbidden with
  | false => exact foundAgreement
  | true =>
      exact booleanNot_agrees _ _ foundAgreement

private def evaluateGuardedTemporalSteps
    (forbidden : Bool)
    (clause : ResolvedGuardedTemporalClause)
    (responsePositions : List Nat) :
    Nat → Nat → List PropertyTraceStep → Bool
  | _, _, [] => true
  | transitionPosition, observationOffset, step :: rest =>
      match triggerPositionsInStep clause transitionPosition observationOffset step with
      | none => false
      | some triggerPositions =>
          let applies := !triggerPositions.isEmpty &&
            guardedTemporalAppliesEvaluate clause (guardInput step)
          (!applies || triggerPositions.all fun first =>
            responseWithinEvaluate forbidden first responsePositions clause.limit.value) &&
          evaluateGuardedTemporalSteps forbidden clause responsePositions
            (transitionPosition + 1) (observationOffset + step.observations.length) rest

private def guardedTemporalStepsDenote
    (forbidden : Bool)
    (clause : ResolvedGuardedTemporalClause)
    (responsePositions : List Nat) :
    Nat → Nat → List PropertyTraceStep → Prop
  | _, _, [] => True
  | transitionPosition, observationOffset, step :: rest =>
      match triggerPositionsInStep clause transitionPosition observationOffset step with
      | none => False
      | some triggerPositions =>
          ((triggerPositions ≠ [] ∧ guardedTemporalAppliesDenote clause (guardInput step)) →
            allHolds triggerPositions fun first =>
              responseWithinDenote forbidden first responsePositions clause.limit.value) ∧
          guardedTemporalStepsDenote forbidden clause responsePositions
            (transitionPosition + 1) (observationOffset + step.observations.length) rest

private theorem evaluateGuardedTemporalSteps_agrees
    (forbidden : Bool)
    (clause : ResolvedGuardedTemporalClause)
    (responsePositions : List Nat)
    (transitionPosition observationOffset : Nat)
    (steps : List PropertyTraceStep) :
    evaluateGuardedTemporalSteps forbidden clause responsePositions
        transitionPosition observationOffset steps = true ↔
      guardedTemporalStepsDenote forbidden clause responsePositions
        transitionPosition observationOffset steps := by
  induction steps generalizing transitionPosition observationOffset with
  | nil => simp [evaluateGuardedTemporalSteps, guardedTemporalStepsDenote]
  | cons step rest inductionHypothesis =>
      cases triggerResult : triggerPositionsInStep clause transitionPosition observationOffset step with
      | none =>
          simp [evaluateGuardedTemporalSteps, guardedTemporalStepsDenote, triggerResult]
      | some triggerPositions =>
          have appliesAgreement :
              (!triggerPositions.isEmpty &&
                guardedTemporalAppliesEvaluate clause (guardInput step)) = true ↔
              triggerPositions ≠ [] ∧
                guardedTemporalAppliesDenote clause (guardInput step) := by
            simp [guardedTemporalApplies_agrees]
          have responsesAgreement :
              triggerPositions.all (fun first =>
                responseWithinEvaluate forbidden first responsePositions clause.limit.value) = true ↔
              allHolds triggerPositions (fun first =>
                responseWithinDenote forbidden first responsePositions clause.limit.value) :=
            allHolds_agrees _ _ _ fun first =>
              responseWithin_agrees forbidden first responsePositions clause.limit.value
          have implicationAgreement := booleanImplication_agrees _ _ _ _
            appliesAgreement responsesAgreement
          simp only [evaluateGuardedTemporalSteps, guardedTemporalStepsDenote, triggerResult]
          rw [Bool.and_eq_true, implicationAgreement, inductionHypothesis]

private def evaluateGuardedTemporal
    (forbidden : Bool)
    (clause : ResolvedGuardedTemporalClause)
    (view : PropertyTraceView) : Bool :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => false
  | some responsePositions =>
      evaluateGuardedTemporalSteps forbidden clause responsePositions 1 0 view.steps

private def guardedTemporalDenotes
    (forbidden : Bool)
    (clause : ResolvedGuardedTemporalClause)
    (view : PropertyTraceView) : Prop :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => False
  | some responsePositions =>
      guardedTemporalStepsDenote forbidden clause responsePositions 1 0 view.steps

private theorem evaluateGuardedTemporal_agrees
    (forbidden : Bool)
    (clause : ResolvedGuardedTemporalClause)
    (view : PropertyTraceView) :
    evaluateGuardedTemporal forbidden clause view = true ↔
      guardedTemporalDenotes forbidden clause view := by
  cases responseResult : checkedPositions clause.response clause.limit.unit view with
  | none => simp [evaluateGuardedTemporal, guardedTemporalDenotes, responseResult]
  | some responsePositions =>
      simpa [evaluateGuardedTemporal, guardedTemporalDenotes, responseResult] using
        evaluateGuardedTemporalSteps_agrees forbidden clause responsePositions 1 0 view.steps

private def parentAppliesEvaluate
    (group : ResolvedPropertyCaseGroup)
    (input : PropertyPredicateInput) : Bool :=
  evaluateCheckedPredicate group.guard input && exceptionAllowsEvaluate group.exception input

private def parentAppliesDenote
    (group : ResolvedPropertyCaseGroup)
    (input : PropertyPredicateInput) : Prop :=
  checkedPredicateDenotes group.guard input ∧ exceptionAllowsDenote group.exception input

private theorem parentApplies_agrees
    (group : ResolvedPropertyCaseGroup)
    (input : PropertyPredicateInput) :
    parentAppliesEvaluate group input = true ↔ parentAppliesDenote group input := by
  simp [parentAppliesEvaluate, parentAppliesDenote,
    evaluateCheckedPredicate_agrees, exceptionAllows_agrees]

private def caseAppliesEvaluate
    (parentApplies : Bool)
    (item : ResolvedPropertyCase)
    (input : PropertyPredicateInput) : Bool :=
  parentApplies && evaluateCheckedPredicate item.guard input &&
    exceptionAllowsEvaluate item.exception input

private def caseAppliesDenote
    (parentApplies : Prop)
    (item : ResolvedPropertyCase)
    (input : PropertyPredicateInput) : Prop :=
  (parentApplies ∧ checkedPredicateDenotes item.guard input) ∧
    exceptionAllowsDenote item.exception input

private theorem caseApplies_agrees
    (parentEvaluate : Bool)
    (parentDenote : Prop)
    (parentAgreement : parentEvaluate = true ↔ parentDenote)
    (item : ResolvedPropertyCase)
    (input : PropertyPredicateInput) :
    caseAppliesEvaluate parentEvaluate item input = true ↔
      caseAppliesDenote parentDenote item input := by
  simp [caseAppliesEvaluate, caseAppliesDenote, parentAgreement,
    evaluateCheckedPredicate_agrees, exceptionAllows_agrees]

private def exclusiveCasesEvaluate
    (parentApplies : Bool)
    (input : PropertyPredicateInput) : List ResolvedPropertyCase → Bool
  | [] => true
  | item :: rest =>
      (!caseAppliesEvaluate parentApplies item input ||
        rest.all fun other => !caseAppliesEvaluate parentApplies other input) &&
      exclusiveCasesEvaluate parentApplies input rest

private def exclusiveCasesDenote
    (parentApplies : Prop)
    (input : PropertyPredicateInput) : List ResolvedPropertyCase → Prop
  | [] => True
  | item :: rest =>
      (caseAppliesDenote parentApplies item input →
        allHolds rest fun other => ¬caseAppliesDenote parentApplies other input) ∧
      exclusiveCasesDenote parentApplies input rest

private theorem exclusiveCases_agrees
    (parentEvaluate : Bool)
    (parentDenote : Prop)
    (parentAgreement : parentEvaluate = true ↔ parentDenote)
    (input : PropertyPredicateInput)
    (items : List ResolvedPropertyCase) :
    exclusiveCasesEvaluate parentEvaluate input items = true ↔
      exclusiveCasesDenote parentDenote input items := by
  induction items with
  | nil => simp [exclusiveCasesEvaluate, exclusiveCasesDenote]
  | cons item rest inductionHypothesis =>
      have itemAgreement := caseApplies_agrees parentEvaluate parentDenote
        parentAgreement item input
      have restAgreement :
          rest.all (fun other => !caseAppliesEvaluate parentEvaluate other input) = true ↔
            allHolds rest (fun other => ¬caseAppliesDenote parentDenote other input) :=
        allHolds_agrees _ _ _ fun other =>
          booleanNot_agrees _ _
            (caseApplies_agrees parentEvaluate parentDenote parentAgreement other input)
      have implicationAgreement := booleanImplication_agrees
        (caseAppliesEvaluate parentEvaluate item input)
        (rest.all fun other => !caseAppliesEvaluate parentEvaluate other input)
        (caseAppliesDenote parentDenote item input)
        (allHolds rest fun other => ¬caseAppliesDenote parentDenote other input)
        itemAgreement restAgreement
      simp [exclusiveCasesEvaluate, exclusiveCasesDenote,
        implicationAgreement, inductionHypothesis]

private def caseClausesEvaluate
    (applies : Bool)
    (input : PropertyPredicateInput)
    (item : ResolvedPropertyCase) : Bool :=
  item.clauses.all fun clause =>
    !applies || evaluateCheckedPredicate clause.expectation input

private def caseClausesDenote
    (applies : Prop)
    (input : PropertyPredicateInput)
    (item : ResolvedPropertyCase) : Prop :=
  allHolds item.clauses fun clause =>
    applies → checkedPredicateDenotes clause.expectation input

private theorem caseClauses_agrees
    (appliesEvaluate : Bool)
    (appliesDenote : Prop)
    (appliesAgreement : appliesEvaluate = true ↔ appliesDenote)
    (input : PropertyPredicateInput)
    (item : ResolvedPropertyCase) :
    caseClausesEvaluate appliesEvaluate input item = true ↔
      caseClausesDenote appliesDenote input item :=
  allHolds_agrees _ _ _ fun clause =>
    booleanImplication_agrees _ _ _ _ appliesAgreement
      (evaluateCheckedPredicate_agrees clause.expectation input)

private def evaluateCaseGroupStep
    (group : ResolvedPropertyCaseGroup)
    (step : PropertyTraceStep) : Bool :=
  let guardValues := guardInput step
  let expectationValues := expectationInput step
  let parent := parentAppliesEvaluate group guardValues
  let complete := !parent || group.cases.any fun item =>
    caseAppliesEvaluate parent item guardValues
  let exclusive := exclusiveCasesEvaluate parent guardValues group.cases
  (!group.complete || complete) && (!group.exclusive || exclusive) &&
    group.cases.all fun item =>
      caseClausesEvaluate (caseAppliesEvaluate parent item guardValues) expectationValues item

private def caseGroupStepDenotes
    (group : ResolvedPropertyCaseGroup)
    (step : PropertyTraceStep) : Prop :=
  let guardValues := guardInput step
  let expectationValues := expectationInput step
  let parent := parentAppliesDenote group guardValues
  ((¬group.complete = true ∨
      (parent → anyHolds group.cases fun item => caseAppliesDenote parent item guardValues)) ∧
    (¬group.exclusive = true ∨ exclusiveCasesDenote parent guardValues group.cases)) ∧
  allHolds group.cases fun item =>
    caseClausesDenote (caseAppliesDenote parent item guardValues) expectationValues item

private theorem evaluateCaseGroupStep_agrees
    (group : ResolvedPropertyCaseGroup)
    (step : PropertyTraceStep) :
    evaluateCaseGroupStep group step = true ↔ caseGroupStepDenotes group step := by
  let guardValues := guardInput step
  let expectationValues := expectationInput step
  let parentEvaluate := parentAppliesEvaluate group guardValues
  let parentDenote := parentAppliesDenote group guardValues
  have parentAgreement : parentEvaluate = true ↔ parentDenote :=
    parentApplies_agrees group guardValues
  have completeAgreement :
      (!parentEvaluate || group.cases.any fun item =>
        caseAppliesEvaluate parentEvaluate item guardValues) = true ↔
      (parentDenote → anyHolds group.cases fun item =>
        caseAppliesDenote parentDenote item guardValues) :=
    booleanImplication_agrees _ _ _ _ parentAgreement
      (anyHolds_agrees _ _ _ fun item =>
        caseApplies_agrees parentEvaluate parentDenote parentAgreement item guardValues)
  have clausesAgreement :
      group.cases.all (fun item =>
        caseClausesEvaluate (caseAppliesEvaluate parentEvaluate item guardValues)
          expectationValues item) = true ↔
      allHolds group.cases (fun item =>
        caseClausesDenote (caseAppliesDenote parentDenote item guardValues)
          expectationValues item) :=
    allHolds_agrees _ _ _ fun item =>
      caseClauses_agrees _ _
        (caseApplies_agrees parentEvaluate parentDenote parentAgreement item guardValues)
        expectationValues item
  have exclusiveAgreement := exclusiveCases_agrees parentEvaluate parentDenote
    parentAgreement guardValues group.cases
  cases completeFlag : group.complete <;> cases exclusiveFlag : group.exclusive <;>
    simp [evaluateCaseGroupStep, caseGroupStepDenotes, guardValues, expectationValues,
      parentEvaluate, parentDenote, completeFlag, exclusiveFlag, completeAgreement,
      exclusiveAgreement, clausesAgreement]

private def evaluateCaseGroup
    (group : ResolvedPropertyCaseGroup)
    (view : PropertyTraceView) : Bool :=
  view.steps.all (evaluateCaseGroupStep group) &&
    group.cases.all fun item =>
      item.temporalClauses.all fun clause =>
        evaluateGuardedTemporal clause.forbidden clause view

private def caseGroupDenotes
    (group : ResolvedPropertyCaseGroup)
    (view : PropertyTraceView) : Prop :=
  allHolds view.steps (caseGroupStepDenotes group) ∧
    allHolds group.cases fun item =>
      allHolds item.temporalClauses fun clause =>
        guardedTemporalDenotes clause.forbidden clause view

private theorem evaluateCaseGroup_agrees
    (group : ResolvedPropertyCaseGroup)
    (view : PropertyTraceView) :
    evaluateCaseGroup group view = true ↔ caseGroupDenotes group view :=
by
  have stepsAgreement :
      view.steps.all (evaluateCaseGroupStep group) = true ↔
        allHolds view.steps (caseGroupStepDenotes group) :=
    allHolds_agrees _ _ _ (evaluateCaseGroupStep_agrees group)
  have temporalAgreement :
      group.cases.all (fun item => item.temporalClauses.all fun clause =>
        evaluateGuardedTemporal clause.forbidden clause view) = true ↔
      allHolds group.cases (fun item => allHolds item.temporalClauses fun clause =>
        guardedTemporalDenotes clause.forbidden clause view) :=
    allHolds_agrees _ _ _ fun item =>
      allHolds_agrees _ _ _ fun clause =>
        evaluateGuardedTemporal_agrees clause.forbidden clause view
  simp [evaluateCaseGroup, caseGroupDenotes, stepsAgreement, temporalAgreement]

private def resolvedPropertyClauseDenotes
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) : Prop :=
  match clause with
  | .stateInvariant _ state => stateInvariantDenotes state view
  | .transitionContract _ precondition postcondition =>
      transitionContractDenotes precondition postcondition view
  | .identityRelation _ relation => identityRelationDenotes relation view
  | .inputOutput _ input output => transitionContractDenotes input output view
  | .ordered _ before after unit => orderedDenotes before after unit view
  | .eventuallyWithin _ trigger response limit =>
      eventuallyWithinDenotes trigger response limit view
  | .quiescentWithin _ trigger forbidden limit =>
      quiescentWithinDenotes trigger forbidden limit view
  | .sameStepCases group => caseGroupDenotes group view
  | .guardedEventuallyWithin guarded =>
      guardedTemporalDenotes false guarded view
  | .guardedQuiescentWithin guarded =>
      guardedTemporalDenotes true guarded view

private def evaluateResolvedPropertyClause
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) : Bool :=
  match clause with
  | .stateInvariant _ state => evaluateStateInvariant state view
  | .transitionContract _ precondition postcondition =>
      evaluateTransitionContract precondition postcondition view
  | .identityRelation _ relation => evaluateIdentityRelation relation view
  | .inputOutput _ input output => evaluateTransitionContract input output view
  | .ordered _ before after unit => evaluateOrdered before after unit view
  | .eventuallyWithin _ trigger response limit =>
      evaluateEventuallyWithin trigger response limit view
  | .quiescentWithin _ trigger forbidden limit =>
      evaluateQuiescentWithin trigger forbidden limit view
  | .sameStepCases group => evaluateCaseGroup group view
  | .guardedEventuallyWithin guarded =>
      evaluateGuardedTemporal false guarded view
  | .guardedQuiescentWithin guarded =>
      evaluateGuardedTemporal true guarded view

/-- Denotation of one clause proven to belong to the exact checked Property and input. -/
def CheckedProperty.denotesClause
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property)
    (clause : { clause // clause ∈ property.clauses }) : Prop :=
  resolvedPropertyClauseDenotes clause.1 input.view

/-- Evaluate one clause only through its exact checked Property input. -/
def evaluatePropertyClause
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property)
    (clause : { clause // clause ∈ property.clauses }) : Bool :=
  evaluateResolvedPropertyClause clause.1 input.view

/-- Structural agreement for every constructor in the portable property core. -/
theorem evaluatePropertyClause_agrees
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property)
    (clause : { clause // clause ∈ property.clauses }) :
    evaluatePropertyClause property input clause = true ↔
      property.denotesClause input clause := by
  change evaluateResolvedPropertyClause clause.1 input.view = true ↔
    resolvedPropertyClauseDenotes clause.1 input.view
  cases clause.1 with
  | stateInvariant _ state =>
      exact evaluateStateInvariant_agrees state input.view
  | transitionContract _ precondition postcondition =>
      exact evaluateTransitionContract_agrees precondition postcondition input.view
  | identityRelation _ relation =>
      exact evaluateIdentityRelation_agrees relation input.view
  | inputOutput _ inputPattern output =>
      exact evaluateTransitionContract_agrees inputPattern output input.view
  | ordered _ before after unit =>
      exact evaluateOrdered_agrees before after unit input.view
  | eventuallyWithin _ trigger response limit =>
      exact evaluateEventuallyWithin_agrees trigger response limit input.view
  | quiescentWithin _ trigger forbidden limit =>
      exact evaluateQuiescentWithin_agrees trigger forbidden limit input.view
  | sameStepCases group =>
      exact evaluateCaseGroup_agrees group input.view
  | guardedEventuallyWithin guarded =>
      exact evaluateGuardedTemporal_agrees false guarded input.view
  | guardedQuiescentWithin guarded =>
      exact evaluateGuardedTemporal_agrees true guarded input.view

structure PropertyTraceSpan where
  firstTransition : Nat
  lastTransition : Nat
  deriving BEq, DecidableEq, Ord, Repr

inductive PropertyObligationKind where
  | clause
  | complete
  | exclusive
  deriving BEq, DecidableEq, Ord, Repr

/-- Canonical identity of one same-step obligation; case identity is part of every case clause. -/
structure PropertyClauseIdentity where
  parentId : DefinitionId
  caseId : Option DefinitionId
  clauseId : DefinitionId
  kind : PropertyObligationKind
  source : SourceLocation
  evaluatedLimit : Option Limit := none
  relatedDefinitionIds : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

/-- Effective applicability of one checked case at one reachable same-step context. -/
structure CaseApplicability where
  caseId : DefinitionId
  source : SourceLocation
  effectiveGuards : List (CheckedPropertyPredicate .guard)
  exceptions : List ResolvedPropertyException
  guardMatched : Bool
  excluded : Bool
  applies : Bool
  clauses : List PropertyClauseIdentity
  deriving BEq, DecidableEq, Repr

/-- Evaluator-owned parent and case applicability at one validated trigger step. -/
structure CaseGroupApplicability where
  propertyId : DefinitionId
  parentId : DefinitionId
  source : SourceLocation
  transitionPosition : Nat
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  effectiveGuards : List (CheckedPropertyPredicate .guard)
  exceptions : List ResolvedPropertyException
  parentGuardMatched : Bool
  parentExcluded : Bool
  parentApplies : Bool
  complete : Bool
  exclusive : Bool
  cases : List CaseApplicability
  deriving BEq, DecidableEq, Repr

/-- The checked formula carried by one jointly analyzed obligation. Temporal applicability remains
fixed at its original trigger, together with the coordinate system used by its declared Limit. -/
inductive JointObligationFormula where
  | sameStep (expectation : CheckedPropertyPredicate .expectation)
  | guardedTemporal
      (forbidden : Bool)
      (trigger response : PropertyPattern)
      (limit : Limit)
  deriving BEq, DecidableEq, Repr

/-- The concrete occurrence and coordinate system that selected an obligation. -/
structure JointTriggerOccurrence where
  field : PropertyTraceField
  value : Option ModelValue
  coordinateUnit : LimitUnit
  coordinate : Nat
  deriving BEq, DecidableEq, Repr

/-- Evaluator-owned observation of one applicable obligation at one exact trigger. -/
structure JointObligationObservation where
  propertyId : DefinitionId
  parentId : DefinitionId
  caseId : Option DefinitionId
  clauseId : DefinitionId
  source : SourceLocation
  transitionPosition : Nat
  triggerCoordinate : Nat
  triggerOccurrence : JointTriggerOccurrence
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  effectiveGuards : List (CheckedPropertyPredicate .guard)
  exceptions : List ResolvedPropertyException
  formula : JointObligationFormula
  satisfied : Bool
  deriving BEq, DecidableEq, Repr

private def sameStepClauseIdentity
    (group : ResolvedPropertyCaseGroup)
    (item : ResolvedPropertyCase)
    (clause : ResolvedPropertySameStepClause) : PropertyClauseIdentity := {
  parentId := group.id
  caseId := some item.id
  clauseId := clause.id
  kind := .clause
  source := clause.source
  relatedDefinitionIds := item.exception.toList.map ResolvedPropertyException.id
}

private def temporalClauseIdentity
    (clause : ResolvedGuardedTemporalClause) : PropertyClauseIdentity := {
  parentId := clause.parentId
  caseId := clause.caseId
  clauseId := clause.id
  kind := .clause
  source := clause.source
  evaluatedLimit := some clause.limit
  relatedDefinitionIds :=
    clause.exception.toList.map ResolvedPropertyException.id ++
      clause.caseException.toList.map ResolvedPropertyException.id
}

/-- Exact same-step and temporal obligation identities declared by one checked named case. -/
def ResolvedPropertyCase.clauseIdentities
    (group : ResolvedPropertyCaseGroup)
    (item : ResolvedPropertyCase) : List PropertyClauseIdentity :=
  item.clauses.map (sameStepClauseIdentity group item) ++
    item.temporalClauses.map temporalClauseIdentity

private def caseApplicabilityAt
    (group : ResolvedPropertyCaseGroup)
    (parentApplies : Bool)
    (input : PropertyPredicateInput)
    (item : ResolvedPropertyCase) : CaseApplicability :=
  let guardMatched := evaluateCheckedPredicate item.guard input
  let exceptionAllows := exceptionAllowsEvaluate item.exception input
  {
    caseId := item.id
    source := item.source
    effectiveGuards := [group.guard, item.guard]
    exceptions := group.exception.toList ++ item.exception.toList
    guardMatched
    excluded := parentApplies && guardMatched && !exceptionAllows
    applies := parentApplies && guardMatched && exceptionAllows
    clauses := item.clauseIdentities group
  }

private def caseGroupApplicabilityAt
    (property : CheckedProperty)
    (group : ResolvedPropertyCaseGroup)
    (transitionPosition : Nat)
    (step : PropertyTraceStep) : CaseGroupApplicability :=
  let input := guardInput step
  let guardMatched := evaluateCheckedPredicate group.guard input
  let exceptionAllows := exceptionAllowsEvaluate group.exception input
  let applies := guardMatched && exceptionAllows
  {
    propertyId := property.id
    parentId := group.id
    source := group.source
    transitionPosition
    priorState := step.priorState
    selectedAction := step.selectedAction
    effectiveGuards := [group.guard]
    exceptions := group.exception.toList
    parentGuardMatched := guardMatched
    parentExcluded := guardMatched && !exceptionAllows
    parentApplies := applies
    complete := group.complete
    exclusive := group.exclusive
    cases := group.cases.map (caseApplicabilityAt group applies input)
  }

/-- Report case applicability only from the exact input already admitted by Property evaluation.
This is the shared checked seam for coverage analysis; it does not reinterpret raw predicates. -/
def analyzeCaseApplicability
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property) : List CaseGroupApplicability :=
  property.clauses.flatMap fun clause => match clause with
    | .sameStepCases group => input.view.steps.zipIdx.map fun (step, index) =>
        caseGroupApplicabilityAt property group (index + 1) step
    | _ => []

private def sameStepJointObservationsAt
    (property : CheckedProperty)
    (group : ResolvedPropertyCaseGroup)
    (transitionPosition : Nat)
    (step : PropertyTraceStep) : List JointObligationObservation :=
  let applicability := caseGroupApplicabilityAt property group transitionPosition step
  let expectationValues := expectationInput step
  group.cases.flatMap fun item =>
    match applicability.cases.find? fun candidate => candidate.caseId == item.id with
    | some candidate => if candidate.applies then
        item.clauses.map fun clause => {
          propertyId := property.id
          parentId := group.id
          caseId := some item.id
          clauseId := clause.id
          source := clause.source
          transitionPosition
          triggerCoordinate := transitionPosition
          triggerOccurrence := {
            field := .selectedAction
            value := step.selectedAction
            coordinateUnit := .semanticTransitions
            coordinate := transitionPosition
          }
          priorState := step.priorState
          selectedAction := step.selectedAction
          effectiveGuards := candidate.effectiveGuards
          exceptions := candidate.exceptions
          formula := .sameStep clause.expectation
          satisfied := evaluateCheckedPredicate clause.expectation expectationValues
        }
      else []
    | none => []

private def guardedTemporalJointObservations
    (property : CheckedProperty)
    (clause : ResolvedGuardedTemporalClause)
    (view : PropertyTraceView) : List JointObligationObservation :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => []
  | some responsePositions =>
      let rec visit
          (transitionPosition observationOffset : Nat)
          (steps : List PropertyTraceStep) : List JointObligationObservation :=
        match steps with
        | [] => []
        | step :: rest =>
            let tail := visit (transitionPosition + 1)
              (observationOffset + step.observations.length) rest
            let triggerOccurrences :=
              stepOccurrences clause.trigger transitionPosition observationOffset step
            match collectPositions (triggerOccurrences.map (positionOf clause.limit.unit)) with
            | none => tail
            | some triggerPositions =>
                if guardedTemporalAppliesEvaluate clause (guardInput step) then
                  (triggerOccurrences.zip triggerPositions |>.map fun (occurrence, triggerCoordinate) => ({
                    propertyId := property.id
                    parentId := clause.parentId
                    caseId := clause.caseId
                    clauseId := clause.id
                    source := clause.source
                    transitionPosition
                    triggerCoordinate
                    triggerOccurrence := {
                      field := clause.trigger.field
                      value := some occurrence.value
                      coordinateUnit := clause.limit.unit
                      coordinate := triggerCoordinate
                    }
                    priorState := step.priorState
                    selectedAction := step.selectedAction
                    effectiveGuards := clause.guard :: clause.caseGuard.toList
                    exceptions := clause.exception.toList ++ clause.caseException.toList
                    formula := .guardedTemporal clause.forbidden clause.trigger clause.response clause.limit
                    satisfied := responseWithinEvaluate clause.forbidden triggerCoordinate
                      responsePositions clause.limit.value
                  } : JointObligationObservation)) ++ tail
                else
                  tail
      visit 1 0 view.steps

/-- Observe jointly analyzable obligations only through the same checked input and evaluator used
for ordinary Property truth. Later trace steps cannot change trigger-time guards or exceptions. -/
def analyzeJointObligations
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property) : List JointObligationObservation :=
  property.clauses.flatMap fun clause => match clause with
    | .sameStepCases group =>
        (input.view.steps.zipIdx.flatMap fun (step, index) =>
          sameStepJointObservationsAt property group (index + 1) step) ++
        (group.cases.flatMap fun item =>
          item.temporalClauses.flatMap fun temporal =>
            guardedTemporalJointObservations property temporal input.view)
    | .guardedEventuallyWithin temporal | .guardedQuiescentWithin temporal =>
        guardedTemporalJointObservations property temporal input.view
    | _ => []

structure PropertyClauseResult where
  propertyId : DefinitionId
  clauseId : DefinitionId
  satisfied : Bool
  traceSpan : Option PropertyTraceSpan
  evaluatedLimit : Option Limit
  semanticProvenance : List DefinitionId
  failedObligations : List PropertyClauseIdentity := []
  deriving BEq, DecidableEq, Repr

structure PropertyEvaluation where
  propertyId : DefinitionId
  satisfied : Bool
  clauses : List PropertyClauseResult
  deriving BEq, DecidableEq, Repr

private def clausePatterns : ResolvedPropertyClause → List PropertyPattern
  | .stateInvariant _ state => [state]
  | .transitionContract _ precondition postcondition => [precondition, postcondition]
  | .identityRelation _ relation => [relation]
  | .inputOutput _ input output => [input, output]
  | .ordered _ before after _ => [before, after]
  | .eventuallyWithin _ trigger response _ => [trigger, response]
  | .quiescentWithin _ trigger forbidden _ => [trigger, forbidden]
  | .sameStepCases group => group.cases.flatMap fun item =>
      item.temporalClauses.flatMap fun clause => [clause.trigger, clause.response]
  | .guardedEventuallyWithin guarded | .guardedQuiescentWithin guarded =>
      [guarded.trigger, guarded.response]

private def clauseLimit : ResolvedPropertyClause → Option Limit
  | .eventuallyWithin _ _ _ limit | .quiescentWithin _ _ _ limit => some limit
  | .guardedEventuallyWithin guarded | .guardedQuiescentWithin guarded =>
      some guarded.limit
  | _ => none

private def predicateReferences : PropertyPredicate → List DefinitionId
  | .atom atom => [atom.reference]
  | .all items | .any items => items.flatMap predicateReferences
  | .not item => predicateReferences item

private def resolvedPredicateReferences
    (predicate : CheckedPropertyPredicate context) : List DefinitionId :=
  predicateReferences predicate.expression

private def caseGroupProvenance (group : ResolvedPropertyCaseGroup) : List DefinitionId :=
  [group.id] ++ resolvedPredicateReferences group.guard ++
    group.exception.toList.flatMap fun exception =>
      exception.id :: resolvedPredicateReferences exception.condition ++
    group.cases.flatMap fun item =>
      item.id :: resolvedPredicateReferences item.guard ++
        item.exception.toList.flatMap (fun exception =>
          exception.id :: resolvedPredicateReferences exception.condition) ++
        (item.clauses.flatMap fun clause =>
          clause.id :: resolvedPredicateReferences clause.expectation) ++
        (item.temporalClauses.flatMap fun clause =>
          [clause.id, clause.trigger.reference, clause.response.reference])

private def guardedTemporalProvenance
    (clause : ResolvedGuardedTemporalClause) : List DefinitionId :=
  clause.id :: resolvedPredicateReferences clause.guard ++
    clause.exception.toList.flatMap fun exception =>
      exception.id :: resolvedPredicateReferences exception.condition

private def guardedTemporalFailures
    (clause : ResolvedGuardedTemporalClause)
    (satisfied : Bool) : List PropertyClauseIdentity :=
  if satisfied then [] else [{
    parentId := clause.parentId
    caseId := clause.caseId
    clauseId := clause.id
    kind := .clause
    source := clause.source
    evaluatedLimit := some clause.limit
    relatedDefinitionIds :=
      clause.exception.toList.map ResolvedPropertyException.id ++
        clause.caseException.toList.map ResolvedPropertyException.id
  }]

private def caseGroupFailures
    (group : ResolvedPropertyCaseGroup)
    (view : PropertyTraceView) : List PropertyClauseIdentity :=
  let stepFailures := view.steps.flatMap fun step =>
    let guardValues := guardInput step
    let expectationValues := expectationInput step
    let parent := parentAppliesEvaluate group guardValues
    let applicable := group.cases.filter fun item =>
      caseAppliesEvaluate parent item guardValues
    let completeFailure := if group.complete && parent && applicable.isEmpty then [{
      parentId := group.id
      caseId := none
      clauseId := group.id
      kind := .complete
      source := group.source
    }] else []
    let exclusiveFailure := if group.exclusive && applicable.length > 1 then [{
      parentId := group.id
      caseId := none
      clauseId := group.id
      kind := .exclusive
      source := group.source
      relatedDefinitionIds := applicable.map ResolvedPropertyCase.id
    }] else []
    let clauseFailures := applicable.flatMap fun item =>
      item.clauses.filterMap fun clause =>
        if evaluateCheckedPredicate clause.expectation expectationValues then
          none
        else
          some {
            parentId := group.id
            caseId := some item.id
            clauseId := clause.id
            kind := .clause
            source := clause.source
            relatedDefinitionIds := item.exception.toList.map ResolvedPropertyException.id
          }
    completeFailure ++ exclusiveFailure ++ clauseFailures
  let temporalFailures := group.cases.flatMap fun item =>
    item.temporalClauses.flatMap fun clause =>
      guardedTemporalFailures clause (evaluateGuardedTemporal clause.forbidden clause view)
  (stepFailures ++ temporalFailures).eraseDups

private def spanOf
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) : Option PropertyTraceSpan :=
  let found := (clausePatterns clause).flatMap fun pattern =>
    (occurrences pattern view).map PropertyOccurrence.transitionPosition
  match found with
  | [] => none
  | first :: rest => some {
      firstTransition := rest.foldl Nat.min first
      lastTransition := rest.foldl Nat.max first
    }

private def resultOf
    (property : CheckedProperty)
    (view : PropertyTraceView)
    (clause : ResolvedPropertyClause) : PropertyClauseResult :=
  let satisfied := evaluateResolvedPropertyClause clause view
  {
  propertyId := property.id
  clauseId := clause.id
  satisfied
  traceSpan := spanOf clause view
  evaluatedLimit := clauseLimit clause
  semanticProvenance := DefinitionId.canonicalSet
    (property.requires ++ (clausePatterns clause).map PropertyPattern.reference ++ match clause with
      | .sameStepCases group => caseGroupProvenance group
      | .guardedEventuallyWithin guarded | .guardedQuiescentWithin guarded =>
          guardedTemporalProvenance guarded
      | _ => [])
  failedObligations := match clause with
    | .sameStepCases group => caseGroupFailures group view
    | .guardedEventuallyWithin guarded | .guardedQuiescentWithin guarded =>
        guardedTemporalFailures guarded satisfied
    | _ => []
  }

/-- Evaluate through the checked gate: the unrestricted trace is reduced to the admitted view
before any clause interpreter runs. -/
def evaluateProperty
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property) :
    PropertyEvaluation :=
  let view := input.view
  let clauses := property.clauses.map (resultOf property view)
  {
    propertyId := property.id
    satisfied := clauses.all PropertyClauseResult.satisfied
    clauses
  }

/-- Three-valued evaluation of a selected prefix; unresolved never supplies a counterexample. -/
inductive PropertyEndpointAnswer where
  | satisfied
  | violated
  | unresolved
  deriving BEq, DecidableEq, Repr

/-- Realized trigger identity is retained independently of whether its obligation succeeds. -/
structure PropertyTriggerEvidence where
  propertyId : DefinitionId
  clauseId : DefinitionId
  transitionPosition : Nat
  occurrence : JointTriggerOccurrence
  deriving BEq, DecidableEq, Repr

/-- Checked endpoint observations used by Query without reinterpreting Property predicates. -/
structure PropertyEndpointEvaluation where
  answer : PropertyEndpointAnswer
  requestedTriggers : List DefinitionId
  realizedTriggers : List PropertyTriggerEvidence
  deriving BEq, DecidableEq, Repr

private def combineEndpointAnswers (answers : List PropertyEndpointAnswer) : PropertyEndpointAnswer :=
  if answers.contains .violated then .violated
  else if answers.contains .unresolved then .unresolved
  else .satisfied

private def endpointCoordinate (unit : LimitUnit) (view : PropertyTraceView) : Option Nat :=
  match unit with
  | .semanticTransitions | .selectedActions => some view.steps.length
  | .observationPositions => some (view.steps.foldl (fun n step => n + step.observations.length) 0)
  | .logicalTime => view.steps.getLast?.bind (·.logicalTime)
  | _ => none

private def temporalEndpointAnswer
    (forbidden : Bool) (trigger : Nat) (responses : List Nat) (limit : Limit)
    (view : PropertyTraceView) : PropertyEndpointAnswer :=
  let found := responses.any fun response => trigger ≤ response && response - trigger ≤ limit.value
  if found then
    if forbidden then .violated else .satisfied
  else
    match endpointCoordinate limit.unit view with
    | some current => if current ≥ trigger + limit.value then
        if forbidden then .satisfied else .violated
      else .unresolved
    | none => .unresolved

private def plainTemporalEndpointAnswer
    (forbidden : Bool) (trigger response : PropertyPattern) (limit : Limit)
    (view : PropertyTraceView) : PropertyEndpointAnswer :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggers, some responses => combineEndpointAnswers
      (triggers.map fun coordinate => temporalEndpointAnswer forbidden coordinate responses limit view)
  | _, _ => .unresolved

private def guardedEndpointAnswer
    (clause : ResolvedGuardedTemporalClause)
    (view : PropertyTraceView) : PropertyEndpointAnswer :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => .unresolved
  | some responses =>
      let rec visit (transitionPosition observationOffset : Nat)
          (steps : List PropertyTraceStep) : List PropertyEndpointAnswer :=
        match steps with
        | [] => []
        | step :: rest =>
            let tail := visit (transitionPosition + 1)
              (observationOffset + step.observations.length) rest
            if guardedTemporalAppliesEvaluate clause (guardInput step) then
              match triggerPositionsInStep clause transitionPosition observationOffset step with
              | none => .unresolved :: tail
              | some triggers => triggers.map (fun coordinate =>
                  temporalEndpointAnswer clause.forbidden coordinate responses clause.limit view) ++ tail
            else tail
      combineEndpointAnswers (visit 1 0 view.steps)

private def clauseEndpointAnswer
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) : PropertyEndpointAnswer :=
  match clause with
  | .stateInvariant _ pattern =>
      if !((valuesAtField .state view).any fun value => value.definitionId == pattern.reference) then
        .unresolved
      else if evaluateStateInvariant pattern view then .satisfied else .violated
  | .eventuallyWithin _ trigger response limit =>
      plainTemporalEndpointAnswer false trigger response limit view
  | .quiescentWithin _ trigger response limit =>
      plainTemporalEndpointAnswer true trigger response limit view
  | .guardedEventuallyWithin clause | .guardedQuiescentWithin clause =>
      guardedEndpointAnswer clause view
  | .sameStepCases group =>
      if !(view.steps.all (evaluateCaseGroupStep group)) then .violated else
        combineEndpointAnswers (group.cases.flatMap fun item =>
          item.temporalClauses.map fun clause => guardedEndpointAnswer clause view)
  | .ordered .. | .identityRelation .. =>
      if evaluateResolvedPropertyClause clause view then .satisfied else .unresolved
  | _ => if evaluateResolvedPropertyClause clause view then .satisfied else .violated

private def plainTriggerPattern : ResolvedPropertyClause → Option PropertyPattern
  | .transitionContract _ trigger _ | .inputOutput _ trigger _ => some trigger
  | .eventuallyWithin _ trigger _ _ | .quiescentWithin _ trigger _ _ => some trigger
  | _ => none

private def requestedClauseTriggers : ResolvedPropertyClause → List DefinitionId
  | .sameStepCases group => group.cases.flatMap fun item =>
      item.clauses.map (·.id) ++ item.temporalClauses.map (·.id)
  | .guardedEventuallyWithin clause | .guardedQuiescentWithin clause => [clause.id]
  | clause => if (plainTriggerPattern clause).isSome then [clause.id] else []

/-- Interpret the exact admitted view as closed or still open, preserving ordinary closed truth
and reporting conditional exercise separately from that truth. -/
def evaluatePropertyEndpoint
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property)
    (runtimePrefix : Bool) : PropertyEndpointEvaluation :=
  let joint := (analyzeJointObligations property input).map fun observation => ({
    propertyId := property.id
    clauseId := observation.clauseId
    transitionPosition := observation.transitionPosition
    occurrence := observation.triggerOccurrence
  } : PropertyTriggerEvidence)
  let plain := property.clauses.flatMap fun clause =>
    match plainTriggerPattern clause with
    | none => []
    | some pattern => (occurrences pattern input.view).map fun occurrence => ({
        propertyId := property.id
        clauseId := clause.id
        transitionPosition := occurrence.transitionPosition
        occurrence := {
          field := pattern.field
          value := some occurrence.value
          coordinateUnit := .semanticTransitions
          coordinate := occurrence.transitionPosition
        }
      } : PropertyTriggerEvidence)
  {
    answer := if runtimePrefix then
      combineEndpointAnswers (property.clauses.map fun clause =>
        clauseEndpointAnswer clause input.view)
      else if (evaluateProperty property input).satisfied then .satisfied else .violated
    requestedTriggers := property.clauses.flatMap requestedClauseTriggers
    realizedTriggers := joint ++ plain
  }

/-- Deliberately closed evaluation preserves the existing Boolean Property authority exactly. -/
theorem evaluatePropertyEndpoint_closed
    (property : CheckedProperty) (input : CheckedPropertyEvaluationInput property) :
    (evaluatePropertyEndpoint property input false).answer =
      (if (evaluateProperty property input).satisfied then .satisfied else .violated) := rfl

/-- Validate the exact Property input and evaluate it without discarding a same-step diagnostic. -/
def evaluatePropertyOnTrace
    (property : CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Except PropertyError PropertyEvaluation := do
  let input ← checkPropertyEvaluationInput property trace
  pure (evaluateProperty property input)

/-- Denotational meaning of every clause in one exact checked Property input. -/
def CheckedProperty.denote
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property) : Prop :=
  allHolds property.clauses fun clause => resolvedPropertyClauseDenotes clause input.view

/-- Whole-Property evaluator/denotation agreement for every admitted legacy and guarded clause. -/
theorem evaluateProperty_agrees
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property) :
    (evaluateProperty property input).satisfied = true ↔ property.denote input := by
  simpa [evaluateProperty, resultOf, CheckedProperty.denote] using
    (allHolds_agrees property.clauses
      (fun clause => evaluateResolvedPropertyClause clause input.view)
      (fun clause => resolvedPropertyClauseDenotes clause input.view) fun clause => by
    cases clause <;> simp [evaluateResolvedPropertyClause, resolvedPropertyClauseDenotes,
      evaluateStateInvariant_agrees, evaluateTransitionContract_agrees,
      evaluateIdentityRelation_agrees, evaluateOrdered_agrees,
      evaluateEventuallyWithin_agrees, evaluateQuiescentWithin_agrees,
      evaluateCaseGroup_agrees, evaluateGuardedTemporal_agrees])

end Umpire

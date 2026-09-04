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

private theorem booleanImplication_agrees
    (left right : Bool)
    (antecedent consequent : Prop)
    (leftAgreement : left = true ↔ antecedent)
    (rightAgreement : right = true ↔ consequent) :
    (!left || right) = true ↔ (antecedent → consequent) := by
  cases left <;> cases right <;> simp_all

private theorem booleanNot_agrees
    (value : Bool)
    (proposition : Prop)
    (agreement : value = true ↔ proposition) :
    (!value) = true ↔ ¬proposition := by
  cases value <;> simp_all

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

def ResolvedPropertyClause.denote
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

def evaluatePropertyClause
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

/-- Structural agreement for every constructor in the portable property core. -/
theorem evaluatePropertyClause_agrees
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) :
    evaluatePropertyClause clause view = true ↔ clause.denote view := by
  cases clause with
  | stateInvariant _ state =>
      exact evaluateStateInvariant_agrees state view
  | transitionContract _ precondition postcondition =>
      exact evaluateTransitionContract_agrees precondition postcondition view
  | identityRelation _ relation =>
      exact evaluateIdentityRelation_agrees relation view
  | inputOutput _ input output =>
      exact evaluateTransitionContract_agrees input output view
  | ordered _ before after unit =>
      exact evaluateOrdered_agrees before after unit view
  | eventuallyWithin _ trigger response limit =>
      exact evaluateEventuallyWithin_agrees trigger response limit view
  | quiescentWithin _ trigger forbidden limit =>
      exact evaluateQuiescentWithin_agrees trigger forbidden limit view

structure PropertyTraceSpan where
  firstTransition : Nat
  lastTransition : Nat
  deriving BEq, DecidableEq, Ord, Repr

structure PropertyClauseResult where
  propertyId : DefinitionId
  clauseId : DefinitionId
  satisfied : Bool
  traceSpan : Option PropertyTraceSpan
  evaluatedLimit : Option Limit
  semanticProvenance : List DefinitionId
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

private def clauseLimit : ResolvedPropertyClause → Option Limit
  | .eventuallyWithin _ _ _ limit | .quiescentWithin _ _ _ limit => some limit
  | _ => none

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
    (clause : ResolvedPropertyClause) : PropertyClauseResult := {
  propertyId := property.id
  clauseId := clause.id
  satisfied := evaluatePropertyClause clause view
  traceSpan := spanOf clause view
  evaluatedLimit := clauseLimit clause
  semanticProvenance := DefinitionId.canonicalSet
    (property.requires ++ (clausePatterns clause).map PropertyPattern.reference)
}

/-- Evaluate through the checked gate: the unrestricted trace is reduced to the admitted view
before any clause interpreter runs. -/
def evaluateProperty
    (property : CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    PropertyEvaluation :=
  let view := property.traceView trace
  let clauses := property.clauses.map (resultOf property view)
  {
    propertyId := property.id
    satisfied := clauses.all PropertyClauseResult.satisfied
    clauses
  }

end Umpire

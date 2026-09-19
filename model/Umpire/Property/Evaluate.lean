import Umpire.Property.Check

/-! Executable and denotational semantics for checked Properties over admitted trace views. -/

namespace Umpire

open Operation Value Value.Encoding

/-- A scalar obtained from a checked cursor, retaining its actual root and structural denotation.
Only the closed scalar projection is available to the Property kernel. -/
private structure PropertyFieldValue where
  path : PropertyFieldPath
  modelValue : ModelValue
  scalar : Scalar
  origin : Raw
  denotation : ∃ raw bound, Field.Denotes origin path.steps (some raw) ∧
    readLiteral bound raw = some scalar
  deriving Repr

/-- Erasure preserves exact model payload identity at the checked cursor boundary. -/
private def PropertyFieldValue.ofCursor
    (root : PropertyFieldRoot) (reference : DefinitionId)
    (cursor : Field.Cursor owner witness side limits type .singular .available)
    (source : SourceLocation) : Except Field.Error PropertyFieldValue := do
  let scalar ← cursor.scalarValue source
  let path : PropertyFieldPath := {
    root, reference, schema := owner.schema witness, side, steps := cursor.path, type }
  let modelValue := ModelValue.named reference (Canonical.key (sequence [
    textData reference.value, Canonical.rpcSchema (owner.schema witness), cursor.origin.value]))
  pure ⟨path, modelValue, scalar.value, cursor.origin.value,
    ⟨scalar.raw, limits.bytes, scalar.denotes, scalar.parsed⟩⟩

/-- Request projections require the very arguments of the selected owner-indexed Action. -/
private def PropertyFieldValue.ofAction
    {owner : RpcOwner} {Request Response Failure : Type}
    {template : ActionTemplate owner Request Response Failure} {limits : Limits}
    (action : ActionInstance template limits)
    (cursor : Field.Cursor owner template.declaration.reference .request limits type .singular .available)
    (same : cursor.origin.value = action.arguments.value) (source : SourceLocation) :
    Except Field.Error PropertyFieldValue := do
  let _ := same
  ofCursor .request template.identity cursor source

/-- Resolve only an exact checked field projection; no absent sentinel is a scalar operand. -/
private def PropertyFieldOperand.resolve (operand : PropertyFieldOperand)
    (values : List PropertyFieldValue) : Option Scalar :=
  match operand with
  | .literal value _ => some value
  | .field path _ => (values.find? (fun value => decide (value.path = path))).map (·.scalar)


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
  | .fields _ => False
  | .present => True
  | .equals literal => literal.denote payload
  | .oneOf literals => anyHolds literals fun literal => literal.denote payload

/-- Executable interpretation of a closed atomic comparison. -/
private def PropertyAtomConstraint.evaluate
    (constraint : PropertyAtomConstraint)
    (payload : String) : Bool :=
  match constraint with
  | .fields _ => false
  | .present => true
  | .equals literal => literal.evaluate payload
  | .oneOf literals => literals.any fun literal => literal.evaluate payload

/-- Atomic comparison evaluation agrees with its denotation. -/
private theorem PropertyAtomConstraint.evaluate_agrees
    (constraint : PropertyAtomConstraint)
    (payload : String) :
    constraint.evaluate payload = true ↔ constraint.denote payload := by
  cases constraint with
  | fields _ => simp [PropertyAtomConstraint.evaluate, PropertyAtomConstraint.denote]
  | present => simp [PropertyAtomConstraint.evaluate, PropertyAtomConstraint.denote]
  | equals literal =>
      exact literal.evaluate_agrees payload
  | oneOf literals =>
      exact anyHolds_agrees _ _ _ fun literal => literal.evaluate_agrees payload

/-- Propositional interpretation of an atom over one same-step field projection. -/
private def PropertyAtom.denote
    (atom : PropertyAtom)
    (values : PropertyPredicateField → List ModelValue)
    (fieldValues : List PropertyFieldValue := []) : Prop :=
  match atom.fieldComparison with
  | some comparison => match comparison.left.resolve fieldValues, comparison.right.resolve fieldValues with
    | some left, some right => comparison.operator.denotes left right
    | _, _ => False
  | none => anyHolds (values atom.field) fun value =>
      value.definitionId = atom.reference ∧ atom.constraint.denote value.value

/-- Executable interpretation of an atom over one same-step field projection. -/
private def PropertyAtom.evaluate
    (atom : PropertyAtom)
    (values : PropertyPredicateField → List ModelValue)
    (fieldValues : List PropertyFieldValue := []) : Bool :=
  match atom.fieldComparison with
  | some comparison => match comparison.left.resolve fieldValues, comparison.right.resolve fieldValues with
    | some left, some right => comparison.operator.matches left right
    | _, _ => false
  | none => (values atom.field).any fun value =>
      decide (value.definitionId = atom.reference) && atom.constraint.evaluate value.value

/-- Atomic field evaluation agrees with its denotation. -/
private theorem PropertyAtom.evaluate_agrees
    (atom : PropertyAtom)
    (values : PropertyPredicateField → List ModelValue)
    (fieldValues : List PropertyFieldValue := []) :
    atom.evaluate values fieldValues = true ↔ atom.denote values fieldValues := by
  unfold PropertyAtom.evaluate PropertyAtom.denote
  cases atom.fieldComparison with
  | none =>
    exact anyHolds_agrees _ _ _ fun value => by
      simp [PropertyAtomConstraint.evaluate_agrees]
  | some comparison =>
    cases left : comparison.left.resolve fieldValues <;>
      cases right : comparison.right.resolve fieldValues <;>
        simp [left, right, PropertyFieldOperator.matches_agrees]

mutual
  /-- Propositional interpretation of the portable Boolean predicate kernel. -/
  private def PropertyPredicate.denote
      (predicate : PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) : Prop :=
    match predicate with
    | .atom value => value.denote values fieldValues
    | .all items => propertyPredicateAllDenote items values fieldValues
    | .any items => propertyPredicateAnyDenote items values fieldValues
    | .not item => ¬item.denote values fieldValues

  private def propertyPredicateAllDenote
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) : Prop :=
    match items with
    | [] => True
    | item :: rest => item.denote values fieldValues ∧ propertyPredicateAllDenote rest values fieldValues

  private def propertyPredicateAnyDenote
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) : Prop :=
    match items with
    | [] => False
    | item :: rest => item.denote values fieldValues ∨ propertyPredicateAnyDenote rest values fieldValues
end

mutual
  /-- Executable interpretation of the portable Boolean predicate kernel. -/
  private def PropertyPredicate.evaluate
      (predicate : PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) : Bool :=
    match predicate with
    | .atom value => value.evaluate values fieldValues
    | .all items => propertyPredicateAllEvaluate items values fieldValues
    | .any items => propertyPredicateAnyEvaluate items values fieldValues
    | .not item => !(item.evaluate values fieldValues)

  private def propertyPredicateAllEvaluate
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) : Bool :=
    match items with
    | [] => true
    | item :: rest => item.evaluate values fieldValues && propertyPredicateAllEvaluate rest values fieldValues

  private def propertyPredicateAnyEvaluate
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) : Bool :=
    match items with
    | [] => false
    | item :: rest => item.evaluate values fieldValues || propertyPredicateAnyEvaluate rest values fieldValues
end

mutual
  /-- Structural evaluator/denotation agreement for every Boolean predicate constructor. -/
  private theorem PropertyPredicate.evaluate_agrees
      (predicate : PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) :
      predicate.evaluate values fieldValues = true ↔ predicate.denote values fieldValues := by
    cases predicate with
    | atom value => exact value.evaluate_agrees values fieldValues
    | all items => exact propertyPredicateAllEvaluate_agrees items values fieldValues
    | any items => exact propertyPredicateAnyEvaluate_agrees items values fieldValues
    | not item =>
        simpa [PropertyPredicate.evaluate, PropertyPredicate.denote] using
          booleanNot_agrees (item.evaluate values fieldValues) (item.denote values fieldValues)
            (PropertyPredicate.evaluate_agrees item values fieldValues)

  private theorem propertyPredicateAllEvaluate_agrees
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) :
      propertyPredicateAllEvaluate items values fieldValues = true ↔
        propertyPredicateAllDenote items values fieldValues := by
    cases items with
    | nil => simp [propertyPredicateAllEvaluate, propertyPredicateAllDenote]
    | cons item rest =>
        simp [propertyPredicateAllEvaluate, propertyPredicateAllDenote,
          PropertyPredicate.evaluate_agrees item values fieldValues,
          propertyPredicateAllEvaluate_agrees rest values fieldValues]

  private theorem propertyPredicateAnyEvaluate_agrees
      (items : List PropertyPredicate)
      (values : PropertyPredicateField → List ModelValue)
      (fieldValues : List PropertyFieldValue := []) :
      propertyPredicateAnyEvaluate items values fieldValues = true ↔
        propertyPredicateAnyDenote items values fieldValues := by
    cases items with
    | nil => simp [propertyPredicateAnyEvaluate, propertyPredicateAnyDenote]
    | cons item rest =>
        simp [propertyPredicateAnyEvaluate, propertyPredicateAnyDenote,
          PropertyPredicate.evaluate_agrees item values fieldValues,
          propertyPredicateAnyEvaluate_agrees rest values fieldValues]
end

/-- A complete same-step input checked against every atom before Boolean evaluation begins. The
predicate index prevents reuse of an input checked for another predicate in the same context. -/
structure CheckedPropertyPredicateInput
    {context : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate context) where
  private input : PropertyPredicateInput
  private fieldValues : List PropertyFieldValue := []
  deriving Repr

/-- Same-step context fixed by input validation. -/
def CheckedPropertyPredicateInput.contextKind
    {context : PropertyPredicateContext}
    {predicate : CheckedPropertyPredicate context}
    (_input : CheckedPropertyPredicateInput predicate) : PropertyPredicateContext :=
  context

/-- Project the complete values of one field from a validated same-step input. -/
def CheckedPropertyPredicateInput.valuesAt
    {context : PropertyPredicateContext}
    {predicate : CheckedPropertyPredicate context}
    (input : CheckedPropertyPredicateInput predicate)
    (field : PropertyPredicateField) : List ModelValue :=
  match field with
  | .priorState => input.input.priorState.toList
  | .selectedAction => input.input.selectedAction.toList
  | .resultingState => input.input.resultingState.toList
  | .outcome => input.input.outcome.toList
  | .expectationFact => input.input.facts.getD []

private def inputFieldValues (input : PropertyPredicateInput) : PropertyPredicateField → List ModelValue
  | .priorState => input.priorState.toList
  | .selectedAction => input.selectedAction.toList
  | .resultingState => input.resultingState.toList
  | .outcome => input.outcome.toList
  | .expectationFact => input.facts.getD []

private def operandModelValues (input : PropertyPredicateInput) : PropertyFieldRoot → List ModelValue
  | .request => input.selectedAction.toList
  | .priorState => input.priorState.toList
  | .resultingState => input.resultingState.toList
  | .outcome => input.outcome.toList
  | .event => input.facts.getD []

private def validateFieldOperands (predicate : CheckedPropertyPredicate context)
    (input : PropertyPredicateInput) (fieldValues : List PropertyFieldValue)
    (expression : PropertyPredicate) : Except PropertyError Unit := do
  match expression with
  | .atom atom =>
    for comparison in atom.fieldComparison do
      for operand in [comparison.left, comparison.right] do
        if let .field path source := operand then
          let matching := fieldValues.filter (·.path == path)
          let fail reason : Except PropertyError Unit := .error {
            kind := .missingPredicateInput, definitionId := predicate.definitionId,
            sourcePath := source.displayPath, sourceLocation := some source,
            offendingValue := reason, relatedDefinitionIds := [path.reference] }
          let [value] := matching | fail "missing or ambiguous field operand"
          -- A capture operand denotes an earlier admitted step, so this step's payloads cannot
          -- carry it; its authority is the retained projection the reading Run recorded.
          if path.capture.isNone && !(operandModelValues input path.root).contains value.modelValue then
            fail "field operand does not denote this immutable model payload"
  | .all items =>
    for item in items do
      validateFieldOperands predicate input fieldValues item
      if !item.evaluate (inputFieldValues input) fieldValues then break
  | .any items =>
    for item in items do
      validateFieldOperands predicate input fieldValues item
      if item.evaluate (inputFieldValues input) fieldValues then break
  | .not item => validateFieldOperands predicate input fieldValues item

private def checkPredicateInputValues (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput) (values : List PropertyFieldValue) :
    Except PropertyError (CheckedPropertyPredicateInput predicate) := do
  validatePropertyPredicateInput predicate input
  validateFieldOperands predicate input values predicate.expression
  pure { input, fieldValues := values }

/-- Check legacy input completeness and every field operand consumed by this Boolean branch. -/
def checkPropertyPredicateInput (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput) : Except PropertyError (CheckedPropertyPredicateInput predicate) :=
  checkPredicateInputValues predicate input []

/-- Produce a complete checked predicate input from an explicit kernel-checked validation proof. -/
def checkedPropertyPredicateInput
    (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput)
    (valid : (checkPropertyPredicateInput predicate input).toOption.isSome = true) :
    CheckedPropertyPredicateInput predicate :=
  (checkPropertyPredicateInput predicate input).toOption.get valid

private def fieldOperands : PropertyPredicate → List PropertyFieldOperand
  | .atom atom => atom.fieldComparison.toList.flatMap fun comparison => [comparison.left, comparison.right]
  | .all items | .any items => items.flatMap fieldOperands
  | .not item => fieldOperands item

/-- Read an operand only through the complete input admitted for this exact Property predicate. -/
def CheckedPropertyPredicateInput.operandValue (input : CheckedPropertyPredicateInput predicate)
    (operand : PropertyFieldOperand) : Option Scalar :=
  if (fieldOperands predicate.expression).contains operand then operand.resolve input.fieldValues else none

/-- Field operand denotation is the actual checked cursor derivation over its immutable model root. -/
def CheckedPropertyPredicateInput.denotesOperand (input : CheckedPropertyPredicateInput predicate)
    (operand : PropertyFieldOperand) (scalar : Scalar) : Prop :=
  match operand with
  | .literal value _ => value = scalar
  | .field path _ => ∃ value ∈ input.fieldValues, value.path = path ∧
      ∃ raw bound, Field.Denotes value.origin path.steps (some raw) ∧ readLiteral bound raw = some scalar

/-- Successful operand reads preserve concrete field selection and exact scalar decoding. -/
theorem CheckedPropertyPredicateInput.operandValue_denotes (input : CheckedPropertyPredicateInput predicate)
    (operand : PropertyFieldOperand) (scalar : Scalar)
    (h : input.operandValue operand = some scalar) : input.denotesOperand operand scalar := by
  have h : operand.resolve input.fieldValues = some scalar := by
    unfold operandValue at h
    split at h
    · exact h
    · contradiction
  cases operand with
  | literal value source => simpa [PropertyFieldOperand.resolve, denotesOperand] using h
  | field path source =>
    unfold PropertyFieldOperand.resolve at h
    cases found : input.fieldValues.find? (fun value => decide (value.path = path)) with
    | none => simp [found] at h
    | some value =>
      have selectedBool := List.find?_some found
      have selected : value.path = path := by simpa using selectedBool
      have same : value.scalar = scalar := by simpa [found] using h
      obtain ⟨raw, bound, denotes, parsed⟩ := value.denotation
      exact ⟨value, List.mem_of_find?_eq_some found, selected, raw, bound,
        selected ▸ denotes, same ▸ parsed⟩

/-- A field projection stays indexed by its generated owner and payload witness until input checking. -/
structure PropertyFieldProjection (owner : RpcOwner) {Request Response : Type}
    (witness : owner.Witness Request Response) where
  private mk ::
  private value : PropertyFieldValue

/-- Project a modeled state, result or event; requests must use the selected Action constructor. -/
def PropertyFieldProjection.ofCursor (root : PropertyFieldRoot) (reference : DefinitionId)
    (cursor : Field.Cursor owner witness side limits type .singular .available)
    (notRequest : root ≠ .request) (source : SourceLocation) :
    Except Field.Error (PropertyFieldProjection owner witness) := do
  let _ := notRequest
  pure ⟨← PropertyFieldValue.ofCursor root reference cursor source⟩

/-- A request projection is tied to the exact immutable arguments of the selected Action. -/
def PropertyFieldProjection.ofAction
    {owner : RpcOwner} {Request Response Failure : Type}
    {template : ActionTemplate owner Request Response Failure} {limits : Limits}
    (action : ActionInstance template limits)
    (cursor : Field.Cursor owner template.declaration.reference .request limits type .singular .available)
    (same : cursor.origin.value = action.arguments.value) (source : SourceLocation) :
    Except Field.Error (PropertyFieldProjection owner template.declaration.reference) := do
  pure ⟨← PropertyFieldValue.ofAction action cursor same source⟩

/-- The exact legacy carrier for this modeled payload; it exposes no raw field evidence. -/
def PropertyFieldProjection.modelValue (projection : PropertyFieldProjection owner witness) : ModelValue :=
  projection.value.modelValue

/-- One admitted projection with its generated index erased. The authority a projection was built
from survives in its structural path, so a single same-step context may carry independently bound
request, prior/resulting state, outcome and event operands whose owners and schemas differ. -/
structure PropertyFieldEvidence where
  private mk ::
  private value : PropertyFieldValue
  deriving Repr

/-- Admit a checked projection into a heterogeneous same-step evidence list. -/
def PropertyFieldProjection.evidence (projection : PropertyFieldProjection owner witness) :
    PropertyFieldEvidence :=
  ⟨projection.value⟩

/-- The structural coordinates this admitted projection denotes. -/
def PropertyFieldEvidence.path (evidence : PropertyFieldEvidence) : PropertyFieldPath :=
  evidence.value.path

/-- Re-key an already admitted projection as the retained value of one exact earlier occurrence.
Only the capture coordinates naming it change: the structural steps, the decoded scalar and its
denotation are the ones the original checked cursor established. Recording which occurrence a
retained value is remains the reading Run's own bookkeeping, not a claim about any later payload. -/
def PropertyFieldEvidence.capturedAs (evidence : PropertyFieldEvidence)
    (key : PropertyFieldCaptureKey) : PropertyFieldEvidence :=
  ⟨{ evidence.value with path := { evidence.value.path with capture := some key } }⟩

/-- Check a same-step input together with the field evidence its operands read, admitting both
this step's projections and the retained captures a correlated Monitor supplies. -/
def checkPropertyPredicateInputWithEvidence (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput) (evidence : List PropertyFieldEvidence) :
    Except PropertyError (CheckedPropertyPredicateInput predicate) :=
  checkPredicateInputValues predicate input (evidence.map (·.value))

/-- A field predicate wraps the existing checked Property predicate without another evaluator. -/
structure CheckedFieldPredicate (context : PropertyPredicateContext) where
  private mk ::
  predicate : CheckedPropertyPredicate context

/-- Check closed operands against the admitted field bindings and the ordinary Property checker.
Each operand names its own binding, so no single generated authority is imposed on the context. -/
def CheckedFieldPredicate.check (context : PropertyCheckContext)
    (declaration : Property) (kind : PropertyPredicateContext)
    (expression : PropertyPredicate) : Except PropertyError (CheckedFieldPredicate kind) := do
  pure ⟨← checkPropertyPredicate context declaration kind expression⟩

/-- Only evidence erased from a checked projection may establish this predicate's input; every
operand still resolves to the single projection whose exact structural path it names. -/
def CheckedFieldPredicate.checkInput (checked : CheckedFieldPredicate kind)
    (input : PropertyPredicateInput) (evidence : List PropertyFieldEvidence) :
    Except PropertyError (CheckedPropertyPredicateInput checked.predicate) :=
  checkPropertyPredicateInputWithEvidence checked.predicate input evidence

/-- Denotational meaning of a validated predicate over one validated same-step input. -/
def CheckedPropertyPredicate.denote
    {predicateContext : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate predicateContext)
    (input : CheckedPropertyPredicateInput predicate) : Prop :=
  predicate.expression.denote input.valuesAt input.fieldValues

/-- Evaluate a predicate only after both its syntax and complete same-step input passed checking. -/
def evaluatePropertyPredicate
    {predicateContext : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate predicateContext)
    (input : CheckedPropertyPredicateInput predicate) : Bool :=
  predicate.expression.evaluate input.valuesAt input.fieldValues

/-- Generic kernel-checked agreement for every admitted Boolean Property predicate. -/
theorem evaluatePropertyPredicate_agrees
    {predicateContext : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate predicateContext)
    (input : CheckedPropertyPredicateInput predicate) :
    evaluatePropertyPredicate predicate input = true ↔ predicate.denote input :=
  predicate.expression.evaluate_agrees input.valuesAt input.fieldValues

private structure PropertyEvaluationPredicateInput extends PropertyPredicateInput where
  fieldValues : List PropertyFieldValue := []

private structure PropertyEvaluationStep extends PropertyTraceStep where
  fieldValues : List PropertyFieldValue := []
  deriving Repr

private structure PropertyEvaluationView where
  initialState : Option ModelValue
  steps : List PropertyEvaluationStep
  deriving Repr

private def validateEvaluationPredicateInput (predicate : CheckedPropertyPredicate kind)
    (input : PropertyEvaluationPredicateInput) : Except PropertyError Unit := do
  validatePropertyPredicateInput predicate input.toPropertyPredicateInput
  validateFieldOperands predicate input.toPropertyPredicateInput input.fieldValues predicate.expression

/-- A trace view whose every guarded same-step input was validated for this exact Property. -/
structure CheckedPropertyEvaluationInput (property : CheckedProperty) where
  private view : PropertyEvaluationView
  deriving Repr

private def guardInput (step : PropertyEvaluationStep) : PropertyEvaluationPredicateInput := {
  context := .before
  priorState := step.priorState
  selectedAction := step.selectedAction
  fieldValues := step.fieldValues
}

private def expectationInput (step : PropertyEvaluationStep) : PropertyEvaluationPredicateInput := {
  context := .after
  priorState := step.priorState
  selectedAction := step.selectedAction
  fieldValues := step.fieldValues
  resultingState := step.resultingState
  outcome := step.outcome
  facts := some step.observations
}

private def valuesInStep
    (field : PropertyTraceField)
    (step : PropertyEvaluationStep) : List ModelValue :=
  match field with
  | .state | .resultingState => step.resultingState.toList
  | .priorState => step.priorState.toList
  | .selectedAction => step.selectedAction.toList
  | .outcome => step.outcome.toList
  | .observation | .relation => step.observations

private def patternHoldsInStep
    (pattern : PropertyPattern)
    (step : PropertyEvaluationStep) : Bool :=
  (valuesInStep pattern.field step).any pattern.evaluate

private def validateResolvedExceptionInput
    (exception : Option CheckedPropertyUnless)
    (input : PropertyEvaluationPredicateInput) : Except PropertyError Unit := do
  match exception with
  | none => pure ()
  | some exception =>
      let _ ← validateEvaluationPredicateInput exception.condition input
      pure ()

private def validateCaseGroupStep
    (group : CheckedPropertyBranches)
    (step : PropertyEvaluationStep) : Except PropertyError Unit := do
  let guardValues := guardInput step
  let expectationValues := expectationInput step
  let _ ← validateEvaluationPredicateInput group.guard guardValues
  validateResolvedExceptionInput group.exception guardValues
  for item in group.cases do
    let _ ← validateEvaluationPredicateInput item.guard guardValues
    validateResolvedExceptionInput item.exception guardValues
    for clause in item.clauses do
      let _ ← validateEvaluationPredicateInput clause.expectation expectationValues
      pure ()

private def validateGuardedTemporalStep
    (clause : CheckedPropertyTemporalClause)
    (step : PropertyEvaluationStep) : Except PropertyError Unit := do
  if patternHoldsInStep clause.trigger step then
    let input := guardInput step
    let _ ← validateEvaluationPredicateInput clause.guard input
    validateResolvedExceptionInput clause.exception input
    for guard in clause.caseGuard do
      let _ ← validateEvaluationPredicateInput guard input
      pure ()
    validateResolvedExceptionInput clause.caseException input

/-- Validate every same-step slot before guarded Boolean reduction begins. Known absent facts are
represented by `some []`; only unavailable fields produce an error. -/
private def validatePropertyEvaluationView
    (property : CheckedProperty)
    (view : PropertyEvaluationView) : Except PropertyError Unit := do
  if let some clause := property.correlatedRules.head? then
    throw {
      kind := .invalidClause
      definitionId := clause.declaration.id
      sourcePath := clause.declaration.source.path
      sourceLocation := some clause.declaration.source
      offendingValue := "correlated rules require admitted operation traces",
      relatedDefinitionIds := [clause.declaration.id] }
  for clause in property.clauses do
    match clause with
    | .branches group =>
        if group.complete || group.exclusive || group.cases.any (fun item => !item.clauses.isEmpty) then
          for step in view.steps do
            validateCaseGroupStep group step
        for item in group.cases do
          for temporal in item.temporalClauses do
            for step in view.steps do
              validateGuardedTemporalStep temporal step
    | .guardedEventuallyWithin guarded | .guardedNeverWithin guarded =>
        for step in view.steps do
          validateGuardedTemporalStep guarded step
    | _ => pure ()
  pure ()

/-- Admit a legacy trace with no field evidence through the same complete input validator. -/
def checkPropertyEvaluationInput
    (property : CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Except PropertyError (CheckedPropertyEvaluationInput property) := do
  let raw := property.traceView trace
  let view : PropertyEvaluationView := {
    initialState := raw.initialState, steps := raw.steps.map fun step => { toPropertyTraceStep := step } }
  validatePropertyEvaluationView property view
  pure { view }

/-- A checked Property whose clauses read same-step field evidence admitted one step at a time. -/
structure CheckedFieldProperty where
  private mk ::
  property : CheckedProperty

/-- Check an ordinary closed Property declaration whose clauses compare admitted field bindings. -/
def CheckedFieldProperty.check (context : PropertyCheckContext)
    (declaration : Property) : Except PropertyError CheckedFieldProperty := do
  pure ⟨← Property.check context (declaration)⟩

/-- Validate aligned checked evidence without exposing raw model payloads to evaluation. -/
def CheckedFieldProperty.checkInput (checked : CheckedFieldProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (evidence : List (List PropertyFieldEvidence)) :
    Except PropertyError (CheckedPropertyEvaluationInput checked.property) := do
  if evidence.length != trace.steps.length then
    throw {
      kind := .missingPredicateInput, definitionId := checked.property.id,
      sourcePath := checked.property.source.displayPath, sourceLocation := some checked.property.source,
      offendingValue := "field projection step count mismatch", relatedDefinitionIds := [] }
  let raw := checked.property.traceView trace
  let view : PropertyEvaluationView := {
    initialState := raw.initialState
    steps := raw.steps.zipWith (fun step fields => {
      toPropertyTraceStep := step, fieldValues := fields.map (·.value) }) evidence }
  validatePropertyEvaluationView checked.property view
  pure { view }

private def predicateValues
    (input : PropertyEvaluationPredicateInput)
    (field : PropertyPredicateField) : List ModelValue :=
  match field with
  | .priorState => input.priorState.toList
  | .selectedAction => input.selectedAction.toList
  | .resultingState => input.resultingState.toList
  | .outcome => input.outcome.toList
  | .expectationFact => input.facts.getD []

private def evaluateCheckedPredicate
    (predicate : CheckedPropertyPredicate context)
    (input : PropertyEvaluationPredicateInput) : Bool :=
  predicate.expression.evaluate (predicateValues input) input.fieldValues

private def checkedPredicateDenotes
    (predicate : CheckedPropertyPredicate context)
    (input : PropertyEvaluationPredicateInput) : Prop :=
  predicate.expression.denote (predicateValues input) input.fieldValues

private theorem evaluateCheckedPredicate_agrees
    (predicate : CheckedPropertyPredicate context)
    (input : PropertyEvaluationPredicateInput) :
    evaluateCheckedPredicate predicate input = true ↔ checkedPredicateDenotes predicate input :=
  predicate.expression.evaluate_agrees (predicateValues input) input.fieldValues

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
  logicalTime : Option Nat
  deriving BEq, DecidableEq, Repr

private def observationOccurrences
    (pattern : PropertyPattern)
    (transitionPosition selectedActionPosition : Nat)
    (logicalTime : Option Nat) : List ModelValue → List PropertyOccurrence
  | [] => []
  | value :: rest =>
      let tail := observationOccurrences pattern transitionPosition selectedActionPosition
        logicalTime rest
      if pattern.evaluate value then
        {
          value
          transitionPosition
          selectedActionPosition
          logicalTime
        } :: tail
      else
        tail

private def optionalOccurrence
    (pattern : PropertyPattern)
    (transitionPosition selectedActionPosition : Nat)
    (logicalTime : Option Nat)
    (value : Option ModelValue) : List PropertyOccurrence :=
  match value with
  | some value =>
      if pattern.evaluate value then [{
        value
        transitionPosition
        selectedActionPosition
        logicalTime
      }] else []
  | none => []

private def stepOccurrences
    (pattern : PropertyPattern)
    (transitionPosition : Nat)
    (step : PropertyEvaluationStep) : List PropertyOccurrence :=
  match pattern.field with
  | .state | .resultingState =>
      optionalOccurrence pattern transitionPosition transitionPosition
        step.logicalTime step.resultingState
  | .priorState =>
      optionalOccurrence pattern (transitionPosition - 1) transitionPosition
        step.logicalTime step.priorState
  | .selectedAction =>
      optionalOccurrence pattern transitionPosition transitionPosition
        step.logicalTime step.selectedAction
  | .outcome =>
      optionalOccurrence pattern transitionPosition transitionPosition
        step.logicalTime step.outcome
  | .observation | .relation =>
      observationOccurrences pattern transitionPosition transitionPosition
        step.logicalTime step.observations

private def traceStepOccurrences
    (pattern : PropertyPattern)
    (transitionPosition : Nat) :
    List PropertyEvaluationStep → List PropertyOccurrence
  | [] => []
  | step :: rest =>
      stepOccurrences pattern transitionPosition step ++
        traceStepOccurrences pattern (transitionPosition + 1) rest

private def occurrences
    (pattern : PropertyPattern)
    (view : PropertyEvaluationView) : List PropertyOccurrence :=
  let initial := if pattern.field == .state then
    optionalOccurrence pattern 0 0 none view.initialState
  else
    []
  initial ++ traceStepOccurrences pattern 1 view.steps

private def positionOf
    (unit : LimitUnit)
    (occurrence : PropertyOccurrence) : Option Nat :=
  match unit with
  | .steps => some occurrence.transitionPosition
  | .actions => some occurrence.selectedActionPosition
  | .logicalTime => occurrence.logicalTime
  | .search => none
  | .plans => none

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
    (view : PropertyEvaluationView) : Option (List Nat) :=
  collectPositions ((occurrences pattern view).map (positionOf unit))

/-- Aligned semantic transition positions used by correlated correspondence proofs. -/
def Property.Correlated.positions (start : Nat) : List Bool → List Nat
  | [] => []
  | hit :: rest => (if hit then [start] else []) ++ Property.Correlated.positions (start + 1) rest

private theorem collectPositions_some (positions : List Nat) :
    collectPositions (positions.map some) = some positions := by
  induction positions with
  | nil => rfl
  | cons position rest ih => simp [collectPositions, ih]

private theorem checkedPositions_semantic (pattern : PropertyPattern) (view : PropertyEvaluationView) :
    checkedPositions pattern .steps view =
      some ((occurrences pattern view).map (·.transitionPosition)) := by
  have position : positionOf .steps = fun occurrence => some occurrence.transitionPosition := rfl
  simpa [checkedPositions, List.map_map, Function.comp_def, position] using
    collectPositions_some ((occurrences pattern view).map (·.transitionPosition))

private theorem observation_positions_mem (pattern : PropertyPattern)
    (start selected : Nat) (time : Option Nat) (values : List ModelValue) (position : Nat) :
    position ∈ (observationOccurrences pattern start selected time values).map
      (·.transitionPosition) ↔ position = start ∧ values.any pattern.evaluate = true := by
  induction values with
  | nil => simp [observationOccurrences]
  | cons value rest ih =>
      simp only [observationOccurrences]
      split <;> simp_all

private theorem step_positions_mem (pattern : PropertyPattern)
    (aligned : pattern.field = .selectedAction ∨ pattern.field = .outcome ∨
      pattern.field = .resultingState ∨ pattern.field = .observation)
    (start position : Nat) (step : PropertyEvaluationStep) :
    position ∈ (stepOccurrences pattern start step).map (·.transitionPosition) ↔
      position = start ∧ patternHoldsInStep pattern step = true := by
  rcases aligned with action | outcome | state | fact
  · cases value : step.selectedAction with
    | none => simp [stepOccurrences, action, optionalOccurrence, value, patternHoldsInStep, valuesInStep]
    | some item => cases hit : pattern.evaluate item <;>
        simp [stepOccurrences, action, optionalOccurrence, value, patternHoldsInStep, valuesInStep, hit]
  · cases value : step.outcome with
    | none => simp [stepOccurrences, outcome, optionalOccurrence, value, patternHoldsInStep, valuesInStep]
    | some item => cases hit : pattern.evaluate item <;>
        simp [stepOccurrences, outcome, optionalOccurrence, value, patternHoldsInStep, valuesInStep, hit]
  · cases value : step.resultingState with
    | none => simp [stepOccurrences, state, optionalOccurrence, value, patternHoldsInStep, valuesInStep]
    | some item => cases hit : pattern.evaluate item <;>
        simp [stepOccurrences, state, optionalOccurrence, value, patternHoldsInStep, valuesInStep, hit]

  · simpa [stepOccurrences, fact, patternHoldsInStep, valuesInStep] using
      observation_positions_mem pattern start start step.logicalTime step.observations position

private theorem correlated_step_positions_mem (pattern : PropertyPattern)
    (aligned : pattern.field = .selectedAction ∨ pattern.field = .outcome ∨
      pattern.field = .resultingState ∨ pattern.field = .observation)
    (start position : Nat) (steps : List PropertyEvaluationStep) :
    position ∈ (traceStepOccurrences pattern start steps).map (·.transitionPosition) ↔
      position ∈ Property.Correlated.positions start (steps.map (patternHoldsInStep pattern)) := by
  induction steps generalizing start with
  | nil => rfl
  | cons step rest ih =>
      simp only [traceStepOccurrences, List.map_append, List.mem_append,
        step_positions_mem pattern aligned, ih, List.map_cons, Property.Correlated.positions]
      cases patternHoldsInStep pattern step <;> simp

private theorem semantic_positions_mem (pattern : PropertyPattern)
    (aligned : pattern.field = .selectedAction ∨ pattern.field = .outcome ∨
      pattern.field = .resultingState ∨ pattern.field = .observation)
    (view : PropertyEvaluationView) (position : Nat) :
    position ∈ (occurrences pattern view).map (·.transitionPosition) ↔
      position ∈ Property.Correlated.positions 1 (view.steps.map (patternHoldsInStep pattern)) := by
  have notState : (pattern.field == .state) = false := by
    rcases aligned with action | outcome | state | fact
    · rw [action]; rfl
    · rw [outcome]; rfl
    · rw [state]; rfl
    · rw [fact]; rfl
  simp only [occurrences, notState, Bool.false_eq_true, ↓reduceIte, List.nil_append]
  exact correlated_step_positions_mem pattern aligned 1 position view.steps

private def valuesAtField
    (field : PropertyTraceField)
    (view : PropertyEvaluationView) : List ModelValue :=
  let initial := match field with
    | .state => view.initialState.toList
    | _ => []
  let fromSteps := view.steps.flatMap fun step =>
    match field with
    | .state | .resultingState => step.resultingState.toList
    | .priorState => step.priorState.toList
    | .selectedAction => step.selectedAction.toList
    | .outcome => step.outcome.toList
    | .observation | .relation => step.observations
  initial ++ fromSteps

private def patternDenotesInStep
    (pattern : PropertyPattern)
    (step : PropertyEvaluationStep) : Prop :=
  anyHolds (valuesInStep pattern.field step) pattern.denote

private theorem patternHoldsInStep_agrees
    (pattern : PropertyPattern)
    (step : PropertyEvaluationStep) :
    patternHoldsInStep pattern step = true ↔ patternDenotesInStep pattern step :=
  anyHolds_agrees _ _ _ pattern.evaluate_agrees

private def evaluateStateInvariant
    (pattern : PropertyPattern)
    (view : PropertyEvaluationView) : Bool :=
  let matching := (valuesAtField .state view).filter fun value => value.definitionId == pattern.reference
  !matching.isEmpty && matching.all fun value => pattern.constraint.evaluate value.value

private def stateInvariantDenotes
    (pattern : PropertyPattern)
    (view : PropertyEvaluationView) : Prop :=
  let matching := (valuesAtField .state view).filter fun value => value.definitionId == pattern.reference
  matching ≠ [] ∧ allHolds matching fun value => pattern.constraint.denote value.value

private theorem evaluateStateInvariant_agrees
    (pattern : PropertyPattern)
    (view : PropertyEvaluationView) :
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
    (view : PropertyEvaluationView) : Bool :=
  view.steps.all fun step =>
    !patternHoldsInStep precondition step || patternHoldsInStep postcondition step

private def transitionContractDenotes
    (precondition postcondition : PropertyPattern)
    (view : PropertyEvaluationView) : Prop :=
  allHolds view.steps fun step =>
    patternDenotesInStep precondition step → patternDenotesInStep postcondition step

private theorem evaluateTransitionContract_agrees
    (precondition postcondition : PropertyPattern)
    (view : PropertyEvaluationView) :
    evaluateTransitionContract precondition postcondition view = true ↔
      transitionContractDenotes precondition postcondition view :=
  allHolds_agrees _ _ _ fun step =>
    booleanImplication_agrees _ _ _ _
      (patternHoldsInStep_agrees precondition step)
      (patternHoldsInStep_agrees postcondition step)

private def evaluateIdentityRelation
    (relation : PropertyPattern)
    (view : PropertyEvaluationView) : Bool :=
  (valuesAtField relation.field view).any relation.evaluate

private def identityRelationDenotes
    (relation : PropertyPattern)
    (view : PropertyEvaluationView) : Prop :=
  anyHolds (valuesAtField relation.field view) relation.denote

private theorem evaluateIdentityRelation_agrees
    (relation : PropertyPattern)
    (view : PropertyEvaluationView) :
    evaluateIdentityRelation relation view = true ↔ identityRelationDenotes relation view :=
  anyHolds_agrees _ _ _ relation.evaluate_agrees

private def evaluateOrdered
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyEvaluationView) : Bool :=
  match checkedPositions before unit view, checkedPositions after unit view with
  | some beforePositions, some afterPositions =>
      beforePositions.any fun first => afterPositions.any fun second => first < second
  | _, _ => false

private def orderedDenotes
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyEvaluationView) : Prop :=
  match checkedPositions before unit view, checkedPositions after unit view with
  | some beforePositions, some afterPositions =>
      anyHolds beforePositions fun first => anyHolds afterPositions fun second => first < second
  | _, _ => False

private theorem evaluateOrdered_agrees
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyEvaluationView) :
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
    (view : PropertyEvaluationView) : Bool :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggerPositions, some responsePositions =>
      triggerPositions.all fun first =>
        responsePositions.any fun second => first ≤ second && second - first ≤ limit.value
  | _, _ => false

private def eventuallyWithinDenotes
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyEvaluationView) : Prop :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggerPositions, some responsePositions =>
      allHolds triggerPositions fun first =>
        anyHolds responsePositions fun second =>
          first ≤ second ∧ second - first ≤ limit.value
  | _, _ => False

private theorem evaluateEventuallyWithin_agrees
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyEvaluationView) :
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
    (view : PropertyEvaluationView) : Bool :=
  match checkedPositions trigger limit.unit view, checkedPositions forbidden limit.unit view with
  | some triggerPositions, some forbiddenPositions =>
      triggerPositions.all fun first =>
        !(forbiddenPositions.any fun second => first ≤ second && second - first ≤ limit.value)
  | _, _ => false

private def quiescentWithinDenotes
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyEvaluationView) : Prop :=
  match checkedPositions trigger limit.unit view, checkedPositions forbidden limit.unit view with
  | some triggerPositions, some forbiddenPositions =>
      allHolds triggerPositions fun first =>
        ¬anyHolds forbiddenPositions fun second =>
          first ≤ second ∧ second - first ≤ limit.value
  | _, _ => False

private theorem evaluateQuiescentWithin_agrees
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyEvaluationView) :
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
    (exception : Option CheckedPropertyUnless)
    (input : PropertyEvaluationPredicateInput) : Bool :=
  match exception with
  | none => true
  | some exception => !(evaluateCheckedPredicate exception.condition input)

private def exceptionAllowsDenote
    (exception : Option CheckedPropertyUnless)
    (input : PropertyEvaluationPredicateInput) : Prop :=
  match exception with
  | none => True
  | some exception => ¬checkedPredicateDenotes exception.condition input

private theorem exceptionAllows_agrees
    (exception : Option CheckedPropertyUnless)
    (input : PropertyEvaluationPredicateInput) :
    exceptionAllowsEvaluate exception input = true ↔ exceptionAllowsDenote exception input := by
  cases exception with
  | none => simp [exceptionAllowsEvaluate, exceptionAllowsDenote]
  | some exception =>
      exact booleanNot_agrees _ _ (evaluateCheckedPredicate_agrees exception.condition input)

private def guardedTemporalAppliesEvaluate
    (clause : CheckedPropertyTemporalClause)
    (input : PropertyEvaluationPredicateInput) : Bool :=
  evaluateCheckedPredicate clause.guard input &&
    exceptionAllowsEvaluate clause.exception input &&
    clause.caseGuard.all (fun guard => evaluateCheckedPredicate guard input) &&
    exceptionAllowsEvaluate clause.caseException input

private def guardedTemporalAppliesDenote
    (clause : CheckedPropertyTemporalClause)
    (input : PropertyEvaluationPredicateInput) : Prop :=
  checkedPredicateDenotes clause.guard input ∧
    exceptionAllowsDenote clause.exception input ∧
    (∀ guard ∈ clause.caseGuard, checkedPredicateDenotes guard input) ∧
    exceptionAllowsDenote clause.caseException input

private theorem guardedTemporalApplies_agrees
    (clause : CheckedPropertyTemporalClause)
    (input : PropertyEvaluationPredicateInput) :
    guardedTemporalAppliesEvaluate clause input = true ↔
      guardedTemporalAppliesDenote clause input := by
  cases caseGuard : clause.caseGuard <;>
    simp [guardedTemporalAppliesEvaluate, guardedTemporalAppliesDenote,
      caseGuard, evaluateCheckedPredicate_agrees, exceptionAllows_agrees, and_assoc]

private def triggerPositionsInStep
    (clause : CheckedPropertyTemporalClause)
    (transitionPosition : Nat)
    (step : PropertyEvaluationStep) : Option (List Nat) :=
  collectPositions ((stepOccurrences clause.trigger transitionPosition step).map
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
    (clause : CheckedPropertyTemporalClause)
    (responsePositions : List Nat) :
    Nat → List PropertyEvaluationStep → Bool
  | _, [] => true
  | transitionPosition, step :: rest =>
      match triggerPositionsInStep clause transitionPosition step with
      | none => false
      | some triggerPositions =>
          let applies := !triggerPositions.isEmpty &&
            guardedTemporalAppliesEvaluate clause (guardInput step)
          (!applies || triggerPositions.all fun first =>
            responseWithinEvaluate forbidden first responsePositions clause.limit.value) &&
          evaluateGuardedTemporalSteps forbidden clause responsePositions
            (transitionPosition + 1) rest

private def guardedTemporalStepsDenote
    (forbidden : Bool)
    (clause : CheckedPropertyTemporalClause)
    (responsePositions : List Nat) :
    Nat → List PropertyEvaluationStep → Prop
  | _, [] => True
  | transitionPosition, step :: rest =>
      match triggerPositionsInStep clause transitionPosition step with
      | none => False
      | some triggerPositions =>
          ((triggerPositions ≠ [] ∧ guardedTemporalAppliesDenote clause (guardInput step)) →
            allHolds triggerPositions fun first =>
              responseWithinDenote forbidden first responsePositions clause.limit.value) ∧
          guardedTemporalStepsDenote forbidden clause responsePositions
            (transitionPosition + 1) rest

private theorem evaluateGuardedTemporalSteps_agrees
    (forbidden : Bool)
    (clause : CheckedPropertyTemporalClause)
    (responsePositions : List Nat)
    (transitionPosition : Nat)
    (steps : List PropertyEvaluationStep) :
    evaluateGuardedTemporalSteps forbidden clause responsePositions
        transitionPosition steps = true ↔
      guardedTemporalStepsDenote forbidden clause responsePositions
        transitionPosition steps := by
  induction steps generalizing transitionPosition with
  | nil => simp [evaluateGuardedTemporalSteps, guardedTemporalStepsDenote]
  | cons step rest inductionHypothesis =>
      cases triggerResult : triggerPositionsInStep clause transitionPosition step with
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
    (clause : CheckedPropertyTemporalClause)
    (view : PropertyEvaluationView) : Bool :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => false
  | some responsePositions =>
      evaluateGuardedTemporalSteps forbidden clause responsePositions 1 view.steps

private def guardedTemporalDenotes
    (forbidden : Bool)
    (clause : CheckedPropertyTemporalClause)
    (view : PropertyEvaluationView) : Prop :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => False
  | some responsePositions =>
      guardedTemporalStepsDenote forbidden clause responsePositions 1 view.steps

private theorem evaluateGuardedTemporal_agrees
    (forbidden : Bool)
    (clause : CheckedPropertyTemporalClause)
    (view : PropertyEvaluationView) :
    evaluateGuardedTemporal forbidden clause view = true ↔
      guardedTemporalDenotes forbidden clause view := by
  cases responseResult : checkedPositions clause.response clause.limit.unit view with
  | none => simp [evaluateGuardedTemporal, guardedTemporalDenotes, responseResult]
  | some responsePositions =>
      simpa [evaluateGuardedTemporal, guardedTemporalDenotes, responseResult] using
        evaluateGuardedTemporalSteps_agrees forbidden clause responsePositions 1 view.steps

private def parentAppliesEvaluate
    (group : CheckedPropertyBranches)
    (input : PropertyEvaluationPredicateInput) : Bool :=
  evaluateCheckedPredicate group.guard input && exceptionAllowsEvaluate group.exception input

private def parentAppliesDenote
    (group : CheckedPropertyBranches)
    (input : PropertyEvaluationPredicateInput) : Prop :=
  checkedPredicateDenotes group.guard input ∧ exceptionAllowsDenote group.exception input

private theorem parentApplies_agrees
    (group : CheckedPropertyBranches)
    (input : PropertyEvaluationPredicateInput) :
    parentAppliesEvaluate group input = true ↔ parentAppliesDenote group input := by
  simp [parentAppliesEvaluate, parentAppliesDenote,
    evaluateCheckedPredicate_agrees, exceptionAllows_agrees]

private def caseAppliesEvaluate
    (parentApplies : Bool)
    (item : CheckedPropertyBranch)
    (input : PropertyEvaluationPredicateInput) : Bool :=
  parentApplies && evaluateCheckedPredicate item.guard input &&
    exceptionAllowsEvaluate item.exception input

private def caseAppliesDenote
    (parentApplies : Prop)
    (item : CheckedPropertyBranch)
    (input : PropertyEvaluationPredicateInput) : Prop :=
  (parentApplies ∧ checkedPredicateDenotes item.guard input) ∧
    exceptionAllowsDenote item.exception input

private theorem caseApplies_agrees
    (parentEvaluate : Bool)
    (parentDenote : Prop)
    (parentAgreement : parentEvaluate = true ↔ parentDenote)
    (item : CheckedPropertyBranch)
    (input : PropertyEvaluationPredicateInput) :
    caseAppliesEvaluate parentEvaluate item input = true ↔
      caseAppliesDenote parentDenote item input := by
  simp [caseAppliesEvaluate, caseAppliesDenote, parentAgreement,
    evaluateCheckedPredicate_agrees, exceptionAllows_agrees]

private def exclusiveCasesEvaluate
    (parentApplies : Bool)
    (input : PropertyEvaluationPredicateInput) : List CheckedPropertyBranch → Bool
  | [] => true
  | item :: rest =>
      (!caseAppliesEvaluate parentApplies item input ||
        rest.all fun other => !caseAppliesEvaluate parentApplies other input) &&
      exclusiveCasesEvaluate parentApplies input rest

private def exclusiveCasesDenote
    (parentApplies : Prop)
    (input : PropertyEvaluationPredicateInput) : List CheckedPropertyBranch → Prop
  | [] => True
  | item :: rest =>
      (caseAppliesDenote parentApplies item input →
        allHolds rest fun other => ¬caseAppliesDenote parentApplies other input) ∧
      exclusiveCasesDenote parentApplies input rest

private theorem exclusiveCases_agrees
    (parentEvaluate : Bool)
    (parentDenote : Prop)
    (parentAgreement : parentEvaluate = true ↔ parentDenote)
    (input : PropertyEvaluationPredicateInput)
    (items : List CheckedPropertyBranch) :
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
    (input : PropertyEvaluationPredicateInput)
    (item : CheckedPropertyBranch) : Bool :=
  item.clauses.all fun clause =>
    !applies || evaluateCheckedPredicate clause.expectation input

private def caseClausesDenote
    (applies : Prop)
    (input : PropertyEvaluationPredicateInput)
    (item : CheckedPropertyBranch) : Prop :=
  allHolds item.clauses fun clause =>
    applies → checkedPredicateDenotes clause.expectation input

private theorem caseClauses_agrees
    (appliesEvaluate : Bool)
    (appliesDenote : Prop)
    (appliesAgreement : appliesEvaluate = true ↔ appliesDenote)
    (input : PropertyEvaluationPredicateInput)
    (item : CheckedPropertyBranch) :
    caseClausesEvaluate appliesEvaluate input item = true ↔
      caseClausesDenote appliesDenote input item :=
  allHolds_agrees _ _ _ fun clause =>
    booleanImplication_agrees _ _ _ _ appliesAgreement
      (evaluateCheckedPredicate_agrees clause.expectation input)

private def evaluateCaseGroupStep
    (group : CheckedPropertyBranches)
    (step : PropertyEvaluationStep) : Bool :=
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
    (group : CheckedPropertyBranches)
    (step : PropertyEvaluationStep) : Prop :=
  let guardValues := guardInput step
  let expectationValues := expectationInput step
  let parent := parentAppliesDenote group guardValues
  ((¬group.complete = true ∨
      (parent → anyHolds group.cases fun item => caseAppliesDenote parent item guardValues)) ∧
    (¬group.exclusive = true ∨ exclusiveCasesDenote parent guardValues group.cases)) ∧
  allHolds group.cases fun item =>
    caseClausesDenote (caseAppliesDenote parent item guardValues) expectationValues item

private theorem evaluateCaseGroupStep_agrees
    (group : CheckedPropertyBranches)
    (step : PropertyEvaluationStep) :
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
    (group : CheckedPropertyBranches)
    (view : PropertyEvaluationView) : Bool :=
  view.steps.all (evaluateCaseGroupStep group) &&
    group.cases.all fun item =>
      item.temporalClauses.all fun clause =>
        evaluateGuardedTemporal clause.forbidden clause view

private def caseGroupDenotes
    (group : CheckedPropertyBranches)
    (view : PropertyEvaluationView) : Prop :=
  allHolds view.steps (caseGroupStepDenotes group) ∧
    allHolds group.cases fun item =>
      allHolds item.temporalClauses fun clause =>
        guardedTemporalDenotes clause.forbidden clause view

private theorem evaluateCaseGroup_agrees
    (group : CheckedPropertyBranches)
    (view : PropertyEvaluationView) :
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
    (clause : CheckedPropertyClause)
    (view : PropertyEvaluationView) : Prop :=
  match clause with
  | .stateInvariant _ state => stateInvariantDenotes state view
  | .transitionContract _ precondition postcondition =>
      transitionContractDenotes precondition postcondition view
  | .identityRelation _ relation => identityRelationDenotes relation view
  | .inputOutput _ input output => transitionContractDenotes input output view
  | .ordered _ before after unit => orderedDenotes before after unit view
  | .eventuallyWithin _ trigger response limit =>
      eventuallyWithinDenotes trigger response limit view
  | .neverWithin _ trigger forbidden limit =>
      quiescentWithinDenotes trigger forbidden limit view
  | .branches group => caseGroupDenotes group view
  | .guardedEventuallyWithin guarded =>
      guardedTemporalDenotes false guarded view
  | .guardedNeverWithin guarded =>
      guardedTemporalDenotes true guarded view

private def evaluateResolvedPropertyClause
    (clause : CheckedPropertyClause)
    (view : PropertyEvaluationView) : Bool :=
  match clause with
  | .stateInvariant _ state => evaluateStateInvariant state view
  | .transitionContract _ precondition postcondition =>
      evaluateTransitionContract precondition postcondition view
  | .identityRelation _ relation => evaluateIdentityRelation relation view
  | .inputOutput _ input output => evaluateTransitionContract input output view
  | .ordered _ before after unit => evaluateOrdered before after unit view
  | .eventuallyWithin _ trigger response limit =>
      evaluateEventuallyWithin trigger response limit view
  | .neverWithin _ trigger forbidden limit =>
      evaluateQuiescentWithin trigger forbidden limit view
  | .branches group => evaluateCaseGroup group view
  | .guardedEventuallyWithin guarded =>
      evaluateGuardedTemporal false guarded view
  | .guardedNeverWithin guarded =>
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

/-- Aligned trigger/response truth values from this capability-limited checked input. -/
def CheckedPropertyEvaluationInput.correlatedCoordinates
    {property : CheckedProperty} (input : CheckedPropertyEvaluationInput property)
    (trigger response : PropertyPattern) : List (Bool × Bool) :=
  input.view.steps.map fun step =>
    (patternHoldsInStep trigger step, patternHoldsInStep response step)

/-- Compose already checked step views for a single semantic-transition clause. Target trace
continuity is the correlated consumer's admission responsibility; Property access remains unchanged. -/
def CheckedPropertyEvaluationInput.appendCorrelated
    {property : CheckedProperty} (first second : CheckedPropertyEvaluationInput property)
    (id : DefinitionId) (trigger response : PropertyPattern) (bound : Nat)
    (_shape : property.clauses = [.eventuallyWithin id trigger response ⟨bound, .steps⟩]) :
    CheckedPropertyEvaluationInput property :=
  ⟨{ first.view with steps := first.view.steps ++ second.view.steps }⟩

/-- Aligned coordinate projection commutes with incremental checked-input composition. -/
theorem CheckedPropertyEvaluationInput.correlatedCoordinates_append
    {property : CheckedProperty} (first second : CheckedPropertyEvaluationInput property)
    (id : DefinitionId) (trigger response : PropertyPattern) (bound : Nat)
    (shape : property.clauses = [.eventuallyWithin id trigger response ⟨bound, .steps⟩]) :
    (first.appendCorrelated second id trigger response bound shape).correlatedCoordinates trigger response =
      first.correlatedCoordinates trigger response ++ second.correlatedCoordinates trigger response := by
  simp [appendCorrelated, correlatedCoordinates, List.map_append]

/-- Existing bounded Property evaluation exposes its position quantification for the supported
aligned correlated fragment, without changing the existing evaluator or its closed-trace meaning. -/
theorem evaluatePropertyClause_correlated_positions
    (property : CheckedProperty) (input : CheckedPropertyEvaluationInput property)
    (clause : { clause // clause ∈ property.clauses })
    (id : DefinitionId) (trigger response : PropertyPattern) (bound : Nat)
    (shape : clause.val = .eventuallyWithin id trigger response ⟨bound, .steps⟩)
    (triggerAligned : trigger.field = .selectedAction)
    (responseAligned : response.field = .outcome ∨ response.field = .resultingState ∨
      response.field = .observation) :
    evaluatePropertyClause property input clause =
      (Property.Correlated.positions 1 ((input.correlatedCoordinates trigger response).map Prod.fst)).all
        (fun first => (Property.Correlated.positions 1 ((input.correlatedCoordinates trigger response).map Prod.snd)).any
          fun second => first ≤ second && second - first ≤ bound) := by
  simp only [evaluatePropertyClause, shape, evaluateResolvedPropertyClause, evaluateEventuallyWithin]
  rw [checkedPositions_semantic, checkedPositions_semantic]
  apply Bool.eq_iff_iff.mpr
  simp only [List.all_eq_true, List.any_eq_true]
  simp only [semantic_positions_mem trigger (Or.inl triggerAligned),
    semantic_positions_mem response (Or.inr responseAligned)]
  simp [CheckedPropertyEvaluationInput.correlatedCoordinates, List.map_map, Function.comp_def]

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
  | neverWithin _ trigger forbidden limit =>
      exact evaluateQuiescentWithin_agrees trigger forbidden limit input.view
  | branches group =>
      exact evaluateCaseGroup_agrees group input.view
  | guardedEventuallyWithin guarded =>
      exact evaluateGuardedTemporal_agrees false guarded input.view
  | guardedNeverWithin guarded =>
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
  effectiveGuards : List (CheckedPropertyPredicate .before)
  exceptions : List CheckedPropertyUnless
  guardMatched : Bool
  excluded : Bool
  applies : Bool
  clauses : List PropertyClauseIdentity
  deriving BEq, DecidableEq, Repr

/-- Evaluator-owned parent and case applicability at one validated trigger step. -/
structure BranchApplicability where
  propertyId : DefinitionId
  parentId : DefinitionId
  source : SourceLocation
  transitionPosition : Nat
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  effectiveGuards : List (CheckedPropertyPredicate .before)
  exceptions : List CheckedPropertyUnless
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
  | sameStep (expectation : CheckedPropertyPredicate .after)
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
  effectiveGuards : List (CheckedPropertyPredicate .before)
  exceptions : List CheckedPropertyUnless
  formula : JointObligationFormula
  satisfied : Bool
  deriving BEq, DecidableEq, Repr

private def sameStepClauseIdentity
    (group : CheckedPropertyBranches)
    (item : CheckedPropertyBranch)
    (clause : CheckedPropertySameStepClause) : PropertyClauseIdentity := {
  parentId := group.id
  caseId := some item.id
  clauseId := clause.id
  kind := .clause
  source := clause.source
  relatedDefinitionIds := item.exception.toList.map CheckedPropertyUnless.id
}

private def temporalClauseIdentity
    (clause : CheckedPropertyTemporalClause) : PropertyClauseIdentity := {
  parentId := clause.parentId
  caseId := clause.caseId
  clauseId := clause.id
  kind := .clause
  source := clause.source
  evaluatedLimit := some clause.limit
  relatedDefinitionIds :=
    clause.exception.toList.map CheckedPropertyUnless.id ++
      clause.caseException.toList.map CheckedPropertyUnless.id
}

/-- Exact same-step and temporal obligation identities declared by one checked named case. -/
def CheckedPropertyBranch.clauseIdentities
    (group : CheckedPropertyBranches)
    (item : CheckedPropertyBranch) : List PropertyClauseIdentity :=
  item.clauses.map (sameStepClauseIdentity group item) ++
    item.temporalClauses.map temporalClauseIdentity

private def caseApplicabilityAt
    (group : CheckedPropertyBranches)
    (parentApplies : Bool)
    (input : PropertyEvaluationPredicateInput)
    (item : CheckedPropertyBranch) : CaseApplicability :=
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
    (group : CheckedPropertyBranches)
    (transitionPosition : Nat)
    (step : PropertyEvaluationStep) : BranchApplicability :=
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
    (input : CheckedPropertyEvaluationInput property) : List BranchApplicability :=
  property.clauses.flatMap fun clause => match clause with
    | .branches group => input.view.steps.zipIdx.map fun (step, index) =>
        caseGroupApplicabilityAt property group (index + 1) step
    | _ => []

private def sameStepJointObservationsAt
    (property : CheckedProperty)
    (group : CheckedPropertyBranches)
    (transitionPosition : Nat)
    (step : PropertyEvaluationStep) : List JointObligationObservation :=
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
            coordinateUnit := .steps
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
    (clause : CheckedPropertyTemporalClause)
    (view : PropertyEvaluationView) : List JointObligationObservation :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => []
  | some responsePositions =>
      let rec visit
          (transitionPosition : Nat)
          (steps : List PropertyEvaluationStep) : List JointObligationObservation :=
        match steps with
        | [] => []
        | step :: rest =>
            let tail := visit (transitionPosition + 1) rest
            let triggerOccurrences :=
              stepOccurrences clause.trigger transitionPosition step
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
      visit 1 view.steps

/-- Observe jointly analyzable obligations only through the same checked input and evaluator used
for ordinary Property truth. Later trace steps cannot change trigger-time guards or exceptions. -/
def analyzeOverlapObligations
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property) : List JointObligationObservation :=
  property.clauses.flatMap fun clause => match clause with
    | .branches group =>
        (input.view.steps.zipIdx.flatMap fun (step, index) =>
          sameStepJointObservationsAt property group (index + 1) step) ++
        (group.cases.flatMap fun item =>
          item.temporalClauses.flatMap fun temporal =>
            guardedTemporalJointObservations property temporal input.view)
    | .guardedEventuallyWithin temporal | .guardedNeverWithin temporal =>
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

private def clausePatterns : CheckedPropertyClause → List PropertyPattern
  | .stateInvariant _ state => [state]
  | .transitionContract _ precondition postcondition => [precondition, postcondition]
  | .identityRelation _ relation => [relation]
  | .inputOutput _ input output => [input, output]
  | .ordered _ before after _ => [before, after]
  | .eventuallyWithin _ trigger response _ => [trigger, response]
  | .neverWithin _ trigger forbidden _ => [trigger, forbidden]
  | .branches group => group.cases.flatMap fun item =>
      item.temporalClauses.flatMap fun clause => [clause.trigger, clause.response]
  | .guardedEventuallyWithin guarded | .guardedNeverWithin guarded =>
      [guarded.trigger, guarded.response]

private def clauseLimit : CheckedPropertyClause → Option Limit
  | .eventuallyWithin _ _ _ limit | .neverWithin _ _ _ limit => some limit
  | .guardedEventuallyWithin guarded | .guardedNeverWithin guarded =>
      some guarded.limit
  | _ => none

private def predicateReferences : PropertyPredicate → List DefinitionId
  | .atom atom => match atom.fieldComparison with
    | none => [atom.reference]
    | some comparison => [comparison.left, comparison.right].filterMap fun operand => match operand with
      | .field path _ => some path.reference
      | .literal _ _ => none
  | .all items | .any items => items.flatMap predicateReferences
  | .not item => predicateReferences item

private def resolvedPredicateReferences
    (predicate : CheckedPropertyPredicate context) : List DefinitionId :=
  predicateReferences predicate.expression

private def caseGroupProvenance (group : CheckedPropertyBranches) : List DefinitionId :=
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
    (clause : CheckedPropertyTemporalClause) : List DefinitionId :=
  clause.id :: resolvedPredicateReferences clause.guard ++
    clause.exception.toList.flatMap fun exception =>
      exception.id :: resolvedPredicateReferences exception.condition

private def guardedTemporalFailures
    (clause : CheckedPropertyTemporalClause)
    (satisfied : Bool) : List PropertyClauseIdentity :=
  if satisfied then [] else [{
    parentId := clause.parentId
    caseId := clause.caseId
    clauseId := clause.id
    kind := .clause
    source := clause.source
    evaluatedLimit := some clause.limit
    relatedDefinitionIds :=
      clause.exception.toList.map CheckedPropertyUnless.id ++
        clause.caseException.toList.map CheckedPropertyUnless.id
  }]

private def caseGroupFailures
    (group : CheckedPropertyBranches)
    (view : PropertyEvaluationView) : List PropertyClauseIdentity :=
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
      relatedDefinitionIds := applicable.map CheckedPropertyBranch.id
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
            relatedDefinitionIds := item.exception.toList.map CheckedPropertyUnless.id
          }
    completeFailure ++ exclusiveFailure ++ clauseFailures
  let temporalFailures := group.cases.flatMap fun item =>
    item.temporalClauses.flatMap fun clause =>
      guardedTemporalFailures clause (evaluateGuardedTemporal clause.forbidden clause view)
  (stepFailures ++ temporalFailures).eraseDups

private def spanOf
    (clause : CheckedPropertyClause)
    (view : PropertyEvaluationView) : Option PropertyTraceSpan :=
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
    (view : PropertyEvaluationView)
    (clause : CheckedPropertyClause) : PropertyClauseResult :=
  let satisfied := evaluateResolvedPropertyClause clause view
  {
  propertyId := property.id
  clauseId := clause.id
  satisfied
  traceSpan := spanOf clause view
  evaluatedLimit := clauseLimit clause
  semanticProvenance := DefinitionId.canonicalSet
    (property.requires ++ (clausePatterns clause).map PropertyPattern.reference ++ match clause with
      | .branches group => caseGroupProvenance group
      | .guardedEventuallyWithin guarded | .guardedNeverWithin guarded =>
          guardedTemporalProvenance guarded
      | _ => [])
  failedObligations := match clause with
    | .branches group => caseGroupFailures group view
    | .guardedEventuallyWithin guarded | .guardedNeverWithin guarded =>
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

private def endpointCoordinate (unit : LimitUnit) (view : PropertyEvaluationView) : Option Nat :=
  match unit with
  | .steps | .actions => some view.steps.length
  | .logicalTime => view.steps.getLast?.bind (·.logicalTime)
  | _ => none

private def temporalEndpointAnswer
    (forbidden : Bool) (trigger : Nat) (responses : List Nat) (limit : Limit)
    (view : PropertyEvaluationView) : PropertyEndpointAnswer :=
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
    (view : PropertyEvaluationView) : PropertyEndpointAnswer :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggers, some responses => combineEndpointAnswers
      (triggers.map fun coordinate => temporalEndpointAnswer forbidden coordinate responses limit view)
  | _, _ => .unresolved

private def guardedEndpointAnswer
    (clause : CheckedPropertyTemporalClause)
    (view : PropertyEvaluationView) : PropertyEndpointAnswer :=
  match checkedPositions clause.response clause.limit.unit view with
  | none => .unresolved
  | some responses =>
      let rec visit (transitionPosition : Nat)
          (steps : List PropertyEvaluationStep) : List PropertyEndpointAnswer :=
        match steps with
        | [] => []
        | step :: rest =>
            let tail := visit (transitionPosition + 1) rest
            if guardedTemporalAppliesEvaluate clause (guardInput step) then
              match triggerPositionsInStep clause transitionPosition step with
              | none => .unresolved :: tail
              | some triggers => triggers.map (fun coordinate =>
                  temporalEndpointAnswer clause.forbidden coordinate responses clause.limit view) ++ tail
            else tail
      combineEndpointAnswers (visit 1 view.steps)

private def clauseEndpointAnswer
    (clause : CheckedPropertyClause)
    (view : PropertyEvaluationView) : PropertyEndpointAnswer :=
  match clause with
  | .stateInvariant _ pattern =>
      if !((valuesAtField .state view).any fun value => value.definitionId == pattern.reference) then
        .unresolved
      else if evaluateStateInvariant pattern view then .satisfied else .violated
  | .eventuallyWithin _ trigger response limit =>
      plainTemporalEndpointAnswer false trigger response limit view
  | .neverWithin _ trigger response limit =>
      plainTemporalEndpointAnswer true trigger response limit view
  | .guardedEventuallyWithin clause | .guardedNeverWithin clause =>
      guardedEndpointAnswer clause view
  | .branches group =>
      if !(view.steps.all (evaluateCaseGroupStep group)) then .violated else
        combineEndpointAnswers (group.cases.flatMap fun item =>
          item.temporalClauses.map fun clause => guardedEndpointAnswer clause view)
  | .ordered .. | .identityRelation .. =>
      if evaluateResolvedPropertyClause clause view then .satisfied else .unresolved
  | _ => if evaluateResolvedPropertyClause clause view then .satisfied else .violated

private def plainTriggerPattern : CheckedPropertyClause → Option PropertyPattern
  | .transitionContract _ trigger _ | .inputOutput _ trigger _ => some trigger
  | .eventuallyWithin _ trigger _ _ | .neverWithin _ trigger _ _ => some trigger
  | _ => none

private def requestedClauseTriggers : CheckedPropertyClause → List DefinitionId
  | .branches group => group.cases.flatMap fun item =>
      item.clauses.map (·.id) ++ item.temporalClauses.map (·.id)
  | .guardedEventuallyWithin clause | .guardedNeverWithin clause => [clause.id]
  | clause => if (plainTriggerPattern clause).isSome then [clause.id] else []

/-- Interpret the exact admitted view as closed or still open, preserving ordinary closed truth
and reporting conditional exercise separately from that truth. -/
def evaluatePropertyEndpoint
    (property : CheckedProperty)
    (input : CheckedPropertyEvaluationInput property)
    (partialTrace : Bool) : PropertyEndpointEvaluation :=
  let joint := (analyzeOverlapObligations property input).map fun observation => ({
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
          coordinateUnit := .steps
          coordinate := occurrence.transitionPosition
        }
      } : PropertyTriggerEvidence)
  {
    answer := if partialTrace then
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

import Umpire.Property.Correlated.Reference
import Umpire.Property.Correlated.Capture

/-!
Checked, bounded operation-correlated Property evaluation. `compile` binds the Property's scope and
key to an admitted semantic producer. `start` fixes an execution scope and Target initial state;
`consume` validates an entire labeled transition before ticking only its operation's obligations.
Every append is immutable and atomic. Whole-stream and offline consumers use `consumeMany`.
-/

namespace Umpire.Property.Correlated

variable {Law : Law → Prop} {Setup : Type}
variable {target : CheckedModel Law Setup ModelValue ModelValue ModelValue ModelValue}

/-- Evaluation limits bound retained transitions, independent triggers, retained capture values
and charged work. A consumer that declares no captures needs no capture budget. -/
structure Limits where
  transitions : Nat
  obligations : Nat
  work : Nat
  captures : Nat := 0
  deriving BEq, DecidableEq, Repr

/-- A rejected append returns no replacement state and never supplies a semantic answer. -/
inductive Error where
  | property (error : PropertyError)
  | unsupported (clauseId : DefinitionId) (reason : String)
  | invalidInitialState
  | invalidTransition (operation : String)
  | wrongScope
  | closed
  | transitionsExhausted
  | obligationsExhausted
  | workExhausted
  | capturesExhausted
  | capture (error : CaptureError)
  deriving BEq, DecidableEq, Repr

private def referenceClause (clause : CheckedPropertyCorrelatedClause) : CheckedPropertyClause :=
  .eventuallyWithin clause.declaration.id clause.triggerPattern clause.responsePattern
    ⟨clause.declaration.bound, .steps⟩

private def patternPredicate (field : PropertyPredicateField) (pattern : PropertyPattern) :
    Option PropertyPredicate := do
  let constraint ← match pattern.constraint with
    | .present => some PropertyAtomConstraint.present
    | .equals value => some (.equals (.text value))
    | _ => none
  pure (.atom ⟨field, pattern.reference, constraint⟩)

private def responsePredicateField (pattern : PropertyPattern) : PropertyPredicateField :=
  match pattern.field with
  | .resultingState => .resultingState
  | .observation => .expectationFact
  | _ => .outcome

private structure CompiledClause where
  original : CheckedPropertyCorrelatedClause
  reference : CheckedProperty
  shape : reference.clauses = [referenceClause original]
  typedTrigger : some original.trigger.expression = patternPredicate .selectedAction original.triggerPattern
  typedResponse : some original.response.expression =
    patternPredicate (responsePredicateField original.responsePattern) original.responsePattern
  triggerAligned : original.triggerPattern.field = .selectedAction
  responseAligned : original.responsePattern.field = .outcome ∨
    original.responsePattern.field = .resultingState ∨ original.responsePattern.field = .observation

/-- The compiled consumer retains the exact checked Property and producer scope binding. -/
structure Compiled (target : CheckedModel Law Setup ModelValue ModelValue ModelValue ModelValue) where
  private mk ::
  property : CheckedProperty
  private clauses : List CompiledClause
  scopeFields : List DefinitionId
  operationField : DefinitionId
  limits : Limits

private def closedPredicate (pattern : PropertyPattern) : Shared.CorrelatedObligation.Predicate := {
  field := match pattern.field with
    | .selectedAction => 1 | .outcome => 2 | .resultingState => 3 | .observation => 4
    | _ => 0
  reference := pattern.reference
  equalsText := match pattern.constraint with | .equals text => some text | _ => none }

private def closedClause (clause : CheckedPropertyCorrelatedClause) : Shared.CorrelatedObligation.Clause := {
  id := clause.declaration.id.value
  bound := clause.declaration.bound
  final := clause.declaration.ending == .final
  trigger := closedPredicate clause.triggerPattern
  response := closedPredicate clause.responsePattern }

/-- Checked source reference retained for portable compiler correspondence. -/
structure PortableReference where
  original : CheckedPropertyCorrelatedClause
  reference : CheckedProperty
  shape : reference.clauses = [referenceClause original]
  triggerAligned : original.triggerPattern.field = .selectedAction
  responseAligned : original.responsePattern.field = .outcome ∨
    original.responsePattern.field = .resultingState ∨ original.responsePattern.field = .observation

/-- Closed clause data derived from the checked supported predicate fragment. -/
def PortableReference.portable (binding : PortableReference) : Shared.CorrelatedObligation.Clause :=
  closedClause binding.original

/-- The exact checked eventual clause for this correlated reference. -/
def PortableReference.clause (binding : PortableReference) :
    { clause // clause ∈ binding.reference.clauses } :=
  ⟨referenceClause binding.original, by simp [binding.shape]⟩

/-- Inspect checked references without exposing the private compilation representation. -/
def Compiled.portableReferences (compiled : Compiled target) : List PortableReference :=
  compiled.clauses.map fun clause =>
    ⟨clause.original, clause.reference, clause.shape, clause.triggerAligned, clause.responseAligned⟩

/-- Closed clause data derived from the checked supported predicate fragment. -/
def Compiled.portableClauses (compiled : Compiled target) : List Shared.CorrelatedObligation.Clause :=
  compiled.portableReferences.map PortableReference.portable

/-- Reject a whole requested consumer when any clause or producer binding is unsupported. -/
def compile (target : CheckedModel Law Setup ModelValue ModelValue ModelValue ModelValue)
    (property : CheckedProperty) (scopeFields : List DefinitionId)
    (operationField : DefinitionId) (limits : Limits) : Except Error (Compiled target) := do
  if let some clause := property.clauses.head? then
    throw (.unsupported clause.id "mixed uncorrelated evaluation")
  if property.correlatedRules.isEmpty then
    throw (.unsupported property.id "no correlated clauses")
  let meanings := target.providers.flatMap (·.meanings)
  let mut clauses := []
  for clause in property.correlatedRules do
    if property.access.meanings.any (fun required => !meanings.contains required) ||
        property.access.capabilities.any (fun required => !target.providers.any (fun provider =>
          provider.contract.id == required.id && provider.contract.version == required.version &&
          provider.contract.behaviorVersion == required.behaviorVersion)) then
      throw (.unsupported clause.declaration.id "Target capability/meaning mismatch")
    if clause.declaration.scope != DefinitionId.canonicalSet scopeFields ||
        clause.declaration.key != operationField then
      throw (.unsupported clause.declaration.id "producer scope/key mismatch")
    let reference ← (Property.check (.ofTarget target) ({
      id := clause.declaration.id
      source := clause.declaration.source
      requires := property.requires
      clauses := [.eventuallyWithin clause.declaration.id clause.triggerPattern clause.responsePattern
        ⟨clause.declaration.bound, .steps⟩] })).mapError Error.property
    if shape : reference.clauses = [referenceClause clause] then
      if triggerAligned : clause.triggerPattern.field = .selectedAction then
        if responseAligned : clause.responsePattern.field = .outcome ∨
            clause.responsePattern.field = .resultingState ∨ clause.responsePattern.field = .observation then
          if typedTrigger : some clause.trigger.expression = patternPredicate .selectedAction clause.triggerPattern then
            if typedResponse : some clause.response.expression =
                patternPredicate (responsePredicateField clause.responsePattern) clause.responsePattern then
              clauses := clauses ++ [⟨clause, reference, shape, typedTrigger, typedResponse,
                triggerAligned, responseAligned⟩]
            else throw (.unsupported clause.declaration.id "typed response/pattern mismatch")
          else throw (.unsupported clause.declaration.id "typed trigger/pattern mismatch")
        else throw (.unsupported clause.declaration.id "unsupported response projection")
      else throw (.unsupported clause.declaration.id "unsupported trigger projection")
    else throw (.unsupported clause.declaration.id "unsupported reference clause")
  pure ⟨property, clauses, DefinitionId.canonicalSet scopeFields, operationField, limits⟩

/-- A successful operation retains its exact admitted coordinates and a kernel-checked fold
invariant. The transition ceiling bounds this history even when no trigger occurs. -/
structure Execution where
  private mk ::
  private compiled : CompiledClause
  input : CheckedPropertyEvaluationInput compiled.reference
  coordinates : List Coordinate
  obligations : List Obligation
  consistent : obligations = consumeMany compiled.original.declaration.bound [] coordinates
  projected : coordinates = (input.correlatedCoordinates compiled.original.triggerPattern
    compiled.original.responsePattern).map (fun point => Coordinate.mk point.1 point.2)

/-- The original checked correlated rule behind this execution. -/
abbrev Execution.clause (execution : Execution) : CheckedPropertyCorrelatedClause := execution.compiled.original

/-- Existing checked Property reference, admitted over exactly this execution's operation trace. -/
abbrev Execution.reference (execution : Execution) : CheckedProperty := execution.compiled.reference

private def Execution.start (clause : CompiledClause) (initial : ModelValue) : Except Error Execution := do
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := ⟨initial, []⟩
  let input ← (checkPropertyEvaluationInput clause.reference trace).mapError Error.property
  if projected : ([] : List Coordinate) = (input.correlatedCoordinates clause.original.triggerPattern
      clause.original.responsePattern).map (fun point => Coordinate.mk point.1 point.2) then
    pure ⟨clause, input, [], [], rfl, projected⟩
  else throw (.unsupported clause.original.declaration.id "invalid initial projection")

private def Execution.consume (execution : Execution)
    (priorState : ModelValue) (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) :
    Except Error (Execution × Bool) := do
  let nextInput ← (checkPropertyEvaluationInput execution.reference
    ⟨priorState, [step]⟩).mapError Error.property
  match shape : nextInput.correlatedCoordinates execution.clause.triggerPattern execution.clause.responsePattern with
  | [pair] =>
      let point := Coordinate.mk pair.1 pair.2
      let input := execution.input.appendCorrelated nextInput execution.clause.declaration.id
        execution.clause.triggerPattern execution.clause.responsePattern execution.clause.declaration.bound
        execution.compiled.shape
      let next : Execution := {
        compiled := execution.compiled
        input
        coordinates := execution.coordinates ++ [point]
        obligations := Correlated.consume execution.clause.declaration.bound execution.obligations point
        consistent := by
          change Shared.CorrelatedObligation.consume _ _ _ = Shared.CorrelatedObligation.consumeMany _ _ _
          rw [consumeMany_append]
          have consistent := execution.consistent
          change execution.obligations = Shared.CorrelatedObligation.consumeMany _ _ _ at consistent
          rw [← consistent]
          rfl
        projected := by
          change execution.coordinates ++ [point] =
            (input.correlatedCoordinates execution.clause.triggerPattern execution.clause.responsePattern).map
              (fun point => Coordinate.mk point.1 point.2)
          have appended := CheckedPropertyEvaluationInput.correlatedCoordinates_append
            execution.input nextInput execution.clause.declaration.id execution.clause.triggerPattern
            execution.clause.responsePattern execution.clause.declaration.bound execution.compiled.shape
          change execution.coordinates ++ [point] =
            ((execution.input.appendCorrelated nextInput execution.clause.declaration.id
              execution.clause.triggerPattern execution.clause.responsePattern
              execution.clause.declaration.bound execution.compiled.shape).correlatedCoordinates
                execution.clause.triggerPattern execution.clause.responsePattern).map
                  (fun point => Coordinate.mk point.1 point.2)
          rw [appended, List.map_append, shape]
          simpa only [List.map_cons, List.map_nil] using
            congrArg (fun points => points ++ [point]) execution.projected }
      pure (next, point.trigger)
  | _ => throw (.unsupported execution.clause.declaration.id "invalid step projection")

/-- Actual retained countdowns agree with independent windows over the admitted operation history. -/
theorem Execution.closed_reference (execution : Execution) :
    execution.obligations.all (fun obligation => decide (obligation = .satisfied)) =
      closedReference execution.clause.declaration.bound execution.coordinates := by
  rw [execution.consistent, closed_agrees]

/-- The exact eventual clause belonging to this execution's checked reference Property. -/
def Execution.referenceClause (execution : Execution) :
    { clause // clause ∈ execution.reference.clauses } :=
  ⟨Correlated.referenceClause execution.clause, by
    simp [Execution.reference, Execution.clause, execution.compiled.shape]⟩

/-- Successful runtime admission carries the complete bridge: countdowns, typed predicate
coordinates, the operation's actual trace projection and existing closed Property authority. -/
theorem Execution.closed_property (execution : Execution) :
    execution.obligations.all (fun obligation => decide (obligation = .satisfied)) =
      evaluatePropertyClause execution.reference execution.input execution.referenceClause := by
  rw [execution.consistent, execution.projected]
  exact checked_eventuallyWithin_agrees execution.reference execution.input execution.referenceClause
    execution.clause.declaration.id execution.clause.triggerPattern execution.clause.responsePattern
    execution.clause.declaration.bound rfl execution.compiled.triggerAligned execution.compiled.responseAligned

private structure Operation where
  key : String
  state : ModelValue
  executions : List Execution
  captures : Captures := Captures.empty

private structure Payload where
  scope : List (DefinitionId × String)
  initial : ModelValue
  operations : List Operation := []
  transitions : Nat := 0
  retainedObligations : Nat := 0
  capturedValues : Nat := 0
  work : Nat := 0
  closed : Bool := false

/-- Monitor state has no public constructor; replacement states come only from successful admission. -/
structure Monitor (compiled : Compiled target) where
  private mk ::
  private payload : Payload

/-- Start a fresh independent run only from a Target-admitted setup and initial state. -/
def Compiled.start [DecidableEq Setup] (compiled : Compiled target)
    (setup : Setup) (initial : ModelValue) (scope : List (DefinitionId × String)) :
    Except Error (Monitor compiled) := do
  if !(setup ∈ target.resolvedSetups) || !(target.machine.initialStates setup).contains initial then
    throw .invalidInitialState
  let scope := scope.mergeSort fun a b => decide (a.1.value ≤ b.1.value)
  if scope.map Prod.fst != compiled.scopeFields || scope.any (·.2.isEmpty) then
    throw .wrongScope
  pure ⟨{ scope, initial }⟩

/-- Raw model input is validated before any predicate or obligation can be evaluated. -/
structure Transition where
  scope : List (DefinitionId × String)
  operationField : DefinitionId
  operation : String
  priorState : ModelValue
  action : ModelValue
  result : Step ModelValue ModelValue ModelValue
  deriving BEq, DecidableEq, Repr

private def predicateInput (context : PropertyPredicateContext) (step : Transition) :
    PropertyPredicateInput := {
  context
  priorState := some step.priorState
  selectedAction := some step.action
  resultingState := some step.result.state
  outcome := some step.result.outcome
  facts := some step.result.facts
}

private def validateCoordinate (clause : CheckedPropertyCorrelatedClause) (step : Transition) :
    Except Error Unit := do
  let _ ← (checkPropertyPredicateInput clause.trigger (predicateInput .before step)).mapError
    Error.property
  let _ ← (checkPropertyPredicateInput clause.response (predicateInput .after step)).mapError
    Error.property
  pure ()

/-- A labeled transition is one of this operation's semantic steps only when the clause's declared
correlation holds over this step's evidence together with the operation's retained captures.
Reading an occurrence this operation never retained -- a future ordinal, or one belonging to a
different operation -- fails admission rather than binding the nearest match, and a correlation
that is false rejects the step instead of spending the operation's window. -/
private def validateCorrelation (clause : CheckedPropertyCorrelatedClause) (step : Transition)
    (evidence : List PropertyFieldEvidence) : Except Error Unit := do
  let some correlation := clause.correlation | pure ()
  let input ← (checkPropertyPredicateInputWithEvidence correlation
    (predicateInput .before step) evidence).mapError Error.property
  if !evaluatePropertyPredicate correlation input then
    throw (.invalidTransition step.operation)

/-- Validate transition authority and continuity before processing every clause atomically. No
poll, duplicate read, acknowledgement or unrelated operation can spend this operation's window. -/
def Monitor.consume {compiled : Compiled target} (run : Monitor compiled) (step : Transition)
    (evidence : List PropertyFieldEvidence := []) : Except Error (Monitor compiled) := do
  let payload := run.payload
  if payload.closed then throw .closed
  if step.scope != payload.scope || step.operationField != compiled.operationField ||
      step.operation.isEmpty then throw .wrongScope
  let operation ← match payload.operations.find? (·.key == step.operation) with
    | some operation => pure operation
    | none => do
        let executions ← compiled.clauses.mapM (fun clause => Execution.start clause payload.initial)
        pure { key := step.operation, state := payload.initial, executions }
  if step.priorState != operation.state ||
      !(target.machine.steps operation.state step.action).contains step.result then
    throw (.invalidTransition step.operation)
  if payload.transitions ≥ compiled.limits.transitions then throw .transitionsExhausted
  let declarations := compiled.clauses.flatMap (·.original.declaration.captures)
  let work := payload.work + payload.retainedObligations +
    16 * compiled.property.correlatedRules.length * (payload.transitions + 1) *
      (1 + target.behaviorTable.transitions.foldl (fun maximum row => max maximum row.facts.length) 0) +
      payload.operations.length +
    (target.machine.steps operation.state step.action).length +
    payload.capturedValues + (1 + declarations.length) * evidence.length
  if work > compiled.limits.work then throw .workExhausted
  -- Correlation reads this step's own evidence together with what the operation already retained,
  -- so an occurrence is bound only after an earlier step admitted it.
  let available := evidence ++ operation.captures.evidence
  let mut executions := []
  let mut created := 0
  for compiledClause in compiled.clauses do
    let clause := compiledClause.original
    validateCoordinate clause step
    validateCorrelation clause step available
    let current ← match operation.executions.find? (·.clause.declaration.id == clause.declaration.id) with
      | some current => pure current
      | none => Execution.start compiledClause payload.initial
    let (next, triggered) ← current.consume step.priorState (.result step.action step.result)
    if triggered then created := created + 1
    executions := executions ++ [next]
  if payload.retainedObligations + created > compiled.limits.obligations then
    throw .obligationsExhausted
  -- This step's own occurrences are retained only once the whole append was admitted, so no
  -- correlation can read the occurrence its own step creates.
  let (captures, charged) ← (operation.captures.record declarations evidence).mapError Error.capture
  if payload.capturedValues + charged > compiled.limits.captures then throw .capturesExhausted
  let next := { operation with state := step.result.state, executions, captures }
  let operations := if payload.operations.any (·.key == step.operation) then
    payload.operations.map fun current => if current.key == step.operation then next else current
    else payload.operations ++ [next]
  pure ⟨{ payload with
    operations
    transitions := payload.transitions + 1
    retainedObligations := payload.retainedObligations + created
    capturedValues := payload.capturedValues + charged
    work }⟩

/-- Whole-stream, incremental and offline evaluation share this exact checked transition fold. -/
def Monitor.consumeMany {compiled : Compiled target} (run : Monitor compiled)
    (steps : List Transition) : Except Error (Monitor compiled) :=
  steps.foldlM (fun run step => run.consume step) run

/-- Field-bearing streams use the exact same atomic append; each step carries the same-step
projections its clause operands read and its declared captures retain. -/
def Monitor.consumeEvidence {compiled : Compiled target} (run : Monitor compiled)
    (steps : List (Transition × List PropertyFieldEvidence)) : Except Error (Monitor compiled) :=
  steps.foldlM (fun run step => run.consume step.1 step.2) run

/-- The operation-correlated histories of successful admissions, with their checked fold invariants. -/
def Monitor.executions {compiled : Compiled target} (run : Monitor compiled) : List (String × Execution) :=
  run.payload.operations.flatMap fun operation =>
    operation.executions.map fun execution => (operation.key, execution)

/-- Inspect the current per-clause answer without closing or fabricating a transition. -/
def Monitor.answers {compiled : Compiled target} (run : Monitor compiled) (incomplete : Bool := false) :
    List (DefinitionId × PropertyEndpointAnswer) :=
  compiled.property.correlatedRules.map fun clause =>
    let obligations := run.payload.operations.flatMap fun operation =>
      (operation.executions.find? (·.clause.declaration.id == clause.declaration.id)).map (·.obligations) |>.getD []
    let ending := if incomplete || !run.payload.closed then
      TraceEnding.«partial» else clause.declaration.ending
    let answer := close ending obligations
    (clause.declaration.id, if incomplete && answer != .violated then .unresolved else answer)

/-- Closing freezes the Monitor; its ending policy distinguishes selected finite traces from prefixes. -/
def Monitor.close {compiled : Compiled target} (run : Monitor compiled) : Monitor compiled :=
  ⟨{ run.payload with closed := true }⟩

/-- Chunk boundaries preserve the exact successful state and the exact first rejection. -/
theorem Monitor.consumeMany_append {compiled : Compiled target} (run : Monitor compiled)
    (first second : List Transition) :
    run.consumeMany (first ++ second) =
      (run.consumeMany first >>= fun next => next.consumeMany second) := by
  simp [consumeMany, List.foldlM_append]

/-- Splitting a field-bearing stream at any chunk boundary retains the same captures, the same
obligations and the same first rejection. -/
theorem Monitor.consumeEvidence_append {compiled : Compiled target} (run : Monitor compiled)
    (first second : List (Transition × List PropertyFieldEvidence)) :
    run.consumeEvidence (first ++ second) =
      (run.consumeEvidence first >>= fun next => next.consumeEvidence second) := by
  simp [consumeEvidence, List.foldlM_append]

/-- Closing twice preserves the same immutable state. Finite-close answers may resolve previously
pending obligations; incomplete runtime prefixes retain their explicit unresolved interpretation. -/
theorem Monitor.close_idempotent {compiled : Compiled target} (run : Monitor compiled) :
    run.close.close = run.close := rfl

end Umpire.Property.Correlated

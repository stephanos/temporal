import Testpilot.Correlated
import Umpire.Case.CorrelatedProofs
import Umpire.Case.Compiler
import Umpire.Case.Projection.Correlated

/-!
Checked lowering of operation-correlated obligations into the closed Testpilot capability. The complete
Target table and admitted projection mappings are serialized; a checked decode equality binds the
actual portable interpreter to that exact shared executable table. Numeric narrowing is checked
before any protobuf constructor can truncate a value.

Keyed captures and correlations lower through the checked modeled-fields to declared-Observations
coverage map. The portable capability names a retained value by the declared evidence field the
projector emits, so every modeled operand a clause reads must have exactly one covered declared
field; an uncovered operand rejects the whole requested fragment before any Driver I/O. A clause
that declares neither captures nor a correlation needs no coverage and keeps its exact existing
encoding.
-/
namespace Umpire.Case.Correlated
open temporal.server.api.testpilot.v1 hiding ModelValue
open Umpire.Case.Projection

variable {Law : Law → Prop} {Setup : Type}
variable {target : CheckedModel Law Setup ModelValue ModelValue ModelValue ModelValue}

private def number (value : Nat) : Except String Int64 :=
  if value ≤ 9223372036854775807 then .ok (Int64.ofInt value) else .error "protobuf signed overflow"
private def atom (value : ModelValue) : temporal.server.api.testpilot.v1.ModelValue :=
  { definition_id := value.definitionId.value, value := value.value }
/-- Each state paired with the fields the Model declares it holds. The Model's own plan carries a
state as the value it tells states apart by; the Contract compares a machine's fields apart, so the
fields are attached here, at the one boundary that has both. -/
abbrev StateFields := List (ModelValue × List ModelValue)

private def fieldsOf (stateFields : StateFields) (state : ModelValue) : List ModelValue :=
  (stateFields.lookup state).getD []

private def lifted (stateFields : StateFields) (state : ModelValue) :
    Shared.SemanticData.StateValue :=
  ⟨state, fieldsOf stateFields state⟩

private def liftedStep (stateFields : StateFields) (step : Step ModelValue ModelValue ModelValue) :
    Testpilot.Correlated.Result :=
  ⟨step.outcome, lifted stateFields step.state, step.facts⟩

private def liftedRule (stateFields : StateFields)
    (rule : Shared.CorrelatedProjection.Rule DefinitionId ModelValue
      (Step ModelValue ModelValue ModelValue)) :
    Shared.CorrelatedProjection.Rule DefinitionId ModelValue Testpilot.Correlated.Result := {
  kind := rule.kind
  fieldCount := rule.fieldCount
  meaning := match rule.meaning with
    | .irrelevant => .irrelevant
    | .submission action => .submission action
    | .confirmed submission steps => .confirmed submission
        (steps.map fun (action, step) => (action, liftedStep stateFields step)) }

private def liftedRows (stateFields : StateFields)
    (rows : List (ModelValue × ModelValue × Step ModelValue ModelValue ModelValue)) :
    List Shared.CorrelatedObligation.Transition :=
  rows.map fun (prior, action, step) =>
    (lifted stateFields prior, action, liftedStep stateFields step)

private def liftedPlan (stateFields : StateFields)
    (plan : Shared.CorrelatedProjection.Plan DefinitionId ModelValue ModelValue
      (Step ModelValue ModelValue ModelValue)) : Testpilot.Correlated.Plan := {
  initial := lifted stateFields plan.initial
  rules := plan.rules.map (liftedRule stateFields)
  transitions := liftedRows stateFields plan.transitions
  limits := plan.limits }

private def output (stateFields : StateFields) (action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) : CorrelatedTransition := {
  action := some (atom action)
  state := some (atom result.state)
  outcome := some (atom result.outcome)
  facts := result.facts.toArray.map atom
  state_fields := (fieldsOf stateFields result.state).toArray.map atom }
private def fieldPolicy (field : EvidenceFieldDeclaration × FieldDisposition) :
    Except String CorrelatedFieldPolicy := do
  let disposition ← match field.2 with
    | .retain => pure CorrelatedFieldDisposition.CORRELATED_FIELD_DISPOSITION_RETAIN
    | .redact => pure .CORRELATED_FIELD_DISPOSITION_REDACT
    | .reject => pure .CORRELATED_FIELD_DISPOSITION_REJECT
    | .hash _ => throw "unsupported field disposition"
  pure {
    field_id := field.1.id.value
    type := some { kind := match field.1.valueType with
      | .text => .SCALAR_KIND_TEXT
      | .natural => .SCALAR_KIND_UINT64
      | .boolean => .SCALAR_KIND_BOOLEAN }
    disposition }
/-- The step condition a pattern lowers to: its correlated step reference tested for presence, or
compared equal with the text it requires. -/
private def stepCondition (field : CorrelatedStepField) (definitionId : DefinitionId)
    (equalsText : Option String) : Expression :=
  let step := Testpilot.Authoring.Expr.correlatedStep field definitionId.value
  match equalsText with
  | none => Testpilot.Authoring.Expr.present step
  | some text => Testpilot.Authoring.Expr.equal step
      (Testpilot.Authoring.Expr.literal (Testpilot.Authoring.Value.text text))

private def pattern (value : PropertyPattern) : Except String Expression := do
  let field ← match value.field with
    | .selectedAction => pure CorrelatedStepField.CORRELATED_STEP_FIELD_ACTION
    | .outcome => pure .CORRELATED_STEP_FIELD_OUTCOME
    | .resultingState => pure .CORRELATED_STEP_FIELD_STATE
    | .observation => pure .CORRELATED_STEP_FIELD_FACT
    | _ => throw "unsupported predicate projection"
  match value.constraint with
  | .present => pure (stepCondition field value.reference none)
  | .equals text => pure (stepCondition field value.reference (some text))
  | _ => throw "unsupported predicate constraint"

/-- The exact executable meaning the emitted capability must decode to. `keyed` is the keyed
fragment this lowering produced from the checked coverage, so a Case that declares no captures
carries the same empty fragment, the same unset capture ceiling and the same bytes as before. -/
private def meaning (stateFields : StateFields) (plan : Projection.Checked target)
    (compiled : Property.Correlated.Compiled target)
    (keyed : List (String × Testpilot.Correlated.Keyed)) : Testpilot.Correlated.Compiled := {
  plan := liftedPlan stateFields plan.executable
  clauses := compiled.portableClauses
  keyed
  scopeFields := compiled.scopeFields
  operationField := compiled.operationField
  sources := plan.sourceDeclaration.sources
  policies := plan.fieldPolicies
  transitions := compiled.limits.transitions
  obligations := compiled.limits.obligations
  work := compiled.limits.work
  captures := if keyed.any (fun entry => !entry.2.captures.isEmpty) then compiled.limits.captures
    else 0 }

/-- One declared capture, named by the declared evidence field its covered coordinates map onto. -/
private def capture (coverage : Projection.Coverage plan) (declaration : PropertyCorrelatedCapture) :
    Except String (CorrelatedCaptureDeclaration × Testpilot.Correlated.Capture) := do
  let some entry := coverage.entryOf? declaration.path
    | throw ("captured coordinates have no declared Observation for " ++ declaration.name.value)
  let lifetime ← number declaration.lifetime
  let wire : CorrelatedCaptureDeclaration :=
    { capture_id := declaration.name.value, field_id := entry.field.value, lifetime }
  pure (wire, ⟨declaration.name, entry.field, entry.scalarKind, declaration.lifetime⟩)

/-- The exact scalar a modeled literal denotes in the portable evidence domain. Only the three kinds
a projected Observation can carry are lowerable; every other exact model scalar rejects here rather
than being narrowed into a value the runtime would compare differently. -/
private def literalOperand (value : Operation.Scalar) :
    Except String (temporal.server.api.testpilot.v1.Value × Shared.SemanticData.Scalar × Nat) :=
  match value with
  | .text text => .ok ({ value := some (.text_value text) }, .text text, 1)
  | .boolean flag => .ok ({ value := some (.bool_value flag) }, .boolean flag, 3)
  | .integer _ number =>
      if number < 0 then .error "unsupported negative correlation literal"
      else if number > 18446744073709551615 then .error "protobuf unsigned overflow"
      else .ok ({ value := some (.unsigned_integer_value (toString number.toNat)) },
        .natural number.toNat, 2)
  | _ => .error "unsupported correlation literal"

/-- One correlation operand, with the declared scalar kind the portable decoder checks it against. -/
private def operand (coverage : Projection.Coverage plan)
    (captures : List Testpilot.Correlated.Capture) (value : PropertyFieldOperand) :
    Except String (Expression × Testpilot.Correlated.Operand × Nat) :=
  match value with
  | .literal scalar _ => do
      let (wire, decoded, kind) ← literalOperand scalar
      pure (Testpilot.Authoring.Expr.literal wire, .literal decoded, kind)
  | .field path _ =>
      match path.capture with
      | some key => do
          let some declaration := captures.find? (·.id == key.name)
            | throw ("unbound capture reference " ++ key.name.value)
          if key.ordinal ≥ declaration.lifetime then
            throw ("capture ordinal beyond declared lifetime for " ++ key.name.value)
          let ordinal ← number key.ordinal
          pure (Testpilot.Authoring.Expr.correlatedCapture key.name.value ordinal,
            .capture key.name key.ordinal, declaration.kind)
      | none => do
          let some entry := coverage.entryOf? path
            | throw ("correlation operand coordinates have no declared Observation for " ++
                path.reference.value)
          pure (Testpilot.Authoring.Expr.evidenceField entry.field.value, .field entry.field,
            entry.scalarKind)

private def predicateField : PropertyPredicateField → Except String CorrelatedStepField
  | .selectedAction => .ok .CORRELATED_STEP_FIELD_ACTION
  | .outcome => .ok .CORRELATED_STEP_FIELD_OUTCOME
  | .resultingState => .ok .CORRELATED_STEP_FIELD_STATE
  | .expectationFact => .ok .CORRELATED_STEP_FIELD_FACT
  | .priorState => .error "unsupported correlation predicate field"

private def predicateCode : PropertyPredicateField → Nat
  | .selectedAction => 1
  | .outcome => 2
  | .resultingState => 3
  | .expectationFact => 4
  | .priorState => 0

mutual
/-- Lower one correlation node, returning its wire form, its exact decoded meaning and the ceiling
its own nesting requires. The depth ceiling emitted for the capability is the exact depth this
correlation needs, so an exhausted depth is never a silently truncated condition. -/
private def correlation (coverage : Projection.Coverage plan)
    (captures : List Testpilot.Correlated.Capture) :
    PropertyPredicate → Except String (Expression × Testpilot.Correlated.Correlation × Nat)
  | .atom value =>
      match value.constraint with
      | .fields comparison => do
          let equal ← match comparison.operator with
            | .equal => pure true
            | .notEqual => pure false
            | _ => throw "unsupported correlation comparison operator"
          let (leftWire, left, leftKind) ← operand coverage captures comparison.left
          let (rightWire, right, rightKind) ← operand coverage captures comparison.right
          if leftKind != rightKind then throw "incompatible correlation operand types"
          let wire := Testpilot.Authoring.Expr.compare
            (if equal then .COMPARISON_OPERATOR_EQUAL else .COMPARISON_OPERATOR_NOT_EQUAL)
            leftWire rightWire
          pure (wire, .comparison equal left right, 1)
      | .present => do
          let field ← predicateField value.field
          pure (stepCondition field value.reference none,
            .predicate ⟨predicateCode value.field, value.reference, none⟩, 1)
      | .equals (.text text) => do
          let field ← predicateField value.field
          pure (stepCondition field value.reference (some text),
            .predicate ⟨predicateCode value.field, value.reference, some text⟩, 1)
      | _ => throw "unsupported correlation constraint"
  | .all items => do
      if items.isEmpty then throw "empty correlation group"
      let (wires, decoded, depth) ← correlations coverage captures items
      pure (Testpilot.Authoring.Expr.all wires, .all decoded, depth + 1)
  | .any items => do
      if items.isEmpty then throw "empty correlation group"
      let (wires, decoded, depth) ← correlations coverage captures items
      pure (Testpilot.Authoring.Expr.any wires, .any decoded, depth + 1)
  | .not _ => throw "unsupported correlation negation"
  termination_by expression => sizeOf expression

private def correlations (coverage : Projection.Coverage plan)
    (captures : List Testpilot.Correlated.Capture) :
    List PropertyPredicate →
      Except String (Array Expression × Testpilot.Correlated.Correlations × Nat)
  | [] => pure (#[], .nil, 0)
  | head :: rest => do
      let (headWire, headDecoded, headDepth) ← correlation coverage captures head
      let (wires, decoded, depth) ← correlations coverage captures rest
      pure (#[headWire] ++ wires, .cons headDecoded decoded, max headDepth depth)
  termination_by items => sizeOf items
end

/-- One lowered clause: its wire form, its keyed fragment and the correlation depth it needs. -/
private structure LoweredClause where
  wire : CorrelatedRule
  keyed : String × Testpilot.Correlated.Keyed
  depth : Nat

private def clause (coverage : Projection.Coverage plan)
    (source : CheckedPropertyCorrelatedClause) : Except String LoweredClause := do
  let captures ← source.declaration.captures.mapM (capture coverage)
  let declared := captures.map Prod.snd
  let correlated ← source.correlation.mapM fun checked =>
    correlation coverage declared checked.expression
  pure {
    wire := Testpilot.Authoring.Contract.correlatedRule source.declaration.id.value
      (← number source.declaration.bound)
      (match source.declaration.ending with
        | .«partial» => .TRACE_ENDING_PARTIAL
        | .final => .TRACE_ENDING_FINAL)
      (← pattern source.triggerPattern) (← pattern source.responsePattern)
      (captures.map Prod.fst).toArray (correlated.map (·.1))
    keyed := (source.declaration.id.value, ⟨declared, correlated.map (·.2.1)⟩)
    depth := (correlated.map (·.2.2)).getD 0 }

/-- Successful lowering carries equality of the data actually decoded for portable execution. `limits`
are the correlated ceilings the checked declarations need; the wire carries none, so they are the
Profile ceilings the capability is decoded under. -/
structure Lowered (plan : Projection.Checked target) (compiled : Property.Correlated.Compiled target) where
  wire : CorrelatedContract
  limits : CorrelatedLimits
  decoded : Testpilot.Correlated.Compiled
  keyed : List (String × Testpilot.Correlated.Keyed)
  stateFields : StateFields
  decoding : Testpilot.Correlated.decode limits wire = .ok decoded
  meaning : decoded = Correlated.meaning stateFields plan compiled keyed
  maximumFacts : plan.executable.transitions.foldl (fun maximum row => max maximum row.2.2.facts.length) 0 =
    target.behaviorTable.transitions.foldl (fun maximum row => max maximum row.facts.length) 0
  candidateCounts : ∀ row ∈ plan.executable.transitions,
    (plan.executable.transitions.filter (fun candidate => candidate.1 == row.1 && candidate.2.1 == row.2.1)).length =
      (target.machine.steps row.1 row.2.1).length
  certificates : ∀ binding ∈ compiled.portableReferences,
    CorrelatedProofs.Certificate binding (liftedRows stateFields plan.executable.transitions)
      plan.initialState

/-- Lower checked correlated declarations and projection together; failure rejects the entire fragment. -/
def lower (plan : Projection.Checked target) (compiled : Property.Correlated.Compiled target)
    (evidenceObservationId : String)
    (coverage : Projection.Coverage plan := Projection.Coverage.empty plan)
    (stateFields : StateFields := []) :
    Except Compiler.Error (Lowered plan compiled) := do
  let failed := fun reason => Compiler.Error.mk compiled.property.id.value
    compiled.property.source reason
  let lowered ← (compiled.property.correlatedRules.mapM (clause coverage)).mapError failed
  let keyed := lowered.map (·.keyed)
  let build : Except String (CorrelatedContract × CorrelatedLimits) := do
    let declaration := plan.sourceDeclaration
    let rules ← declaration.rules.mapM fun rule => do
      let fields ← rule.fields.mapM fieldPolicy
      let (meaning, submission, outputs) := match rule.meaning with
        | .irrelevant => (CorrelatedEvidenceMeaning.CORRELATED_EVIDENCE_MEANING_IRRELEVANT, none, [])
        | .submission action => (.CORRELATED_EVIDENCE_MEANING_SUBMISSION, some (atom action), [])
        | .confirmed required steps => (.CORRELATED_EVIDENCE_MEANING_CONFIRMED,
            required.map atom,
            steps.map fun (action, result) => output stateFields action result)
      pure (CorrelatedProjectionRule.mk rule.kind.value meaning submission outputs.toArray fields.toArray default)
    -- A capability that declares neither captures nor a correlation leaves both ceilings unset, so
    -- its meaning is exactly the one it had before the keyed capability existed.
    let declaresCaptures := keyed.any fun entry => !entry.2.captures.isEmpty
    let limits : CorrelatedLimits := {
      max_events := ← number declaration.limits.events
      max_buffered := ← number declaration.limits.buffered
      max_keys := ← number declaration.limits.keys
      max_support := ← number declaration.limits.support
      max_projection_work := ← number declaration.limits.work
      max_event_bytes := ← number declaration.limits.eventSize
      max_semantic_transitions := ← number compiled.limits.transitions
      max_obligations := ← number compiled.limits.obligations
      max_obligation_work := ← number compiled.limits.work
      max_captures := ← number (if declaresCaptures then compiled.limits.captures else 0)
      max_correlation_depth := ← number (lowered.foldl (fun ceiling row => max ceiling row.depth) 0) }
    pure (Testpilot.Authoring.Contract.correlated declaration.id.value
      plan.behaviorFingerprint.render evidenceObservationId declaration.operationField.value
      (declaration.scopeFields.toArray.map DefinitionId.value)
      (declaration.sources.toArray.map DefinitionId.value)
      (atom plan.initialState)
      (plan.executable.transitions.toArray.map fun (prior, action, result) =>
        { output stateFields action result with
          prior_state := some (atom prior)
          prior_fields := (fieldsOf stateFields prior).toArray.map atom })
      rules.toArray (lowered.map (·.wire)).toArray
      ((fieldsOf stateFields plan.initialState).toArray.map atom), limits)
  let (wire, limits) ← build.mapError failed
  match decoding : Testpilot.Correlated.decode limits wire with
  | .error reason => throw (failed reason)
  | .ok decoded =>
      if agreement : decoded = meaning stateFields plan compiled keyed then
        let certificates ← (CorrelatedProofs.certifyAll compiled.portableReferences
          (liftedRows stateFields plan.executable.transitions) plan.initialState).mapError failed
        if maximumFacts : plan.executable.transitions.foldl
            (fun maximum row => max maximum row.2.2.facts.length) 0 =
            target.behaviorTable.transitions.foldl (fun maximum row => max maximum row.facts.length) 0 then
          if candidateCounts : ∀ row ∈ plan.executable.transitions,
              (plan.executable.transitions.filter (fun candidate =>
                candidate.1 == row.1 && candidate.2.1 == row.2.1)).length =
                  (target.machine.steps row.1 row.2.1).length then
            pure ⟨wire, limits, decoded, keyed, stateFields, decoding, agreement, maximumFacts,
              candidateCounts, certificates.down⟩
          else throw (failed "portable work candidate multiplicity differs from checked kernel")
        else throw (failed "portable work maximum fact count differs from checked description")
      else throw (failed "portable correlated meaning roundtrip mismatch")

/-- Carry the checked property/source mapping in opaque Umpire provenance, not executable metadata. -/
def Lowered.contractLowering {plan : Projection.Checked target} {compiled : Property.Correlated.Compiled target} (lowered : Lowered plan compiled) : Compiler.ContractLowering :=
  .correlated ⟨compiled.property.id.value, compiled.property.behaviorFingerprint.render, .property⟩
    lowered.wire (compiled.property.correlatedRules.map fun clause => {
      ruleId := clause.declaration.id.value
      propertyId := compiled.property.id.value
      propertyFingerprint := compiled.property.behaviorFingerprint.render
      projectionId := plan.sourceDeclaration.id.value
      projectionFingerprint := plan.behaviorFingerprint.render
      source := clause.declaration.source })

/-- Any row executable by the actual decoded Contract is authorized by the original Target. -/
theorem Lowered.table_authorized {plan : Projection.Checked target} {compiled : Property.Correlated.Compiled target} (lowered : Lowered plan compiled)
    (row) (member : row ∈ lowered.decoded.plan.transitions) :
    target.machine.authoritativeStep row.1.atom row.2.1
      ⟨row.2.2.outcome, row.2.2.state.atom, row.2.2.facts⟩ := by
  rw [lowered.meaning] at member
  -- A lifted row is its own row with its states' fields attached, so the Model step it stands for
  -- is the one the checked plan authorized.
  obtain ⟨original, original_member, same⟩ := List.mem_map.mp member
  subst same
  exact plan.table_authorized original original_member

/-- Each actual decoded window selects its checked source clause through the lowering certificate.
The witness is built from the exact retained rows, never from an independently supplied truth value. -/
theorem Lowered.window_property {plan : Projection.Checked target}
    {compiled : Property.Correlated.Compiled target} (lowered : Lowered plan compiled)
    {history : List (Shared.CorrelatedObligation.Admitted lowered.decoded.plan.transitions)}
    (window : Shared.CorrelatedObligation.Window lowered.decoded.clauses history) :
    ∃ binding ∈ compiled.portableReferences,
      ∃ input : CheckedPropertyEvaluationInput binding.reference,
        CorrelatedProofs.HistoryInput binding plan.initialState (history.map (·.value)) input ∧
        window.obligations.all (fun obligation => decide (obligation = .satisfied)) =
          evaluatePropertyClause binding.reference input binding.clause := by
  have member := window.clause.property
  have clauses : lowered.decoded.clauses = compiled.portableClauses := congrArg (·.clauses) lowered.meaning
  have sourceMember := Eq.mp (congrArg (fun cs => window.clause.val ∈ cs) clauses) member
  obtain ⟨binding, member, same⟩ := List.mem_map.mp sourceMember
  have certificate := lowered.certificates binding member
  have table : lowered.decoded.plan.transitions =
      liftedRows lowered.stateFields plan.executable.transitions := by rw [lowered.meaning]; rfl
  have checked : CorrelatedProofs.Certificate binding lowered.decoded.plan.transitions plan.initialState := by
    rw [table]; exact certificate
  obtain ⟨input, admitted, answer⟩ := checked.closed_property window same.symm
  exact ⟨binding, member, input, admitted, answer⟩

/-- Actual decoded scope/source/field admission is the checked projector's admission boundary. -/
theorem Lowered.evidence_validation {plan : Projection.Checked target}
    {compiled : Property.Correlated.Compiled target} (lowered : Lowered plan compiled)
    (scope : List (DefinitionId × String)) (event : Projection.Event) :
    lowered.decoded.validateEvent scope event =
      (plan.validateEvent scope event).mapError (fun _ => "invalid evidence") := by
  rw [lowered.meaning]
  unfold Testpilot.Correlated.Compiled.validateEvent Projection.Checked.validateEvent
  simp only [Correlated.meaning, liftedPlan, plan.executable_limits]
  rfl

/-- Every window returned by actual wire observation admission retains checked Property correspondence. -/
theorem Lowered.observed_property {plan : Projection.Checked target}
    {compiled : Property.Correlated.Compiled target} (lowered : Lowered plan compiled)
    (before after : Testpilot.Correlated.Monitor lowered.decoded) (sequence : Nat)
    (wire : CorrelatedEvidence) (_admitted : before.observe sequence wire = .ok after)
    (operation : Shared.CorrelatedObligation.Operation lowered.decoded.plan.transitions lowered.decoded.clauses)
    (_operation : operation ∈ after.monitor.operations)
    (window : Shared.CorrelatedObligation.Window lowered.decoded.clauses operation.history)
    (_window : window ∈ operation.windows) :
    ∃ binding ∈ compiled.portableReferences,
      ∃ input : CheckedPropertyEvaluationInput binding.reference,
        CorrelatedProofs.HistoryInput binding plan.initialState (operation.history.map (·.value)) input ∧
        window.obligations.all (fun obligation => decide (obligation = .satisfied)) =
          evaluatePropertyClause binding.reference input binding.clause :=
  lowered.window_property window

end Umpire.Case.Correlated

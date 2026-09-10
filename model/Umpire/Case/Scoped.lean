import Testpilot.Scoped
import Umpire.Case.ScopedProofs
import Umpire.Case.Compiler
import Umpire.Observation.Evaluation.Scoped

/-!
Checked lowering of operation-scoped obligations into the closed Testpilot capability. The complete
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
namespace Umpire.Case.Scoped
open temporal.server.api.testpilot.v1
open Umpire.Observation

variable {Law : Law → Prop} {Setup : Type}
variable {target : CheckedModel Law Setup ModelValue ModelValue ModelValue ModelValue}

private def number (value : Nat) : Except String Int64 :=
  if value ≤ 9223372036854775807 then .ok (Int64.ofInt value) else .error "protobuf signed overflow"
private def atom (value : ModelValue) : ScopedValue :=
  { definition_id := value.definitionId.value, value := value.value }
private def output (action : ModelValue) (result : Step ModelValue ModelValue ModelValue) :
    ScopedTransition := {
  action := some (atom action)
  resulting_state := some (atom result.state)
  outcome := some (atom result.outcome)
  facts := result.facts.toArray.map atom }
private def fieldPolicy (field : EvidenceFieldDeclaration × FieldDisposition) :
    Except String ScopedFieldPolicy := do
  let disposition ← match field.2 with
    | .retain => pure ScopedFieldDisposition.SCOPED_FIELD_DISPOSITION_RETAIN
    | .redact => pure .SCOPED_FIELD_DISPOSITION_REDACT
    | .reject => pure .SCOPED_FIELD_DISPOSITION_REJECT
    | .hash _ => throw "unsupported field disposition"
  pure {
    field_id := field.1.id.value
    type := some { kind := match field.1.valueType with
      | .text => .SCALAR_KIND_TEXT
      | .natural => .SCALAR_KIND_NATURAL
      | .boolean => .SCALAR_KIND_BOOLEAN }
    disposition }
private def pattern (value : PropertyPattern) : Except String ScopedPredicate := do
  let field ← match value.field with
    | .selectedAction => pure ScopedPredicateField.SCOPED_PREDICATE_FIELD_ACTION
    | .modelOutcome => pure .SCOPED_PREDICATE_FIELD_OUTCOME
    | .resultingState => pure .SCOPED_PREDICATE_FIELD_RESULTING_STATE
    | .observation => pure .SCOPED_PREDICATE_FIELD_FACT
    | _ => throw "unsupported predicate projection"
  let predicate : ScopedPredicate := { field, definition_id := value.reference.value }
  match value.constraint with
  | .present => pure { predicate with constraint := some (.present true) }
  | .equals text => pure { predicate with constraint := some (.equals_text text) }
  | _ => throw "unsupported predicate constraint"

/-- The exact executable meaning the emitted capability must decode to. `keyed` is the keyed
fragment this lowering produced from the checked coverage, so a Case that declares no captures
carries the same empty fragment, the same unset capture ceiling and the same bytes as before. -/
private def meaning (plan : Projection.Checked target) (compiled : Property.Scoped.Compiled target)
    (keyed : List (String × Testpilot.Scoped.Keyed)) : Testpilot.Scoped.Compiled := {
  plan := plan.executable
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
private def capture (coverage : Projection.Coverage plan) (declaration : PropertyScopedCapture) :
    Except String (ScopedCaptureDeclaration × Testpilot.Scoped.Capture) := do
  let some entry := coverage.entryOf? declaration.path
    | throw ("captured coordinates have no declared Observation for " ++ declaration.name.value)
  let lifetime ← number declaration.lifetime
  let wire : ScopedCaptureDeclaration :=
    { capture_id := declaration.name.value, field_id := entry.field.value, lifetime }
  pure (wire, ⟨declaration.name, entry.field, entry.scalarKind, declaration.lifetime⟩)

/-- The exact scalar a modeled literal denotes in the portable evidence domain. Only the three kinds
a projected Observation can carry are lowerable; every other exact model scalar rejects here rather
than being narrowed into a value the runtime would compare differently. -/
private def literalOperand (value : Operation.Scalar) :
    Except String (temporal.server.api.testpilot.v1.Value × Shared.SemanticData.Scalar × Nat) :=
  match value with
  | .text text => .ok ({ value := some (.text text) }, .text text, 1)
  | .boolean flag => .ok ({ value := some (.bool_value flag) }, .boolean flag, 3)
  | .integer _ number =>
      if number ≥ 0 then
        .ok ({ value := some (.natural (toString number.toNat)) }, .natural number.toNat, 2)
      else .error "unsupported negative correlation literal"
  | _ => .error "unsupported correlation literal"

/-- One correlation operand, with the declared scalar kind the portable decoder checks it against. -/
private def operand (coverage : Projection.Coverage plan)
    (captures : List Testpilot.Scoped.Capture) (value : PropertyFieldOperand) :
    Except String (ScopedOperand × Testpilot.Scoped.Operand × Nat) :=
  match value with
  | .literal scalar _ => do
      let (wire, decoded, kind) ← literalOperand scalar
      pure ({ operand := some (.literal wire) }, .literal decoded, kind)
  | .field path _ =>
      match path.capture with
      | some key => do
          let some declaration := captures.find? (·.id == key.name)
            | throw ("unbound capture reference " ++ key.name.value)
          if key.ordinal ≥ declaration.lifetime then
            throw ("capture ordinal beyond declared lifetime for " ++ key.name.value)
          let ordinal ← number key.ordinal
          let reference : ScopedCaptureRef := { capture_id := key.name.value, ordinal }
          pure ({ operand := some (.capture reference) }, .capture key.name key.ordinal,
            declaration.kind)
      | none => do
          let some entry := coverage.entryOf? path
            | throw ("correlation operand coordinates have no declared Observation for " ++
                path.reference.value)
          pure ({ operand := some (.field_id entry.field.value) }, .field entry.field,
            entry.scalarKind)

private def predicateField : PropertyPredicateField → Except String ScopedPredicateField
  | .selectedAction => .ok .SCOPED_PREDICATE_FIELD_ACTION
  | .modelOutcome => .ok .SCOPED_PREDICATE_FIELD_OUTCOME
  | .resultingState => .ok .SCOPED_PREDICATE_FIELD_RESULTING_STATE
  | .expectationFact => .ok .SCOPED_PREDICATE_FIELD_FACT
  | .priorState => .error "unsupported correlation predicate field"

private def predicateCode : PropertyPredicateField → Nat
  | .selectedAction => 1
  | .modelOutcome => 2
  | .resultingState => 3
  | .expectationFact => 4
  | .priorState => 0

mutual
/-- Lower one correlation node, returning its wire form, its exact decoded meaning and the ceiling
its own nesting requires. The depth ceiling emitted for the capability is the exact depth this
correlation needs, so an exhausted depth is never a silently truncated condition. -/
private def correlation (coverage : Projection.Coverage plan)
    (captures : List Testpilot.Scoped.Capture) :
    PropertyPredicate → Except String (ScopedCorrelation × Testpilot.Scoped.Correlation × Nat)
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
          let wire : ScopedComparison := {
            operator := if equal then .SCOPED_COMPARISON_OPERATOR_EQUAL
              else .SCOPED_COMPARISON_OPERATOR_NOT_EQUAL
            left := some leftWire, right := some rightWire }
          pure ({ condition := some (.comparison wire) }, .comparison equal left right, 1)
      | .present => do
          let field ← predicateField value.field
          let wire : ScopedPredicate :=
            { field, definition_id := value.reference.value, constraint := some (.present true) }
          pure ({ condition := some (.predicate wire) },
            .predicate ⟨predicateCode value.field, value.reference, none⟩, 1)
      | .equals (.text text) => do
          let field ← predicateField value.field
          let wire : ScopedPredicate :=
            { field, definition_id := value.reference.value, constraint := some (.equals_text text) }
          pure ({ condition := some (.predicate wire) },
            .predicate ⟨predicateCode value.field, value.reference, some text⟩, 1)
      | _ => throw "unsupported correlation constraint"
  | .all items => do
      if items.isEmpty then throw "empty correlation group"
      let (wires, decoded, depth) ← correlations coverage captures items
      pure ({ condition := some (.all { operands := wires }) }, .all decoded, depth + 1)
  | .any items => do
      if items.isEmpty then throw "empty correlation group"
      let (wires, decoded, depth) ← correlations coverage captures items
      pure ({ condition := some (.any { operands := wires }) }, .any decoded, depth + 1)
  | .not _ => throw "unsupported correlation negation"
  termination_by expression => sizeOf expression

private def correlations (coverage : Projection.Coverage plan)
    (captures : List Testpilot.Scoped.Capture) :
    List PropertyPredicate →
      Except String (Array ScopedCorrelation × Testpilot.Scoped.Correlations × Nat)
  | [] => pure (#[], .nil, 0)
  | head :: rest => do
      let (headWire, headDecoded, headDepth) ← correlation coverage captures head
      let (wires, decoded, depth) ← correlations coverage captures rest
      pure (#[headWire] ++ wires, .cons headDecoded decoded, max headDepth depth)
  termination_by items => sizeOf items
end

/-- One lowered clause: its wire form, its keyed fragment and the correlation depth it needs. -/
private structure LoweredClause where
  wire : ScopedClause
  keyed : String × Testpilot.Scoped.Keyed
  depth : Nat

private def clause (coverage : Projection.Coverage plan)
    (source : ResolvedPropertyScopedClause) : Except String LoweredClause := do
  let captures ← source.declaration.captures.mapM (capture coverage)
  let declared := captures.map Prod.snd
  let correlated ← source.correlation.mapM fun checked =>
    correlation coverage declared checked.expression
  pure {
    wire := ScopedClause.mk source.declaration.id.value .SCOPED_CLOCK_OPERATION_TRANSITIONS
      (← number source.declaration.bound)
      (match source.declaration.endpoint with
        | .runtimePrefix => .SCOPED_ENDPOINT_RUNTIME_PREFIX
        | .deliberatelyClosed => .SCOPED_ENDPOINT_DELIBERATELY_CLOSED)
      (some (← pattern source.triggerPattern)) (some (← pattern source.responsePattern))
      (captures.map Prod.fst).toArray (correlated.map (·.1)) default
    keyed := (source.declaration.id.value, ⟨declared, correlated.map (·.2.1)⟩)
    depth := (correlated.map (·.2.2)).getD 0 }

/-- Successful lowering carries equality of the data actually decoded for portable execution. -/
structure Lowered (plan : Projection.Checked target) (compiled : Property.Scoped.Compiled target) where
  wire : ScopedContract
  decoded : Testpilot.Scoped.Compiled
  keyed : List (String × Testpilot.Scoped.Keyed)
  decoding : Testpilot.Scoped.decode wire = .ok decoded
  meaning : decoded = Scoped.meaning plan compiled keyed
  maximumFacts : plan.executable.transitions.foldl (fun maximum row => max maximum row.2.2.facts.length) 0 =
    target.behaviorTable.transitions.foldl (fun maximum row => max maximum row.facts.length) 0
  candidateCounts : ∀ row ∈ plan.executable.transitions,
    (plan.executable.transitions.filter (fun candidate => candidate.1 == row.1 && candidate.2.1 == row.2.1)).length =
      (target.machine.steps row.1 row.2.1).length
  certificates : ∀ binding ∈ compiled.portableReferences,
    ScopedProofs.Certificate binding plan.executable.transitions plan.initialState

/-- Lower checked scoped declarations and projection together; failure rejects the entire fragment. -/
def lower (plan : Projection.Checked target) (compiled : Property.Scoped.Compiled target)
    (evidenceObservationId : String)
    (coverage : Projection.Coverage plan := Projection.Coverage.empty plan) :
    Except Compiler.LoweringError (Lowered plan compiled) := do
  let failed := fun reason => Compiler.LoweringError.mk compiled.property.id.value
    compiled.property.source reason
  let lowered ← (compiled.property.scopedClauses.mapM (clause coverage)).mapError failed
  let keyed := lowered.map (·.keyed)
  let build : Except String ScopedContract := do
    let declaration := plan.sourceDeclaration
    let rules ← declaration.rules.mapM fun rule => do
      let fields ← rule.fields.mapM fieldPolicy
      let (meaning, submission, outputs) := match rule.meaning with
        | .irrelevant => (ScopedEvidenceMeaning.SCOPED_EVIDENCE_MEANING_IRRELEVANT, none, [])
        | .submission action => (.SCOPED_EVIDENCE_MEANING_SUBMISSION, some (atom action), [])
        | .confirmed required steps => (.SCOPED_EVIDENCE_MEANING_CONFIRMED,
            required.map atom, steps.map fun (action, result) => output action result)
      pure (ScopedProjectionRule.mk rule.kind.value meaning submission outputs.toArray fields.toArray default)
    -- A capability that declares neither captures nor a correlation leaves both ceilings unset, so
    -- its encoding and meaning are exactly the ones it had before the keyed capability existed.
    let declaresCaptures := keyed.any fun entry => !entry.2.captures.isEmpty
    let limits : ScopedLimits := {
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
    pure {
      version := 1
      projection_id := declaration.id.value
      projection_fingerprint := plan.behaviorFingerprint.render
      evidence_observation_id := evidenceObservationId
      scope_fields := declaration.scopeFields.toArray.map DefinitionId.value
      operation_field := declaration.operationField.value
      sources := declaration.sources.toArray.map DefinitionId.value
      initial_state := some (atom plan.initialState)
      transitions := plan.executable.transitions.toArray.map fun (prior, action, result) =>
        { output action result with prior_state := some (atom prior) }
      projection_rules := rules.toArray
      clauses := (lowered.map (·.wire)).toArray
      limits := some limits }
  let wire ← build.mapError failed
  match decoding : Testpilot.Scoped.decode wire with
  | .error reason => throw (failed reason)
  | .ok decoded =>
      if agreement : decoded = meaning plan compiled keyed then
        let certificates ← (ScopedProofs.certifyAll compiled.portableReferences
          plan.executable.transitions plan.initialState).mapError failed
        if maximumFacts : plan.executable.transitions.foldl
            (fun maximum row => max maximum row.2.2.facts.length) 0 =
            target.behaviorTable.transitions.foldl (fun maximum row => max maximum row.facts.length) 0 then
          if candidateCounts : ∀ row ∈ plan.executable.transitions,
              (plan.executable.transitions.filter (fun candidate =>
                candidate.1 == row.1 && candidate.2.1 == row.2.1)).length =
                  (target.machine.steps row.1 row.2.1).length then
            pure ⟨wire, decoded, keyed, decoding, agreement, maximumFacts, candidateCounts, certificates.down⟩
          else throw (failed "portable work candidate multiplicity differs from checked kernel")
        else throw (failed "portable work maximum fact count differs from checked description")
      else throw (failed "portable scoped meaning roundtrip mismatch")

/-- Carry the checked property/source mapping in opaque Umpire provenance, not executable metadata. -/
def Lowered.contractLowering {plan : Projection.Checked target} {compiled : Property.Scoped.Compiled target} (lowered : Lowered plan compiled) : Compiler.ContractLowering :=
  .scoped ⟨compiled.property.id.value, compiled.property.behaviorFingerprint.render, .property⟩
    lowered.wire (compiled.property.scopedClauses.map fun clause => {
      clauseId := clause.declaration.id.value
      propertyId := compiled.property.id.value
      propertyFingerprint := compiled.property.behaviorFingerprint.render
      projectionId := plan.sourceDeclaration.id.value
      projectionFingerprint := plan.behaviorFingerprint.render
      source := clause.declaration.source })

/-- Any row executable by the actual decoded Contract is authorized by the original Target. -/
theorem Lowered.table_authorized {plan : Projection.Checked target} {compiled : Property.Scoped.Compiled target} (lowered : Lowered plan compiled)
    (row) (member : row ∈ lowered.decoded.plan.transitions) :
    target.machine.authoritativeStep row.1 row.2.1 row.2.2 := by
  rw [lowered.meaning] at member
  exact plan.table_authorized row member

/-- Each actual decoded window selects its checked source clause through the lowering certificate.
The witness is built from the exact retained rows, never from an independently supplied truth value. -/
theorem Lowered.window_property {plan : Projection.Checked target}
    {compiled : Property.Scoped.Compiled target} (lowered : Lowered plan compiled)
    {history : List (Shared.ScopedObligation.Admitted lowered.decoded.plan.transitions)}
    (window : Shared.ScopedObligation.Window lowered.decoded.clauses history) :
    ∃ binding ∈ compiled.portableReferences,
      ∃ input : CheckedPropertyEvaluationInput binding.reference,
        ScopedProofs.HistoryInput binding plan.initialState (history.map (·.value)) input ∧
        window.obligations.all (fun obligation => decide (obligation = .satisfied)) =
          evaluatePropertyClause binding.reference input binding.clause := by
  have member := window.clause.property
  have clauses : lowered.decoded.clauses = compiled.portableClauses := congrArg (·.clauses) lowered.meaning
  have sourceMember := Eq.mp (congrArg (fun cs => window.clause.val ∈ cs) clauses) member
  obtain ⟨binding, member, same⟩ := List.mem_map.mp sourceMember
  have certificate := lowered.certificates binding member
  have table : lowered.decoded.plan.transitions = plan.executable.transitions := by rw [lowered.meaning]; rfl
  have checked : ScopedProofs.Certificate binding lowered.decoded.plan.transitions plan.initialState := by
    rw [table]; exact certificate
  obtain ⟨input, admitted, answer⟩ := checked.closed_property window same.symm
  exact ⟨binding, member, input, admitted, answer⟩

/-- Actual decoded scope/source/field admission is the checked projector's admission boundary. -/
theorem Lowered.evidence_validation {plan : Projection.Checked target}
    {compiled : Property.Scoped.Compiled target} (lowered : Lowered plan compiled)
    (scope : List (DefinitionId × String)) (event : Projection.Event) :
    lowered.decoded.validateEvent scope event =
      (plan.validateEvent scope event).mapError (fun _ => "invalid evidence") := by
  rw [lowered.meaning]
  unfold Testpilot.Scoped.Compiled.validateEvent Projection.Checked.validateEvent
  simp only [Scoped.meaning, plan.executable_limits]
  rfl

/-- Every window returned by actual wire observation admission retains checked Property correspondence. -/
theorem Lowered.observed_property {plan : Projection.Checked target}
    {compiled : Property.Scoped.Compiled target} (lowered : Lowered plan compiled)
    (before after : Testpilot.Scoped.Run lowered.decoded) (sequence : Nat)
    (wire : ScopedEvidence) (_admitted : before.observe sequence wire = .ok after)
    (operation : Shared.ScopedObligation.Operation lowered.decoded.plan.transitions lowered.decoded.clauses)
    (_operation : operation ∈ after.monitor.operations)
    (window : Shared.ScopedObligation.Window lowered.decoded.clauses operation.history)
    (_window : window ∈ operation.windows) :
    ∃ binding ∈ compiled.portableReferences,
      ∃ input : CheckedPropertyEvaluationInput binding.reference,
        ScopedProofs.HistoryInput binding plan.initialState (operation.history.map (·.value)) input ∧
        window.obligations.all (fun obligation => decide (obligation = .satisfied)) =
          evaluatePropertyClause binding.reference input binding.clause :=
  lowered.window_property window

end Umpire.Case.Scoped

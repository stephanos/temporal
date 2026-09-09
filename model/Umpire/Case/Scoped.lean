import Testpilot.Scoped
import Umpire.Case.ScopedProofs
import Umpire.Case.Compiler
import Umpire.Observation.Evaluation.Scoped

/-!
Checked lowering of operation-scoped obligations into the closed Testpilot capability. The complete
Target table and admitted projection mappings are serialized; a checked decode equality binds the
actual portable interpreter to that exact shared executable table. Numeric narrowing is checked
before any protobuf constructor can truncate a value.
-/
namespace Umpire.Case.Scoped
open temporal.server.api.testpilot.v1
open Umpire.Observation

variable {Law : LawDefinition → Prop} {Setup : Type}
variable {target : CheckedTarget Law Setup ModelValue ModelValue ModelValue ModelValue}

private def number (value : Nat) : Except String Int64 :=
  if value ≤ 9223372036854775807 then .ok (Int64.ofInt value) else .error "protobuf signed overflow"
private def atom (value : ModelValue) : ScopedValue :=
  { definition_id := value.definitionId.value, value := value.value }
private def output (action : ModelValue) (result : TransitionResult ModelValue ModelValue ModelValue) :
    ScopedTransition := {
  action := some (atom action)
  resulting_state := some (atom result.resultingState)
  outcome := some (atom result.modelOutcome)
  facts := result.observations.toArray.map atom }
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

private def meaning (plan : Projection.Checked target) (compiled : Property.Scoped.Compiled target) :
    Testpilot.Scoped.Compiled := {
  plan := plan.executable
  clauses := compiled.portableClauses
  scopeFields := compiled.scopeFields
  operationField := compiled.operationField
  sources := plan.sourceDeclaration.sources
  policies := plan.fieldPolicies
  transitions := compiled.limits.transitions
  obligations := compiled.limits.obligations
  work := compiled.limits.work }

/-- Successful lowering carries equality of the data actually decoded for portable execution. -/
structure Lowered (plan : Projection.Checked target) (compiled : Property.Scoped.Compiled target) where
  wire : ScopedContract
  decoded : Testpilot.Scoped.Compiled
  decoding : Testpilot.Scoped.decode wire = .ok decoded
  meaning : decoded = Scoped.meaning plan compiled
  maximumFacts : plan.executable.transitions.foldl (fun maximum row => max maximum row.2.2.observations.length) 0 =
    target.behaviorDescription.transitions.foldl (fun maximum row => max maximum row.observations.length) 0
  candidateCounts : ∀ row ∈ plan.executable.transitions,
    (plan.executable.transitions.filter (fun candidate => candidate.1 == row.1 && candidate.2.1 == row.2.1)).length =
      (target.kernel.steps row.1 row.2.1).length
  certificates : ∀ binding ∈ compiled.portableReferences,
    ScopedProofs.Certificate binding plan.executable.transitions plan.initialState

/-- Lower checked scoped declarations and projection together; failure rejects the entire fragment. -/
def lower (plan : Projection.Checked target) (compiled : Property.Scoped.Compiled target)
    (evidenceObservationId : String) : Except Compiler.LoweringError (Lowered plan compiled) := do
  let failed := fun reason => Compiler.LoweringError.mk compiled.property.id.value
    compiled.property.source reason
  -- The portable contract carries no keyed captures yet, so lowering a clause that declares them
  -- would silently drop its correlation and let offline replay disagree with the model.
  if let some clause := compiled.property.scopedClauses.find? fun clause =>
      !clause.declaration.captures.isEmpty || clause.correlation.isSome then
    throw (failed ("unsupported keyed field captures in scoped clause " ++ clause.declaration.id.value))
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
    let clauses ← compiled.property.scopedClauses.mapM fun clause => do
      pure (ScopedClause.mk clause.declaration.id.value .SCOPED_CLOCK_OPERATION_TRANSITIONS
        (← number clause.declaration.bound)
        (match clause.declaration.endpoint with
          | .runtimePrefix => .SCOPED_ENDPOINT_RUNTIME_PREFIX
          | .deliberatelyClosed => .SCOPED_ENDPOINT_DELIBERATELY_CLOSED)
        (some (← pattern clause.triggerPattern)) (some (← pattern clause.responsePattern)) default)
    let limits : ScopedLimits := {
      max_events := ← number declaration.limits.events
      max_buffered := ← number declaration.limits.buffered
      max_keys := ← number declaration.limits.keys
      max_support := ← number declaration.limits.support
      max_projection_work := ← number declaration.limits.work
      max_event_bytes := ← number declaration.limits.eventSize
      max_semantic_transitions := ← number compiled.limits.transitions
      max_obligations := ← number compiled.limits.obligations
      max_obligation_work := ← number compiled.limits.work }
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
      clauses := clauses.toArray
      limits := some limits }
  let wire ← build.mapError failed
  match decoding : Testpilot.Scoped.decode wire with
  | .error reason => throw (failed reason)
  | .ok decoded =>
      if agreement : decoded = meaning plan compiled then
        let certificates ← (ScopedProofs.certifyAll compiled.portableReferences
          plan.executable.transitions plan.initialState).mapError failed
        if maximumFacts : plan.executable.transitions.foldl
            (fun maximum row => max maximum row.2.2.observations.length) 0 =
            target.behaviorDescription.transitions.foldl (fun maximum row => max maximum row.observations.length) 0 then
          if candidateCounts : ∀ row ∈ plan.executable.transitions,
              (plan.executable.transitions.filter (fun candidate =>
                candidate.1 == row.1 && candidate.2.1 == row.2.1)).length =
                  (target.kernel.steps row.1 row.2.1).length then
            pure ⟨wire, decoded, decoding, agreement, maximumFacts, candidateCounts, certificates.down⟩
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
    target.kernel.authoritativeStep row.1 row.2.1 row.2.2 := by
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

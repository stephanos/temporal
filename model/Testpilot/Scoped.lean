import Testpilot.Protocol
import Shared.SemanticData
import Shared.ScopedProjection
import Shared.ScopedObligation

/-!
The version-one scoped Contract interpreter decodes closed protobuf data into a finite projection
table and passive response windows. Its core is also used by checked model projection. No model,
Target callback, Driver, or wall clock participates in this interpreter.
-/
namespace Testpilot.Scoped
open temporal.server.api.testpilot.v1
open Shared.SemanticData
open Shared

abbrev Result := Shared.SemanticData.Result Atom Atom Atom
abbrev Field := ScopedProjection.EvidenceField Name Scalar
abbrev Event := ScopedProjection.Event Name Field
abbrev Plan := ScopedProjection.Plan Name Atom Atom Result

abbrev Predicate := Shared.ScopedObligation.Predicate
abbrev Predicate.mk := Shared.ScopedObligation.Predicate.mk
abbrev Clause := Shared.ScopedObligation.Clause
abbrev Clause.mk := Shared.ScopedObligation.Clause.mk

structure Compiled where
  plan : Plan
  clauses : List Clause
  scopeFields : List Name
  operationField : Name
  sources : List Name
  policies : List (Name × List (Name × Nat × Nat))
  transitions : Nat
  obligations : Nat
  work : Nat
  deriving BEq, DecidableEq

private def validId (value : String) : Bool :=
  !value.isEmpty && value.length ≤ 256 && value.toList.all (fun c =>
    c.toNat < 128 && (c.isAlphanum || c == '_' || c == '-' || c == '.'))
private def uniqueIds (values : List String) : Bool :=
  !values.isEmpty && values.all validId && values.eraseDups == values
private def required (value : Option α) : Except String α := match value with | some value => .ok value | none => .error "missing field"
private def natural (value : Int64) : Except String Nat :=
  if value.toInt < 0 then .error "negative semantic number" else .ok value.toInt.toNat
private def positive (value : Int64) : Except String Nat := do
  let n ← natural value
  if n == 0 then throw "nonpositive resource limit"
  pure n

def atom (value : ScopedValue) : Atom := ⟨⟨value.definition_id⟩, value.value⟩
private def checkedAtom (value : ScopedValue) : Except String Atom := do
  if !value.«Unknown.Fields».isEmpty || !validId value.definition_id then throw "invalid semantic value"
  pure (atom value)
private def result (value : ScopedTransition) : Except String Result := do
  if !value.«Unknown.Fields».isEmpty then throw "unknown transition field"
  pure ⟨← checkedAtom (← required value.outcome), ← checkedAtom (← required value.resulting_state),
    ← value.facts.toList.mapM checkedAtom⟩

private def predicate (value : ScopedPredicate) : Except String Predicate := do
  if !value.«Unknown.Fields».isEmpty || !validId value.definition_id then throw "invalid predicate"
  let field ← match value.field with
    | .SCOPED_PREDICATE_FIELD_ACTION => pure 1
    | .SCOPED_PREDICATE_FIELD_OUTCOME => pure 2
    | .SCOPED_PREDICATE_FIELD_RESULTING_STATE => pure 3
    | .SCOPED_PREDICATE_FIELD_FACT => pure 4
    | _ => throw "unsupported predicate field"
  let equalsText ← match value.constraint with
    | some (.present true) => pure none
    | some (.equals_text text) => pure (some text)
    | _ => throw "unsupported predicate constraint"
  pure ⟨field, ⟨value.definition_id⟩, equalsText⟩

/-- Decode exact v1 executable meaning, rejecting unsupported clocks, endpoints and numeric values. -/
def decode (wire : ScopedContract) : Except String Compiled := do
  if wire.version != 1 then throw "unsupported scoped capability version"
  if !wire.«Unknown.Fields».isEmpty then throw "unknown capability field"
  if !validId wire.projection_id || wire.projection_fingerprint.isEmpty ||
      !validId wire.evidence_observation_id || !validId wire.operation_field ||
      !uniqueIds wire.scope_fields.toList || !uniqueIds wire.sources.toList ||
      wire.scope_fields.contains wire.operation_field then throw "invalid projection binding"
  let limits ← required wire.limits
  if !limits.«Unknown.Fields».isEmpty then throw "unknown limits field"
  let projectionLimits : ScopedProjection.Limits := {
    events := ← positive limits.max_events
    buffered := ← positive limits.max_buffered
    keys := ← positive limits.max_keys
    support := ← positive limits.max_support
    work := ← positive limits.max_projection_work
    eventSize := ← positive limits.max_event_bytes }
  if projectionLimits.buffered > projectionLimits.events || projectionLimits.keys > projectionLimits.events then
    throw "incompatible projection limits"
  if wire.transitions.isEmpty || wire.projection_rules.isEmpty || wire.clauses.isEmpty then
    throw "empty scoped capability"
  let transitions ← wire.transitions.toList.mapM fun tr => do
    if !tr.«Unknown.Fields».isEmpty then throw "unknown transition field"
    pure (← checkedAtom (← required tr.prior_state), ← checkedAtom (← required tr.action), ← result tr)
  let policies ← wire.projection_rules.toList.mapM fun rule => do
    let fields ← rule.fields.toList.mapM fun field => do
      if !field.«Unknown.Fields».isEmpty || !validId field.field_id then throw "invalid field policy"
      let type ← required field.type
      if !type.«Unknown.Fields».isEmpty then throw "unknown field type"
      let kind ← match type.kind with
        | .SCALAR_KIND_TEXT => pure 1
        | .SCALAR_KIND_NATURAL => pure 2
        | .SCALAR_KIND_BOOLEAN => pure 3
        | _ => throw "unsupported field type"
      let disposition ← match field.disposition with
        | .SCOPED_FIELD_DISPOSITION_RETAIN => pure 1
        | .SCOPED_FIELD_DISPOSITION_REDACT => pure 2
        | .SCOPED_FIELD_DISPOSITION_REJECT => pure 3
        | _ => throw "unsupported field policy"
      pure (Name.mk field.field_id, kind, disposition)
    if (fields.map (·.1)).eraseDups != fields.map (·.1) then throw "duplicate field policy"
    pure (Name.mk rule.kind, fields)
  if !uniqueIds (wire.projection_rules.toList.map (·.kind)) then throw "invalid evidence kinds"
  let rules ← wire.projection_rules.toList.mapM fun rule => do
    if !rule.«Unknown.Fields».isEmpty then throw "unknown projection rule field"
    let meaning ← match rule.meaning with
      | .SCOPED_EVIDENCE_MEANING_IRRELEVANT => do
          if rule.submission.isSome || !rule.outputs.isEmpty then throw "irrelevant evidence has outputs"
          pure .irrelevant
      | .SCOPED_EVIDENCE_MEANING_SUBMISSION =>
          let action ← checkedAtom (← required rule.submission)
          if !rule.outputs.isEmpty || !transitions.any (·.2.1 == action) then throw "invalid submission"
          pure (.submission action)
      | .SCOPED_EVIDENCE_MEANING_CONFIRMED => do
          if rule.outputs.isEmpty then throw "confirmed evidence requires outputs"
          let submission ← rule.submission.mapM checkedAtom
          if let some action := submission then
            if !wire.projection_rules.any (fun other =>
                other.meaning == .SCOPED_EVIDENCE_MEANING_SUBMISSION &&
                other.submission.map atom == some action) then throw "missing submission mapping"
          let outputs ← rule.outputs.toList.mapM fun output => do
            if output.prior_state.isSome then throw "output supplied a prior state"
            let action ← checkedAtom (← required output.action)
            let result ← result output
            if !transitions.any (fun row => row.2.1 == action && row.2.2 == result) then
              throw "projection output absent from table"
            pure (action, result)
          pure (.confirmed submission outputs)
      | _ => throw "unsupported projection meaning"
    pure (ScopedProjection.Rule.mk ⟨rule.kind⟩ rule.fields.size meaning)
  if !uniqueIds (wire.clauses.toList.map (·.clause_id)) then throw "invalid clause identities"
  let clauses ← wire.clauses.toList.mapM fun clause => do
    if !clause.«Unknown.Fields».isEmpty then throw "unknown clause field"
    if clause.clock != .SCOPED_CLOCK_OPERATION_TRANSITIONS then throw "unsupported semantic clock"
    let deliberatelyClosed ← match clause.endpoint with
      | .SCOPED_ENDPOINT_RUNTIME_PREFIX => pure false
      | .SCOPED_ENDPOINT_DELIBERATELY_CLOSED => pure true
      | _ => throw "unsupported endpoint"
    let trigger ← predicate (← required clause.trigger)
    let response ← predicate (← required clause.response)
    if trigger.field != 1 || response.field < 2 then throw "unsupported clause"
    pure (Clause.mk clause.clause_id (← natural clause.bound) deliberatelyClosed trigger response)
  pure {
    plan := { initial := ← checkedAtom (← required wire.initial_state), rules, transitions, limits := projectionLimits }
    clauses
    scopeFields := wire.scope_fields.toList.map Name.mk
    operationField := ⟨wire.operation_field⟩
    sources := wire.sources.toList.map Name.mk
    policies
    transitions := ← positive limits.max_semantic_transitions
    obligations := ← positive limits.max_obligations
    work := ← positive limits.max_obligation_work }

/-- Immutable state allocated independently for each decoded Contract execution. -/
structure Run (compiled : Compiled) where
  projection : ScopedProjection.Run compiled.plan Field
  monitor : ScopedObligation.Monitor compiled.plan.transitions compiled.clauses
  closed : Bool := false

/-- Bind a fresh execution scope without observing or dispatching any work. -/
def Compiled.start (compiled : Compiled) (scope : List (Name × String)) : Except String (Run compiled) := do
  if scope.map Prod.fst != compiled.scopeFields || scope.any (·.2.isEmpty) then throw "wrong scope"
  pure { projection := { scope }, monitor := { initial := compiled.plan.initial } }

private def identitySize (identity : ScopedProjection.Identity Name) : Nat :=
  identity.source.value.length + 1 + identity.scope.foldl (fun size (key, value) =>
    size + key.value.length + value.length) 0

/-- The same declared semantic byte accounting used by source evidence projection. -/
def eventSize (event : Event) : Nat :=
  identitySize event.identity + event.operation.length + event.kind.value.length +
    event.runSequences.foldl (fun size sequence => size + (toString sequence).length) 0 +
    event.parents.foldl (fun size parent => size + identitySize parent) 0 +
    event.fields.foldl (fun size field => size + field.id.value.length +
      (field.value.map Scalar.size).getD 1) 0

private def scalarKind : Scalar → Nat
  | .text _ => 1 | .natural _ => 2 | .boolean _ => 3

/-- Validate scope, field authority and evidence support before transactional projection. -/
def Compiled.validateEvent (compiled : Compiled) (scope : List (Name × String)) (event : Event) :
    Except String Unit :=
  (ScopedProjection.validateEvidence scalarKind eventSize compiled.plan.limits compiled.sources
    compiled.policies scope event).mapError (fun _ => "invalid evidence")

/-- Consume only validated evidence; the shared projector controls all semantic-step emissions. -/
def Run.admit {compiled : Compiled} (run : Run compiled) (event : Event) :
    Except String (Run compiled) := do
  if run.closed then throw "closed"
  compiled.validateEvent run.projection.scope event
  let projection ← (run.projection.admit Name.value (·.resultingState) eventSize event).mapError
    (fun _ => "projection rejected")
  let limits : ScopedObligation.MonitorLimits :=
    ⟨compiled.transitions, compiled.obligations, compiled.work⟩
  let monitor ← (projection.steps.drop run.projection.steps.length).foldlM
    (fun monitor step => monitor.consume limits step.operation
      ⟨(step.priorState, step.action, step.result), step.member⟩) run.monitor
  pure { run with projection, monitor }

private def wireIdentity (wire : ScopedIdentity) : Except String (ScopedProjection.Identity Name) := do
  if !wire.«Unknown.Fields».isEmpty then throw "unknown identity field"
  let scope ← wire.scope.toList.mapM fun binding => do
    if !binding.«Unknown.Fields».isEmpty then throw "unknown binding field"
    pure (Name.mk binding.field_id, binding.value)
  pure ⟨scope, ⟨wire.source⟩, ← natural wire.ordinal⟩

private def wireScalar (wire : temporal.server.api.testpilot.v1.Value) : Except String Scalar := do
  if !wire.«Unknown.Fields».isEmpty then throw "unknown scalar field"
  match wire.value with
  | some (.text text) => pure (.text text)
  | some (.natural text) =>
      match text.toNat? with
      | some value => if toString value == text then pure (.natural value) else throw "noncanonical natural"
      | none => throw "invalid natural"
  | some (.bool_value value) => pure (.boolean value)
  | _ => throw "unsupported evidence scalar"

/-- Decode one typed Observation and attach recorder-owned support, retaining first support on duplicates. -/
def Run.observe {compiled : Compiled} (run : Run compiled) (sequence : Nat) (wire : ScopedEvidence) :
    Except String (Run compiled) := do
  if sequence == 0 || !wire.«Unknown.Fields».isEmpty then throw "invalid evidence envelope"
  let identity ← wireIdentity (← required wire.identity)
  let parents ← wire.parents.toList.mapM wireIdentity
  let fields ← wire.fields.toList.mapM fun field => do
    if !field.«Unknown.Fields».isEmpty then throw "unknown evidence field"
    let value ← match field.value with | none => pure none | some value => some <$> wireScalar value
    pure (ScopedProjection.EvidenceField.mk (Name.mk field.field_id) value)
  let sequences := (run.projection.accepted.find? (·.identity == identity)).map (·.runSequences)
    |>.getD [sequence]
  run.admit ⟨identity, wire.operation, ⟨wire.kind⟩, parents, sequences, fields⟩

/-- Chunk boundaries do not create semantic coordinates or close pending evidence. -/
def Run.admitMany {compiled : Compiled} (run : Run compiled) (events : List Event) :
    Except String (Run compiled) := events.foldlM Run.admit run

/-- Exact incremental/offline identity, including the first rejected chunk. -/
theorem Run.admitMany_append {compiled : Compiled} (run : Run compiled) (first second : List Event) :
    run.admitMany (first ++ second) =
      (run.admitMany first >>= fun next => next.admitMany second) := by
  simp [admitMany, List.foldlM_append]

/-- Inspect current answers; missing evidence forces unresolved but never repairs a violation. -/
def Run.answers {compiled : Compiled} (run : Run compiled) (incomplete : Bool := false) : List Nat :=
  (run.monitor.answers run.closed (incomplete || !run.projection.pending.isEmpty)).map fun answer =>
    match answer with | .satisfied => 2 | .violated => 3 | .unresolved => 0

/-- Close without inventing a semantic transition or a wall-clock horizon. -/
def Run.close {compiled : Compiled} (run : Run compiled) : Run compiled := { run with closed := true }

end Testpilot.Scoped

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

/-- One correlation operand. Only declared evidence is readable: an exact literal, one declared
evidence field of the step being admitted, or one retained earlier occurrence of a declared
capture. No private Slot or raw payload is reachable from here. -/
inductive Operand where
  | literal (value : Scalar)
  | field (id : Name)
  | capture (id : Name) (ordinal : Nat)
  deriving BEq, DecidableEq

mutual
/-- The closed correlation vocabulary. `all` and `any` are evaluated left to right and stop at the
first decisive operand, so an operand an earlier one made irrelevant is never read and cannot fail
admission. -/
inductive Correlation where
  | predicate (value : Predicate)
  | comparison (equal : Bool) (left right : Operand)
  | all (operands : Correlations)
  | any (operands : Correlations)
/-- The operands of one `all` or `any` group, in declaration order. -/
inductive Correlations where
  | nil
  | cons (head : Correlation) (tail : Correlations)
end
deriving instance DecidableEq for Correlation, Correlations
deriving instance BEq for Correlation, Correlations

/-- One declared per-operation capture: which declared evidence field each occurrence retains and
how many occurrences one operation keeps. Occurrences are numbered from zero in admission order. -/
structure Capture where
  id : Name
  field : Name
  /-- The declared scalar kind of the retained field, so a comparison reading this capture is
  checked against the same type the projection declares. -/
  kind : Nat
  lifetime : Nat
  deriving BEq, DecidableEq

/-- The keyed declarations of one clause. A clause that declares neither keeps its exact existing
meaning and admits every labeled transition its projection emits. -/
structure Keyed where
  captures : List Capture
  correlation : Option Correlation
  deriving BEq, DecidableEq

structure Compiled where
  plan : Plan
  clauses : List Clause
  /-- Keyed declarations by clause id; clause ids are unique, so this is a total lookup. -/
  keyed : List (String × Keyed)
  scopeFields : List Name
  operationField : Name
  sources : List Name
  policies : List (Name × List (Name × Nat × Nat))
  transitions : Nat
  obligations : Nat
  work : Nat
  captures : Nat
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

private def scalarKind : Scalar → Nat
  | .text _ => 1 | .natural _ => 2 | .boolean _ => 3

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

/-- The one declared scalar kind of a retained evidence field. A field two projection rules declare
at different kinds has no single type a comparison could be checked against, so reading it rejects
rather than comparing values of different kinds. -/
private def retainedKind (retained : List (Name × Nat)) (id : Name) : Except String Nat :=
  match (retained.filter (·.1 == id)).map Prod.snd |>.eraseDups with
  | [kind] => .ok kind
  | [] => .error "unretained correlation field operand"
  | _ => .error "ambiguous retained field type"

/-- One correlation operand with its declared scalar kind, checked against the declarations that can
supply it: a literal decodes to an exact admitted scalar, a field must be one this projection
retains, and a capture reference must name a capture this clause declared at an ordinal its lifetime
keeps. -/
private def operand (retained : List (Name × Nat)) (captures : List Capture)
    (wire : ScopedOperand) : Except String (Operand × Nat) := do
  if !wire.«Unknown.Fields».isEmpty then throw "unknown operand field"
  match wire.operand with
  | some (.literal value) =>
      let value ← wireScalar value
      pure (.literal value, scalarKind value)
  | some (.field_id id) =>
      if !validId id then throw "unretained correlation field operand"
      pure (.field ⟨id⟩, ← retainedKind retained ⟨id⟩)
  | some (.capture reference) =>
      if !reference.«Unknown.Fields».isEmpty || !validId reference.capture_id then
        throw "invalid capture reference"
      let ordinal ← natural reference.ordinal
      let some declaration := captures.find? (·.id == ⟨reference.capture_id⟩)
        | throw "unbound capture reference"
      if ordinal ≥ declaration.lifetime then throw "capture ordinal beyond declared lifetime"
      pure (.capture ⟨reference.capture_id⟩ ordinal, declaration.kind)
  | _ => throw "unsupported correlation operand"

mutual
/-- Decode one correlation node under the declared depth ceiling. An exhausted depth is an explicit
rejection, never a silently truncated condition. -/
private def correlationOf (retained : List (Name × Nat)) (captures : List Capture) (depth : Nat)
    (wire : ScopedCorrelation) : Except String Correlation :=
  match depth with
  | 0 => throw "correlation depth exhausted"
  | remaining + 1 => do
    if !wire.«Unknown.Fields».isEmpty then throw "unknown correlation field"
    match wire.condition with
    | some (.predicate value) => pure (.predicate (← predicate value))
    | some (.comparison value) => do
        if !value.«Unknown.Fields».isEmpty then throw "unknown comparison field"
        let equal ← match value.operator with
          | .SCOPED_COMPARISON_OPERATOR_EQUAL => pure true
          | .SCOPED_COMPARISON_OPERATOR_NOT_EQUAL => pure false
          | _ => throw "unsupported comparison operator"
        let (left, leftKind) ← operand retained captures (← required value.left)
        let (right, rightKind) ← operand retained captures (← required value.right)
        if leftKind != rightKind then throw "incompatible correlation operand types"
        pure (.comparison equal left right)
    | some (.all group) => do
        if !group.«Unknown.Fields».isEmpty || group.operands.isEmpty then
          throw "empty correlation group"
        pure (.all (← correlationsOf retained captures remaining group.operands.toList))
    | some (.any group) => do
        if !group.«Unknown.Fields».isEmpty || group.operands.isEmpty then
          throw "empty correlation group"
        pure (.any (← correlationsOf retained captures remaining group.operands.toList))
    | _ => throw "unsupported correlation condition"
  termination_by (depth, 0)

private def correlationsOf (retained : List (Name × Nat)) (captures : List Capture) (depth : Nat)
    (wires : List ScopedCorrelation) : Except String Correlations :=
  match wires with
  | [] => pure .nil
  | head :: rest => do
      pure (.cons (← correlationOf retained captures depth head)
        (← correlationsOf retained captures depth rest))
  termination_by (depth, wires.length + 1)
end

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
  -- Only a field this projection actually retains can supply a capture or a correlation operand;
  -- redacted and rejected fields carry no value to read.
  let retainedFields := (policies.flatMap fun rule =>
    rule.2.filterMap fun field =>
      if field.2.2 == 1 then some (field.1, field.2.1) else none).eraseDups
  -- A capability that declares neither captures nor a correlation leaves both ceilings unset and
  -- keeps its exact existing encoding and meaning.
  -- Capture identities are one namespace across the capability: a retained occurrence is named by
  -- its capture id and ordinal alone, so two clauses declaring one id would alias the same stream.
  let declaredCaptures := wire.clauses.toList.flatMap (·.captures.toList.map (·.capture_id))
  if !declaredCaptures.isEmpty && !uniqueIds declaredCaptures then
    throw "invalid capture identities"
  let capturesDeclared := !declaredCaptures.isEmpty
  let correlationDeclared := wire.clauses.toList.any (·.correlation.isSome)
  let captureLimit ← if capturesDeclared then positive limits.max_captures
    else natural limits.max_captures
  let depthLimit ← if correlationDeclared then positive limits.max_correlation_depth
    else natural limits.max_correlation_depth
  let clauseData ← wire.clauses.toList.mapM fun clause => do
    if !clause.«Unknown.Fields».isEmpty then throw "unknown clause field"
    if clause.clock != .SCOPED_CLOCK_OPERATION_TRANSITIONS then throw "unsupported semantic clock"
    let deliberatelyClosed ← match clause.endpoint with
      | .SCOPED_ENDPOINT_RUNTIME_PREFIX => pure false
      | .SCOPED_ENDPOINT_DELIBERATELY_CLOSED => pure true
      | _ => throw "unsupported endpoint"
    let trigger ← predicate (← required clause.trigger)
    let response ← predicate (← required clause.response)
    if trigger.field != 1 || response.field < 2 then throw "unsupported clause"
    let captures ← clause.captures.toList.mapM fun declaration => do
      if !declaration.«Unknown.Fields».isEmpty || !validId declaration.capture_id ||
          !validId declaration.field_id then throw "invalid capture declaration"
      let kind ← (retainedKind retainedFields ⟨declaration.field_id⟩).mapError fun reason =>
        if reason == "unretained correlation field operand" then
          "capture names an unretained evidence field" else reason
      pure (Capture.mk ⟨declaration.capture_id⟩ ⟨declaration.field_id⟩ kind
        (← positive declaration.lifetime))
    let correlation ← clause.correlation.mapM (correlationOf retainedFields captures depthLimit)
    pure (Clause.mk clause.clause_id (← natural clause.bound) deliberatelyClosed trigger response,
      clause.clause_id, Keyed.mk captures correlation)
  pure {
    plan := { initial := ← checkedAtom (← required wire.initial_state), rules, transitions, limits := projectionLimits }
    clauses := clauseData.map (·.1)
    keyed := clauseData.map (·.2)
    scopeFields := wire.scope_fields.toList.map Name.mk
    operationField := ⟨wire.operation_field⟩
    sources := wire.sources.toList.map Name.mk
    policies
    transitions := ← positive limits.max_semantic_transitions
    obligations := ← positive limits.max_obligations
    work := ← positive limits.max_obligation_work
    captures := captureLimit }

/-- One retained occurrence: the operation that retained it, the capture it belongs to, its ordinal
and the exact admitted value. Entries are only appended, so an ordinal already recorded keeps the
value it was admitted with rather than being replaced by a later match. -/
structure Retained where
  operation : String
  capture : Name
  ordinal : Nat
  value : Scalar
  deriving BEq, DecidableEq

/-- Immutable state allocated independently for each decoded Contract execution. -/
structure Run (compiled : Compiled) where
  projection : ScopedProjection.Run compiled.plan Field
  monitor : ScopedObligation.Monitor compiled.plan.transitions compiled.clauses
  captures : List Retained := []
  capturedValues : Nat := 0
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

/-- Validate scope, field authority and evidence support before transactional projection. -/
def Compiled.validateEvent (compiled : Compiled) (scope : List (Name × String)) (event : Event) :
    Except String Unit :=
  (ScopedProjection.validateEvidence scalarKind eventSize compiled.plan.limits compiled.sources
    compiled.policies scope event).mapError (fun _ => "invalid evidence")

/-- Resolve one operand against this step's declared evidence and the operation's retained
occurrences. An occurrence the operation never retained -- a future ordinal, or one belonging to a
different operation -- has no value here, so admission fails rather than binding the nearest match. -/
private def operandValue (fields : List Field) (retained : List Retained) (operation : String) :
    Operand → Except String Scalar
  | .literal value => .ok value
  | .field id =>
      match (fields.find? (·.id == id)).bind (·.value) with
      | some value => .ok value
      | none => .error "missing correlation field operand"
  | .capture id ordinal =>
      match retained.find? fun entry =>
        entry.operation == operation && entry.capture == id && entry.ordinal == ordinal with
      | some entry => .ok entry.value
      | none => .error "missing retained capture occurrence"

mutual
/-- Whether this labeled transition satisfies the clause's declared correlation. -/
private def correlationHolds (fields : List Field) (retained : List Retained) (operation : String)
    (action : Atom) (result : Result) : Correlation → Except String Bool
  | .predicate value => .ok (value.holds action result)
  | .comparison equal left right => do
      let left ← operandValue fields retained operation left
      let right ← operandValue fields retained operation right
      pure (if equal then left == right else left != right)
  | .all operands => correlationAll fields retained operation action result operands
  | .any operands => correlationAny fields retained operation action result operands

private def correlationAll (fields : List Field) (retained : List Retained) (operation : String)
    (action : Atom) (result : Result) : Correlations → Except String Bool
  | .nil => .ok true
  | .cons head tail => do
      if ← correlationHolds fields retained operation action result head then
        correlationAll fields retained operation action result tail
      else pure false

private def correlationAny (fields : List Field) (retained : List Retained) (operation : String)
    (action : Atom) (result : Result) : Correlations → Except String Bool
  | .nil => .ok false
  | .cons head tail => do
      if ← correlationHolds fields retained operation action result head then pure true
      else correlationAny fields retained operation action result tail
end

/-- Retain this step's occurrence of every declared capture. A step that supplies no value at the
declared field records nothing; an operation that already holds its declared lifetime rejects. -/
private def retain (declarations : List Capture) (fields : List Field) (operation : String)
    (state : List Retained × Nat) : Except String (List Retained × Nat) :=
  declarations.foldlM (fun state declaration =>
    match (fields.find? (·.id == declaration.field)).bind (·.value) with
    | none => .ok state
    | some value =>
        let ordinal := (state.1.filter fun entry =>
          entry.operation == operation && entry.capture == declaration.id).length
        if ordinal ≥ declaration.lifetime then .error "capture lifetime exhausted"
        else .ok (state.1 ++ [⟨operation, declaration.id, ordinal, value⟩], state.2 + 1)) state

/-- Consume only validated evidence; the shared projector controls all semantic-step emissions.
A declared correlation decides which emitted steps are the operation's semantic steps at all: it
reads this step's own evidence together with what the operation already retained, so an occurrence
binds only after an earlier step admitted it, and this step's occurrences are retained only once
the whole append was admitted. -/
def Run.admit {compiled : Compiled} (run : Run compiled) (event : Event) :
    Except String (Run compiled) := do
  if run.closed then throw "closed"
  compiled.validateEvent run.projection.scope event
  let projection ← (run.projection.admit Name.value (·.resultingState) eventSize event).mapError
    (fun _ => "projection rejected")
  let limits : ScopedObligation.MonitorLimits :=
    ⟨compiled.transitions, compiled.obligations, compiled.work⟩
  let mut monitor := run.monitor
  let mut retained := run.captures
  let mut charged := 0
  for step in projection.steps.drop run.projection.steps.length do
    for declaration in compiled.keyed do
      if let some correlation := declaration.2.correlation then
        if !(← correlationHolds event.fields retained step.operation step.action step.result
            correlation) then
          throw "correlation rejected this operation's step"
    monitor ← monitor.consume limits step.operation
      ⟨(step.priorState, step.action, step.result), step.member⟩
    for declaration in compiled.keyed do
      let next ← retain declaration.2.captures event.fields step.operation (retained, charged)
      retained := next.1
      charged := next.2
  if run.capturedValues + charged > compiled.captures then throw "captures exhausted"
  pure { run with
    projection
    monitor
    captures := retained
    capturedValues := run.capturedValues + charged }

private def wireIdentity (wire : ScopedIdentity) : Except String (ScopedProjection.Identity Name) := do
  if !wire.«Unknown.Fields».isEmpty then throw "unknown identity field"
  let scope ← wire.scope.toList.mapM fun binding => do
    if !binding.«Unknown.Fields».isEmpty then throw "unknown binding field"
    pure (Name.mk binding.field_id, binding.value)
  pure ⟨scope, ⟨wire.source⟩, ← natural wire.ordinal⟩

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

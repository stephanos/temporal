import Testpilot.Scoped

/-!
Portable field and capture evaluation for the version-one scoped capability.

The capability under test is the closed one an offline reader decodes: declared evidence fields,
per-operation keyed captures of those fields, and a correlation over this step's fields and the
operation's retained occurrences. Every expected value here is written out by hand from the
declarations below rather than read back from the encoder, so the codec cannot serve as its own
oracle. The Lean model's lowering is deliberately not involved: this module authors the portable
contract directly, the way an independent producer would.
-/

open temporal.server.api.testpilot.v1
open Shared.SemanticData

namespace Testpilot.Tests.Fields

private def number (value : Int) : Int64 := Int64.ofInt value
private def value (definitionId text : String) : ScopedValue :=
  { definition_id := definitionId, value := text }

private def state := value "state" "ready"
private def request := value "action" "request"
private def reply := value "action" "reply"
private def quiet := value "outcome" "quiet"
private def responded := value "outcome" "response"

private def transition (action : ScopedValue) (outcome : ScopedValue) : ScopedTransition := {
  prior_state := some state, action := some action
  resulting_state := some state, outcome := some outcome }
private def output (action : ScopedValue) (outcome : ScopedValue) : ScopedTransition :=
  { transition action outcome with prior_state := none }

/-- Each evidence kind declares its own field, exactly as the two distinct operand paths a model
correlation names would be projected. -/
private def policy (fieldId : String)
    (disposition := ScopedFieldDisposition.SCOPED_FIELD_DISPOSITION_RETAIN) : ScopedFieldPolicy :=
  { field_id := fieldId, type := some { kind := .SCALAR_KIND_TEXT }, disposition }

private def rule (kind : String) (action outcome : ScopedValue) (fields : Array ScopedFieldPolicy) :
    ScopedProjectionRule := {
  kind, meaning := .SCOPED_EVIDENCE_MEANING_CONFIRMED
  outputs := #[output action outcome], fields }

private def requestRule (fields : Array ScopedFieldPolicy := #[policy "requested"]) :=
  rule "request" request quiet fields
private def replyRule := rule "reply" reply responded #[policy "replied"]
/-- A reply-shaped step that declares no fields at all, so a correlation reading one finds none. -/
private def silentRule := rule "silent" reply responded #[]

private def capture (lifetime : Nat := 2) (field := "requested") (id := "seen") :
    ScopedCaptureDeclaration :=
  { capture_id := id, field_id := field, lifetime := number lifetime }

private def literal (text : String) : ScopedOperand :=
  { operand := some (.literal { value := some (.text text) }) }
private def field (id : String) : ScopedOperand := { operand := some (.field_id id) }
private def natural (text : String) : ScopedOperand :=
  { operand := some (.literal { value := some (.natural text) }) }
private def retained (ordinal : Nat) (id := "seen") : ScopedOperand :=
  { operand := some (.capture { capture_id := id, ordinal := number ordinal }) }

private def comparison (left right : ScopedOperand)
    (operator := ScopedComparisonOperator.SCOPED_COMPARISON_OPERATOR_EQUAL) : ScopedCorrelation :=
  { condition := some (.comparison { operator, left := some left, right := some right }) }
private def triggered : ScopedCorrelation :=
  { condition := some (.predicate {
      field := .SCOPED_PREDICATE_FIELD_ACTION, definition_id := "action"
      constraint := some (.equals_text "request") }) }
private def anyOf (operands : Array ScopedCorrelation) : ScopedCorrelation :=
  { condition := some (.any { operands }) }
private def allOf (operands : Array ScopedCorrelation) : ScopedCorrelation :=
  { condition := some (.all { operands }) }

/-- The request step creates the occurrence its own correlation would read, so the trigger disjunct
decides it before the capture operand is reached. -/
private def correlation (ordinal : Nat := 0) : ScopedCorrelation :=
  anyOf #[triggered, comparison (field "replied") (retained ordinal)]

private def clause (bound : Nat) (captures : Array ScopedCaptureDeclaration := #[capture])
    (requirement : Option ScopedCorrelation := some (correlation))
    (endpoint := ScopedEndpoint.SCOPED_ENDPOINT_RUNTIME_PREFIX) (clauseId := "response") :
    ScopedClause := {
  clause_id := clauseId, clock := .SCOPED_CLOCK_OPERATION_TRANSITIONS
  bound := number bound, endpoint
  trigger := some {
    field := .SCOPED_PREDICATE_FIELD_ACTION, definition_id := "action"
    constraint := some (.equals_text "request") }
  response := some {
    field := .SCOPED_PREDICATE_FIELD_OUTCOME, definition_id := "outcome"
    constraint := some (.equals_text "response") }
  captures, correlation := requirement }

private def budget (captures : Nat := 8) (depth : Nat := 4) : ScopedLimits := {
  max_events := number 16, max_buffered := number 8, max_keys := number 8
  max_support := number 256, max_projection_work := number 1000000
  max_event_bytes := number 512, max_semantic_transitions := number 32
  max_obligations := number 16, max_obligation_work := number 1000000000
  max_captures := number captures, max_correlation_depth := number depth }

private def contract (clauses : Array ScopedClause) (limits : ScopedLimits := budget)
    (rules : Array ScopedProjectionRule := #[requestRule, replyRule, silentRule]) :
    ScopedContract := {
  version := 1, projection_id := "projection", projection_fingerprint := "fingerprint"
  evidence_observation_id := "evidence", scope_fields := #["run"], operation_field := "operation"
  sources := #["source"], initial_state := some state
  transitions := #[transition request quiet, transition reply responded]
  projection_rules := rules, clauses, limits := some limits }

private def evidence (ordinal : Nat) (kind : String) (count : String) (operation := "a")
    (fieldId := if kind == "request" then "requested" else "replied") : ScopedEvidence := {
  identity := some {
    scope := #[{ field_id := "run", value := "run-1" }], source := "source"
    ordinal := number ordinal }
  operation, kind
  fields := #[{ field_id := fieldId, value := some { value := some (.text count) } }] }

private def scope : List (Name × String) := [(⟨"run"⟩, "run-1")]

/-- Replay a stream offline and report the clause answer, or the exact first rejection. -/
private def replay (wire : ScopedContract) (events : List ScopedEvidence)
    (incomplete : Bool := false) : Except String (List Nat) := do
  let compiled ← Testpilot.Scoped.decode wire
  let initial ← compiled.start scope
  let run ← events.zipIdx.foldlM (fun run (event, index) => run.observe (index + 1) event) initial
  pure (run.close.answers incomplete)

private def answers (wire : ScopedContract) (events : List ScopedEvidence)
    (incomplete : Bool := false) : Option (List Nat) := (replay wire events incomplete).toOption
private def rejection (wire : ScopedContract) (events : List ScopedEvidence) : Option String :=
  match replay wire events with
  | .error reason => some reason
  | .ok _ => none

-- A reply whose declared field equals the retained request occurrence is one of the operation's
-- semantic steps and answers its bounded window; a different value is not this operation's reply.
#guard answers (contract #[clause 1]) [evidence 0 "request" "1", evidence 1 "reply" "1"] == some [2]
#guard rejection (contract #[clause 1]) [evidence 0 "request" "1", evidence 1 "reply" "2"] ==
  some "correlation rejected this operation's step"
#guard answers (contract #[clause 1 (requirement := none)])
  [evidence 0 "request" "1", evidence 1 "reply" "2"] == some [2]

-- Only occurrences an earlier admitted step retained are bound: a future ordinal has no value, an
-- occurrence belongs to the operation that retained it, and a step supplying no value has none.
#guard rejection (contract #[clause 1 (requirement := some (correlation 1))])
  [evidence 0 "request" "1", evidence 1 "reply" "1"] == some "missing retained capture occurrence"
#guard rejection (contract #[clause 1])
  [evidence 0 "request" "1", evidence 1 "reply" "1" "b"] == some "missing retained capture occurrence"

-- Repeated triggers retain independent occurrences: ordinal zero keeps the value it was admitted
-- with rather than being replaced by the latest match, and ordinal one is the second occurrence.
#guard answers (contract #[clause 2])
  [evidence 0 "request" "1", evidence 1 "request" "2", evidence 2 "reply" "1"] == some [2]
#guard rejection (contract #[clause 2])
  [evidence 0 "request" "1", evidence 1 "request" "2", evidence 2 "reply" "2"] ==
  some "correlation rejected this operation's step"
#guard answers (contract #[clause 2 (requirement := some (correlation 1))])
  [evidence 0 "request" "1", evidence 1 "request" "2", evidence 2 "reply" "2"] == some [2]

-- Retained occurrences never move a countdown: the clause still violates its inclusive deadline,
-- and an unresolved prefix stays unresolved until a deliberate close decides it.
#guard answers (contract #[clause 0]) [evidence 0 "request" "1", evidence 1 "reply" "1"] == some [3]
#guard answers (contract #[clause 1]) [evidence 0 "request" "1"] == some [0]
#guard answers (contract #[clause 1 (endpoint := .SCOPED_ENDPOINT_DELIBERATELY_CLOSED)])
  [evidence 0 "request" "1"] == some [3]
#guard answers (contract #[clause 1]) [evidence 0 "request" "1", evidence 1 "reply" "1"] true ==
  some [0]

-- Declared ceilings reject atomically rather than retaining beyond what was declared.
#guard rejection (contract #[clause 2] (limits := budget (captures := 1)))
  [evidence 0 "request" "1", evidence 1 "request" "2"] == some "captures exhausted"
#guard rejection (contract #[clause 2 (captures := #[capture (lifetime := 1)])])
  [evidence 0 "request" "1", evidence 1 "request" "2"] == some "capture lifetime exhausted"

/-- Decoding alone rejects a capability whose declarations could never bind. -/
private def decodeError (wire : ScopedContract) : Option String :=
  match Testpilot.Scoped.decode wire with
  | .error reason => some reason
  | .ok _ => none
private def decodes (wire : ScopedContract) : Bool := (Testpilot.Scoped.decode wire).isOk

#guard decodes (contract #[clause 1])
#guard decodeError (contract #[clause 1 (captures := #[capture (field := "absent")])]) ==
  some "capture names an unretained evidence field"
#guard decodeError (contract #[clause 1 (captures := #[capture (lifetime := 0)])]) ==
  some "nonpositive resource limit"
#guard decodeError (contract #[clause 1 (requirement := some (correlation 2))]) ==
  some "capture ordinal beyond declared lifetime"
#guard decodeError (contract #[clause 1 (requirement := some
  (comparison (field "replied") (retained 0 "other")))]) == some "unbound capture reference"
#guard decodeError (contract #[clause 1 (requirement := some
  (comparison (field "absent") (retained 0)))]) == some "unretained correlation field operand"
#guard decodeError (contract #[clause 1 (requirement := some (anyOf #[]))]) ==
  some "empty correlation group"
#guard decodeError (contract #[clause 1] (limits := budget (depth := 1))) ==
  some "correlation depth exhausted"
#guard decodeError (contract #[clause 1] (limits := budget (captures := 0))) ==
  some "nonpositive resource limit"
#guard decodes (contract #[clause 1 (requirement := some
  (comparison (literal "1") (retained 0)))])

-- A field the projection redacts carries no value, so it can supply neither a capture nor a
-- correlation operand.
#guard decodeError (contract #[clause 1]
  (rules := #[requestRule #[policy "requested" .SCOPED_FIELD_DISPOSITION_REDACT], replyRule])) ==
  some "capture names an unretained evidence field"

-- Nested groups compose under the declared depth, and `all` stops at its first false operand, so
-- the capture operand a false conjunct made irrelevant is never read.
#guard answers (contract #[clause 1 (requirement := some
  (anyOf #[triggered, allOf #[comparison (field "replied") (literal "1"),
    comparison (field "replied") (retained 0)]]))])
  [evidence 0 "request" "1", evidence 1 "reply" "1"] == some [2]
#guard rejection (contract #[clause 1 (requirement := some
  (anyOf #[triggered, allOf #[comparison (field "replied") (literal "9"),
    comparison (field "replied") (retained 1)]]))])
  [evidence 0 "request" "1", evidence 1 "reply" "1"] ==
  some "correlation rejected this operation's step"

-- Missing evidence never satisfies a comparison: a step that declares no field at all cannot bind
-- the operand its correlation reads.
#guard rejection (contract #[clause 1])
  [evidence 0 "request" "1", { evidence 1 "silent" "" with fields := #[] }] ==
  some "missing correlation field operand"

-- A malformed or incomplete evidence value keeps its declared meaning: the codec admits only the
-- exact scalar forms this capability declares, and never approximates one.
private def valued (wire : temporal.server.api.testpilot.v1.Value) : ScopedEvidence :=
  { evidence 0 "request" "1" with
    fields := #[{ field_id := "requested", value := some wire }] }
#guard rejection (contract #[clause 1]) [valued { value := some (.natural "01") }] ==
  some "noncanonical natural"
#guard rejection (contract #[clause 1]) [valued { value := some (.bytes_value ⟨#[1]⟩) }] ==
  some "unsupported evidence scalar"
#guard rejection (contract #[clause 1]) [valued { value := some (.natural "1") }] ==
  some "invalid evidence"
#guard rejection (contract #[clause 1])
  [{ evidence 0 "request" "1" with fields := #[] }] == some "invalid evidence"

-- A retained occurrence is named by its capture id and ordinal alone, so capture identities are one
-- namespace across the capability: two clauses declaring one id would alias the same stream.
#guard decodeError (contract #[clause 1, clause 1 (clauseId := "second")]) ==
  some "invalid capture identities"
#guard decodes (contract #[clause 1,
  clause 1 (captures := #[capture (id := "other")])
    (requirement := some (comparison (field "replied") (retained 0 "other"))) (clauseId := "second")])
-- A correlation operand may only name a capture its own clause declared.
#guard decodeError (contract #[clause 1 (captures := #[capture (id := "other")]),
  clause 1 (captures := #[]) (requirement := some (comparison (field "replied") (retained 0 "other")))
    (clauseId := "second")]) == some "unbound capture reference"

-- Comparison operands must share one declared scalar kind: a value of a different kind is rejected
-- rather than compared and found unequal.
#guard decodeError (contract #[clause 1 (requirement := some
  (comparison (field "replied") (natural "1")))]) == some "incompatible correlation operand types"
private def naturalReply : ScopedFieldPolicy :=
  { policy "replied" with type := some { kind := .SCALAR_KIND_NATURAL } }
#guard decodeError (contract #[clause 1 (requirement := some
  (comparison (field "replied") (retained 0)))]
  (rules := #[requestRule, rule "reply" reply responded #[naturalReply], silentRule])) ==
  some "incompatible correlation operand types"

-- A field two rules declare at different kinds has no single type to check against.
#guard decodeError (contract #[clause 1]
  (rules := #[requestRule,
    rule "reply" reply responded #[{ policy "requested" with
      type := some { kind := .SCALAR_KIND_NATURAL } }], silentRule])) ==
  some "ambiguous retained field type"

-- A capability that declares neither captures nor a correlation keeps its exact prior meaning, and
-- leaves both new ceilings unset.
private def bare : ScopedContract :=
  contract #[clause 1 (captures := #[]) (requirement := none)]
    (limits := { budget with max_captures := 0, max_correlation_depth := 0 })
#guard answers bare [evidence 0 "request" "1", evidence 1 "reply" "2"] == some [2]
#guard (Testpilot.Scoped.decode bare).toOption.map (fun compiled => compiled.captures == 0 &&
  compiled.keyed == [("response", ⟨[], none⟩)]) == some true

end Testpilot.Tests.Fields

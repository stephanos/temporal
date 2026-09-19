import Testpilot.Correlated

/-!
Portable field and capture evaluation for the version-one correlated capability.

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
private def value (definitionId text : String) : ModelValue :=
  { definition_id := definitionId, value := text }

private def state := value "state" "ready"
private def request := value "action" "request"
private def reply := value "action" "reply"
private def quiet := value "outcome" "quiet"
private def responded := value "outcome" "response"

private def transition (action : ModelValue) (outcome : ModelValue) : CorrelatedTransition := {
  prior_state := some state, action := some action
  state := some state, outcome := some outcome }
private def output (action : ModelValue) (outcome : ModelValue) : CorrelatedTransition :=
  { transition action outcome with prior_state := none }

/-- Each evidence kind declares its own field, exactly as the two distinct operand paths a model
correlation names would be projected. -/
private def policy (fieldId : String)
    (disposition := CorrelatedFieldDisposition.CORRELATED_FIELD_DISPOSITION_RETAIN) : CorrelatedFieldPolicy :=
  { field_id := fieldId, type := some { kind := .SCALAR_KIND_TEXT }, disposition }

private def rule (kind : String) (action outcome : ModelValue) (fields : Array CorrelatedFieldPolicy) :
    CorrelatedProjectionRule := {
  kind, meaning := .CORRELATED_EVIDENCE_MEANING_CONFIRMED
  outputs := #[output action outcome], fields }

private def requestRule (fields : Array CorrelatedFieldPolicy := #[policy "requested"]) :=
  rule "request" request quiet fields
private def replyRule := rule "reply" reply responded #[policy "replied"]
/-- A reply-shaped step that declares no fields at all, so a correlation reading one finds none. -/
private def silentRule := rule "silent" reply responded #[]

private def capture (lifetime : Nat := 2) (field := "requested") (id := "seen") :
    CorrelatedCaptureDeclaration :=
  { capture_id := id, field_id := field, lifetime := number lifetime }

private def reference (arm : Reference.reference_Type) : Expression :=
  { expression := some (.reference { reference := some arm }) }
private def literal (text : String) : Expression :=
  { expression := some (.literal { value := some (.text_value text) }) }
private def field (id : String) : Expression := reference (.evidence_field_id id)
private def unsigned (text : String) : Expression :=
  { expression := some (.literal { value := some (.unsigned_integer_value text) }) }
private def retained (ordinal : Nat) (id := "seen") : Expression :=
  reference (.correlated_capture { capture_id := id, ordinal := number ordinal })

private def comparison (left right : Expression)
    (operator := ComparisonOperator.COMPARISON_OPERATOR_EQUAL) : Expression :=
  { expression := some (.compare { operator, left := some left, right := some right }) }
/-- The step condition matching a step that carries exactly `text` of `definitionId` at `field`. -/
private def step (field : CorrelatedStepField) (definitionId text : String) : Expression :=
  comparison (reference (.correlated_step { field, definition_id := definitionId })) (literal text)
private def triggered : Expression := step .CORRELATED_STEP_FIELD_ACTION "action" "request"
private def anyOf (operands : Array Expression) : Expression :=
  { expression := some (.any { operands }) }
private def allOf (operands : Array Expression) : Expression :=
  { expression := some (.all { operands }) }

/-- The request step creates the occurrence its own correlation would read, so the trigger disjunct
decides it before the capture operand is reached. -/
private def correlation (ordinal : Nat := 0) : Expression :=
  anyOf #[triggered, comparison (field "replied") (retained ordinal)]

private def clause (bound : Nat) (captures : Array CorrelatedCaptureDeclaration := #[capture])
    (requirement : Option Expression := some (correlation))
    (ending := TraceEnding.TRACE_ENDING_PARTIAL) (ruleId := "response") :
    CorrelatedRule := {
  rule_id := ruleId, clock := .CORRELATED_CLOCK_OPERATION_TRANSITIONS
  bound := number bound, ending
  trigger := some triggered
  response := some (step .CORRELATED_STEP_FIELD_OUTCOME "outcome" "response")
  captures, correlation := requirement }

private def budget (captures : Nat := 8) (depth : Nat := 4) : CorrelatedLimits := {
  max_events := number 16, max_buffered := number 8, max_keys := number 8
  max_support := number 256, max_projection_work := number 1000000
  max_event_bytes := number 512, max_semantic_transitions := number 32
  max_obligations := number 16, max_obligation_work := number 1000000000
  max_captures := number captures, max_correlation_depth := number depth }

private def contract (correlatedRules : Array CorrelatedRule)
    (rules : Array CorrelatedProjectionRule := #[requestRule, replyRule, silentRule]) :
    CorrelatedContract := {
  projection_id := "projection", projection_fingerprint := "fingerprint"
  evidence_observation_id := "evidence", scope_fields := #["run"], operation_field := "operation"
  sources := #["source"], initial_state := some state
  transitions := #[transition request quiet, transition reply responded]
  projection_rules := rules, rules := correlatedRules }

private def evidence (ordinal : Nat) (kind : String) (count : String) (operation := "a")
    (fieldId := if kind == "request" then "requested" else "replied") : CorrelatedEvidence := {
  identity := some {
    scope := #[{ field_id := "run", value := some { value := some (.text_value "run-1") } }], evidence_source := "source"
    ordinal := number ordinal }
  operation, kind
  fields := #[{ field_id := fieldId, value := some { value := some (.text_value count) } }] }

private def scope : List (Name × String) := [(⟨"run"⟩, "run-1")]

/-- Replay a stream offline under the correlated ceilings `limits` and report the clause answer, or
the exact first rejection. -/
private def replay (wire : CorrelatedContract) (events : List CorrelatedEvidence)
    (incomplete : Bool := false) (limits : CorrelatedLimits := budget) : Except String (List Nat) := do
  let compiled ← Testpilot.Correlated.decode limits wire
  let initial ← compiled.start scope
  let run ← events.zipIdx.foldlM (fun run (event, index) => run.observe (index + 1) event) initial
  pure (run.close.answers incomplete)

private def answers (wire : CorrelatedContract) (events : List CorrelatedEvidence)
    (incomplete : Bool := false) (limits : CorrelatedLimits := budget) : Option (List Nat) :=
  (replay wire events incomplete limits).toOption
private def rejection (wire : CorrelatedContract) (events : List CorrelatedEvidence)
    (limits : CorrelatedLimits := budget) : Option String :=
  match replay wire events (limits := limits) with
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
-- occurrence belongs to the operation that retained it, and a step supplying no value has none. A
-- comparison with an absent operand is false, so the step is not this operation's reply.
#guard rejection (contract #[clause 1 (requirement := some (correlation 1))])
  [evidence 0 "request" "1", evidence 1 "reply" "1"] == some "correlation rejected this operation's step"
#guard rejection (contract #[clause 1])
  [evidence 0 "request" "1", evidence 1 "reply" "1" "b"] ==
  some "correlation rejected this operation's step"

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
#guard answers (contract #[clause 1 (ending := .TRACE_ENDING_FINAL)])
  [evidence 0 "request" "1"] == some [3]
#guard answers (contract #[clause 1]) [evidence 0 "request" "1", evidence 1 "reply" "1"] true ==
  some [0]

-- Declared ceilings reject atomically rather than retaining beyond what was declared.
#guard rejection (contract #[clause 2])
  [evidence 0 "request" "1", evidence 1 "request" "2"] (limits := budget (captures := 1)) ==
  some "captures exhausted"
#guard rejection (contract #[clause 2 (captures := #[capture (lifetime := 1)])])
  [evidence 0 "request" "1", evidence 1 "request" "2"] == some "capture lifetime exhausted"

/-- Decoding alone rejects a capability whose declarations could never bind. -/
private def decodeError (wire : CorrelatedContract) (limits : CorrelatedLimits := budget) :
    Option String :=
  match Testpilot.Correlated.decode limits wire with
  | .error reason => some reason
  | .ok _ => none
private def decodes (wire : CorrelatedContract) : Bool := (Testpilot.Correlated.decode budget wire).isOk

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
#guard decodeError (contract #[clause 1]) (limits := budget (depth := 1)) ==
  some "correlation depth exhausted"

-- A correlated condition shares the one expression language, so a reference that belongs to another
-- context rejects wherever it appears, and a trigger reads only the step's action.
#guard decodeError (contract #[clause 1 (requirement := some
  (comparison (field "replied") (reference (.slot_id "slot"))))]) ==
  some "reference is not admitted in this expression context"
private def presentProjected : Expression :=
  { expression := some (.present { operand := some (reference (.projected_value default)) }) }
#guard decodeError (contract #[{ clause 1 with trigger := some presentProjected }]) ==
  some "reference is not admitted in this expression context"
#guard decodeError (contract #[{ clause 1 with
    trigger := some (step .CORRELATED_STEP_FIELD_OUTCOME "outcome" "response") }]) ==
  some "unsupported clause"
#guard decodeError (contract #[clause 1]) (limits := budget (captures := 0)) ==
  some "nonpositive resource limit"
#guard decodes (contract #[clause 1 (requirement := some
  (comparison (literal "1") (retained 0)))])

-- A field the projection redacts carries no value, so it can supply neither a capture nor a
-- correlation operand.
#guard decodeError (contract #[clause 1]
  (rules := #[requestRule #[policy "requested" .CORRELATED_FIELD_DISPOSITION_REDACT], replyRule])) ==
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

-- Missing evidence never satisfies a comparison: a step that declares no field at all compares
-- false under either operator, while a disjunction can still admit it through another operand.
#guard rejection (contract #[clause 1])
  [evidence 0 "request" "1", { evidence 1 "silent" "" with fields := #[] }] ==
  some "correlation rejected this operation's step"
#guard rejection (contract #[clause 1 (requirement := some
  (anyOf #[triggered, comparison (field "replied") (retained 1) .COMPARISON_OPERATOR_NOT_EQUAL]))])
  [evidence 0 "request" "1", evidence 1 "reply" "2"] ==
  some "correlation rejected this operation's step"
#guard answers (contract #[clause 1 (requirement := some
  (anyOf #[triggered, comparison (field "replied") (retained 1),
    comparison (field "replied") (retained 0)]))])
  [evidence 0 "request" "1", evidence 1 "reply" "1"] == some [2]

-- A malformed or incomplete evidence value keeps its declared meaning: the codec admits only the
-- exact scalar forms this capability declares, and never approximates one.
private def valued (wire : temporal.server.api.testpilot.v1.Value) : CorrelatedEvidence :=
  { evidence 0 "request" "1" with
    fields := #[{ field_id := "requested", value := some wire }] }
#guard rejection (contract #[clause 1]) [valued { value := some (.unsigned_integer_value "01") }] ==
  some "noncanonical unsigned integer"
#guard rejection (contract #[clause 1])
  [valued { value := some (.unsigned_integer_value "18446744073709551616") }] ==
  some "unsigned integer overflow"
#guard rejection (contract #[clause 1]) [valued { value := some (.bytes_value ⟨#[1]⟩) }] ==
  some "unsupported evidence scalar"
#guard rejection (contract #[clause 1]) [valued { value := some (.unsigned_integer_value "1") }] ==
  some "invalid evidence"
#guard rejection (contract #[clause 1])
  [{ evidence 0 "request" "1" with fields := #[] }] == some "invalid evidence"

-- A retained occurrence is named by its capture id and ordinal alone, so capture identities are one
-- namespace across the capability: two clauses declaring one id would alias the same stream.
#guard decodeError (contract #[clause 1, clause 1 (ruleId := "second")]) ==
  some "invalid capture identities"
#guard decodes (contract #[clause 1,
  clause 1 (captures := #[capture (id := "other")])
    (requirement := some (comparison (field "replied") (retained 0 "other"))) (ruleId := "second")])
-- A correlation operand may only name a capture its own clause declared.
#guard decodeError (contract #[clause 1 (captures := #[capture (id := "other")]),
  clause 1 (captures := #[]) (requirement := some (comparison (field "replied") (retained 0 "other")))
    (ruleId := "second")]) == some "unbound capture reference"

-- Comparison operands must share one declared scalar kind: a value of a different kind is rejected
-- rather than compared and found unequal.
#guard decodeError (contract #[clause 1 (requirement := some
  (comparison (field "replied") (unsigned "1")))]) == some "incompatible correlation operand types"
private def unsignedReply : CorrelatedFieldPolicy :=
  { policy "replied" with type := some { kind := .SCALAR_KIND_UINT64 } }
#guard decodeError (contract #[clause 1 (requirement := some
  (comparison (field "replied") (retained 0)))]
  (rules := #[requestRule, rule "reply" reply responded #[unsignedReply], silentRule])) ==
  some "incompatible correlation operand types"

-- A field two rules declare at different kinds has no single type to check against.
#guard decodeError (contract #[clause 1]
  (rules := #[requestRule,
    rule "reply" reply responded #[{ policy "requested" with
      type := some { kind := .SCALAR_KIND_UINT64 } }], silentRule])) ==
  some "ambiguous retained field type"

-- A capability that declares neither captures nor a correlation keeps its exact prior meaning under
-- a Profile that leaves both capture ceilings unset.
private def bare : CorrelatedContract :=
  contract #[clause 1 (captures := #[]) (requirement := none)]
private def bareBudget : CorrelatedLimits := { budget with max_captures := 0, max_correlation_depth := 0 }
#guard answers bare [evidence 0 "request" "1", evidence 1 "reply" "2"] (limits := bareBudget) == some [2]
#guard (Testpilot.Correlated.decode bareBudget bare).toOption.map (fun compiled => compiled.captures == 0 &&
  compiled.keyed == [("response", ⟨[], none⟩)]) == some true

/-! ### A STATE condition reads the state and every field the machine keeps

The Go evaluator reads `state` and `state_fields` off the admitted transition;
`Shared.CorrelatedObligation.Predicate.holds` reads the same two, and this is the claim that says
so on one worked transition. A field is read as itself, so the wrong member of the right field is
false rather than a substring of the state's own spelling. -/

private def atom (definitionId spelling : String) : Atom := ⟨Name.mk definitionId, spelling⟩

/-- One step of a machine whose state is a phase and an attempt count. -/
private def backingOff : Shared.SemanticData.Result StateValue Atom Atom :=
  ⟨atom "outcome" "accepted",
    ⟨atom "state" "backingOff-1", [atom "phase" "backingOff", atom "attempts" "1"]⟩,
    [atom "fact" "pendingAttempts"]⟩

/-- The same step of a Model whose states are atoms: no fields, and every reference reads as it
did before a machine had any. -/
private def atomOnly : Shared.SemanticData.Result StateValue Atom Atom :=
  ⟨atom "outcome" "accepted", ⟨atom "state" "backingOff-1", []⟩, [atom "fact" "pendingAttempts"]⟩

private def stateCondition (definitionId : String) (equalsText : Option String) :
    Shared.CorrelatedObligation.Predicate :=
  ⟨3, Name.mk definitionId, equalsText⟩

private def taken : Atom := atom "action" "transportFault"

#guard (stateCondition "state" (some "backingOff-1")).holds taken backingOff
#guard (stateCondition "phase" (some "backingOff")).holds taken backingOff
#guard (stateCondition "attempts" (some "1")).holds taken backingOff
#guard (stateCondition "attempts" none).holds taken backingOff

#guard !(stateCondition "phase" (some "scheduled")).holds taken backingOff
#guard !(stateCondition "attempts" (some "0")).holds taken backingOff
#guard !(stateCondition "cancel" none).holds taken backingOff

#guard (stateCondition "state" (some "backingOff-1")).holds taken atomOnly
#guard !(stateCondition "phase" (some "backingOff")).holds taken atomOnly

/- A field is not a fact and a fact is not a field: the two are read under different step fields, so
neither answers the other's reference. -/
#guard !(Shared.CorrelatedObligation.Predicate.mk 4 (Name.mk "phase") none).holds taken backingOff
#guard !(stateCondition "fact" none).holds taken backingOff

end Testpilot.Tests.Fields

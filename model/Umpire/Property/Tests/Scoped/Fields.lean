import Umpire.Property.Scoped
import Umpire.Observation.Evaluation.Scoped
import Umpire.Model.Table
import Umpire.Property.Elab
import Umpire.Operation.Action
import Umpire.Value.Field
import Umpire.Shared.Test

/-!
Keyed field captures composed with the existing scoped bounded-response obligations.

The Target's actions are the model payloads of checked request projections, so a step's typed
`count` field is exactly the field evidence its clause reads. One declared capture retains the
count of every request occurrence; the clause's correlation requires a reply to carry the count of
the request occurrence it names. Retained captures never change a countdown: they decide which
labeled transitions are the operation's semantic steps at all.
-/

namespace Umpire.Property.ScopedFieldTests

open Umpire Operation Value

private def id := DefinitionId.of
private def source : SourceLocation := { path := "Umpire/Property/Tests/Scoped/Fields.lean" }

private def schema : Schema := ⟨"M", [{
  name := "M", protoSyntax := "proto3", descriptor := "m", fileContext := "", references := [],
  valueShape := some (.message [
    ⟨1, "count", .integer .int32, .singular, .implicit (.integer .int32 0), none⟩,
    ⟨2, "label", .text, .singular, .optional, none⟩,
    ⟨3, "choice", .text, .singular, .oneof "pick", none⟩]) }]⟩
private def owner : RpcOwner where
  Witness _ _ := Unit
  schema _ := ⟨"example.Call", schema, schema, [], false, false⟩
private def valueLimits : Value.Limits := ⟨8, 10000, 1024, 100⟩

private def template (identity : DefinitionId) := do
  let binding ← checkRpc owner (Request := Unit) (Response := Unit) ()
    (owner.schema (Request := Unit) (Response := Unit) ())
  ActionTemplate.check (rpc Empty binding) identity

/-- A request operand must denote the very arguments of the selected Action, so every projection
here is built through the Action constructor rather than a bare cursor. -/
private def requestProjection (identity : DefinitionId)
    (value : Checked owner (Request := Unit) (Response := Unit) () .request valueLimits)
    (cursor : Field.Cursor owner (Request := Unit) (Response := Unit) () .request valueLimits
      (.integer .int32) .singular .available) :
    Except Field.Error (PropertyFieldProjection owner (Request := Unit) (Response := Unit) ()) := do
  if same : cursor.origin.value = value.value then
    let selected ← (template identity).mapError fun _ => Field.Error.mk source "M" "template"
    have witness : selected.declaration.reference = () := by
      change (selected.declaration.reference : Unit) = ()
      exact Subsingleton.elim (α := Unit) selected.declaration.reference ()
    let arguments : Checked owner selected.declaration.reference .request valueLimits :=
      witness.symm ▸ value
    let field : Field.Cursor owner selected.declaration.reference .request valueLimits
        (.integer .int32) .singular .available := witness.symm ▸ cursor
    let projected ← PropertyFieldProjection.ofAction (ActionInstance.mk (template := selected) arguments)
      field (by simpa [field, arguments] using same) source
    pure (witness ▸ projected)
  else throw ⟨source, "M", "wrong arguments"⟩

private def projection (identity : DefinitionId) (count : Int) :
    Except Field.Error (PropertyFieldProjection owner (Request := Unit) (Response := Unit) ()) := do
  let value ← (Value.check owner (Request := Unit) (Response := Unit) () .request valueLimits
    (message "M" [(1, literal (.integer .int32 count))])).mapError
    fun error => Field.Error.mk source error.path error.reason
  let reference ← Field.reference owner () .request "M" 1 source
  let cursor ← (Field.root value).field reference source
  let cursor ← cursor.refine (.integer .int32) .singular .available source
  requestProjection identity value cursor

private def trigger := id "test.trigger"
private def replied := id "test.reply"

#guard ([trigger, replied] : List DefinitionId).all fun identity =>
  ([1, 2] : List Int).all fun count => (projection identity count).isOk

/-- The distinct fallback keeps a broken fixture visible instead of collapsing two counts onto one
model payload; the guard above proves every projection this module uses is admitted. -/
private def payload (identity : DefinitionId) (count : Int) : ModelValue :=
  ((projection identity count).toOption.map PropertyFieldProjection.modelValue).getD
    ⟨identity, "unavailable-" ++ toString count⟩
private def evidenceAt (identity : DefinitionId) (count : Int) : List PropertyFieldEvidence :=
  ((projection identity count).toOption.map PropertyFieldProjection.evidence).toList

private def value (key payload : String) : ModelValue := ⟨id key, payload⟩
private def state := value "test.state" "ready"
private def quiet := value "test.outcome" "quiet"
private def response := value "test.outcome" "response"
private def result (responded : Bool) : Step ModelValue ModelValue ModelValue := {
  state := state
  outcome := if responded then response else quiet
  facts := []
}

private def kinds : List (String × DefinitionKind) := [
  ("test.target", .target), ("test.kernel", .machine), ("test.provider", .provider),
  ("test.capability", .capability), ("test.state", .state), ("test.trigger", .action),
  ("test.reply", .action), ("test.outcome", .outcome)]
private def definitions : List DefinitionMetadata := kinds.map fun (name, kind) =>
  Shared.Test.definitionMetadata name kind source (name ++ "/v1")
private def provider : Provider (fun _ => True) := {
  id := id "test.provider"
  source
  contract := { id := id "test.capability", behaviorVersion := "scoped-fields-test/v1", requiredLaws := [] }
  meanings := (kinds.drop 4).map fun (name, kind) =>
    { definitionId := id name, kind, behaviorVersion := name ++ "/meaning-v1" }
  lawProofs := []
}

private def counts : List Int := [1, 2]
private def table : FiniteTable Unit ModelValue ModelValue ModelValue ModelValue := {
  setups := [⟨(), "default"⟩]
  states := [⟨state, "ready"⟩]
  actions := counts.flatMap (fun count =>
    [⟨payload trigger count, "request-" ++ toString count⟩,
      ⟨payload replied count, "reply-" ++ toString count⟩])
  outcomes := [⟨quiet, "quiet"⟩, ⟨response, "response"⟩]
  facts := []
  initial := [⟨(), [state]⟩]
  transitions := counts.flatMap (fun count =>
    [⟨"request-" ++ toString count, state, payload trigger count, [result false]⟩,
      ⟨"reply-" ++ toString count, state, payload replied count, [result true]⟩])
}

private def targetResult := table.checkTypedModel {
  id := id "test.target"
  source
  definitions
  requiredCapabilities := [id "test.capability"]
  metadata := { id := id "test.kernel", source }
} (Providers.empty.provide provider)

#guard targetResult.isOk

private abbrev TestTarget := CheckedModel (fun _ => True) Unit ModelValue ModelValue ModelValue ModelValue
private def bindings : List PropertyFieldBinding :=
  [trigger, replied].map (PropertyFieldBinding.ofWitness owner (Request := Unit) (Response := Unit) ())
private def context (target : TestTarget) : PropertyCheckContext :=
  { PropertyCheckContext.ofTarget target with fieldBindings := bindings }

private def path (root : PropertyFieldRoot) (reference : DefinitionId) : PropertyFieldPath :=
  { root, reference, schema := owner.schema (Request := Unit) (Response := Unit) (),
    side := .request, steps := [.field "M" 1], type := .integer .int32 }
private def triggerPath := path .request trigger
private def replyPath := path .request replied
private def captureName := id "test.capture.count"
private def capturedPath (ordinal : Nat) (name : DefinitionId := captureName) : PropertyFieldPath :=
  { triggerPath with capture := some ⟨name, ordinal⟩ }

private def capture (lifetime : Nat := 4) (key : DefinitionId := id "test.operation") :
    PropertyScopedCapture := { name := captureName, key, path := triggerPath, lifetime }

/-- A reply belongs to this operation only when its own `count` equals the captured request
occurrence. The request disjunct short-circuits before the capture operand is read, so the step
that creates occurrence zero never has to bind it. -/
private def correlation (ordinal : Nat := 0) (name : DefinitionId := captureName) : PropertyPredicate :=
  .any [.atom { field := .selectedAction, reference := trigger },
    PropertyPredicate.compareFields .equal (.field replyPath source)
      (.field (capturedPath ordinal name) source) source]

private def clause (bound : Nat) (endpoint : PropertyScopedEndpoint := .runtimePrefix)
    (captures : List PropertyScopedCapture := [capture])
    (requirement : Option PropertyPredicate := some (correlation)) : PropertyScopedClause := {
  id := id "test.scoped.fields"
  source
  trigger := .atom { field := .selectedAction, reference := trigger }
  response := .atom {
    field := .modelOutcome
    reference := id "test.outcome"
    constraint := .equals (.text "response") }
  scope := [id "test.run"]
  key := id "test.operation"
  clock := .operationTransitions
  bound
  endpoint
  captures
  correlation := requirement
}

private def declaration (temporal : PropertyScopedClause) : Property := {
  id := id "test.property.fields"
  source
  requires := [id "test.capability"]
  clauses := []
  scopedClauses := [temporal]
}

private def runLimits : Scoped.Limits :=
  { transitions := 1000, obligations := 1000, work := 1000000, captures := 1000 }
private def scope : List (DefinitionId × String) := [(id "test.run", "run-1")]

private def step (operation : String) (identity : DefinitionId) (count : Int) :
    Scoped.Transition × List PropertyFieldEvidence :=
  ({ scope
     operationField := id "test.operation"
     operation
     priorState := state
     action := payload identity count
     result := result (identity == replied) }, evidenceAt identity count)
private def request (operation : String) (count : Int) := step operation trigger count
private def reply (operation : String) (count : Int) := step operation replied count

/-- Runtime-prefix and deliberately-closed answers of one admitted stream. -/
private structure Outcome where
  live : List PropertyEndpointAnswer
  closed : List PropertyEndpointAnswer
  deriving BEq, DecidableEq, Repr

/-- Consume the stream in the given chunks, so a split inside unresolved evidence is observable. -/
private def runChunks (temporal : PropertyScopedClause)
    (chunks : List (List (Scoped.Transition × List PropertyFieldEvidence)))
    (budget : Scoped.Limits := runLimits) : Except Scoped.Error Outcome :=
  match targetResult with
  | .error _ => throw Scoped.Error.invalidInitialState
  | .ok target => do
      let property ← (Property.check (context target) ((declaration temporal))).mapError
        Scoped.Error.property
      let compiled ← Scoped.compile target property [id "test.run"] (id "test.operation") budget
      let initial ← compiled.start () state scope
      let consumed ← chunks.foldlM (fun run chunk => run.consumeEvidence chunk) initial
      pure ⟨consumed.answers.map Prod.snd, consumed.close.answers.map Prod.snd⟩

private def run (temporal : PropertyScopedClause)
    (steps : List (Scoped.Transition × List PropertyFieldEvidence))
    (budget : Scoped.Limits := runLimits) : Except Scoped.Error Outcome :=
  runChunks temporal [steps] budget

private def answer (temporal : PropertyScopedClause)
    (steps : List (Scoped.Transition × List PropertyFieldEvidence))
    (budget : Scoped.Limits := runLimits) : Option PropertyEndpointAnswer :=
  (run temporal steps budget).toOption.bind (·.closed.head?)

private def error? (temporal : PropertyScopedClause)
    (steps : List (Scoped.Transition × List PropertyFieldEvidence))
    (budget : Scoped.Limits := runLimits) : Option Scoped.Error :=
  match run temporal steps budget with
  | .ok _ => none
  | .error failure => some failure

/-- Drop a step's evidence without changing the labeled transition it reports. -/
private def unwitnessed (step : Scoped.Transition × List PropertyFieldEvidence) :
    Scoped.Transition × List PropertyFieldEvidence := (step.1, [])

private def errorKind? (temporal : PropertyScopedClause)
    (steps : List (Scoped.Transition × List PropertyFieldEvidence))
    (budget : Scoped.Limits := runLimits) : Option PropertyErrorKind :=
  match error? temporal steps budget with
  | some (.property failure) => some failure.kind
  | _ => none

-- A reply whose captured count matches the request occurrence is one of the operation's semantic
-- steps and answers its bounded window; a mismatched count is not this operation's reply at all.
#guard answer (clause 1) [request "a" 1, reply "a" 1] == some .satisfied
#guard error? (clause 1) [request "a" 1, reply "a" 2] == some (.invalidTransition "a")
#guard answer (clause 1 (requirement := none)) [request "a" 1, reply "a" 2] == some .satisfied

-- Only occurrences an earlier admitted step retained are bound. Ordinal one has not occurred, this
-- operation never retained operation "a"'s occurrence, and a step with no evidence supplies none.
#guard errorKind? (clause 1 (requirement := some (correlation 1))) [request "a" 1, reply "a" 1] ==
  some .missingPredicateInput
#guard errorKind? (clause 1) [request "a" 1, reply "b" 1] == some .missingPredicateInput
#guard errorKind? (clause 1) [request "a" 1, unwitnessed (reply "a" 1)] == some .missingPredicateInput

-- A correlation may only name a capture its own clause declared, under its own operation key.
#guard errorKind? (clause 1 (captures := [])) [request "a" 1] == some .unsupportedPredicateInput
#guard errorKind? (clause 1 (captures := [capture (key := id "test.run")])) [request "a" 1] ==
  some .invalidClause
#guard errorKind? (clause 1 (captures := [capture (lifetime := 0)])) [request "a" 1] ==
  some .invalidClause
#guard errorKind? (clause 1 (requirement := some (correlation 0 (id "test.other"))))
  [request "a" 1] == some .unsupportedPredicateInput

-- A capture operand must also name the exact coordinates its declaration retains and an ordinal
-- that declaration keeps; neither could ever bind, so both reject before any evidence is admitted.
private def strayCapture : PropertyPredicate :=
  PropertyPredicate.compareFields .equal (.field replyPath source)
    (.field { replyPath with capture := some ⟨captureName, 0⟩ } source) source
#guard errorKind? (clause 1 (requirement := some strayCapture)) [request "a" 1] ==
  some .unsupportedPredicateInput
#guard errorKind? (clause 1 (captures := [capture (lifetime := 1)])
  (requirement := some (correlation 1))) [request "a" 1] == some .unsupportedPredicateInput

-- Two projections at one capture's exact declared coordinates leave the occurrence ambiguous.
#guard error? (clause 1) [((request "a" 1).1, evidenceAt trigger 1 ++ evidenceAt trigger 1)] ==
  some (.capture (.ambiguous captureName))

-- Repeated triggers retain independent occurrences: ordinal zero keeps the value it was admitted
-- with rather than being replaced by the latest match, and ordinal one is the second occurrence.
#guard answer (clause 2) [request "a" 1, request "a" 2, reply "a" 1] == some .satisfied
#guard error? (clause 2) [request "a" 1, request "a" 2, reply "a" 2] == some (.invalidTransition "a")
#guard answer (clause 2 (requirement := some (correlation 1)))
  [request "a" 1, request "a" 2, reply "a" 2] == some .satisfied

-- Matching responses keep the original inclusive deadline and endpoint semantics.
#guard answer (clause 0) [request "a" 1, reply "a" 1] == some .violated
#guard (run (clause 1) [request "a" 1]).toOption == some ⟨[.unresolved], [.unresolved]⟩
#guard (run (clause 1 .deliberatelyClosed) [request "a" 1]).toOption ==
  some ⟨[.unresolved], [.violated]⟩

-- A rejected capture or exhausted budget publishes no state, and cannot repair a proved violation.
#guard error? (clause 1) [request "a" 1, request "a" 2] { runLimits with captures := 1 } ==
  some .capturesExhausted
#guard error? (clause 1 (captures := [capture (lifetime := 1)])) [request "a" 1, request "a" 2] ==
  some (.capture (.exhausted captureName))
/-- Whether a rejected append leaves the established violation and the retained captures intact. -/
private def rejectionPreserves (rejected : List (Scoped.Transition × List PropertyFieldEvidence))
    (budget : Scoped.Limits := runLimits) : Option (Bool × List PropertyEndpointAnswer) := do
  let target ← targetResult.toOption
  let property ← (Property.check (context target)
    ((declaration (clause 0 .deliberatelyClosed)))).toOption
  let compiled ← (Scoped.compile target property [id "test.run"] (id "test.operation")
    budget).toOption
  let initial ← (compiled.start () state scope).toOption
  let violated ← (initial.consumeEvidence [request "a" 1]).toOption
  pure ((violated.consumeEvidence rejected).isOk, violated.close.answers.map Prod.snd)
#guard rejectionPreserves [reply "a" 2] == some (false, [.violated])
#guard rejectionPreserves [request "a" 2] { runLimits with captures := 1 } ==
  some (false, [.violated])

-- Two interleaved operations retain separate captures, and every chunk boundary inside the stream
-- preserves the same retained values, the same obligations and the same answers.
private def interleaved : List (Scoped.Transition × List PropertyFieldEvidence) :=
  [request "a" 1, request "b" 2, reply "a" 1, reply "b" 2]
#guard (run (clause 3) interleaved).toOption == some ⟨[.satisfied], [.satisfied]⟩
#guard error? (clause 3) [request "a" 1, request "b" 2, reply "a" 2] == some (.invalidTransition "a")
#guard (List.range (interleaved.length + 1)).all fun split =>
  (runChunks (clause 3) [interleaved.take split, interleaved.drop split]).toOption ==
    (run (clause 3) interleaved).toOption

-- A scoped Property that declares no captures keeps its exact canonical metadata, so existing
-- declarations do not acquire a new fingerprint from the default-empty extension.
#guard (do
  let target ← targetResult.toOption
  let bare ← (Property.check (context target)
    ((declaration (clause 1 (captures := []) (requirement := none))))).toOption
  let keyed ← (Property.check (context target) ((declaration (clause 1)))).toOption
  pure ((bare.canonicalMetadata.splitOn "\"captures\"").length == 1 &&
    (bare.canonicalMetadata.splitOn "\"correlation\"").length == 1 &&
    (keyed.canonicalMetadata.splitOn "\"captures\"").length == 2 &&
    (keyed.canonicalMetadata.splitOn "\"correlation\"").length == 2 &&
    bare.behaviorFingerprint != keyed.behaviorFingerprint)) == some true

-- The admitted-evidence adapter supplies no typed field values, so a clause that declares keyed
-- captures is rejected there rather than admitted into a run whose captures could never bind.
private def plan (target : TestTarget) := Observation.Projection.check target {
  id := id "test.projection"
  scopeFields := [id "test.run"]
  operationField := id "test.operation"
  sources := [id "test.source"]
  rules := [
    { kind := id "test.request", meaning := .confirmed none [(payload trigger 1, result false)] },
    { kind := id "test.reply", meaning := .confirmed none [(payload replied 1, result true)] }]
  limits := {
    events := 1000, buffered := 1000, keys := 1000, support := 10000
    work := 1000000, eventSize := 10000 }
} () state

#guard (do
  let target ← targetResult.toOption
  let projection ← (plan target).toOption
  let keyed ← (Property.check (context target) ((declaration (clause 1)))).toOption
  let bare ← (Property.check (context target)
    ((declaration (clause 1 (captures := []) (requirement := none))))).toOption
  pure ((match Observation.Scoped.compile projection keyed runLimits with
      | .error (.property (.unsupported clauseId _)) => clauseId == id "test.scoped.fields"
      | _ => false) &&
    (Observation.Scoped.compile projection bare runLimits).isOk)) == some true

-- A retained occurrence's presence was decided by the cursor that admitted it, so optional and
-- oneof-selected coordinates can be captured and read back. The same coordinates read as this
-- step's own operand still require a presence fact established in their Boolean branch.
private def optionalPath : PropertyFieldPath :=
  { triggerPath with steps := [.field "M" 2, .establish], type := .text }
private def oneofPath : PropertyFieldPath :=
  { triggerPath with steps := [.field "M" 3, .select "pick"], type := .text }
private def expected : PropertyFieldOperand := .literal (.text "expected") source

private def retaining (retained : PropertyFieldPath) : PropertyScopedClause :=
  clause 1 (captures := [{ capture with path := retained }])
    (requirement := some (PropertyPredicate.compareFields .equal
      (.field { retained with capture := some ⟨captureName, 0⟩ } source) expected source))

private def reading (unretained : PropertyFieldPath) : PropertyScopedClause :=
  clause 1 (captures := [])
    (requirement := some (PropertyPredicate.compareFields .equal (.field unretained source)
      expected source))

#guard error? (retaining optionalPath) [] == none
#guard error? (retaining oneofPath) [] == none
#guard errorKind? (reading optionalPath) [] == some .invalidClause
#guard errorKind? (reading oneofPath) [] == some .invalidClause

/-- info: 'Umpire.Property.Scoped.Captures.record_extends' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Scoped.Captures.record_extends
/-- info: 'Umpire.Property.Scoped.Run.consumeEvidence_append' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Scoped.Run.consumeEvidence_append
/-- info: 'Umpire.Property.Scoped.Execution.closed_property' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Scoped.Execution.closed_property

end Umpire.Property.ScopedFieldTests

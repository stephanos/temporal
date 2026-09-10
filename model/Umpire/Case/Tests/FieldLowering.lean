import Umpire.Case.Scoped
import Umpire.Property.Authoring
import Umpire.Model.Table
import Umpire.Shared.Test

/-!
Whole-Case field lowering with checked coverage.

The Target's outcomes are the model payloads of checked projections, so a step's typed `count` is
exactly the value a declared Observation reports. One coverage entry binds those modeled coordinates
to the declared evidence field `test.count`; a clause captures every admitted occurrence of it and
admits a reply only when the first occurrence its own operation retained is one.

The same declared evidence drives three evaluators, and they must agree: the model kernel over the
projections the coverage rebuilds, the evidence-driven model adapter over projected Observations,
and the portable interpreter over the lowered Case, replayed at every chunk boundary. Expected
answers, the emitted keyed fragment and the constructed request value are written out here rather
than read back from any of them.
-/

namespace Umpire.Case.FieldLoweringTests

open Umpire Operation Value
open temporal.server.api.testpilot.v1

private abbrev CaseArtifact := temporal.server.api.testpilot.v1.Case
private abbrev PortableValue := temporal.server.api.testpilot.v1.Value

private def id := DefinitionId.of
private def source : SourceLocation := { path := "Umpire/Case/Tests/FieldLowering.lean" }

private def schema : Schema := ⟨"M", [{
  name := "M", protoSyntax := "proto3", descriptor := "m", fileContext := "", references := [],
  valueShape := some (.message [
    ⟨1, "count", .integer .int32, .singular, .implicit (.integer .int32 0), none⟩,
    ⟨2, "tags", .text, .map .text, .implicit (.text ""), none⟩,
    ⟨3, "counts", .integer .int32, .repeated, .implicit (.integer .int32 0), none⟩,
    ⟨4, "label", .text, .singular, .optional, none⟩]) }]⟩
private def owner : RpcOwner where
  Witness _ _ := Unit
  schema _ := ⟨"example.Call", schema, schema, [], false, false⟩
private def valueLimits : Value.Limits := ⟨8, 10000, 1024, 100⟩

/-- One admitted outcome projection: the exact `count` a modeled result carries. -/
private def projection (reference : DefinitionId) (count : Int) :
    Except Field.Error (PropertyFieldProjection owner (Request := Unit) (Response := Unit) ()) := do
  let value ← (Value.check owner (Request := Unit) (Response := Unit) () .response valueLimits
    (Value.message "M" [(1, Value.literal (.integer .int32 count))])).mapError
    fun error => Field.Error.mk source error.path error.reason
  let member ← Field.reference owner () .response "M" 1 source
  let cursor ← (Field.root value).field member source
  let cursor ← cursor.refine (.integer .int32) .singular .available source
  PropertyFieldProjection.ofCursor .outcome reference cursor (by decide) source

private def accepted := id "test.accepted"

#guard ([1, 2] : List Int).all fun count => (projection accepted count).isOk

/-- The distinct fallback keeps a broken fixture visible instead of collapsing two counts onto one
model payload; the guard above proves every projection this module uses is admitted. -/
private def payload (count : Int) : ModelValue :=
  ((projection accepted count).toOption.map PropertyFieldProjection.modelValue).getD
    ⟨accepted, "unavailable-" ++ toString count⟩

private def value (key text : String) : ModelValue := ⟨id key, text⟩
private def state := value "test.state" "ready"
private def trigger := value "test.trigger" "request"
private def replied := value "test.reply" "reply"
private def responded := value "test.outcome" "response"
private def result (outcome : ModelValue) : Step ModelValue ModelValue ModelValue :=
  { state := state, outcome := outcome, facts := [] }

private def kinds : List (String × DefinitionKind) := [
  ("test.target", .target), ("test.kernel", .machine), ("test.provider", .provider),
  ("test.capability", .capability), ("test.state", .state), ("test.trigger", .action),
  ("test.reply", .action), ("test.accepted", .outcome), ("test.outcome", .outcome)]
private def definitions : List DefinitionMetadata := kinds.map fun (name, kind) =>
  Shared.Test.definitionMetadata name kind source (name ++ "/v1")
private def provider : Provider (fun _ => True) := {
  id := id "test.provider"
  source
  contract := { id := id "test.capability", behaviorVersion := "case-fields-test/v1", requiredLaws := [] }
  meanings := (kinds.drop 4).map fun (name, kind) =>
    { definitionId := id name, kind, behaviorVersion := name ++ "/meaning-v1" }
  lawProofs := []
}

/-- Selecting the trigger never selects its outcome: the Target owns both admitted counts. -/
private def table : FiniteTable Unit ModelValue ModelValue ModelValue ModelValue := {
  setups := [⟨(), "default"⟩]
  states := [⟨state, "ready"⟩]
  actions := [⟨trigger, "request"⟩, ⟨replied, "reply"⟩]
  outcomes := [⟨payload 1, "accepted-1"⟩, ⟨payload 2, "accepted-2"⟩, ⟨responded, "response"⟩]
  facts := []
  initial := [⟨(), [state]⟩]
  transitions := [
    ⟨"request", state, trigger, [result (payload 1), result (payload 2)]⟩,
    ⟨"reply", state, replied, [result responded]⟩]
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
private def context (target : TestTarget) : PropertyCheckContext :=
  { PropertyCheckContext.ofTarget target with
    fieldBindings := [accepted, id "test.trigger"].map
      (PropertyFieldBinding.ofWitness owner (Request := Unit) (Response := Unit) ()) }

private def countField := id "test.count"
private def tagField := id "test.tag"
private def flagField := id "test.flag"
private def acceptedPath : PropertyFieldPath :=
  { root := .outcome, reference := accepted,
    schema := owner.schema (Request := Unit) (Response := Unit) (), side := .response,
    steps := [.field "M" 1], type := .integer .int32 }
/-- A keyed map entry reaches its reported value without supplying anything else, so it is both
covered and rebuildable. -/
private def tagPath : PropertyFieldPath :=
  { acceptedPath with steps := [.field "M" 2, .key (.text "k"), .establish], type := .text }
/-- A presence read and a repeated element after the first describe values besides the reported
one; the first element is rebuilt as the one-element list that supplies exactly it. -/
private def presencePath : PropertyFieldPath :=
  { acceptedPath with steps := [.field "M" 4, .present], type := .boolean }
private def firstIndexPath : PropertyFieldPath :=
  { acceptedPath with steps := [.field "M" 3, .index 0] }
private def indexPath : PropertyFieldPath :=
  { acceptedPath with steps := [.field "M" 3, .index 1] }
/-- Request coordinates are constructed by the Program, never rebuilt from projected evidence. -/
private def requestPath : PropertyFieldPath :=
  { acceptedPath with root := .request, reference := id "test.trigger", side := .request }
private def requestTagPath : PropertyFieldPath :=
  { tagPath with root := .request, reference := id "test.trigger", side := .request }
private def captureName := id "test.capture.count"

private def retainedCount : List (EvidenceFieldDeclaration × FieldDisposition) :=
  [(⟨countField, .natural⟩, .retain)]
/-- A second evidence kind declares the map-entry and presence fields, so the request rules keep
supplying exactly the one field every request event carries. -/
private def retainedDetail : List (EvidenceFieldDeclaration × FieldDisposition) :=
  [(⟨tagField, .text⟩, .retain), (⟨flagField, .boolean⟩, .retain)]

private def declaration (fields : List (EvidenceFieldDeclaration × FieldDisposition) := retainedCount) :
    Observation.Projection.Declaration ModelValue ModelValue ModelValue ModelValue := {
  id := id "test.projection"
  scopeFields := [id "test.run"]
  operationField := id "test.operation"
  sources := [id "test.source"]
  rules := [
    { kind := id "test.request.one", fields, meaning := .confirmed none [(trigger, result (payload 1))] },
    { kind := id "test.request.two", fields, meaning := .confirmed none [(trigger, result (payload 2))] },
    { kind := id "test.reply", meaning := .confirmed none [(replied, result responded)] },
    { kind := id "test.detail", fields := retainedDetail, meaning := .irrelevant }]
  limits := {
    events := 64, buffered := 32, keys := 16, support := 256
    work := 1000000000, eventSize := 512 }
}

private def plan (target : TestTarget)
    (fields : List (EvidenceFieldDeclaration × FieldDisposition) := retainedCount) :=
  Observation.Projection.check target (declaration fields) () state

private def mapping : Observation.Projection.FieldMapping := ⟨acceptedPath, countField⟩

/-- A reply belongs to this operation only when the first count it retained is one. The trigger
disjunct decides the request step before the capture operand is reached, so the step that creates
occurrence zero never has to bind it. -/
private def correlation (ordinal : Nat := 0) (name : DefinitionId := captureName) : PropertyPredicate :=
  .any [.atom { field := .selectedAction, reference := id "test.trigger" },
    PropertyPredicate.compareFields .equal
      (.field { acceptedPath with capture := some ⟨name, ordinal⟩ } source)
      (.literal (.integer .int32 1) source) source]

private def capture (lifetime : Nat := 4) (path : PropertyFieldPath := acceptedPath) :
    PropertyScopedCapture := { name := captureName, key := id "test.operation", path, lifetime }

private def clause (bound : Nat := 1) (endpoint : PropertyScopedEndpoint := .runtimePrefix)
    (captures : List PropertyScopedCapture := [capture])
    (requirement : Option PropertyPredicate := some correlation) : PropertyScopedClause := {
  id := id "test.scoped.fields"
  source
  trigger := .atom { field := .selectedAction, reference := id "test.trigger" }
  response := .atom { field := .modelOutcome, reference := id "test.outcome" }
  scope := [id "test.run"]
  key := id "test.operation"
  clock := .operationTransitions
  bound
  endpoint
  captures
  correlation := requirement
}

private def property (target : TestTarget) (temporal : PropertyScopedClause) :=
  checkProperty (context target) (.portable {
    id := id "test.property.fields"
    source
    requires := [id "test.capability"]
    clauses := []
    scopedClauses := [temporal] })

private def runLimits : Property.Scoped.Limits :=
  { transitions := 64, obligations := 32, work := 100000000, captures := 32 }
private def scope : List (DefinitionId × String) := [(id "test.run", "run-1")]

/-! ### One stream of declared Observations, shared by every evaluator -/

/-- One admitted Observation. `count` is the declared evidence field the coverage map binds. -/
private structure Report where
  ordinal : Nat
  kind : String
  operation : String := "a"
  count : Option Nat := none
  deriving Repr

private def request (ordinal count : Nat) (operation := "a") : Report :=
  ⟨ordinal, if count == 1 then "test.request.one" else "test.request.two", operation, some count⟩
private def reply (ordinal : Nat) (operation := "a") : Report :=
  ⟨ordinal, "test.reply", operation, none⟩

private def modelEvent (report : Report) : Observation.Projection.Event := {
  identity := { scope, source := id "test.source", ordinal := report.ordinal }
  operation := report.operation
  kind := id report.kind
  runSequences := [report.ordinal + 1]
  fields := report.count.toList.map fun count => ⟨countField, some (.natural count)⟩ }

private def wireEvent (report : Report) : ScopedEvidence := {
  identity := some {
    scope := #[{ field_id := "test.run", value := "run-1" }]
    source := "test.source", ordinal := Int64.ofInt report.ordinal }
  operation := report.operation
  kind := report.kind
  fields := (report.count.toList.map fun count =>
    ({ field_id := "test.count", value := some { value := some (.natural (toString count)) } } :
      ScopedEvidenceField)).toArray }

private def transition (report : Report) : Property.Scoped.Transition :=
  let (action, outcome) := match report.kind with
    | "test.request.one" => (trigger, payload 1)
    | "test.request.two" => (trigger, payload 2)
    | _ => (replied, responded)
  { scope
    operationField := id "test.operation"
    operation := report.operation
    priorState := state
    action
    result := result outcome }

/-- The shared answer alphabet: satisfied, violated, or still unresolved. -/
private def code : PropertyEndpointAnswer → Nat
  | .satisfied => 2
  | .violated => 3
  | .unresolved => 0

/-! ### The three evaluators -/

/-- The model kernel, over the projections the coverage rebuilds from the declared evidence. -/
private def modelAnswers (temporal : PropertyScopedClause) (reports : List Report)
    (split : Nat := 0) : Option (List Nat) := do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits [mapping]).toOption
  let checked ← (property target temporal).toOption
  let compiled ← (Property.Scoped.compile target checked [id "test.run"] (id "test.operation")
    runLimits).toOption
  let initial ← (compiled.start () state scope).toOption
  let steps ← reports.mapM fun report => do
    let evidence ← (coverage.evidence (modelEvent report).fields).toOption
    pure (transition report, evidence)
  let run ← (initial.consumeEvidence (steps.take split) >>= fun next =>
    next.consumeEvidence (steps.drop split)).toOption
  pure (run.close.answers.map fun answer => code answer.2)

/-- The evidence-driven model adapter, over exactly the projected Observations. -/
private def adapterAnswers (temporal : PropertyScopedClause) (reports : List Report)
    (split : Nat := 0) : Option (List Nat) := do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits [mapping]).toOption
  let checked ← (property target temporal).toOption
  let compiled ← (Observation.Scoped.compile projected checked runLimits coverage).toOption
  let initial ← (Observation.Scoped.start projected compiled () scope coverage).toOption
  let events := reports.map modelEvent
  let run ← (initial.admitMany (events.take split) >>= fun next =>
    next.admitMany (events.drop split)).toOption
  pure (run.close.answers.map fun answer => code answer.2)

/-! ### The lowered Case -/

private def requestValue (count : Int) : PortableValue :=
  { value := some (.signed_integer (toString count)) }

private def program (assigned : Int := 7) : Program :=
  Testpilot.Authoring.Program.make "fields.program" #[] #[]
    #[Testpilot.Authoring.Program.observation "evidence" (Testpilot.Authoring.Types.singular
      (Testpilot.Authoring.Types.messageType "temporal.server.api.testpilot.v1.ScopedEvidence"))]
    #[Testpilot.Authoring.Program.controller "controller"
      #[Testpilot.Authoring.Program.node "start"
        (Testpilot.Authoring.Program.invokeRPC "source" "/example.Call/Do"
          #[Testpilot.Authoring.Program.requestAssignment
              (Testpilot.Authoring.Path.make #[Testpilot.Authoring.Path.field "count"])
              (Testpilot.Authoring.ProgramExpr.literal (requestValue assigned)),
            Testpilot.Authoring.Program.requestAssignment
              (Testpilot.Authoring.Path.make
                #[Testpilot.Authoring.Path.mapKey "tags" { value := some (.text "k") }])
              (Testpilot.Authoring.ProgramExpr.literal { value := some (.text "v") })])
        (Testpilot.Authoring.Program.instructionLimits 1000 1 1 4096)]]
    (Testpilot.Authoring.Program.cleanup "cleanup" #[])
    (Testpilot.Authoring.Program.limits 4 16 16 16 16 32 8 8 4096 4096 10000 1000)

private def inputCoverage (assigned : Int := 7) : Coverage.InputMapping :=
  { path := requestPath, value := .integer .int32 assigned
    entrypointId := "controller", instructionId := "start" }

/-- A keyed map entry is an exact request assignment target, so its coverage names the key too. -/
private def tagInputCoverage : Coverage.InputMapping :=
  { path := requestTagPath, value := .text "v"
    entrypointId := "controller", instructionId := "start" }

private def caseCoverage (assigned : Int := 7) : Coverage.Request :=
  { inputs := [inputCoverage assigned, tagInputCoverage], clauses := [id "test.scoped.fields"] }

/-- Lower one requested Case, from the checked coverage through to the assembled artifact. -/
private def compiledCase (temporal : PropertyScopedClause)
    (mappings : List Observation.Projection.FieldMapping := [mapping])
    (requested : Coverage.Request := caseCoverage) (assigned : Int := 7)
    (fields : List (EvidenceFieldDeclaration × FieldDisposition) := retainedCount) :
    Except String CaseArtifact := do
  let target ← targetResult.mapError fun _ => "target"
  let projected ← (plan target fields).mapError fun _ => "projection"
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits mappings).mapError
    (·.reason)
  let checked ← (property target temporal).mapError fun _ => "property"
  let compiled ← (Property.Scoped.compile target checked [id "test.run"] (id "test.operation")
    runLimits).mapError fun _ => "scoped compile"
  let lowered ← (Scoped.lower projected compiled "evidence" coverage).mapError (·.construct)
  (Compiler.compile {
    version := { major := 1 }
    caseId := "fields.case"
    producerId := "umpire.case.fields"
    definitions := [⟨checked.id.value, checked.behaviorFingerprint.render, .property⟩]
    sources := [source]
    knownGaps := []
    program := program assigned
    contractId := "fields"
    properties := [lowered.contractLowering]
    contractLimits := Testpilot.Authoring.Monitor.limits 16 32 64 16 100000 1000000000 32 65536
    coverage := requested
  }).mapError (·.construct)

private def capabilityOf (temporal : PropertyScopedClause) : Except String ScopedContract := do
  let artifact ← compiledCase temporal
  let some contract := artifact.contract | throw "missing contract"
  let some capability := contract.«scoped» | throw "missing capability"
  pure capability

/-- The portable interpreter over the lowered Case, replayed at one chunk boundary. -/
private def portableAnswers (temporal : PropertyScopedClause) (reports : List Report)
    (split : Nat := 0) : Option (List Nat) := do
  let capability ← (capabilityOf temporal).toOption
  let compiled ← (Testpilot.Scoped.decode capability).toOption
  let initial ← (compiled.start [(⟨"test.run"⟩, "run-1")]).toOption
  let observe := fun (run : Testpilot.Scoped.Run compiled) (report : Report) =>
    run.observe (report.ordinal + 2) (wireEvent report)
  let run ← (((reports.take split).foldlM observe initial) >>= fun next =>
    (reports.drop split).foldlM observe next).toOption
  pure run.close.answers

/-! ### The three evaluators agree, at every chunk boundary -/

private structure Scenario where
  name : String
  bound : Nat
  reports : List Report
  expected : Option (List Nat)

private def scenarios : List Scenario := [
  ⟨"matched", 1, [request 0 1, reply 1], some [2]⟩,
  ⟨"unmatched-count", 1, [request 0 2, reply 1], none⟩,
  ⟨"pending", 1, [request 0 1], some [0]⟩,
  ⟨"deadline", 0, [request 0 1, reply 1], some [3]⟩,
  ⟨"second-occurrence", 2, [request 0 1, request 1 2, reply 2], some [2]⟩,
  ⟨"foreign-operation", 3, [request 0 1 "a", request 1 2 "b", reply 2 "a", reply 3 "b"], none⟩,
  ⟨"missing-evidence", 1, [⟨0, "test.request.one", "a", none⟩, reply 1], none⟩]

private def agrees (scenario : Scenario) : Bool :=
  (List.range (scenario.reports.length + 1)).all fun split =>
    let temporal := clause scenario.bound
    modelAnswers temporal scenario.reports split == scenario.expected &&
      adapterAnswers temporal scenario.reports split == scenario.expected &&
      portableAnswers temporal scenario.reports split == scenario.expected

#guard scenarios.all agrees

-- Corrupting the declared evidence a correlation reads is a link failure: the step is not one of
-- this operation's semantic steps at all, so both evaluators reject the append instead of reporting
-- a product violation, and neither lets missing evidence satisfy the comparison.
#guard (do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits [mapping]).toOption
  let checked ← (property target (clause)).toOption
  let compiled ← (Property.Scoped.compile target checked [id "test.run"] (id "test.operation")
    runLimits).toOption
  let initial ← (compiled.start () state scope).toOption
  let steps ← [request 0 2, reply 1].mapM fun report => do
    let evidence ← (coverage.evidence (modelEvent report).fields).toOption
    pure (transition report, evidence)
  pure (match initial.consumeEvidence steps with
    | .error failure => failure == Property.Scoped.Error.invalidTransition "a"
    | .ok _ => false)) == some true

#guard (do
  let capability ← (capabilityOf (clause)).toOption
  let compiled ← (Testpilot.Scoped.decode capability).toOption
  let initial ← (compiled.start [(⟨"test.run"⟩, "run-1")]).toOption
  let admitted ← (initial.observe 2 (wireEvent (request 0 2))).toOption
  pure (match admitted.observe 3 (wireEvent (reply 1)) with
    | .error reason => reason == "correlation rejected this operation's step"
    | .ok _ => false)) == some true

/-! ### The emitted capability means exactly what the model declared -/

/-- The keyed fragment written out here from the declarations above, not read back from the
encoder. -/
private def expectedKeyed : List (String × Testpilot.Scoped.Keyed) :=
  [("test.scoped.fields", ⟨[⟨captureName, countField, 2, 4⟩],
    some (.any (.cons (.predicate ⟨1, id "test.trigger", none⟩)
      (.cons (.comparison true (.capture captureName 0) (.literal (.natural 1))) .nil)))⟩)]

private def decodedKeyed (temporal : PropertyScopedClause) :
    Except String (List (String × Testpilot.Scoped.Keyed)) := do
  let compiled ← Testpilot.Scoped.decode (← capabilityOf temporal)
  pure compiled.keyed

#guard (decodedKeyed (clause)).toOption == some expectedKeyed

-- The emitted wire clause names the declared evidence field, and both ceilings are the exact ones
-- this capability needs.
#guard ((do
  let capability ← capabilityOf (clause)
  let some wire := capability.clauses[0]? | throw "missing clause"
  let some declared := wire.captures[0]? | throw "missing capture"
  let some limits := capability.limits | throw "missing limits"
  pure (declared.capture_id == "test.capture.count" && declared.field_id == "test.count" &&
    declared.lifetime == 4 && wire.correlation.isSome &&
    limits.max_captures == 32 && limits.max_correlation_depth == 2)) :
    Except String Bool).toOption == some true

-- A Case that declares neither captures nor a correlation leaves both ceilings unset, so its
-- encoding and meaning are exactly the ones it had before the keyed capability existed.
#guard ((do
  let capability ← capabilityOf (clause (captures := []) (requirement := none))
  let some limits := capability.limits | throw "missing limits"
  let some wire := capability.clauses[0]? | throw "missing clause"
  pure (limits.max_captures == 0 && limits.max_correlation_depth == 0 &&
    wire.captures.isEmpty && wire.correlation.isNone)) : Except String Bool).toOption == some true

/-! ### Coverage is inspected before anything is assembled -/

private def rejects (result : Except String α) (reason : String) : Bool :=
  match result with
  | .error actual => actual == reason
  | .ok _ => false

-- A capture whose coordinates no declared Observation supplies rejects the whole Case.
#guard rejects (compiledCase (clause) (mappings := []))
  "captured coordinates have no declared Observation for test.capture.count"

-- Coverage is admitted only against the projection declaration the Case will run under.
private def coverageRejects (mappings : List Observation.Projection.FieldMapping)
    (fields : List (EvidenceFieldDeclaration × FieldDisposition) := retainedCount)
    (reason : String) : Bool :=
  match targetResult with
  | .error _ => false
  | .ok target => match plan target fields with
    | .error _ => false
    | .ok projected =>
        match Observation.Projection.Coverage.check projected valueLimits mappings with
        | .error failure => failure.reason == reason
        | .ok _ => false

#guard coverageRejects [⟨acceptedPath, id "test.missing"⟩]
  (reason := "unknown or unretained declared evidence field")
#guard coverageRejects [mapping] (fields := [(⟨countField, .natural⟩, .redact)])
  (reason := "unknown or unretained declared evidence field")
#guard coverageRejects [mapping] (fields := [(⟨countField, .text⟩, .retain)])
  (reason := "declared evidence type is not the modeled field type")
#guard coverageRejects [⟨{ acceptedPath with capture := some ⟨captureName, 0⟩ }, countField⟩]
  (reason := "coverage names field coordinates, never a retained occurrence")
#guard coverageRejects [mapping, ⟨{ acceptedPath with root := .resultingState }, countField⟩]
  (reason := "declared evidence field is already covered")
#guard coverageRejects [⟨{ acceptedPath with steps := [.field "M" 9] }, countField⟩]
  (reason := "unknown field 9")

-- A keyed map entry is rebuilt exactly at its covered coordinates; a presence read and a repeated
-- element are not, because the payload witnessing them would carry data no Observation reported.
#guard (do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits
    [mapping, ⟨tagPath, tagField⟩]).toOption
  let values ← (coverage.evidence [⟨tagField, some (.text "v")⟩]).toOption
  pure (values.map PropertyFieldEvidence.path == [tagPath])) == some true

#guard coverageRejects [⟨presencePath, flagField⟩]
  (reason := "a presence read reports data no declared Observation supplies")
#guard coverageRejects [⟨indexPath, countField⟩]
  (reason := "a repeated element after the first reports data no declared Observation supplies")

-- The first repeated element is rebuilt at exactly its covered coordinates.
#guard (do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits
    [⟨firstIndexPath, countField⟩]).toOption
  let values ← (coverage.evidence [⟨countField, some (.natural 1)⟩]).toOption
  pure (values.map PropertyFieldEvidence.path == [firstIndexPath])) == some true

-- A request mapping is covered for portable lowering only. Replay skips it, so a lowering-only
-- mapping cannot fail an evidence-driven Run that never reads it.
#guard (do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits
    [mapping, ⟨requestTagPath, tagField⟩]).toOption
  let checked ← (property target (clause)).toOption
  let compiled ← (Observation.Scoped.compile projected checked runLimits coverage).toOption
  let initial ← (Observation.Scoped.start projected compiled () scope coverage).toOption
  let run ← (initial.admitMany ([request 0 1, reply 1].map modelEvent)).toOption
  pure (run.close.answers.map fun answer => code answer.2)) == some [2]

-- A request operand is constructed by the Program, so no declared Observation rebuilds it.
#guard (do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let coverage ← (Observation.Projection.Coverage.check projected valueLimits
    [⟨requestPath, countField⟩]).toOption
  let checked ← (property target (clause (captures := [capture (path := requestPath)])
    (requirement := none))).toOption
  pure (match Observation.Scoped.compile projected checked runLimits coverage with
    | .error (.property (.unsupported _ reason)) =>
        reason == "request operand is not rebuildable from projected evidence"
    | _ => false)) == some true

-- The adapter admits a capture-bearing clause only under a coverage that supplies it.
#guard (do
  let target ← targetResult.toOption
  let projected ← (plan target).toOption
  let checked ← (property target (clause)).toOption
  pure (match Observation.Scoped.compile projected checked runLimits with
    | .error (.property (.unsupported clauseId _)) => clauseId == id "test.scoped.fields"
    | _ => false)) == some true

/-! ### Requested input construction and clause lowering are inspected too -/

-- The Program constructs the covered input field with exactly the modeled value.
#guard (compiledCase (clause)).isOk
#guard rejects (compiledCase (clause) (assigned := 8))
  "request assignment constructs a different value than the modeled field"
#guard rejects (compiledCase (clause)
  (requested := { caseCoverage with
    inputs := [{ inputCoverage with instructionId := "absent" }] }))
  "unknown instruction absent"
#guard rejects (compiledCase (clause)
  (requested := { caseCoverage with inputs := [{ tagInputCoverage with
    path := { requestTagPath with steps := [.field "M" 2, .key (.text "other"), .establish] } }] }))
  "covered input field has 0 request assignments"
#guard rejects (compiledCase (clause)
  (requested := { caseCoverage with clauses := [id "test.absent"] }))
  "requested clause was not lowered exactly once"
#guard rejects (compiledCase (clause)
  (requested := { caseCoverage with inputs := [{ inputCoverage with path := acceptedPath }] }))
  "input coverage names request coordinates; results and events are covered by Observations"
#guard rejects (compiledCase (clause)
  (requested := { caseCoverage with inputs := [{ inputCoverage with
    path := { presencePath with root := .request, reference := id "test.trigger", side := .request }
    value := .boolean true }] }))
  "a presence read is not a request assignment target"

/-! ### Trust -/

/-- info: 'Umpire.Case.Scoped.Lowered.window_property' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Case.Scoped.Lowered.window_property
/-- info: 'Umpire.Case.Scoped.Lowered.evidence_validation' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Case.Scoped.Lowered.evidence_validation
/-- info: 'Umpire.Observation.Scoped.Run.admitMany_append' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Observation.Scoped.Run.admitMany_append

end Umpire.Case.FieldLoweringTests

import Lean.Data.Json
import Temporal.Feature.Nexus.Caller.Model
import Umpire.Examples.Switch

/-!
# The inspector

`umpire-inspect <id>` prints the canonical Plan Artifact of one registered Query, `umpire-list`
the registry, and `umpire-explain <id>` one registered Query's checked lineage. The registry is the
caller Model's Queries beside Umpire's Switch example: what the Model file declares is what the
inspector shows, and a Query whose form selects no Artifact (a verify Query) is listed with none.

The caller Model is a command file, so `query`, `property` and their kin are keywords here; the
binders below are spelled around them.
-/

namespace Temporal.Tool.Inspect

open _root_.Umpire

inductive InspectionFailure where
  | declaration (error : DefinitionError)
  | propertyCheck (error : PropertyError)
  | behaviorCheck (error : ScenarioError)
  | queryCheck (error : QueryError)
  | admission (subject : String) (context : String)
  | planning (subject : String)
  | knownGap (error : KnownGapError)
  deriving BEq, DecidableEq, Repr

/-- One registered Query: its identity, its lineage for `explain`, and its Artifact for `inspect`. -/
structure Scenario where
  id : String
  kind : String
  explanation : CanonicalJson
  result : Except InspectionFailure Plan

abbrev ScenarioRegistry : Type := List Scenario

structure InspectorResult where
  status : Nat
  stdout : String
  stderr : String
  deriving BEq, DecidableEq, Repr

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def diagnostic (kind subject context : String) : String :=
  "{\"kind\":" ++ quote kind ++
    ",\"subject\":" ++ quote subject ++
    ",\"context\":" ++ quote context ++ "}\n"

private def failureJson : InspectionFailure → String
  | .declaration error => canonicalDefinitionErrorJson error ++ "\n"
  | .propertyCheck error => canonicalPropertyErrorJson error ++ "\n"
  | .behaviorCheck error => canonicalScenarioErrorJson error ++ "\n"
  | .queryCheck error => canonicalQueryErrorJson error ++ "\n"
  | .admission subject context => diagnostic "admission-failure" subject context
  | .planning subject => diagnostic "planning-failure" subject "no portable artifact"
  | .knownGap error => diagnostic "known-gap-check-failed" error.code.value error.kind.name

private def failed (failure : InspectionFailure) : InspectorResult :=
  { status := 1, stdout := "", stderr := failureJson failure }

private def succeeded (stdout : String) : InspectorResult := { status := 0, stdout, stderr := "" }

def runInspector (registry : ScenarioRegistry) (args : List String) : InspectorResult :=
  match args with
  | [requested] =>
      match registry.find? (fun entry => entry.id == requested) with
      | none => {
          status := 1
          stdout := ""
          stderr := diagnostic "unknown-scenario" requested "scenario registry"
        }
      | some entry =>
          match entry.result with
          | .error failure => failed failure
          | .ok spec => succeeded (canonicalPlanBytes spec)
  | _ => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-arguments" "inspect" "expected exactly one scenario identity"
    }

/-- The constructor of an admission failure, which is all a listing needs to say about it. -/
private def admissionContext : Command.AdmissionError → String
  | .invalidTarget _ => "invalid-target"
  | .invalidVocabulary _ => "invalid-vocabulary"
  | .admission _ => "admission"
  | .notSelected outcome _ _ => "not-selected: " ++ outcome.name
  | .instances reason => "instances: " ++ reason

private def fingerprint (value : BehaviorFingerprint) : CanonicalJson := .string value.render

/-- A command-authored Query as a registry entry: the Query's identity names it, its checked
Property, Scenario and Target are its lineage, and its planned Artifact (when its form selects
one) is what `inspect` prints. -/
private def authoredScenario {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {declared : Command.DeclaredModel Setup State Action Outcome Fact}
    (queryId : String)
    (admitted : Except Command.AdmissionError (Command.CheckedModel declared)) : Scenario :=
  match admitted with
  | .error error => {
      id := queryId
      kind := "query"
      explanation := .object [
        ("id", .string queryId),
        ("kind", .string "query"),
        ("admission", .string (admissionContext error))]
      result := .error (.admission queryId (admissionContext error))
    }
  | .ok checked => {
      id := checked.query.id.value
      kind := "query"
      explanation := .object [
        ("id", .string checked.query.id.value),
        ("kind", .string "query"),
        ("source", .string checked.query.source.path),
        ("form", .string checked.query.form.name),
        ("behaviorFingerprint", fingerprint checked.query.behaviorFingerprint),
        ("property", .object [
          ("id", .string checked.property.id.value),
          ("behaviorFingerprint", fingerprint checked.property.behaviorFingerprint)]),
        ("behavior", .object [
          ("id", .string checked.behavior.id.value),
          ("behaviorFingerprint", fingerprint checked.behavior.behaviorFingerprint)]),
        ("target", .object [
          ("id", .string checked.target.id.value),
          ("behaviorFingerprint", fingerprint checked.target.behaviorFingerprint)]),
        ("outcome", .string checked.run.result.outcome.name),
        ("witness", CanonicalJson.ofOption (fun (witness : Scenario.Trace) =>
          .array (witness.trace.steps.map fun step => .string step.selectedAction.value))
          checked.witness),
        ("artifactChecksum", CanonicalJson.ofOption (fun (spec : Plan) =>
          .string spec.artifactChecksum.render) checked.run.artifact)]
      result := match checked.run.artifact with
        | some spec => .ok spec
        | none => .error (.planning checked.query.id.value)
    }

/-- The caller Model's Queries, in the order the Model declares them, then the Switch example. -/
def productionRegistry : ScenarioRegistry := [
  authoredScenario "temporal.nexus.caller.query.syncCompletion"
    Temporal.Feature.Nexus.Caller.syncCompletion,
  authoredScenario "temporal.nexus.caller.query.asyncCompletion"
    Temporal.Feature.Nexus.Caller.asyncCompletion,
  authoredScenario "temporal.nexus.caller.query.asyncFailure"
    Temporal.Feature.Nexus.Caller.asyncFailure,
  authoredScenario "temporal.nexus.caller.query.handlerError"
    Temporal.Feature.Nexus.Caller.handlerError,
  authoredScenario "temporal.nexus.caller.query.retry"
    Temporal.Feature.Nexus.Caller.retry,
  authoredScenario "temporal.nexus.caller.query.scheduleToStartTimeout"
    Temporal.Feature.Nexus.Caller.scheduleToStartTimeout,
  authoredScenario "temporal.nexus.caller.query.startToCloseTimeout"
    Temporal.Feature.Nexus.Caller.startToCloseTimeout,
  authoredScenario "temporal.nexus.caller.query.terminalHolds"
    Temporal.Feature.Nexus.Caller.terminalHolds,
  { id := _root_.Umpire.Examples.Switch.exactActionQueryId.value
    kind := "example"
    explanation := .object [
      ("id", .string _root_.Umpire.Examples.Switch.exactActionQueryId.value),
      ("kind", .string "example"),
      ("artifactChecksum",
        .string _root_.Umpire.Examples.Switch.compiledArtifact.artifactChecksum.render)]
    result := .ok _root_.Umpire.Examples.Switch.compiledArtifact }]

/-- The `list` command: every registered identity with its kind, in registry order. -/
def runList (registry : ScenarioRegistry) : InspectorResult :=
  succeeded (CanonicalJson.prettyBytes (.array (registry.map fun entry =>
    .object [("id", .string entry.id), ("kind", .string entry.kind)])))

/-- The `explain` command: one registered Query's checked lineage. -/
def runExplain (registry : ScenarioRegistry) (requested : String) : InspectorResult :=
  match registry.find? (fun entry => entry.id == requested) with
  | none => {
      status := 1
      stdout := ""
      stderr := diagnostic "unknown-query" requested "scenario registry"
    }
  | some entry => succeeded (CanonicalJson.prettyBytes entry.explanation)

def runCli (args : List String) : InspectorResult :=
  match args with
  | ["list"] => runList productionRegistry
  | ["explain", requested] => runExplain productionRegistry requested
  | "explain" :: _ => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-arguments" "explain"
        "expected exactly one canonical query identity"
    }
  | _ => runInspector productionRegistry args

end Temporal.Tool.Inspect

def main (args : List String) : IO UInt32 := do
  let result := Temporal.Tool.Inspect.runCli args
  IO.print result.stdout
  IO.eprint result.stderr
  if result.status == 0 then
    pure 0
  else
    pure 1

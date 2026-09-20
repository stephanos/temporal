import Temporal.Tool.Inspect

namespace Temporal.Tool.InspectTests

open _root_.Umpire
open Temporal.Tool.Inspect

/-- The registry is the caller Model's eight Queries in declaration order, then the Switch example. -/
example : productionRegistry.map (fun entry => (entry.id, entry.kind)) = [
    ("temporal.nexus.caller.query.syncCompletion", "query"),
    ("temporal.nexus.caller.query.asyncCompletion", "query"),
    ("temporal.nexus.caller.query.asyncFailure", "query"),
    ("temporal.nexus.caller.query.handlerError", "query"),
    ("temporal.nexus.caller.query.retry", "query"),
    ("temporal.nexus.caller.query.scheduleToStartTimeout", "query"),
    ("temporal.nexus.caller.query.startToCloseTimeout", "query"),
    ("temporal.nexus.caller.query.terminalHolds", "query"),
    ("switch.query.exact-action", "example")
  ] := by
  native_decide

/-- `list` prints the registry, deterministically, and nothing on stderr. -/
example : runCli ["list"] = runCli ["list"] ∧
    (runCli ["list"]).status = 0 ∧
    (runCli ["list"]).stderr = "" ∧
    (runCli ["list"]).stdout = CanonicalJson.prettyBytes (.array (productionRegistry.map fun entry =>
      .object [("id", .string entry.id), ("kind", .string entry.kind)])) := by
  native_decide

/-- Every registered identity explains to its lineage with exit 0; the explanation of a found
Query names its witness and Artifact, a verify Query neither. -/
example : (productionRegistry.map fun entry => (runCli ["explain", entry.id]).status) =
      List.replicate productionRegistry.length 0 ∧
    (productionRegistry.map fun entry => (runCli ["explain", entry.id]).stdout ==
      CanonicalJson.prettyBytes entry.explanation).all id = true ∧
    ((runCli ["explain", "temporal.nexus.caller.query.asyncCompletion"]).stdout.splitOn
      "\"witness\": [").length = 2 ∧
    ((runCli ["explain", "temporal.nexus.caller.query.terminalHolds"]).stdout.splitOn
      "\"witness\": null").length = 2 ∧
    ((runCli ["explain", "temporal.nexus.caller.query.terminalHolds"]).stdout.splitOn
      "\"artifactChecksum\": null").length = 2 := by
  native_decide

def invalidExplanationSelectors : List String := [
  "missing-query",
  "Temporal.nexus.caller.query.asyncCompletion",
  "temporal.nexus.caller.query.async",
  "temporal.nexus.caller.query",
  "asyncCompletion",
  "temporal.nexus.caller.property.completionSucceeds",
  ""
]

example : invalidExplanationSelectors.map (fun selector => runCli ["explain", selector]) =
    invalidExplanationSelectors.map fun selector => {
      status := 1
      stdout := ""
      stderr :=
        "{\"kind\":\"unknown-query\",\"subject\":" ++
          Lean.Json.compress (.str selector) ++
          ",\"context\":\"scenario registry\"}\n"
    } := by
  native_decide

example : [runCli ["explain"], runCli ["explain",
    "temporal.nexus.caller.query.asyncCompletion", "extra"]] =
    List.replicate 2 {
      status := 1
      stdout := ""
      stderr :=
        "{\"kind\":\"invalid-arguments\",\"subject\":\"explain\"," ++
          "\"context\":\"expected exactly one canonical query identity\"}\n"
    } := by
  native_decide

def expectedSwitchStdout : String :=
  canonicalPlanBytes _root_.Umpire.Examples.Switch.compiledArtifact

def repeatedSwitchOutput : List String :=
  (List.range 2).map fun _ => (runCli [_root_.Umpire.Examples.Switch.exactActionQueryId.value]).stdout

example : runCli [_root_.Umpire.Examples.Switch.exactActionQueryId.value] = {
    status := 0
    stdout := expectedSwitchStdout
    stderr := ""
  } := by
  native_decide

example : repeatedSwitchOutput = List.replicate 2 expectedSwitchStdout := by
  native_decide

/-- The Artifact `inspect` prints for a found Query is the Query's own planned Artifact. -/
private def plannedArtifact (queryId : String) : Option String :=
  (productionRegistry.find? (·.id == queryId)).bind fun entry =>
    entry.result.toOption.map canonicalPlanBytes

/-! Every found Query of the caller Model is available through the inspector; the verify Query has
no Artifact to print and says so. -/
example :
    (["temporal.nexus.caller.query.syncCompletion",
      "temporal.nexus.caller.query.asyncCompletion",
      "temporal.nexus.caller.query.asyncFailure",
      "temporal.nexus.caller.query.handlerError",
      "temporal.nexus.caller.query.retry",
      "temporal.nexus.caller.query.scheduleToStartTimeout",
      "temporal.nexus.caller.query.startToCloseTimeout"].map fun id =>
      (runCli [id]).status == 0 && (runCli [id]).stdout == (plannedArtifact id).getD "") =
      List.replicate 7 true ∧
    runCli ["temporal.nexus.caller.query.terminalHolds"] = {
      status := 1
      stdout := ""
      stderr :=
        "{\"kind\":\"planning-failure\",\"subject\":\"temporal.nexus.caller.query.terminalHolds\"," ++
          "\"context\":\"no portable artifact\"}\n"
    } := by
  native_decide

example : runCli ["missing-scenario"] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"unknown-scenario\",\"subject\":\"missing-scenario\"," ++
        "\"context\":\"scenario registry\"}\n"
  } := by
  native_decide

example : runCli [] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"invalid-arguments\",\"subject\":\"inspect\"," ++
        "\"context\":\"expected exactly one scenario identity\"}\n"
  } := by
  native_decide

end Temporal.Tool.InspectTests

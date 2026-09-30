import Temporal.Feature.Nexus.Caller.Model
import Lean.Data.Json

/-!
# The Lean side of the Go parity harness

Writes what the production Nexus caller Model computes -- its tables, refinement rows, Query
outcomes and witnesses, and exploration targets -- as JSON, through the same public names the
Model's own pins read. It changes nothing under `model/`; `dump.sh` runs it against the built
modules.
-/

open Lean (Json)
open Umpire
open Umpire.Command
open Temporal.Feature.Nexus.Caller

namespace UmpireGo.Dump

def str (s : String) : Json := Json.str s
def strs (xs : List String) : Json := Json.arr (xs.map str).toArray

def catalogKey {α : Type} [BEq α] (catalog : FiniteCatalog α) (value : α) : String :=
  ((catalog.find? (·.value == value)).map (·.key)).getD "?"

/-- `Umpire.Command.reachableFrom`'s walk -- sweep the rows in table order, appending each newly
reached result state, until a sweep adds nothing -- without the `Finite` state instance that
function takes for its sweep count, which a composed state does not carry. -/
partial def reach {St A O F : Type} [BEq St] (rows : List (FiniteTransitionRow St A O F))
    (seen : List St) : List St :=
  let grown := rows.foldl (init := seen) fun seen row =>
    if seen.contains row.source then
      row.results.foldl (init := seen) fun seen result =>
        if seen.contains result.state then seen else seen ++ [result.state]
    else seen
  if grown.length == seen.length then seen else reach rows grown

/-- A declared model's table read through its own catalogs, so every machine -- declared, derived
or composed -- is written the same way. -/
def tableJson {S St A O F : Type} [BEq S] [BEq St] [BEq A] [BEq O] [BEq F]
    (name : String) (model : DeclaredModel S St A O F) : Json :=
  let t := model.table
  let sk := catalogKey t.states
  let ak := catalogKey t.actions
  let ok := catalogKey t.outcomes
  let fk := catalogKey t.facts
  let reachable := reach t.transitions model.initial
  Json.mkObj [
    ("machine", str name),
    ("states", strs (t.states.map (·.key))),
    ("actions", strs (t.actions.map (·.key))),
    ("outcomes", strs (t.outcomes.map (·.key))),
    ("facts", strs (t.facts.map (·.key))),
    ("starts", strs (model.initial.map sk)),
    ("ends", strs (model.terminal.map sk)),
    ("reachable", strs (reachable.map sk)),
    ("transitions", Json.arr (t.transitions.map fun row => Json.mkObj [
      ("key", str row.key),
      ("source", str (sk row.source)),
      ("action", str (ak row.action)),
      ("results", Json.arr (row.results.map fun step => Json.mkObj [
        ("outcome", str (ok step.outcome)),
        ("state", str (sk step.state)),
        ("facts", strs (step.facts.map fk))]).toArray)]).toArray)]

def idsJson {S St A O F : Type} [BEq S] [BEq St] [BEq A] [BEq O] [BEq F]
    (model : DeclaredModel S St A O F) : Json :=
  Json.mkObj [
    ("target", str model.targetId.value),
    ("states", strs (model.stateIds.map (·.value))),
    ("stateFields", Json.arr (model.stateFieldIds.map fun (n, id) =>
      Json.arr #[str n, str id.value]).toArray),
    ("actions", strs (model.actionIds.map (·.value))),
    ("outcomes", strs (model.outcomeIds.map (·.value))),
    ("facts", strs (model.factIds.map (·.value)))]

def atomJson (a : ModelValue) : Json :=
  Json.mkObj [("id", str a.definitionId.value), ("value", str a.value)]

def traceJson (t : Scenario.Trace) : Json :=
  Json.mkObj [
    ("initial", atomJson t.trace.initialState),
    ("steps", Json.arr (t.trace.steps.map fun s => Json.mkObj [
      ("action", atomJson s.selectedAction),
      ("outcome", atomJson s.outcome),
      ("state", atomJson s.state),
      ("facts", Json.arr (s.facts.map atomJson).toArray)]).toArray)]

def write (dir name : String) (j : Json) : IO Unit :=
  IO.FS.writeFile (dir ++ "/" ++ name) (j.pretty ++ "\n")

end UmpireGo.Dump

open UmpireGo.Dump in
def main (args : List String) : IO UInt32 := do
  let dir := args.headD "."
  write dir "table-nexusProduct.json" (tableJson "nexusProduct" nexusProduct)
  write dir "table-nexusProtocol.json" (tableJson "nexusProtocol" nexusProtocol)
  write dir "table-workerPolling.json" (tableJson "polling" Temporal.Feature.Worker.polling)
  write dir "table-handlerWorker.json" (tableJson "handlerWorker" handlerWorker)
  write dir "table-nexusCaller.json" (tableJson "nexusCaller" nexusCaller)
  write dir "ids-workerPolling.json" (idsJson Temporal.Feature.Worker.polling)
  write dir "ids-handlerWorker.json" (idsJson handlerWorker)
  write dir "ids-nexusCaller.json" (idsJson nexusCaller)
  write dir "ids-nexusProduct.json" (idsJson nexusProduct)
  write dir "ids-nexusProtocol.json" (idsJson nexusProtocol)
  write dir "refinement-nexusProtocol.json" (Json.mkObj [
    ("rejected", match nexusProtocol.refinement.rejected with
      | some r => str (toString (repr r)) | none => Json.null),
    ("rows", Json.arr (nexusProtocol.refinement.rows.map fun (k, v) =>
      Json.mkObj [("key", str k), ("product", match v with | some p => str p | none => Json.null)]).toArray)])
  -- The canonical metadata each fingerprint hashes the semantic part of, for the Case producer's
  -- parity: the target once, and the Scenario, the Query and the witness Property of every
  -- functional Query.
  match syncCompletion with
  | .ok checked =>
      IO.FS.writeFile (dir ++ "/canonical-target-nexusProtocol.txt")
        (checked.realizable.target.canonicalMetadata ++ "\n")
  | .error _ => pure ()
  let canonicalOf := fun (name : String) (q : Except AdmissionError (CheckedModel nexusProtocol)) =>
    match q with
    | .ok checked => Json.mkObj [
        ("query", str name),
        ("targetFingerprint", str checked.realizable.target.behaviorFingerprint.render),
        ("scenario", str checked.realizable.behavior.canonicalMetadata),
        ("scenarioFingerprint", str checked.realizable.behavior.behaviorFingerprint.render),
        ("queryCanonical", str checked.query.canonicalMetadata),
        ("queryFingerprint", str checked.query.behaviorFingerprint.render),
        ("property", str checked.realizable.property.canonicalMetadata),
        ("propertyFingerprint", str checked.realizable.property.behaviorFingerprint.render)]
    | .error _ => Json.mkObj [("query", str name), ("error", str "admission failed")]
  for (name, q) in [("syncCompletion", syncCompletion), ("asyncCompletion", asyncCompletion),
      ("asyncFailure", asyncFailure), ("handlerError", handlerError), ("retry", retry),
      ("scheduleToStartTimeout", scheduleToStartTimeout), ("startToCloseTimeout", startToCloseTimeout)] do
    write dir s!"canonical-{name}.json" (canonicalOf name q)
  let queries := [("syncCompletion", syncCompletion), ("asyncCompletion", asyncCompletion),
    ("asyncFailure", asyncFailure), ("handlerError", handlerError), ("retry", retry),
    ("scheduleToStartTimeout", scheduleToStartTimeout), ("startToCloseTimeout", startToCloseTimeout)]
  for (name, q) in queries do
    write dir s!"query-{name}.json" (match q with
      | .ok checked => Json.mkObj [
          ("query", str name),
          ("outcome", str checked.run.result.outcome.name),
          ("witness", match checked.witness with | some w => traceJson w | none => Json.null)]
      | .error _ => Json.mkObj [("query", str name), ("outcome", str "admission failed")])
  write dir "query-stoppedWorkerRepliesNothing.json" (match stoppedWorkerRepliesNothing with
    | .ok checked => Json.mkObj [("query", str "stoppedWorkerRepliesNothing"),
        ("outcome", str checked.run.result.outcome.name)]
    | .error _ => Json.mkObj [("query", str "stoppedWorkerRepliesNothing"), ("outcome", str "admission failed")])
  write dir "query-terminalHolds.json" (match terminalHolds with
    | .ok checked => Json.mkObj [("query", str "terminalHolds"),
        ("outcome", str checked.run.result.outcome.name)]
    | .error _ => Json.mkObj [("query", str "terminalHolds"), ("outcome", str "admission failed")])
  write dir "targets-nexusCallerExploration.json"
    (Json.arr (nexusCallerExploration.targets.map fun t => (Umpire.CanonicalJson.compact t.json) |> Json.parse |>.toOption.getD Json.null).toArray)
  return 0

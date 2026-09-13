import Umpire.Case.Compiler

/-! Case-local names and model value spellings: the derived names, the rows that map them back, and
the rejections that keep renaming injective. -/

namespace Umpire.Case.LocalNamesTests

open Umpire
open Umpire.Case
open Umpire.Case.LocalNames
open Testpilot.Authoring
open temporal.server.api.testpilot.v1 hiding LocalName ModelValueFingerprint

/-! ### Local names -/

private def rejects {α : Type} (result : Except Error α) (expected : Error) : Bool :=
  match result with
  | .error error => error == expected
  | .ok _ => false

private def ids := ["temporal.x.evidence.operation-identity", "temporal.x.action.schedule",
  "temporal.x.state.schedule", "b.x", "a.b.x"]

-- A name is the last segment, extended leftward only while another Definition ID shares the suffix;
-- a Definition ID that is another's whole suffix keeps every segment.
#guard ids.map (localName ids) ==
  ["operation-identity", "action.schedule", "state.schedule", "b.x", "a.b.x"]

-- One local name for two Definition IDs rejects naming both.
#guard rejects (checkNames [⟨"x", "a.x"⟩, ⟨"x", "b.x"⟩]) (.sharedName "x" "a.x" "b.x")

-- Two local names for one Definition ID reject naming both.
#guard rejects (checkNames [⟨"x", "a.x"⟩, ⟨"a.x", "a.x"⟩]) (.splitDefinition "a.x" "x" "a.x")

#guard (checkNames [⟨"x", "a.x"⟩, ⟨"y", "a.y"⟩]).toOption.isSome

/-! ### Spellings -/

private def definition := "test.outcome.completed"
private def firstKey := Operation.Canonical.keyPrefix ++ "1_2_3"
private def secondKey := Operation.Canonical.keyPrefix ++ "4_5_6"
private def firstFingerprint := Fingerprint.sha256Hex firstKey
private def secondFingerprint := Fingerprint.sha256Hex secondKey
private def firstDisambiguated := "completed-" ++ String.ofList (firstFingerprint.toList.take 8)
private def secondDisambiguated := "completed-" ++ String.ofList (secondFingerprint.toList.take 8)
private def names (id : String) : String := if id == "test.command.start" then "start" else id

-- A lone structural key takes its definition's last segment; two keys that would share it take
-- fingerprint disambiguators; a Definition ID takes its local name; a declared spelling stays. Only
-- a spelling the name table alone does not give records a fingerprint.
#guard (spell names definition [firstKey]).toOption ==
  some [⟨firstKey, "completed", some firstFingerprint⟩]
#guard (spell names definition [firstKey, secondKey, firstKey, "test.command.start", "pending"]).toOption ==
  some [⟨firstKey, firstDisambiguated, some firstFingerprint⟩,
    ⟨secondKey, secondDisambiguated, some secondFingerprint⟩,
    ⟨"test.command.start", "start", none⟩, ⟨"pending", "pending", none⟩]

-- A declared spelling that shares a structural key's base spelling takes a disambiguator too.
#guard (spell names definition [firstKey, "completed"]).toOption ==
  some [⟨firstKey, firstDisambiguated, some firstFingerprint⟩,
    ⟨"completed", "completed-" ++ String.ofList ((Fingerprint.sha256Hex "completed").toList.take 8),
      some (Fingerprint.sha256Hex "completed")⟩]

-- A declared spelling that equals another encoding's disambiguated spelling stays ambiguous, and
-- rejects naming the definition and both encodings.
#guard rejects (spell names definition [firstKey, secondKey, firstDisambiguated])
  (.ambiguousSpelling definition firstDisambiguated firstKey firstDisambiguated)

/-! ### Production -/

private def property : Provenance.DefinitionBinding :=
  { definitionId := "test.property", behaviorFingerprint := "test-property/v1", kind := .property }

private def value (definitionId value : String) : temporal.server.api.testpilot.v1.ModelValue :=
  { definition_id := definitionId, value }

private def capability (response : String) (third : String := "done") : CorrelatedContract :=
  Testpilot.Authoring.Contract.correlated "test.projection" "sha256:projection" "evidence" "test.scope.operation"
    #["test.scope.run"] #["test.source.history"] (value "test.state.pending" "pending")
    #[{ prior_state := some (value "test.state.pending" "pending")
        action := some (value "test.action.schedule" "test.command.start")
        state := some (value "test.state.pending" "pending")
        outcome := some (value definition firstKey) },
      { prior_state := some (value "test.state.pending" "pending")
        action := some (value "test.action.schedule" "test.command.start")
        state := some (value "test.state.pending" "pending")
        outcome := some (value definition secondKey) },
      { prior_state := some (value "test.state.pending" "pending")
        action := some (value "test.action.schedule" "test.command.start")
        state := some (value "test.state.pending" "pending")
        outcome := some (value definition third) }]
    #[{ kind := "test.evidence.completed", meaning := .CORRELATED_EVIDENCE_MEANING_IRRELEVANT }]
    #[Testpilot.Authoring.Contract.correlatedRule "test.clause.completion" 1 .TRACE_ENDING_PARTIAL
      (Expr.present (Expr.correlatedStep .CORRELATED_STEP_FIELD_ACTION "test.action.schedule"))
      (Expr.equal (Expr.correlatedStep .CORRELATED_STEP_FIELD_OUTCOME definition)
        (Expr.literal (Value.text response)))]

private def input (capability : CorrelatedContract) : Compiler.Input := {
  version := { major := 1 }
  caseId := "test.case"
  producerId := "test.producer"
  definitions := [property]
  sources := []
  knownGaps := []
  program := Program.make "test.program" #[] #[] #[] #[] (Program.cleanup "cleanup" #[])
  contractId := "test.contract"
  properties := [.correlated property capability []]
}

private def rows (output : temporal.server.api.testpilot.v1.Case) :=
  (output.provenance.map fun provenance =>
    (provenance.local_names.toList.map fun row => (row.local_name, row.definition_id),
     provenance.model_value_fingerprints.toList.map fun row =>
      (row.local_name, row.spelling, row.fingerprint))).getD ([], [])

-- The compiled Case carries local names and spellings, the step condition's text takes the spelling
-- of the encoding it compares, and provenance maps every renamed name and value back.
#guard match Compiler.compile (input (capability firstKey)) with
  | .ok output =>
      rows output ==
        ([("projection", "test.projection"), ("run", "test.scope.run"),
          ("operation", "test.scope.operation"), ("history", "test.source.history"),
          ("pending", "test.state.pending"), ("start", "test.command.start"),
          ("schedule", "test.action.schedule"), ("outcome.completed", definition),
          ("evidence.completed", "test.evidence.completed"), ("completion", "test.clause.completion")],
         [("outcome.completed", firstDisambiguated, firstFingerprint),
          ("outcome.completed", secondDisambiguated, secondFingerprint)]) &&
      (match output.contract.bind (·.«correlated») with
        | some wire =>
            wire.transitions.map (fun row => (row.action.map (·.value), row.outcome.map (·.value))) ==
              #[(some "start", some firstDisambiguated), (some "start", some secondDisambiguated),
                (some "start", some "done")] &&
            (match wire.rules.toList.map (·.response.bind (·.expression)) with
              | [some (Expression.expression_Type.compare comparison)] =>
                  (match comparison.left.bind (·.expression), comparison.right.bind (·.expression) with
                    | some (.reference { reference := some (.correlated_step step), .. }),
                      some (.literal { value := some (.text_value text), .. }) =>
                        step.definition_id == "outcome.completed" && text == firstDisambiguated
                    | _, _ => false)
              | _ => false)
        | none => false)
  | .error _ => false

-- A value spelling that stays ambiguous after disambiguation rejects the Case at production, naming
-- the definition and both encodings.
#guard match Compiler.compile (input (capability firstKey (third := firstDisambiguated))) with
  | .error error =>
      error.sourceDefinitionId == definition &&
        error.construct == s!"model-value-spelling {firstDisambiguated} of {definition} spells {firstKey} and {firstDisambiguated}"
  | .ok _ => false

end Umpire.Case.LocalNamesTests

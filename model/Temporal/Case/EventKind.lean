import Temporal.API

/-!
# Recorded history event kinds

An `evidence` line names the recorded event that confirms one Action. The admitted names are the
`HistoryEvent.Attributes` oneof of the generated schema, read from that schema rather than listed
here: a server release that adds an event kind adds it to the admitted list with no edit, and a
misspelling rejects against what the schema actually carries.

The author-facing spelling is the field name without its `_event_attributes` suffix, in
lowerCamelCase -- `nexus_operation_started_event_attributes` is written `nexusOperationStarted`.
-/

namespace Temporal.Case.EventKind

/-- The generated message whose oneof names every recorded history event kind. -/
def historyEventMessage : String := "temporal.api.history.v1.HistoryEvent"

/-- The oneof within it. Every admitted evidence kind is one of its fields. -/
def attributesOneof : String := "attributes"

private def attributesSuffix : String := "_event_attributes"

/-- The history read whose response closure carries the `HistoryEvent` schema. Reading the kinds
off this method's own closure is what ties the admitted list to the generated API rather than to a
list maintained here. -/
private def historyMethod := Temporal.Api.Workflowservice.V1.WorkflowService.getWorkflowExecutionHistory

private def historyReference : Temporal.API.MethodReference historyMethod := by constructor

/-- Every `attributes` oneof field of the generated `HistoryEvent`, in descriptor order. -/
def attributeFields : List String :=
  match (historyReference.schema.response.nodes.find? (·.name == historyEventMessage)).bind
      (·.valueShape) with
  | some (.message fields _) =>
      fields.filterMap fun field =>
        match field.presence with
        | .oneof name => if name == attributesOneof then some field.name else none
        | _ => none
  | _ => []

private def capitalizeFirst (segment : String) : String :=
  match segment.toList with
  | [] => segment
  | first :: rest => String.ofList (first.toUpper :: rest)

/-- `nexus_operation_started_event_attributes` reads `nexusOperationStarted`. A field that does not
end in the suffix keeps its whole name, so an unexpected shape stays visible rather than silently
losing characters. -/
def spelling (field : String) : String :=
  let stem := if field.endsWith attributesSuffix
    then (field.dropEnd attributesSuffix.length).toString
    else field
  match stem.splitOn "_" with
  | [] => stem
  | head :: rest => head ++ String.join (rest.map capitalizeFirst)

/-- Every event kind an `evidence` line may name, in descriptor order. -/
def admitted : List String := attributeFields.map spelling

/-- The generated attributes field one authored kind names. -/
def attributesField? (kind : String) : Option String :=
  attributeFields.find? fun field => spelling field == kind

/-- What a rejection lists: every admitted spelling, comma separated. -/
def admittedList : String := ", ".intercalate admitted

/-- Resolve one authored kind to its generated attributes field, or say what is admitted. The
message is the located diagnostic an `evidence` line reports. -/
def resolve (kind : String) : Except String String :=
  match attributesField? kind with
  | some field => .ok field
  | none => .error s!"unknown history event kind '{kind}'; admitted: {admittedList}"

end Temporal.Case.EventKind

import Temporal.API
import Temporal.Case.EventKind
import Umpire.Command

/-!
# Resolving a field relation's paths against the generated API

A `property … relates:` line names typed fields: an action's input or result field through the
protobuf message its `schema:` names, or a field of the recorded history event that confirms the
step. `Umpire` hands each dotted path to the platform; this module is Temporal's answer, and what
it answers is read off the generated descriptors.

An action's `schema:` message is the request (or response) of one of the workflow-service methods
a realization invokes, and the field path is walked from that message through the method's
descriptor closure: each segment is a field of the message reached so far, an optional message or a
oneof member is a presence the read establishes, and the path ends at a scalar. A recorded event
kind is one arm of the generated `HistoryEvent`'s `attributes` oneof, read back through
`GetWorkflowExecutionHistory`: the walk begins at the response, through `history` and the event it
records, into the arm the kind names, and continues with the author's segments -- the same
coordinates a hand-written field Property spelled out step by step.

The answer names the constant that holds the method's schema rather than carrying the schema, so
a Model stores no descriptor.
-/

namespace Temporal.Case.FieldPath

open Umpire
open Umpire.Command

private def startMethod := Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution
private def startReference : Temporal.API.MethodReference startMethod := by constructor
private def historyMethod := Temporal.Api.Workflowservice.V1.WorkflowService.getWorkflowExecutionHistory
private def historyReference : Temporal.API.MethodReference historyMethod := by constructor

/-- The generated `StartWorkflowExecution` schema, as a resolved operand names it. -/
def startWorkflowSchema : Operation.RpcSchema := startReference.schema

/-- The generated `GetWorkflowExecutionHistory` schema, which every recorded event is read
through. -/
def getWorkflowExecutionHistorySchema : Operation.RpcSchema := historyReference.schema

/-- The methods whose request or response an action's `schema:` may name, each with the constant
that names its schema. A message outside these rejects; a Model that needs another method adds it
here, which keeps the admitted set tied to what a realization invokes. -/
private def methods : List (Lean.Name × Operation.RpcSchema) := [
  (``startWorkflowSchema, startWorkflowSchema),
  (``getWorkflowExecutionHistorySchema, getWorkflowExecutionHistorySchema)]

/-- One walk through a message: the steps so far, the presence reads they traversed, and where the
walk stands. -/
private structure Walk where
  steps : List Value.Field.Step
  presence : List (List Value.Field.Step)

private def Walk.field (walk : Walk) (containing : String) (field : Operation.ValueField) : Walk :=
  let steps := walk.steps ++ [Value.Field.Step.field containing field.number]
  match Value.Field.availability field with
  | .optional =>
      { steps := steps ++ [Value.Field.Step.establish]
        presence := walk.presence ++ [steps ++ [Value.Field.Step.present]] }
  | .oneof group =>
      { steps := steps ++ [Value.Field.Step.select group]
        presence := walk.presence ++ [steps ++ [Value.Field.Step.present]] }
  | .available => { steps, presence := walk.presence }

/-- The field one segment names inside one message of the schema. -/
private def fieldOf (schema : Operation.Schema) (containing segment : String) :
    Except String Operation.ValueField :=
  match (Value.Field.schemaFields schema).find? fun (node, field) =>
      node == containing && field.name == segment with
  | some (_, field) => .ok field
  | none => .error s!"'{segment}' is not a field of {containing}"

/-- Walk the author's segments from one message, ending at a scalar. -/
private partial def walk (schema : Operation.Schema) (containing : String) (segments : List String)
    (state : Walk) : Except String (Walk × Operation.Singular) := do
  match segments with
  | [] => throw s!"the path ends at message {containing}; name one of its fields"
  | segment :: rest =>
      let field ← fieldOf schema containing segment
      match field.cardinality with
      | .repeated =>
          throw s!"'{segment}' is a repeated field; a path through a repeated field selects by a \
correlation key, which no relation declares"
      | .map _ => throw s!"'{segment}' is a map field, which a relation cannot read through"
      | .singular => pure ()
      let state := state.field containing field
      match field.type, rest with
      | .message name, [] => throw s!"'{segment}' is a message ({name}); name one of its fields"
      | .message name, rest => walk schema name rest state
      | .floating _, _ => throw s!"'{segment}' is a floating-point field, which is not comparable"
      | .unsupported reason, _ => throw s!"'{segment}' is {reason}, which is not comparable"
      | scalar, [] => pure (state, scalar)
      | _, extra :: _ => throw s!"'{segment}' is a scalar; '{extra}' is not a field inside it"

/-- An action's input or result field: the message the action's `schema:` names is the request of
one of the admitted methods, and the path is walked from that request or from the method's
response. -/
private def actionOperand (kind : FieldOperandKind) (messages : List String)
    (segments : List String) : Except String ResolvedField := do
  let side : Value.Side := if kind == .input then .request else .response
  let some (schemaName, rpc) := methods.find? fun (_, rpc) => messages.contains rpc.request.root
    | throw s!"'{", ".intercalate messages}' is not the request of a method the realization \
invokes"
  let schema := if side == .request then rpc.request else rpc.response
  let (state, type) ← walk schema schema.root segments { steps := [], presence := [] }
  pure { steps := state.steps, type, side, schemaName, presence := state.presence }

/-- A recorded event's field: read back through the history response, into the `attributes` arm
the kind names, then along the author's segments. -/
private def observationOperand (kind : String) (segments : List String) :
    Except String ResolvedField := do
  let some attributes := EventKind.attributesField? kind
    | throw s!"'{kind}' is not a recorded history event kind, so it carries no schema to name a \
field of"
  let schema := getWorkflowExecutionHistorySchema.response
  let history ← fieldOf schema schema.root "history"
  let state := (Walk.mk [] []).field schema.root history
  let some historyMessage := (match history.type with | .message name => some name | _ => none)
    | throw "the history response carries no history message"
  let events ← fieldOf schema historyMessage "events"
  let state := { state with
    steps := state.steps ++
      [Value.Field.Step.field historyMessage events.number, Value.Field.Step.index 0] }
  let arm ← fieldOf schema EventKind.historyEventMessage attributes
  let state := state.field EventKind.historyEventMessage arm
  let some armMessage := (match arm.type with | .message name => some name | _ => none)
    | throw s!"'{kind}' carries no message"
  let (state, type) ← walk schema armMessage segments state
  pure { steps := state.steps
         type
         side := .response
         schemaName := ``getWorkflowExecutionHistorySchema
         presence := state.presence }

/-- The platform's answer for one operand. -/
def resolve (spec : FieldOperandSpec) : Except String ResolvedField :=
  match spec.kind with
  | .input | .result => actionOperand spec.kind spec.schema spec.segments
  | .observation =>
      match spec.schema with
      | [kind] => observationOperand kind spec.segments
      | _ => .error "an observation operand names one recorded event kind"

initialize Umpire.Command.installFieldResolver resolve

end Temporal.Case.FieldPath

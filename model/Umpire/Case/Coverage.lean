import Testpilot.Authoring
import Umpire.Case
import Umpire.Property.Fields

/-!
Whole-Case coverage: the checked map from selected modeled fields and requested clauses onto the
Case that will execute them.

A Case has two independent construction boundaries. A modeled input field is *constructed* by the
Program: exactly one request assignment of one named instruction must target its coordinates and
supply its exact value. A modeled result or event field is *observed*: the projection-level
`Umpire.Observation.Projection.Coverage` maps it onto a declared Observation, and the scoped
lowering consumes that map. A requested clause is *lowered*: it must appear exactly once among the
Case's compiled scoped clause bindings.

`Umpire.Case.Compiler.Input.coverage` carries the requested map, and `compile` admits it before it
assembles anything. A requested input field with no assignment, an assignment that constructs a
different value, an unsupported coordinate, and a requested clause that was not lowered are all
whole-Case source-owned rejections, so no unsupported mapping can reach Driver I/O.
-/

namespace Umpire.Case.Coverage

open Umpire
open Umpire.Case
open temporal.server.api.testpilot.v1

/-- A source-owned coverage rejection naming the modeled field or clause at fault. -/
structure Error where
  subject : String
  reason : String
  deriving BEq, DecidableEq, Repr

/-- One modeled input field and the Program request assignment that must construct it. -/
structure InputMapping where
  /-- Request-rooted modeled coordinates; result and event fields are covered by Observations. -/
  path : PropertyFieldPath
  /-- The exact modeled value the assignment must supply at those coordinates. -/
  value : Operation.Scalar
  entrypointId : String
  instructionId : String
  deriving BEq, DecidableEq, Repr

/-- The coverage a Case requests: how its modeled input fields are constructed and which clauses it
requires to have been lowered. A Case that requests neither keeps its existing meaning. -/
structure Request where
  inputs : List InputMapping := []
  clauses : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

/-- The exact portable value one modeled scalar constructs. Integer signedness and enum identity are
preserved, and bytes are the concrete bytes; a floating-point value has no exact construction here
and rejects with its own diagnostic rather than being approximated. -/
def scalarValue : Operation.Scalar → Except String temporal.server.api.testpilot.v1.Value
  | .text text => .ok { value := some (.text text) }
  | .boolean flag => .ok { value := some (.bool_value flag) }
  | .bytes bytes => .ok { value := some (.bytes_value ⟨bytes.toArray⟩) }
  | .integer kind number =>
      if kind.signed then .ok { value := some (.signed_integer (toString number)) }
      else if number ≥ 0 then .ok { value := some (.unsigned_integer (toString number)) }
      else .error "unsigned request field cannot construct a negative value"
  | .enumeration _ number =>
      if number ≥ -2147483648 && number ≤ 2147483647 then
        .ok { value := some (.enum_value { number := Int32.ofInt number }) }
      else .error "enum number is outside the int32 range"
  | .floating _ _ => .error "unsupported floating-point request construction"

/-- The Program field path a modeled operand's structural steps construct. Presence steps construct
nothing: an optional field is present because the assignment supplied it, and a oneof member is
selected because it is the one supplied. -/
def targetPath (schema : Operation.Schema) : List Value.Field.Step → Except String (List String)
  | [] => .ok []
  | .field containing number :: rest => do
      let some field := (Value.Field.schemaFields schema).find? fun item =>
        item.1 == containing && item.2.number == number
        | .error ("unknown containing schema or field " ++ containing)
      pure (field.2.name :: (← targetPath schema rest))
  | .establish :: rest => targetPath schema rest
  | .select _ :: rest => targetPath schema rest
  | step :: _ => .error ("unsupported input coverage step " ++ reprStr step)

/-- Exact equality of the two constructible value forms. A value form this coverage cannot
construct is never equal to a covered field's value. -/
private def sameValue (left right : temporal.server.api.testpilot.v1.Value) : Bool :=
  match left.value, right.value with
  | some (.text first), some (.text second) => first == second
  | some (.bool_value first), some (.bool_value second) => first == second
  | some (.bytes_value first), some (.bytes_value second) => first.toList == second.toList
  | some (.signed_integer first), some (.signed_integer second) => first == second
  | some (.unsigned_integer first), some (.unsigned_integer second) => first == second
  | some (.enum_value first), some (.enum_value second) => first.number == second.number
  | _, _ => false

private def assignmentSegments (path : FieldPath) : Except String (List String) :=
  path.segments.toList.mapM fun segment =>
    if segment.selector.isSome then .error "unsupported request assignment selector"
    else .ok segment.field

/-- Admit one requested input mapping against the Program that must construct it. -/
private def checkInput (program : Program) (mapping : InputMapping) : Except String Unit := do
  if mapping.path.root != .request then
    throw "input coverage names request coordinates; results and events are covered by Observations"
  if mapping.path.side != .request then throw "input coverage names the request payload"
  let expected ← targetPath mapping.path.schema.request mapping.path.steps
  let value ← scalarValue mapping.value
  let some entrypoint := program.entrypoints.toList.find? (·.entrypoint_id == mapping.entrypointId)
    | throw ("unknown entrypoint " ++ mapping.entrypointId)
  let some instruction := entrypoint.instructions.toList.find?
    (·.instruction_id == mapping.instructionId)
    | throw ("unknown instruction " ++ mapping.instructionId)
  let some (.invoke_rpc call) := instruction.instruction.bind (·.instruction)
    | throw ("instruction " ++ mapping.instructionId ++ " constructs no request")
  let mut matched := 0
  for assignment in call.request_assignments.toList do
    let some target := assignment.target | throw "request assignment has no target"
    if (← assignmentSegments target) == expected then
      matched := matched + 1
      let some expression := assignment.value | throw "request assignment supplies no value"
      let some (.literal supplied) := expression.expression
        | throw "covered input field is not constructed from an exact value"
      if !sameValue supplied value then
        throw "request assignment constructs a different value than the modeled field"
  if matched != 1 then
    throw ("covered input field has " ++ toString matched ++ " request assignments")

/-- Admit the requested coverage against the Case being assembled. Every requested input field must
be constructed exactly once by the named instruction, and every requested clause must appear exactly
once among the lowered scoped clause bindings. -/
def check (program : Program) (clauses : List CaseScopedClauseBinding) (request : Request)
    (caseId : String) : Except Error Unit := do
  if (request.inputs.map (·.path)).eraseDups.length != request.inputs.length then
    throw ⟨caseId, "duplicate input field coverage"⟩
  if request.clauses.eraseDups.length != request.clauses.length then
    throw ⟨caseId, "duplicate clause coverage"⟩
  for mapping in request.inputs do
    match checkInput program mapping with
    | .error reason => throw ⟨mapping.path.reference.value, reason⟩
    | .ok () => pure ()
  for clause in request.clauses do
    if (clauses.filter (·.clauseId == clause.value)).length != 1 then
      throw ⟨clause.value, "requested clause was not lowered exactly once"⟩

end Umpire.Case.Coverage

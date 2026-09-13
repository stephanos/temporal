import Testpilot.Authoring
import Umpire.Provenance
import Umpire.Case.Projection.Coordinates

/-!
Whole-Case coverage: the checked map from selected modeled fields and requested clauses onto the
Case that will execute them.

A Case has two independent construction boundaries. A modeled input field is *constructed* by the
Program: exactly one request assignment of one named instruction must target its coordinates and
supply its exact value. A modeled result or event field is *observed*: the projection-level
`Umpire.Case.Projection.Coverage` maps it onto a declared Observation, and the correlated
lowering consumes that map. A requested clause is *lowered*: it must appear exactly once among the
Case's compiled correlated rule bindings.

`Umpire.Case.Compiler.Input.coverage` carries the requested map, and `compile` admits it before it
assembles anything. A requested input field with no assignment, an assignment that constructs a
different value, an unsupported coordinate, and a requested clause that was not lowered are all
whole-Case source-owned rejections, so no unsupported mapping can reach Driver I/O.

The coordinates a mapping names are walked by `Coverage.targetPath`, the construct use of the one
coordinate walker in `Umpire.Case.Projection.Coordinates`.
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

/-! ### Enum value names

An Umpire value shape records only an enum's numbers, so a name is read from the enum node's exact
descriptor bytes, an `EnumDescriptorProto`. The reader below walks those bytes by hand: the protobuf
library's decoder and `String` iteration both depend on `Classical.choice`, which the lowering that
names enum literals does not. -/

/-- The value of one lowercase hexadecimal digit byte. -/
private def hexValue (digit : UInt8) : Option Nat :=
  if 48 ≤ digit ∧ digit ≤ 57 then some (digit.toNat - 48)
  else if 97 ≤ digit ∧ digit ≤ 102 then some (digit.toNat - 87)
  else none

/-- The bytes a hex-encoded descriptor spells. -/
private def hexBytes (text : String) : Option ByteArray := Id.run do
  let digits := text.toUTF8
  if digits.size % 2 != 0 then return none
  let mut bytes := ByteArray.empty
  for index in [0:digits.size / 2] do
    let some high := hexValue (digits.get! (2 * index)) | return none
    let some low := hexValue (digits.get! (2 * index + 1)) | return none
    bytes := bytes.push (high * 16 + low).toUInt8
  return some bytes

/-- The base-128 varint at `position` and the position after it. -/
private def readVarint (bytes : ByteArray) (position : Nat) : Option (Nat × Nat) := Id.run do
  let mut value := 0
  for offset in [0:10] do
    if position + offset ≥ bytes.size then return none
    let byte := (bytes.get! (position + offset)).toNat
    value := value + (byte % 128) * 128 ^ offset
    if byte < 128 then return some (value, position + offset + 1)
  return none

/-- One field of a message's wire bytes: its number, wire type, varint value (wire type 0) and
payload bounds (wire type 2). -/
private structure WireField where
  number : Nat
  wireType : Nat
  value : Nat := 0
  start : Nat := 0
  stop : Nat := 0

/-- The fields of the message whose wire bytes lie in `[start, stop)`, or none when they are not
well-formed. -/
private def wireFields (bytes : ByteArray) (start stop : Nat) : Option (Array WireField) := Id.run do
  let mut fields := #[]
  let mut position := start
  for _ in [start:stop] do
    if position ≥ stop then break
    let some (tag, next) := readVarint bytes position | return none
    let number := tag / 8
    match tag % 8 with
    | 0 =>
        let some (value, after) := readVarint bytes next | return none
        fields := fields.push { number, wireType := 0, value }
        position := after
    | 2 =>
        let some (length, payload) := readVarint bytes next | return none
        fields := fields.push { number, wireType := 2, start := payload, stop := payload + length }
        position := payload + length
    | 1 => position := next + 8
    | 5 => position := next + 4
    | _ => return none
  if position == stop then some fields else none

/-- The text of an identifier's bytes; protobuf names are ASCII. -/
private def asciiText (bytes : ByteArray) (start stop : Nat) : Option String := Id.run do
  let mut text := ""
  for index in [start:stop] do
    let byte := (bytes.get! index).toNat
    if byte ≥ 128 then return none
    text := text.push (Char.ofNat byte)
  return some text

/-- The name `number` has in the enum `enumName` that `schema` declares. -/
def enumValueName (schema : Operation.Schema) (enumName : String) (number : Int) :
    Except String String := do
  let some node := schema.nodes.find? (·.name == enumName)
    | throw ("the schema declares no enum " ++ enumName)
  let malformed := "enum " ++ enumName ++ " has no well-formed descriptor"
  let some bytes := hexBytes node.descriptor | throw malformed
  let some fields := wireFields bytes 0 bytes.size | throw malformed
  for value in fields do
    if value.number != 2 || value.wireType != 2 then continue
    let some parts := wireFields bytes value.start value.stop | throw malformed
    -- An int32 is encoded as the 64-bit two's complement of its value.
    let declared : Int := match parts.find? (fun part => part.number == 2 && part.wireType == 0) with
      | some part => if part.value ≥ 2 ^ 63 then part.value - 2 ^ 64 else part.value
      | none => 0
    if declared != number then continue
    let some name := parts.find? (fun part => part.number == 1 && part.wireType == 2)
      | throw malformed
    let some text := asciiText bytes name.start name.stop | throw malformed
    return text
  throw ("enum " ++ enumName ++ " declares no number " ++ toString number)

/-- The exact portable value one modeled scalar constructs. Integer signedness and enum identity are
preserved, an enum value is named as `schema` declares it, and bytes are the concrete bytes; a
floating-point value has no exact construction here and rejects with its own diagnostic rather than
being approximated. -/
def scalarValue (schema : Operation.Schema) :
    Operation.Scalar → Except String temporal.server.api.testpilot.v1.Value
  | .text text => .ok { value := some (.text_value text) }
  | .boolean flag => .ok { value := some (.bool_value flag) }
  | .bytes bytes => .ok { value := some (.bytes_value ⟨bytes.toArray⟩) }
  | .integer kind number =>
      if kind.signed then .ok { value := some (.signed_integer_value (toString number)) }
      else if number ≥ 0 then .ok { value := some (.unsigned_integer_value (toString number)) }
      else .error "unsigned request field cannot construct a negative value"
  | .enumeration enumName number => do
      if number < -2147483648 || number > 2147483647 then
        throw "enum number is outside the int32 range"
      pure { value := some (.enum_value { name := ← enumValueName schema enumName number }) }
  | .floating _ _ => .error "unsupported floating-point request construction"

/-- The key a map selector writes for one modeled scalar; protobuf map keys are text, integers or
booleans. -/
def pathKey : Operation.Scalar → Except String Testpilot.Authoring.Path.Key
  | .text text => .ok (.text text)
  | .boolean flag => .ok (.boolean flag)
  | .integer _ number => .ok (.integer number)
  | _ => .error "a map key is text, an integer or a boolean"

/-- Exact equality of the two constructible value forms. A value form this coverage cannot
construct is never equal to a covered field's value. -/
private def sameValue (left right : temporal.server.api.testpilot.v1.Value) : Bool :=
  match left.value, right.value with
  | some (.text_value first), some (.text_value second) => first == second
  | some (.bool_value first), some (.bool_value second) => first == second
  | some (.bytes_value first), some (.bytes_value second) => first.toList == second.toList
  | some (.signed_integer_value first), some (.signed_integer_value second) => first == second
  | some (.unsigned_integer_value first), some (.unsigned_integer_value second) => first == second
  | some (.enum_value first), some (.enum_value second) => first.name == second.name
  | _, _ => false

/-- The Program field path a modeled input field's coordinates construct, as `Path.make` spells it.
`Path.make` is the one printer, so an assignment that targets the field spells exactly this path. -/
private def assignmentTarget (schema : Operation.Schema) (steps : List Value.Field.Step) :
    Except String String := do
  let segments ← targetPath schema pathKey steps
  pure (Testpilot.Authoring.Path.make (segments.map fun (field, key) => match key with
    | none => Testpilot.Authoring.Path.field field
    | some key => Testpilot.Authoring.Path.mapKey field key).toArray)

/-- Admit one requested input mapping against the Program that must construct it. -/
private def checkInput (program : Program) (mapping : InputMapping) : Except String Unit := do
  if mapping.path.root != .request then
    throw "input coverage names request coordinates; results and events are covered by Observations"
  if mapping.path.side != .request then throw "input coverage names the request payload"
  let expected ← assignmentTarget mapping.path.schema.request mapping.path.steps
  let value ← scalarValue mapping.path.schema.request mapping.value
  let some entrypoint := program.entrypoints.toList.find? (·.entrypoint_id == mapping.entrypointId)
    | throw ("unknown entrypoint " ++ mapping.entrypointId)
  let some instruction := entrypoint.instructions.toList.find?
    (·.instruction_id == mapping.instructionId)
    | throw ("unknown instruction " ++ mapping.instructionId)
  let some (.invoke_rpc call) := instruction.instruction.bind (·.instruction)
    | throw ("instruction " ++ mapping.instructionId ++ " constructs no request")
  let mut matched := 0
  for assignment in call.request_assignments.toList do
    if assignment.target == expected then
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
once among the lowered correlated rule bindings. -/
def check (program : Program) (rules : List Provenance.CorrelatedRuleBinding) (request : Request)
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
    if (rules.filter (·.ruleId == clause.value)).length != 1 then
      throw ⟨clause.value, "requested clause was not lowered exactly once"⟩

end Umpire.Case.Coverage

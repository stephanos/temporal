import Fixture.API
import Umpire.Value

open Fixture.Messaging.Internal.V1.MessagingService
open Fixture.Messaging.Public.V1
open Umpire.Operation

private def error? (result : Except Error α) : Option Error :=
  match result with
  | .ok _ => none
  | .error error => some error

#guard (Fixture.API.bindUnary unary).toOption.isSome
#guard (Fixture.API.bindUnary alternate).toOption.isSome
#guard error? (Fixture.API.bindUnary upload) ==
  some (.unsupportedStreaming "fixture.messaging.internal.v1.MessagingService.Upload")
#guard error? (Fixture.API.bindUnary download) ==
  some (.unsupportedStreaming "fixture.messaging.internal.v1.MessagingService.Download")
#guard error? (Fixture.API.bindUnary chat) ==
  some (.unsupportedStreaming "fixture.messaging.internal.v1.MessagingService.Chat")

private def unaryReference : Fixture.API.MethodReference unary := by constructor
private def alternateReference : Fixture.API.MethodReference alternate := by constructor

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
#check Fixture.API.bindUnary unary alternateReference

#guard error? (Fixture.API.bindUnary unary unaryReference alternateReference.schema) ==
  some (.wrongMethod "fixture.messaging.internal.v1.MessagingService.Unary"
    "fixture.messaging.internal.v1.MessagingService.Alternate")
#guard error? (Fixture.API.bindUnary unary unaryReference
  { unaryReference.schema with request := { unaryReference.schema.request with nodes := [] } }) ==
  some (.incompatibleRequest "fixture.messaging.internal.v1.MessagingService.Unary")
#guard error? (Fixture.API.bindUnary unary unaryReference
  { unaryReference.schema with request := { unaryReference.schema.request with
      nodes := unaryReference.schema.request.nodes.map fun node =>
        { node with descriptor := node.descriptor ++ "00" } } }) ==
  some (.incompatibleRequest "fixture.messaging.internal.v1.MessagingService.Unary")
#guard error? (Fixture.API.bindUnary unary unaryReference
  { unaryReference.schema with schemaInputs := [] }) ==
  some (.incompatibleSchema "fixture.messaging.internal.v1.MessagingService.Unary")
#guard error? (Fixture.API.bindUnary unary unaryReference
  { unaryReference.schema with response := unaryReference.schema.request }) ==
  some (.incompatibleResponse "fixture.messaging.internal.v1.MessagingService.Unary")
#guard error? (Fixture.API.bindUnary unary unaryReference
  { unaryReference.schema with serverStreaming := true }) ==
  some (.incompatibleStreaming "fixture.messaging.internal.v1.MessagingService.Unary")

private def forged : Fixture.API.Proto.Method Message Reply :=
  { unary with fullName := "fixture.messaging.internal.v1.MessagingService.Forged" }

/-- error: Tactic `constructor` failed -/
#guard_msgs (error, substring := true) in
#check Fixture.API.bindUnary forged

private def wrongRequest : Fixture.API.Proto.Method Reply Reply :=
  { fullName := unary.fullName, clientStreaming := false, serverStreaming := false, deprecated := false }

/-- error: Tactic `constructor` failed -/
#guard_msgs (error, substring := true) in
#check Fixture.API.bindUnary wrongRequest

private def wrongResponse : Fixture.API.Proto.Method Message Message :=
  { fullName := unary.fullName, clientStreaming := false, serverStreaming := false, deprecated := false }

/-- error: Tactic `constructor` failed -/
#guard_msgs (error, substring := true) in
#check Fixture.API.bindUnary wrongResponse

private def forgedUnary : Fixture.API.Proto.Method Message Reply :=
  { upload with clientStreaming := false }

/-- error: Tactic `constructor` failed -/
#guard_msgs (error, substring := true) in
#check Fixture.API.bindUnary forgedUnary

example : unaryReference.schema.fullName =
    "fixture.messaging.internal.v1.MessagingService.Unary" := rfl

private def unaryOperation : Except Error (RpcDeclaration Fixture.API.rpcOwner Message Reply Empty) :=
  (Fixture.API.bindUnary unary).map (rpc Empty)

#guard unaryOperation.toOption.map (·.identity.value) ==
  some "fixture.messaging.internal.v1.MessagingService.Unary"

example (declaration : RpcDeclaration Fixture.API.rpcOwner Request Response Failure) :
    declaration.schema = declaration.reference.2.schema := declaration.schema_eq

namespace ConcreteValues
open Umpire.Value
private def limits : Limits := ⟨16, 100000, 4096, 100⟩
private def value := message "fixture.messaging.public.v1.Message" [
  (5, literal (.text "")), (6, literal (.bytes [0, 255])),
  (7, message "fixture.messaging.public.v1.Message.Nested" [
    (1, literal (.enumeration "fixture.messaging.public.v1.Message.Nested.State" 99))])]
private def admitted := check Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .request limits value
#guard admitted.toOption.isSome
#guard admitted.toOption.map (·.value) == some (message "fixture.messaging.public.v1.Message" [
  (2, map []), (5, literal (.text "")), (6, literal (.bytes [0, 255])),
  (7, message "fixture.messaging.public.v1.Message.Nested" [
    (1, literal (.enumeration "fixture.messaging.public.v1.Message.Nested.State" 99))])])
#guard (admitted.toOption.bind fun value =>
  (decodeCanonical Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .request limits (encode value)).toOption.map (·.value)) ==
  admitted.toOption.map (·.value)
#guard (unaryReference.schema.request.nodes.find? (·.name == "fixture.protobuf.compat.v1.LegacyOptions")).bind
  (fun node => match node.valueShape with
    | some (.message fields _) => (fields.find? (·.number == 2)).bind (·.defaultValue)
    | _ => none) == some (.integer .int32 7)

#guard (unaryReference.schema.request.nodes.find? (·.name == "fixture.protobuf.compat.v1.LegacyOptions")).bind
  (fun node => match node.valueShape with
    | some (.message fields _) => (fields.find? (·.number == 1)).bind fun field =>
      match field.defaultValue with
      | some (.text value) => some (value.toList.map Char.toNat)
      | _ => none
    | _ => none) == some [7, 8, 12, 11, 34, 92, 10, 13, 9, 0, 127, 233, 128512]

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
#check (admitted : Except Umpire.Value.Error
  (Checked Fixture.API.rpcOwner (Request := Bool) (Response := Nat) ⟨unary, unaryReference⟩ .request limits))
end ConcreteValues

namespace FieldAccess
open Umpire.Value
private def source : Umpire.SourceLocation := ⟨"field-clause.lean", 19, 4, "authored"⟩
#guard (Fixture.API.fields unary .request).length == 15
#guard (Fixture.API.fieldReference unary .request "fixture.messaging.public.v1.Message" 6 source).toOption.isSome
private def limits : Limits := ⟨16, 100000, 8192, 100⟩
private def input := message "fixture.messaging.public.v1.Message" [
  (2, map [(.text "z", message "fixture.messaging.shared.v1.Shared" [(1, literal (.text "last"))])]),
  (3, literal (.text "")), (5, literal (.text "")), (6, literal (.bytes [0, 255])),
  (7, message "fixture.messaging.public.v1.Message.Nested" [
    (1, literal (.enumeration "fixture.messaging.public.v1.Message.Nested.State" 99))]),
  (8, message "fixture.protobuf.compat.v1.LegacyOptions" [
    (1, literal (.text "legacy")), (3, repeated [literal (.integer .int32 (-1)), literal (.integer .int32 7)])])]
private def admitted := check Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .request limits input
private def selectField (number : Nat) (type : Singular) (card : Cardinality)
    (availability : Field.Availability) := do
  let value ← admitted.mapError fun (e : Umpire.Value.Error) => Field.Error.mk source e.path e.reason
  let ref ← Fixture.API.fieldReference unary .request "fixture.messaging.public.v1.Message" number source
  let cursor ← (Field.root value).field ref source
  cursor.refine type card availability source
private def reason (result : Except Field.Error α) :=
  match result with | .error error => some error.reason | .ok _ => none
#guard (selectField 6 .bytes .singular .available >>= (·.scalar source)).toOption == some (.bytes [0, 255])
#guard (selectField 5 .text .singular .optional >>= (·.present source)).toOption.map (·.datum) ==
  some (some (literal (.boolean true)))
#guard reason (selectField 5 .text .singular .available) == some "availability mismatch"
#guard reason (selectField 6 .text .singular .available) == some "field type mismatch"
#guard reason (selectField 3 .text .singular (.oneof "choice") >>= (·.select "other" source)) ==
  some "oneof group mismatch"
#guard (selectField 3 .text .singular (.oneof "choice") >>= (·.select "choice" source) >>=
  (·.scalar source)).toOption == some (.text "")
#guard reason (selectField 4 (.integer .int64) .singular (.oneof "choice") >>= (·.select "choice" source)) ==
  some "oneof member is not selected"
#guard ((do
  let nested ← selectField 7 (.message "fixture.messaging.public.v1.Message.Nested") .singular .optional >>=
    (·.establish source)
  let ref ← Fixture.API.fieldReference unary .request "fixture.messaging.public.v1.Message.Nested" 1 source
  let state ← nested.field ref source
  let state ← state.refine (.enumeration "fixture.messaging.public.v1.Message.Nested.State") .singular .available source
  state.scalar source) : Except Field.Error Scalar).toOption ==
  some (.enumeration "fixture.messaging.public.v1.Message.Nested.State" 99)
#guard ((do
  let attrs ← selectField 2 (.message "fixture.messaging.shared.v1.Shared") (.map .text) .available
  let selected ← attrs.lookup (.text "z") source >>= (·.establish source)
  let ref ← Fixture.API.fieldReference unary .request "fixture.messaging.shared.v1.Shared" 1 source
  let id ← selected.field ref source
  let id ← id.refine .text .singular .available source
  id.scalar source) : Except Field.Error Scalar).toOption == some (.text "last")
#guard (selectField 2 (.message "fixture.messaging.shared.v1.Shared") (.map .text) .available >>=
  (·.lookup (.text "absent") source) >>= (·.present source)).toOption.map (·.datum) ==
  some (some (literal (.boolean false)))
private def samples := do
  let legacy ← selectField 8 (.message "fixture.protobuf.compat.v1.LegacyOptions") .singular .optional >>=
    (·.establish source)
  let ref ← Fixture.API.fieldReference unary .request "fixture.protobuf.compat.v1.LegacyOptions" 3 source
  let values ← legacy.field ref source
  values.refine (.integer .int32) .repeated .available source
#guard (samples >>= (·.length source)).toOption == some 2
#guard (samples >>= (·.index 0 source) >>= (·.scalar source)).toOption == some (.integer .int32 (-1))
#guard (samples >>= (·.index 1 source) >>= (·.scalar source)).toOption == some (.integer .int32 7)
#guard reason (samples >>= (·.index 2 source)) == some "repeated index out of range"
#guard reason (Fixture.API.fieldReference unary .request "fixture.messaging.public.v1.Reply" 1 source) ==
  some "unknown containing schema or field"
#guard reason (Fixture.API.fieldReference unary .request "fixture.messaging.public.v1.Message" 99 source) ==
  some "unknown containing schema or field"

private def alien : RpcOwner where
  Witness _ _ := Unit
  schema _ := unaryReference.schema
/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (value : Checked Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .request limits) :
    Field.Cursor alien (Request := Message) (Response := Reply) () .request limits
      (.message "fixture.messaging.public.v1.Message") .singular .available := Field.root value

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (value : Checked Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .request limits)
    (ref : Field.Reference Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .response "fixture.messaging.public.v1.Message") :=
  (Field.root value).field ref source

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (value : Checked Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .request limits)
    (ref : Field.Reference Fixture.API.rpcOwner ⟨unary, unaryReference⟩ .request "fixture.messaging.shared.v1.Shared") :=
  (Field.root value).field ref source

example (value : Checked Fixture.API.rpcOwner ⟨unary, unaryReference⟩ side bounds) :
    Field.Denotes value.value (Field.root value).path (Field.root value).datum :=
  (Field.root value).correspondence
end FieldAccess

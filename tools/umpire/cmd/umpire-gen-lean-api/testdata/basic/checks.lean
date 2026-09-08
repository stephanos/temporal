import Fixture.API

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

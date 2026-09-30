import Umpire.Operation.Canonical

/-! Structural admission and identity controls independent of any generated product API or Target
semantics. -/

namespace Umpire.Operation.Tests

private def schema : RpcSchema := {
  fullName := "example.Service.Send"
  request := ⟨"example.Request", [⟨"example.Request", "proto3", "field-1-string", "", [], none⟩]⟩
  response := ⟨"example.Response", [⟨"example.Response", "proto3", "field-1-bool", "", [], none⟩]⟩
  schemaInputs := []
  clientStreaming := false
  serverStreaming := false
}

private def owner (expected : RpcSchema) : RpcOwner where
  Witness _ _ := Unit
  schema _ := expected

private def check (expected candidate : RpcSchema) :=
  checkRpc (owner expected) (Request := Unit) (Response := Unit) () candidate

private def error? (result : Except Error α) : Option Error :=
  match result with
  | .ok _ => none
  | .error error => some error

#guard (check schema schema).toOption.isSome
#guard error? (check schema { schema with fullName := "example.Service.Other" }) ==
  some (.wrongMethod "example.Service.Send" "example.Service.Other")
#guard error? (check schema
  { schema with request := { schema.request with nodes := [] } }) ==
  some (.incompatibleRequest "example.Service.Send")
#guard error? (check schema { schema with response := schema.request }) ==
  some (.incompatibleResponse "example.Service.Send")
#guard error? (check schema { schema with serverStreaming := true }) ==
  some (.incompatibleStreaming "example.Service.Send")
#guard error? (check { schema with clientStreaming := true }
  { schema with clientStreaming := true }) ==
  some (.unsupportedStreaming "example.Service.Send")
#guard (sdkCommand String Nat String (.of "example.send")).toOption.map (·.identity.value) ==
  some "example.send"
#guard (event String (.of "example.sent")).toOption.map (·.identity.value) == some "example.sent"
#guard error? (event String (.of "bad")) == some (.invalidIdentity (.of "bad"))

private def wideSchema : RpcSchema := { schema with
  request := { schema.request with nodes := (List.range 200).map fun n =>
    ⟨"example.Request", "proto3", "field-" ++ toString n, "context", ["example.Other"], none⟩ }
  schemaInputs := schema.response.nodes }

private def identityKey (s : RpcSchema) : String := Canonical.key (Canonical.rpcSchema s)

-- Identity separates the method, both payload signature roots, and the interaction shape.
#guard identityKey schema != identityKey { schema with fullName := "example.Service.Other" }
#guard identityKey schema !=
  identityKey { schema with request := { schema.request with root := "example.Other" } }
#guard identityKey schema !=
  identityKey { schema with response := { schema.response with root := "example.Other" } }
#guard identityKey schema != identityKey { schema with clientStreaming := true }
#guard identityKey schema != identityKey { schema with serverStreaming := true }
-- Every descriptor-closure component still reaches identity, through the closure digest.
#guard identityKey schema != identityKey wideSchema
#guard identityKey schema != identityKey { schema with
  request := { schema.request with
    nodes := schema.request.nodes.map fun n => { n with descriptor := "field-1-bytes" } } }
#guard identityKey schema != identityKey { schema with
  request := { schema.request with
    nodes := schema.request.nodes.map fun n => { n with fileContext := "other-file" } } }
#guard identityKey schema != identityKey { schema with
  request := { schema.request with
    nodes := schema.request.nodes.map fun n => { n with references := ["example.Other"] } } }
#guard identityKey schema != identityKey { schema with
  request := { schema.request with nodes := schema.request.nodes.map fun n => { n with
    valueShape := some (.message [⟨1, "payload", .text, .singular, .optional, none⟩] none) } } }
#guard identityKey schema != identityKey { schema with schemaInputs := schema.response.nodes }
-- Distinct closures receive distinct identities across a family of single-edit descriptors: the
-- fold's separation is exercised rather than assumed.
#guard (((List.range 512).map fun n => identityKey { schema with
  request := { schema.request with
    nodes := [⟨"example.Request", "proto3", "field-" ++ toString n, "", [], none⟩] } }).eraseDups).length
  == 512
-- The closure reaches identity only through that fixed-width digest, so the key stays bounded
-- however large the closure grows.
#guard (identityKey wideSchema).length < 4096
#guard (identityKey schema).length < 4096

example {owner : RpcOwner} {Request Response : Type}
    {reference : owner.Witness Request Response} (binding : CheckedRpc owner reference) :
    binding.schema.fullName = (owner.schema reference).fullName := by rw [binding.schema_eq]

example (declaration : Declaration .unaryRpc Request Response Failure) : False := by
  obtain h | h := declaration.modelOwned <;> cases h

end Umpire.Operation.Tests

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
#check (Umpire.Operation.sdkCommand String Unit Empty (.of "example.send") :
  Except Umpire.Operation.Error (Umpire.Operation.Declaration .event String Unit Empty))

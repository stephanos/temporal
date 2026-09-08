import Umpire.Core
import Umpire.Operation.ValueSchema

/-!
Typed structural operation declarations. RPC admission compares a candidate with the schema selected
by its generated witness; it does not infer provenance from phantom types. SDK commands and events
have separate kinds. These declarations carry signatures only: Target owns all result alternatives.
Descriptor encodings below are exact structural metadata, never concrete request or response values.
-/

namespace Umpire.Operation

/-- One normalized descriptor and its explicit references, independent of Lean spelling.
`descriptor` and `fileContext` contain exact hex-encoded Protobuf descriptor bytes, not value hashes. -/
structure SchemaNode where
  name : String
  protoSyntax : String
  descriptor : String
  fileContext : String
  references : List String
  valueShape : Option ValueShape := none
  deriving BEq, DecidableEq, Repr

/-- A complete, sorted, visited-set descriptor closure rooted at one message. -/
structure Schema where
  root : String
  nodes : List SchemaNode
  deriving BEq, DecidableEq, Repr

/-- Full method identity and both structural signatures, including unsupported streaming shapes. -/
structure RpcSchema where
  fullName : String
  request : Schema
  response : Schema
  schemaInputs : List SchemaNode
  clientStreaming : Bool
  serverStreaming : Bool
  deriving BEq, DecidableEq, Repr

/-- Admission failures identify the method or authored declaration that owns the bad binding. -/
inductive Error where
  | wrongMethod (expected actual : String)
  | incompatibleSchema (method : String)
  | incompatibleRequest (method : String)
  | incompatibleResponse (method : String)
  | incompatibleStreaming (method : String)
  | unsupportedStreaming (method : String)
  | invalidIdentity (identity : DefinitionId)
  deriving BEq, DecidableEq, Repr

/-- Structural authority indexed by payload types. Consumers select this owner explicitly;
matching schema text never converts a witness from a different owner. -/
structure RpcOwner where
  Witness : Type → Type → Type
  schema : {Request Response : Type} → Witness Request Response → RpcSchema

/-- A unary binding retains kernel-checked equality to its expected generated schema. -/
structure CheckedRpc (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) where
  private mk ::
  schema : RpcSchema
  agrees : schema = owner.schema reference
  unary : schema.clientStreaming = false ∧ schema.serverStreaming = false

/-- Compare against a generator-selected schema; callers must obtain that selection from its owner. -/
def checkRpc (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (candidate : RpcSchema) :
    Except Error (CheckedRpc owner reference) := do
  let expected := owner.schema reference
  if candidate.fullName != expected.fullName then
    throw (.wrongMethod expected.fullName candidate.fullName)
  if candidate.schemaInputs != expected.schemaInputs then throw (.incompatibleSchema expected.fullName)
  if candidate.request != expected.request then throw (.incompatibleRequest expected.fullName)
  if candidate.response != expected.response then throw (.incompatibleResponse expected.fullName)
  if candidate.clientStreaming != expected.clientStreaming ||
      candidate.serverStreaming != expected.serverStreaming then
    throw (.incompatibleStreaming expected.fullName)
  if h : candidate = expected then
    if unary : candidate.clientStreaming = false ∧ candidate.serverStreaming = false then
      pure ⟨candidate, h, unary⟩
    else throw (.unsupportedStreaming expected.fullName)
  else throw (.wrongMethod expected.fullName candidate.fullName)

/-- Admission cannot substitute another method or schema, including when phantom types coincide. -/
theorem CheckedRpc.schema_eq (binding : CheckedRpc owner reference) :
    binding.schema = owner.schema reference := binding.agrees

/-- Interaction kinds stay distinct even when their payload signatures happen to coincide. -/
inductive Kind where
  | unaryRpc
  | sdkCommand
  | event
  deriving BEq, DecidableEq, Repr

/-- Typed connectivity without any state transition, response selection, or execution authority. -/
structure Declaration (kind : Kind) (Request Response Failure : Type) where
  private mk ::
  identity : DefinitionId
  modelOwned : kind = .sdkCommand ∨ kind = .event

/-- RPC connectivity retains an owner's witness at the exact request and response types.
A consumer must keep its selected owner in this type through schema and payload lowering. -/
structure RpcDeclaration (owner : RpcOwner) (Request Response Failure : Type) where
  private mk ::
  reference : owner.Witness Request Response
  binding : CheckedRpc owner reference

/-- The declaration's schema is still checked against its retained typed witness. -/
def RpcDeclaration.schema (declaration : RpcDeclaration owner Request Response Failure) : RpcSchema :=
  declaration.binding.schema

/-- Stable RPC identity comes from the admitted schema, never an independently supplied name. -/
def RpcDeclaration.identity (declaration : RpcDeclaration owner Request Response Failure) :
    DefinitionId := DefinitionId.of declaration.schema.fullName

/-- Declaration conversion cannot erase or replace the owner's payload-indexed schema authority. -/
theorem RpcDeclaration.schema_eq (declaration : RpcDeclaration owner Request Response Failure) :
    declaration.schema = owner.schema declaration.reference := declaration.binding.schema_eq

/-- Declare RPC connectivity from an admitted binding. Failure alternatives remain model-owned. -/
def rpc {owner : RpcOwner} {Request Response : Type}
    {reference : owner.Witness Request Response} (Failure : Type)
    (binding : CheckedRpc owner reference) :
    RpcDeclaration owner Request Response Failure :=
  ⟨reference, binding⟩

/-- Declare an SDK command's submission/return/error signature, separately from RPCs and events. -/
def sdkCommand (Request Response Failure : Type) (identity : DefinitionId) :
    Except Error (Declaration .sdkCommand Request Response Failure) :=
  match identity.validate with
  | .ok () => .ok ⟨identity, Or.inl rfl⟩
  | .error _ => .error (.invalidIdentity identity)

/-- Declare semantic event data; events have no service response or transport failure signature. -/
def event (Payload : Type) (identity : DefinitionId) :
    Except Error (Declaration .event Payload Unit Empty) :=
  match identity.validate with
  | .ok () => .ok ⟨identity, Or.inr rfl⟩
  | .error _ => .error (.invalidIdentity identity)

end Umpire.Operation

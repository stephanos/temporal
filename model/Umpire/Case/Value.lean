import Umpire.Core

/-!
Typed values, protobuf paths, immutable Slots, and declared Observations for the Umpire Case IR.

Cardinality is part of `ValueType`, and the expression constructors form the closed vocabulary used
by Program guards and Contract predicates. Opaque capabilities have a type but no value constructor.
-/

namespace Umpire.Case

private instance : Repr ByteArray where
  reprPrec bytes _ := repr bytes.data

/-- An incompatible Case vocabulary version and its additive revision. -/
structure FormatVersion where
  major : Nat
  minor : Nat := 0
  deriving BEq, DecidableEq, Repr

/-- The protobuf scalar kinds accepted by typed request and projection paths. -/
inductive ScalarKind where
  | text
  | natural
  | boolean
  | bytes
  | int32
  | int64
  | uint32
  | uint64
  | sint32
  | sint64
  | fixed32
  | fixed64
  | sfixed32
  | sfixed64
  | float
  | double
  deriving BEq, DecidableEq, Repr

/-- The closed categories of source-model incompleteness retained in a Case. -/
inductive CaseKnownGapKind where
  | capabilityContract
  | input
  | interpretation
  | claim
  deriving BEq, DecidableEq, Repr

/-- One typed protobuf message value retained without interpreting its fields. -/
structure MessageValue where
  typeUrl : String
  bytes : ByteArray
  deriving BEq, Repr

/-- A closed runtime value. Collection order is semantic and map entries retain source order. -/
inductive Value where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  | bytes (value : ByteArray)
  | signedInteger (value : Int)
  | unsignedInteger (value : Nat)
  | floatingPoint (value : Float)
  | enumValue (number : Int)
  | messageValue (value : MessageValue)
  | listValue (values : List Value)
  | mapValue (entries : List (Value × Value))
  deriving BEq, Repr

/-- A singular scalar, named enum/message, whole `Any`, or private opaque capability type. -/
inductive SingularType where
  | scalar (kind : ScalarKind)
  | enumeration (protobufType : String)
  | message (protobufType : String)
  | any
  | opaqueCapability
  deriving BEq, DecidableEq, Repr

/-- A descriptor type whose singular, repeated, or map cardinality is explicit. -/
inductive ValueType where
  | singular (type : SingularType)
  | repeated (element : SingularType)
  | map (key : ScalarKind) (value : SingularType)
  deriving BEq, DecidableEq, Repr

/-- The closed selectors available at one protobuf field-path segment. -/
inductive FieldSelector where
  | repeated
  | mapKey (key : Value)
  | presence
  | oneof (selectedField : String)
  deriving BEq, Repr

/-- One named protobuf field and its optional traversal selector. -/
structure FieldPathSegment where
  field : String
  selector : Option FieldSelector := none
  deriving BEq, Repr

/-- A descriptor-checked protobuf payload path. -/
structure FieldPath where
  segments : List FieldPathSegment
  deriving BEq, Repr

/-- A stable reference to one instruction in one entrypoint. -/
structure InstructionReference where
  entrypointId : String
  instructionId : String
  deriving BEq, DecidableEq, Repr

/-- A reference to one immutable Slot. -/
structure SlotReference where
  slotId : String
  deriving BEq, DecidableEq, Repr

/-- The typed fields exposed by every instruction outcome. -/
inductive InstructionOutcomeField where
  | status
  | protocolCode
  | sdkFailureCode
  | detail
  | value
  deriving BEq, DecidableEq, Repr

/-- The declared type of one instruction-outcome field. -/
structure OutcomeFieldSchema where
  field : InstructionOutcomeField
  type : ValueType
  deriving BEq, Repr

/-- The complete fields a later guard may read from one instruction outcome. -/
structure InstructionOutcomeSchema where
  fields : List OutcomeFieldSchema
  deriving BEq, Repr

/-- A typed reference to one field of an instruction outcome. -/
structure InstructionOutcomeReference where
  instruction : InstructionReference
  field : InstructionOutcomeField
  deriving BEq, DecidableEq, Repr

/-- A reference to one declared Observation on the current Run Event. -/
structure ObservationReference where
  observationId : String
  deriving BEq, DecidableEq, Repr

/-- A rule-local reference to one declared single-assignment Contract capture. -/
structure CaptureReference where
  captureId : String
  deriving BEq, DecidableEq, Repr

/-- The generic Run Event metadata available to Contract predicates. -/
inductive RunEventField where
  | sequence
  | elapsedMilliseconds
  | kind
  | entrypointId
  | activationId
  | instructionId
  | attempt
  | sourceId
  deriving BEq, DecidableEq, Repr

/-- Closed typed expressions shared by Program guards and Contract transition predicates. -/
inductive ValueExpression where
  | literal (value : Value)
  | slot (reference : SlotReference)
  | outcome (reference : InstructionOutcomeReference)
  | observation (reference : ObservationReference)
  | capture (reference : CaptureReference)
  | runEvent (field : RunEventField)
  | path (source : ValueExpression) (path : FieldPath)
  | present (operand : ValueExpression)
  | equals (left right : ValueExpression)
  | lessThan (left right : ValueExpression)
  | lessThanOrEqual (left right : ValueExpression)
  | greaterThan (left right : ValueExpression)
  | greaterThanOrEqual (left right : ValueExpression)
  | negation (operand : ValueExpression)
  | all (operands : List ValueExpression)
  | any (operands : List ValueExpression)
  deriving BEq, Repr

/-- Whether a Slot carries an ordinary value or private Host authority. -/
inductive SlotKind where
  | value
  | opaqueCapability
  deriving BEq, DecidableEq, Repr

/-- One immutable, single-assignment Program Slot. -/
structure SlotSchema where
  slotId : String
  type : ValueType
  kind : SlotKind := .value
  deriving BEq, Repr

/-- One typed Run Event field visible to Contracts. -/
structure ObservationSchema where
  observationId : String
  type : ValueType
  deriving BEq, Repr

end Umpire.Case

namespace Temporal.Proto

structure Bytes where
  digest : String
  size : Nat
  deriving DecidableEq, Repr

structure MessageRef where
  descriptor : String
  remainingDepth : Nat
  deriving DecidableEq, Repr

structure FieldDescriptor where
  fullName : String
  jsonName : String
  number : Int
  kind : String
  typeName : String
  mapKeyType : String
  mapValueType : String
  presence : Bool
  oneofName : String
  repeated : Bool
  mapField : Bool
  recursive : Bool
  deprecated : Bool
  deriving DecidableEq, Repr

structure MessageDescriptor where
  fullName : String
  fields : List FieldDescriptor
  deriving DecidableEq, Repr

structure EnumDescriptor where
  fullName : String
  values : List (String × Int)
  allowAliases : Bool
  deriving DecidableEq, Repr

structure FileDescriptor where
  path : String
  packageName : String
  syntaxName : String
  dependencies : List String
  deriving DecidableEq, Repr

structure Method (Request Response : Type) where
  fullName : String
  clientStreaming : Bool
  serverStreaming : Bool
  deprecated : Bool
  deriving DecidableEq, Repr

structure MethodDescriptor where
  fullName : String
  inputType : String
  outputType : String
  clientStreaming : Bool
  serverStreaming : Bool
  deprecated : Bool
  deriving DecidableEq, Repr

structure ServiceDescriptor where
  fullName : String
  methods : List MethodDescriptor
  deriving DecidableEq, Repr

end Temporal.Proto

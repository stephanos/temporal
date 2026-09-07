module

public import Protobuf
public import Protobuf.Json
meta import Protobuf.Elab
meta import Protobuf.Notation
meta import Protobuf.Notation.Enum
meta import Protobuf.Notation.Message
meta import Protobuf.Notation.Mutual

public section

/-!
Generated Lean declarations for the exact Testpilot Case protobuf import closure.

The schema is loaded from its checked-in source during elaboration. The package-level `PROTOC`
override selects the repository-supported compiler.
-/

open Protobuf
open scoped Protobuf.Notation

#load_proto_file "../proto/internal/temporal/server/api/testpilot/v1/case.proto" in "../proto/internal"

module

public import Protobuf
public import Protobuf.Json
public import Testpilot.Carried
meta import Testpilot.Carried
meta import Protobuf.Elab
meta import Protobuf.Notation
meta import Protobuf.Notation.Enum
meta import Protobuf.Notation.Message
meta import Protobuf.Notation.Mutual

public section

/-!
Generated Lean declarations for the exact Testpilot Case and Run protobuf import closures.

The schema is loaded from its checked-in source during elaboration. The package-level `PROTOC`
override selects the repository-supported compiler.

The Case closure excludes the Run, so `Testpilot.Authoring` needs `run.proto` loaded beside
`case.proto`. `#load_proto_file` re-declares every file of the closure it loads, so two loads that
share `value.proto` would declare it twice; one `protoc` call over both roots yields each file once,
and the library's own compiler turns that set into declarations.

The typed worker instructions carry public API messages, so `instruction.proto` imports their files
and `protoc` resolves those imports from `proto/api.binpb`. `Testpilot.Carried` compiles that part of
the closure once; the call below skips its files, and rejects an API import it does not name.
-/

open Protobuf
open scoped Protobuf.Notation

run_cmd do
  let protoc := (← IO.getEnv "PROTOC").getD "protoc"
  let encoded ← IO.FS.withTempFile fun _ descriptorSet => do
    discard <| IO.Process.run {
      cmd := protoc
      args := #[
        "--proto_path=../proto/internal",
        "--descriptor_set_in=../proto/api.binpb",
        "--include_imports",
        "--retain_options",
        s!"--descriptor_set_out={descriptorSet}",
        "../proto/internal/temporal/server/api/testpilot/v1/case.proto",
        "../proto/internal/temporal/server/api/testpilot/v1/run.proto"
      ]
    }
    IO.FS.readBinFile descriptorSet
  let wire ← Lean.ofExcept <| (Binary.Get.run (Binary.getThe Encoding.Message) encoded).toExcept
  let descriptors ← Lean.ofExcept <|
    google.protobuf.FileDescriptorSet.«protobuf.internal».fromMessage wire
  let precompiled := #["google/protobuf/descriptor.proto"] ++ Testpilot.Carried.files.toArray
  for file in descriptors.file do
    let name := file.name.getD ""
    unless name.startsWith "temporal/server/api/testpilot/" || name == "google/protobuf/any.proto"
        || precompiled.contains name do
      throwError "the protocol imports '{name}', which Testpilot.Carried.files does not name; add it \
there so it is compiled once"
  let commands ← Lean.ofExcept <|
    (Protobuf.Versions.compile_proto descriptors precompiled).run
  commands.forM Lean.Elab.Command.elabCommand

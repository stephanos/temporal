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
Generated Lean declarations for the public Temporal API messages the Testpilot protocol carries.

The typed worker instructions carry `temporal.api.command.v1.Command`,
`temporal.api.nexus.v1.StartOperationResponse`, `temporal.api.nexus.v1.HandlerError`,
`temporal.api.common.v1.Payload` and `temporal.api.failure.v1.Failure`, so `instruction.proto`
imports their files and the Case closure grows by those files' own import closure. That closure is
compiled here, once, from the checked-in `proto/api.binpb`, and `Testpilot.Protocol` compiles the
protocol's own files against it: the public API changes far less often than the protocol, and
keeping the two apart keeps a protocol edit's rebuild at the protocol's own cost. `files` is the
closure by name; `Testpilot.Protocol` rejects an import outside it, so a new API import is added
here rather than compiled twice.
-/

open Protobuf
open scoped Protobuf.Notation

namespace Testpilot.Carried

/-- The public API files the protocol's imports reach, transitively: every file `protoc` includes
for `instruction.proto` that is not the protocol's own and not `google/protobuf/any.proto`, which
`value.proto` imported before the typed instructions existed. -/
def files : List String := [
  "google/protobuf/duration.proto",
  "google/protobuf/empty.proto",
  "google/protobuf/field_mask.proto",
  "google/protobuf/timestamp.proto",
  "google/protobuf/wrappers.proto",
  "temporal/api/activity/v1/message.proto",
  "temporal/api/callback/v1/message.proto",
  "temporal/api/command/v1/message.proto",
  "temporal/api/common/v1/message.proto",
  "temporal/api/compute/v1/config.proto",
  "temporal/api/compute/v1/provider.proto",
  "temporal/api/compute/v1/scaler.proto",
  "temporal/api/deployment/v1/message.proto",
  "temporal/api/enums/v1/activity.proto",
  "temporal/api/enums/v1/command_type.proto",
  "temporal/api/enums/v1/common.proto",
  "temporal/api/enums/v1/deployment.proto",
  "temporal/api/enums/v1/event_type.proto",
  "temporal/api/enums/v1/nexus.proto",
  "temporal/api/enums/v1/reset.proto",
  "temporal/api/enums/v1/task_queue.proto",
  "temporal/api/enums/v1/workflow.proto",
  "temporal/api/failure/v1/message.proto",
  "temporal/api/nexus/v1/message.proto",
  "temporal/api/sdk/v1/event_group_marker.proto",
  "temporal/api/sdk/v1/user_metadata.proto",
  "temporal/api/taskqueue/v1/message.proto",
  "temporal/api/workflow/v1/message.proto"]

/-- The descriptors with every deprecation option cleared. The library declares an attribute per
deprecated declaration, and names a deprecated enum value by an unqualified name it cannot resolve;
nothing here reads a deprecation, so the options are dropped before compilation rather than
compiled into a failing attribute. -/
partial def withoutDeprecation (file : google.protobuf.FileDescriptorProto) :
    google.protobuf.FileDescriptorProto :=
  { file with
    message_type := file.message_type.map clearMessage
    enum_type := file.enum_type.map clearEnum }
where
  clearEnum (declared : google.protobuf.EnumDescriptorProto) : google.protobuf.EnumDescriptorProto :=
    { declared with
      options := declared.options.map fun options => { options with deprecated := none }
      value := declared.value.map fun value =>
        { value with options := value.options.map fun options => { options with deprecated := none } } }
  clearMessage (declared : google.protobuf.DescriptorProto) : google.protobuf.DescriptorProto :=
    { declared with
      options := declared.options.map fun options => { options with deprecated := none }
      field := declared.field.map fun field =>
        { field with options := field.options.map fun options => { options with deprecated := none } }
      nested_type := declared.nested_type.map clearMessage
      enum_type := declared.enum_type.map clearEnum }

end Testpilot.Carried

run_cmd do
  let encoded ← IO.FS.readBinFile "../proto/api.binpb"
  let wire ← Lean.ofExcept <| (Binary.Get.run (Binary.getThe Encoding.Message) encoded).toExcept
  let descriptors ← Lean.ofExcept <|
    google.protobuf.FileDescriptorSet.«protobuf.internal».fromMessage wire
  let carried := (descriptors.file.filter fun file =>
    Testpilot.Carried.files.contains (file.name.getD "")).map Testpilot.Carried.withoutDeprecation
  for name in Testpilot.Carried.files do
    unless carried.any (·.name == some name) do
      throwError "proto/api.binpb carries no '{name}'; Testpilot.Carried.files names the API files the protocol imports"
  let commands ← Lean.ofExcept <|
    (Protobuf.Versions.compile_proto { descriptors with file := carried }
      #["google/protobuf/descriptor.proto"]).run
  commands.forM Lean.Elab.Command.elabCommand

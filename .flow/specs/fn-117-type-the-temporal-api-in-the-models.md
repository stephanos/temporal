# Type the Temporal API in the Models

> HTML render lens: `.flow/artifacts/fn-117-type-the-temporal-api-in-the-models/spec.html` — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

A Model says which Temporal API messages its actions carry and how a run calls the server, reads its answers and recognises evidence. Today it says all of that in string literals. A review on 2026-10-01 counted them in the Models:

| What | Example | Count |
| --- | --- | --- |
| Full type and method names | `"temporal.api.workflowservice.v1.StartActivityExecutionRequest"` | 28, of which 23 distinct |
| Field paths in assignments | `Assignment("task_queue.name", …)` | 25 |
| Fields of written-out messages | `ProtoField("non_retryable", Flag(true))` | 27 |
| Read and evidence paths | `Path(Projected, "delivery_admission.decision")` | 17 |
| Enum values | `EnumName("ACTIVITY_EXECUTION_STATUS_" + value)` | 6 |

A misspelled field path compiles. Go catches it later, when it lowers a Case, and reports the Scala line. A misspelled action `schema` is caught by nothing, because no Go code reads that field of the IR.

The owner wants full typed support: a Model names a message, a field, a method and an enum value as typed values, the compiler rejects one that does not exist or has the wrong type, and an editor completes them. This spec delivers that for the feature developer who writes realizations, and for the reviewer who should not have to check spellings against proto files.

## Architecture & Data Models
<!-- scope: technical -->

**ScalaPB classes for the Temporal API are generated and put on the Models' compile classpath.** The gate generates them from the same descriptors Go links (`temporal.api`, the Testpilot IR and the well-known types a realization writes, such as `Duration`), packages them once as a jar with a stamp, and regenerates only when the descriptors change. fn-113 Part B already brings ScalaPB into the lifter; this spec reuses that toolchain.

**The DSL takes those types where it took strings.**

| Today | After |
| --- | --- |
| `.schema("temporal.api.…StartActivityExecutionRequest")` | `.schema[StartActivityExecutionRequest]` |
| `Rpc(role, "/…WorkflowService/StartActivityExecution", assignments)` | the method as a value, with assignments typed by its request message and reads typed by its response |
| `Assignment("task_queue.name", Environment(b))` | `_.taskQueue.name := Environment(b)` |
| `Path(Projected, "delivery_admission.decision")` | a field selection on the message the projection holds |
| `EnumName("ACTIVITY_EXECUTION_STATUS_PAUSED")` | the enum value itself |
| `Proto("temporal.api.failure.v1.Failure", ProtoField("message", Text(…)), …)` | the message written with its typed fields |

**The lifter turns the typed forms back into the names the IR carries.** The IR schema does not change. It keeps message names, method names and field paths as text, which is what Go validates and lowers. The lifter reads a field selection from the typed tree and writes the proto field path, reads an enum value and writes its proto name, and reads a message type and writes its full name.

**A Model still holds no real message.** An assignment gives a field a symbolic operand (an environment binding, the run id, a learned value), which no message value can hold. The Models use the generated types to name and type things, and nothing in Scala builds or sends a Temporal message.

## API Contracts
<!-- scope: technical -->

### Proven compiler surface (available now)

`io.temporal.api.workflowservice.v1.StartActivityExecutionRequest` and `StartActivityExecutionResponse` are generated message types. `WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION` has static type `io.grpc.MethodDescriptor[StartActivityExecutionRequest, StartActivityExecutionResponse]` and unary runtime kind; its proto name is `/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution`. `StartActivityExecutionRequest.getTaskQueue.name` compiles and describes `task_queue.name`; `request.taskQueue.name` does not compile because `taskQueue` is `Option[TaskQueue]`. `StartActivityExecutionResponse.runId` describes `run_id`.

`ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED` is a generated enum value with proto full name `temporal.api.enums.v1.ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED`. `History.events.map(_.eventId)` describes `events[*].event_id`. `HistoryEvent.attributes.nexusOperationScheduledEventAttributes` is an `Option[NexusOperationScheduledEventAttributes]` and describes `attributes<nexus_operation_scheduled_event_attributes>`. `InstructionOutcome.deliveryAdmission` is an `Option[DeliveryAdmission]`. `Payload.metadata` has exact Scala type `Map[String, com.google.protobuf.ByteString]`. These expressions compiled in task 1; TASTy inspection recovered each proto name from descriptors. The linked API jar now provides these classes, including the service, on Model and lifter classpaths.

### Selected symbolic DSL surface (to implement in tasks 3–5)

Keep Action as the declaration point. `.schema[Message]` accepts only `scalapb.GeneratedMessage` subtypes and appends one schema in call order, so `.schema[StartOperationResponse].schema[HandlerError]` preserves the Nexus two-schema order. Its lifter obtains each full proto name from the generated companion descriptor, not the Scala class name. This overload coexists with the current string form until task 8.

Use `Instruction.rpc(role, WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION)(assignments, reads)` as the typed unary constructor. Its method argument fixes `Req=StartActivityExecutionRequest` and `Rsp=StartActivityExecutionResponse`; assignment selectors have root `Req`, response-read selectors have root `Rsp`. The method full name comes from the generated service descriptor and must agree with the gRPC constant and unary request/response type arguments. Retain `Instruction.Rpc(role, String, …)` only as the migration bridge. Do not make the method name a free string in the typed constructor.

Use `Recorded.read(method, responseSelection)` and `Recorded.single(method, responseSelection)` to retain `Req`, `Rsp`, and the selected projection type. An `Evidence.read(id, …, recorded)` declaration returns `EvidenceRef[Req, Projected]` while lowering the existing evidence ID. `Instruction.poll(evidenceRef, role)(assignments, until)` derives assignment root `Req` and condition root `Projected` from that reference. The author must not pass an unrelated poll method, request root, or projection root. Repeated `read` projections retain their element type, while `single` uses its selected message type. Direct selection of a repeated message field preserves its bare terminal spelling; `.map(item => item)` preserves explicit terminal `[*]`. Both are existing Go read spellings and keep the same typed projection root; the lifter must not append a wildcard that the selector did not declare. Model-owned evidence and role IDs remain strings and keep their existing emitted values.

Use one generic field selector carrier `Field[Root, Value]` for assignments, response reads, evidence fields, operation keys, guards and projected conditions. Field lambdas start at the declared root, with accessors matching generated ScalaPB spellings; nested optional messages unwrap via generated getters or a typed option traversal, repeated fields via `.map`, and oneof arms via generated oneof selectors. The lifter reads typed trees and descriptors to emit current plain, `[*]` and `<member>` IR path grammar. `Environment[T]`, `LearnedValue[T]`, `Literal[T]`, `Run` (String), equality, comparisons and enum operands retain value type; `Assignment[Root, T]` requires the field and operand's `T` to agree. These wrappers remain symbolic declarations and never instantiate/send a protobuf message.

Use `Proto[Message](typedFields…)` for symbolic constant messages; typed fields accept ScalaPB enum values and nested/oneof/repeated values of their declared type. For `Payload.metadata`, a symbolic map entry has `String` key and `com.google.protobuf.ByteString` value; existing UTF-8 encoding remains the emitted `Utf8` value. Map-entry reads stay unsupported. The only explicit message-root escape is for payloads whose origin is dynamic: `Operand.Projected.as[InstructionOutcome]` in RunEvent key/guard contexts (and an explicitly named HistoryEvent root where a history attribute is dynamically supplied). This does not erase known RPC/read/poll roots, and Go's existing path/type validation remains in force.

This section selects names and type relationships for downstream implementation; the constructors are not in the repository yet. Task 3 must compile its first positive/negative fixtures against these names before task 4 fills out all selectors and task 5 fills out constants. No service transport, client call or API behavior hints are part of this contract.

**Lifted meaning is frozen.** Every IR file is equal before and after this spec except for source positions. The typed forms lift to the same names and paths the strings spelled. Generated positive-fixture JSON and Query-manifest position metadata may move with those same source edits; compare exact pre-edit bytes and permit only structured `position` fields and manifest `Position` strings to change. All other generated data, Case JSON bytes and frozen golden inputs remain unchanged. Position metadata must retain the same source file and refer to the corresponding moved declaration.

## Edge Cases & Constraints
<!-- scope: technical -->

- **ScalaPB must fit.** It has to generate and compile the Temporal API for Scala 3.9.0. fn-113 R4 checks that for the small IR schema; the Temporal API is hundreds of messages and has to be checked on its own. R1 does that before anything else.
- **Build time.** The owner prefers ScalaPB classes as long as compile time with a warm cache does not explode. The generated classes are compiled once into a jar and reused until the descriptors change, so a Model edit does not recompile them. R1 measures the warm-cache compile of the Models against today's, and more than twice today's counts as exploding.
- **Shapes beyond a plain field.** Realizations read every element of a repeated field (`history.events[*]`) and one arm of a oneof (`attributes<nexus_operation_scheduled_event_attributes>`). Each has a typed form, and the lifter writes the same path text the IR has today. The existing map entries are constant-message writes (`Payload.metadata`), whose keys and values become typed too. Go rejects map-entry read paths today; this spec introduces none.
- **Typed operands.** A literal or an enum value of the wrong type for a field does not compile. A symbolic operand carries the type it was declared with. Where the type of a value is not known statically, such as the payload a Run Event's guard reads, the path is written against a named message type.
- **Testpilot's own messages.** Evidence paths such as `delivery_admission.decision` are fields of the Testpilot IR's `InstructionOutcome`. They are typed the same way, from the Testpilot proto.
- **Descriptors.** The classes are generated from the descriptor sets the repository already builds (`proto/api.binpb` and the internal protos). A missing or stale descriptor set fails the gate with the make target that builds it named.
- **Go keeps validating.** The checks at lowering stay as they are. They guard an IR file that was edited by hand or lifted by an older lifter.
- **Libraries in the Models.** Keeping the Models free of libraries is guidance, and the owner decided on 2026-10-01 that it is not a hard rule. This spec is the one place a Model gains a dependency: the generated API classes and the ScalaPB runtime they need.
- **Fallback.** If R1 fails, the same author surface is delivered by a typed catalog that a small generator writes from the descriptors, with no runtime library. The acceptance criteria below hold for either mechanism.
- **Order.** fn-115 has moved the model and replaced the shell scripts with one gate program before this spec starts. This spec follows fn-113, which brings the ScalaPB toolchain, and precedes fn-112, whose realization helpers are then written against typed protos from the start.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Before any other implementation, a reviewable report shows that ScalaPB generates and compiles the Temporal API, the Testpilot IR and the needed well-known types for Scala 3.9.0; states the cold build time, the jar size, and the warm-cache time to compile the Models after a one-line edit, beside today's; and shows the lifter reading one nested field selection, one enum value, one message type and one unary method with typed request/response selections from a compiled Model and writing the proto names the IR has today. The descriptor closure includes the service and its imports, rather than assuming the existing message-only descriptor set is complete. The user owns the report commit. Errors: if generation or compilation fails, or the lifter cannot recover a proto name, the spec switches to the fallback and records why; if the warm-cache compile of the Models is more than twice today's, the task first tries generating only the messages the Models reference and the required service metadata, and switches to the fallback if that is still above twice; the cold build time is reported and does not decide.
- **R2:** An action names its schemas by type. The string form of `schema` is gone from the DSL and the lifter. Errors: a type that is no protobuf message does not compile.
- **R3:** A command that calls the server names its method as a value. Its assignments are typed by the method's request message and its response reads by the method's response message. A poll takes a typed reference to its read-evidence declaration: the `Recorded.Read` or `Single` source carries the method's request type and its selected projected-value type through that declaration into the poll. The lifter emits the same evidence ID as today. Errors: a field of another request/response message, or a poll assignment or condition with a root different from its evidence's request or projected value, does not compile.
- **R4:** Every existing field path a Model writes (assignments, response reads, poll conditions, evidence fields, operation keys, guards) is a typed selection, including repeated elements and oneof arms. Typed map entries in constant messages are covered by R7; unsupported map-entry read paths are not introduced. Errors: a path whose root message cannot be known statically names that message type explicitly; none is left as free text.
- **R5:** A literal, an enum value or a typed symbolic operand of the wrong type for its field does not compile. Errors: an operand whose type is unknown by design is listed in the done summary with the reason, and Go's check at lowering still covers it.
- **R6:** Enum values are written as the generated values. No enum name is built by string concatenation.
- **R7:** Constant messages a realization writes out are written with typed fields. `Proto`, `ProtoField` and `ProtoEntry` with string names are gone from the Models. Errors: a field the message does not have does not compile.
- **R8:** No Model contains a string literal that is a proto package, message, method, field path or enum value name, and a check in the model gate fails on a new one. Errors: an id that only looks like one, such as a Definition ID or a role id, is told apart by the check and stays.
- **R9:** The baseline goldens (`tools/umpire/model/testdata/migration`, `tools/umpire/lower/testdata/migration`) pass, and every checked-in IR file is equal before and after except for source positions. Errors: any other difference stops the task.
- **R10:** The lifter has fixtures that must not compile for a misspelled field, a field of the wrong message, a literal or symbolic value of the wrong type and an unknown enum value, and fixtures that lift for a repeated element, a oneof arm and typed map-entry construction in a constant message.
- **R11:** Go's validation of names and paths at lowering is unchanged, and its tests still pass (no error surface).
- **R12:** The model gate, `make lint-model` and the Go tests of the Umpire tooling pass at the closing task. A Model edit does not regenerate or recompile the API classes. The done summary states the count of proto string literals removed and the warm and cold build times before and after.

## Boundaries
<!-- scope: business -->

- No IR schema change. The IR keeps names and paths as text.
- No change to what any Model says.
- No Scala code builds, sends or parses a real Temporal message. Go runs every Case.
- No change to Go's lowering or validation.
- No typed form for ids that are the Model's own (roles, evidence kinds, commands). Those are fn-112 for the standalone activity and fn-114 for the other Models.
- The realization script helpers are fn-112. This spec types what they are built from.
- No new map-read path language, broad generated API drift gate or new CI coverage.

## Decision Context
<!-- scope: both — conditionally substructured -->

Two mechanisms were compared: ScalaPB classes on the Models' classpath, and a typed catalog written by a generator of our own.

The catalog's main advantage was that it kept the Models free of any library. The owner then said that is guidance and not a rule. What remains in the catalog's favor is a smaller build and no dependence on ScalaPB supporting Scala 3.9.0, and both of those are measurable in a day.

ScalaPB is preferred because a catalog with typed nested fields, oneofs, enums and methods typed by request and response is a reimplementation of what ScalaPB's generator already does, and the owner prefers a maintained library over code the project has to own. It also gives the real message types, which an editor documents and completes. The catalog stays as the fallback so that a failed check does not end the work.

Typing only names and leaving paths as strings was rejected. Field paths are the most numerous strings and the ones most easily misspelled.

The existing map use is a constant-message write, while Go's path walker rejects map reads.
Typing that write preserves the complete current author surface and the no-Go-change boundary;
adding a new map-read language is outside this spec. Descriptor-driven generation and focused
stamp invalidation stay in scope, while broad drift verification and CI expansion remain
declined in `.flow/memory/declined/generated-api-drift-verification.md`.

## Implementation approach

Task 1 resolves the whole-API versus referenced-closure generation choice using R1's two-times
warm-compile threshold and proves the actual selector and method spelling. Task 2 integrates the
chosen artifact and records the exact author contract. The reusable DSL then gains one generic
typing seam for schemas/methods, fields/operands and symbolic constant messages. The activity and
Nexus migrations are separate tasks so each preserves and verifies a bounded corpus. The final
task retires migration forms, checks proto strings, and updates the documentation with closure
evidence. Eight cohesive tasks avoid making the complete Model migration one large task.

Developers gain compiler checking and editor completion; Go consumers and deployment behavior
keep the same IR contracts. Generated artifacts add a descriptor-keyed build step. Model edits
reuse it, and missing or stale input errors name the build target. No runtime service or security
surface is introduced.

## Quick commands

Use the existing gate and fixture harness: `mise exec -- scala-cli test model/gate`,
`mise exec -- scala-cli test model/lifter`,
`CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`,
and `mise exec -- make lint-model`. Separate complete Go coverage uses `-json -tags test_dep -p 1 -parallel 1`
and one package at a time on this host. Reuse applicable passing results under `MILESTONES.md`;
the closing task records new and reused coverage, exact commands, exits and wall times.

## Early proof point

Task `fn-117.1` proves complete descriptors, typed name recovery and the warm-compile threshold.
If ScalaPB fails those checks, it proves the selective-generation or typed-catalog fallback
before any public API or Model migration begins.

## Requirement coverage

| Req | Task(s) | Gap justification |
| --- | --- | --- |
| R1 | 1, 2 | — |
| R2 | 3, 6, 7, 8 | — |
| R3 | 3, 4, 6, 7, 8 | — |
| R4 | 4, 6, 7, 8 | — |
| R5 | 4, 5, 6, 7, 8 | — |
| R6 | 5, 6, 7, 8 | — |
| R7 | 5, 6, 7, 8 | — |
| R8 | 6, 7, 8 | — |
| R9 | 6, 7, 8 | — |
| R10 | 3, 4, 5, 7, 8 | — |
| R11 | 8 | — |
| R12 | 2, 8 | — |

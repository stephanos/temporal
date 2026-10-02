# Type the Temporal API in the Models

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

The author surface, as a sketch. It shows only what this spec changes: names of declarations are still strings here, and `rpc`, `poll` and `message` stand for whatever constructors the realization declarations have when this spec starts. fn-112 builds its script helpers on top. The first task after R1 settles the exact spelling and records it here.

```scala
val start = action("start", caller)
  .creates(activity)
  .schema[StartActivityExecutionRequest]

rpc(WorkflowService.startActivityExecution)(
  _.activityType.name := Literal(activityType),
  _.taskQueue.name    := Environment(taskQueueBinding),
  _.requestId         := Run
)

poll(describe)(until = _.info.status === ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED)

val handlerFailure = message[Failure](
  _.message := "operation failed",
  _.applicationFailureInfo.nonRetryable := true
)
```

**Lifted meaning is frozen.** Every IR file is equal before and after this spec except for source positions. The typed forms lift to the same names and paths the strings spelled.

## Edge Cases & Constraints
<!-- scope: technical -->

- **ScalaPB must fit.** It has to generate and compile the Temporal API for Scala 3.9.0. fn-113 R4 checks that for the small IR schema; the Temporal API is hundreds of messages and has to be checked on its own. R1 does that before anything else.
- **Build time.** The owner prefers ScalaPB classes as long as compile time with a warm cache does not explode. The generated classes are compiled once into a jar and reused until the descriptors change, so a Model edit does not recompile them. R1 measures the warm-cache compile of the Models against today's, and more than twice today's counts as exploding.
- **Shapes beyond a plain field.** Realizations read every element of a repeated field (`history.events[*]`), one arm of a oneof (`attributes<nexus_operation_scheduled_event_attributes>`) and map entries. Each has a typed form, and the lifter writes the same path text the IR has today.
- **Typed operands.** A literal or an enum value of the wrong type for a field does not compile. A symbolic operand carries the type it was declared with. Where the type of a value is not known statically, such as the payload a Run Event's guard reads, the path is written against a named message type.
- **Testpilot's own messages.** Evidence paths such as `delivery_admission.decision` are fields of the Testpilot IR's `InstructionOutcome`. They are typed the same way, from the Testpilot proto.
- **Descriptors.** The classes are generated from the descriptor sets the repository already builds (`proto/api.binpb` and the internal protos). A missing or stale descriptor set fails the gate with the make target that builds it named.
- **Go keeps validating.** The checks at lowering stay as they are. They guard an IR file that was edited by hand or lifted by an older lifter.
- **Libraries in the Models.** Keeping the Models free of libraries is guidance, and the owner decided on 2026-10-01 that it is not a hard rule. This spec is the one place a Model gains a dependency: the generated API classes and the ScalaPB runtime they need.
- **Fallback.** If R1 fails, the same author surface is delivered by a typed catalog that a small generator writes from the descriptors, with no runtime library. The acceptance criteria below hold for either mechanism.
- **Order.** fn-115 has moved the model and replaced the shell scripts with one gate program before this spec starts. This spec follows fn-113, which brings the ScalaPB toolchain, and precedes fn-112, whose realization helpers are then written against typed protos from the start.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Before any other work, a committed report shows that ScalaPB generates and compiles the Temporal API, the Testpilot IR and the needed well-known types for Scala 3.9.0; states the cold build time, the jar size, and the warm-cache time to compile the Models after a one-line edit, beside today's; and shows the lifter reading one nested field selection, one enum value and one message type from a compiled Model and writing the proto names the IR has today. Errors: if generation or compilation fails, or the lifter cannot recover a proto name, the spec switches to the fallback and records why; if the warm-cache compile of the Models is more than twice today's, the task first tries generating only the messages the Models reference, and switches to the fallback if that is still above twice; the cold build time is reported and does not decide.
- **R2:** An action names its schemas by type. The string form of `schema` is gone from the DSL and the lifter. Errors: a type that is no protobuf message does not compile.
- **R3:** A command that calls the server names its method as a value. Its assignments are typed by the method's request message and its response reads by the method's response message. Errors: a field of another message does not compile.
- **R4:** Every field path a Model writes (assignments, response reads, poll conditions, evidence fields, operation keys, guards) is a typed selection, including repeated elements, oneof arms and map entries. Errors: a path whose root message cannot be known statically names that message type explicitly; none is left as free text.
- **R5:** A literal, an enum value or a typed symbolic operand of the wrong type for its field does not compile. Errors: an operand whose type is unknown by design is listed in the done summary with the reason, and Go's check at lowering still covers it.
- **R6:** Enum values are written as the generated values. No enum name is built by string concatenation.
- **R7:** Constant messages a realization writes out are written with typed fields. `Proto`, `ProtoField` and `ProtoEntry` with string names are gone from the Models. Errors: a field the message does not have does not compile.
- **R8:** No Model contains a string literal that is a proto package, message, method, field path or enum value name, and a check in the model gate fails on a new one. Errors: an id that only looks like one, such as a Definition ID or a role id, is told apart by the check and stays.
- **R9:** The baseline goldens pass, and every checked-in IR file is equal before and after except for source positions. Errors: any other difference stops the task.
- **R10:** The lifter has fixtures that must not compile for a misspelled field, a field of the wrong message, a value of the wrong type and an unknown enum value, and fixtures that lift for a repeated element, a oneof arm and a map entry.
- **R11:** Go's validation of names and paths at lowering is unchanged, and its tests still pass (no error surface).
- **R12:** The model gate, `make lint-scala` and the Go tests of the Umpire tooling pass at the closing task. A Model edit does not regenerate or recompile the API classes. The done summary states the count of proto string literals removed and the warm and cold build times before and after.

## Boundaries
<!-- scope: business -->

- No IR schema change. The IR keeps names and paths as text.
- No change to what any Model says.
- No Scala code builds, sends or parses a real Temporal message. Go runs every Case.
- No change to Go's lowering or validation.
- No typed form for ids that are the Model's own (roles, evidence kinds, commands). Those are fn-112 for the standalone activity and fn-114 for the other Models.
- The realization script helpers are fn-112. This spec types what they are built from.

## Decision Context
<!-- scope: both — conditionally substructured -->

Two mechanisms were compared: ScalaPB classes on the Models' classpath, and a typed catalog written by a generator of our own.

The catalog's main advantage was that it kept the Models free of any library. The owner then said that is guidance and not a rule. What remains in the catalog's favor is a smaller build and no dependence on ScalaPB supporting Scala 3.9.0, and both of those are measurable in a day.

ScalaPB is preferred because a catalog with typed nested fields, oneofs, enums and methods typed by request and response is a reimplementation of what ScalaPB's generator already does, and the owner prefers a maintained library over code the project has to own. It also gives the real message types, which an editor documents and completes. The catalog stays as the fallback so that a failed check does not end the work.

Typing only names and leaving paths as strings was rejected. Field paths are the most numerous strings and the ones most easily misspelled.

## Parked unknowns

- Whether the generated jar covers the whole Temporal API or only the messages the Models reference. R1's measurements decide it.
- Whether twice today's warm compile is the right line for "explodes". It is this spec's reading of the owner's condition; the owner can set another number when R1 reports.

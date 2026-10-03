# Typed Temporal API spike (fn-117.1)

**Decision:** use ScalaPB 0.11.20 for the complete linked descriptor closure, with generated gRPC method constants for typed unary methods. The warm Model edit with that jar took 0.77 s against 0.87 s for today's classpath (0.89×, below the 2× limit of 1.74 s). Selective generation and the typed-catalog fallback were therefore not triggered. The cold build is reported separately and is not a rejection rule. This is a spike only; no Model, DSL, lifter, gate, Go lowerer or checked-in generated artifact changed.

## Descriptor input and toolchain

`proto/api.binpb` contains 54 files and **omits** `temporal/api/workflowservice/v1/service.proto`, although `cmd/tools/getproto/files.go` links that Go descriptor. The reproducible scratch assembler at `.flow/tmp/fn117-1/descriptors.go` starts with the existing `proto/api.binpb` and adds the linked workflow service, all linked `temporal/server/api/testpilot/v1` and Umpire IR files, and their transitive imports through `protoreflect.FileDescriptor`. It sorts the 68 file names and marshals deterministically. The exact inputs are in `.flow/tmp/fn117-1/descriptor-files.txt`; they include `google/protobuf/duration.proto`, Google API annotations, Nexus annotations, `temporal/api/workflowservice/v1/request_response.proto`, the service, nine Testpilot files and Umpire IR. No file is inferred from a package-name string alone. A future gate must fail with `make proto/api.binpb` named when the source set is missing or stale.

The pinned tools actually used were protoc 29.5 (`mise exec -- protoc --version`), scala-cli 1.17.1 (`mise exec -- scala-cli version`), Scala 3.9.0/JDK 27 (`model/project.scala`, `model/lifter/project.scala`), the existing `model/gen/protoc-gen-scala` at ScalaPB 0.11.20, and ScalaPB runtime-grpc 0.11.20 for the generated method constants. The plugin options were `flat_package,scala3_sources,grpc`; [ScalaPB documents](https://scalapb.github.io/docs/customizations/) the Scala 3 source and package options and [its gRPC documentation](https://scalapb.github.io/docs/grpc/) describes generated method/client support. No service is called by the probe.

SHA-256: `proto/api.binpb` = `34c45d9e97ba4b44b0f53ef0b2217cb3fc165c627f019047080ef4e36d232988`; complete set = `300d1be3b95f9aeeeb4d3f7035715cf90b368e0f7ce24f55d3b9e6009814d9b4`. The scratch generator does not change getproto. A missing `proto/api.binpb` error in that assembler names its existing make target.

## Compiled author surface and lifted names

The scratch `Probe.scala` compiled against the final cold jar to `probe-cold.jar` (exit 0), and `Inspect.scala` read its **compiled TASTy**, not its source text (exit 0). The descriptor-backed inspector checks the generated ScalaPB and gRPC descriptors against selections in the tree. Its `RECOVERED` lines in `.flow/tmp/fn117-1/inspect-cold.log` are:

```text
message=temporal.api.workflowservice.v1.StartActivityExecutionRequest
request=task_queue.name response=run_id
method=/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution
input=temporal.api.workflowservice.v1.StartActivityExecutionRequest
output=temporal.api.workflowservice.v1.StartActivityExecutionResponse unary=true
enum=temporal.api.enums.v1.ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED
repeated=events[*].event_id oneof=attributes<nexus_operation_scheduled_event_attributes>
testpilot=delivery_admission
```

The inspector obtains the method from the tree's `WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION` selection and checks its `io.grpc.MethodDescriptor[StartActivityExecutionRequest, StartActivityExecutionResponse]` type arguments against the service descriptor's input/output and unary method type. It reads the request tree as `request.getTaskQueue.name`, maps both selected Scala fields through their ScalaPB descriptors, and recovers the exact existing `task_queue.name` IR path. It likewise maps `response.runId` to `run_id`, the generated enum value to its proto full name, and the message type to its descriptor. The repeated, oneof and Testpilot names are also derived from their respective compiled method bodies before descriptor lookup; an unexpected shape or changed proto path is refused. A one-line mutation from `_.eventId` to `_.version` compiled (exit 0, `probe-changed-compile.log`), then inspection rejected `events[*].version` (expected exit 1, `inspect-changed.log`). This is a bounded recovery probe for task 3 to incorporate; the current lifter was not changed.

The sketch `_.taskQueue.name` is invalid because `taskQueue` is `Option[TaskQueue]`; the generated `getTaskQueue.name` compiles and selects the same field path. A negative compile in `.flow/tmp/fn117-1/invalid-compile.log` exits 1 on `request.taskQueue.name` and on reversing the method's request/response type parameters. Repeated reads can use `history.events.map(_.eventId)` to represent `events[*].event_id`; a oneof arm is selected as `event.attributes.nexusOperationScheduledEventAttributes` (`Option[NexusOperationScheduledEventAttributes]`) and lifts to `attributes<nexus_operation_scheduled_event_attributes>`. `Payload.metadata` is `Map[String, ByteString]`; the scratch constructor `Payload(metadata = Map(key -> value))` compiles, establishing the key/value types. Production Models must express that map symbolically through typed fields/operands; the scratch function is never run and no Model constructs a protobuf message. `InstructionOutcome.deliveryAdmission` is another `Option`, so nested paths require a typed unwrap/getter analogous to the request path.

## Comparable timings and preservation

Each command below ran once in the foreground; exit status and `/usr/bin/time -p` wall time are in the named logs under `.flow/tmp/fn117-1/`. The complete cold run used **new task-local generated-source and jar paths**, so ScalaPB generated and scala-cli compiled the classes again. Dependency artifacts were already cached: neither cold log has a `Downloading` line. The earlier gRPC jar build logged dependency downloads in `package-grpc.log` and is not used as the clean cold measurement.

| Work | Exit | Wall time | Log |
| --- | ---: | ---: | --- |
| complete descriptor assembly, 54 → 68 files | 0 | 10.35 s (Go build included) | `descriptors.log` |
| cold ScalaPB generation, 957 Scala sources | 0 | 2.01 s | `cold-generate.log` |
| cold ScalaPB compilation and jar packaging | 0 | 31.92 s | `cold-package.log` |
| today's Model+DSL corpus, priming compile | 0 | 2.05 s | `model-real-baseline-prime.log` |
| today's corpus, one-line Model edit | 0 | **0.87 s** | `model-real-baseline-edit.log` |
| same corpus plus the final ScalaPB gRPC jar/runtime, priming compile | 0 | 2.20 s | `model-final-candidate-prime.log` |
| same corpus/classpath, one-line Model edit | 0 | **0.77 s** | `model-final-candidate-edit.log` |

The cold generation plus packaging is **33.93 s**, excluding Go assembler startup and dependency downloads. The jar is **29,938,670 bytes** (28.55 MiB), SHA-256 `f20e85969c43dfc1a51903c2cca17e6f6529bdc2d23242222140016d679e59be`. Its generated classes are compiled once; the warm candidate command passed the existing jar and did not invoke protoc or the jar build. The candidate's only additional classpath inputs were that jar and `com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20` (including transitive gRPC API jars). The ScalaPB gRPC dependency is a compile-time typing cost and must be declared in the task-2 artifact setup.

For the edit comparison, `.flow/tmp/fn117-1/temporal-copy/` contains copies of **all 13 current `model/temporal` Scala sources**, plus the original `model/project.scala` and `model/umpire` sources in both commands. Before the edit, the copied standalone activity `Model.scala` had the same SHA-256 as the working source (`9f0dae7fdd68b9c48a2776be0dbe7d102b668f7fc3c0057b82ee7bc8a94bf958`). The baseline compile was primed; its one-line edit changed `Party("caller")` to `Party("caller-spike")`. The final jar's classpath was primed on the same corpus with the caller literal at `caller-spike2`; its one-line edit changed the same Model line to `Party("caller-spike3")`. `model-edit.diff` shows the final single-line difference; both edit logs contain `Compiling project` and `Compiled project`. The commands were, with the same mise shell environment and source paths:

```sh
mise exec -- scala-cli compile model/project.scala model/umpire .flow/tmp/fn117-1/temporal-copy
mise exec -- scala-cli compile model/project.scala model/umpire .flow/tmp/fn117-1/temporal-copy --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
```

The required baseline `mise exec -- scala-cli compile model/lifter` passed before probing. Unaffected broad checks were reused from `.flow/tmp/fn113-16-evidence.json` and the later `.flow/tmp/fn113-3-completion-fix/{lifter-test-result.json,check-model-result.json}`: Go package tests/vet/lint and the Scala lifter/check-mode gate/lint were green for the unchanged production inputs. No getproto code changed, so its focused Go tests and Go lint were not triggered. Hashes of all **six IR files, 16 Case JSON files plus their manifest, and six positive fixture JSON outputs plus `rejects.txt`** were captured before and after in `artifacts-before.sha256` and `artifacts-after.sha256`; `artifacts.diff` is empty (exit 0).

## Reproduction and next task

From the repository root, first ensure the existing `proto/api.binpb` target is current. The exact scratch programs and logs are in `.flow/tmp/fn117-1/`; the main steps are:

```sh
mise exec -- go run .flow/tmp/fn117-1/descriptors.go proto/api.binpb .flow/tmp/fn117-1/api-complete.binpb > .flow/tmp/fn117-1/descriptor-files.txt
mise exec -- protoc --plugin=protoc-gen-scala=model/gen/protoc-gen-scala --descriptor_set_in=.flow/tmp/fn117-1/api-complete.binpb --scala_out=flat_package,scala3_sources,grpc:.flow/tmp/fn117-1/generated-cold $(cat .flow/tmp/fn117-1/descriptor-files.txt)
mise exec -- scala-cli --power package --library .flow/tmp/fn117-1/generated-cold --scala 3.9.0 --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -f -o .flow/tmp/fn117-1/api-scalapb-cold.jar
mise exec -- scala-cli --power package --library .flow/tmp/fn117-1/Probe.scala --scala 3.9.0 --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -f -o .flow/tmp/fn117-1/probe-cold.jar
mise exec -- scala-cli run .flow/tmp/fn117-1/Inspect.scala --scala 3.9.0 --dep org.scala-lang::scala3-tasty-inspector:3.9.0 --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --jar .flow/tmp/fn117-1/probe-cold.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -- .flow/tmp/fn117-1/probe-cold.jar
```

Task 2 can integrate a descriptor-keyed, stamped jar built from this closure and put it on the Models' compile classpath. It must retain the method service descriptor and the generated typed method constant, reuse the jar on a Model edit, and fail with the existing make target named for absent inputs. The symbolic DSL and generic TASTy-to-descriptor recovery belong to later tasks. There is no reason under R1's measured threshold to implement a referenced-message closure or own a typed catalog.

# Testpilot

Testpilot runs behavior through Temporal and Workers using bounded Cases. Callers decode or construct a
`testpilot/v1` Case, prepare it against an immutable `Profile`, then execute the
returned `PreparedCase` through a caller-owned `Driver`.

Exact Case 1.0 is the only admitted format. A Program's symbolic binding graph is the closed set of
text binding IDs its roles and expressions reference, which `Prepare` derives and resolves against the
Profile; a resource-free Program references none. The Case owns the IDs and relationships; the Profile
owns their physical values. Symbolic endpoint IDs are not transport addresses, and bindings grant no
capabilities. Preparation also derives what a Case no longer writes: an instruction's outcome fields
follow from its instruction, the worker activations a reservation carrier reserves follow from the
Profile's carriers, and an instruction limit the Case omits takes the Profile's
`InstructionDefaults`.

`Prepare` performs static admission without Driver I/O, snapshots the Case and Profile, resolves
private prepared resources, and includes the complete binding fingerprint in Prepared Case identity.
`PreparedCase.Run` checks the Driver identity, calls `Driver.Validate` without target I/O, creates the
Monitor, and only then opens a per-Run `Session`. Validation failure produces no Session, Run, Verdict,
or effect. Scheduling, recording, expression admission, and Contract evaluation stay private to this
package. The reusable Temporal Driver lives in `common/testing/testpilot/temporal`; functional
fixtures and provisioning remain under `tests/`. Drivers cannot replace the prepared Contract evaluator.

Driver authors import this package. Its `Session`, handle, `Coordinate`, role-policy and `Opcode` types
are aliases of the `common/testing/testpilot/contract` leaf, which imports neither private execution nor
the IR, so a package that needs only that vocabulary may import the leaf instead. Execution hands every
effect and reservation handle back to the Session that issued it; refusing a handle it did not issue is
that Session's decision.

A Case may ask its Driver for a deliberate outage. `InjectFault` is a declared instruction like any
other: the Profile must authorize it, the role it names must be a task-queue role, and the Run
records one `FAULT_INJECTED` event per realized outage, carrying the fault in its `fault_injected`
payload, which a Contract reads through a path. Nothing about a requested fault is evidence until
that event exists.

A worker instruction may carry the Temporal API message the Driver realizes through its SDK
(fn-85 R10): `WorkflowCommand` carries a `temporal.api.command.v1.Command`, `NexusHandlerReply` a
`temporal.api.nexus.v1.StartOperationResponse` or `HandlerError`, and `NexusOperationCompletion` a
`temporal.api.common.v1.Payload` or `temporal.api.failure.v1.Failure`. The message is carried whole,
not evaluated: preparation admits it against the Driver-reach table in
`internal/execution/typed.go`, which names per message the fields the Driver realizes and the fields
it does not, so a field the Driver cannot set rejects `unsupported` at the field's own path, an
invalid or over-ceiling duration rejects `malformed` or `limit_exceeded` at its field, a reply the
activation does not admit rejects at its node, and a command whose type the Profile's
`CommandTypes` does not list rejects `unsupported` at `command_type`. `temporal.DeriveProfile` lists
the command types the worker Driver realizes (`worker.CommandTypes`) and nothing more. A command's
endpoint field names the Case's endpoint role; the Driver resolves it to the bound resource. The
Await of a scheduled command yields the handler's payload whole, as an `Any`, where the Await of the
untyped start yields text.

A Case may also declare where its operation-correlated evidence comes from. A response read can
lift a projected value into a declared `CorrelatedEvidence` Observation through guarded rules, which is
the only way a Program supplies the evidence a `Contract.correlated` Correlated Contract reads. A
Correlated Contract that admits no evidence answers inconclusive: silence is not a satisfied property.

## Field paths and enum literals

A Case reads and writes protobuf fields through field paths: `PathExpression.path`, a request
assignment's `target`, a response read's `path` and an evidence-lift rule's `operation`. Each is a
string in one grammar, which preparation parses and types against the message it addresses:

```text
path     = "" | segment { "." segment }
segment  = name [ selector ]
selector = "<" name ">" | "[*]" | "[" key "]" | "?"
key      = JSON string | [ "-" ] digit { digit } | "true" | "false"
name     = ( letter | "_" ) { letter | digit | "_" }
```

- A `name` is a protobuf field name, not its JSON name. The empty path is the whole value.
- `<member>` after a oneof's name reads that oneof's member `member`, absent unless it is the selected
  one: `attributes<nexus_operation_completed_event_attributes>.scheduled_event_id`.
- `[*]` fans out over every element of a repeated field, so the path's value is a list:
  `history.events[*]`.
- `[key]` reads the entry of a map field with that key, absent when no entry has it. The key's kind
  must be the map's: a JSON string for text keys (`labels["key"]`), a canonical base-10 integer for
  integer keys (`counts[-42]`), `true` or `false` for boolean keys.
- A final `?` reads whether a presence-tracking field is set, as a boolean: `child.optional_text?`.

A segment takes at most one selector. `Testpilot.Authoring.Path.make` is the one Lean printer and
spells a text key escaping only the quote, the backslash and control characters. A path outside the
grammar, an unknown field or member, or a key of the wrong kind rejects at preparation located at the
path's field and quoting its text.

An enum literal is `EnumValue { name }`. Preparation resolves the name against the enum its context
expects (the other operand of a comparison, or the field a request assignment targets) and rejects an
undeclared name, a name where the expected type is no enum, and an enum literal with no expected type,
quoting the name. A value read from a protobuf message carries its name too; a number the enum does
not declare is spelled in decimal, which no literal names.

## Extending the protocol

The protocol has no compatibility promise, so an extension changes it in place. The same places change
in the same order for every kind of extension; the lists below say what each place needs for a new
instruction, fault kind, Run Event payload and expression reference, and the worker-stop fault kind
is traced through all of them as the worked example. fn-85 R10's typed worker instructions are the
first planned use.

### Every extension

1. **Protocol.** Edit the file that owns the concept under
   `proto/internal/temporal/server/api/testpilot/v1`. A new message or enum carries a leading comment,
   field numbers stay dense from 1, and a new oneof arm is appended;
   `TestProtocolMessagesCarryLeadingComments` enforces the first two. A new file must be reachable from
   `case.proto` or `run.proto` and is listed in `TESTPILOT_PROTOCOL_PROTOS` (the Makefile),
   `testpilotProtocolSchemas` (`model/lakefile.lean`), `protocolFiles` (`protocol_test.go`) and the file
   list in `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go`.
   A Run-only message never enters the Case closure (`TestCaseImportClosureExcludesRunOnlyMessages`).
   A field that carries a public API message imports that message's file from `proto/api.binpb`
   (`--descriptor_set_in`, already passed by `make proto`, `umpire-check-testpilot-protocol` and
   `Testpilot/Protocol.lean`); the file's whole import closure is compiled once into Lean by
   `Testpilot/Carried.lean`, so each file the closure adds is appended to `Testpilot.Carried.files`,
   which `Testpilot.Protocol` checks.
2. **Generated code.** `make proto` regenerates `api/testpilot/v1` and runs the api-linter; a singular
   enum field naming an enum from another file of the package compiles only through the rewrite in
   `cmd/tools/protogen/enum_references.go`. `Testpilot/Protocol.lean` elaborates both closures with one
   `protoc` call, and `make umpire-check-testpilot-protocol` checks it.
3. **`Testpilot.Authoring`.** Add the constructor Producers write (`model/Testpilot/Authoring.lean`) and
   guard it in `model/Testpilot/Tests/Authoring.lean`; `make umpire-check-testpilot-authoring` decodes
   the Lean ProtoJSON strictly in Go.
4. **Go interpreter or evaluator.** Instructions bind in `internal/execution` and run in its scheduler
   or the worker interpreter; references and paths bind in `internal/ir`; Contracts evaluate in
   `internal/verification`.
5. **The table that classifies it.** `ir.RunEventPayloadOf` for payloads, `ir.admittedReferences` for
   references, `execution.InstructionOpcode` for instructions.
6. **Profile Opcode and Driver.** `contract.Opcode` and `contract.Session` in the Driver leaf, then the
   Temporal Drivers under `temporal/`.
7. **Conformance class or unit test.** The six Driver-independent classes under
   `testdata/case-runtime-conformance` change only for a new Verdict shape or rejection; everything else
   is pinned by a focused unit test beside the code, and live behavior by a `TestTestpilot*` test.
8. **Retired-vocabulary gate.** A rename or removal adds the old spelling to `buildRetiredRules` in
   `tools/umpire/internal/retiredvocabulary/check.go` with a line in
   `TestRetiredRulesHoldTheGlossaryRenamedProtocolNames`, and a removed descriptor name to
   `TestProtocolUsesCohesivePublicVocabulary`. A new name that matches a retired rule fails the gate.
9. **Fixtures.** Regenerate through `make umpire-gen-case-runtime-conformance`, never by hand;
   `make umpire-check-case-runtime-conformance` fails on a stale fixture.

### A new instruction

1. A message in `instruction.proto` and an arm appended to `Instruction.instruction`. The arm's field
   number is its Opcode.
2. `make proto`.
3. A `Testpilot.Authoring.Program` constructor beside `Program.injectFault`.
4. `execution.InstructionOpcode` and `opcodeContext` (which entrypoint kind may declare it), a binder in
   `admission.bindInstruction` and `admission.bindNodeDataflow`, its outcome fields in
   `admission.bindOutcomes`, dispatch in `scheduler.acceptEffect` and any event it records in
   `scheduler.publishCompletion`. A workflow instruction also runs in `workflowInterpreter.execute`
   (`temporal/worker/interpreter.go`); a Nexus-handler instruction in `Session.interpretNexus`. An
   instruction that starts a Nexus operation is also named by `execution.startsNexusOperation`,
   which the carrier route derivation and `bindAwait` read, and by the worker's
   `startsNexusOperation` and `addInstructionBindings`, which prepare its dispatch route and
   endpoint. An instruction that carries a public API message gets a row per carried message in the
   Driver-reach table (`execution/typed.go`), which `TestDriverReachTableNamesEveryField` requires to
   name every field of every carried message, and the Driver's interpreter reads only the fields
   the row names realized.
5. `InstructionOpcode` is the table: `TestInstructionOpcodesCoverTheInstructionTable` requires every
   oneof arm to map to the Opcode of its field number.
6. Append the Opcode to `contract.Opcode`, move `contract.MaxOpcode` and alias it in the facade's
   `contract.go`; `temporal.DeriveProfile` authorizes it through `testpilot.InstructionOpcode`, and
   a workflow command's type through `worker.CommandTypes`. A new Driver effect adds a
   `contract.Session` method, implemented by the server, worker and composite Sessions and by every
   test Session.
7. Focused tests beside the binder and the Driver, and a Driver test per carried message
   (`temporal/worker/typed_test.go`); a preparation rejection the instruction adds is a variant of
   the `static-preparation-rejection` conformance class (`productionManifest` in the generator,
   `Temporal.Testpilot.Conformance` in Lean, the facade Profile in `conformance_test.go`).
8. and 9. As above.

### A new fault kind

1. A value of `FaultKind` in `instruction.proto`. `InjectFault.kind` and the recorded
   `FaultInjected.kind` share the enum. The enum's comment admits only worker-lifecycle transitions on
   one activation queue, so any other outage amends it.
2. `make proto`.
3. No new constructor: `Program.injectFault` takes any kind.
4. Widen the kind range `admission.bindFault` admits. Dispatch and recording carry the kind unchanged,
   and a Contract enum literal resolves by name against the `FaultKind` field it is compared with.
5. No table change: `FAULT_INJECTED` already carries the `fault_injected` arm.
6. No new Opcode: `contract.InjectFault` authorizes every kind. The Driver that realizes the outage
   maps the kind to behavior.
7. Unit tests of admission, the Driver transition and a Contract reading the kind.
8. and 9. As above.

### A new Run Event payload

1. A message in `run.proto` (Run-only: add it to `runOnlyMessages` in `protocol_test.go`) and an arm
   appended to `RunEvent.payload`; a new kind appended to `RunEventKind` in `event.proto`.
2. `make proto`.
3. `Run.event` already takes any `payload`. A Contract reads the arm as
   `Expr.path Expr.runEventPayload "<arm>.<field>"`, so no reference is added.
4. The component that records the event sets the arm (`scheduler.publishCompletion` for instruction
   events). `ir.CheckRunEventPayload` records a mismatch as an `INVARIANT` diagnostic
   `payload_kind_mismatch`. A new kind moves `ir.MaxRunEventKind`.
5. `ir.RunEventPayloadOf`: the arm each kind may carry and whether it requires it. Contract preparation
   admits a path into the arm only for filters whose kinds carry it.
6. A Driver change only when a Driver supplies the payload's data.
7. `TestRunEventPayloadTableNamesEveryArm`, `TestCheckRunEventPayloadMatchesTheKind`,
   `TestRunEventPayloadPathsBindThroughTheArmTheyName`,
   `TestRecorderRejectsPayloadKindMismatchAsInvariant` and
   `TestPrepareLocatesPayloadPathsTheFilterCannotCarry` each gain the arm.
8. and 9. As above.

### A new expression reference

1. An arm appended to `Reference.reference` in `expression.proto`, with its message when the reference
   is structured.
2. `make proto`.
3. An `Expr` constructor beside `Expr.capture` and `Expr.runEventPayload`. A correlated reference is
   also admitted by `Testpilot.Correlated.decode`.
4. `ir.ReferenceKind` and `compiler.reference` in `internal/ir/expression.go`, and the resolver of the
   context that admits it: execution for the Program context, verification for the Contract context,
   and `verification/correlated_prepare.go` and `correlated.go` for the correlated context, which admits
   and evaluates its own conditions.
5. `ir.admittedReferences`, the context table. A reference outside its contexts rejects `unknown` at its
   path.
6. No Opcode or Driver change.
7. `TestExpressionContextsRejectReferencesOutsideThem` covers every arm in every context; the
   `static-preparation-rejection/expression-context` conformance variant pins the rejection's shape.
8. and 9. As above.

### Worked example: `FAULT_KIND_WORKER_STOP`

1. **Protocol.** `FAULT_KIND_WORKER_STOP = 1` in `FaultKind` (`instruction.proto`), requested by
   `InjectFault { role_id, kind }` and recorded as `FaultInjected { role_id, kind }` (`run.proto`).
2. **Generated code.** `make proto` writes `testpilotspb.FAULT_KIND_WORKER_STOP`, and
   `Testpilot.Protocol` generates the Lean constructor `FaultKind.FAULT_KIND_WORKER_STOP`. No file was
   added.
3. **`Testpilot.Authoring`.** `Program.injectFault "queue" .FAULT_KIND_WORKER_STOP`, guarded by
   `injectFaultNamesRoleAndKind`. Producers reach it two ways: `Umpire.faultKindOf`
   (`model/Umpire/Variations/Lowering.lean`) maps `Umpire.workerStopCapabilityId` to the kind, and
   `faultKindName` (`model/Temporal/Testpilot/WorkerOutage.lean`) names it in an exhaustive match, so a
   new kind is a Lean error there until it is named.
4. **Go interpreter and evaluator.** `admission.bindFault` (`internal/execution/dataflow.go`) admits
   kinds from `FAULT_KIND_WORKER_STOP` to `FAULT_KIND_WORKER_RESUME` on a task-queue role.
   `scheduler.acceptEffect` calls `Session.InjectFault` with the kind, and `scheduler.publishCompletion`
   records `RUN_EVENT_KIND_FAULT_INJECTED` with the `fault_injected` payload after a successful outcome.
   The worker-outage Contract compares `path(run_event.payload, fault_injected.kind)` with
   `EnumValue { name: "FAULT_KIND_WORKER_STOP" }`, which preparation resolves against that field's enum.
5. **Table.** `ir.RunEventPayloadOf(RUN_EVENT_KIND_FAULT_INJECTED)` is the required `fault_injected`
   arm; unchanged.
6. **Opcode and Driver.** `contract.InjectFault` and `contract.Session.InjectFault`, unchanged. The
   worker Driver realizes it: `worker.PlanOutages` resolves the role's queue, `OutagePlan.resolve`
   (`temporal/worker/outage.go`) maps the kind to a stop and rejects any kind it does not name, and
   `Outage.Begin` returns the `Settle` that stops the Run's dedicated SDK worker. The server `Session`
   refuses every fault, and the composite `compositeSession` routes faults to the worker Session.
7. **Tests.** `TestPrepareAdmitsFaultInjection` and `TestSchedulerRecordsOneFaultEventPerInstruction`
   (`internal/execution/fault_test.go`), `TestEvaluatorMatchesRecordedFaultPayload`
   (`internal/verification/fault_test.go`), `TestFaultTransitionsTheNamedQueue` and its siblings
   (`temporal/worker/outage_test.go`), and live `TestTestpilotWorkerOutageCase`. No conformance class
   covers faults.
8. **Retired vocabulary.** Nothing to retire for an added kind.
9. **Equivalence mapping.** An added enum value changes no baseline fixture, so no step; a Producer
   that starts writing it into `worker-outage-case.json` needs one.
10. **Fixtures.** `worker-outage-case.json` is rendered from `Temporal.Testpilot.WorkerOutage` (listed
    as `worker-outage` in `model/Temporal/Tool/Testpilot.lean`) by
    `make umpire-gen-case-runtime-conformance`.

## Preparation diagnostics

`NewCatalog`, `Prepare`, and `ProfileSpec.BindingFingerprint` return errors discoverable as
`*testpilot.PreparationError` through `errors.As`, including after ordinary error wrapping:

```go
prepared, err := testpilot.Prepare(source, profile)
if err != nil {
    var diagnostic *testpilot.PreparationError
    if errors.As(err, &diagnostic) {
        log.Printf("preparation %s at %s: %s", diagnostic.Category, diagnostic.Path, diagnostic.Detail)
    }
    return err
}
```

The six stable categories are `malformed`, `unknown`, `type_mismatch`, `unavailable`,
`unsupported`, and `limit_exceeded`. `Path` retains the admission input vocabulary and its
256-byte bound; `Detail` is human-readable, not a stable string API. Existing error messages,
including Contract rule context, are retained. Missing or invalid Profile/Catalog preconditions
are malformed; Profile binding validation, including binding ceilings, is also malformed without
changing its rejection limits. A Case declares no resource ceilings: Profile ceiling violations and a
Case bound (an instruction timeout or attempt count) above its Profile ceiling retain their existing
categories.

These diagnostics cover static admission, including correlated Contracts. ProtoJSON decoding errors
and runtime Run/Driver failures keep their own error contracts. No diagnostic wire format is added.
See [the public error contract](preparation_error.go) and [Temporal Driver ownership](temporal/README.md).

## Running a Case from the command line

`tools/umpire/cmd/umpire-run` is the black-box consumer of these bytes. Given a fixture path, a gRPC
address, an HTTP address, and the namespace, task queue and optional Nexus endpoint the Case binds
to, it derives the Profile the Case implies through `temporal.DeriveProfile`, prepares the unchanged
bytes, opens a composite Driver with its own SDK worker, runs once, and prints the Run disposition, the
cleanup status, the Verdict status, and one line per rule Verdict.

Its exit codes separate the answer from the infrastructure: `0` satisfied, `1` violated, `2`
inconclusive, `3` preparation, infrastructure, or Run error. That is why a CI caller can tell an
unreachable server from a Run that really was inconclusive.

With `--create` it provisions the resources it names and deletes them on exit; without it they must
already exist and none is ever deleted. Only Cases whose Profile `DeriveProfile` derives are
runnable; a typed fixture rejects with its admission category on stderr. It links the Driver and the
SDK, never the functional test cluster.

It links no functional test cluster at all, and the only server packages in its transitive closure
are the three the Driver's own Nexus support already pulled in through `common/dynamicconfig`,
`common/persistence` and `chasm`. A unit test pins both halves of that, so a new coupling is a
failing test rather than a silent regression.

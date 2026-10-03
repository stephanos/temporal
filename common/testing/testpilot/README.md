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
or effect. `PreparedCase.Evaluate` replays a closed Run's events through the same prepared Contract
with no Driver and no target, the offline semantic replay: it returns the Verdict that reading gives
and, per violated Rule instance or correlated rule, the Run Event whose evidence resolved it and that
evidence (a Rule instance's observation ids, or none when its Deadline violated it; a correlated
rule's evidence kind). Scheduling, recording, expression admission, and Contract evaluation stay private to this
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

Delivery controls reach inside the server: a hold is done once the server
holds a dispatch of the activity the Run starts on the role's queue, and a release delivers the
held dispatch to the server's authoritative admission and is done once what admission committed for
it is observed. A successful release's outcome carries that decision as its `delivery_admission`, and
the response-loss control below also carries its committed decision. Only an environment that runs the server can supply the control, so a
Profile says whether its environment does (`ProfileSpec.DeliveryControl`), and a Case that holds a
delivery is refused under a Profile that does not, at preparation, naming the instruction.

`ADMISSION_RESPONSE_LOSS` releases a held dispatch and replaces one successful history admission
response with `Unavailable`. It succeeds after a retry with the same execution, stamp and request ID
confirms that replacement reached the caller. The outcome carries the original committed attempt;
an obsolete retry never establishes a rejection. The environment must supply `HeldDelivery` loss
control before any target I/O. Cleanup closes the hold and disables the response hook.

A worker instruction may carry the Temporal API message the Driver realizes through its SDK
(fn-85 R10): `WorkflowCommand` carries a `temporal.api.command.v1.Command`, `NexusHandlerReply` a
`temporal.api.nexus.v1.StartOperationResponse` or `HandlerError`, and `NexusOperationCompletion` a
`temporal.api.common.v1.Payload` or `temporal.api.failure.v1.Failure`. The message is carried whole,
not evaluated: preparation admits it against the Driver-reach table in
`internal/execution/typed.go`, which names per message the fields the Driver realizes and the fields
it does not, so a field the Driver cannot set rejects `unsupported` at the field's own path, an
invalid or over-ceiling duration rejects `malformed` or `limit_exceeded` at its field, a reply the
activation does not admit rejects at its node, and a command whose type the Profile's
`CommandTypes` does not list rejects `unsupported` at `command_type`, naming the type.
`temporal.DeriveProfile` lists the command types the worker Driver realizes (`worker.CommandTypes`:
scheduling a Nexus operation and scheduling an activity) and nothing more. A command's endpoint
field names the Case's endpoint role, and an activity schedule's task queue a task-queue role; the
Driver resolves each to the bound resource. The Await of a scheduled command yields the payload the
handler or the activity answered, whole, as an `Any`, where the Await of the untyped start yields
text.

A Case may also declare where its operation-correlated evidence comes from. A response read can
lift a projected value into a declared `CorrelatedEvidence` Observation through guarded rules, which is
the only way a Program supplies the evidence a `Contract.correlated` Correlated Contract reads. A
Correlated Contract that admits no evidence answers inconclusive: silence is not a satisfied property.

A Program declares each kind of that evidence once, in `Program.evidence`: the recorded data it is
read from, the Run coordinates that scope it, the path of its operation key and the fields it
exposes. The source is one of three. A history event kind is an arm of the recorded `HistoryEvent`'s
attributes oneof, lifted by a history read whose rule names the declaration (`evidence_id`) and
spells nothing else. A Run Event kind is lifted by the runtime as it records the event, out of the
event's payload: an injected fault becomes `faultInjected` evidence keyed by the role it stopped. A
read is a repeated field in the response of a unary RPC, polled from a controller by a
`ReadEvidence` instruction until an element satisfies its `until` or the instruction times out, and
every element the condition selects is lifted; `pendingAttempts` reads
`DescribeWorkflowExecution`'s `pending_nexus_operations.attempt` keyed by `scheduled_event_id`.
A read that sets `single` reads the one message at its path instead, with the same method
authorization, descriptor checks, polling and limits: a response that lacks the message supplies
nothing, as an empty repeated field does, and the message is one event at most.

A Run Event declaration may carry a `guard`, a boolean over the event's payload in the evidence-lift
context, which selects the events of the kind that are evidence. An event it rejects is no
occurrence and takes no ordinal; a guard that might have no value rejects at preparation, so it is
never read as false; and an event two declarations accept is evidence of neither and makes the Run
incomplete. The record of a worker reservation is a `DIAGNOSTIC` event whose outcome names the
activity attempt, so a guard that compares `activity_attempt.sdk_attempt` as greater than zero and
`activity_attempt.delivery_id` as not empty selects the attempts a worker was delivered, keyed by
`activity_attempt.activity_run_id`, and leaves out a position recorded as not needed. A presence
check would not: a scalar the record leaves at zero still reads as a value. That record says what
the worker was delivered and offered. It is never the server's acceptance of the offer, and the
offered response is an enum, which no evidence field reads.

A Run Event declaration may also name the one controller `instruction` whose events it
reads: that instruction's completion or timeout, the fault it realized, and the record of each
reservation it carried. And it may be `run_keyed`: the operation key of its evidence is then the
Run's own ID, the value a Program input reads as its Run reference, and it writes no operation
path. That joins the Run's record of a call to evidence read back from the target under a key the
Case set to the Run's ID.

Kinds may share a source. One emitter numbers a source's evidence in one dense stream, whichever
kind each piece is: the instruction whose lift names the kinds, or the Run for its own events, and
never both. Within a source and under one operation key each recorded kind is declared once: a
history arm, or a Run Event kind at an instruction under a guard. A Correlated Contract's projection rules name the
declarations by kind, so the kind, source, key path and fields are written once and cannot drift;
a reference to an undeclared kind, the same recorded kind declared twice under a source and key
path, or a source the Run and an instruction would both count, rejects at preparation. A Program
that declares nothing keeps the spelled-out lift rules, which slot-bound reads still use.

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

A segment takes at most one selector. A text key is spelled escaping only the quote, the backslash
and control characters. A path outside the
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
instruction, fault kind, Run Event payload and expression reference and for a removal, and the
worker-stop fault kind is traced through all of them as the worked example. fn-85 R10's typed worker instructions are the
first planned use.

### Every extension

1. **Protocol.** Edit the file that owns the concept under
   `proto/internal/temporal/server/api/testpilot/v1`. A new message or enum carries a leading comment,
   field numbers stay dense from 1, and a new oneof arm is appended;
   `TestProtocolMessagesCarryLeadingComments` enforces the first two. A new file must be reachable from
   `case.proto` or `run.proto` and is listed in `TESTPILOT_PROTOCOL_PROTOS` (the Makefile) and
   `protocolFiles` (`protocol_test.go`).
   A Run-only message never enters the Case closure (`TestCaseImportClosureExcludesRunOnlyMessages`).
   A field that carries a public API message imports that message's file from `proto/api.binpb`
   (`--descriptor_set_in`, already passed by `make proto` and `umpire-check-testpilot-protocol`).
2. **Generated code.** `make proto` regenerates `api/testpilot/v1` and runs the api-linter; a singular
   enum field naming an enum from another file of the package compiles only through the rewrite in
   `cmd/tools/protogen/enum_references.go`. `make umpire-check-testpilot-protocol` compiles the
   protocol closure with one `protoc` call and checks its leading comments.
3. **The Producer.** Add the builder the lowering writes the element with
   (`tools/umpire/lower/internal/producer/build.go`) and a lowering test beside it. An element a Model
   has to ask for also needs its realization declaration in `model/umpire/realize`, its field in the
   Umpire IR and its case in the lifter and in `tools/umpire/lower/realization.go`;
   `model/SEMANTICS.md` (Realizations) says what each declaration lowers to.
4. **Go interpreter or evaluator.** Instructions bind in `internal/execution` and run in its scheduler
   or the worker interpreter; references and paths bind in `internal/ir`; Contracts evaluate in
   `internal/verification`.
5. **The table that classifies it.** `ir.RunEventPayloadOf` for payloads, `ir.admittedReferences` for
   references, one row in `execution.opcodes` for instructions.
6. **Profile Opcode and Driver.** `contract.Opcode` and `contract.Session` in the Driver leaf, then the
   Temporal Drivers under `temporal/`.
7. **Conformance class or unit test.** The six Driver-independent classes under
   `testdata/case-runtime-conformance` change only for a new Verdict shape or rejection; everything else
   is pinned by a focused unit test beside the code, and live behavior by a `TestTestpilot*` test.
8. **Retired names.** A rename or removal adds the removed descriptor name to
   `TestProtocolUsesCohesivePublicVocabulary` (`protocol_test.go`), which fails when a retired name
   is declared again.
9. **Fixtures.** Regenerate the lowered Cases through `make umpire-gen-model`,
   `make umpire-gen-fixtures` and `make canary-gen-case`, never by hand; `make umpire-check-cases`,
   `make umpire-check-fixtures` and `make canary-check-case` fail on a stale one. The fixtures under
   `testdata/case-runtime-conformance` are retained bytes that no live target renders: a change
   that would move one of them needs a decision first, and `conformance_test.go` fails on it until
   then.

A Contract Rule may declare instance values and a list of Rule instances (`ContractRule.instances`).
Preparation binds the Rule once and charges every ceiling per Rule instance, as the expansion (one
plain Rule per instance, each instance value inlined as a literal) would be charged, so an extension
that adds binding or evaluation work to a Contract charges it per instance too. The expansion is
written once, as `ir.ExpandRule`, and `ir.HasRuleInstances` gates the expanded-surface charge that
`execution.Prepare` and `verification.Prepare` both apply. A differential test
(`TestRuleInstancesEvaluateAsTheirExpansion`) holds each Contract with instances to its expansion's
Verdict and admission.

### A new instruction

1. A message in `instruction.proto` and an arm appended to `Instruction.instruction`. The arm's field
   number is its Opcode; a removed arm's successors move up, so the numbers stay dense from 1.
2. `make proto`.
3. A builder in the Producer beside `InvokeRPC` and `ReadEvidence`.
4. One row in `opcodes` (`internal/execution/dataflow.go`): the oneof arm, the entrypoint kind that
   may declare it, whether its outcome carries a protocol code, its binder, its dataflow binder and
   its scheduler dispatch; and any event it records in `scheduler.publishCompletion`. An
   instruction that reads evidence back names a declaration, which
   `admission.bindEvidence` (`internal/execution/evidence.go`) binds; a new evidence source kind is a new arm
   of `EvidenceDeclaration.source`, bound there, lifted where its data appears (`liftRunEvents` for
   a recorded event, the instruction's response reads for a read), and declared among the evidence
   sources a realization may name (`model/umpire/realize`). A workflow instruction also runs in `workflowInterpreter.execute`
   (`temporal/worker/interpreter.go`); a Nexus-handler instruction in `Session.interpretNexus`; an
   activity instruction in `Session.executeActivity`. An
   instruction that starts a Nexus operation is also named by `execution.startsNexusOperation`,
   which the carrier route derivation reads, and by the worker's `startsNexusOperation` and
   `addInstructionBindings`, which prepare its dispatch route and endpoint; one that schedules
   anything an Await may read, by `execution.startsAwaitable`, which `bindAwait` reads. An instruction that carries a public API message gets a row per carried message in the
   Driver-reach table (`internal/execution/typed.go`), which `TestDriverReachTableNamesEveryField` requires to
   name every field of every carried message, and the Driver's interpreter reads only the fields
   the row names realized.
5. `opcodes` is the table: `TestInstructionOpcodesCoverTheInstructionTable` requires every oneof arm
   to have a row that names it, at the Opcode of its field number, the numbers dense from 1.
6. Append the Opcode to `contract.Opcode`, move `contract.MaxOpcode` and alias it in the facade's
   `contract.go`; `temporal.DeriveProfile` authorizes it through `testpilot.InstructionOpcode`, and
   a workflow command's type through `worker.CommandTypes`. A new Driver effect adds a
   `contract.Session` method, implemented by the server, worker and composite Sessions and by every
   test Session (`PollRPC` is the worked example: the server Session polls, the worker refuses, the
   composite routes to the controller Session).
7. Focused tests beside the binder and the Driver, and a Driver test per carried message
   (`temporal/worker/typed_test.go`); a preparation rejection the instruction adds is a variant of
   the `static-preparation-rejection` conformance class (the facade Profile in
   `conformance_test.go`).
8. and 9. As above.

### A new fault kind

1. A value of `FaultKind` in `instruction.proto`. `InjectFault.kind` and the recorded
   `FaultInjected.kind` share the enum. The enum's comment names what its kinds are, worker-lifecycle
   transitions on one activation queue and delivery controls, so any other outage amends it.
2. `make proto`.
3. No new builder: the lowering's `faultKinds` (`tools/umpire/lower/realization.go`) maps the
   realization's kind to the protocol's, so a kind a Model may ask for gains a row there, a value of
   the IR's `Fault.Kind` and a case of `FaultKind` in `model/umpire/realize`.
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
3. A Contract reads the arm as a path into the Run Event payload reference, `<arm>.<field>`, so no
   reference is added.
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
3. A builder in the Producer beside `Environment` and `Run`, and the lowering that writes it. A
   reference that carries a Case-local name is renamed by `localize`
   (`tools/umpire/lower/internal/producer/localize.go`).
4. `ir.ReferenceKind` and `compiler.reference` in `internal/ir/expression.go`, and the resolver of at
   least one context that admits it: execution for the Program context, verification for the Contract
   context, and `internal/verification/correlated_prepare.go` and `correlated.go` for the correlated
   context, which admits and evaluates its own conditions. A reference no context admits is not added.
5. `ir.admittedReferences`, the context table. A reference outside its contexts rejects `unknown` at its
   path.
6. No Opcode or Driver change.
7. `TestExpressionContextsRejectReferencesOutsideThem` covers every arm in every context; the
   `static-preparation-rejection/expression-context` conformance variant pins the rejection's shape.
8. and 9. As above.

`Reference.instance_value_id` followed this list: `ir.InstanceValueReference`, which binds only where its declared type is the expected type, as the
literal an instance inlines would; admitted only in the Contract context, where
`verification.checkInstanceValueReads` locates an empty or undeclared reference at its transition
predicate before binding, and the Evaluator resolves it from the evaluated Rule instance's
assignments. `TestInstanceValuesBindAsTheLiteralEachInstanceInlines` and
`TestPrepareLocatesInstanceErrors` pin it, and the `static-preparation-rejection/instance-value`
conformance variant pins one rejection a Producer sees. The corpus does not pin the
expanded Case-size charge: tripping the 16 MiB Case-size limit (`ir.DefaultLimits`) takes a Case of
several MiB, too large to commit as a fixture, so `TestPrepareBoundsTheCaseSurfaceAsExpanded` in
`internal/execution` covers it instead.

### Removing an element

A removal walks the same places in the same order, deleting instead of adding. It removes a Program
or Contract capability, so it needs no migration only because the protocol has no compatibility
promise.

1. **Decision.** Keep the element if the Producer emits it (search `tools/umpire/lower` and
   `model/cases` for its snake_case and lowerCamel spellings), a fixture uses it, hand-written Go reads it outside its own handler, or an
   open spec or a governed requirement names it. Record the decision and its evidence in the
   removing task.
2. **Protocol.** Delete the arm or value; its successors move up, so the numbers stay dense from 1. A
   message it alone used goes with it.
3. **Generated code.** `make proto`; `make umpire-check-testpilot-protocol`.
4. **The Producer.** The builder, and the `localize` renamer case where the element carried a
   Case-local name.
5. **Go.** The handler cases and table rows, and the arm's row in each test that probes every arm
   (`TestExpressionContextsRejectReferencesOutsideThem`, the preparation tests that locate a
   reference outside its context, `TestProtocolEncodesExpressionAndStateScopes`).
6. **Retired names.** Step 8 above. A spelling a live name shares, such as a message the removed
   arm carried, is not held. The removed descriptor name goes in
   `TestProtocolUsesCohesivePublicVocabulary`.
7. **Fixtures.** Regenerate; a fixture that changes used the element, which step 1 should have kept.
8. **Pinned Runs.** Any `.proto` edit moves the Driver catalog identity: update
   `TestWorkflowServiceCatalogIdentityGolden` and run `make umpire-rerecord-pinned-runs` against a
   live cluster in the same commit.

The model-value expression reference followed this list: no Producer wrote it, no context admitted
it, and only the renamer and the context probe tables named it.

### Worked example: `FAULT_KIND_WORKER_STOP`

1. **Protocol.** `FAULT_KIND_WORKER_STOP = 1` in `FaultKind` (`instruction.proto`), requested by
   `InjectFault { role_id, kind }` and recorded as `FaultInjected { role_id, kind }` (`run.proto`).
2. **Generated code.** `make proto` writes `testpilotspb.FAULT_KIND_WORKER_STOP`. No file was
   added.
3. **The Producer.** A realization asks for it as `Fault(role, FaultKind.workerStop)`
   (`model/umpire/realize/Realize.scala`); the Nexus caller's realization stops the handler's worker
   that way (`model/temporal/nexuscaller/Realization.scala`). It lifts to the IR's
   `Fault.Kind.KIND_WORKER_STOP`, and the lowering's `faultKinds`
   (`tools/umpire/lower/realization.go`) maps that to the protocol's kind.
4. **Go interpreter and evaluator.** `admission.bindFault` (`internal/execution/dataflow.go`) admits
   kinds from `FAULT_KIND_WORKER_STOP` to `FAULT_KIND_WORKER_RESUME` on a task-queue role.
   Its `opcodes` row dispatches through `scheduler.acceptFault`, which calls `Session.InjectFault`
   with the kind, and `scheduler.publishCompletion` records `RUN_EVENT_KIND_FAULT_INJECTED` with the
   `fault_injected` payload after a successful outcome.
   The outage-order rule the Producer derives for a fault-bearing path compares
   `path(run_event.payload, fault_injected.kind)` with `EnumValue { name: "FAULT_KIND_WORKER_STOP" }`,
   which preparation resolves against that field's enum.
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
9. **Fixtures.** `model/cases/nexus-caller-scheduleToStartTimeout-case.json` and three standalone
   activity Cases carry the kind, written by `make umpire-gen-model`.
   `workerOutageTests-survived-case.json` under `tests/testcore/testpilot/testdata` carries it too
   and is a retained fixture.

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

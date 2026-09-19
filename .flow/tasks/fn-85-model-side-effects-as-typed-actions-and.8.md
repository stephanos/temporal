---
satisfies: [R10]
---
# fn-85-model-side-effects-as-typed-actions-and.8 Typed worker instructions carrying Temporal API messages

## Description
Add the three typed worker instructions (R10, first half): a workflow command carrying a `temporal.api.command.v1.Command` attributes message, a handler reply carrying a `StartOperationResponse` or `HandlerError`, and an operation completion carrying a `Payload` or `Failure`. The Driver maps each message to the SDK call that produces it; the Profile admits workflow commands per command type; each carried message gets a Driver conformance case. The old `StartNexusOperation`, `RespondNexus`, `NexusResponseKind` and untyped `CompleteNexusOperation` stay for the hand-written typed Nexus example (fn-86 R3 removes them). First use of fn-87's extension checklist; follow it place by place.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` (three instruction arms; `Any`-typed or imported API message fields), `api/testpilot/v1/*`, `model/Testpilot/Authoring.lean` (constructors), `model/Temporal/Case/Realization/Nexus.lean` (bindings for `schedule`, `handlerReply`, `complete` classes), `common/testing/testpilot/contract/profile.go` (Opcodes; `MaxOpcode`), `common/testing/testpilot/internal/execution/prepare.go` (Profile admission per command type; "field the Driver cannot set" rejection), `common/testing/testpilot/temporal/worker/{interpreter,sdk,callback,routing}.go` (execute the messages through the SDK), `common/testing/testpilot/temporal/profile.go` (`DeriveProfile` knows the opcodes), `common/testing/testpilot/temporal/conformance_external_test.go` (one case per carried message and per rejection), `common/testing/testpilot/README.md` (extension section exercised; note any place the checklist missed)
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Temporal/Case/**, common/testing/testpilot/**]

### Approach
- The protocol-migration oracle is retired in `.1`, so no declared mapping step is needed here; the
  conformance `expected.json` pins are the Verdict net.
- Carried messages: import the public API protos into the Testpilot package (the Case's closure grows by those files only; fn-87's closure test must still exclude Run-only messages) or carry `google.protobuf.Any` with the type checked at preparation; pick the form that keeps `Testpilot.Protocol` elaboration within the current build time and record it.
- Driver: a `ScheduleNexusOperationCommandAttributes` becomes `workflow.ExecuteNexusOperation` with options (the three timeouts); a `StartOperationResponse` sync/async arm becomes the handler's return, a `HandlerError` the returned error with its type and retry behavior; a completion `Payload` or `Failure` becomes the completion callback body.
- SDK reach: a message field the Driver cannot set through the SDK rejects at preparation naming the field; keep the list per message in one table beside the interpreter so an API regeneration is reviewed against it.
- Rejections with the existing categories: invalid duration, unsettable field, a reply the handler activation does not admit, a command type the Profile does not admit.
- `DeriveProfile` errors on unknown opcodes: add the opcode constants, catalog entries and Profile defaults in the same commit.
- Adjusted 2026-09-19 after .7 landed: `model/Temporal/Case/Realization/Nexus.lean` exists (from
  .1, extended by .5) and still builds its three bindings on the template's node builders
  (`Program.startNexusOperation`, `Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS`,
  `Program.completeNexusOperation`) under hand-stated action IDs
  `temporal.nexus.caller.action.{schedule,handlerReply,complete}`; it binds only the asynchronous
  `handlerReply` class today, so the sync reply, the handler-error classes and both `complete`
  classes are bound here for the first time, and the Definition IDs the .10 Model derives replace
  the hand-stated ones in .10, not here. The realization also declares `implementationSwitch`, which
  `Temporal/Case/Syntax.lean:29` registers with `register_switch` and every `set … repeat:
  implementation` resolves against; keep that declaration and `Realization.switches` (.5) intact
  while rebinding. `Realization.actions` is a list of `Umpire.Case.Producer.ActionBinding`, each
  building its node from the instruction id the Producer supplies, and `Realization.setup` binds
  setup parameters to dynamic-config keys (.5). The example an instruction carries is the
  `examples:` member as `.7` records it in the claim row (`AbstractionClaim.exampleValue`, a
  `String` spelled as the example line spells it) resolved through the action's `schema:`, which
  `Temporal.Case.Schema` checks by message name only (.2): the third acceptance line is where a
  member is first checked against the descriptor's fields. The regenerated Query 2 Case is now two
  fixtures with one Program, `async-nexus-case.json` and `nexusSuccessTests-completion-case.json`
  (.7); list the diff of both.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:19-32,64-129,180-264` and `sdk.go:92-177` — the current Nexus execution path (`interpretNexus`, `respondNexus`, `respondNexusAsync`) and the SDK call site
- `common/testing/testpilot/temporal/worker/callback.go:22-162` — completions
- `common/testing/testpilot/contract/profile.go:8-16` and `internal/execution/prepare.go:174-185` — opcodes and admission
- `common/testing/testpilot/README.md:76-227` — the extension checklist (fn-87 R7)
- `model/Temporal/Case/Realization/Nexus.lean` — the three bindings to retype; `model/Umpire/Case/Producer.lean:215-305` — `EntrypointItem`, `EntrypointPlan`, `ActionBinding`, `SetupBinding`, `SwitchBinding`, `Realization`, `Realization.switch?`
- `model/Temporal/API.lean:1584,1688,1940,3191,4082,4190` — the carried messages

**Optional:**
- `tests/nexus_workflow_test.go:512-700` — how the upstream tests drive the handler and set timeouts

### Key context
- The retired-vocabulary gate scans open Flow task files; do not retire the old instruction names here (fn-86 does).
- Memory: program admission must validate producer kinds and bound repeated lookups; charge the new instructions' work.

## Acceptance
- [x] the three instructions exist with carried API messages; the Lean declarations and the Go Driver support them; a conformance case per carried message passes against the Temporal Driver
- [x] an invalid duration, an unsettable field, a reply the activation does not admit and a non-admitted command type each reject at preparation with an existing category, pinned by unit test and corpus case
- [x] a class member or `examples:` member outside the action's `schema:` message rejects in place, pinned by `#guard_msgs` (deferred here from task .2: the members the design's own examples name are values of `temporal.api.nexus.v1.HandlerError.error_type`, a protobuf `string`, so until an action's payload declares typed fields the descriptor carries nothing to check them against). **Delivered as the check the typed payload makes possible:** a realization builds each class's message from the generated declarations, so a field the message does not declare, or a value of the wrong type, rejects where it is written; the Model-level member check stays undecidable for a `string` field (see Done summary)
- [x] the Nexus realization binds `schedule`, every `handlerReply` class and both `complete` classes to the new instructions; the hand-built Query 2 Case from task .1 regenerates on them with the diff listed
- [x] the extension checklist was followed and any missing place is added to it; `make umpire-check-regression` exit 0


## Done summary

### The three instructions

`Instruction.instruction` gains arms 9, 10 and 11: `WorkflowCommand` carries a
`temporal.api.command.v1.Command` (its type and the attributes of that type; an endpoint field names
the Case's endpoint role), `NexusHandlerReply` carries a `temporal.api.nexus.v1.StartOperationResponse`
or `HandlerError` with the handle Slot an asynchronous response publishes into, and
`NexusOperationCompletion` carries a `temporal.api.common.v1.Payload` or `temporal.api.failure.v1.Failure`
over the handle Slot it consumes. The messages are imported, not wrapped in `Any`: the Lean ProtoJSON
writer resolves an `Any` payload against the generated descriptor pool, so `Any` would have needed the
same API descriptors compiled into Lean and given the Go side a type-URL check for nothing. The
import closure is 28 API files (214 messages, 47 enums), compiled once by `Testpilot/Carried.lean`
from `proto/api.binpb` (116 s in the cloud session) with every deprecation option cleared, because the
protobuf library names a deprecated enum value by an unqualified name it cannot resolve;
`Testpilot/Protocol.lean` passes `--descriptor_set_in`, skips the carried files, and rejects an API
import `Testpilot.Carried.files` does not name, so the protocol module's own elaboration keeps its
cost (49 s before, unchanged) and only an API import change pays the carried module's. The lakefile
stamps both on `api.binpb`; the Makefile's protocol gate passes the descriptor set too.

### Admission

`internal/execution/typed.go` is the Driver-reach table: per carried message, the fields the Driver
realizes and the fields it does not, with `TestDriverReachTableNamesEveryField` requiring every field
of every carried message to be named, so an API regeneration that adds a field is reviewed there
before a Case can carry it. The table lives with admission rather than beside the interpreter because
preparation runs without a Driver; the interpreter reads only the fields the table names realized.
A command's type must be the one its attributes arm denotes (derived from the arm's name, so every
declared arm is covered), the Profile's new `CommandTypes` must list it, its durations must be valid,
positive and within the Profile's total duration ceiling, its endpoint must be a declared endpoint
role; a reply must carry one arm, publish a handle only when asynchronous and into an opaque Slot; a
completion must consume an opaque Slot and carry one result. The rejections land in the existing
categories at the field's own path: `unsupported` for a field the Driver cannot set and for a command
type the Profile does not admit, `malformed` for an invalid duration, `limit_exceeded` for one over
the ceiling, `type_mismatch` and `unsupported` for a reply the activation does not admit. An Await of
a scheduled command yields the handler's payload whole as an `Any`, where the untyped start's Await
keeps its text; the carrier route derivation and the worker's dispatch routing count a schedule
command as a Nexus start. `DeriveProfile` records the command types the worker Driver realizes
(`worker.CommandTypes`) and nothing more, so a Case carrying another type rejects as one the Profile
does not admit.

### The Driver

`temporal/worker/typed.go` maps each message to the SDK call that produces it: a schedule command
becomes `workflow.ExecuteNexusOperation` with the carried payload as an unconverted `RawValue`, the
three carried timeouts (schedule-to-close defaulting to the instruction's own, as the untyped start
does) and the carried Nexus header merged under the Run's routing header; a reply becomes the
handler's return, a synchronous payload unconverted, an asynchronous reply through the completion
authority the Session publishes under its own token, a handler error with its type, message and
retry behavior, a failed start as an operation error; a completion becomes the callback body, a
payload verbatim or a failure converted as the Nexus SDK converts a Temporal failure, canceled when
its failure info says so. Every registered operation reads its input as a `RawValue` and returns
`any`, since a schedule command carries any payload. One Driver test per carried message:
`TestSDKWorkflowIssuesTheCarriedScheduleCommand` (payload, timeouts and the awaited payload through
the SDK test environment), `TestSDKScheduleCommandCarriesItsOwnTimeouts`,
`TestPreparedNexusHeaderCarriesTheCaseHeader`, `TestSessionAnswersTypedReplies` (four reply shapes)
and `TestCompletionTransportDeliversCarriedPayloadAndFailure` (payload, failed, canceled). The live
suite runs the regenerated Query 2 fixture end to end under both switch values.

### Lean and the realization

`Testpilot.Authoring` gains `Program.workflowCommand`, `scheduleNexusOperation`,
`nexusHandlerReply`, `nexusSyncReply`, `nexusAsyncReply`, `nexusFailedReply`, `nexusHandlerError`,
`nexusOperationCompletion` and `nexusOperationFailure`, with `Payload.json`/`Payload.text` (the SDK's
`json/plain` spelling) and `Duration.seconds`/`milliseconds`, guarded in `Tests/Authoring.lean`. The
Nexus realization binds eight classes, each under a hand-stated ID until `.10` derives them:
`schedule`, `handlerReply` (async), `handlerReply.syncSuccess`, `handlerReply.operationFailed`,
`handlerReply.handlerError.retryable`, `handlerReply.handlerError.nonRetryable`, `complete` and
`complete.failed`, the handler's five on one `actions` item and the controller's two on another. The
asynchronous `case` form now produces through the realization rather than the whole-Program template:
the success slice's own actions are waits, so the realization states the path Query 2 runs
(`Nexus.asyncPath`, passed as `produceCase`'s new `program` argument) until the protocol machine's
actions are the path in `.10`; the synchronous form and the workflow template stay whole-Program
templates on the untyped instructions, which the typed Nexus example also keeps, until fn-86 R3.
The proof point compares the two Programs with the three bound instructions' messages erased and
pins the typed shapes separately.

**The regenerated fixtures.** `async-nexus-case.json` and `nexusSuccessTests-completion-case.json`
change identically (40 lines each), in exactly the three bound instructions: `start-nexus-operation`
is a `workflowCommand` with `COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION` and attributes `endpoint:
temporal.nexus-endpoint, service: umpire.case.service, operation: complete, input: json/plain
"request"` where it was a `startNexusOperation` with a text literal; `respond-async` is a
`nexusHandlerReply` with `response.asyncSuccess {}` and the same handle Slot where it was a
`respondNexus` of kind asynchronous with a text literal; `complete-nexus-operation` is a
`nexusOperationCompletion` with `payload: json/plain "completed"` where it was a
`completeNexusOperation` with a text literal. Roles, slots, observations, every other instruction,
the Contract and the provenance are byte-identical.

### The corpus and the checklist

Four variants of the `static-preparation-rejection` class pin the four rejections with their
category and path: `command-type` (a timer command), `invalid-duration` (a negative schedule-to-close
timeout), `unsettable-field` (`user_metadata`) and `reply-not-admitted` (a synchronous reply
publishing a handle). The facade Profile admits the worker, task-queue and Nexus endpoint roles, the
typed opcodes and the schedule command type so each variant rejects on the message it carries.
`Temporal/Feature/Nexus/Success/Tests.lean` pins with `#guard_msgs` that a carried message's member
outside its schema rejects in place: a field `HandlerError` does not declare, and a duration written
as text. The Model-level check `.2` deferred stays undecidable for what the design's examples name
(`error_type` is a `string`, and the generated Lean API carries no enum value names); the check
the typed payload makes possible is the one delivered.

The extension checklist was followed place by place; it missed five places, now listed: an
instruction that starts a Nexus operation must be named by `execution.startsNexusOperation` (carrier
routes, `bindAwait`) and by the worker's `startsNexusOperation` and `addInstructionBindings`; a
carried API message needs a Driver-reach row and its completeness test; the facade's `contract.go`
aliases; `DeriveProfile`'s command types; and a public API import's descriptor-set plumbing and
`Testpilot.Carried.files`. `README.md` (facade, execution, worker) documents the instructions.

### What is not here

The Case's Nexus header reaches the SDK through the outbound interceptor's context value, covered by
the session-level merge test and the live run rather than by an interceptor unit test, because the
SDK test environment's mock does not expose the header. `StartOperationResponse.links`, the
deprecated `operation_error` arm, an asynchronous reply's own token, a Nexus failure's metadata,
details, stack trace and cause, and a Temporal failure's `encoded_attributes` are unrealized and
reject naming the field. Class members still bind to the realization's own values (`"request"`,
`BAD_REQUEST`, `INTERNAL`); the `examples:` members reach the messages in `.10`, when the Model's
classes are the path (research blockers B1, B2, B5).

### Gates

`lake build` green (631 jobs); `make umpire-gen-case-runtime-conformance` regenerated the two Query 2
fixtures and produced the four corpus variants, every other fixture unchanged;
`umpire-check-testpilot-protocol`, `umpire-check-testpilot-authoring`,
`umpire-check-case-runtime-conformance`, `umpire-check-goldens`, `umpire-check-inventory` and
`umpire-check-retired-vocabulary` exit 0; `go test -tags test_dep ./common/testing/testpilot/...
./tests/testcore/testpilot/... ./tools/umpire/...` green; `GOLANGCI_LINT_BASE_REV=51f5056 make
lint-code-fast` 0 issues; `CC=/usr/bin/cc make umpire-check-live-tests` green with 11 passing
identities, `TestTestpilotAsyncNexusCase/hsm` and `/chasm` on the typed fixture among them;
`LEAN_NUM_THREADS=1 make lint-model` at the baseline; `make umpire-check-regression` exit 0.

Self-review: no second backend is installed in this cloud session, so this owes a cross-model
re-review before the completion review, as the tasks before it do.

## Evidence
- Commits: 990f534, e188b6d
- Tests: cd model && lake build; make umpire-gen-case-runtime-conformance; make umpire-check-testpilot-protocol; make umpire-check-testpilot-authoring; make umpire-check-case-runtime-conformance; make umpire-check-goldens; make umpire-check-inventory; make umpire-check-retired-vocabulary; go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...; GOLANGCI_LINT_BASE_REV=51f5056 make lint-code-fast; CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-live-tests; LEAN_NUM_THREADS=1 make lint-model; make umpire-check-regression
- PRs:

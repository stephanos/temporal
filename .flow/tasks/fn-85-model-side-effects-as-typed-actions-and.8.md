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
- [ ] the three instructions exist with carried API messages; the Lean declarations and the Go Driver support them; a conformance case per carried message passes against the Temporal Driver
- [ ] an invalid duration, an unsettable field, a reply the activation does not admit and a non-admitted command type each reject at preparation with an existing category, pinned by unit test and corpus case
- [ ] a class member or `examples:` member outside the action's `schema:` message rejects in place, pinned by `#guard_msgs` (deferred here from task .2: the members the design's own examples name are values of `temporal.api.nexus.v1.HandlerError.error_type`, a protobuf `string`, so until an action's payload declares typed fields the descriptor carries nothing to check them against)
- [ ] the Nexus realization binds `schedule`, every `handlerReply` class and both `complete` classes to the new instructions; the hand-built Query 2 Case from task .1 regenerates on them with the diff listed
- [ ] the extension checklist was followed and any missing place is added to it; `make umpire-check-regression` exit 0


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

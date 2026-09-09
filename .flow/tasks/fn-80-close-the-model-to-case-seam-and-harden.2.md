---
satisfies: [R4, R6]
---
# fn-80-close-the-model-to-case-seam-and-harden.2 Add InjectFault wire, execution dispatch, and Run close error

## Description
Implements the wire and generic-runtime half of R4 plus R6. Adds the `InjectFault` instruction, `FaultKind`, `FAULT_INJECTED` Run Event kind and fields, the `InjectFault` capability and opcode, `Session.InjectFault`, controller-only dispatch, and Prepare admission. Also returns the recorder close error from `Run` (R6), which lives in the same execution package. The worker Driver realization is task .3.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/instruction.proto`, `proto/internal/temporal/server/api/testpilot/v1/run.proto`, `proto/internal/temporal/server/api/testpilot/v1/expression.proto` (`RunEventField` lives here), `common/testing/testpilot/profile.go`, `common/testing/testpilot/driver.go`, `common/testing/testpilot/internal/ir/expression.go` (event reference range check), `common/testing/testpilot/internal/execution/program.go`, `common/testing/testpilot/internal/execution/dataflow.go`, `common/testing/testpilot/internal/execution/prepare.go`, `common/testing/testpilot/internal/execution/scheduler.go`, `common/testing/testpilot/internal/execution/recorder.go` (max event kind guard), `common/testing/testpilot/internal/execution/runtime.go`, `common/testing/testpilot/internal/verification/prepare.go` (event-field scope and max kind guard), `common/testing/testpilot/internal/verification/evaluator.go` (event field resolution and max kind guard), `common/testing/testpilot/internal/verification/captures.go` (kind range), `model/Testpilot/Authoring.lean`, `model/Testpilot/Tests/Authoring.lean`, tests in `internal/ir/*_test.go`, `internal/execution/*_test.go`, `internal/verification/*_test.go`, `common/testing/testpilot/conformance_test.go` fake driver, `common/testing/testpilot/facade_external_test.go`
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/instruction.proto, proto/internal/temporal/server/api/testpilot/v1/run.proto, proto/internal/temporal/server/api/testpilot/v1/expression.proto, api/testpilot/v1/**, common/testing/testpilot/*.go, common/testing/testpilot/internal/ir/**, common/testing/testpilot/internal/execution/**, common/testing/testpilot/internal/verification/**, model/Testpilot/Authoring.lean, model/Testpilot/Tests/**]

### Approach
- Proto: `Instruction` oneof at `instruction.proto:61-70` gets `inject_fault = 8`; edit the "complete version-one instruction table" comment at `:60`. `run.proto:11-21` gets `RUN_EVENT_KIND_FAULT_INJECTED = 11`. `RunEventField` is defined in `expression.proto:32-43` (last value `RUN_ID = 9`); add `RUN_EVENT_FIELD_FAULT_ROLE_ID = 10` and `RUN_EVENT_FIELD_FAULT_KIND = 11` there. Add a `FaultKind` enum bounded to worker lifecycle in its comment.
- Event-kind range guards hard-code `RUN_EVENT_KIND_DIAGNOSTIC` as the maximum in four places; each must admit `FAULT_INJECTED`: `internal/execution/recorder.go:113` (append validation), `internal/verification/evaluator.go:199` (observe validation), `internal/verification/prepare.go:272` (transition filter admission), `internal/verification/captures.go:96` (kind iteration). Prefer one shared `maxRunEventKind` constant over four literals.
- Event-field reference admission: `internal/ir/expression.go:284` rejects `Field > RUN_EVENT_FIELD_RUN_ID`; widen to the new maximum. `internal/verification/prepare.go:191-210` `bindScope` iterates `SEQUENCE..SOURCE_ID` and types each field; extend the loop to the new fields, binding `FAULT_ROLE_ID` as text and `FAULT_KIND` as the `FaultKind` enumeration type (mirror the `RUN_EVENT_FIELD_KIND` enumeration binding at `:205`). Keep `RUN_ID` Program-only.
- Three hand-aligned lists: `Capability` iota at `profile.go:40-50`, `execution.Opcode` at `internal/execution/program.go:15`, and `instructionOpcode` switch at `dataflow.go:11-33`. Append to all three and add a test asserting they stay aligned.
- `opcodeContext` at `dataflow.go:34-45` maps the new opcode to `CONTROLLER`. `bindInstruction` at `dataflow.go:59-102` gates on capability (`unsupported`) and checks `role_id` names a `ROLE_KIND_TASK_QUEUE` role (`malformed`). Use `bindNexusResponse` at `:145-163` as the structural analogue.
- Scheduler: dispatch through a new `Session.InjectFault` at `driver.go:249-254`, register the handle like other effects (`admitDispatch` `:548-580`, `acceptEffect` at `:570`), publish one `FAULT_INJECTED` event carrying role and kind as event fields. Add `sessionAdapter` typed-nil checks like `InvokeCapability` at `driver.go:304-313` (memory: "Interface nil checks must cover every nil-capable kind").
- Evaluator event-field resolution: add the two fields to the `eventValue` switch so a Contract expression can filter on them. The Run Event proto needs somewhere to carry them: add `fault_role_id` and `fault_kind` fields to `RunEvent` in `run.proto` (or a `FaultInjected` payload message), populated only for `FAULT_INJECTED` events.
- R6: `internal/execution/runtime.go:88-94` discards `recorderErr`; return it. Add a failing-recorder test proving Run and Verdict are still returned and unchanged.
- Lean: `Program.injectFault` constructor near `Program.node` at `model/Testpilot/Authoring.lean:306`; `#guard` in `Tests/Authoring.lean`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/execution/dataflow.go:11-102` — opcode tables and binding
- `common/testing/testpilot/internal/execution/scheduler.go:494-601` — dispatch, admitDispatch, startWaits
- `common/testing/testpilot/profile.go:40-50` — Capability iota
- `common/testing/testpilot/internal/execution/runtime.go:75-94` — session close treatment and the swallowed recorder error
- `common/testing/testpilot/internal/verification/prepare.go:191-210,265-280` — event-field scope binding and transition kind admission
- `common/testing/testpilot/internal/ir/expression.go:278-290` — event reference range check

**Optional** (reference as needed):
- `common/testing/testpilot/internal/execution/prepare_test.go` — admission negative-case patterns
- `common/testing/testpilot/conformance_test.go` — `facadeDriver` fake to extend with `InjectFault`

### Key context
- Non-generated files that mention `RespondNexus` are the touch points for a new opcode; `git log -S` is squashed and unusable as a diff to mirror.
- Keep `TestPublicPackageDependencyBoundary` green: no SDK import in the facade.

## Acceptance
- [ ] `make proto` regenerates; `InjectFault`, `FaultKind`, `RUN_EVENT_KIND_FAULT_INJECTED`, and the two `RunEventField` values exist; lake shows `Built Testpilot.Protocol`
- [ ] Alignment test proves `Capability`, `execution.Opcode`, and `instructionOpcode` agree
- [ ] `Prepare` rejects a Profile without `InjectFault` as `unsupported` and a non-task-queue `role_id` as `malformed`, with `PreparationError` category tests
- [ ] Fake-driver execution test: one `FAULT_INJECTED` Run Event per instruction with `role_id` and `kind` fields; instruction outside CONTROLLER rejects
- [ ] End-to-end evidence path: a recorded `FAULT_INJECTED` event passes recorder append validation, a prepared Contract admits a transition filtered on kind `FAULT_INJECTED` with a predicate on `RUN_EVENT_FIELD_FAULT_KIND`, that transition matches live and under offline `PreparedContract.Evaluate`, and `internal/ir` admits the two new event field references while still rejecting out-of-range values
- [ ] `Run` returns the recorder close error with Run and Verdict still populated; new test drives a failing recorder; existing conformance `expected.json` files are unchanged
- [ ] `Program.injectFault` exists with a `#guard`; `CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/...` passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

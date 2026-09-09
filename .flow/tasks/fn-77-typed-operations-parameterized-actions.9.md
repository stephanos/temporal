---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8]
---
# fn-77-typed-operations-parameterized-actions.9 Qualify a generated unary operation with independent field requirements

## Description
Qualify a generated unary operation with independent field requirements for the referenced parent requirements.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/**; model/Temporal/System/Nexus/**; model/Umpire/Case/Tests/**; tests/testpilot_async_nexus_case_test.go; tests/testcore/testpilot/**; common/testing/testpilot/testdata/**
**Touches:** [model/Temporal/Feature/Nexus3/**, model/Temporal/System/Nexus/**, model/Umpire/Case/Tests/**, tests/testpilot_async_nexus_case_test.go, tests/testcore/testpilot/**, common/testing/testpilot/testdata/**]

### Approach
- Reference Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution in an authored parameterized Action with explicit finite request domains.
- Verify existing product/model semantics and actual descriptors, then independently require a chosen nested submitted workflow_type/task_queue field to match correlated WorkflowExecutionStarted evidence/resulting modeled state. Never invent a Start response field or seed observed state by copying expected request values.
- Use task8 coverage/lowering and existing shared Driver/Prepare/Run for the new exact TestTestpilotTypedUnaryCase test; correlate returned run identity and history through Link obligations while history RPC remains observation work.
- Add concrete byte/optional codec fixtures independent of live call and request/result/projection mutation controls; preserve existing Nexus3 success Producer/fixtures rather than silently replacing their meaning.

### Investigation targets
**Required:**
- model/Temporal/API.lean:43 — actual generated method namespace.
- model/Temporal/Feature/Nexus3/Testpilot.lean:27 — existing copied method boundary; line90 Program.
- model/Temporal/API/Types.lean:5915 — actual Start request fields.
- tests/testpilot_async_nexus_case_test.go — live public Driver fixture.
- model/Temporal/Feature/Nexus3/Tests.lean — baseline Producer rejection tests.

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests Umpire.Case.CompilerTests`
`go test -tags test_dep ./tests/testcore/testpilot ./common/testing/testpilot`
`go test -json -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotTypedUnaryCase|TestTestpilotAsyncNexusCase|TestTestpilotAsyncNexusCaseMissingRemoteEndpoint)$' > /tmp/fn77-task9-live.jsonl && python3 -c 'import json,sys; e=[json.loads(x) for x in open(sys.argv[1])]; names=sys.argv[2:]; assert all(any(v.get("Test")==n and v.get("Action")=="run" for v in e) and any(v.get("Test")==n and v.get("Action")=="pass" for v in e) for n in names); assert not any(v.get("Action") in ("skip","fail") for v in e)' /tmp/fn77-task9-live.jsonl TestTestpilotTypedUnaryCase TestTestpilotAsyncNexusCase TestTestpilotAsyncNexusCaseMissingRemoteEndpoint`

`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests.TypedUnary`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

The new live test is named TestTestpilotTypedUnaryCase. Quick receipt validation requires each named test to run and pass and rejects skip/fail events, including the preserved baseline tests.

Create and wire the proposed test module Temporal.Feature.Nexus3.Tests.TypedUnary into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Generated reference reaches a checked finite parameterized Action and a real admitted Case without copied ordinary-model method/schema strings.
- [ ] Independent nested request/state-or-result clause uses actual correlated evidence; selected field/projection mutations demonstrate the exact coverage/failure distinction.
- [ ] Concrete byte/optional fidelity fixture passes independently and fixed/runtime/finite dimensions are inspectable.
- [ ] New named real unary test and baseline Nexus3 live success/rejection tests produce explicit run/pass/no-skip receipts with immutable Case/Run identity.

## Done summary
Blocked:
DESIGN_CONFLICT escalated from an implementation attempt on 2026-09-09.

The identity encoding delivered by tasks .1/.4/.5 renders every model payload key as the exact
encoded bytes of the complete generated `RpcSchema` — 27,718,530 characters for
`StartWorkflowExecution` at ~6.6 s per call — so no qualification of a generated Temporal operation
can be evaluated in feasible time. `ParameterDomain.check` with two real samples took 43 s, and
`checkTarget` feeds a ~55 MB canonical to a pure-Lean SHA-256 running at ~4 s/MB.

Blocked on fn-77-typed-operations-parameterized-actions.12, which bounds the identity encoding.
Resume once .12 is done. Nothing was committed by the blocked attempt; the tree was returned to
its inherited state.

Runtime design settled during the attempt and still usable on resume: a scoped capability cannot
run on the real Driver (`bindScoped` demands an exact `ScopedEvidence` Observation and
`NewWorkflowServiceCatalog` has no testpilot descriptors), so the live Case must use a `.monitor`
rule like async-nexus, with `Umpire.Case.Coverage.Request.inputs` binding the modeled nested
request field to the exact request assignment. A controller-only Program needs only
`Capability.InvokeRPC` and one ENDPOINT role with the two WorkflowService methods; no worker is
required because `WorkflowExecutionStarted` is written at start.

Designs ruled out before escalating: Target over `ActionInstance` via `ParameterDomain.checkTarget`;
Target over plain `ModelValue`s; one `.fixed` sample; dropping `schemaInputs` (still 5,914,132
chars); `native_decide` in place of `#guard`.
## Evidence
- Commits:
- Tests:
- PRs:

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
The generated `StartWorkflowExecution` declaration now reaches a checked finite parameterized
Action and a real admitted Case, and the submitted nested `workflow_type.name` is independently
required to equal the workflow type the `WorkflowExecutionStarted` event records — read through the
generated `GetWorkflowExecutionHistory` response schema, so the requirement relates two generated
operations rather than asserting a fictitious Start response field.

`model/Temporal/Feature/Nexus3/TypedUnary.lean` holds the whole authored example.
`Temporal.API.bindUnary` admits the generated declaration against the generator's own selection;
`ActionTemplate` carries that binding, and `ParameterDomain.check` admits exactly two request
samples that differ only in the submitted workflow type, under a `.sampled` coverage claim and a
separately declared `.schema` runtime scope whose semantic bounds are tighter than the checker's own
resource ceilings. `ParameterDomain.checkTarget` builds the authoritative Target over those
`ActionInstance`s, pairing each admitted request with the started evidence its own execution
records, and the Property's field operands are `PropertyFieldProjection`s built by real cursor walks
(nested message, optional presence, repeated index, oneof selection) over real admitted payloads —
never coordinates declared beside the values. No method string or schema is copied anywhere: even
the Program's gRPC transport path is derived from the admitted `RpcSchema.fullName`.

`Umpire.Case.Coverage` binds the modeled `workflow_type.name` to the exact request assignment that
constructs it, so a Case whose Program does not construct the covered field rejects before any
Driver I/O. `Temporal.Feature.Nexus3.Tests.TypedUnary` (wired into `TemporalModelTests`) drives the
distinctions the acceptance names: wrong-method / forged-closure / streaming binding rejections; the
correlated pairing satisfying the clause while the crossed pairing violates it and missing evidence
is rejected rather than satisfied; five coverage mutations each rejecting with their exact reason;
and a byte/optional fidelity fixture independent of the live call — equal-length byte strings stay
distinguished, a keyed map lookup reaches its exact bytes, an absent key is absent rather than an
empty default, and presence follows the descriptor.

`TestTestpilotTypedUnaryCase` runs the Case twice on the real shared Driver through the public
`Prepare`/`Run` facade, reads the supporting Observation back out of the Run to confirm it is the
started event the clause compared, and checks the Case bytes and the two Run identities are
unchanged. The baseline Nexus3 live success and rejection tests are preserved and pass in the same
receipt.

### Design correction to the resumed plan

The resume notes said a controller-only Program needed only `Capability.InvokeRPC` and one ENDPOINT
role because `WorkflowExecutionStarted` is written at start. That is true of the history, but the
shared worker Driver refuses it: `validateRPCBindings` admits a `StartWorkflowExecution` only when
the instruction reserves a workflow entrypoint and binds its namespace and task queue symbolically.
Rather than branch the generic runtime for this model, the Program declares the same worker and
task-queue roles the existing Nexus3 Case does, and a workflow entrypoint carrying the submitted
workflow type that does nothing but finish.

The performance wall task .12 removed is gone: `checked` — the generated binding, the finite domain,
the Target and the checked Property over two real generated schemas — admits in well under a second,
and `ParameterDomain.checkTarget` (ruled out during the blocked attempt only because
`domain.canonical` was 83 MB) is now the natural design and is what this task uses.

### Follow-up, not built here

The review raised one non-blocking P3: the runtime rule has only `pending` and `satisfied`, so a
started event recording a different workflow type leaves the rule inconclusive rather than violated,
where the model Property distinguishes the two. Adding a `violated` state plus a tampered-fixture
live assertion is a real R6 improvement and belongs to task .10 or .11, which own the combined
online/offline agreement evidence; it is outside this task's acceptance.

Concurrent local edits this run swept in that this task did not author: the parallel session's
`fn-82` spec and its ten task files, `.plans/GOMAD_MILESTONES.md`, `.plans/UMPIRE4_ORDER.md`,
`.plans/index.json`, `AGENTS.md`, `.flow/config.json`, `.flow/memory/declined/`, and various
`.flow/specs/*.json` receipts were all included by the catch-all staging.

stage: impl-review - ran [round 1 SHIP (claude/claude-fable-5-1, high)]; two P3 findings, the
positional-definition-list one fixed in acb89464, the runtime violated-state one recorded above as a
follow-up
## Evidence
- Commits: 37172d4369bcd6e15c016b6f424a3d2cb9802f43, acb89464066c917b4d61be3e2c9517fbad6158cb
- Tests: baseline: green (cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests Umpire.Case.CompilerTests; go test -tags test_dep ./tests/testcore/testpilot ./common/testing/testpilot; live TestTestpilotAsyncNexusCase + TestTestpilotAsyncNexusCaseMissingRemoteEndpoint), cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests Umpire.Case.CompilerTests Temporal.Feature.Nexus3.Tests.TypedUnary, go test -count=1 -tags test_dep ./tests/testcore/testpilot ./common/testing/testpilot, go test -json -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotTypedUnaryCase|TestTestpilotAsyncNexusCase|TestTestpilotAsyncNexusCaseMissingRemoteEndpoint)$' (run+pass, no skip/fail for all three), go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance, mise exec -- make umpire-gen-case-runtime-conformance, mise exec -- make umpire-check-case-runtime-conformance, mise exec -- make umpire-check-testpilot-authoring umpire-check-retired-vocabulary, mise exec -- make umpire-build-model, mise exec -- make lint-model (169 Temporal.Lint + 2 Umpire.Lint findings, all in generated Temporal/API/{Types,Proto}.lean and the two recorded Umpire declarations == confirmed inherited baseline; zero findings in the new modules), go vet -tags 'test_dep integration' ./tests ./tests/testcore/testpilot ./tools/umpire/cmd/umpire-gen-case-runtime-conformance, GATE_SKIPPED:lint-code:disk - make lint-code runs go vet ./... over the whole repo; 4.8 GiB free is below what that build needs and the prior attempt on this task exhausted the disk. Substituted a bounded go vet over the three touched Go packages (green).
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)

---
satisfies: [R1, R3, R4, R5, R6, R7, R8]
---
# fn-77-typed-operations-parameterized-actions.10 Qualify two Nexus SDK operations with captured field relations

## Description
Qualify two Nexus SDK operations with captured field relations for the referenced parent requirements.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/**; model/Temporal/System/Nexus/**; model/Umpire/Case/Tests/**; tests/testpilot_async_nexus_case_test.go; tests/testcore/testpilot/**; common/testing/testpilot/testdata/**
**Touches:** [model/Temporal/Feature/Nexus3/**, model/Temporal/System/Nexus/**, model/Umpire/Case/Tests/**, tests/testpilot_async_nexus_case_test.go, tests/testcore/testpilot/**, common/testing/testpilot/testdata/**]

### Approach
- Extend the existing workflow-owned Start/Await/completion example to two explicitly modeled operation identities using distinct SDK-command/event declarations and Target-owned alternatives.
- Capture a typed earlier command field under its operation key and independently require correlated completion fields within the existing scoped bounded response. Retain exact scheduled-operation identity through Link correspondence.
- Generate via task8 mapping/proofs; run the new exact TestTestpilotTypedNexusOperationsCase through existing shared Driver, handler and public facade. Preserve submission/acknowledgement/confirmation/resolution boundaries and all existing cancellation/cleanup behavior without adding operation cancellation.
- Exercise wrong-operation identity, missing/late/partial response, repeated triggers and captured-value mutation; trace every product assertion to its clause and extra correlation to Link obligation.

### Investigation targets
**Required:**
- model/Temporal/Feature/Nexus3/Nexus.lean — existing model requirements.
- model/Temporal/Feature/Nexus3/Testpilot.lean:90 — real SDK Program composition.
- model/Temporal/System/Nexus/ImplementationLink.lean — correlated evidence authority.
- common/testing/testpilot/temporal/worker/interpreter.go — shared SDK execution owner (read-only reuse).
- tests/testpilot_async_nexus_case_test.go — existing live integration.

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests Umpire.Property.Tests.Scoped.Evidence Umpire.Case.CompilerTests`
`go test -tags test_dep ./common/testing/testpilot ./tests/testcore/testpilot`
`go test -json -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotTypedNexusOperationsCase|TestTestpilotAsyncNexusCase|TestTestpilotAsyncNexusCaseMissingRemoteEndpoint)$' > /tmp/fn77-task10-live.jsonl && python3 -c 'import json,sys; e=[json.loads(x) for x in open(sys.argv[1])]; names=sys.argv[2:]; assert all(any(v.get("Test")==n and v.get("Action")=="run" for v in e) and any(v.get("Test")==n and v.get("Action")=="pass" for v in e) for n in names); assert not any(v.get("Action") in ("skip","fail") for v in e)' /tmp/fn77-task10-live.jsonl TestTestpilotTypedNexusOperationsCase TestTestpilotAsyncNexusCase TestTestpilotAsyncNexusCaseMissingRemoteEndpoint`

`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests.TypedNexus`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

The new live test is named TestTestpilotTypedNexusOperationsCase. Quick receipt validation requires each named test to run and pass and rejects skip/fail events, including the preserved baseline tests.

Create and wire the proposed test module Temporal.Feature.Nexus3.Tests.TypedNexus into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Two real workflow-owned SDK operations retain typed command/completion identity and keyed prior-value comparison under existing scoped semantics; no RPC substitution.
- [ ] Wrong correlation fails Link, differing authored field fails the responsible clause, and missing/partial evidence stays unresolved without synthetic deadline.
- [ ] Captured fields remain operation/run-local and immutable; repeated/interleaved controls retain exact supporting evidence.
- [ ] Named live two-operation and preserved baseline tests show run/pass/no-skip; one Property edit changes its derived Contract without duplicate handwritten assertions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

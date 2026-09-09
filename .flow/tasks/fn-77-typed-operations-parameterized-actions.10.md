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
Two workflow-owned Nexus operations now run in one Case, each a separate modeled operation identity
whose scheduled evidence it retains under its own operation key, and a completion is required to
reference the scheduled event its own operation was scheduled at.

`model/Temporal/Feature/Nexus3/TypedNexus.lean` holds the whole authored example. The submission is
an SDK command, not an RPC: `Operation.sdkCommand` declares one identity per scheduled operation
and `Operation.event` declares the scheduled and completed semantic events. The only generated RPC
references are environment work — `StartWorkflowExecution` starts the workflow the operations run
in, `GetWorkflowExecutionHistory` supplies the response schema every event field is read through —
and both are admitted by `Temporal.API.bindUnary` so no method string or schema is copied; even the
Program's two transport paths are derived from the admitted `RpcSchema.fullName` through
`CaseSupport.methodPath`, which moved there from `TypedUnary` so both examples share one derivation.

Two requirements sit on the same evidence and fail in different ways. The **Link** is the scoped
clause's correlation: each operation captures, under its own key, the operation identity its
scheduled evidence recorded, and a later step belongs to that operation only when the retained
identity is one the model declares — an undeclared one fails admission with `invalidTransition`
rather than reporting a product violation. The **authored field requirement** is a same-step clause:
the `scheduled_event_id` a completed event records must equal the `event_id` the operation's prior
state carries. The Target owns both the correlated and the crossed completion, so selecting the
await Action never selects which arrives and the crossed one is a clause violation. The bounded
response is the scoped clause itself: scheduled-only closes unresolved, a completion after the
window closes is violated, and neither answer comes from a synthetic deadline.

`model/Umpire/Case/Observed.lean` is the new deep module the derived Contract rests on: it derives
the runtime read path a modeled operand's coordinates describe, rooted at the declared Observation's
own message, mirroring `Coverage.targetPath` in the other direction. `Umpire.Case.Tests.ObservedPath`
(wired into `UmpireTests`) drives it over a small schema — plain fields, both oneof members, a wrong
group, an unselected member, and a presence read, repeated element, keyed map lookup and cardinality
each rejecting by name. Every field path in the two derived rules is `Observed.pathOf` applied to the
same `PropertyFieldPath` the model Property compares, so a coordinate edit moves the runtime read.

`Temporal.Feature.Nexus3.Tests.TypedNexus` (wired into `TemporalModelTests`) drives the model:
declaration rejections for an unnameable command or event identity and wrong-method/streaming
bindings; matched, interleaved, late, scheduled-only, tampered-identity, repeated-occurrence and
foreign-operation streams through the compiled scoped consumer; correlated, crossed and
missing-evidence answers from the field Property; and the derived read paths written out rather than
read back. `TestTestpilotTypedNexusOperationsCase` runs the Case twice on the real shared Driver
through the public `Prepare`/`Run` facade, reads each rule's supporting Observations back out of the
Run to confirm they are the scheduled event of that rule's own operation and the completion
referencing it, and the baseline Nexus3 live success and rejection tests pass in the same receipt.

### Program ceilings this Case had to declare

Two operations do not fit the shared single-operation ceilings, and the two that had to move are
coupled: the Contract's per-event work is a multiple of the Program's `max_response_bytes`, so the
history that records both operations needs a bigger response budget than 4096 while two rules over
that budget must still fit `max_work_per_event`. The Case declares `max_response_bytes := 8192` and a
20s cleanup budget for its two handler entrypoints; the measured window is 6144 (below it the history
read is `resourceexhausted`) to ~8192 (above it contract admission exceeds the work ceiling).

### Known Gap recorded on the Case

The bounded-response clause has no runtime counterpart: the Driver evaluates a scoped capability only
from declared `ScopedEvidence` Observations and no instruction of this Program emits one. The Case
declares that as a Known Gap rather than letting the recorded Property imply an online window.

### Follow-ups, not built here

- Task .9's non-blocking P3 — a runtime `violated` state plus a tampered-fixture live assertion — was
  offered to this task and is declined: the acceptance here does not name it, so it stays with .11.
- The review raised two P3s left open: `TypedNexusProfile` and the live-binding setup duplicate
  their async-nexus counterparts and want one shared builder, and the Link admits any *declared*
  identity because a correlation operand cannot name the scope key (a crossed-but-declared identity
  is separated by the field requirement and by the per-identity runtime rules instead; both answers
  are now pinned by `#guard`).

Concurrent local edits this run swept in that this task did not author: the user committed a `wip`
snapshot (`b8c30e29`) mid-run that already contained an early version of these files together with
the parallel session's `fn-81`/`fn-82` spec and task files, `.plans/GOMAD_MILESTONES.md`,
`.plans/UMPIRE4_ORDER.md` and regenerated `tools/gomad3` wire tables; that commit is inside this
task's evidence range because it precedes both of this task's own commits.

stage: impl-review - ran [round 1 SHIP (claude/claude-fable-5-1, high)]; one P2 and four P3 findings,
the P2 (model-only window unrecorded) and two P3s (over-claimed derived-Contract wording, unpinned
crossed-declared identity) fixed in 390ffd18 with every gate re-run; the remaining two recorded above
as follow-ups
## Evidence
- Commits: b8c30e2948d268c28f56b1577c6f61f2a3070c9b, 09feba23f697e5f4e4d5ce95d12fe5e4409cf26b, 390ffd1821213108c0350112e8bdff6f1d715e3e
- Tests: baseline: green via handoff (verified at 2c77b700 by fn-77-typed-operations-parameterized-actions.9), cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests Umpire.Property.Tests.Scoped.Evidence Umpire.Case.CompilerTests Temporal.Feature.Nexus3.Tests.TypedNexus, cd model && mise exec -- lake build TemporalModelTests UmpireTests, go test -count=1 -tags test_dep ./common/testing/testpilot ./tests/testcore/testpilot ./tools/umpire/cmd/umpire-gen-case-runtime-conformance, go test -json -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotTypedNexusOperationsCase|TestTestpilotAsyncNexusCase|TestTestpilotAsyncNexusCaseMissingRemoteEndpoint)$' (run+pass, no skip/fail for all three), mise exec -- make umpire-gen-case-runtime-conformance, mise exec -- make umpire-check-case-runtime-conformance, mise exec -- make umpire-check-testpilot-authoring umpire-check-retired-vocabulary, mise exec -- make umpire-build-model, mise exec -- make lint-model (169 Temporal.Lint + 2 Umpire.Lint findings == confirmed inherited baseline; zero findings in the new modules), go vet -tags 'test_dep integration' ./tests ./tests/testcore/testpilot ./tools/umpire/cmd/umpire-gen-case-runtime-conformance, GATE_SKIPPED:lint-code:disk - make lint-code runs go vet ./... over the whole repo; 4.4 GiB free is below what that build needs. Substituted a bounded go vet over the three touched Go packages (green).
- PRs:
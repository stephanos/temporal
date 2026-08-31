---
satisfies: [R3, R4]
---
# fn-21-nexus-duplicate-observation-control.4 Capture fault realization as closed causal evidence

## Description
Project the requested fault, completed real receipt, injected marker, separate counts, and shared correlation into fn-19/fn-18 operational artifacts for R3/R4. Prove operational success only when all existing phase/source/cleanup obligations close.

### Transport/semantic boundary (normative)

This task ends at R4's admitted transport truth: the real callback and labeled synthetic contribution remain distinct participant-command RawEvidence facts, and RawEvidence retains the complete requested/completed history chain. It intentionally does not coalesce those records into Task `.7`'s semantic profile or derive that profile's `faultTarget`; Task `.5` owns both operations in the existing fn-20 raw-to-semantic adapter before Observation Evaluation. No RunEvaluation or Lean implementation belongs in Task `.4`.

**Size:** M
**Files:** `tools/umpire/runtime/evidence.go`, `tools/umpire/internal/artifactv2/evidence.go`, `tools/umpire/internal/runtimeengine/evidence.go`, `tools/umpire/internal/runtimeengine/engine.go`, `tools/umpire/temporal/nexus/participant.go`, `tools/umpire/temporal/nexus/evidence.go`, `tools/umpire/temporal/nexus/output.go`, `tools/umpire/temporal/nexus/evidence_test.go`, `tools/umpire/temporal/nexus/integration_test.go`, `tools/umpire/temporal/nexus/participant_test.go`, `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/**`
**Touches:** [tools/umpire/runtime/evidence.go, tools/umpire/internal/artifactv2/evidence.go, tools/umpire/internal/runtimeengine/evidence.go, tools/umpire/internal/runtimeengine/engine.go, tools/umpire/temporal/nexus/participant.go, tools/umpire/temporal/nexus/evidence.go, tools/umpire/temporal/nexus/output.go, tools/umpire/temporal/nexus/evidence_test.go, tools/umpire/temporal/nexus/integration_test.go, tools/umpire/temporal/nexus/participant_test.go, tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/**]

### Approach
- Reuse fn-19's four-source accumulator and fn-18 Run/RawEvidence constructors; extend the existing participant fact producer and accumulator with only the domain-neutral synthetic-contribution count and marker fields required by Task `.7`'s checked profile.
- Bind the one control attempt to the planned occurrence, exact non-null fault identity, accepted status, and one receipt fact. Project mechanical callback count one, synthetic-contribution count one, and the marker as unique ordered transport facts sharing the real run/operation/cancellation correlation.
- Retain one normal history chain with one requested and one completed cancellation event, explicit phase/source closure, and zero open handles; never label the injected observation as a second SDK/server callback or history request.
- Charge every extra fact/field before append under existing limits and emit the mutation table's operational `incomplete`/status-2 Known Gap and gap on N+1.
- Build and fn-18-admit the four-member output in memory; independently mutate callback count, contribution count, marker, fault, receipt, correlation, causality, order, disposition, closure, and capacity and pin every case to the exact table row before one bounded live control.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md:13-31` — four-source evidence/output pattern
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.3.md:13-30` — accumulator, limits, and independent oracle
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md:83-84` — exact phase/control attempt records
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md:136-180` — Run/RawEvidence/set invariants
- `.flow/specs/fn-21-nexus-duplicate-observation-control.md` — closed mutation/status table

### Acceptance
- [ ] The admitted faulted run has one fault-bound accepted control receipt and four closed sources with exact causal/reference closure.
- [ ] Evidence truthfully records one real callback and one separate labeled synthetic contribution while retaining the normal requested/completed lifecycle.
- [ ] Full realization/closure/cleanup is operationally succeeded; every mutation produces the exact invariant/failed/incomplete outcome in the parent table.
- [ ] Unsafe payload/header/error/authority material cannot enter retained fields.
- [ ] The normal evidence fixture remains byte-identical and cross-run facts cannot bind.
## Acceptance
- [ ] R3 operational status remains independent from the induced semantic failure.
- [ ] R4 exact fault-realization evidence is bounded, causal, closed, and admitted.

## Done summary
Implemented and admitted the closed R3/R4 duplicate-delivery transport evidence contract.

- The synthetic contribution is a separate `umpire.evidence.kind.participant-command` fact with exact callback/synthetic counts, marker, fault receipt, capability, and shared correlations.
- Accepted faulted control receipts retain the exact four additional fault-binding identities; normal receipts remain the original four-field shape and normal fixture bytes are unchanged.
- Artifact admission permits only the closed four- or eight-field receipt shapes. Runtime output validation retains structural/safety admission while leaving semantic missing/count/conflict/disposition cases to fn-20 Observation Evaluation.
- The persisted faulted four-member run set was regenerated from the exact admitted fixture. Focused live, mutation, package aggregate, RunEvaluation aggregate, and model-lint gates pass. Full Go lint remains inherited red with 1,387 repository findings; all 19 task-path reports are on pre-existing lines and no lint auto-edits occurred.
- Mandatory Codex implementation review reached SHIP with zero introduced findings. The normative task boundary preserves distinct raw callback/synthetic facts and the complete history chain; Task `.5` owns raw-to-semantic coalescing and `faultTarget` derivation in the fn-20 adapter.
## Evidence
- Commits: 7e31acfcd34b13c3faa13f76982723af908c05c3, 58cf618ab46e537b3e7d1b7eeaa353485c979751, 7122f8ac93dd18184280362c7362e7fda04754cd, d9be6d23949d74414a8c3075f40fbfdcfd5ce626, 19e17b40e2fd1e52b690f65ca644983afb85c3eb, db74045b34db117e4dca17b4ab1bf9fc68a1d613, 2f727d5eaa5e462abe8b594912b9e05dd8e2a5ab, 97d52a485b6effb6f5434db64eebb89e5ddbcc64
- Tests: RED: TMPDIR=/private/tmp/umpire-fn21-task4-tmp go test -tags test_dep ./tools/umpire/temporal/nexus -run '^TestLiveFaultedCallerClosureReturnsClosedFaultRealizationEvidence$' -count=1 (expected participant-command, observed custom marker kind), RED: TMPDIR=/private/tmp/umpire-fn21-task4-tmp go test -tags test_dep ./tools/umpire/temporal/nexus -run '^(TestValidateExecutionClosureAdmitsFaultedEvidenceForEvaluation|TestLiveFaultedCallerClosureReturnsClosedFaultRealizationEvidence)$' -count=1 (8-field control receipt rejected; custom kind observed), TMPDIR=/private/tmp/umpire-fn21-task4-tmp go test -tags test_dep ./tools/umpire/runtime ./tools/umpire/internal/artifactv2 ./tools/umpire/internal/runtimeengine ./tools/umpire/temporal/nexus -count=1, TMPDIR=/private/tmp/umpire-fn21-task4-tmp go test -tags test_dep ./tools/umpire/runevaluation -count=1, make lint-model, inherited red: make GOLANGCI_LINT_FIX=false lint-code (1,387 repository findings; 19 task-path reports, zero introduced; no auto-edits), git diff 18d58b84898fa46e8f043ab05a065043648b5eb5 --name-only -- tools/umpire/temporal/nexus/testdata/caller-closure-input-set tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set (empty), Codex impl-review receipt /tmp/impl-review-receipt-fn-21-nexus-duplicate-observation-control.4.json: SHIP, zero introduced findings
- PRs:
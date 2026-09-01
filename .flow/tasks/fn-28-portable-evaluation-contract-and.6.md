---
satisfies: [R4, R5]
---

# fn-28-portable-evaluation-contract-and.6 Deepen the resident executor and Evidence-closure lifecycle
## Description

Compose contract admission, the existing runner, explicit Evidence closure, portable evaluation, cleanup, and local decision mapping behind one small resident executor interface.

**Size:** L
**Files:** `tools/umpire/executor/**`, `tools/umpire/runner/**`, `tools/umpire/temporal/local/**`
**Touches:** [`tools/umpire/executor/**`, `tools/umpire/runner/**`, `tools/umpire/temporal/local/**`]

### Approach
- Expose one request/result seam; keep phases, adapters, resource accounting, and status mapping internal to the module.
- Reuse an attached authority across bounded requests while assigning fresh run correlations and owning only per-run workers/endpoints/workflows; never close the enclosing cluster/client.
- Wait for contract-declared terminal receipts and source closure within explicit Limits. Mark the executor poisoned after uncertain cleanup and reject further work.
- Guard `idle`/`active`/`poisoned` atomically: reject overlap as typed pre-I/O `busy`/`inconclusive`, return to idle only after complete cleanup, and never queue requests internally.

### Investigation targets

**Required** (read before coding):
- Existing `runner.Run`, runtime engine phases, `nexus.Binding`, and local authority/resource ownership.
- Parent contract/evaluator tasks and current cleanup/source-closure validation.
- Existing cancellation and failure classification tests.

## Acceptance
- [ ] A caller can execute a complete contract through one small interface without orchestrating admission, execution, evaluation, or cleanup phases.
- [ ] Multiple closed runs reuse the resident process/authority safely; run identity or resource leakage and post-uncertain-cleanup reuse fail closed.
- [ ] Eventual closure, deadline, cancellation, cleanup, race, and N/N+1 tests preserve independent statuses and never infer absence from quiet time.
- [ ] Overlap loses atomically before runtime I/O, active cancellation cannot expose idle early, and poisoned state permanently rejects reuse.

## Done summary
Implemented a single-flight resident Umpire executor that composes strict contract/input admission, fresh run correlation, bounded runner execution, explicit source-closure evaluation, cleanup-safe reuse, and typed busy/poisoned/canceled/internal results. Focused lifecycle coverage proves exact N/N+1 status preservation, explicit closure, sequential reuse, atomic overlap rejection, cancellation/deadline cleanup barriers, and permanent poisoning after cleanup uncertainty.

baseline: green for proto generation, focused contract/evaluator Go, tagged integration command, Lean portable-contract target, and model lint; inherited red at `model/Umpire/SemanticInventory/KnownGaps.lean:296` in `make umpire-check-regression`; repo `make lint-code` resource/debt stall classified at baseline and not repeated.
verification: focused Go, final exact Go quick gate, tagged integration command, Go vet, diff-scoped golangci-lint, Lean target, and `make lint-model` green; race build blocked before test execution by clang ENOSPC after consuming more than 7 GiB of local disk workspace.
stage: impl-review - ran [2026-09-01T23:24:10Z..2026-09-01T23:28:55Z]
## Evidence
- Commits: dd631aa861346487e3328f8a8a660e6789db4c3d
- Tests: baseline: green (make proto; local generator-version drift restored), cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/..., go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$' (green; no tests to run until task .9), make lint-model, INHERITED_RED: make umpire-check-regression (model/Umpire/SemanticInventory/KnownGaps.lean:296; unchanged from pre-edit baseline), INHERITED_RESOURCE_STALL: make lint-code (classified at pre-edit baseline; not repeated per conductor instruction), go test -count=1 -tags test_dep ./tools/umpire/executor/... ./tools/umpire/runner/..., go vet -tags test_dep ./tools/umpire/executor/... ./tools/umpire/runner/..., .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=562b77b2b151e3c2903708485cde0163e9ed6c7b --config=.github/.golangci.yml ./tools/umpire/executor/... ./tools/umpire/runner/... (0 issues), TOOLING_RESOURCE_BLOCKED: go test -race -count=1 -tags test_dep ./tools/umpire/executor/... (clang ENOSPC before test execution after more than 7 GiB free was consumed), git diff --check, impl-review: SHIP (codex:gpt-5.6-sol:high)
- PRs:
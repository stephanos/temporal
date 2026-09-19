---
satisfies: [R4, R8]
---

# fn-28-portable-evaluation-contract-and.10 Prove fail-closed closure, mutation, and resident reuse
## Description

Complete the cross-module negative matrix for portable contracts, eventual Evidence closure, HTTP transport, executor reuse, and disposable-cluster cleanup.

**Size:** L
**Files:** `tools/umpire/evaluationcontract/*_test.go`, `tools/umpire/portableevaluation/*_test.go`, `tools/umpire/executor/*_test.go`, `tools/umpire/executorhttp/*_test.go`, `tests/umpire4_portable_executor_test.go`
**Touches:** [`tools/umpire/evaluationcontract/*_test.go`, `tools/umpire/portableevaluation/*_test.go`, `tools/umpire/executor/*_test.go`, `tools/umpire/executorhttp/*_test.go`, `tests/umpire4_portable_executor_test.go`]

### Approach
- Mutate every binding/operator/Limit/closure/status seam independently and require the responsible stage to reject it without partial success.
- Exercise delayed-but-closed Evidence, deadline-before-closure, post-closure records, duplicate/source-crossed facts, stale run correlations, overlapping requests, cancellation, cleanup uncertainty, and poisoned-executor reuse.
- Keep global/model claims outside the matrix; these tests prove only one exact contract's local evaluation behavior.

### Investigation targets

**Required** (read before coding):
- Parent tasks `.2`, `.4`, `.6`, `.8`, and `.9` test matrices.
- Existing Umpire mutation tests and source-closure semantics.
- Repository race/fuzz and eventual-consistency test patterns; use `require.Eventually`, never sleep.

## Acceptance
- [ ] Exact N succeeds and N+1 fails for contract, body, Evidence, operator, and time/work Limits at the responsible seam.
- [ ] Missing/late/ambiguous/conflicting/unsupported Evidence and uncertain cleanup produce `inconclusive`, never pass or an invented violation.
- [ ] Crossed bindings, stale correlations, unknown operators, overlapping admission, cancellation leaks, post-closure Evidence, and reuse after poisoning fail closed under unit, race, fuzz, and tagged integration tests.

## Done summary
Completed the portable evaluation fail-closed matrix with exact serialized-contract, expression-depth, time, and work boundaries; closed-source missing and source-crossed Evidence cases; HTTP wire fuzzing; and tagged live crossed-fixture/reuse coverage. Scoped unit, race, fuzz, tagged integration, vet, and lint verification passed after reclaiming only the rebuildable Go build cache; the pre-edit regression-vocabulary and repo-wide lint ENOSPC failures remain inherited.

The exact unit and integration gate receipts could not be written because Flow correctly refused to warrant a receipt while the preserved user-owned `.plans/UMPIRE4_ORDER.md` edit remained dirty; both commands themselves exited zero.

stage: impl-review - ran
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d531962aa08fdb45bb00d28447b16fe62b291c88
- Tests: baseline: green (make proto), baseline: green ((cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests)), baseline: green (go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/...), baseline: green (go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$'), baseline: green (make lint-model), baseline: red (make umpire-check-regression failed pre-edit: active-vocabulary guard rejects model/Umpire/SemanticInventory/KnownGaps.lean:296), baseline: red (make lint-code failed pre-edit: ENOSPC while typechecking unrelated tools/umpire2/internal/action/environment_test.go), go clean -cache (authorized rebuildable Go build-cache recovery after ENOSPC), go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/..., go test -p=1 -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... ./tools/umpire/executorhttp/..., go test -race -p=1 -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/..., go test -race -p=1 -count=1 -tags test_dep ./tools/umpire/executor/..., go test -race -p=1 -count=1 -tags test_dep ./tools/umpire/executorhttp/..., go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract -run '^$' -fuzz '^FuzzAdmitRejectsSingleByteContractMutations$' -fuzztime=100x, go test -count=1 -tags test_dep ./tools/umpire/portableevaluation -run '^$' -fuzz '^FuzzEvaluateFailsClosed$' -fuzztime=100x, go test -count=1 -tags test_dep ./tools/umpire/executorhttp -run '^$' -fuzz '^FuzzHandlerWireSurfaceFailsClosed$' -fuzztime=100x, go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$', go vet -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... ./tools/umpire/executorhttp/..., .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=1d53c88f36cc9a05ff270f0c9d33bc20e21b3959 --config=.github/.golangci.yml ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... ./tools/umpire/executorhttp/..., git diff --check 1d53c88f36cc9a05ff270f0c9d33bc20e21b3959..HEAD, impl-review codex: SHIP
- PRs:
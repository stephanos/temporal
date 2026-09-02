---
satisfies: [R5, R6]
---

# fn-28-portable-evaluation-contract-and.7 Add the attached disposable-cluster authority adapter
## Description

Add the narrow adapter that lets the resident executor use a caller-owned Temporal SDK client and namespace while retaining strict ownership of Umpire-created workers and run resources.

**Size:** M
**Files:** `tools/umpire/temporal/local/attached.go`, `tools/umpire/temporal/local/attached_test.go`
**Touches:** [`tools/umpire/temporal/local/attached.go`, `tools/umpire/temporal/local/attached_test.go`]

### Approach
- Accept only the minimum attached-authority capabilities needed by the existing local environment; do not import `tests/testcore` into production packages.
- Treat the cluster and SDK client as borrowed, start fresh run-owned workers/task queues, and stop/delete only resources acquired by Umpire.
- Exercise the adapter with focused fakes before the tagged test supplies the concrete `testcore.NewEnv` implementation.

### Investigation targets

**Required** (read before coding):
- `tools/umpire/temporal/local/environment.go` authority and resource accounting.
- `tests/testcore.TestEnv` SDK client, namespace, worker, and cleanup ownership.
- Parent executor interface and existing runner adapter contract.

## Acceptance
- [ ] The exported attached-authority seam is minimal and has both existing loopback/fake and later testcore adapters.
- [ ] Borrowed cluster/client resources are never stopped or reported as Umpire-owned; every Umpire-created worker/resource closes exactly once.
- [ ] Drift, nil/incomplete authority, cancellation, cleanup failure, and reuse tests fail closed.

## Done summary
Implemented the attached Temporal authority factory over a borrowed client/namespace/endpoint, with per-run worker ownership, exact input revalidation, scoped isolation evidence, and deterministic resource receipts. Focused fakes prove authority drift, incomplete bindings, namespace reuse, cleanup cancellation/failure, blocked lifecycle cancellation, eventual exact-once closure, and that the borrowed client is never owned or closed; the concrete testcore integration remains assigned to task .9.

baseline: red (make umpire-check-regression failed pre-edit at model/Umpire/SemanticInventory/KnownGaps.lean:296 on the inherited Temporal-owned prefix); red (make lint-code failed pre-edit with the inherited ENOSPC typecheck stall); all other Quick commands passed pre-edit.
verification: green (focused attached-local Go tests, aggregate Umpire Go tests, tagged integration command, proto generation, portable-contract Lean build, model lint, Go vet, and diff-scoped golangci). The optional race attempt hit the same inherited ENOSPC class and was not retried; proto's 18 generator-version formatting side effects were restored only for paths clean before generation.
stage: impl-review - ran [2026-09-02T00:04:47Z..2026-09-02T00:21:48Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7d6d13936629a76f0d3b66ea3a123b71d234711c, f0d008def7176dc9a0fbdc12693da8c922dcc307
- Tests: make proto, cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, go test -count=1 -tags test_dep ./tools/umpire/temporal/local/..., go vet -tags test_dep ./tools/umpire/temporal/local/..., .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=372acd4eb40e8dda10473a3b2dc163f30b2b73fc --config=.github/.golangci.yml ./tools/umpire/temporal/local/..., go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/..., go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$', make lint-model, INHERITED_BASELINE_RED:make umpire-check-regression - KnownGaps.lean:296 Temporal-owned prefix, INHERITED_BASELINE_RED:make lint-code - ENOSPC during typecheck, INHERITED_OPTIONAL_RACE:go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/... - ENOSPC during dependency vet config generation
- PRs:

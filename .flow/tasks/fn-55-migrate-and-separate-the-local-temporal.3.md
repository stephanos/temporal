---
satisfies: [R1, R2, R4, R6]
---
# fn-55-migrate-and-separate-the-local-temporal.3 Remove the legacy authority and extract the private seam

## Description
After every live caller supplies an attached factory, delete the deprecated concrete authority and zero-option fallback, then extract the remaining private authority seam and resource algebra from environment orchestration. Make ownership names and receipts describe only Umpire-owned resources.

**Size:** M
**Files:** `tools/umpire/temporal/local/environment.go`, `tools/umpire/temporal/local/authority.go`, `tools/umpire/temporal/local/attached.go`, `tools/umpire/temporal/local/environment_test.go`, `tools/umpire/temporal/local/authority_test.go`, `tools/umpire/temporal/local/attached_test.go`, `tools/umpire/temporal/local/lifecycle_test.go`, `tools/umpire/temporal/nexus/runner.go`, `tools/umpire/runner/runner_test.go`
**Touches:** [tools/umpire/temporal/local/environment.go, tools/umpire/temporal/local/authority.go, tools/umpire/temporal/local/attached.go, tools/umpire/temporal/local/environment_test.go, tools/umpire/temporal/local/authority_test.go, tools/umpire/temporal/local/attached_test.go, tools/umpire/temporal/local/lifecycle_test.go, tools/umpire/temporal/nexus/runner.go, tools/umpire/runner/runner_test.go]

### Approach
- Begin from fn-53's final environment layout and leave its isolation state machine, probe coordination, receipt construction, and locking in their post-fn-53 owners.
- Remove `local.NewFactory`, the concrete legacy starter/authority, its import and conformance assertion, and the temporary Nexus zero-value fallback retained during tasks `.1`/`.2`.
- Move the unchanged private `authorityStarter`/`temporalAuthority` contracts plus resource-kind translation into `authority.go`; do not add a second interface or exported declaration.
- Rename private ownership concepts that imply a server/client when they actually represent the Umpire environment wrapper/worker. Preserve the public resource kinds, identities, ordering, and attached-path receipts while ensuring borrowed TestEnv cluster/client resources never appear.
- Keep `environment.go` responsible for preparation validation/precedence, lifecycle state, receipts, synchronization, isolation, and translation of authority ownership into runtime resources.
- Retain and strengthen deterministic fake coverage for nil/partial start, connection failure, worker creation/start cancellation, stop failure/cancellation, residual ownership, repeated cleanup, successful retry, binding drift, and concurrent distinct factories. Attached worker tests use fakes and no live server.
- Preserve every existing comment with its declaration; rewrite only the legacy-specific environment ownership comment that becomes false.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/temporal/local/environment.go:79-174,341-370,472-547,782-913` — factory, receipts, ownership translation, seam, and legacy authority
- `tools/umpire/temporal/local/attached.go:18-98,123-162,204-342,380-390` — borrowed binding, worker ownership, stop semantics, and stale conformance assertion
- `tools/umpire/temporal/local/environment_test.go:22-138` — real/default factory cases to remove or convert to fakes
- `tools/umpire/temporal/local/lifecycle_test.go:314-409` — recording authority and resource-order oracle
- `tools/umpire/temporal/local/attached_test.go:91-152,286-373` — deterministic binding, cancellation, and cleanup coverage
- `tools/umpire/runner/runner_test.go:114-127,294-329` — adapters that currently embed zero-value Nexus behavior
- `.flow/tasks/fn-28-portable-evaluation-contract-and.7.md:14-29` — accepted TestEnv/attached ownership precedent

### Key context
- This task intentionally removes the public zero-option factory. `NewAttachedFactory` is the only real local authority constructor afterward.
- A borrowed SDK client is never an Umpire connection resource. The environment marker represents the per-run wrapper, not the TestEnv cluster.
- Do not import TestEnv into production or package-local tests; fakes are the failure oracle because TestEnv startup uses `t.Fatalf`.

## Acceptance
- [ ] `local.NewFactory`, the concrete legacy starter/authority, its import/conformance assertion, and every Nexus fallback path are deleted with no compatibility wrapper or global registry.
- [ ] `Environment`, `WorkerRegistration`, `AsEnvironment`, `AttachedAuthority`, and `NewAttachedFactory` retain their exact exported shapes; the private starter/authority seam is not widened.
- [ ] `authority.go` owns only the private contracts and Umpire-owned resource algebra; environment orchestration, isolation, synchronization, receipts, and translation remain outside it.
- [ ] Attached preparation and cleanup report only the Umpire environment wrapper and worker, never the borrowed TestEnv cluster/client, with unchanged identities/order and zero open handles after success.
- [ ] Exact fake-backed matrices cover invalid/canceled preparation, nil/partial authority, connect failure, worker acquisition/start cancellation, stop failure/cancellation, residual ownership, repeated cleanup, retry, binding drift, and distinct-factory concurrency.
- [ ] Existing comments are preserved except the obsolete legacy ownership statement, which is rewritten accurately.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/temporal/local/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/...` passes.
- [ ] `go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/...` passes.


## Done summary
Removed the deprecated local test-server authority and Nexus fallback, extracted the unchanged private authority contracts plus Umpire-owned environment/worker algebra, and made the runner reject a missing factory before participant construction. Deterministic fake matrices now cover the required preparation, authority, worker, cleanup, drift, retry, and concurrency cases; focused unit/race and 270-job aggregate regression gates pass, while the parent integration and global lint gates remain inherited red only with no Umpire4 failure and zero task-scoped lint findings.

stage: impl-review - ran [2026-09-04T07:00:55Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f546aefa3bd2a18437ed9a0bac79c3cc8a25ec92
- Tests: baseline: green via handoff (verified at abf00193 by fn-55-migrate-and-separate-the-local-temporal.2); make fmt-imports baseline passed; INHERITED_RED: make lint-code baseline (1,378 repository findings), TMPDIR=$(pwd -P)/.flow/tmp/go-tmp CC=$(xcrun --find clang) SDKROOT=$(xcrun --show-sdk-path) go test -count=1 -tags test_dep ./tools/umpire/temporal/local/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/... ./tools/umpire/runevaluation/..., TMPDIR=$(pwd -P)/.flow/tmp/go-tmp CC=$(xcrun --find clang) SDKROOT=$(xcrun --show-sdk-path) go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/..., INHERITED_RED: TMPDIR=$(pwd -P)/.flow/tmp/go-tmp CC=$(xcrun --find clang) SDKROOT=$(xcrun --show-sdk-path) go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire' (only unchanged Umpire3 participant build-path and Umpire2 probe/coverage failures; no Umpire4 failure), TMPDIR=$(pwd -P)/.flow/tmp/go-tmp CC=$(xcrun --find clang) SDKROOT=$(xcrun --show-sdk-path) make umpire-check-regression, make fmt-imports, INHERITED_RED: TMPDIR=$(pwd -P)/.flow/tmp/go-tmp CC=$(xcrun --find clang) SDKROOT=$(xcrun --show-sdk-path) make lint-code (1,374 repository findings versus 1,378 pre-edit baseline; unrelated auto-edit restored), TMPDIR=$(pwd -P)/.flow/tmp/go-tmp CC=$(xcrun --find clang) SDKROOT=$(xcrun --show-sdk-path) .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules,test_dep,integration' --timeout 10m --fix=false --new-from-rev=abf00193c84403508a9f86c7932f7d917074e512 --config=.github/.golangci.yml ./tools/umpire/runner/... ./tools/umpire/temporal/local/... ./tools/umpire/temporal/nexus/... (0 issues), git diff --check, Codex impl-review /tmp/impl-review-receipt-fn-55-migrate-and-separate-the-local-temporal.3.json: SHIP
- PRs:
---
satisfies: [R2, R4, R7]
---
# fn-19-bounded-local-temporal-execution-and.4 Implement the isolated ephemeral Temporal authority adapter

## Description
Implement R2/R4's sole local environment using the repository's real loopback `temporaltest` server lifecycle.

**Size:** M
**Files:** `tools/umpire/temporal/local/profile.go`, `tools/umpire/temporal/local/environment.go`, `tools/umpire/temporal/local/environment_test.go`, `tools/umpire/temporal/local/lifecycle_test.go`
**Touches:** [tools/umpire/temporal/local/profile.go, tools/umpire/temporal/local/environment.go, tools/umpire/temporal/local/environment_test.go, tools/umpire/temporal/local/lifecycle_test.go]

### Approach
- Consume Task `.9`'s error-returning context-aware lifecycle API for runtime-owned explicit startup/stop; own workers, clients, namespace, and server in workers→clients→server teardown order.
- Match the exact Task `.1` profile Definition ID/Behavior Fingerprint/capabilities and expose no address, namespace, credential, executable, or arbitrary option input.
- Derive run-owned workflow/task-queue/worker correlation IDs, hash environment-only identities for evidence, and serialize no live handles.
- Apply explicit contexts to every SDK/server call and force single-attempt/no-default-retry options.
- Test concurrent-process-style isolated instances, partial startup, repeated cleanup requests, cancellation, and injected lifecycle failures.

### Investigation targets
**Required** (read before coding):
- `temporaltest/server.go`, `options.go`, `server_test.go:31`
- `temporaltest/internal/lite_server.go`
- Task `.9` bounded lifecycle API and failure semantics
- official Go SDK v1.44.0 client/worker lifecycle docs
- parent authority/isolation contract

### Acceptance
- [ ] Every started handle is owned and released in exact order under the cleanup budget.
- [ ] Two instances cannot share namespace/client/worker/server state or evidence identities.
- [ ] No code path can dial a caller-selected or pre-existing server.
- [ ] Lifecycle failures become sanitized runtime codes and never leak raw configuration or handles.

## Acceptance
- [ ] R4 isolated authority adapter is bounded and local-only.
- [ ] Real `temporaltest` lifecycle tests pass without shared state.
- [ ] No reusable runtime file imports Temporal-specific types.

## Done summary
Implemented the sole closed local Temporal authority adapter: a zero-option factory now owns one real loopback `temporaltest` server, namespace, client, and worker set per run; retains only run-bound digest identities; forces single-attempt workflow options; and performs bounded retryable workers→clients→server cleanup. Lifecycle failures are sanitized, operation facts have collision-free identities, and timeout-only cleanup remains incomplete while concrete failures dominate.

Verification is green for the task-owned runtime, local adapter, temporaltest, Lean local-profile, race, vet, and diff checks. The spec-level Nexus package, CLI package, Nexus execution Lean target, and root run target remain inherited-red because their later-task surfaces do not exist yet; gate receipts were non-blockingly unavailable because the inherited false-symlink config path leaves the checkout dirty. Memory capture was attempted after NEEDS_WORK→SHIP but the repository memory store is not initialized.

stage: impl-review - ran [2026-08-29T15:44:00Z..2026-08-29T15:57:40Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a6e51af88ffd801bfeee31dd3407f0e517c9ef0f, 3f15d5eba5804259906791bc389fab9f7dc3ee71, ac1caed296f6b2920d1b8e3a93854f75805d5a87
- Tests: baseline: red (resumed already-started fn19.4 TDD state did not yet compile because the recording authority still implemented the prior client seam), go test -count=1 ./tools/umpire/temporal/local/... -run TestCanceledCleanupRetainsOwnershipAndCanBeRetried (RED then GREEN), go test -count=1 ./tools/umpire/temporal/local/... -run 'TestLifecycleFactsHaveDistinctOperationIdentities|TestCleanupDeadlineReturnsTimeoutCompatibleReceipt|TestConcreteCleanupFailureDominatesExpiredDeadline' (RED then GREEN), go test -count=1 ./tools/umpire/temporal/local/... -run TestCleanupDeadlineReachedDuringStopReturnsTimeoutCompatibleReceipt (RED then GREEN), go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./tools/umpire/temporal/local/..., go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, go test -race -count=1 ./tools/umpire/temporal/local/..., go vet ./tools/umpire/temporal/local/..., git diff --check, inherited red: go test -count=1 ./tools/umpire/temporal/nexus/... (later-task package absent), inherited red: go test -count=1 ./tools/umpire/cmd/umpire-local-run/... (later-task package absent), inherited red: cd model && mise exec -- lake build Temporal.Feature.Nexus.ExecutionTests (later-task target absent), inherited red: make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=umpire.local.caller-closure.run-1 (later-task target absent), flowctl codex impl-review fn-19-bounded-local-temporal-execution-and.4 --base 164c9b6a232f426598bec6438d6239453ba1a03f (SHIP)
- PRs:

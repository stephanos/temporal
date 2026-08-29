---
satisfies: [R4, R7]
---
# fn-19-bounded-local-temporal-execution-and.9 Add a production-safe bounded temporaltest lifecycle API

## Description
Create the R4/R7 prerequisite that makes the existing loopback server usable by an operational runtime without panics, inaccessible partial state, or unbounded teardown.

**Size:** M
**Files:** `temporaltest/server.go`, `temporaltest/options.go`, `temporaltest/internal/lite_server.go`, `temporaltest/server_test.go`, `temporaltest/lifecycle_test.go`
**Touches:** [temporaltest/server.go, temporaltest/options.go, temporaltest/internal/lite_server.go, temporaltest/server_test.go, temporaltest/lifecycle_test.go]

### Approach
- Add context-aware error-returning server/client/worker startup operations; preserve existing helper APIs as wrappers so unrelated tests retain behavior and comments.
- Publish an owned partial lifecycle handle before each fallible acquisition, unwind successfully acquired resources on startup failure, and return joined typed errors rather than panic.
- Add bounded worker/server shutdown with explicit SDK worker stop timeout, deterministic workers→clients→server order, idempotent ownership state, and a context deadline result.
- Make testing integration optional: no error path may dereference a nil `testing.T`, while `WithT` keeps automatic test cleanup.
- Add deterministic failure injection around each acquire/release boundary and prove no inaccessible partial server, client, or worker survives.

### Investigation targets
**Required** (read before coding):
- `temporaltest/server.go:35-127`
- `temporaltest/options.go`
- `temporaltest/internal/lite_server.go` startup/stop APIs
- Go SDK v1.44.0 worker `Start`/`Stop` and `WorkerStopTimeout` contracts
- parent local-authority and cleanup Limits

### Acceptance
- [ ] Every startup/client/worker failure returns an error and unwinds all earlier acquisitions without panic.
- [ ] Context expiry Limits shutdown and reports exactly which owned resources remain; a later cleanup call remains safe.
- [ ] Nil testing integration is safe, while existing `NewServer(WithT(t))` behavior and callers continue to pass.
- [ ] Failure injection covers every acquisition/release boundary and preserves existing comments.

## Acceptance
- [ ] R4/R7 production-safe temporaltest lifecycle is error-returning, bounded, and cleanup-safe.
- [ ] Existing and new temporaltest suites pass.
- [ ] No runtime-specific or Umpire-specific vocabulary enters temporaltest.

## Done summary
Added error-returning, context-aware temporaltest server/client/worker lifecycle APIs with typed residual ownership, bounded once-only cleanup, explicit SDK worker stop timeouts, and panic-oriented compatibility wrappers. Deterministic tests cover each acquisition and release boundary, cancellation between boundaries, resumable cleanup deadlines, nil testing integration, and legacy dial limits while preserving the existing lifecycle comments.

Baseline and verification: runtime, temporaltest, LocalProfileTests, temporaltest race, and vet are green. The local/Nexus/CLI packages, Nexus ExecutionTests, and root run target were absent both before and after this task and remain inherited-red later-task surfaces. Gate receipt creation was non-blockingly unavailable because the inherited false-symlink config path leaves the checkout dirty. Memory capture was attempted after NEEDS_WORK to SHIP, but memory is not initialized.

stage: impl-review - ran [2026-08-29T13:53:53Z..2026-08-29T13:59:44Z]
## Evidence
- Commits: c1a46aee0907a62589379d22679d2fa26be84838, 2eafc4b31000ce114098d17f61a4c453abcc9857
- Tests: go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, go test -race -count=1 ./temporaltest/..., go vet ./temporaltest/..., INHERITED_RED: go test -count=1 ./tools/umpire/temporal/local/... (package absent before and after; later-task surface), INHERITED_RED: go test -count=1 ./tools/umpire/temporal/nexus/... (package absent before and after; later-task surface), INHERITED_RED: go test -count=1 ./tools/umpire/cmd/umpire-local-run/... (package absent before and after; later-task surface), INHERITED_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.ExecutionTests (module absent before and after; later-task surface), INHERITED_RED: make umpire-run-local SET=... (target absent before and after; later-task surface)
- PRs:
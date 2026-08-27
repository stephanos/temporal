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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

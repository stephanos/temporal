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
- Match the exact Task `.1` profile identity/digest/capabilities and expose no address, namespace, credential, executable, or arbitrary option input.
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

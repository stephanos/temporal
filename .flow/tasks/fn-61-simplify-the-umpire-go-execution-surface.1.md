---
satisfies: [R1, R5]
---
# fn-61-simplify-the-umpire-go-execution-surface.1 Establish the root resident executor facade

## Description
Create the deep construction and execution seam for R1 before moving any existing implementation packages. The facade should compose the current attached Temporal factory, Nexus adapter, portable executor, and provenance authority while exposing only the plan executor.

**Size:** M
**Files:** `tools/umpire/umpire.go`, `tools/umpire/umpire_test.go`, `tools/umpire/executor/portable_executor.go`, `tests/umpire4_testenv_test.go`, `tests/umpire4_portable_grpc_executor_test.go`
**Touches:** [tools/umpire/umpire.go, tools/umpire/umpire_test.go, tools/umpire/executor/portable_executor.go, tests/umpire4_testenv_test.go, tests/umpire4_portable_grpc_executor_test.go]

### Approach
- Add the root constructor over the existing `PortableExecutor.Execute` seam; accept only the attached Temporal authority and model-provenance verifier that the caller truly owns.
- Keep a private composition seam for focused fault-injection tests without exporting runner adapters or runtime types.
- Migrate one direct execution and the tagged `testcore.NewEnv` gRPC proof to construct the same facade, covering an external plan plus trusted positive and negative model plans.
- Retain single-flight, poisoning, result authority, and canonical transport mapping in their current owners during this proof task.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/portable_executor.go:79-181` — existing deep execution seam and lifecycle
- `tools/umpire/executorgrpc/server.go:17-58` — transport-independent executor interface
- `tools/umpire/temporal/local/attached.go:18-83` — caller-owned authority boundary
- `tools/umpire/temporal/nexus/runner.go:11-79` — adapter composition to hide
- `tests/umpire4_portable_grpc_executor_test.go:33-107` — self-hosted disposable-cluster proof

**Optional** (reference as needed):
- `tests/umpire4_testenv_test.go:23-45` — current testcore environment assembly

### Key context
Model plans require an independent host-configured provenance verifier; hiding it by trusting plan-carried provenance would change authority and is forbidden. Preserve typed-nil handling for the attached authority and verifier boundary.

### Acceptance
- [ ] Direct Go and gRPC integration paths use the same facade and `Execute(PortableTestPlan)` contract.
- [ ] Normal construction exposes no runner adapter, input binding, runtime authority, participant, or raw Evidence type.
- [ ] External, trusted passing, trusted failing, malformed, forged-provenance, overlap, cancellation/deadline, and cleanup-poisoning behavior remains covered at the appropriate facade or internal test seam.
- [ ] No proto/Lean/generated API, trust policy, concurrency, or existing comment changes.
- [ ] Focused root, executor, gRPC, and tagged integration tests pass with `-tags test_dep` plus `integration` where required.

## Acceptance
- [ ] The root facade proves R1 without leaking the adapter graph.
- [ ] Focused direct and tagged gRPC tests retain exact portable behavior.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

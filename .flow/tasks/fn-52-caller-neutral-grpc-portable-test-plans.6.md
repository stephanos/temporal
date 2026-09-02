---
satisfies: [R4, R8, R9, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.6 Expose and qualify the bounded gRPC executor

## Description
Add the thin gRPC adapter over the deep executor, qualify it against a disposable Temporal cluster, and reconcile architecture/operator documentation for R4, R8-R10. Fn-29 remains the owner of production canary implementation.

**Size:** M
**Files:** `tools/umpire/executorgrpc/**`, `tests/umpire4_portable_grpc_executor_test.go`, `tools/umpire/portableevaluation/README.md`, `tools/umpire/runevaluation/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [tools/umpire/executorgrpc/**, tests/umpire4_portable_grpc_executor_test.go, tools/umpire/portableevaluation/README.md, tools/umpire/runevaluation/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Implement only the generated unary Execute method and translate pre-admission failures to the specified canonical gRPC codes.
- Preserve typed post-admission results, server-side cleanup after client cancellation, single-flight admission, and permanent poisoning after uncertain cleanup.
- Rework the fn-28 disposable-cluster pattern through a real generated gRPC client; cover one external-authored plan and the Lean-generated normal/negative controls.
- Exercise 10x concurrency, deadline/cancellation, malformed input, forged provenance, N/N+1 bounds, crossed Evidence, and no-automatic-retry behavior.
- Document gRPC as the caller-neutral successor while preserving HTTP v1 documentation as historical/current compatibility.
- Document the fn-29 handoff: a protected controller pins and provenance-validates one Lean plan before invoking this interface; public Temporal gRPC remains a distinct downstream seam.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/executorhttp/handler.go:20-170` — existing bounded transport adapter
- `tests/umpire4_portable_executor_test.go:32-120` — disposable-cluster executor proof
- `tools/umpire/portableevaluation/README.md:109-153` — current HTTP and decision contract
- `model/ARCHITECTURE.md:310-328` — current portable path
- `.flow/specs/fn-29-bounded-production-canary-execution-and.md:233-268` — protected canary entry and workflow

**Optional** (reference as needed):
- `model/Umpire/ARCHITECTURE.md:499-554` — reusable portable contract architecture

### Acceptance
- [ ] Generated clients call the unary gRPC executor and receive the specified result/status split.
- [ ] Disposable-cluster tests prove external plan-local pass, Lean model-scoped pass, trustworthy fail, closure failures, cancellation cleanup, and fresh run isolation without a Lean runtime.
- [ ] Ten-call overlap dispatches once and returns bounded pre-I/O failures for the rest; poison and deadline behavior are deterministic.
- [ ] Fn-28 HTTP tests and bytes remain unchanged and passing.
- [ ] Architecture, operator, runtime, and canary-handoff docs describe both interfaces and claim scopes without stale Lean-only assertions.
- [ ] `make proto`, focused unit/integration tests, `make lint-model`, `make umpire-check-regression`, and `make lint-code` pass.

## Acceptance
- [ ] R4 gRPC behavior, R8 compatibility, R9 canary handoff, and R10 documentation/tests are complete.
- [ ] All focused and aggregate commands pass.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

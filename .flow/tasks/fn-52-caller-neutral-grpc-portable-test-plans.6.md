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
Implemented the generated unary gRPC adapter as a thin bounded transport over the resident portable executor and qualified it with one tagged `testcore.NewEnv` integration test using a real generated client/server. The proof covers external plan-local pass, trusted model-bound pass/fail, closure failure, fresh isolation, ten-call overlap, malformed/forged/crossed input, N/N+1, cancellation/deadline cleanup, poison, and no automatic retry without Lean or nested test spawning.

The approved task5 dependency repair adds the checked local-profile adapter prerequisites to Lean-generated portable plans while filtering those adapter-only prerequisites from the fn-18 artifact projection. The task5 fixtures were regenerated, a Lean assertion and Go artifact-equivalence guard were added, and fn-28 descriptors, generated messages, HTTP code, fixtures, and integration proof remain byte-identical and operational.

All task-owned post-review gates passed: proto generation, focused Lean and Go tests, the tagged disposable-cluster integration, model lint (236/236), Umpire regression (270 jobs), scoped non-mutating Go lint (0 issues), and fn-28 identity/unit/integration checks. `make lint-code` reproduces the exact pre-edit inherited baseline of 1379 issues (220 errcheck, 5 exhaustive, 211 forbidigo, 5 govet, 798 revive, 136 staticcheck, 4 testifylint); its formatter side effect was reversed. The local Darwin cgo header issue was safely remediated for Go tests with Xcode `clang` and `SDKROOT`.

baseline: red (`make lint-code` failed pre-edit with the same 1379 inherited issues; canonical Go commands initially hit the inherited PATH-selected Lean clang Darwin-header failure, safely remediated with Xcode `CC`/`SDKROOT`)

stage: impl-review - ran [2026-09-04T00:26:27Z..2026-09-04T00:32:47Z] (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7811e0dccf2be387aed872aec101443f0ffde5d2
- Tests: RED: model/Temporal/Tool/PortableEvaluationContractTests.lean rejected missing checked local-profile prerequisites before the approved task5 dependency repair, make proto (pass; unrelated generated formatting drift reversed), cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests (pass), CC="$(xcrun --find clang)" SDKROOT="$(xcrun --show-sdk-path)" go test -count=1 -tags test_dep ./tools/umpire/testplan/... ./tools/umpire/executor/... ./tools/umpire/executorgrpc/... ./tools/umpire/portableevaluation/... (pass), CC="$(xcrun --find clang)" SDKROOT="$(xcrun --show-sdk-path)" go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableGRPCExecutor$' (pass), make lint-model (pass: 236/236), make umpire-check-regression (pass: 270 jobs), make lint-code (inherited baseline red: 1379 issues — 220 errcheck, 5 exhaustive, 211 forbidigo, 5 govet, 798 revive, 136 staticcheck, 4 testifylint), CC="$(xcrun --find clang)" SDKROOT="$(xcrun --show-sdk-path)" .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules,test_dep,integration' --timeout 10m --fix=false --new-from-rev=26c9d73283f1d384411fba6290f1c75fe1417c0c --config=.github/.golangci.yml ./tools/umpire/executor/... ./tools/umpire/executorgrpc/... ./tests/... (pass: 0 issues), git diff --exit-code 26c9d73283f1d384411fba6290f1c75fe1417c0c -- proto/internal/temporal/server/api/umpire/v1/message.proto api/umpire/v1/message.pb.go api/umpire/v1/message.go-helpers.pb.go tools/umpire/evaluationcontract tools/umpire/executor/testdata tools/umpire/executorhttp tests/umpire4_portable_executor_test.go (pass), CC="$(xcrun --find clang)" SDKROOT="$(xcrun --show-sdk-path)" go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/executorhttp/... (pass), CC="$(xcrun --find clang)" SDKROOT="$(xcrun --show-sdk-path)" go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$' (pass), impl-review codex:gpt-5.6-sol:high (SHIP; 0 introduced findings, 0 pre-existing findings; R4/R8/R9/R10 met)
- PRs:
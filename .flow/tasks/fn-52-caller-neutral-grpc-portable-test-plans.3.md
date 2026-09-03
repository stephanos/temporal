---
satisfies: [R3, R6, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.3 Enforce provenance and result claim scopes

## Description
Implement the external versus model-compiled provenance seam and make result authority explicit for R3 and R6. The plan carries bindings, while the executor host supplies independent trust verification.

**Size:** M
**Files:** `tools/umpire/testplan/**`, `tools/umpire/executor/**`, focused provenance fixtures/tests
**Touches:** [tools/umpire/testplan/**, tools/umpire/executor/**]

### Approach
- Add a narrow host-injected provenance verifier; the semantic plan cannot supply trust anchors or credentials.
- Admit external plans directly as plan-local and require exact verified bindings for model-bound scope.
- Reject missing, invalid, expired, unsupported, or crossed requested model provenance before runtime I/O; never downgrade it silently.
- Carry validated provenance, claim scope, Known Gaps, and unresolved external obligations into every result.
- Ensure required external obligations prevent complete model-bound success without pretending the executor performed them.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/executor.go:48-143` — resident admission and result lifecycle
- `tools/umpire/evaluationcontract/validate.go` — binding validation patterns
- `proto/internal/temporal/server/api/umpire/v1/message.proto:328-343` — current model-bound contract fields
- `.flow/specs/fn-29-bounded-production-canary-execution-and.md:193-231` — canary provenance and claim separation

**Optional** (reference as needed):
- `tools/umpire/artifact/set.go` — artifact identity and relation checks

### Acceptance
- [ ] Any client can submit a valid external plan and receive only plan-local scope.
- [ ] Only independently validated exact model provenance permits model-bound scope.
- [ ] Forgery, trust-anchor injection, expiry, mismatch, missing verifier, and downgrade mutations fail before I/O.
- [ ] Required and advisory external obligations have distinct deterministic effects and remain visible in results.
- [ ] Provenance validation and result-scope mutation tests pass with `-tags test_dep`.

## Acceptance
- [ ] R3 caller-neutral authority and R6 provenance safety are complete.
- [ ] Claim scope and obligations cannot be forged, dropped, or silently downgraded.
- [ ] Focused tests pass.

## Done summary
Implemented host-verified portable-plan authority: external plans remain plan-local, model plans require exact independently verified provenance, plan-owned result scope cannot be forged by producers, and unresolved required model obligations downgrade success to inconclusive. Updated the committed roadmap progress to 3 of 6 tasks complete with task 4 next.

Baseline: repository-wide Go and integration Quick commands were inherited red because downstream executorgrpc/integration work does not exist yet and the local Darwin cgo toolchain cannot find stddef.h; make proto, the Lean contract build, make lint-model, and make umpire-check-regression were green. Literal make lint-code initially exhausted disk and, after remediation, exposed the repository's inherited 1,379-finding backlog; changed-package read-only lint is green with zero issues.

Environment remediation: `go clean -cache` was used twice to recover space exhausted by repository-wide Go analysis.

Verification note: the post-change aggregate regression's portable-evaluation path passed, then its whole-repository promotion snapshot raced with concurrent Flow file creation; the failed promotion sub-check and the remaining vocabulary sub-check both passed immediately when run against stable state. The global plan-index check is currently blocked by those concurrently authored, not-yet-registered fn-53 through fn-58 records; fn-52 validation is green with zero warnings.

stage: impl-review - ran [2026-09-03T19:12Z..2026-09-03T19:15:13Z] (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b00d2903af6c0189ac5d5a16e5c756b02b64fa7c
- Tests: make proto, cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/testplan/... ./tools/umpire/executor/..., make lint-model, make umpire-check-regression (portable-evaluation path green; promotion repository-status snapshot raced with concurrent Flow writes), make umpire-check-promotion, make umpire-check-legacy-vocabulary, make lint-code (inherited red: 1379 repository findings; no task-file findings), CGO_ENABLED=0 .bin/golangci-lint-v2.13.1 run --verbose --build-tags disable_grpc_modules,,test_dep, --timeout 10m --fix=false --new-from-rev=c6617d70468d81eb7d9a1cc9155c2120f4c9fb1c --config=.github/.golangci.yml ./tools/umpire/testplan/... ./tools/umpire/executor/..., make umpire-check-plan-index (concurrent red: unregistered fn-53 through fn-58 records), flowctl validate --spec fn-52-caller-neutral-grpc-portable-test-plans --json, git diff --check
- PRs:
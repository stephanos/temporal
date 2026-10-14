---
satisfies: [R5, R6, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.4 Execute and evaluate typed plans through existing modules

## Description
Adapt the existing deep executor, bounded runner, Evidence closure, and portable evaluator to consume the typed plan for R5-R6 without creating a second semantic evaluator.

**Size:** M
**Files:** `tools/umpire/executor/**`, `tools/umpire/runner/**`, `tools/umpire/portableevaluation/**`, `tools/umpire/internal/artifactv2/**`, focused runtime tests
**Touches:** [tools/umpire/executor/**, tools/umpire/runner/**, tools/umpire/portableevaluation/**, tools/umpire/internal/artifactv2/**]

### Approach
- Project the typed execution program into the existing runner adapter seam instead of teaching callers to orchestrate phases.
- Reuse the existing Observation, link, Property, work-charge, Evidence-Link, and decision implementations; support direct plan-trace clauses when no model link is present.
- Preserve explicit source closure, causal/source-local ordering, run/plan correlation, independent statuses, and cleanup poisoning.
- Enforce hard maxima and charge work before execution/evaluation.
- Charge every variable result item before append. If a complete result would exceed the admitted result budget, discard partial semantic success and return the reserved typed inconclusive result with a result-byte-limit diagnostic; never truncate an accepted Evidence Link.
- Keep runtime endpoints, credentials, and environment selection exclusively in the configured adapter.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runner/runner.go:83-180` — bounded orchestration seam
- `tools/umpire/executor/executor.go:82-143` — deep execution interface
- `tools/umpire/portableevaluation/evaluator.go:15-120` — existing evaluator entry point
- `tools/umpire/portableevaluation/observation.go` — Evidence admission and trace construction
- `tools/umpire/internal/artifactv2/artifact.go:27-150` — current execution-plan representation

**Optional** (reference as needed):
- `tools/umpire/executor/executor_test.go` — single-flight and poisoning tests

### Acceptance
- [ ] External and model-compiled typed plans execute through one runner/evaluator pipeline and return scoped detailed results.
- [ ] Complete positive and trustworthy negative Evidence produce pass/fail; missing, conflicted, unsupported, unclosed, or cleanup-uncertain cases are inconclusive.
- [ ] Cross-plan/run/source Evidence, post-closure records, N+1 work/output bytes, cancellation, and adapter capability mutations fail at the responsible seam.
- [ ] Runtime result N fits exactly; N+1 returns the bounded typed incomplete result without partial semantic output, truncated Evidence Links, or a transport/internal error.
- [ ] Ten concurrent calls cannot queue or dispatch more than the admitted single flight; uncertain cleanup poisons reuse.
- [ ] No new evaluator, environment selector, credential surface, automatic retry, or persistent state is introduced.
- [ ] Focused runner, executor, and portable-evaluation tests pass with `-tags test_dep`.
## Acceptance
- [ ] R5 uses the existing runner/evaluator with all independent statuses intact.
- [ ] R6 failure, result-envelope, concurrency, cancellation, and cleanup bounds hold.
- [ ] Focused tests pass.
## Done summary
Implemented caller-neutral portable-plan execution through the existing admitted artifact runner and portable evaluator, including direct trace projection, exact work/result charging, scoped outcomes, single flight, cancellation, and cleanup poisoning. Runtime binding slots now remain typed through the checked adapter seam, result overflow preserves run correlation, and runner invariants cannot fabricate semantic results.

External and model-verified plans share one execution path. Focused tests cover positive and trustworthy negative decisions, unclosed/crossed Evidence, exact N/N+1 work and result bytes, slot resolution and preconditions, adapter mutation, cancellation, ten-call contention, and poisoned reuse.

baseline: red (canonical Go and integration commands require the later `executorgrpc`/integration tasks and local Darwin cgo lacks `stddef.h`; literal `make lint-code` had the same inherited 1379-issue repository baseline before edits)

Verification: focused CGO-disabled testplan/runner/executor/portableevaluation suite passed; focused non-mutating golangci passed with 0 issues; `make umpire-check-regression` passed with 270 jobs; `make lint-model` passed with 265/236 jobs. Final canonical Go/integration commands reproduced only the inherited missing `executorgrpc` and Darwin cgo failures. Final literal `make lint-code` reproduced exactly 1379 inherited issues (220 errcheck, 5 exhaustive, 211 forbidigo, 5 govet, 798 revive, 136 staticcheck, 4 testifylint); its formatter side effect was restored.

Review fixes were captured in `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md`.

stage: impl-review - ran [2026-09-03T20:35:22Z..2026-09-03T20:55:03Z] (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 30492bd589365faf7e126581ae2d98b776498106, 37bf487dc5df6cefefc7303fef111ccc1f644d4c
- Tests: make proto (baseline green; no task4 proto source changes), cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests (baseline green; no task4 Lean source changes), CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/testplan/... ./tools/umpire/runner/... ./tools/umpire/executor/... ./tools/umpire/portableevaluation/..., CGO_ENABLED=0 .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/testplan/... ./tools/umpire/runner/... ./tools/umpire/executor/... ./tools/umpire/portableevaluation/..., make lint-model, make umpire-check-regression, INHERITED_BASELINE_RED: go test -count=1 -tags test_dep ./tools/umpire/testplan/... ./tools/umpire/executor/... ./tools/umpire/executorgrpc/... ./tools/umpire/portableevaluation/... (later-task executorgrpc package absent; local Darwin cgo stddef.h unavailable; scoped packages pass with CGO_ENABLED=0), INHERITED_BASELINE_RED: go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableGRPCExecutor$' (local Darwin cgo stddef.h unavailable; integration surface belongs to later task), INHERITED_BASELINE_RED: make lint-code (1379 unchanged repository-wide issues: errcheck 220, exhaustive 5, forbidigo 211, govet 5, revive 798, staticcheck 136, testifylint 4; focused lint 0 issues)
- PRs:
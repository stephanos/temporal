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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

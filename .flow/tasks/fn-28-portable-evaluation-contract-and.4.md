---
satisfies: [R3, R4, R8]
---

# fn-28-portable-evaluation-contract-and.4 Interpret portable Observation, link, and Property clauses in Go
## Description

Implement the generic bounded Go interpreter that consumes one admitted contract plus closed Raw Evidence and produces the existing detailed Run Evaluation dimensions without invoking Lean.

**Size:** L
**Files:** `tools/umpire/portableevaluation/**`
**Touches:** [`tools/umpire/portableevaluation/**`]

### Approach
- Normalize only contract-declared fields, validate source/correlation/causal and source-local order, enforce closure, construct the exact System trace, apply the bundled link, and evaluate the bundled Property clauses.
- Preserve `satisfied`, `violated`, `unknown`, `conflict`, and `unsupported`; never turn missing closure, ambiguity, unsupported data, or deadline into success or a false violation.
- Implement only the parent spec's exact version-one operator table, tagged types, diagnostic mapping, canonical evaluation order, and precharged work accounting; keep every operator switch exhaustive and fail closed on unknown values.

### Investigation targets

**Required** (read before coding):
- Parent spec, contract proto/admission, and existing `runevaluation` protocol/result validation.
- `Umpire.Observation.Evaluation`, `Umpire.Observation.Check`, and caller-closure Run Evaluation tests.
- Existing Evidence Link, disposition, causal-order, source-closure, and Limit representations.

## Acceptance
- [ ] Focused fixtures cover every version-one operator's success/failure/type/missing/N/N+1 branches plus accepted/satisfied, accepted/violated, unknown, conflict, unsupported, and incomplete closure without Lean.
- [ ] Every accepted Model Fact and clause retains auditable Evidence Links and exact contract bindings.
- [ ] Mutation, N/N+1, cancellation, race, fuzz, and lint checks pass without adding a second model registry.

## Done summary
Implemented the bounded portable Go evaluator for Observation, exact Implementation Link, and Property clauses, including exact expected-run/source-closure correlation, destination capability validation, auditable Evidence Links, deterministic work accounting, and fail-closed result preflighting. Added focused success, failure, mutation, N/N+1, cancellation, race, fuzz, and lint coverage; the cumulative executor command remains inherited-red because task .6 has not created `tools/umpire/executor` yet, while the known regression and repository-wide lint debts remain unchanged.

stage: impl-review - ran [2026-09-01T20:07:48Z..2026-09-01T20:23:48Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b26171754eb445bc4bce8877a4e00533e68f7dc0, de6c6824f4deddb51933ccad03310e797dd08d41, 4b987fa54b76ee86aa9900221b86fc12d95a4499
- Tests: make proto, cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/..., go test -race -count=1 -tags test_dep ./tools/umpire/portableevaluation/..., go test -tags test_dep ./tools/umpire/portableevaluation -run '^$' -fuzz '^FuzzEvaluateFailsClosed$' -fuzztime=3s, go vet -tags test_dep ./tools/umpire/portableevaluation/..., .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/portableevaluation/..., go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$', make lint-model, INHERITED_RED: go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... - future task .6 has not created tools/umpire/executor; evaluationcontract and portableevaluation pass, INHERITED_RED: make umpire-check-regression - KnownGaps.lean:296 pre-edit baseline, INHERITED_RED: make lint-code - 1373 repository-wide pre-edit findings, GATE_SKIPPED:unittest:green-receipt 645f481e - baseline reused from prior post-gate pass
- PRs:

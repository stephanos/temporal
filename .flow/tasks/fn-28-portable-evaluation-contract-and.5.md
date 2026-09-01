---
satisfies: [R2, R3]
---

# fn-28-portable-evaluation-contract-and.5 Prove portable evaluator parity with Lean Run Evaluation
## Description

Generate checked normal and duplicate-delivery contract fixtures and prove the Go interpreter agrees with the existing Lean Run Evaluation on every stable detailed status, trace coordinate, clause verdict, Evidence Link, Known Gap, and semantic outcome.

**Size:** L
**Files:** `tools/umpire/portableevaluation/testdata/**`, `tools/umpire/portableevaluation/parity_test.go`, `model/Temporal/Tool/PortableEvaluationContractTests.lean`, `Makefile`
**Touches:** [`tools/umpire/portableevaluation/testdata/**`, `tools/umpire/portableevaluation/parity_test.go`, `model/Temporal/Tool/PortableEvaluationContractTests.lean`, `Makefile`]

### Approach
- Produce deterministic protobuf fixtures through the Lean ProtoJSON plus structural Go packer path; never hand-author expected semantic clauses in Go.
- Compare stable semantic meaning rather than transport timestamps, run IDs, task queues, or raw trace bytes.
- Prove Lean/Go parity for every version-one operator branch and work boundary, then cross-mutate each accepted contract/evidence pair and require both paths to fail closed or reach the same non-success class.

### Investigation targets

**Required** (read before coding):
- Existing normal/duplicate Run Evaluation tests, generated CI portability test, and semantic outcome renderer.
- Parent tasks `.2`–`.4` and existing Make regeneration/drift targets.
- Current checked-in caller-closure input and execution fixtures.

## Acceptance
- [ ] Normal parity is operationally successful, Observation accepted, link applied, Property satisfied, with matching stable outcome.
- [ ] Duplicate-delivery parity is operationally successful, Observation accepted, link applied, Property violated only for uniqueness, with matching stable outcome.
- [ ] Regeneration is deterministic and a drift check prevents stale or hand-edited protobuf fixtures.
- [ ] Every approved operator, operand type, missing/type-error branch, canonical order rule, and exact work N/N+1 boundary has direct Lean/Go parity coverage.

## Done summary
Generated deterministic normal, duplicate-delivery, any-operator, branch, and Run Evaluation fixtures and proved the portable Go evaluator against production Lean semantics or explicit Lean boundary proofs for every approved v1 operator, type/missing/status branch, correlation case, and exact N/N+1 work boundary. Canonical ProtoJSON, duplicate normalization, optional correlation, natural count lowering, candidate selection, and runtime Known Gap handling were aligned; fixture drift is now enforced by the regression target.

Focused Go, fixture drift, and Lean lint gates pass. Inherited reds remain unchanged: the aggregate Go command awaits task .6's `tools/umpire/executor`, `make umpire-check-regression` reaches the known `KnownGaps.lean:296` failure after the new drift gate passes, and repository-wide `make lint-code` retains its pre-existing 1373 findings/resource stall.

stage: impl-review - ran [2026-09-01T21:37:21Z..2026-09-01T22:29:27Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0f542bdd430603bb503508df8114f3e3d4b14a3f, 1bb20f448566e47c28bc2edbdb6730968d196bd6, 909aa856cf7f173891f85e23196f8572eb6afd22
- Tests: make proto (baseline green), cd model && mise exec -- lake env lean Temporal/Tool/PortableEvaluationContractTests.lean, make umpire-check-portable-evaluation-fixtures, go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/..., go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$', make lint-model, INHERITED_RED: go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... - future task .6 has not created tools/umpire/executor; touched packages pass, INHERITED_RED: make umpire-check-regression - new portable fixture drift prerequisite passes, then pre-existing KnownGaps.lean:296 fails, INHERITED_RED: make lint-code - pre-edit repository-wide 1373 findings/resource stall; not repeated by conductor instruction
- PRs:

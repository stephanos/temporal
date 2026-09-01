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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

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
Finalized task `.5` after auditing implementation commits `0f542bdd430603bb503508df8114f3e3d4b14a3f`, `1bb20f448566e47c28bc2edbdb6730968d196bd6`, and `909aa856cf7f173891f85e23196f8572eb6afd22` plus the authoritative Codex SHIP review over `cf098fb1f84dd14252a0b73192c879798c916590..909aa856cf7f173891f85e23196f8572eb6afd22`. The isolated portable fixture drift check passed twice with serialized Lean/checker execution, so the earlier truncated JSON/EOF did not reproduce; the only warranted change makes three parity-test switches explicitly exhaustive and clears all task-package lint findings.

Normal and duplicate-delivery status/trace/link/clause/Known Gap parity, every v1 operator/type/missing branch, canonical order/crossed pairs, and exact evaluation-work N/N+1 limits remain covered and green. The unrelated user modifications in `config/development.yaml` and `schema/elasticsearch/visibility/index_template_v7.json` remain untouched.

baseline: red (repository-wide `make lint-model` built all 203 targets and was then killed with inherited exit 137; the relevant Lean target and all focused task gates passed; bare `make proto` initially lacked `protoc`, then the repository-pinned `mise exec -- make proto` completed successfully)

GATE_CLASSIFY_FULL: unrelated user-owned `config/development.yaml` working-tree modification

stage: impl-review - ran [2026-09-02T02:30:04Z..2026-09-02T02:31:39Z] (SHIP; 0 introduced and 0 pre-existing findings)
## Evidence
- Commits: 18911eead68f29b26ad9e88c2e7422c70ec7179a
- Tests: TOOLING_RECOVERY: bare make proto failed before generation because protoc was absent from PATH; mise exec -- make proto passed, cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests (pass), go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... (pass pre-edit and post-review), go test -count=1 -tags test_dep integration ./tests -run ^TestUmpirePortableCanaryExecutor$ (pass; no matching tests at this task revision), INHERITED_RED: make lint-model built 203 targets, then was killed with exit 137, make umpire-check-portable-evaluation-fixtures with workspace TMPDIR/GOTMPDIR/TEST_TELEMETRY_DIR and flock .flow/tmp/lake-build.lock (pass twice; no truncated JSON/EOF), go test -count=1 -tags test_dep ./tools/umpire/portableevaluation -run focused parity matrix (pass), go test -race -count=1 -tags test_dep ./tools/umpire/portableevaluation/... (pass), .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/portableevaluation/... (pass: 0 issues after three inherited task-local revive findings were fixed), .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml --new-from-patch .flow/tmp/fn28_5_tmp/task5-final.patch ./tools/umpire/portableevaluation/... (pass: 0 issues), GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification, NO_RECEIPT: gate receipt was not warrantable while unrelated user-owned config/development.yaml remained dirty, AUTHORITATIVE_REVIEW_SHIP: cf098fb1f84dd14252a0b73192c879798c916590..909aa856cf7f173891f85e23196f8572eb6afd22, FINALIZATION_REVIEW_SHIP: 14c71cd69b93277be430a3cea5f059cc7bd0b626..18911eead68f29b26ad9e88c2e7422c70ec7179a
- PRs:
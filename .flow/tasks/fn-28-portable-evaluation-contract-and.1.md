---
satisfies: [R1]
---

# fn-28-portable-evaluation-contract-and.1 Define the protobuf evaluation-contract vocabulary
## Description

Add the smallest versioned protobuf vocabulary that can carry one closed per-test evaluation contract and its detailed/local result, then reconcile the normative Umpire terms and rules with that portable interpreter seam.

**Size:** L
**Files:** `proto/internal/temporal/server/api/umpire/v1/message.proto`, `api/umpire/v1/**`, `proto/image.bin`, `.plans/UMPIRE4_SPEC.md`
**Touches:** [`proto/internal/temporal/server/api/umpire/v1/message.proto`, `api/umpire/v1/**`, `proto/image.bin`, `.plans/UMPIRE4_SPEC.md`]

### Approach
- Encode the parent spec's exact version-one operator table, tagged operand/result types, diagnostic mapping, canonical evaluation order, and per-operator work accounting without embedding arbitrary code or environment selectors.
- Follow the repository's internal proto package/path convention and generate ordinary Go bindings with `make proto`.
- Add stable Umpire terms/rules stating that the contract is model-derived portable data, not a second behavioral authority, and that unknown or unsupported semantics cannot pass locally.

### Investigation targets

**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md` and `.plans/LEAN_GUIDELINES.md`.
- Existing `proto/internal/temporal/server/api/*/v1` packages and generated `api/*/v1` output.
- Existing `artifactv2` bindings, Limits, Known Gaps, Observation, link, verdict, and Result shapes.

## Acceptance
- [ ] The proto covers exactly the approved per-test execution/evaluation vocabulary and operator table and has no arbitrary extension or whole-world claim surface.
- [ ] Unknown fields/enum/operator semantics can be detected and rejected by strict admission.
- [ ] `make proto`, focused proto lint, and Umpire spec checks pass with generated outputs included.

## Done summary
Finalized the existing versioned, closed Umpire protobuf evaluation-contract vocabulary from commit 4fb4b7658e8a442fd52455ead88897d0f0a3b6e3 after confirming its schema, generated bindings, and normative specification are byte-identical to the authoritative SHIP-reviewed implementation. Focused protobuf, Lean, and Go gates are green; repository-wide lint/model/regression failures were reproduced pre-edit and remain inherited outside this task.

The authoritative implementation review is SHIP for 1794943014bc2fb572c9925e3cfbd8722694a91f..4fb4b7658e8a442fd52455ead88897d0f0a3b6e3. The finalization-base review saw an intentionally empty a012c95b3178dc02636dd9968270a214e4a88f6c..HEAD diff and returned scope-only NEEDS_WORK with zero code findings; no product change was warranted.

baseline: red (environment/repo-wide inherited failures; focused task gates green)

stage: impl-review - ran [2026-09-02T01:44:00Z..2026-09-02T01:44:53Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: red (environment/repo-wide inherited failures; focused task gates green), make proto, cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/..., go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$', go test -count=1 -tags test_dep ./api/umpire/v1, INHERITED_RED: make lint-model (exit 137 after successful model build; environment OOM), INHERITED_RED: make umpire-check-regression (later-task portable parity fixture generation returned truncated JSON), INHERITED_RED: make lint-code (1948 pre-existing repository-wide findings), GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification
- PRs:

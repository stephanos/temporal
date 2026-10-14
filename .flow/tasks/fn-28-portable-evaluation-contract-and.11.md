---
satisfies: [R7, R8]
---

# fn-28-portable-evaluation-contract-and.11 Document portable canary decisions and deferred fleet boundaries
## Description

Synchronize the Umpire architecture, generated contract workflow, resident executor interface, disposable-cluster test, eventual Evidence closure, local decision semantics, and explicitly deferred whole-world/fleet work.

**Size:** M
**Files:** `tools/umpire/portableevaluation/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`, `docs/**`
**Touches:** [`tools/umpire/portableevaluation/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`, `docs/**`]

### Approach
- Explain that Lean compiles contracts ahead of time while the canary independently executes and evaluates one exact contract without Lean.
- Keep detailed statuses separate and document the conservative `pass`/`fail`/`inconclusive` mapping, explicit Evidence closure, and stable-vs-dynamic comparison fields.
- State that fleet scheduling, leases, persistence, crash recovery, production deployment, release eligibility, Claim Assessment, and whole-world claims remain deferred.

### Investigation targets

**Required** (read before coding):
- Parent spec and completed implementation behavior.
- Existing Umpire architecture, glossary, component map, roadmap, and Run Evaluation documentation.
- Exact test commands and generated fixture/drift workflow.

## Acceptance
- [ ] Documentation matches the shipped protobuf schema, compiler, executor interface, HTTP transport, tagged test, statuses, Limits, and failure behavior exactly.
- [ ] The roadmap no longer names the obsolete external staging blocker and accurately gates P3 on the completed self-hosted portability proof.
- [ ] Docs explicitly exclude whole-world and deferred production/fleet claims; documentation tests, links, plan index, and global Flow validation pass.

## Done summary
Documented the shipped portable Evaluation Contract end to end: Lean ahead-of-time compilation, deterministic protobuf packing and admission, the fixed Go interpreter, explicit bounded Evidence closure, resident execution, bounded HTTP protobuf transport, independent statuses, conservative local decisions, parity fields, fixture workflow, tagged disposable-cluster proof, and explicit whole-model/fleet/production/release exclusions. Updated the Umpire architecture, component inventory, delivery roadmap, Run Evaluation handoff, and plan-authority index so fn-28 is the completed self-hosted portability prerequisite rather than an obsolete external-staging blocker.

Verification passed for documentation links, plan-index validation, global Flow validation, portable fixture drift, the focused documentation test, Lean model lint, and task-scoped no-fix Go lint. The pre-edit baseline retained two unrelated repository-wide failures: `make umpire-check-regression` stops at `model/Umpire/SemanticInventory/KnownGaps.lean:296` because a reusable Umpire artifact names `Temporal.Tool.RunEvaluation`, and `make lint-code` reports 1,373 pre-existing findings; the task-scoped alternatives are green and this documentation-only diff introduced no executable changes.

stage: impl-review - ran [2026-09-02T06:50Z..2026-09-02T06:55:22Z] (Codex SHIP; 0 introduced and 0 pre-existing findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c31bb299d8705ddadc7da449489464c5e1257e90
- Tests: BASELINE_GREEN: make proto, BASELINE_GREEN: cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, BASELINE_GREEN: go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/..., BASELINE_GREEN: go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$', make lint-model, make umpire-check-portable-evaluation-fixtures, go test -count=1 -tags test_dep ./tools/umpire/regression -run '^TestHermeticCIDocumentationStatesBoundedClaim$', test -f tools/umpire/portableevaluation/README.md && test -f proto/internal/temporal/server/api/umpire/v1/message.proto, make umpire-check-plan-index, flowctl validate --all --json (0 errors; 203 inherited warnings), .bin/golangci-lint-v2.13.1 run --build-tags test_dep --timeout 10m --fix=false --new-from-rev=5f4bd5b19eb30d2c9f546381e84f07305f8af894 --config=.github/.golangci.yml ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... ./tools/umpire/executorhttp/..., git diff --check, INHERITED_RED: make umpire-check-regression (pre-edit failure at model/Umpire/SemanticInventory/KnownGaps.lean:296: Temporal.Tool.RunEvaluation vocabulary in reusable Umpire artifact), INHERITED_RED: make lint-code (pre-edit repository baseline: 1373 existing findings; task-scoped no-fix lint passed with 0 issues)
- PRs:
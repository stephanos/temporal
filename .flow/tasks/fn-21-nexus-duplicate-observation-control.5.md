---
satisfies: [R5, R6, R7]
---
# fn-21-nexus-duplicate-observation-control.5 Interpret the negative control through Observation Evaluation, Implementation Link, and Property

## Description
### Umpire4 reconciliation (normative)

Interpret the duplicate-observation control only through the existing System Observation -> checked Implementation Link -> Feature Property chain. Preserve operational, realization, observation, Implementation Link, and property outcomes independently.

Task `.4` deliberately retains the truthful raw transport shape: one callback participant fact, one separate synthetic participant-command fact, and the complete six-event history chain. This task owns the existing fn-20 raw-to-semantic adapter projection that coalesces those two participant facts into the one checked duplicate-delivery semantic record, selects the four history records declared by Task `.7`, derives `faultTarget` from the completed real cancellation receipt, and preserves both raw evidence identities in causal/provenance support. The adapter must select this path only from the exact compiled duplicate-delivery program/profile/mapping identities; the normal path remains byte- and behavior-identical.

The existing Run Evaluation composition remains the only semantic authority. Integrate Task `.7`'s checked observed-trace translation at its Implementation Link application seam without creating a second evaluator, weakening the strict conformance API, or treating the observed translation as a Target-authority proof.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Exercise Task `.7`'s already-checked fault-specific mapping through the existing fn-20 Run Evaluation authority for R5/R6/R7. Keep reusable Observation Evaluation, the Property evaluator/declaration, and Go controller semantics unchanged.

**Size:** M
**Files:** `model/Umpire/Observation/Check.lean`, `model/Umpire/Observation/Tests/Check.lean`, `model/Temporal/Tool/RunEvaluation/Protocol.lean`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`, `model/Temporal/Tool/RunEvaluationMutationTests.lean`, `tools/umpire/runevaluation/run_evaluation.go`, `tools/umpire/runevaluation/protocol.go`, `tools/umpire/runevaluation/integration_test.go`, `tools/umpire/runevaluation/mutation_test.go`
**Touches:** [model/Umpire/Observation/Check.lean, model/Umpire/Observation/Tests/Check.lean, model/Temporal/Tool/RunEvaluation/Protocol.lean, model/Temporal/Tool/RunEvaluation.lean, model/Temporal/Tool/RunEvaluationTests.lean, model/Temporal/Tool/RunEvaluationMutationTests.lean, tools/umpire/runevaluation/run_evaluation.go, tools/umpire/runevaluation/protocol.go, tools/umpire/runevaluation/integration_test.go, tools/umpire/runevaluation/mutation_test.go]

### Approach
- Resolve only Task `.7`'s checked mapping/profile from Task `.2`'s compiled configuration and Task `.1`'s ExperimentSpec; project Task `.4`'s distinct raw facts into its exact one-participant/four-history checked semantic bundle, including `faultTarget`, only after every receipt/correlation/causality/closure identity closes. Every identity/source-schema drift follows the parent preflight or unsupported row.
- Reuse one Run Evaluation composition authority for strict conformance and checked observed-trace translation. Preserve the existing strict link behavior for normal count one, expose no authority claim for observed count two, and retain one artifact/result family and one Go controller.
- Require complete Evidence Links/dispositions for delivery true, ownership true, semantic cancellation count two, callback count one, synthetic-contribution count one, and their exact receipt/correlation relation; pass the accepted trace through fn-4 and the unchanged pure caller-closure Property.
- Pin a complete verdict partition in which only the at-most-one cancellation clause is responsible and the overall semantic status is violated; independently assert accepted-outcome identity inputs/exclusions.
- Implement one independent oracle row per parent mutation-table entry: tooling status 1 versus operational failed/incomplete plus semantic unknown/conflict/unsupported, including incomplete Property partition as an output-invariant tooling failure.
- Compare against the unchanged satisfied normal set and rechecking/republishing the same immutable set; do not assert byte/destination identity across separate live executions.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-21-nexus-duplicate-observation-control.7.md` — checked mapping and source-schema contract
- `.flow/tasks/fn-20-local-execution-semantic-conformance.2.md:13-35` — exact Temporal checker seam
- `.flow/tasks/fn-20-local-execution-semantic-conformance.5.md:13-33` — independent cross-layer mutation pattern
- `model/Temporal/Feature/Nexus/CallerClosure.lean:441-462` — unchanged pure Property clauses
- `.flow/specs/fn-21-nexus-duplicate-observation-control.md` — exact mutation/status table

### Acceptance
- [ ] The exact faulted set qualifies to delivery=true, ownership=true, semantic cancellation-count=2 from callback-count=1 plus synthetic-count=1 with complete provenance.
- [ ] Only the uniqueness clause is responsible for semantic `violated`; every other required Property result is resolved.
- [ ] Normal evidence remains accepted/satisfied and the reusable Property/evaluator/API is unchanged.
- [ ] Every mutation matches its exact parent-table owning layer, status, Observation Evaluation, and publication result.
- [ ] No mapping compilation remains in this downstream task and no Go semantic evaluator, second mapper, altered Property, or new artifact family exists.
## Acceptance
- [ ] R5 targeted accepted violation is produced by the existing semantic authority.
- [ ] R6 paired normal/faulted semantic and identity assertions pass.
- [ ] R7 Property, reusable-package, and single-authority boundaries hold.

## Done summary
Implemented the exact duplicate-delivery Run Evaluation path through the existing controller, protocol, Observation Evaluation, Implementation Link, and Property composition authorities.

- Preserved the truthful six-history/two-participant raw set while identity-gating a four-history/one-participant semantic projection with derived completed-cancellation `faultTarget` and both raw participant identities in causal support.
- Reused one private Run Evaluation composition kernel for strict and authority-free observed translation; the normal strict API and canonical output behavior remain unchanged.
- Added exact fault request/response closure gates, rejected mixed normal/fault identities, and retained the existing artifact/result family and Go controller.
- Pinned the accepted uniqueness-only violation, seven-link provenance, idempotent recheck, the full mutation partition, crossed-identity failures, callback correlation absence/drift, control-receipt disposition handling, and excess-field schema rejection.
- Fixed review findings by independently closing both raw participant correlations, separating fault semantic qualification from normal transport validation, and replacing the quadratic participant merge with a fixed-field linear projection.

stage: impl-review - ran [Codex backend], final SHIP

Verification passed: focused and aggregate Lean targets, full `make lint-model` build/replay/lint, tagged RunEvaluation and Nexus Go suites with physical Darwin TMPDIR, and focused RunEvaluation Go lint with zero findings. Full `make lint-code` remains an inherited repository-wide red with 1,374 unrelated findings; its six unrelated auto-edits were exactly inverted and no protected paths were touched.
## Evidence
- Commits: 9d5d861c0a8b57d7bb6669c9e0ffba9249961892, f8d9566ca2503c0d57c38e1f58c78ea4a9798c9c, 30f5bb5592d72dfdb6f71ef60a656cc44ec5ad8a, 95f9878c516b3b76dd463c44f24b5cf2743fc67c
- Tests: mise exec -- lake build Temporal.Tool.RunEvaluationTests Temporal.Tool.RunEvaluationMutationTests Umpire.Observation.Tests.Check, mise exec -- lake build TemporalModelTests, make lint-model, TMPDIR=/private/tmp go test -tags test_dep -count=1 ./tools/umpire/runevaluation/..., TMPDIR=/private/tmp go test -tags test_dep -count=1 ./tools/umpire/temporal/nexus/..., .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules,,test_dep,' --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/runevaluation/...
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)

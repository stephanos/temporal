---
satisfies: [R5, R6, R7]
---
# fn-21-nexus-duplicate-observation-control.5 Interpret the negative control through Observation refinement and Property

## Description
### Umpire4 reconciliation (normative)

Interpret the duplicate-observation control only through the existing System Observation -> checked Refinement -> Feature Property chain. Preserve operational, realization, observation, refinement, and property outcomes independently.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Exercise Task `.7`'s already-checked fault-specific mapping through the existing fn-20 conformance authority for R5/R6/R7. Keep reusable qualification, the Property evaluator/declaration, and Go controller semantics unchanged.

**Size:** M
**Files:** `model/Temporal/Tool/ConformanceTests.lean`, `model/Temporal/Tool/ConformanceMutationTests.lean`, `tools/umpire/conformance/integration_test.go`, `tools/umpire/conformance/mutation_test.go`
**Touches:** [model/Temporal/Tool/ConformanceTests.lean, model/Temporal/Tool/ConformanceMutationTests.lean, tools/umpire/conformance/integration_test.go, tools/umpire/conformance/mutation_test.go]

### Approach
- Resolve only Task `.7`'s checked mapping/profile from Task `.2`'s compiled configuration and Task `.1`'s ExperimentSpec; every identity/source-schema drift follows the parent preflight or unsupported row.
- Require complete derivations/dispositions for delivery true, ownership true, semantic cancellation count two, callback count one, synthetic-contribution count one, and their exact receipt/correlation relation; pass the qualified trace through fn-4 and the unchanged pure caller-closure Property.
- Pin a complete verdict partition in which only the at-most-one cancellation clause is responsible and the overall semantic status is violated; independently assert qualified-outcome identity inputs/exclusions.
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
- [ ] Normal evidence remains qualified/satisfied and the reusable Property/evaluator/API is unchanged.
- [ ] Every mutation matches its exact parent-table owning layer, status, qualification, and publication result.
- [ ] No mapping compilation remains in this downstream task and no Go semantic evaluator, second mapper, altered Property, or new artifact family exists.
## Acceptance
- [ ] R5 targeted qualified violation is produced by the existing semantic authority.
- [ ] R6 paired normal/faulted semantic and identity assertions pass.
- [ ] R7 Property, reusable-package, and single-authority boundaries hold.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

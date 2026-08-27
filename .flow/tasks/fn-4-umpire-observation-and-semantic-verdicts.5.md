---
satisfies: [R1, R2, R3, R4, R5, R6, R7]
---
# fn-4-umpire-observation-and-semantic-verdicts.5 Add independent cross-layer mutation assurance

## Description
Add domain-neutral model, mapping, and property mutations that prove each layer fails at its own boundary for R1-R7.

**Size:** M
**Files:** `model/Umpire/Observation/Tests/Mutations.lean`, `model/Umpire/Observation/Tests.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Observation/Tests/Mutations.lean, model/Umpire/Observation/Tests.lean, model/UmpireTests.lean]

### Approach

- Use record-update mutations at the model, mapping, and property layers.
- Mutate expression operators/types/information-flow labels, evidence bounds, and semantic coordinates independently.
- Assert exact rejecting boundary and reason for each mutant.
- Keep expected traces/derivations literal and independent from implementations under test.
- Assemble focused tests into the reusable root.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Tests/Evaluation.lean:9-13` — mutation pattern.
- `model/Umpire/Property/Tests/Evaluation.lean:45-84` — independent evaluator cases.
- `model/Umpire/Property/Tests/Validation.lean:9-43` — exact error assertions.
- `model/UmpireTests.lean:1-10` — test-root assembly.

## Acceptance
- [ ] Model mutations fail independent qualified-trace comparison.
- [ ] Mapping mutations fail at compilation, qualification, derivation, ordering, or disposition as expected.
- [ ] Clear-value taint, bound limit-plus-one, and missing/duplicate/shifted coordinate mutants fail at their named boundaries.
- [ ] Property mutations alter semantic verdicts without altering qualification.
- [ ] Wrong-layer/shared-oracle controls prove fixture independence.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests UmpireTests` passes.

## Done summary
Added independent literal mutation fixtures spanning model comparison, mapping compilation, qualification bounds, derivation/order/disposition validation, and Property verdicts. Exact boundary reasons, wrong-layer controls, aggregate roots, and all Quick commands pass.

baseline: green
stage: impl-review - ran [2026-08-27T08:28:24Z..2026-08-27T08:40:49Z]
## Evidence
- Commits: 95d19b825c29ce2376a1efb0ac7ead0b1e4bca0c, 21878b3e66f7a89570deb286dabda24c8f908d6e, 431b74f7f8da010a2e2ed23444bdff1456926bb3
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation, cd model && mise exec -- lake build Umpire.Observation.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests, make umpire-check-regression, cd model && mise exec -- lake build Umpire.Observation.Tests UmpireTests
- PRs:
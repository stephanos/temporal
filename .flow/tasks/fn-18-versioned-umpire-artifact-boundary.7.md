---
satisfies: [R1, R5, R8]
---
# fn-18-versioned-umpire-artifact-boundary.7 Add strict coverage-report reading and exact checkpoint persistence

## Description
Implement R5's persistence handoff for fn-17 without redefining coverage state, scoring, or resume behavior.

**Size:** M
**Files:** `model/Umpire/Artifact/Coverage.lean`, `model/Umpire/Exploration/Persistence.lean`, `model/Umpire/Exploration/PersistenceTests.lean`, `tools/umpire/artifact/coverage.go`, `tools/umpire/artifact/coverage_test.go`
**Touches:** [model/Umpire/Artifact/Coverage.lean, model/Umpire/Exploration/Persistence.lean, model/Umpire/Exploration/PersistenceTests.lean, tools/umpire/artifact/coverage.go, tools/umpire/artifact/coverage_test.go]

### Approach
- Reuse fn-17's canonical report projection byte-for-byte and implement strict Go admission/structural/digest validation.
- Define the exact checkpoint wrapper over immutable CoverageState plus report and selected/pinned ExperimentSpec bindings.
- Recompute report/checkpoint/state identities and enforce disjoint partitions, count/set equality, coordinate/goal credit references, budget ceiling, cursor, direct/equivalent interaction, and report/state/spec agreement.
- Keep resume compatibility and coverage meaning delegated to fn-17 after admission.
- Mutate every digest, partition, credit, selected binding, ceiling, cursor, termination, and interaction relationship.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Exploration/State.lean`, `Report.lean`, `Engine.lean`
- fn-17 R3/R5/R6/R7 and exact report/checkpoint handoff
- parent spec `Normative v1 wire contract` coverage schemas

### Acceptance
- [ ] Fn-17 report persisted bytes remain authoritative; Go never sorts, scores, or repairs them.
- [ ] Checkpoint wraps, rather than duplicates, the exact state and all selected/pinned bindings.
- [ ] Structural/digest/reference tampering fails deterministically.
- [ ] Admitted checkpoint values are handed back to fn-17 for semantic resume compatibility.

## Acceptance
- [ ] R5 report/checkpoint persistence is exact and bounded.
- [ ] No second coverage or resume model exists.
- [ ] Cross-language persistence tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

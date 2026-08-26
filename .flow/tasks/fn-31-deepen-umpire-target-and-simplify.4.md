---
satisfies: [R2, R3, R5]
---
# fn-31-deepen-umpire-target-and-simplify.4 Migrate the current Temporal Nexus target families

## Description
Adopt the public Target boundary in the existing Nexus examples and CallerClosure without changing Feature meaning (R2, R3, R5).

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/CallerClosure.lean`, `model/Temporal/Feature/Nexus/CallerClosureTests.lean`, `model/Temporal/Feature/Nexus/Examples/**`
**Touches:** [model/Temporal/Feature/Nexus/CallerClosure.lean, model/Temporal/Feature/Nexus/CallerClosureTests.lean, model/Temporal/Feature/Nexus/Examples/**]

### Approach
- Migrate BasicLifecycle and BasicOperations before CallerClosure.
- Preserve target kernels, properties, behaviors, queries, and canonical artifacts.
- Do not physically split CallerClosure merely to mirror the logical template.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:200-380` — smaller target path
- `model/Temporal/Feature/Nexus/CallerClosure.lean:250-360` — provider/connector assembly
- `model/Temporal/Feature/Nexus/CallerClosure.lean:400-720` — extraction/planner plumbing
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — canonical family contract

### Acceptance
- [ ] All current Nexus targets use the ordinary checked interface.
- [ ] Existing valid/invalid behavior and artifacts remain equivalent.
- [ ] Feature code gains no System or Verify dependency.

## Acceptance
- [ ] R2/R3 are demonstrated by Temporal families.
- [ ] R5 compatibility and regression fixtures pass.
- [ ] No unnecessary physical decomposition or lost comments.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R2, R3, R5]
---
# fn-31-deepen-umpire-target-and-simplify.4 Migrate Temporal Nexus target authors and query consumers

## Description
Adopt the public Target boundary in the BasicLifecycle and CallerClosure target authors and migrate the BasicOperations queries as consumers of the shared BasicLifecycle target, without changing Feature meaning (R2, R3, R5).

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/CallerClosure.lean`, `model/Temporal/Feature/Nexus/CallerClosureTests.lean`, `model/Temporal/Feature/Nexus/Examples/**`
**Touches:** [model/Temporal/Feature/Nexus/CallerClosure.lean, model/Temporal/Feature/Nexus/CallerClosureTests.lean, model/Temporal/Feature/Nexus/Examples/**]

### Approach
- Migrate BasicLifecycle before its BasicOperations query consumers, then migrate CallerClosure.
- Keep BasicOperations bound to `BasicLifecycle.target`; do not introduce a BasicOperations target or duplicate lifecycle semantics.
- Opt BasicLifecycle and CallerClosure into Target-owned finite planning once and preserve their existing role/action-domain compatibility tokens verbatim; downstream Query derivation copies rather than reconstructs them.
- Preserve target kernels, properties, behaviors, queries, and canonical artifacts.
- Do not physically split CallerClosure merely to mirror the logical template.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:250-367` — shared target, completeness, and planner path
- `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean:1-33` — downstream reuse of `BasicLifecycle.target`
- `model/Temporal/Feature/Nexus/CallerClosure.lean:384-416` — declaration, composition, and extraction
- `model/Temporal/Feature/Nexus/CallerClosure.lean:560-686` — completeness, ordering, and planner plumbing
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — canonical family contract

### Acceptance
- [ ] BasicLifecycle and CallerClosure use the ordinary checked Target interface; BasicOperations consumes the migrated BasicLifecycle target through the public Query/Planning path.
- [ ] Existing valid/invalid behavior and artifacts remain equivalent.
- [ ] Existing BasicLifecycle and CallerClosure role/action-domain token strings and canonical Query JSON remain byte-identical.
- [ ] No duplicate BasicOperations target or target-owned query meaning is introduced.
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

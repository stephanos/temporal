---
satisfies: [R2, R3, R5]
---
# fn-31-deepen-umpire-target-and-simplify.4 Migrate Temporal Nexus target authors and query consumers

## Description
Adopt the public Target boundary in the Nexus Lifecycle and Experimental CallerClosure target
authors and migrate all Operations queries as consumers of the shared Lifecycle target, without
changing Feature meaning (R2, R3, R5).

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Lifecycle.lean`, `model/Temporal/Feature/Nexus/LifecycleTests.lean`, `model/Temporal/Feature/Nexus/Operations.lean`, `model/Temporal/Feature/Nexus/OperationsTests.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Lifecycle.lean, model/Temporal/Feature/Nexus/LifecycleTests.lean, model/Temporal/Feature/Nexus/Operations.lean, model/Temporal/Feature/Nexus/OperationsTests.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean]

### Approach
- Migrate Lifecycle before its Operations query consumers, then migrate Experimental CallerClosure.
- Keep Operations bound to `Lifecycle.target`; do not introduce an Operations target or duplicate lifecycle semantics.
- Opt Lifecycle and Experimental CallerClosure into Target-owned finite planning once and preserve their existing role/action-domain compatibility tokens verbatim; downstream Query derivation copies rather than reconstructs them.
- Preserve target kernels, properties, behaviors, queries, and canonical artifacts.
- Do not physically split CallerClosure merely to mirror the logical template.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Lifecycle.lean:295-419` — shared target, completeness, and planner path
- `model/Temporal/Feature/Nexus/Operations.lean:1-45` — downstream reuse of `Lifecycle.target`
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean:384-418` — declaration, composition, and extraction
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean:560-704` — completeness, ordering, and planner plumbing
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean` — canonical family contract

### Acceptance
- [ ] Lifecycle and Experimental CallerClosure use the ordinary checked Target interface; Operations consumes the migrated Lifecycle target through the public Query/Planning path.
- [ ] Existing valid/invalid behavior and artifacts remain equivalent.
- [ ] Existing Lifecycle and Experimental CallerClosure role/action-domain token strings and canonical Query JSON remain byte-identical.
- [ ] AsyncStart, Cancellation, and SuccessfulCompletion migrate without a duplicate Operations target or target-owned query meaning.
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

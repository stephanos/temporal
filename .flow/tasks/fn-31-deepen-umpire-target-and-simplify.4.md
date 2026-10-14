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
Migrated Nexus Lifecycle and Experimental CallerClosure onto Target-owned finite planning through `AuthoredTarget`/`checkedTarget`, with Query completeness and Planning kernels derived through the public seams. Operations remains a consumer of the single shared `Lifecycle.target` for AsyncStart, Cancellation, and SuccessfulCompletion; existing semantic behavior, exact compatibility tokens, invalid cases, and CallerClosure artifacts are preserved.

Codex review found that the first Operations canonical-JSON assertions were tautological. The fix adds three checked-in golden Query fixtures—the only review-proven expansion beyond the six declared files—and now pins each simple lifecycle query byte-for-byte. The unrelated dirty `.plans/UMPIRE4_SPEC.md` and `.flow/memory/declined/generated-api-drift-verification.md` remain untouched; green gate receipts were not warrantable while the plan file stayed dirty. Project memory capture was attempted after NEEDS_WORK→SHIP but skipped because memory is not initialized.

baseline: green
stage: impl-review - ran [2026-08-27T04:40:42Z..2026-08-27T04:46:23Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 984554d5b7276e07be53da2f56707da8c1b3605e, bf44cef0094d3af5cff77b2b9aae72937074da2f
- Tests: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model
- PRs:

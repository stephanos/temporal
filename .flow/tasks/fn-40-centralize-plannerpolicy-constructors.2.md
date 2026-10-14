---
satisfies: [R2, R3]
---
# fn-40-centralize-plannerpolicy-constructors.2 Migrate authored PlannerPolicy consumers

## Description
Migrate the remaining authored Umpire and Temporal policy consumers to the canonical interface while preserving generic and deliberate exceptional policies (R2, R3).

**Size:** M
**Files:** `model/Umpire/Planning/Tests/Fixtures.lean`, `model/Umpire/Planning/Tests/Artifacts.lean`, `model/Umpire/Examples/Switch.lean`, `model/Umpire/Tests/MigrationCompatibility.lean`, `model/Temporal/Feature/Nexus/Lifecycle.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`
**Touches:** [model/Umpire/Planning/Tests/Fixtures.lean, model/Umpire/Planning/Tests/Artifacts.lean, model/Umpire/Examples/Switch.lean, model/Umpire/Tests/MigrationCompatibility.lean, model/Temporal/Feature/Nexus/Lifecycle.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean]

### Approach
- Replace ordinary shortest/exhaustive record literals with the canonical values, standardizing the Switch and Lifecycle policies from seeds 23/29 to 17.
- Adapt parameterized Planning fixtures around the constructors without losing arbitrary `SearchStrategy`, seed, breadth-first, or non-default identity coverage.
- Keep MigrationCompatibility's strategy-changing record update explicit because it intentionally preserves its base seed.
- Preserve all existing comments and verify that shortest/exhaustive selected traces remain unchanged.
- Use a focused repository search to distinguish prohibited default magic-number literals from deliberate seed mutations.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Tests/Fixtures.lean:282-360` — generic strategy/seed helpers
- `model/Umpire/Planning/Tests/Artifacts.lean:9-80` — seed identity and checksum checks
- `model/Umpire/Examples/Switch.lean:554-572` — seed-23 shortest policy
- `model/Umpire/Tests/MigrationCompatibility.lean:330-350` — deliberate representation update
- `model/Temporal/Feature/Nexus/Lifecycle.lean:519-530` — seed-29 shared policy
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean:652-694` — repeated seed-17 policies

### Key context
- Seeds remain part of Query identity for all strategies, so the source migration intentionally makes downstream canonical fixtures stale until the next task refreshes the complete owned surface.

### Acceptance
- [ ] Ordinary model policies use the canonical constructors and no unexplained seed-17/23/29 record literal remains.
- [ ] Generic breadth-first/arbitrary-seed fixtures and deliberate seed/strategy record updates remain tested.
- [ ] Shortest/exhaustive planner outcomes are unchanged despite the canonical identity migration.
- [ ] Existing comments remain intact.
- [ ] Focused Umpire Planning, Switch source, Lifecycle, and CallerClosure Lean targets compile; golden-fixture suites, including Operations tests, are deferred to Task 3 after fixture refresh.
## Acceptance
- [ ] Authored caller migration satisfies R2 without narrowing generic fixture coverage.
- [ ] Seed-standardized policies preserve non-seeded traversal behavior.
- [ ] Focused non-golden Lean targets compile with existing comments preserved; golden-fixture suites are explicitly deferred to Task 3.
## Done summary
Centralized authored planner-policy consumers on the canonical constructors, standardizing Switch and Lifecycle on seed `17` while preserving generic arbitrary-strategy/seed, breadth-first, explicit compatibility-update, and seed-identity coverage. Focused non-golden Lean builds pass; broad Quick gates expose only the intentionally stale identity/checksum/generated-view fixtures assigned to task 3, plus the inherited pre-edit regression-vocabulary findings.

stage: impl-review - ran [2026-09-02T22:37:10Z..2026-09-02T22:39:39Z] | SHIP (codex)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3990afab7ed27c7237862560d51253b9e1a0a3e8
- Tests: baseline: green — cd model && mise exec -- lake build UmpireTests, baseline: green — cd model && mise exec -- lake build Umpire.Examples.Switch Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, baseline: red (make umpire-check-regression failed pre-edit only on model/Umpire/ARCHITECTURE.md:439 and model/Umpire/SemanticInventory/KnownGaps.lean:296 Temporal-vocabulary findings; inherited and owner-waived in task 1), baseline: green — make lint-model, cd model && mise exec -- lake build Umpire.Planning.Tests.Artifacts Umpire.Planning.Tests.Outcomes Umpire.Planning.Tests.Enumeration Umpire.Examples.Switch Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.Experimental.CallerClosureTests (green: 37 jobs), cd model && mise exec -- lake build UmpireTests (expected task-3 fixture staleness only: Umpire.ExecutionHandoffTests, Umpire.Artifact.Tests.Runtime, Umpire.Artifact.Tests.Codecs, Umpire.Examples.SwitchTests exact Query/artifact/checksum bytes), cd model && mise exec -- lake build Umpire.Examples.Switch Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests (expected task-3 fixture staleness only: AsyncStartTests, CancellationTests, SuccessfulCompletionTests Query bytes and Operations.PlanningTests artifact bytes), make umpire-check-regression (expected task-3 fixture staleness: switch.query.exact-action fixture artifact checksum differs from inspector output; inherited vocabulary findings are downstream and were recorded at baseline), make lint-model (expected task-3 fixture staleness only: Operations AsyncStart/Cancellation/Planning/SuccessfulCompletion, VariationSpace, NexusDiscovery, Exploration, Switch, Artifact Codecs, ExecutionHandoff, and Artifact Runtime tests; linter stops because their object files are intentionally stale)
- PRs:

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

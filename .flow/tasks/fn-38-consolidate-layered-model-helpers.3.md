---
satisfies: [R3, R5]
---
# fn-38-consolidate-layered-model-helpers.3 Migrate planning and observation Umpire fixtures

## Description
Adopt the established `Umpire.Shared.Test` seam in the Query, Planning, and Observation fixture families for R3. Keep this separate from Task `.2` so each cohesive concern group has a bounded review surface.

**Size:** M
**Files:** `model/Umpire/Query/Tests/Fixtures.lean`, `model/Umpire/Planning/Tests/Fixtures.lean`, `model/Umpire/Observation/Tests/Fixtures.lean`
**Touches:** [model/Umpire/Query/Tests/Fixtures.lean, model/Umpire/Planning/Tests/Fixtures.lean, model/Umpire/Observation/Tests/Fixtures.lean]

### Approach
- Replace duplicate ID, source, and compatible metadata constructor bodies with delegation to `Umpire.Shared.Test`.
- Preserve Query/Planning canonical behavior parameters and Observation's simpler metadata shape instead of forcing one overloaded default set.
- Leave target definition lists, error projections, model values, and checked concern fixtures in their current modules.
- Retain existing fixture names and imports so downstream tests do not migrate.
- Preserve all comments around semantic fixtures and defaults.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Tests/Fixtures.lean:9-37` — repeated constructors with canonical behavior.
- `model/Umpire/Planning/Tests/Fixtures.lean:9-37` — matching planning constructor shape.
- `model/Umpire/Observation/Tests/Fixtures.lean:10-25` — differing metadata signature that must remain explicit.
- `model/Umpire/Shared/Test.lean` — helper contract established by Task `.2`.

**Optional** (reference as needed):
- `model/Umpire/Observation/Tests.lean` — focused fixture consumer behavior.

## Acceptance
- [ ] Query, Planning, and Observation fixture consumers compile through their existing module paths and names.
- [ ] Only identical constructor mechanics are shared; differing metadata defaults and concern data remain explicit.
- [ ] Existing comments and observable fixture values are unchanged.
- [ ] `cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Umpire.Observation.Tests` passes.

## Done summary
Routed the Query, Planning, and Observation fixture constructors through `Umpire.Shared.Test` while preserving every existing declaration name, source path, canonical behavior, concern-specific documentation/default, comment, and observable fixture value. Focused and aggregate builds, model lint, and the regression gate pass.

stage: impl-review - ran (SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1bd59b4b3406712e5fb9ea78ade6571d36fe36ab
- Tests: cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Umpire.Observation.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, make lint-model, make umpire-check-regression
- PRs:

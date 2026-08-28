---
satisfies: [R1, R3, R5]
---
# fn-38-consolidate-layered-model-helpers.2 Centralize core Umpire test fixtures

## Description
Create the Umpire test-support seam and migrate the Target, Behavior, and Property fixture families for R3. This group shares foundational authoring concerns and can move without coupling the planning/observation fixture group.

**Size:** M
**Files:** `model/Umpire/Shared/Test.lean`, `model/Umpire/Target/Tests/Fixtures.lean`, `model/Umpire/Behavior/Tests/Fixtures.lean`, `model/Umpire/Property/Tests/Fixtures.lean`
**Touches:** [model/Umpire/Shared/Test.lean, model/Umpire/Target/Tests/Fixtures.lean, model/Umpire/Behavior/Tests/Fixtures.lean, model/Umpire/Property/Tests/Fixtures.lean]

### Approach
- Import the narrow production helper/core modules from `Umpire.Shared.Test`; do not import concern facades or Temporal modules.
- Centralize ID, source, and metadata constructor mechanics only where signatures and defaults agree.
- Parameterize the Target-specific path and digest/default differences rather than normalizing them silently.
- Keep semantic fixtures, definition lists, and concern-specific checked values in their current fixture namespaces.
- Preserve existing fixture names for their test consumers, using local delegation where that avoids caller churn.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Tests/Fixtures.lean:9-26` — path-parameterized source and metadata defaults.
- `model/Umpire/Behavior/Tests/Fixtures.lean:9-25` — repeated constructor shape with Behavior-owned data.
- `model/Umpire/Property/Tests/Fixtures.lean:9-39` — repeated constructors and definition list.
- `model/Umpire/Core.lean:56-70` — exact metadata and source field contracts.

**Optional** (reference as needed):
- `model/Umpire/TargetTests.lean` — focused Target aggregate root.

## Acceptance
- [ ] All three fixture families reuse `Umpire.Shared.Test` only for the compatible constructor mechanics.
- [ ] Target-specific paths/defaults and concern-specific fixture data remain unchanged and owner-local.
- [ ] Existing fixture consumer imports and names compile unchanged.
- [ ] `cd model && mise exec -- lake build Umpire.TargetTests Umpire.Behavior.Tests Umpire.Property.Tests` passes.

## Done summary
Added `Umpire.Shared.Test` and routed the Target, Behavior, and Property fixture constructors through it while preserving every existing fixture name, source path, semantic default, comment, and consumer import. Focused and aggregate builds, model lint, and regression gates pass; the lint retry required only sequential cache warming after a virtiofs/Lake output race and no source change.

stage: impl-review - ran (SHIP)
## Evidence
- Commits: bf40691fcd41a024450314261f4f81ea3275fc1f
- Tests: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Behavior.Tests Umpire.Property.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, make lint-model, make umpire-check-regression
- PRs:
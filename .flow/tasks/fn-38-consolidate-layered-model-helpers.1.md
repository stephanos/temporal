---
satisfies: [R1, R2, R5]
---
# fn-38-consolidate-layered-model-helpers.1 Extract source-compatible Umpire construction helpers

## Description
Create the Umpire-owned production helper seam and use the Switch example as the compatibility proof for R1, R2, and R5. Keep the helper interface small enough that Temporal can reuse Umpire value construction without importing example semantics.

**Size:** M
**Files:** `model/Umpire/Shared.lean`, `model/Umpire/Examples/Switch.lean`, `model/Umpire/Examples/SwitchTests.lean`
**Touches:** [model/Umpire/Shared.lean, model/Umpire/Examples/Switch.lean, model/Umpire/Examples/SwitchTests.lean]

### Approach
- Add `Umpire.Shared` with the narrowest Umpire core import that provides Definition ID, Source Location, and Definition Metadata types.
- Extract only the repeated construction shape; keep caller-specific paths, positions, documentation, canonical behavior, version, and defaults as explicit inputs.
- Retain `Umpire.Examples.Switch` declarations at their current names and visibility, delegating private construction through the shared seam.
- Preserve all existing comments and keep `Umpire.Shared` out of the production umbrella unless an existing consumer contract requires it.
- Extend focused Switch assertions only where existing coverage does not pin source and metadata fidelity.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:56-70` — owning Umpire value types and field contracts.
- `model/Umpire/Examples/Switch.lean:5-12` — repeated ID and public source construction.
- `model/Umpire/Examples/Switch.lean:54-62` — repeated metadata constructor and defaults.
- `model/Umpire/Examples/SwitchTests.lean` — focused example behavior and source assertions.

**Optional** (reference as needed):
- `model/Umpire/Tests/MigrationCompatibility.lean:59-61` — existing facade-level compatibility assertion.

## Acceptance
- [ ] `Umpire.Shared` compiles with no Temporal import and exposes only the proven reusable construction seam.
- [ ] Existing Switch public names, types, visibility, source values, metadata, and behavior remain unchanged.
- [ ] Existing comments and doc comments remain present and accurate.
- [ ] `cd model && mise exec -- lake build Umpire.Examples.SwitchTests` passes.

## Done summary
Added the narrow `Umpire.Shared` construction seam and routed Switch's existing private/public facades through it without changing source, metadata, serialized fixtures, behavior, comments, or consumer imports. Added complete value-level Switch metadata compatibility coverage; focused and aggregate Lean builds, model lint, and regression gates pass.

stage: impl-review - ran (SHIP)
## Evidence
- Commits: 5a20ffe1e7e6d74ba0a4fc0f1ce2dda46762e936
- Tests: cd model && mise exec -- lake build Umpire.Examples.SwitchTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, make lint-model, make umpire-check-regression
- PRs:
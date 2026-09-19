---
satisfies: [R1, R2, R4, R5]
---
# fn-38-consolidate-layered-model-helpers.4 Extract Temporal construction behind existing facades

## Description
Create the Temporal production helper seam and migrate the repeated Nexus construction for R2, R4, and R5. Existing feature modules continue to own their public declarations and semantic definitions.

**Size:** M
**Files:** `model/Temporal/Shared.lean`, `model/Temporal/Feature/Nexus/Lifecycle.lean`, `model/Temporal/Feature/Nexus/Operations.lean`, `model/Temporal/Feature/Nexus/Observation.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`
**Touches:** [model/Temporal/Shared.lean, model/Temporal/Feature/Nexus/Lifecycle.lean, model/Temporal/Feature/Nexus/Operations.lean, model/Temporal/Feature/Nexus/Observation.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean]

### Approach
- Add `Temporal.Shared` for Temporal-specific path/source construction and compose `Umpire.Shared` for Umpire-owned value construction.
- Keep feature semantics, definition lists, and public `source` declarations in their current namespaces; delegate only private/repeated mechanics.
- Preserve each module's exact source path, line/column values, metadata defaults, and canonical behavior text.
- Use narrow leaf imports and avoid any dependency on `Temporal.System` or test-support modules.
- Do not create `Temporal.Shared.Test`; current test candidates remain single-owner or configuration-specific.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Lifecycle.lean:9-16,69-77` — public source and repeated metadata construction.
- `model/Temporal/Feature/Nexus/Operations.lean:10-17` — public source and ID wrapper.
- `model/Temporal/Feature/Nexus/Observation.lean:16-27` — source and synthetic definition ownership.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean:9-16,127-135` — experimental facade and matching metadata constructor.
- `model/Umpire/Shared.lean` — lower-layer seam established by Task `.1`.

## Acceptance
- [ ] Lifecycle, Operations, Observation, and Caller Closure retain their existing public declarations, types, visibility, and import paths.
- [ ] Source locations, metadata, identity, canonical behavior, and generated/serialized outputs remain unchanged.
- [ ] `Temporal.Shared` imports neither `Temporal.System` nor test modules, and no broad `Temporal.Shared.Test` is introduced.
- [ ] `cd model && mise exec -- lake build TemporalModelTests TemporalExperimentalTests` passes.

## Done summary
Added the internal `Temporal.Shared` construction seam over `Umpire.Shared` and routed the Lifecycle, Operations, Observation, and experimental Caller Closure facades through it without moving their public declarations or feature meaning. Source identity, metadata defaults, canonical behavior, comments, pretty-printed fixtures, and generated/serialized values remain unchanged; focused and aggregate builds, model lint, and regression pass.

stage: impl-review - ran (SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 254da06dfd4c01795c59aeb69b29d8e9207346c0
- Tests: baseline: green via handoff (green verified at 1bd59b4b3 by fn-38-consolidate-layered-model-helpers.3), git diff --check, cd model && mise exec -- lake build Temporal.Shared Temporal.Feature.Nexus.Lifecycle Temporal.Feature.Nexus.Operations Temporal.Feature.Nexus.Observation Temporal.Feature.Nexus.Experimental.CallerClosure, cd model && mise exec -- lake build TemporalModelTests TemporalExperimentalTests, Temporal.Shared boundary scan: imports only Umpire.Shared; no Temporal.Shared.Test exists, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, make lint-model, make umpire-check-regression
- PRs:

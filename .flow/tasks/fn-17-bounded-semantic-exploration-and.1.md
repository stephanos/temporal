---
satisfies: [R1]
---
# fn-17-bounded-semantic-exploration-and.1 Rename Query seeded strategy and define bounded exploration requests

## Description
Hard-rename the Query seed-rotation strategy to `seeded`, then define the closed checked inputs shared by bounded exhaustive and uncovered-coordinate selection.

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Planning/Engine.lean`, repository-wide Query callers and identity fixtures, `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/Core.lean`, `model/Umpire/Exploration/Language.lean`, `model/Umpire/Exploration/Tests/Validation.lean`
**Touches:** [model/Umpire/Query/**, model/Umpire/Planning/**, model/Temporal/**, model/Umpire/Exploration.lean, model/Umpire/Exploration/**, model/UmpireTests.lean, model/TemporalModelTests.lean]

### Approach
- Rename Query `SearchStrategy.coverageGuided` and canonical `coverage-guided` to `seeded` / `seeded` as a hard cut, preserving the existing deterministic seed-rotation behavior.
- Update repository-wide constructor matches, callers, authored Query values, artifacts, generated views, and canonical identity expectations; add no compatibility alias.
- Define one checked Space source, the `exhaustive | uncoveredCoordinate` policy, an explicit `experiment-specs` Limit, canonical Model Coordinates, pinned input, and structured failures.
- Reject empty/oversized bounds, an unknown guided coordinate, and incompatible pinned contracts before compilation or selection.
- Keep the types pure and free of runtime, persistence, campaign, promotion, or alternate-source fields.
- Preserve existing comments in every touched file.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:8-20` — current misleading Query strategy name.
- `model/Umpire/Planning/Engine.lean:300-318` — seed consumption semantics that must remain unchanged.
- `.flow/tasks/fn-40-centralize-plannerpolicy-constructors.1.md` — required post-rename constructor handoff.
- `model/Umpire/Search.lean` — established Limit and strategy patterns.
- `model/Umpire/Space/Language.lean` — checked Space and coordinate vocabulary.
- `model/Umpire/Artifact.lean` — canonical ExperimentSpec identities.

## Acceptance
- [ ] Query-level `seeded` retains the prior deterministic traversal, every repository caller and canonical identity is updated, and the old constructor/text spelling is rejected without an alias.
- [ ] The checked request exposes exactly the retained two policies and one finite Space source.
- [ ] Invalid Limits, coordinates, and pinned bindings fail with canonical typed errors.
- [ ] Focused validation tests pass and existing comments remain intact.

## Done summary
Introduced the checked bounded-exploration request boundary, typed experiment-spec limits, structural coordinate validation, canonical pinned-plan validation, and the hard-renamed seeded query strategy. Focused validation, aggregate model builds, and Lean lint pass; the absent downstream Selection/Session/Nexus modules and unrelated Go lint error in tools/umpire1/monitor_test.go remain inherited baseline failures.

stage: impl-review - ran
## Evidence
- Commits: 8afb493106016d7fe0814e172cfbe26857708e9d
- Tests: cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation, cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation Umpire.Property.Tests.Validation Umpire.Query.Tests.Identity, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, baseline: red (cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection failed pre-edit: downstream module absent), baseline: red (cd model && mise exec -- lake build Umpire.Exploration.Tests.Session failed pre-edit: downstream module absent), baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests failed pre-edit: downstream module absent), GOLANGCI_LINT_FIX=false make lint-code (inherited failure: tools/umpire1/monitor_test.go:17,19 undefined v1)
- PRs:
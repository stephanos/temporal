---
satisfies: [R2, R4, R5, R7]
---
# fn-39-make-the-temporal-nexus-feature-model.3 Mirror and name Lifecycle tests

## Description
Reorganize Lifecycle tests along the semantic/target seam and give every current assertion a descriptive declaration name (R2, R4, R5, R7). Retain the original aggregate test import.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/LifecycleTests.lean`, `model/Temporal/Feature/Nexus/Lifecycle/SemanticsTests.lean`, `model/Temporal/Feature/Nexus/Lifecycle/TargetTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/LifecycleTests.lean, model/Temporal/Feature/Nexus/Lifecycle/*Tests.lean, model/TemporalModelTests.lean]

### Approach
- Move transition and terminal-state assertions to `SemanticsTests`; move target composition, identity, finite-planning, and provider-failure assertions to `TargetTests`.
- Keep `LifecycleTests.lean` as the stable aggregate and preserve `compatibilityTargetAuthors` in its existing public namespace.
- Replace each anonymous `example` with a name that states the behavior or compatibility property without changing its proposition or proof.
- Preserve command-based checks, public fixture values, and every existing comment.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/LifecycleTests.lean:8-67` — lifecycle behavior, identity, target, and planning assertions.
- `model/Temporal/Feature/Nexus/LifecycleTests.lean:74-135` — missing/conflicting provider fixtures and assertions.
- `.flow/tasks/fn-38-consolidate-layered-model-helpers.7.md:13-37` — predecessor compatibility inventory and test requirements.
- `model/TemporalModelTests.lean:1-23` — stable aggregate imports and compatibility list.

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/OperationsTests.lean:163-204` — nearby aggregate compatibility style.

### Acceptance
- [ ] Stable LifecycleTests import and public compatibility inventory remain unchanged.
- [ ] Every existing Lifecycle assertion is present under a descriptive declaration name in the matching focused test module.
- [ ] Transition negatives, provider errors, identities, canonical target values, and finite planning retain the same coverage.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests TemporalModelTests` passes.

## Acceptance
- [ ] R2, R4, R5, and R7 task-scoped checks pass.
- [ ] No existing assertion or comment is lost.

## Done summary
Mirrored Lifecycle coverage into focused SemanticsTests and TargetTests modules: all 16 anonymous assertions now have descriptive names while the stable aggregate, public fixtures, command checks, propositions, proofs, and comments remain compatible. The exact task build, regression, and model lint pass; the spec-level Temporal.Feature.NexusTests target remains intentionally deferred to fn39.5, and the protected inherited config symlink stat prevented only optional gate-receipt creation.

stage: impl-review - ran [2026-08-28T22:17:19Z..2026-08-28T22:19:49Z]
## Evidence
- Commits: b4ceb7eff791e26c1203241b5159a22d55421ffa
- Tests: baseline: green (cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests TemporalModelTests), baseline: green (make umpire-check-regression), baseline: green (make lint-model), cd model && mise exec -- lake build Temporal.Feature.Nexus.Lifecycle.SemanticsTests Temporal.Feature.Nexus.Lifecycle.TargetTests Temporal.Feature.Nexus.LifecycleTests TemporalModelTests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests TemporalModelTests, make umpire-check-regression, make lint-model, git diff --check, NOT_RUN: spec-level Temporal.Feature.NexusTests target is intentionally introduced by fn39.5; fn39.3 used its exact task acceptance build, NO_RECEIPT: unittest gate receipt was not warrantable because protected inherited config/development.yaml symlink stat kept the worktree dirty
- PRs:
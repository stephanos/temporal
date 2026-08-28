---
satisfies: [R7]
---
# fn-41-finitemachine-target-authoring.6 Run final model gates

## Description
Run final integrated verification after the API, migrations, compatibility tests, and documentation settle (R7). This is a verification-only gate; any failure returns to the task that owns the affected file instead of accumulating unrelated fixes here.

**Size:** S
**Files:** none (verification-only)
**Touches:** []

## Approach
- Run the focused Target, Feature Nexus, System Nexus, Implementation Link, and Temporal aggregate builds from the final tree.
- Run the complete model build, regression, and import/lint gates using the repository entry points.
- Inspect the final diff for unexpected Definition ID, source, fingerprint, Query, plan, Artifact, generated-fixture, or public-declaration changes.
- Audit touched Lean files for preserved existing comments and for forbidden `sorry`/`admit` or newly unjustified trusted proof shortcuts.
- Confirm unrelated pre-existing worktree changes remain outside the task-owned diff.

## Investigation targets
**Required** (read before verification):
- `model/README.md:246-287` — focused and complete model verification commands.
- `Makefile:989-992` — model build entry point.
- `Makefile:1275-1288` — model lint and regression entry points.
- `.plans/LEAN_GUIDELINES.md:191-261` — proof trust, regression, lint, and review gates.

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/Fixtures/` — generator-owned compatibility fixtures that must remain unchanged.
- `model/Umpire/Examples/Fixtures/` — Switch golden fixtures that must remain unchanged.

## Acceptance
- [ ] `cd model && mise exec -- lake build Umpire.Target.Tests.FiniteMachine Umpire.TargetTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests` passes.
- [ ] `make umpire-build-model` passes.
- [ ] `make umpire-check-regression` passes without fixture regeneration.
- [ ] `make lint-model` passes.
- [ ] No `sorry`/`admit`, unjustified trusted proof shortcut, comment loss, generated fixture drift, or unrelated worktree edit is present in the final diff.

## Done summary
Verified the integrated FiniteMachine authoring tree without changing implementation or fixtures. The focused 73-job build, 200-job complete model build, 186-job regression gate, and 158-job model lint all passed; the initial lint attempt was baseline-red only from a transient Lake/virtiofs `Umpire.Planning.c` ENOENT and the exact retry passed after an isolated module warmup.

The final spec diff has no forbidden `sorry`/`admit`, removed Lean comments, fixture drift, or unrelated edits; added `native_decide` uses are confined to executable regression tests. The two known false symlink stats remain untouched, and gate receipt storage was skipped because `config/development.yaml` keeps the worktree dirty.

stage: impl-review - ran (SHIP, zero findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: cd model && mise exec -- lake build Umpire.Target.Tests.FiniteMachine Umpire.TargetTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests, make umpire-build-model, make umpire-check-regression, make lint-model, cd model && mise exec -- lake build Umpire.Planning (tooling warmup after transient baseline ENOENT), git diff --check b9634dcdc49dca812ad14476243a4118f305bf28..HEAD, git diff --name-only b9634dcdc49dca812ad14476243a4118f305bf28..HEAD -- model/Temporal/Feature/Nexus/Fixtures model/Umpire/Examples/Fixtures (no output), git diff b9634dcdc49dca812ad14476243a4118f305bf28..HEAD -- '*.lean' | rg '^\+.*\b(sorry|admit)\b' (no output), added native_decide audit: test files only, removed Lean comment audit: no output, NO_RECEIPT: worktree dirty outside the ignore set (config/development.yaml) - receipt not warrantable
- PRs:

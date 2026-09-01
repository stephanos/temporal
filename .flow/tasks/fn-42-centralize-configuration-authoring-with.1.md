---
satisfies: [R1, R2, R4, R5]
---
# fn-42-centralize-configuration-authoring-with.1 Add ConfigUseSpec and focused validation coverage

## Description
Add the shared typed authoring seam and prove its projection, checking, and proof-extraction contracts (R1, R2, R4, R5). Keep the task independently buildable before either owner module migrates.

**Size:** M
**Files:** `model/Temporal/System/Configuration/Core.lean`, `model/Temporal/System/Configuration/Tests/Fixtures.lean`, `model/Temporal/System/Configuration/Tests/Validation.lean`, `model/Temporal/System/Configuration/Tests/Catalog.lean`
**Touches:** [model/Temporal/System/Configuration/Core.lean, model/Temporal/System/Configuration/Tests/Fixtures.lean, model/Temporal/System/Configuration/Tests/Validation.lean, model/Temporal/System/Configuration/Tests/Catalog.lean]

### Approach
- Add `ConfigUseSpec α` in the namespace of its principal type beside the existing authoring records, with public docstrings and one field for each independently authored expectation.
- Provide semantic projections to `SettingClassification`, `ConfigInterpretation α`, and `ConfigUseDefinition α`; keep the current low-level types available as expert and negative-test seams.
- Route `.check` through `checkConfigUseDefinition` and add a proof-taking `.checked` extractor that does not call `native_decide` internally or create an unchecked fallback.
- Extend shared fixtures with a representative spec and add executable checks for projection equality, successful extraction, key/identity/impact/schema/default/policy drift, opaque-default replacement, and exact diagnostic preservation.
- Preserve all existing comments and keep imports narrower than the `Temporal.System.Configuration` facade.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Configuration/Core.lean:124-157` — current classification, interpretation, and definition records to project
- `model/Temporal/System/Configuration/Core.lean:551-709` — authoritative diagnostics, opaque handling, and checked-definition path to reuse
- `model/Temporal/System/Configuration/Tests/Fixtures.lean:1-58` — shared valid classification/interpretation fixture
- `model/Temporal/System/Configuration/Tests/Validation.lean:11-138` — negative diagnostic style

**Optional** (reference as needed):
- `model/Temporal/System/Configuration/Tests/Catalog.lean:30-98` — opaque-default replacement coverage
- `.plans/LEAN_GUIDELINES.md:34-64` — principal-type deep interfaces and public declaration rules

### Key context
- Authored expected key, identity, schema, and default must remain independent of the current generated setting; no `fromGeneratedSetting` shortcut.
- Existing public checked definitions already use private native-decision witnesses. Keep the proof explicit at owner call sites and avoid widening the shared helper's trust boundary.

### Lifecycle reconciliation

The implementation and its original review remain reachable at commit `e579e0c95baa2aed35f5ddb02f9d6af8c8e6427f`.
This reopened lifecycle run verifies that exact `e579e0c95^..e579e0c95` implementation range rather
than introducing a duplicate source change.

### Acceptance
- [ ] A representative valid spec projects one key/identity consistently and produces the same checked metadata as the existing records.
- [ ] Mutated key, identity, impacts, schema, default, policy, opaque metadata/value, and decoder cases retain the exact existing `ConfigError` outputs.
- [ ] `.checked` requires an explicit success proof and contains no hidden native decision or unchecked construction path.
- [ ] New public declarations have useful Lean docstrings and every existing comment remains intact.
- [ ] `cd model && mise exec -- lake build Temporal.System.Configuration.Tests` passes without new warnings or import-boundary violations.

## Acceptance
- [ ] Typed authoring projections and delegated validation satisfy R1/R2.
- [ ] Positive, drift-negative, opaque-default, and proof-trust checks cover R4/R5.
- [ ] Focused configuration tests pass with comments preserved.

## Done summary
Added the documented `ConfigUseSpec` authoring seam with single-source projections, delegated validation, and proof-only checked extraction. Focused coverage proves projection and checked-metadata parity plus exact full diagnostics for key, identity, impacts, schema, default, policy, opaque replacement, and decoder drift.

baseline: green (`cd model && mise exec -- lake build Temporal.System.Configuration.Tests`; `cd model && mise exec -- lake build TemporalModelTests`; `make umpire-build-model`; `make lint-model`)

stage: impl-review - ran (codex; SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e579e0c95baa2aed35f5ddb02f9d6af8c8e6427f
- Tests: baseline: green (cd model && mise exec -- lake build Temporal.System.Configuration.Tests; cd model && mise exec -- lake build TemporalModelTests; make umpire-build-model; make lint-model), cd model && mise exec -- lake build Temporal.System.Configuration.Tests, cd model && mise exec -- lake build TemporalModelTests, make umpire-build-model, make lint-model
- PRs:

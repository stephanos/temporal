---
satisfies: [R1, R4]
---
# fn-9-umpire-reusable-dsl-package-split.1 Establish the Umpire Core library seam

## Description
Establish the reusable library root and migrate Core plus its tests (R1, R4). This is the early proof point and intentionally leaves the existing tree available for later staged migration.

**Size:** M
**Files:** `model/lakefile.toml`, `model/Umpire.lean`, `model/UmpireTests.lean`, `model/Umpire/Core.lean`, `model/Umpire/CoreTests.lean`
**Touches:** [model/lakefile.toml, model/Umpire.lean, model/UmpireTests.lean, model/Umpire/Core.lean, model/Umpire/CoreTests.lean]

### Approach
- Add the single `Umpire` library and generic test root alongside the existing targets.
- Move the reusable identity, trace, kernel, capability, checked-target, and canonicalization declarations under `namespace Umpire`; preserve existing comments.
- Port the deterministic Core tests without weakening canonical or composition assertions.
- Keep Core free of Temporal and Nexus imports; do not add compatibility aliases.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/Semantics.lean:1-153` — reusable identity, trace, and kernel surface
- `model/Temporal/Experiment/Semantics.lean:218-623` — checked target composition and canonicalization
- `model/Temporal/Experiment/SemanticsTests.lean:389-520` — deterministic composition and canonical tests
- `model/lakefile.toml:1-16` — current library/test target conventions

**Optional** (reference as needed):
- `model/Temporal.lean:1-6` — current aggregate-root imports

### Key context
The intermediate stage is additive so `make umpire-check-regression` remains green. Existing comments must move unchanged unless a path statement would become false.

## Acceptance
- [ ] `Umpire` and `UmpireTests` build the new Core surface independently of Temporal/Nexus.
- [ ] Core declarations and existing comments are preserved under `Umpire.*` with no compatibility alias.
- [ ] Deterministic Core tests retain checked-target, error, canonicalization, and digest coverage.
- [ ] A dependency scan finds no Temporal or Nexus import in the new Umpire files.
- [ ] `make umpire-check-regression` remains green through the additive stage.

## Done summary
Added the single Umpire Lake library seam, moved the reusable Core declarations under `Umpire.*`, and ported the full deterministic Core suite under `UmpireTests` while preserving comments and the additive legacy targets. Baseline was green; the focused Umpire build, forbidden-import scan, and stable regression gate all pass.

stage: impl-review - ran (SHIP; completed 2026-08-25T19:12:18.581566Z)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b7283ffcb3d73ba262f4bae296b0ad07cd05c16b
- Tests: cd model && mise exec -- lake build Umpire UmpireTests, ! rg -n '^import .*(Temporal|Nexus)' model/Umpire.lean model/UmpireTests.lean model/Umpire/Core.lean model/Umpire/CoreTests.lean, make umpire-check-regression
- PRs:

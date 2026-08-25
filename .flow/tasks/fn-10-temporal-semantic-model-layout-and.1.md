---
satisfies: [R2, R3, R7]
---
# fn-10-temporal-semantic-model-layout-and.1 Extract shared and matching configuration modules

## Description
Extract the shared configuration checker/resolver and matching-owned interpretations into the approved System modules (R2, R3, R7). Keep the not-yet-extracted callback declarations in the former combined module temporarily so the intermediate branch remains buildable; task 2 extracts them and converts that module to an import-only bridge.

**Size:** M
**Files:** `model/Temporal/System/Configuration.lean`, `model/Temporal/System/Configuration/Core.lean`, `model/Temporal/System/Matching/Configuration.lean`, `model/Temporal/Umpire/Config.lean`
**Touches:** [model/Temporal/System/Configuration.lean, model/Temporal/System/Configuration/Core.lean, model/Temporal/System/Matching/Configuration.lean, model/Temporal/Umpire/Config.lean]

### Approach
- Mechanically move shared declarations before namespace cleanup so existing comments stay attached.
- Keep generic classification/use types and checkers, validation, resolution, provenance, immutable views, and catalog conformance behind `Temporal.System.Configuration`.
- Move only matching-owned classifications, interpretations, contexts, and typed uses to `Temporal.System.Matching.Configuration`.
- Split the mixed authored classification list explicitly: Matching entries move now, Callback entries stay temporarily for task 2, and only catalog-wide/shared checking stays in Configuration.
- Enforce one-way imports: Matching may import Configuration; Configuration may not import Matching or Callback.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Umpire/Config.lean:11-845` — shared checker, resolver, view, and fixture semantics
- `model/Temporal/Umpire/Config.lean:1089-1243` — mixed authored classifications, interpretations, contexts, and typed uses to partition
- `model/Umpire/Planning.lean:1` — one-line facade convention

**Optional** (reference as needed):
- `model/Temporal/Umpire/ConfigTests.lean:9-370` — current shared resolution coverage

### Acceptance
- [ ] New shared and matching modules compile with the approved namespaces.
- [ ] Configuration retains existing resolution, validation, provenance, and catalog-fixture behavior.
- [ ] Matching-specific declarations have moved; callback-specific declarations remain intact and explicitly staged for task 2.
- [ ] Configuration imports neither Callback nor Matching; Matching depends on Configuration in one direction.
- [ ] Existing configuration regression coverage still builds and passes.
- [ ] Existing comments remain attached to the declarations they explain.
## Acceptance
- [ ] Shared configuration and matching semantics compile under their new namespaces.
- [ ] Focused existing configuration tests pass without semantic changes.
- [ ] Import direction and comment preservation are verified.

## Done summary
Extracted the shared configuration checker/resolver into a deep Temporal.System.Configuration core plus facade and moved Matching-owned classifications, interpretations, contexts, and typed uses into Temporal.System.Matching.Configuration. Callback declarations remain explicitly staged in the former combined module for task 2, with existing configuration coverage and comments preserved; baseline and final regression checks were green.

stage: impl-review - ran [2026-08-25T22:29:47Z..2026-08-25T22:33:05Z]
## Evidence
- Commits: c16ee07873693ccb8699f53190ea179a9288aec4
- Tests: baseline: green (make umpire-check-regression), cd model && mise exec -- lake build Temporal.System.Configuration Temporal.System.Matching.Configuration Temporal.Umpire.Config TemporalUmpireTests, make umpire-check-regression
- PRs:
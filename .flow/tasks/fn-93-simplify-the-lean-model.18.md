---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.18 Shrink Temporal.System.Configuration (B5, decision D5)

## Description
Lane B5. A `ConfigUseSpec` takes `setting := Settings.x` and keeps only owner-authored fields (impacts, sampling, change effect, decode); `SettingClassification`, `ConfigInterpretation` and `ConfigUseDefinition` collapse into it; `matchesSetting`'s restated checks go because `settingIdentity` already pins key, policy, schema, codec and default. The Go regexp schema mirror in `Callback/Configuration.lean` stays.

### Owner decision
- **D5 — shrink rather than retire. Recommended default: shrink.** Alternative: retire the package (~2,800 lines). Record first in the Done summary. If "retire" is chosen instead, delete the package, its tests and the facade export, and add its names to the vocabulary gate.

**Size:** M
**Files:** `model/Temporal/System/Configuration/Core.lean` (1,100; types at ~37, 131, 150, 164; `matchesSetting` ~249-257), `model/Temporal/System/Configuration.lean`, `model/Temporal/System/Callback/Configuration.lean` (545), `model/Temporal/System/Matching/Configuration.lean` (84), tests under `model/Temporal/System/Configuration/Tests/**` and `model/Temporal/System/Callback/ConfigurationTests.lean`, vocabulary gate, `model/README.md:16`
**Touches:** [model/Temporal/System/Configuration/**, model/Temporal/System/Configuration.lean, model/Temporal/System/Callback/**, model/Temporal/System/Matching/**, model/Temporal/System.lean, tools/umpire/internal/retiredvocabulary/**, model/README.md]

### Approach
- Keep `settingIdentity` SHA pins and the Go-schema mirror byte-for-byte.
- Pin before/after: every current configuration test's observable (classification, interpretation, decode result) is reproduced by the collapsed spec; list any test that only checked the removed restatement.
- Collapsed type names enter the vocabulary gate as compound identifiers.

### Investigation targets
**Required:**
- `model/Temporal/System/Configuration/Core.lean:30-260`
- `model/Temporal/System/Callback/Configuration.lean`
- `tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog/render.go:19` — generated doc text naming the package (generator unchanged)

### Quick commands
```sh
cd model && lake build Temporal TemporalModelTests
make umpire-check-goldens umpire-check-retired-vocabulary
```

## Acceptance
- [ ] D5 recorded
- [ ] One `ConfigUseSpec` form; the three collapsed types and `matchesSetting`'s restated checks gone
- [ ] `settingIdentity` pins and the Go-schema mirror unchanged; everything byte-identical (R10)


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R3]
---
# fn-40-centralize-plannerpolicy-constructors.3 Refresh canonical artifacts and generated views

## Description
Refresh the complete canonical fixture/checksum and generated-view surface affected by the seed-17 Query identity migration, then run the full model gates (R3).

**Size:** M
**Files:** affected files under `model/Umpire/Examples/Fixtures/`, `model/Umpire/Examples/testdata/`, `model/Umpire/Artifact/Tests/Fixtures/`, `model/Temporal/Feature/Nexus/Fixtures/`, `model/Umpire/Examples/Generated/`, `model/Umpire/Artifact/Tests/Codecs.lean`, and generated/checksum consumers under `tools/umpire/`
**Touches:** [model/Umpire/Examples/Fixtures/**, model/Umpire/Examples/testdata/**, model/Umpire/Artifact/Tests/Fixtures/**, model/Umpire/Artifact/Tests/Codecs.lean, model/Temporal/Feature/Nexus/Fixtures/**, model/Umpire/Examples/Generated/**, tools/umpire/cmd/umpire-gen-regression-views/**, tools/umpire/regression/**, tools/umpire/internal/artifactv2/**]

### Approach
- Produce fresh canonical Query and ExperimentSpec bytes from the migrated model declarations and update every byte/checksum fixture as a complete owned set.
- Regenerate the managed Go and Markdown regression views with the repository generator; do not hand-maintain generated output.
- Update independent artifact-v2 canonical fixtures and checksum assertions reached by the changed Switch artifact identity.
- Search for obsolete pre-migration fingerprints/checksums to catch consumers outside the obvious fixture directories.
- Run focused byte-equality checks followed by the repository model regression and import/documentation lint gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/SwitchTests.lean:14-18` — canonical Switch Query/artifact fixtures
- `model/Temporal/Feature/Nexus/OperationsTests.lean:17-30` — canonical Lifecycle-derived fixtures
- `model/Umpire/Tests/MigrationCompatibility.lean:214-225` — relocated-layout artifact equality
- `model/Umpire/Artifact/Tests/Codecs.lean:17-29` — v2 canonical byte fixtures
- `Makefile:1019-1030` — regression-view generation/check entry points
- `model/README.md:248-288` — focused and full model verification commands

### Key context
- Changing a meaning-bearing Query seed changes query fingerprints, nested DrivePlan/ExperimentSpec checksums, and generated views even when the selected trace is unchanged. Partial fixture refreshes are invalid.

### Acceptance
- [ ] All checked Query/artifact fixtures contain the post-migration identities and valid recomputed checksums.
- [ ] `make umpire-check-regression-views` reports no generated-view drift after regeneration.
- [ ] No stale pre-migration fingerprint or checksum remains in model/tool consumers.
- [ ] `make umpire-check-regression` and `make lint-model` pass.
## Acceptance
- [ ] Complete canonical fixture and generated-view sets satisfy R3.
- [ ] Focused byte/checksum tests, regression-view drift check, full regression, and model lint pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

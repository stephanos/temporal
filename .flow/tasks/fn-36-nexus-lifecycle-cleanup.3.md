---
satisfies: [R3, R5]
---
# fn-36-nexus-lifecycle-cleanup.3 Rebase the caller closure fixture and generated projections

## Description
Move the caller-closure fixture and update every owned registry/generator/projection path, then regenerate the managed outputs (R3, R5).

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/testdata/**`, `model/Temporal/Feature/Nexus/Experimental/testdata/**`, `Makefile`, `tools/umpire/cmd/umpire-gen-regression-projections/**`, `tools/umpire/regression/**`, `model/Temporal/Tool/Generated/Regressions.md`
**Touches:** [model/Temporal/Feature/Nexus/testdata/**, model/Temporal/Feature/Nexus/Experimental/testdata/**, Makefile, tools/umpire/cmd/umpire-gen-regression-projections/**, tools/umpire/regression/**, model/Temporal/Tool/Generated/Regressions.md]

### Approach
- Move the fixture under Experimental/testdata and update the Makefile registry plus generator catalog/test source paths.
- Regenerate the fixture/projection surfaces from the moved Lean scenario and canonical generator; do not hand-edit generated Go or Markdown outputs.
- Update focused regression assertions that intentionally pin canonical and repository source locations.
- Verify stable semantic identities/digests and expected source-provenance-only artifact changes.

### Investigation targets
**Required** (read before coding):
- `Makefile:125-127` — fixture registry.
- `tools/umpire/cmd/umpire-gen-regression-projections/catalog.go:18-24` — generator-owned catalog input.
- `tools/umpire/cmd/umpire-gen-regression-projections/render_test.go:19-49` — pinned source/fixture output tests.
- `tools/umpire/regression/projection_test.go:1-35` — regression source-path assertions.
- `model/Temporal/Tool/Generated/Regressions.md:1-15` — generated output ownership marker.

### Key context
- Use canonical lowercase `tools/` paths on this case-insensitive workspace.
- Source path is provenance; scenario/query/property declaration identities remain stable.

### Acceptance
- [ ] Fixture exists only under Experimental/testdata and matches inspector output byte-for-byte.
- [ ] Generator catalog/tests and generated Go/Markdown projections use the new source and fixture paths.
- [ ] Regeneration is deterministic and projection checks pass.
- [ ] No semantic declaration identity or semantic digest changes arise solely from the move.

## Acceptance
- [ ] R3 experimental fixture/provenance remains semantically stable.
- [ ] R5 all managed path-bearing outputs are regenerated from owners.
- [ ] Projection and full regression checks pass.
- [ ] No generated output is hand-edited.

## Done summary
Moved the caller-closure JSON fixture under Experimental/testdata, updated its registry and generator-owned source paths, and canonically regenerated Go/Markdown projections. Semantic fingerprint and declaration identities stayed stable; only source provenance paths changed. Deterministic regeneration and inspector byte equality passed. stage: plan-sync - skipped(config: planSync.enabled != true). No commit was created per repository instructions.
## Evidence
- Commits:
- Tests: GOCACHE=<task-cache> make umpire-gen-regression-projections (twice, identical hashes), GOCACHE=<task-cache> make umpire-check-regression-projections, GOCACHE=<task-cache> make umpire-check-regression, cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.CallerClosureTests TemporalExperimentalTests
- PRs:
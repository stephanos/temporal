---
satisfies: [R2, R6, R7]
---
# fn-5-umpire-discovery-promotion-and-artifact.6 Select and render the broader stable regression set

## Description
Replace the one-entry Go manifest with checked catalog selection and publish aggregate projections for R2/R6/R7.

**Size:** M
**Files:** `tools/umpire/internal/generate/regression/catalog.go`, `tools/umpire/internal/generate/regression/generate.go`, `tools/umpire/internal/generate/regression/render.go`, `tools/umpire/internal/generate/regression/generate_test.go`, `tools/umpire/internal/generate/regression/render_test.go`, `model/Umpire/Examples/testdata/switch-experiment-spec.json`, `tools/umpire/regression/catalog_generated_test.go`, `model/Temporal/Tool/Generated/Regressions.md`
**Touches:** [tools/umpire/internal/generate/regression/**, model/Umpire/Examples/testdata/switch-experiment-spec.json, tools/umpire/regression/catalog_generated_test.go, model/Temporal/Tool/Generated/Regressions.md]

### Approach

- Load only the validated generated catalog projection and select the exact `stableRegression` set; resolve every entry through its validated Temporal projection binding, canonical inspector selector, checked-in fixture path, and unique projection key.
- Reuse the existing Switch exact-action fixture without renaming or changing its current semantic artifact.
- Replace per-entry output ownership with one set-level aggregate output configuration, then render one aggregate Go file and one aggregate Markdown file in canonical identity order while retaining the ordinary `RequireProjection` wrapper.
- Validate all candidates before one transactional complete-set publication.
- Provide a non-mutating check path that reuses task `.3`'s shared exact candidate-set comparison seam to compare current aggregate outputs directly.

### Investigation targets

**Required:**
- `tools/umpire/internal/generate/regression/catalog.go:1-76` — one-entry manifest to retire.
- `tools/umpire/internal/generate/regression/render.go:1-140` — current per-record rendering.
- `tools/umpire/internal/generate/regression/generate.go:96-178` — validate-before-publish path.
- `tools/umpire/regression/projection.go:37-105` — thin wrapper and strict fixture contract.
- `model/Temporal/Tool/Inspect.lean:53-77` — current two-scenario registry.
- `model/Umpire/Examples/SwitchTests.lean:44-68` — canonical artifact assertions.

### Quick commands

`go test -count=1 -tags test_dep ./tools/umpire/internal/generate/regression ./tools/umpire/regression`

## Acceptance
- [ ] The selected manifest is derived from the checked catalog and contains exactly the two initial stable identities.
- [ ] Each selected entry resolves through exactly one projection binding; missing, duplicate, stale, or unsafe fixture bindings fail before inspection.
- [ ] Aggregate Go/Markdown paths are owned once at set level; multiple stable entries cannot collide or overwrite the output map.
- [ ] Switch and caller-closure fixtures match canonical inspector bytes and semantic fingerprints.
- [ ] Aggregate Go and Markdown contain both entries once in canonical order.
- [ ] Missing/extra/stale/colliding entries and unsafe paths fail before publication.
- [ ] Publication remains transactional under validation, write, concurrency, and interruption failures.
- [ ] Check mode detects stale or missing aggregate outputs without writing or temporary regeneration.
- [ ] Generated wrappers remain projection-only and use `require` through `RequireProjection`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

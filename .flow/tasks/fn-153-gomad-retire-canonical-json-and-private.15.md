---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.15 Migrate architecture effect fixtures before removing their source owner

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** internal/gomadtool/architecture effect/error-provenance/initialization/mutation/range tests; testdata/architecture effects.json and seven audit regressions (no consumer migration code).
**Touches:** [tools/gomad3/internal/gomadtool/architecture/*_test.go, tools/gomad3/testdata/architecture/effects.json, tools/gomad3/testdata/architecture/audit-regressions/**]

### Approach

- Replace effects_test's real canonical.go purity fixture with a surviving stdlib encoder/strict-decoder behavior fixture; retain positive and negative host-effect/callback detection.
- Rebind synthetic canonicaljson fixture package names and expected callback symbols in error-provenance, initialization, mutation, range and audit cases to actual surviving ownership.
- Behavior pin: each old fixture's accepted/rejected effect/provenance result must match after rebinding; do not delete a regression merely because its package spelling disappeared.
- Do not modify production owner deletion here; retirement follows once every real and synthetic consumer is ready.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/gomadtool/architecture/effects_test.go:223`
- `tools/gomad3/internal/gomadtool/architecture/error_provenance_test.go`
- `tools/gomad3/internal/gomadtool/architecture/initialization_test.go`
- `tools/gomad3/internal/gomadtool/architecture/mutation_test.go`
- `tools/gomad3/internal/gomadtool/architecture/range_test.go`
- `tools/gomad3/testdata/architecture/effects.json`
- `tools/gomad3/architecture_test.go:115`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./internal/gomadtool/architecture

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] All affected fixtures retain their original positive/negative effect and provenance coverage after owner/callback rebinding.
- [ ] No fixture requires the real generic encoder source or advertises removed-package ownership.
- [ ] Architecture test assertions remain meaningful; invalid callback, purity and error-flow mutations are still caught.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

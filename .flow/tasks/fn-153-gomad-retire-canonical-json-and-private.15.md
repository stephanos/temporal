---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.15 Migrate architecture effect fixtures before removing their source owner

## Description
Superseded by fn-165.6's retirement of the old analyzer, with retained host/dynamic fixture coverage migrated under fn-165.4. This annotation does not complete this deferred task or revive Native work; final replacement acceptance belongs to fn-165.

The original analyzer-specific approach below is retained as historical context.

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
Rebound the seven fn165-retained audit-effect fixtures from the retired `internal/canonicaljson` synthetic path to their surviving `helper` package. All 36 effect categories, expected details, pure-file memberships, callback bodies, positive/negative behavior, and initializer coverage remain unchanged. The live architecture and purity consumers pass, and no affected fixture advertises the removed owner.

The parent task's original module-root architecture command is obsolete after fn165 removed that package. Aggregate validation reaches only the known stale `golang-x-sys-v047-darwin-arm64` compatibility approval owned by fn153.17.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 155b8d085c8392b4d9d8f7bb85b99e65c3af7ecc, 7ad83c852e796a9a597014f501478ef77d6587e9
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 -run '^TestArchitectureEffectFixtures$' ./internal/archcheck, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck ./internal/puritycheck, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/hostfs ./world
- PRs:
## Revival trigger (2026-10-13)

Revived after fn153.12 completed with a SHIP implementation review, satisfying the last dependency and freezing the final replay, minimizer, corpus, and shared Runner migrations that architecture fixtures could still name. Because fn165 retired the old analyzer and moved retained host/dynamic fixture coverage, this task must first inventory the current tree and prove whether any live or synthetic canonical-JSON ownership remains; it should make only the smallest fixture correction still required and may close as a verified no-op if fn165 already satisfies every retained acceptance point. It must not revive Native production work, delete meaningful negative controls, or alter final generated approvals owned by fn153.17. The forced local claim is required only because Flow chain admission still sees completed parent work on an unpushed remote branch; pushing remains outside the requested scope.

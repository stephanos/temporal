---
satisfies: [R1, R4]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.6 Race overlay embedding, identity separation and determinism inventory rows

## Description
Touches: tools/gomad/internal/backends/wasm/toolchain/overlay.go (embedding and ImplementationSHA256), overlay_test.go, internal/backends/wasm/determinism/inventory.json. Embed race_wasm.go.txt and the wasm runtime/race package into the overlay builder behind a profile parameter so that the cooperative overlay's ImplementationSHA256 input sequence is byte-unchanged and a second, race-profile identity covers the race sources. Add overlay_test coverage for the -race build of the choices fixture. Add inventory rows for every new overlay source, naming each as a deterministic input (shadow addresses derive from the deterministic guest heap). Depends on .5.

## Acceptance
Cooperative ImplementationSHA256 equals its pre-task value; the race-profile overlay identity changes when any race source changes (test); overlay_test builds the choices fixture with -race; inventory rows exist for every new source and the inventory lane passes.

## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.

## Evidence
- Commits:
- Tests:
- PRs:

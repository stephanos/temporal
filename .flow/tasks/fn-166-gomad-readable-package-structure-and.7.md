---
satisfies: [R7, R6]
---
# fn-166-gomad-readable-package-structure-and.7 Group shared packages under evidence and model

## Description
Move runner, backends, record/artifact/captured/qualification (as evidence), choice/world/simulation clock and network/volume/iowire (as model), target (into backends), hostfs+hostexec (as host) and puritycheck (into archcheck) under internal/; delete the external Runner consumer test, its fixture module and the public-signature and type-alias rules. Update archcheck owner, edge and pure-set tables, generators, Make targets and documentation. Land in the same integrated candidate as the R5 renames, fn-153 and fn-154.

Carried from .4 and .9 (2026-10-13): move inspection to `runner/inspect` (needs plan reading moved into `runner/internal/campaign` first so `Inspect` uses no Runner-private projections and Runner tests keep their fakes without an import cycle; 36 Runner identifiers named from `backends/wasm` and the 7 `SimulationClock` methods in identity-bound `runner/simulation_broker.go` must move with it to halve Runner's exports), and remove the remaining `tools/gomad3integration` module (`TestPublicWrappers`, `testdata/tagged`) together with the library-contract checks; update the `soak.json` description path when its manifest digest is rotated anyway.

## Acceptance
The module top level holds only cmd, internal and testdata, and internal/ holds runner, backends, evidence, model, host and archcheck; Git renames preserve history; full WASM gate set passes on the integrated candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

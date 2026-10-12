---
satisfies: [R7, R6]
---
# fn-166-gomad-readable-package-structure-and.7 Group shared packages under evidence and model

## Description
Move runner, backends, record/artifact/captured/qualification (as evidence), choice/world/simulation clock and network/volume/iowire (as model), target (into backends), hostfs+hostexec (as host) and puritycheck (into archcheck) under internal/; delete the external Runner consumer test, its fixture module and the public-signature and type-alias rules. Update archcheck owner, edge and pure-set tables, generators, Make targets and documentation. Land in the same integrated candidate as the R5 renames, fn-153 and fn-154.

## Acceptance
The module top level holds only cmd, internal and testdata, and internal/ holds runner, backends, evidence, model, host and archcheck; Git renames preserve history; full WASM gate set passes on the integrated candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R7, R6]
---
# fn-166-gomad-readable-package-structure-and.7 Group shared packages under evidence and model

## Description
Move record, artifact, captured and qualification under evidence/; choice, world, simulation/clock, simulation/network and volume under model/; target into backends. Update archcheck owner, edge and pure-set tables, generators, Make targets and documentation. Land in the same integrated candidate as the R5 renames, fn-153 and fn-154.

## Acceptance
The module top level holds only cmd, runner, backends, evidence, model, internal and testdata; Git renames preserve history; full WASM gate set passes on the integrated candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

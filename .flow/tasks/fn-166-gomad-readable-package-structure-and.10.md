---
satisfies: [R10, R6]
---
# fn-166-gomad-readable-package-structure-and.10 Move gomad3sim into gomad3 as internal/model/cluster (rotation batch)

## Description
Replace gomad3sim's single server common/collection use with the standard library, move tools/gomad3sim into tools/gomad3/model/cluster, and split it into the public cluster API and internal runtime-domain wire codecs and history records. Update WASM backend, native build and overlay references. Land in the same integrated candidate as fn-153, fn-154 and fn-166 tasks 6-8.

## Acceptance
tools/gomad3sim no longer exists; gomad3 core imports no server package; full WASM gate set and native source checks pass on the integrated candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

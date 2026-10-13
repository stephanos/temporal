---
satisfies: [R10, R6]
---
# fn-166-gomad-readable-package-structure-and.10 Move gomad3sim into gomad3 as internal/model/cluster (rotation batch)

## Description
Replace gomad3sim's single server common/collection use with the standard library, move tools/gomad3sim into tools/gomad3/model/cluster, and split it into the public cluster API and internal runtime-domain wire codecs and history records. Update WASM backend, native build and overlay references. Land in the same integrated candidate as fn-153, fn-154 and fn-166 tasks 6-8.

### Codec split binding

Establish lower-level internal record/identity DTO ownership before extracting wire codecs, so cluster API and codecs depend downward without a codec → parent cluster import. Include every moved semantic source in the cluster implementation identity. Integrate fn154 history removal before relocating these active write surfaces. Preserve the explicitly required native source and final WASM gates.
## Acceptance
tools/gomad3sim no longer exists; gomad3 core imports no server package; full WASM gate set and native source checks pass on the integrated candidate.

### Revival (2026-10-13)

Resume with fn166.6/.7 from a1f0be7e96, which includes the fn154 payload-free history owner. Establish lower DTO/codec ownership before moving the API and record all relocated semantic files in implementation identity. Root serializes integration with the independent oracle and network corpus lanes; final WASM/native-source gates remain mandatory.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

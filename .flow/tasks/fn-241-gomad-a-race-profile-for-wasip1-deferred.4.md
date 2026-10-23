---
satisfies: [R4]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.4 gomad.wasi-race/v1 profile identity, capacities, qualification cells, on/off docs

## Description
Touches: profile definitions and identity (gomad.wasi-race/v1), tools/gomad/internal/backends/wasm/QUALIFICATION.md, ARCHITECTURE.md, README/operator docs, limits configuration.
Define gomad.wasi-race/v1 as a third profile composing the cooperative overlay with the race overlay, bound to the rebuilt tool binaries, race overlay sources, shadow policy and recalibrated limits. Measure fuel, virtual-time and memory capacities on the one-box workload under instrumentation and record them beside the cooperative measurements. Add qualification cells for the race profile. Document how to select the profile per run and that artifacts replay only on their recorded profile. Prove the cooperative profile is byte-unchanged: its overlay identity, rewrite table, qualified expectations and installation identities match the pre-task values. Depends on task 3.

## Acceptance
gomad.wasi-race/v1 has a recorded identity and refuses mismatched installations; measured capacities and qualification cells exist on the one-box workload; operator docs cover on/off selection; a race artifact is refused on the cooperative profile; cooperative overlay identity, rewrite table and qualified expectations are unchanged; make -C tools/gomad validate green.


## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:

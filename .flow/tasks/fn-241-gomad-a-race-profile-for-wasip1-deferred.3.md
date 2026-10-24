---
satisfies: [R3]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.3 Race report class: wire fields, forcing schedule, retention and replay

## Description
Touches: race_wasm.go.txt (report emission on the existing choice/observation wire), rust/gomad/src/evidence (new classified target failure), rust/gomad/src/replay, CLI.md. Define the race report as a new target failure class with a stable name carrying both PCs, both access kinds, both goroutine creation sites, both raw stacks, and the forcing schedule (seed and choice ordinal at which the first access ran). Retain it like any target failure; replay the artifact on the race profile and require the identical report. Document the exact field layout in CLI.md. Symbolization is .9. Depends on .8.
## Acceptance
Report class name is stable and documented; a retained race artifact replays to an identical report on the race profile; the record carries both PCs, kinds, creation sites, stacks, seed and ordinal; evidence goldens for the new class exist.
## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:

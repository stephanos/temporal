---
satisfies: [R2]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.2 FastTrack core: vector clocks, epochs, goroutine seeding, sync edges, ordered/unordered corpus

## Description
Touches: race_wasm.go.txt and the wasm runtime/race package. Implement FastTrack in Go under one P: one vector clock per goroutine, per-granule last-write epoch and read epoch-or-clock, 8-byte granules; racegostart seeds the child clock from the parent; raceacquire/racerelease/racereleasemerge from the stock runtime call sites (channels, semaphores, sync) update clocks; a race is an access whose epoch is not ordered before the current goroutine's clock. Shadow storage in this task is a simple map keyed by granule so the algorithm can be tested before .7 replaces it. Add a Go corpus of small guests (ordered by channel, ordered by mutex, ordered by WaitGroup, unordered write/write, unordered write/read, map assign vs iterate) with golden reports. No atomics or locks on detector state. Depends on .6.
## Acceptance
Every corpus guest produces its golden report (or none) deterministically across two runs; the unordered map assign/iterate guest reports the pair without a crash; no nondeterministic input appears in the inventory; the detector code has no atomics or locks on shadow state.
## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R2]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.2 FastTrack detector with span-allocated shadow; one-seed detection of PR 10549

## Description
Touches: race_wasm.go.txt and the wasm runtime/race package (detector), tools/gomad/internal/backends/wasm/toolchain/** (identity inputs), overlay tests.
Implement FastTrack in Go: per-goroutine vector clocks, per-granule last-write epoch and read epoch-or-clock, 8-byte granules; racegostart seeds the child clock; raceacquire/racerelease/racereleasemerge from the stock runtime call sites update clocks. Shadow is allocated per heap span from racemalloc/racefree, outside the Go heap; seeded allocation sampling is the recorded fallback if the Temporal test server does not fit under the 4 GiB wasm32 ceiling. Run the §3 functional workload (history.cacheSizeBasedLimit on, small host-level cache limit, concurrent UpdateWorkflowExecution calls to one workflow) at the pre-fix PR 10549 revision and at the fixed revision. Depends on task 1 (toolchain path). Prerequisite owned elsewhere: the §3 functional test must exist in the Temporal test suite.

## Acceptance
One seed on the §3 workload reports the unordered pair naming mapassign in FindOrCreate and mapiterinit in GetSize at the pre-fix revision; the fixed revision produces no report; any sampling policy is recorded in the profile; the detector is deterministic across two runs of the same seed (identical report); no nondeterministic input added to the overlay inventory.


## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:

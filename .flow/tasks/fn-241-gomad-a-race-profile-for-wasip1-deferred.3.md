---
satisfies: [R3]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.3 Retained symbolized race reports, forcing schedule, sync/atomic edges

## Description
Touches: tools/gomad/internal/backends/wasm/** (failure classification, retention, inspect symbolization), race_wasm.go.txt (sync/atomic wrappers, report emission), CLI.md.
Emit a race as a classified target failure with both PCs, both access kinds, both goroutine creation sites and both stacks, symbolized through the fn-229 inspect path, plus the forcing schedule (seed and choice ordinal of the first access). Implement the sync/atomic runtime wrappers for wasm so atomics perform the operation and record acquire/release edges. Emit the happens-before fingerprint (digest of final vector clocks) as an observation record on the choice wire. Verify on the §3 workload that the stale-size variant (MutableState size read on the lock-failure path) is reported and that an atomic-guarded counter fixture is not. Depends on task 2.

## Acceptance
Race report class has a stable name and documented fields; reports replay exactly on the race profile and symbolize both stacks; the stale-size variant is reported; an atomic.Int64-guarded counter fixture produces no report; the fingerprint appears as an observation record and is never forced; CLI.md documents the report.


## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R3]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.10 sync/atomic edges, stale-size variant, atomic-counter negative fixture, happens-before fingerprint

## Description
Touches: race_wasm.go.txt (sync/atomic runtime wrappers, fingerprint emission), corpus fixtures. Implement the sync/atomic wrappers for wasm so each performs the operation and records an acquire/release edge, correct under one P. Verify on the §3 workload that the stale-size variant (MutableState size read on the lock-failure path) is reported, and add an atomic.Int64-guarded counter fixture that must produce no report. Emit the happens-before fingerprint (digest of the final vector clocks) as an observation record on the existing choice wire, flagged observation and never forced; it is not consumed here. Depends on .3.

## Acceptance
Stale-size variant reported with its PCs; the atomic counter fixture produces no report; fingerprint appears as an observation record in the retained evidence and the replay preflight refuses it as a forced choice (test).

## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R1]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.5 Stub runtime/race for wasm: no-op entry points, instrumented choices fixture matches uninstrumented

## Description
Touches: new overlay source race_wasm.go.txt (//go:build race && wasip1 && wasm) and a wasm runtime/race package replacing the .syso. Implement the roughly 65 runtime race entry points (raceread, racewrite, the range, PC and object variants, raceacquire/racerelease/racereleasemerge, racemalloc/racefree, racegostart/racegoend, racefuncenter/racefuncexit, raceinit/racefini, the sync/atomic wrappers) as no-ops with a real raceinit/racefini so the compiler's instrumented calls link. Early proof point of the spec: build testdata/choices with -race for wasip1/wasm from the rebuilt toolchain and run it under the cooperative overlay. If the stub cannot be made to link, stop and return the spec to the owner. Depends on .1.

## Acceptance
The choices fixture builds with -race and runs to the same outcome class and identical choice records as the uninstrumented build; racefuncenter/racefuncexit are no-ops or stack-depth bookkeeping only; entry-point inventory recorded against stock runtime/race.go.

## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.

## Evidence
- Commits:
- Tests:
- PRs:

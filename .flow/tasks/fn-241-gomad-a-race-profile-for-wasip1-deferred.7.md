---
satisfies: [R2, R4]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.7 Span-allocated shadow memory with recorded sampling fallback

## Description
Touches: race_wasm.go.txt (racemalloc/racefree, shadow allocator). Replace the .2 map with shadow allocated per heap span outside the Go heap, two words per 8-byte granule, driven by racemalloc and racefree, so GC accounting is unaffected. Measure shadow footprint on the one-box Temporal test server under instrumentation against the 4 GiB wasm32 ceiling. If it does not fit, implement seeded allocation sampling that skips a recorded fraction of spans; the policy (none or fraction and seed derivation) is a profile identity input and is written into every record. Depends on .2.

## Acceptance
Corpus from .2 passes unchanged on span shadow; shadow footprint and heap size on the one-box server are recorded; sampling, if enabled, is deterministic per seed and appears in the record and the profile identity; a run with sampling cannot claim 'no race' for skipped spans (test).

## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.

## Evidence
- Commits:
- Tests:
- PRs:

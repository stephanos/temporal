---
satisfies: [R3]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.9 Symbolize both race stacks through the inspect path

## Description
Touches: rust/gomad/src/evidence/symbolize and the inspect verb. Symbolize both access stacks and both creation sites of a race report through the fn-229 inspect path (DWARF/name sections of the instrumented module), and show them in inspect output. Instrumented modules have different code layout; verify symbolization against the §3 workload report and the .2 corpus. Depends on .3.

## Acceptance
inspect on a race artifact prints both stacks symbolized to function, file and line for the §3 report and every corpus guest; an unsymbolizable frame is reported as such, never dropped; symbolize goldens updated.

## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R4]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.4 Race profile capacities, qualification cells, operator documentation and closing audit

## Description
Touches: rust/gomad/src/qualify (cells for the race profile), measurement limits, QUALIFICATION.md, CLI.md, ARCHITECTURE.md, README. Measure fuel, virtual-time and memory capacities on the one-box workload under instrumentation and record them beside the cooperative measurements; replace the .11 placeholders and rotate the race profile identity. Add qualification cells for the race profile on the one-box workload. Document turning the profile on and off per run, that artifacts replay only on their recorded profile, and the sampling caveat. Closing audit: cooperative identities unchanged, no cooperative expectation touched, inventory complete. Depends on .3, .9, .10 and .11.
## Acceptance
Measured race-profile capacities and qualification cells exist for the one-box workload; operator docs cover selection, replay-on-profile and sampling; cooperative overlay identity, rewrite table and qualified expectations are unchanged; make -C tools/gomad validate green; all R1–R4 acceptance evidence linked from this task.
## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.
## Evidence
- Commits:
- Tests:
- PRs:

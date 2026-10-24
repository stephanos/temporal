---
satisfies: [R2]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.8 One-seed detection of PR 10549 on the section 3 workload; fixed revision is clean

## Description
Touches: race profile fixtures and the §3 functional workload (history.cacheSizeBasedLimit on, small host-level cache limit, concurrent UpdateWorkflowExecution calls to one workflow) at the pre-fix PR 10549 revision and the fixed revision. Prerequisite owned elsewhere: the §3 functional test exists in the Temporal suite and the benchmark row reads reachable: no. Run both revisions under the race profile; the pre-fix run must report the pair mapassign in FindOrCreate / mapiterinit in GetSize from one seed, the fixed run no report. Record both as the benchmark-row change. Depends on .7.

## Acceptance
One seed reports the pair at the pre-fix revision naming both functions; the fixed revision produces no report; two runs of the same seed produce identical reports; any false positive on the fixed code is recorded with its PCs as a finding.

## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.

## Evidence
- Commits:
- Tests:
- PRs:

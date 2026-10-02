---
satisfies: [R2]
---
# fn-106-gomad-close-the-remaining-tests-gaps.2 Guard MemStats.LastGC from the host clock

## Description
Make gcMarkTermination use virtual time when gomadEnabled (patch + regenerate), update the clock inventory, run the runtime tier.

## Acceptance
- inventory shows no unguarded time_now in mgc.go; runtime tier passes


## Done summary
Classified, not fixed. gcMarkTermination reads time_now to stamp MemStats.LastGC; the patch policy (toolchain/patch.go prohibitedRuntimeArea) keeps every mgc* and mstats* runtime file out of the patch so it never touches the collector, and time_now on linux/amd64 is platform assembly, also prohibited. Widening that prohibition for a reporting field was rejected: LastGC is written but never read by runtime control flow, so it cannot perturb scheduling, GC pacing, or replay evidence unless a target prints the value. The escape stays pinned in toolchain/clock_inventory_test.go (runtime/mgc.go time_now, escape with finding) and recorded in MILESTONES.md Open findings.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make test-toolchain (inventory pins the escape)
- PRs:
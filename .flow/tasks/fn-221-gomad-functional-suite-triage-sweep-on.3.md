---
satisfies: [R3, R4]
---
# fn-221-gomad-functional-suite-triage-sweep-on.3 Rank reasons and recommend follow-on specs

## Description
From the retained matrix, rank distinct reason strings by count, report the actual outcome of each known suspect (freeport loopback port allocation in tests/testcore/test_cluster.go, os/exec, cgo, ringpop static hosts, cloud archivers, non-SQLite persistence selections), and recommend one follow-on spec per reason class with an estimate of how many tests it would unblock. Note which findings feed fn-185 (host-side cost) and fn-188 (determinism contract) without taking over their scope. Append the report to the matrix artifact.

**Size:** S
**Files:** .flow/artifacts/fn221-support-matrix-<date>.md

## Acceptance
- [ ] Ranked reason table with counts
- [ ] Each known suspect's actual outcome stated
- [ ] Each recommendation maps to a reason class with an unblock estimate
- [ ] No support wording

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R1]
---
# fn-106-gomad-close-the-remaining-tests-gaps.1 Bisect and fix the linux seed-17 replay divergence

## Description
Reproduce on linux/amd64 (Rosetta container locally, fork CI as ground truth), revert 6bc11ef7d and 440552d2c in turn, identify the channel, fix it, and restore the linux expectations and two-seed smoke gate.

## Acceptance
- seed 17 of the F5/F6 suites replays exactly on linux across repeated runs
- temporal.json linux expectations back to qualified where measured; smoke gate on seeds 11 and 17


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

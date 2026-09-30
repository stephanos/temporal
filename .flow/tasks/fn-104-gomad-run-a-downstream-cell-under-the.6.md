---
satisfies: [R5, R6]
---
# fn-104-gomad-run-a-downstream-cell-under-the.6 Measure the downstream smoke test and record the result

## Description
Run gomad analyze (linked) and gomad qualify --repeat 2 on seeds 11 and 17 from the downstream checkout against this branch's toolchain, with the downstream pack in an external directory and any downstream seams applied as local, uncommitted scratch edits that are reverted afterwards. Classify every non-qualified outcome (capability blocker, unmodeled operation, watchdog, evidence divergence) with its finding. Record the result in GOMAD_MILESTONES.md F9 and GOMAD_CLOUD.md without naming downstream components; update README/ARCHITECTURE for downstream targets.

## Acceptance
- the smoke test qualifies with exact replay or every outcome is classified with a finding
- the downstream checkout is clean afterwards; no downstream names in committed docs


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

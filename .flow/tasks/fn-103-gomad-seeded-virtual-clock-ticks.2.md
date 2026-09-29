---
satisfies: [R3]
---

# fn-103-gomad-seeded-virtual-clock-ticks.2 Decide the default tick policy from measurement and requalify
# fn-103-gomad-configurable-virtual-clock-tick.2 Measure the tick on the tie-excluded suites and retire their exclusions

## Description
Run the tie-excluded suites and the smoke selection under the seeded policy across several seeds; decide the default from the evidence (R3); if seeded becomes default, requalify the core, representative and smoke sets and resolve tie exclusions (fixed upstream, pinned per workload with a finding, or seed-named); record in the milestone doc. No full ./tests runs.
## Acceptance
- report lists exclusions removed; manifest updated; validate green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

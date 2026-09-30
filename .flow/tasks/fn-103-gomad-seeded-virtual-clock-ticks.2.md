---
satisfies: [R3]
---

# fn-103-gomad-seeded-virtual-clock-ticks.2 Measure forward on the tie-excluded suites and decide the default
# fn-103-gomad-configurable-virtual-clock-tick.2 Measure the tick on the tie-excluded suites and retire their exclusions

## Description
Run the tie-excluded suites and the smoke selection under forward across several seeds; if the smoke set stays green and the tie failures resolve, make forward the default, requalify the core, representative and smoke sets, and resolve tie exclusions (fixed upstream, pinned with a finding, or seed-named); otherwise keep strict and record the evidence. No full ./tests runs.
## Acceptance
- report lists exclusions removed; manifest updated; validate green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

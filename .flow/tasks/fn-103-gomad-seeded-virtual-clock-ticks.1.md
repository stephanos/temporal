---
satisfies: [R1, R2, R4]
---

# fn-103-gomad-seeded-virtual-clock-ticks.1 Implement seeded, fixed, and strict tick policies in the runtime and profile
# fn-103-gomad-configurable-virtual-clock-tick.1 Implement the configurable clock tick in the runtime and profile

## Description
Implement the tick policies (seeded mixture with a dedicated seed-derived stream, fixed, strict) at read or wake, profile + identity fields, CLI/manifest plumbing, replay mismatch check, runtime fixture and a core workload per policy, docs; strict reproduces today byte-for-byte.
## Acceptance
- tick off: all gates unchanged
- tick on: fixture and core workload repeat and replay exactly

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

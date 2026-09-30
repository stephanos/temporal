---
satisfies: [R1, R2, R4]
---

# fn-103-gomad-seeded-virtual-clock-ticks.1 Implement forward and strict tick policies in the runtime and profile
# fn-103-gomad-configurable-virtual-clock-tick.1 Implement the configurable clock tick in the runtime and profile

## Description
Scope cut 2026-09-29: seeded and fixed moved to fn-105-gomad-follow-ups-deferred-scope.6. Implement the forward policy (every draw at least 1 ns at the configured application point, read or wake) next to strict (today): runtime tick, profile and Campaign/Artifact identity fields, CLI and manifest plumbing, replay mismatch check, a runtime fixture and a core workload under forward, docs. strict reproduces today byte-for-byte.
## Acceptance
- tick off: all gates unchanged
- tick on: fixture and core workload repeat and replay exactly

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

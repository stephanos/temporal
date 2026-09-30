---
satisfies: [R6]
---
# fn-105-gomad-follow-ups-deferred-scope.6 D6: seeded and fixed virtual-clock tick policies

## Description
Origin: fn-103 R1/R3. The user explicitly keeps this deferred on 2026-09-30. Forward alone removes the known tie failures; seeded mixtures and fixed quanta are exploration features without a demonstrated workload that needs them. Revive when a specific bug class needs deliberate timestamp ties or constant increments, retaining that evidence before implementation. On revival, add the policies, per-workload manifest setting, a runtime fixture and core workload per policy, and the COMPAT-5 evidence. No clock implementation is required by the deferral decision.

## Acceptance
Implementation remains deferred until a specific bug class and its need for deliberate ties or constant increments are recorded. On revival, each policy repeats and replays exactly on a runtime fixture and a core workload, with manifest settings, execution identity, and COMPAT-5 evidence. Preserve strict and forward behavior. Recording the deferral does not claim implementation or close the task as implemented.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

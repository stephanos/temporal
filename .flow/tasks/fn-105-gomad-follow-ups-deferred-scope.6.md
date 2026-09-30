---
satisfies: [R1, R2]
---
# fn-105-gomad-follow-ups-deferred-scope.6 D6: seeded and fixed virtual-clock tick policies

## Description
Origin: fn-103 R1/R3. Deferred 2026-09-29: forward alone removes the known tie failures; seeded mixtures and fixed quanta are exploration features without a demonstrated bug. Revive when a bug class needing deliberate ties or constant quanta is found. Adds the policies, per-workload manifest setting, a runtime fixture and core workload per policy, and the COMPAT-5 evidence.

## Acceptance
Revival trigger recorded; each policy repeats and replays exactly on a runtime fixture and a core workload; strict and forward unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

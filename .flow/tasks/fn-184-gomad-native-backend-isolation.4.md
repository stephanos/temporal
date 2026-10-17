---
satisfies: [R5]
---
# fn-184-gomad-native-backend-isolation.4 Measure, document and hand over the isolated candidate

## Description
Repeat the task .1 measurement on the frozen integrated candidate on the same quiet host: default-set and selected-set package count, production/test lines and wall time for build, vet, lint, test-harness, world-test, test-wasm; report medians with ranges and the host identity. Update tools/gomad/README.md, ARCHITECTURE.md (module layout, maintenance gates, host tooling boundary) and MILESTONES.md (native section header, constraints) to describe the selection, the default-build refusal and the preserved native gates, stating that native execution remains deferred under fn-128/fn-149. Produce one batch handover mapping tasks .1 to .4 and R1 to R5 to identities, commands, exits and the independent review verdict.

**Size:** S
**Files:** tools/gomad/README.md, tools/gomad/ARCHITECTURE.md, MILESTONES.md, .flow/artifacts/fn184-*.md

## Acceptance
- [ ] Before/after table retained with host identity, medians and ranges
- [ ] README, ARCHITECTURE and MILESTONES describe selection, refusal and preserved gates; no native qualification claimed
- [ ] Batch handover maps every task and R-ID to evidence and the independent review

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

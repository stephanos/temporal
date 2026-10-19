---
satisfies: [R3, R4]
---
# fn-224-gomad-ringpop-membership-on-wasm-within.3 Add a multi-node ringpop partition/heal scenario and document it

## Description
Write a small guest program under tools/gomad/internal/backends/wasm/testdata/ringpop that joins a ringpop ring using its node address and boot arguments and emits an output marker when its membership view has converged on the expected member set. Write a gomad-wasm.scenario/v1 fixture under internal/backends/wasm/command/testdata/scenario/ with three or four nodes, a partition between two, a timed heal and the convergence markers, using only existing schema fields. Explore it over a bounded seed set; every seed converges or retains a replayable failure with its first-divergence report. Document the fixture and exact commands in internal/backends/wasm/CLI.md next to the existing scenario section.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/testdata/ringpop/**, tools/gomad/internal/backends/wasm/command/testdata/scenario/ringpop.json, tools/gomad/internal/backends/wasm/CLI.md

## Acceptance
- [ ] Scenario fixture and guest program checked in using existing schema fields only
- [ ] Every explored seed converges or retains a replayable failure with a first-divergence report
- [ ] Commands and fixture documented in the WASM operator guide
- [ ] No new helper imports, model types or scenario fields

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

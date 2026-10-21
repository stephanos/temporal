---
satisfies: [R3, R4]
---
# fn-224-gomad-ringpop-membership-on-wasm-within.3 Add a multi-node ringpop partition/heal scenario and document it

## Description
Write a small guest program under tools/gomad/internal/backends/wasm/testdata/ringpop that joins a ringpop ring using its node address and boot arguments and emits an output marker when its membership view has converged on the expected member set. Write a gomad-wasm.scenario/v1 fixture under internal/backends/wasm/command/testdata/scenario/ with three or four nodes, a partition between two, a timed heal and the convergence markers, using only existing schema fields. Explore it over a bounded seed set; every seed converges or retains a replayable failure with its first-divergence report. Document the fixture and exact commands in internal/backends/wasm/CLI.md next to the existing scenario section.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/testdata/ringpop/**, tools/gomad/internal/backends/wasm/command/testdata/scenario/ringpop.json, tools/gomad/internal/backends/wasm/CLI.md

**Touches:** tools/gomad/internal/backends/wasm/testdata/ringpop/**, tools/gomad/internal/backends/wasm/command/testdata/scenario/ringpop.json, tools/gomad/internal/backends/wasm/CLI.md

## Acceptance
- [x] Retained lifecycle/network evidence proves partition then heal executed before each node's fresh post-heal membership observation. Initial convergence cannot trigger final completion; guests remain alive until all required observations are complete. Missing actions/observations fail the convergence claim.
- [x] Explicit bounded success retention is enabled; every successful convergence artifact is replayed and the same ordered fault-phase proof reappears. Retained failures replay under the existing contract; overflow/missing evidence never counts as exact replay.
- [x] Build the nested-module guest with an absolute --working-dir naming the guest module (tools/gomad/internal/backends/wasm/testdata/ringpop) and go-run .; retain the resolved scenario and installation paths. (Amended 2026-10-21: the parent-module form --working-dir tools/gomad with go-run ./internal/backends/wasm/testdata/ringpop is refused by the Go toolchain because the guest is its own nested module; evidence in the fn224 scenario artifact.)
- [x] Scenario fixture and guest program checked in using existing schema fields only
- [x] Every explored seed converges or retains a replayable failure with a first-divergence report
- [x] Commands and fixture documented in the WASM operator guide
- [x] No new helper imports, model types or scenario fields

## Done summary
Three-node ringpop partition/heal scenario: the fixture is command/testdata/scenario/ringpop.json with ringpop-config.json, and the guest is the nested module testdata/ringpop. It uses existing schema fields only and is documented in CLI.md under "Explore a typed cluster scenario".

Evidence candidate 39694cd12b (gomad 92c69594f2 + fn-224 ab7625c647; binaries built from it), seeds 7 and 19, with explicit success retention:
- Both seeds succeed.
- candidate3/faultphase.py passes on both retained successes. Network evidence holds all 8 c<->a/b partition transitions (ok) before the 8 heals. Every node prints bootstrapped, then the partitioned view (the guest asserts a and b dropped c), then a fresh post-heal convergence over 3 consecutive full-member polls, and re-checks at the 90 s deadline. All nodes exit 0 at the wait.
- Both successful convergence artifacts replay with reproduced=true result=success choice-replay=none.

The same results hold for the earlier ab7625c647 candidate. The cohort's integrated-helper 120 s runs also succeed and replay (fn-235.2).

Amended criterion: the parent-module working-dir form is refused by the Go toolchain because the guest is a nested module. The guest-module working-dir with go-run . is used and documented in the task, the spec quick command and CLI.md.

Note: the gomad tip 3ad88e1cde Go launcher cannot run the scenario. Its cluster inspection rejects the v2 decision records from 38572d7e49, and this is unrelated to fn-224 (fn-232 owner).

Independent re-review: SHIP-WITH-FIXES; it would close .3 after the command amendment, which is applied.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8280497564, ab7625c647, fe9bd00343, 3d4f4704cb, 2256f61d7a
- Tests: candidate3/run.sh on 39694cd12b: scenario seeds 7,19 success, faultphase pass, replay reproduced=true, TestOperatorRingpopScenarioPreflight exit 0, evidence/fn224/review2/codex2.out: SHIP-WITH-FIXES
- PRs:
---
satisfies: [R5]
---
# fn-235-gomad-fail-cluster-runs-fast-and.2 Measure and remove superlinear World inspection commitment cost in long cluster runs

## Description
**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/world/**, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/**, tools/gomad/internal/backends/wasm/command/testdata/scenario/** and retained fn-235 measurement reports.

Found in fn-224.3 (DIAGNOSIS.md): a three-node ringpop scenario slows from ~3 s wall per virtual second toward ~8 (84 s virtual at 5 min, 132 s at 15 min). Suspected site: guest debug exchange per scheduling decision (runtime_inspect_wasm.go.txt:99) makes the helper rebuild the World inspection commitment (cluster/handoff/broker.rs:1227) by hashing full history; the cache is per World revision and every brokered operation bumps it. One perf sample (of the sweep helper, not this scenario) showed 74% SHA-256 there. First measure on the fn-224 ringpop scenario (wall per virtual second at 10/60/120 s, profile with a private copy of the helper and perf --no-buildid-cache), then compute the commitment incrementally or only when inspect reads it. Commitment bytes and replay identity must not change unless a new identity is declared. Performance claims follow fn-185 measurement rules.

## Acceptance
- [ ] Fresh retained replay of the actual measured long-running workload reproduces its commitment/evidence and outcome; timing results and generic golden scenarios alone do not close R5. Any changed commitment contract has explicit identity rotation and the required current-candidate qualification; rotation alone is not acceptance.
- [ ] Retained measurement of wall per virtual second over a long three-node run, before and after, same host and window, with repeated runs, median/range, exact source/helper/module/configuration identities and commands
- [ ] Profile attributes the cost to a named site before any change
- [ ] Wall per virtual second stays flat (within measured spread) across the run after the fix; commitment and replay identities unchanged or explicitly rotated
- [ ] Golden scenario corpora stay green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

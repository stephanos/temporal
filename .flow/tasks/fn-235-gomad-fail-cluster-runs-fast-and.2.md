---
satisfies: [R5]
---
# fn-235-gomad-fail-cluster-runs-fast-and.2 Measure and remove superlinear World inspection commitment cost in long cluster runs

## Description
**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/world/**, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/**, tools/gomad/internal/backends/wasm/command/testdata/scenario/** and retained fn-235 measurement reports.

Found in fn-224.3 (DIAGNOSIS.md): a three-node ringpop scenario slows from ~3 s wall per virtual second toward ~8 (84 s virtual at 5 min, 132 s at 15 min). Suspected site: guest debug exchange per scheduling decision (runtime_inspect_wasm.go.txt:99) makes the helper rebuild the World inspection commitment (cluster/handoff/broker.rs:1227) by hashing full history; the cache is per World revision and every brokered operation bumps it. One perf sample (of the sweep helper, not this scenario) showed 74% SHA-256 there. First measure on the fn-224 ringpop scenario (wall per virtual second at 10/60/120 s, profile with a private copy of the helper and perf --no-buildid-cache), then compute the commitment incrementally or only when inspect reads it. Commitment bytes and replay identity must not change unless a new identity is declared. Performance claims follow fn-185 measurement rules.

## Acceptance
- [x] Fresh retained replay of the actual measured long-running workload reproduces its commitment/evidence and outcome; timing results and generic golden scenarios alone do not close R5. Any changed commitment contract has explicit identity rotation and the required current-candidate qualification; rotation alone is not acceptance.
- [x] Retained measurement of wall per virtual second over a long three-node run, before and after, same host and window, with repeated runs, median/range, exact source/helper/module/configuration identities and commands
- [x] Profile attributes the cost to a named site before any change
- [x] Wall per virtual second stays flat (within measured spread) across the run after the fix; commitment and replay identities unchanged or explicitly rotated
- [x] Golden scenario corpora stay green

**Status (2026-10-21):** measured, fixed and independently reviewed. Evidence: `.flow/artifacts/fn235-commitment-cost-2026-10-21.md`; raw data under `/home/agent/wt/evidence/fn235/cohort/`. The review is `/home/agent/wt/evidence/fn235/review/codex.out`, verdict NEEDS-WORK on one item: every acceptance item is met except current-candidate qualification of the rotated inspection contract.

- Fresh retained replay of the measured workload reproduces on the measured candidate `92c69594f2`.
- Owner decision 2026-10-21: current-candidate qualification of the rotated inspection contract transfers to fn-232.13 (section "fn-235 R5 rotated inspection-contract receiver"), which must reconcile its receipts into the fn-235 artifact. The obligation is pending there, not waived.
- No further measurement or source change is needed.

## Done summary
Measured, attributed, fixed and replayed (`.flow/artifacts/fn235-commitment-cost-2026-10-21.md`, raw `/home/agent/wt/evidence/fn235/cohort/`). Before: wall per virtual second grows 4.6 s → 17.9 s over the 120 s three-node ringpop run; after: flat within spread (post-heal 0.209/0.206/0.187 s per virtual s), median command wall 25.9× lower, three ABBA counted pairs. Profile attributed 87.13% to World inspection commitment before the change, then endpoint network inspection; both computed through the composed path. Canonical World/network/choice records unchanged; declared rotations `gomad.inspection-state-sha256/v2`, helper implementation identity, ModelSHA256 d6ca1cda… → e5541190…. All counted after-side runs and an uninstrumented release build replay with reproduced=true; helper release suite, four feature corpora and cluster/world/network Go tests pass. Independent review (`/home/agent/wt/evidence/fn235/review/codex.out`) NEEDS-WORK only on current-candidate qualification of the rotated contract; by owner decision 2026-10-21 that obligation transfers to fn-232.13 ("fn-235 R5 rotated inspection-contract receiver") and remains pending there.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1726d4ebce, d742092f68, 503f7afd1a
- Tests: fn235 cohort ABBA x3, replay after-1..3 reproduced=true
- PRs:
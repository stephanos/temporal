---
satisfies: [R1, R2, R3, R4]
---
# fn-235-gomad-fail-cluster-runs-fast-and.1 Classify broker capacity exhaustion and end manual cluster runs promptly with retained partial evidence

## Description
Rust helper owner (wasmhost/src/cluster/handoff/broker.rs, cluster/execution/manual.rs): add the golden two-node scenario with a tiny observation_bytes first (hangs to watchdog before the fix), then map world::Error::Capacity from broker World calls to Failure::Capacity(dimension) and emit the terminal result/abort to the controller as soon as an instance fails, stopping the rest through the existing terminate path and retaining each instance partial streams and abort virtual time. Go consumes it with the smallest change; note the R-ID for fn-232s Rust Runner. Merge after coordinating with the fn-232 helper-as-library batch (helper identity rotates). Reproducer and site analysis: /home/agent/wt/evidence/fn224/scenario/DIAGNOSIS.md; worker prototype branch fn224-cluster-stall-proto (instrumentation only, do not merge as is).

## Acceptance
- [x] Golden tiny-budget scenario ends as capacity naming payload-bytes within 10 s wall with --execution-timeout>=3m
- [ ] inspect shows each node partial streams and the limit; replay reproduces the capacity outcome
- [x] Other manual-cluster instance/broker failures also end without waiting for the watchdog
- [x] Existing scenario corpora (durable, network, config) and fn-224 ringpop scenario stay green

## Done summary

Reopened on 2026-10-21 after an additional genuine concurrent-start regression.
The earlier completion below records the then-verified cases, not current
acceptance. A peer admitted by the host but aborted before its first guest hook
retains `configured=false`, `finished=false`, zero calls and zero memory, while
the shared helper inspection incorrectly claims complete evidence with start
boundary 136 and terminal boundary 0. The original Go validator correctly refuses
those incarnation coordinates, losing the retained failure artifact. R3 remains
open until the actual admitted owner gets its real terminal coordinate without
inventing guest execution, and the original failing alternate case and linked
partial-peer inspection pass. Raw byte-transparent capture is retained at
`/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-replay-alternate-wire/1356714.stdout`;
the original unwrapped reference failure remains in fn232-replay evidence.
Manual and linked cluster failures now stop peers deterministically and preserve the original capacity or application failure, reached crash/stop boundaries, per-node partial streams and abort virtual time. R1-R4 are verified on the integrated helper inputs at 294b5e66e5, unchanged through b301cbc88c. Fresh installed CLI tests passed in 59.557 seconds: payload-bytes capacity 0.722 seconds, guest failures 6.287/0.540 seconds, and deadlock 7.595 seconds, all under strict 10-second assertions with three-minute watchdogs. Inspect and exact replay, linked binary World decoding/repeated publication, durable/canary/network detached scenarios and tamper refusals pass. The original pinned three-node ringpop120 scenario succeeds with module b0462afc1b7a2011d17bcc780b852a1e922dd97a7ee0407e4a28d1e22283a818. Independent read-only review returns SHIP for R1-R4. Logs: /home/agent/wt/evidence/fn235/root-final294-installed-failure-gate.log; /home/agent/wt/evidence/fn235/commitment/final294-release-120-warmup/measurement.json; final294 build/test receipts under /home/agent/wt/evidence/fn232-replay. Prior RED regressions remain in failure-repair-HANDOVER.md. Task .2 performance acceptance and fn232.13 qualification remain open; this completion does not claim either.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9f3364f156, 7693689f38, dad95ac6f0, 5452fcf463, a959c074ba, c33b3dde63, df9f9d27d1, 25e32e9b65, fa23d8e6c7, 294b5e66e5, 1ba2bb38eb
- Tests: Fresh integrated installed CLI: GOMAXPROCS=2 GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go GOMAD_WASM_TEST_HELPER=/home/agent/wt/evidence/fn235/commitment/helper-final294-release go test -tags test_dep -count=1 -p=1 -parallel=1 -timeout=10m -v -run ^TestInstalledOperator(TypedScenarioDetachedReplay|AsymmetricGuestFailureEndsPromptly|ClusterFailureEndsPromptly)$ ./internal/backends/wasm/command (exit 0, 59.557s; root-final294-installed-failure-gate.log), Final294 original fn224 ringpop120 actual CLI success (exit 0; commitment/final294-release-120-warmup/{command.json,measurement.json,stdout.jsonl}), Final294 all four host role builds and all-target Clippy; 50 host library tests and full helper test suite (exit 0; /home/agent/wt/evidence/fn232-replay/final294-* receipts), Fresh Go implementation identity, current corpora, broker order and cluster inspection tests (exit 0, packages 0.181s/0.046s), make lint-code-fast GOLANGCI_LINT_BASE_REV=447fee0f75 GOLANGCI_LINT_FIX=false (exit 0; root-integrated-exploration-lint-code-fast.log), Independent fn235.1 R1-R4 acceptance review SHIP; performance task .2 and final Rust qualification excluded
- PRs:

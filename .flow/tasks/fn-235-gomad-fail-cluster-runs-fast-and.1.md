---
satisfies: [R1, R2, R3, R4]
---
# fn-235-gomad-fail-cluster-runs-fast-and.1 Classify broker capacity exhaustion and end manual cluster runs promptly with retained partial evidence

## Description
**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/cluster/**, tools/gomad/internal/backends/wasm/wasi/**, tools/gomad/internal/backends/wasm/cluster/**, tools/gomad/internal/backends/wasm/command/** and CLI/scenario golden fixtures. Completed task; reconcile any follow-up with .2 shared cluster/scenario writes.

Rust helper owner (wasmhost/src/cluster/handoff/broker.rs, cluster/execution/manual.rs): add the golden two-node scenario with a tiny observation_bytes first (hangs to watchdog before the fix), then map world::Error::Capacity from broker World calls to Failure::Exhausted retaining dimension, limit, used and delta and emit the terminal result/abort to the controller as soon as an instance fails, stopping the rest through the existing terminate path and retaining each instance partial streams and abort virtual time. Go consumes it with the smallest change; note the R-ID for fn-232s Rust Runner. Merge after coordinating with the fn-232 helper-as-library batch (helper identity rotates). Reproducer and site analysis: /home/agent/wt/evidence/fn224/scenario/DIAGNOSIS.md; worker prototype branch fn224-cluster-stall-proto (instrumentation only, do not merge as is).

## Acceptance
- [x] Golden tiny-budget scenario ends as capacity naming payload-bytes within 10 s wall with --execution-timeout>=3m
- [x] inspect shows each node partial streams and the limit; replay reproduces the capacity outcome
- [x] Other manual-cluster instance/broker failures also end without waiting for the watchdog
- [x] Existing scenario corpora (durable, network, config) and fn-224 ringpop scenario stay green

## Done summary
R1-R4 accepted after repairing and retesting the genuinely reopened R3 edge case. Host-admitted peer aborted before its first guest hook now has real terminal coordinates (start136/terminal137, formerly terminal0), preserving configured=false/finished=false, zero calls/memory and empty choice/WASI/registry. The helper never invents guest execution or a nonmember owner. Failed linked lifecycle prefixes retain original authored actions; implicit cleanup invents no Wait. A complete successful control passes before the corresponding open-peer prefix is refused.

Final exact helper SHA256 71f7dc4760f3c5ba2cfd957978aa0f636940f866ea21f018c9c9967ddc2199df, host ebb89eee447833839c4fae40cd03499d90a3705e45b6ddbbe243e73d7ed332d7, pinned repair source46aca8c3ec (production repairs integrated rootc419/cac; corpus734/7d12). Fresh installed CLI three-test gate exit0 in61.654s: capacity0.778s, asymmetric guest6.480/0.556s, deadlock7.218s, all strict<=10s with3m execution watchdogs. Named payload-bytes4096/used4096/delta64, partial streams, abortvirtual1005000001, inspect accounting and exact replay pass. Durable/canary/network detached replay and tamper regressions also pass.

The original unchanged concurrent-start Go API, own Go exact replay/inspection, linked trap/exit/capacity/prefix/repeat/World decoding, successful control/negative, and migrated Rust alternate import-v2 each exit0 on the final exact helper. The first import attempt had a missing reference-metadata receipt in the test setup; that failure is retained, and the continuation authenticates actual raw execution/module/wrapper digests without normalization before the successful fresh import. Final helper158 library+89 integration tests and all-target Clippy pass; duplicate observable unit tests were replaced by real corpus assertions. Integrated734 helper full tests/Clippy and host feature62tests/all-target Clippy also pass with authenticated relevant source-tree equality.

Fresh independent failfast_reopened_acceptance reviewer inspected source and raw original/repaired frames, actual final receipts and assertions, returning SHIP with no findings for R1-R4. Same-family Codex reviewer; no build/test claims attributed to reviewer beyond its inspected evidence.

stage: impl-review - ran (fresh failfast_reopened_acceptance; same-family Codex reviewer)
stage: plan-sync - skipped(config: planSync.enabled != true)

Authoritative raw evidence: /home/agent/wt/evidence/fn232-replay/final235-* and integrated734-* (argv,status,stdout,stderr); original RED frame /Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-replay-alternate-wire/1356714.stdout. Original ringpop120 success and prior RED watchdog proof remain referenced by the root handover/earlier Git task summary. fn235.2 performance acceptance and fn232.13 final frozen qualification remain open, with no waiver or pass supplied here.
## Evidence
- Commits: c419998757, cac37807a9, 734bc7c566, 7d12c6d58a, 294b5e66e5
- Tests: final235-installed-strict: exit0,61.654s; capacity0.778s/guest6.480s+0.556s/deadlock7.218s all<=10s with3mwatchdogs; inspect/streams/aborttime/exactreplay/detachedcanaries/tamper pass, final235-original-alternate: exit0; unchanged original concurrent GoAPI own inspection+exactreplay on helper71f7, final235-linked-prefix: exit0; trap/exit/capacity+prefix exactreplay/repeats/Worlddecode, successful complete control then incomplete success prefix refusal, final235-alternate-import-v2: exit0; actual Rust alternate own replay, repeated bytes, terminal coordinates and empty never-entered guest facts, final235-helper-tests:158library+89integrationpass; final235-helper-clippy:exit0, integrated734-helpertests/Clippy and hostfeature62tests/alltargetClippy:all exit0/source-freeze0, fresh independent failfast_reopened_acceptance: SHIP R1-R4 nofindings
- PRs:

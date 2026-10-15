# fn-174.6 R5 re-measurement after the transcript-hash fix (Linux/arm64, 2026-10-15)

**Verdict: R5 PASS.** Encoding + transcript-hash per-seed median reduction is 93.2 % / 93.1 % / 92.5 %
(seeds 0 / 1000003 / 2000006). The bound is >= 80 % (candidate <= 0.20 x baseline); measured 0.068 / 0.069 / 0.075.

This supersedes the 2026-10-14 shortfall record (`fn174-host-call-profile-2026-10-14.md`, 71.1 / 73.1 / 73.2 %),
which is kept as historical evidence. The fix was `0a83649fc5` (merge of `fn174-hash`): sha2 `asm`
(aarch64 SHA-2 instructions), byte-identical digests.

## Host and identities
- Real Linux/arm64 host, 12 cores (`host_cpuinfo` in each JSON lists `sha2`), boot `ad1ec4fe-2bb8-4c96-bf7f-a714910117e6`.
- Both sides ran in the same boot, one after the other.
- The host was quiet: load at stage start 0.25 / 0.99 / 1.32 (< 1.5), with no other build/test/review processes and the fn-176 worker suspended.
- Candidate source `3ad0fc9b3f` (fn174-hash tip, content-equal to the gomad fn-174 code at `0a83649fc5`); diagnostic helper sha256 `3c62aee4ba55d9fb053d31172c0ce89b706f8a32d65ab14377df527c3a00cca3` (contains `sha256h`).
- Baseline source `418c95a30b` (pre-fn-174), diagnostic helper sha256 `ea7ce53a47bc34327327fefe08288913cfe936c775f72a22c004f8a593d3dec5` (unchanged from the 2026-10-14 run).
- Candidate release helper for the runtime-cost identity check: sha256 `7c713db8c7f3fe35e907e10b6b698637aba4807055033f04207ee1eb75043479`, built by `make -C internal/backends/wasm helper` from `3ad0fc9b3f` (sha2 `asm`, 32 `sha256h`; matches `helper_sha256` in `runtime-cost-go-metrics.json`). It differs from the worker's `19493564…cab3` only because the target directory differs.
- Shared module `tests.wasm` sha256 `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`; selection `^TestActivityTestSuite$`; 3 repetitions per seed per side; collector hash in `fn174-collector.sha256`.

## Results (encoding_plus_hash_wall_nanos)
| Seed | Baseline median (range) | Candidate median (range) | Reduction |
| --- | --- | --- | --- |
| 0 | 247.4 ms (238.9-249.2) | 16.8 ms (16.7-16.9) | 93.2 % PASS |
| 1000003 | 242.1 ms (242.0-255.4) | 16.8 ms (16.8-16.8) | 93.1 % PASS |
| 2000006 | 241.5 ms (241.5-242.1) | 18.0 ms (17.2-18.2) | 92.5 % PASS |

Gain against spread: the gain (about 225 ms per seed) exceeds the combined range of both sides (at most 14 ms) by more than an order of magnitude.

Go-owner/helper-owner evidence identity: `TestHelperRuntimeCostProfile` PASS (12.07 s) on the candidate release helper `7c713db8…3479`.

Raw data: `fn174-host-call-profile-rerun-2026-10-15/` (both JSON reports, runtime-cost metrics, stage log, helper hashes);
full trees in `/home/agent/wt/perf-measure-20261014/fn174-{candidate,baseline,runtime-cost}` (local).

## Rerun
```sh
git -C /home/agent/wt/merge merge --ff-only fn174-hash
# rebuild diag-CAND per /home/agent/wt/fast-RUNBOOK.md fn-174 step 1 (single-link copy)
RERUN="fn174-candidate fn174-baseline fn174-runtime-cost" ONLY="fn174-candidate fn174-baseline fn174-runtime-cost" /home/agent/wt/measure-all.sh
```

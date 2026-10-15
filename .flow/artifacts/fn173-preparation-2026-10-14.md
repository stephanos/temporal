# fn-173.5 warm preparation and evidence equality (Linux/arm64, 2026-10-15 UTC)

Owner policy: MILESTONES "WASM performance acceptance host". Baseline and candidate ran on the
same real Linux/arm64 host in the same boot, back to back, on a quiet host (other workers suspended).

## Verdict

- **R6: PASS.** Median warm `prepare_nanos` is **467.3 ms** (min 426.3, max 495.9 ms, 5 fresh-helper runs,
  cache hit on every run), against the re-measured pre-fn-173 baseline of **3534.2 ms** (min 3394.4, max 6443.9 ms).
  `cache_read + deserialize` is 391.9 ms (< 1 s), so the parked mapping decision is not needed
  (`parked_mapping_owner_decision_required=false`).
- **R2 (end to end, success path): PASS.** Terminal frame to helper reap: median **7.35 ms** (7.01-7.94 ms), bound 1 s.
- **Same-seed evidence: identical.** `candidate_comparison_sha256 = baseline_comparison_sha256 =
  d5d5a04de5b9d170a608cd8dadb290f345b66443b72295b7a39e66ff2be9e803` (typed GMWT owner comparison of choice,
  WASI, network, stdout/stderr and terminal evidence; seed 0). Each series is internally identical
  (baseline raw evidence 188138833574…, candidate 859a0420c8a3…). The raw digests differ across series by
  construction: fn-174 replaced the JSONL transcript encoding with the binary transcript, so
  the comparison runs through `warmPreparationBehavior` (see perf-RUNBOOK B4).
- `acceptance.json`: `accepted=true`, `failure=""`.

## Host

- `Linux 7.0.14 #1 SMP PREEMPT aarch64`, 12 CPUs, 15 GiB, no swap. `/proc/cpuinfo`: CPU architecture 8,
  implementer 0x61, Features `fp asimd aes pmull sha1 sha2 crc32 atomics … i8mm bf16`. It is a native arm64
  Linux guest (not emulated); the drivers' `diagnosticPerformanceHostEvidence` admitted it.
- boot_id `ad1ec4fe-2bb8-4c96-bf7f-a714910117e6` (both series). Load average at stage start: 1.07 (B2), 0.95 (B4).
- CPU evidence copy: `/home/agent/wt/measure-logs/cpuinfo.txt`.

## Identities

| | Baseline (pre-fn-173) | Candidate |
| --- | --- | --- |
| Source | `a90b43307c` (historical collector on pre-no-unwind parent `15119dbbeb`) | `measure` `15b491281c` (gomad `10abe5f573` + B4 reader fix) |
| Helper (diagnostic-profile) sha256 | `5a77630b9b11ab591ba02caa36bc069e3e70a82345bd1151a04c99b353591216` | `8638b64376f158662e870e9fdff915f312ec83565d1a7b5c8c1c21f607215f96` |
| Engine configuration | `cranelift-fuel-nan-canonical-v1` | `cranelift-fuel-nan-canonical-nounwind-v2` |
| Model sha256 | `b30ccfca…387a` | `42c1f68b…6e46` |
| Module sha256 (shared) | `5835781d9896516a4a6cf1b3ab5dc10fc7fe8cb14a92297776d7c93e13d0f56c` | same |
| Runtime identity | `1d17508f…7f7e` | same |
| Config sha256 | `05bec491…1032` | same |
| Compiled artifact / envelope bytes | 724 644 936 / 724 645 381 (cache v2) | **693 908 480** / 693 908 906 (cache v3) |

Selection: Temporal `./tests` `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` (sqlite), seed 0, 4 GiB.

## Raw numbers (ns, measured runs)

Candidate:

| run | prepare | cache_read | envelope_verify | deserialize | cache_identity | whole run | terminal→reap |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 495 850 250 | 171 863 791 | 28 780 209 | 249 438 709 | 39 626 500 | 934 844 667 | 7 349 250 |
| 2 | 426 318 292 | 185 375 875 | 28 740 416 | 168 052 000 | 38 974 416 | 857 167 917 | 7 346 042 |
| 3 | 467 271 542 | 177 825 583 | 30 220 875 | 214 116 875 | 39 753 584 | 920 379 334 | 7 940 750 |
| 4 | 476 875 333 | 184 029 292 | 29 439 291 | 219 327 584 | 38 710 792 | 928 979 708 | 7 151 292 |
| 5 | 441 641 750 | 180 561 000 | 28 892 333 | 188 722 875 | 38 403 667 | 895 175 459 | 7 011 417 |
| median | **467 271 542** | 180 561 000 | 28 892 333 | 214 116 875 | 38 974 416 | 920 379 334 | 7 346 042 |

Baseline (module_read ≈ 50 ms and module_hash ≈ 925 ms are also on its per-run path; the candidate has 0 for both):

| run | prepare | cache_read | envelope_verify | deserialize | cache_identity | whole run | terminal→reap |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 6 443 917 711 | 519 564 917 | 2 008 161 501 | 2 847 044 168 | 38 035 750 | 7 518 348 836 | 10 524 042 |
| 2 | 3 770 239 585 | 403 711 209 | 1 973 352 959 | 366 734 792 | 39 381 875 | 4 752 406 836 | 9 335 625 |
| 3 | 3 534 207 751 | 268 163 208 | 1 978 957 917 | 225 613 458 | 41 206 667 | 4 315 066 919 | 8 274 500 |
| 4 | 3 528 487 877 | 235 295 542 | 2 048 249 168 | 216 851 792 | 42 031 875 | 4 386 411 960 | 9 280 334 |
| 5 | 3 394 390 794 | 191 928 375 | 1 973 542 751 | 207 114 292 | 38 323 666 | 4 244 326 668 | 9 367 625 |
| median | **3 534 207 751** | 268 163 208 | 1 978 957 917 | 225 613 458 | 39 381 875 | 4 386 411 960 | 9 335 625 |

Envelope integrity: ≈ 693.9 MB verified in a median 28.9 ms ≈ 24 GB/s (R5 bound ≥ 2 GB/s).

## Evidence paths (outside the repo)

- Candidate: `/home/agent/wt/perf-evidence/warm/candidate/warm-preparation-2238557815/acceptance.json` (+ `candidate-collection.json`, per-run observations, `untimed-evidence-canonical.bin`)
- Baseline: `/home/agent/wt/perf-evidence/warm/baseline/warm-preparation-3393136570/series.json`
- Logs: `/home/agent/wt/measure-logs/fn173-B2.log`, `fn173-B4.log`; stage ledger `stages.log`.

## Driver fix needed for this run

The first B4 attempt refused the baseline with "historical raw observation does not support the declared
run". The merged-tree reader required the historical `TerminalToReapNanos` (monotonic) to equal exactly the
wall-clock difference of the JSON timestamps, which drop the monotonic reading. The real runs differed by
-42..+41 ns. Commit `15b491281c` allows max(10 µs, 0.1 %) and adds
`TestWarmPreparationReapConsistentToleratesOnlyClockSlew`. B4 was rerun in the same boot as B2.

## Rerun

`/home/agent/wt/perf-RUNBOOK.md` sections A and B; serial driver `RERUN="fn173-B2 fn173-B4" ONLY="fn173-B2 fn173-B4" /home/agent/wt/measure-all.sh`
(B2: `TestHistoricalWarmPreparationBaseline` in `$BASE/tools/gomad3/backends/wasm`; B4: `TestTemporalWarmPreparationAcceptance`
in `$WT/tools/gomad/internal/backends/wasm` with `GOMAD_WASM_WARM_BASELINE_RECEIPT` = the B2 `series.json`).

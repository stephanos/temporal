# fn-174.6 local host-call cost on the Temporal activity selection (Linux/arm64, 2026-10-15 UTC; host local date 2026-10-14)

Owner policy: MILESTONES "WASM performance acceptance host". Baseline and candidate ran serially on the
same real Linux/arm64 host and boot (`ad1ec4fe-2bb8-4c96-bf7f-a714910117e6`), quiet host, same module and seeds.

## Verdict

- **R5 encoding + transcript-hash reduction: FAIL (shortfall).** Per-seed median reduction is
  **71.1 % / 73.1 % / 73.2 %** (seeds 0 / 1000003 / 2000006); the bound is ≥ 80 % (candidate ≤ 0.20 × baseline).
  Encoding alone fell by 93-95 % (≈ 86-90 ms → ≈ 5-6 ms). The transcript-hash nested span fell only
  ≈ 59-61 % (≈ 147-153 ms → ≈ 60-61 ms) and is now 90 % of the remaining cost. The span count fell
  333 165 → 101 895 (seed 0), but per-span hash wall rose from ≈ 441 ns to ≈ 603 ns.
  The run-to-run spread is small (baseline ±1 %, candidate one outlier per seed), so the shortfall is not noise.
- **Runtime-cost fixture Go-owner/helper-owner evidence identity: PASS** (`TestHelperRuntimeCostProfile`, 14.8 s,
  release helper from `make -C internal/backends/wasm helper`).
- **Docs:** protocol.md/README already describe the binary transcript and helper-owned namespace (fast-batch);
  `grep 'canonical WASI JSON lines'` finds nothing.
- fn-174.6 **stays open**: the R5 performance threshold is not met. Follow-up for the owner: reduce the
  per-record `transcript_hash` cost (for example batched hashing, or fewer, larger digest updates), or
  decide on the threshold. Not implemented here.

## Host

`Linux 7.0.14 aarch64`, 12 CPUs, 15 GiB, no swap; `/proc/cpuinfo` CPU architecture 8, implementer 0x61,
Features `fp asimd … i8mm bf16` (copy: `/home/agent/wt/measure-logs/cpuinfo.txt`); `required_host_measured=true` on both sides.

## Identities

| | Baseline (pre-fn-174) | Candidate |
| --- | --- | --- |
| Source commit | `418c95a30b1fe45198f81db8d081249128e23c1a` (`/home/agent/wt/fast-base`, collector copied in unchanged) | `15b491281cdafbfcd7e8b6c8ea4447ac8fbc6208` (`measure`) |
| Diagnostic helper sha256 (`--features diagnostic-profile`, single-link copy) | `ea7ce53a47bc34327327fefe08288913cfe936c775f72a22c004f8a593d3dec5` | `14c3934735f96b7e52f8ae4329dbf75947c0b765eed3cbe827df0997cb43a99e` |
| Model sha256 | `sha256:7d4a6567f5cc6bd96f4664900e4ea0b1ad7e1c7681db61a3aeeab3f56d4bcfe1` | `sha256:42c1f68b5ffb266c50fe93d0cdf73bcfec7afa2e01fa6fb5cb95838a06d56e46` |
| Module sha256 (shared `tests.wasm`) | `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f` | same |
| Collector `host_call_profile_test.go` sha256 | `7aa9649516e405af9983abaae9403b147e6fc97665e75df77099c1652b867fc3` | same (file folded into `integration_test.go` by `6be76036ed`; the standalone file is `git show 6e5dec62d3:tools/gomad/internal/backends/wasm/wasi/host_call_profile_test.go`) |

Selection `^TestActivityTestSuite$` (sqlite), seeds 0, 1000003, 2000006, 3 repetitions each (evidence identity across
repetitions enforced by the collector), one untimed preparation run.

## Per-seed results (ns)

| seed | role | encoding count | encoding ns (median, min-max) | transcript_hash count | transcript_hash ns (median, min-max) | encoding+hash ns (median, min-max) | calls |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 0 | baseline | 436900 | 85,612,626 (85,287,409-85,880,065) | 333165 | 146,939,623 (146,931,605-148,649,798) | 232,819,688 (232,544,231-233,937,207) | 209441 |
| 0 | candidate | 66496 | 5,917,065 (5,089,093-6,578,567) | 101895 | 61,412,796 (60,740,276-75,785,740) | 67,318,843 (66,501,889-81,702,805) | 209441 |
| 0 | **reduction** | | | | | **71.1%** (need ≥ 80%; ratio 0.289, need ≤ 0.200) | |
| 1000003 | baseline | 460003 | 88,142,488 (88,092,574-89,125,285) | 354624 | 152,878,694 (152,763,503-153,868,768) | 240,971,268 (240,905,991-242,994,053) | 221077 |
| 1000003 | candidate | 68053 | 4,704,117 (4,681,472-5,620,132) | 103530 | 59,681,982 (58,832,834-60,156,820) | 64,860,937 (63,514,306-65,302,114) | 221077 |
| 1000003 | **reduction** | | | | | **73.1%** (need ≥ 80%; ratio 0.269, need ≤ 0.200) | |
| 2000006 | baseline | 455307 | 89,503,126 (88,144,418-97,150,129) | 350941 | 153,334,010 (153,112,622-154,299,544) | 242,837,136 (242,443,962-250,262,751) | 218663 |
| 2000006 | candidate | 67025 | 4,964,492 (4,728,927-11,191,886) | 103080 | 60,272,966 (60,169,795-60,801,677) | 65,134,287 (65,001,893-71,993,563) | 218663 |
| 2000006 | **reduction** | | | | | **73.2%** (need ≥ 80%; ratio 0.268, need ≤ 0.200) | |

## Evidence paths

- `/home/agent/wt/perf-measure-20261014/fn174-candidate/host-call-profile.json`, `…/fn174-baseline/host-call-profile.json` (per-run rows, digests, cpuinfo)
- `/home/agent/wt/perf-measure-20261014/fn174-runtime-cost/` (go/helper choice, metrics, pprof)
- Logs: `/home/agent/wt/measure-logs/fn174-{candidate,baseline,runtime-cost}.log`

## Rerun

`/home/agent/wt/fast-RUNBOOK.md` "fn-174 R5" (steps 1-5), or
`RERUN="fn174-candidate fn174-baseline fn174-runtime-cost" ONLY="fn174-candidate fn174-baseline fn174-runtime-cost" /home/agent/wt/measure-all.sh`.
Helpers must be single-link copies (`$M/diag-<tree>/gomad3-wasmhost`): Cargo hardlinks `target/release/<bin>`,
and helper admission refuses link count 2.

## fn-187 rerun mapping (2026-10-16)

The results above were produced by the `go test` producer of their recorded
checkout and are not rewritten. On a checkout that contains
`tools/gomad/internal/backends/wasm/cmd/wasmmeasure` (fn-187.2 onward) the same
producer runs as

```sh
cd tools/gomad && go build -o /abs/bin/wasmmeasure ./internal/backends/wasm/cmd/wasmmeasure
env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  /abs/bin/wasmmeasure <subcommand> --repo <absolute checkout> [--go <go>] <flags>
```

(wasmmeasure builds and runs the `--repo` checkout's own producers),

with every `GOMAD_*` variable of the recorded command passed as its flag
(`GOMAD_WASM_X_Y` is `--x-y`, `GOMAD_STOCK_GO` is `--stock-go`; an older
`GOMAD3_STOCK_GO` is `--stock-go` too) and no `GOMAD_*` environment, which the
command refuses. Admission, identity and host-evidence checks and the retained
artifact schemas are unchanged; new `rerun` records name the wasmmeasure
command. Producer mapping for this artifact:

| Recorded producer | wasmmeasure subcommand |
| --- | --- |
| `TestHelperRuntimeCostProfile` | `runtime-cost-profile` |

The candidate and baseline steps ran through `/home/agent/wt/measure-all.sh`,
which is not retained, so their producers are not mapped here.

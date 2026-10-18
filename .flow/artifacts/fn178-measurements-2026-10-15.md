# fn-178 matched release measurements and scheduling decisions (2026-10-15)

Tasks: fn-178.10 (parallel vs serialized, threads vs fibers) and fn-178.11 (R4/R8 speedup record, R5 memory).
Host: real Linux/arm64 (aarch64), kernel 7.0.14, 12 cores, 15 GiB, boot id `e1418388-bdd1-43e6-9075-095dc81cda06`, cpuinfo sha256 `c8bc4e34c28e21c46f5053758653996edebce4b7278861ea1578bcf918dbb641`. The conductor suspended all other workers for the timed runs (quiet host). Stock Go 1.27.1 sha256 `1675694e…5c64`; Rust/Cargo 1.94.1.

Raw machine-readable summary: [fn178-measurements-2026-10-15.json](fn178-measurements-2026-10-15.json) (per-seed distributions, per-run scheduler counters, derived bounds). Baton spike raw rounds: [fn178-baton-spike-2026-10-15.json](fn178-baton-spike-2026-10-15.json).

## Inputs and identities

| | Baseline (retained fn172) | Candidate (fn-178 in-process cluster) |
| --- | --- | --- |
| Producer | `TestMatchedReleaseClusterBaseline` | `TestMatchedReleaseClusterCandidate` (`gomad_cluster_candidate` tag) |
| Source | measurement checkout `51604889c9` = retained `15b491281c` + collector-only patches (instrumentation patch sha256 `330764a6…d6f5`) | frozen `b4aad2b1c73f4b2f604df0224ca728df099304d5` |
| Release helper (authenticated, single-job pinned Cargo, empty features) | sha256 `c03f65c0e9c884fa3a3dce0db4889380dab76e47a037099a0373d356ee8d1f26` | sha256 `2dbaa51d178585dbdf98f97bad6a0506cb14fe774910ba264d976048f2e90d1a` (driver helper-build, window 4) |
| Network owner | `go-reference` (per-process Go broker) | `in-process-rust` (owner digest = helper sha256) |
| Report | `/Users/stephan/Workspace/skunkworks/gomad-fn178-baseline-20261015.L1uv7u/matched-release-baseline-3480265058/matched-release-report.json` (sha256 `825ccc4c…8153`) | `/Users/stephan/Workspace/skunkworks/gomad-fn178-window4-20261015/candidate/matched-release-candidate-584232640/matched-release-report.json` (sha256 `82012ba2…6fbb`) |
| Run | window 1, 2026-10-15T19:31:51Z–20:30:25Z, exit 0, 3514 s | window 4, 22:15:22Z–22:30:24Z, exit 0, 902 s |
| Scenario | Temporal two-cluster partition/heal/whole-cluster-crash: crash, reopen, prefix replay, full replay | same |
| Seeds × reps | 7/19/42, 1 warmup + 3 warm each; all 12 passed; 3 distinct choice traces | same; all 12 passed; 3 distinct choice traces |

Candidate binding: the measurement is bound to `b4aad2b1c7`. The qualified frozen candidate is `e75ac7df25`. `git diff --stat b4aad2b1c7..e75ac7df25 -- tools/` is exactly `qualification/sqlite_durable_test.go` (test-only) and `QUALIFICATION.md` (doc), 19 insertions/33 deletions: no helper, wasi, production source or identity pin. The measured scenario never executes either file (conductor-accepted, 2026-10-15).

Exact rerun commands are recorded in each report's `rerun` field. Window scripts: `/home/agent/wt/fn178-tmp/quiet-window-fast.sh` (baseline, baton) and `/home/agent/wt/fn178-tmp/window2.sh` (candidate). Candidate and baseline test binaries were prebuilt (`go test -c`) outside the timed phase.

## R4/R8: authorized execution and parent Go cost (warm repetitions, seconds)

Primary metric: earliest helper authorization through latest classified and owned-reaped helper completion across every process and all four phases. Parent CPU is the whole-scenario RUSAGE_SELF delta (Go parent: broker/models in the baseline, cluster client in the candidate) and is compared as CPU, not added to wall.

| Seed | Baseline authorized (3 samples) | Candidate authorized (3 samples) | Median ratio | Candidate below every baseline sample | Parent CPU median base → cand | Whole-scenario wall median base → cand |
| --- | --- | --- | --- | --- | --- | --- |
| 7 | 237.4 / 199.1 / 486.2 (median 237.4) | 36.7 / 36.9 / 37.5 (median 36.9) | 6.43x | yes | 161.0 → 26.2 | 243.3 → 38.7 |
| 19 | 234.0 / 283.5 / 256.4 (median 256.4) | 34.9 / 34.2 / 34.9 (median 34.9) | 7.35x | yes | 165.6 → 24.6 | 265.2 → 36.5 |
| 42 | 136.2 / 176.9 / 166.5 (median 166.5) | 50.8 / 59.0 / 38.5 (median 50.8) | 3.28x | yes | 124.7 → 34.5 | 172.7 → 52.2 |

R4 holds on every measured seed: candidate authorized execution is lower than the baseline at the same seed, and the candidate maximum is below the baseline minimum (gain far exceeds spread). Parent Go CPU drops 4.7–6.7x (R8). The seed-7 baseline range is wide (one 486 s repetition); the conclusion does not depend on it.

## R5: memory

Candidate: the largest single cluster-helper own-image peak RSS (one sample per completed helper process, covering every resident guest) is 3.58 GiB, within the 8 GiB resident limit. The largest per-store guest linear-memory peak is 172 MiB against each store's 4 GiB `memory_bytes`. Every warm repetition hit the compile cache with zero compile time (asserted by the producer). The multi-instance qualification success/canary cells evaluate the same R5 rule through `EvaluateBudget` (see the frozen driver run).

## fn-178.10: parallel vs serialized (decision)

Scheduler counters are summed once per completed helper per scenario (both helpers, all four phases). `T` is the primary envelope; `G` the guest residual (execute wall minus timed host-import spans, an upper bound including ABI glue); `O` the runnable-overlap wall time.

- Runnable overlap: 51.2–51.4% of World-visible operation samples had more than one instance runnable; overlap duration fraction 0.51–0.52 of active time.
- Guest share `G/T`: 0.148–0.192.
- Best-case two-store bound `S_upper = T / (T − min(G/2, T/2, O))`: 1.08–1.106 across all nine warm runs (max 1.106).

Decision: stay strictly serialized. The best-case bound is far below the 1.5x threshold; guest execution is under a fifth of authorized execution, so even perfect two-way overlap of guest code cannot reach 1.5x while World/clock-visible operations stay globally ordered. No parallel follow-up is opened. The scheduler seam remains for a future re-evaluation if guest share grows (for example with fn-182 or heavier workloads).

## fn-178.10: threads vs fibers (decision)

Current-host spike (`store-handoff-spike 200000 7`, binary sha256 `d7234590…633b`, 7 interleaved rounds, quiet host): thread baton Condvar 11,164.5 ns/switch (median; raw 9.8–15.9 µs), park/unpark 10,523.4 ns, async fibers 151.5 ns; typed sync import 2.0 ns (sync store) / 2.1 ns (async store), never-yielding async import 21.9 ns.

Baton handoffs per scenario (guest idles, `handoffs`): 2,586 (seed 7) and 2,614 (seeds 19, 42); grant samples 2,590/2,618. Share = handoffs × 11,164.5 ns / T = 0.05–0.09% of authorized execution.

Decision: threads stay. The share is roughly 60x below the 5% trigger, so moving to Wasmtime async fibers (async feature, four new Cargo.lock packages, async handoff imports, `call_async`) is not justified. No Cargo.lock change.

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
| `TestMatchedReleaseClusterCandidate` | matched-release-candidate (no `gomad_cluster_candidate` tag any more) |
| `TestMatchedReleaseClusterBaseline` | none in this tree: the baseline producer lived in the measurement checkout and reruns there unchanged |

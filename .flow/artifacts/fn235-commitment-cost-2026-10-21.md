# fn-235.2: World inspection commitment cost in long cluster runs (R5)

Raw evidence for the matched cohort is in `/home/agent/wt/evidence/fn235/cohort/`. Its contents:

- Scripts: `cohort.py`, `run-cohort.sh`, `analyze.py`, `build.sh`.
- Instrumentation: `progress-instrumentation-10s.patch`.
- Per-run `<side>-<rep>/` directories with `command.json`, `scenario.json`, host snapshots, stdout/stderr, progress JSONL and `measurement.json`.
- Results: `analysis.json`, `replay/`, `compare-before1-after1.json`, `release-check/`.

Pre-change attribution and implementation history are in `/home/agent/wt/evidence/fn235/commitment/` (`HANDOVER` = `../commitment-HANDOVER.md`). Integrated gates are in `/home/agent/wt/evidence/fn235/gates-tip/`.

## Attribution before the change

The actual three-node ringpop scenario ran for 60 s under the pre-change helper. Its profile (`commitment/before-60-profile/perf.data`, `perf-report.txt`; private helper copy, `perf --no-buildid-cache`, zero lost samples) attributes:

- 87.13% inclusive to broker inspection World commitment (`cluster/handoff/broker.rs`, rebuilding the commitment over full history on every revision);
- 75.90% self to SHA-256;
- 48.23% to the World state digest.

After the World-only change, the residual named site was endpoint network inspection: 62.51% (`commitment/after-60-profile2`). The same composed path was then extended to it. After the change, a phase profile of the integrated helper showed no history-growing site, only bounded current-state validation at a near-identical share in both windows (`commitment/final294-progress-120-phase-profile-retry/`). Red/green regressions pin the fix:

- World: 3,001 history materializations → 0 (`0bb28f6104`).
- Network: 1,000 native-history rehashes → 0 (`5bb93e0d4e`).

## Change (integrated in `gomad`)

- `d742092f68` binds inspection World v2 to its transition chain, with incremental accounting.
- `503f7afd1a` chains live network inspection transcripts.
- `dcf3c307e2` seals network history.
- `c04eadea6d`/`711e0a9d58` rotate the corpus mirrors and symbol goldens.

Declared rotations:

- inspection World encoding v2;
- `gomad.inspection-state-sha256/v2`;
- helper implementation identity, through the source-bound mechanism;
- installation `ModelSHA256` from `d6ca1cda…` to `e5541190…`.

Canonical full World, network and native evidence bytes, guest scheduling and choice records are unchanged (below). Prior encodings are refused at admission.

## Matched cohort (same host, same window, 2026-10-21 10:28–11:58 PDT)

| | before | after |
| --- | --- | --- |
| Source | `e6b832e036` (fn-235 branch base, pre-change) | `92c69594f2` (`gomad`; same cluster/World/network inspection source as the tip `3ad88e1cde`) |
| Launcher | `sha256:7c3a0511ccb0…` | `sha256:389584d37157…` |
| Helper (+10 s checkpoint patch) | `sha256:e00cbfe3b617…` | `sha256:0551d3acb014…` |
| Model | `sha256:d6ca1cda8a26…` | `sha256:e55411900b15…` |

Both sides used the same:

- Guest: `testdata/ringpop` from fn-224, module `sha256:b0462afc1b7a…` on both sides.
- Scenario: the checked-in `ringpop.json` with only the deadline set to 120 s (partition at 10 s, heal at 30 s).
- Run settings: seed 7, `ringpop-config.json`, 1 GiB memory, 2^50 fuel, 40 m/45 m watchdogs.
- Compiler: frozen stock go1.27.1 `sha256:1675694ef690…`.

Each side used its own runtime source. The helper patch only appends host-monotonic wall time at each 10 virtual seconds to a JSONL file. It changes no guest, model, evidence or scheduling behavior. The instrumented and release helpers give identical inspection fields, World and choice records; only identity fields differ (`release-check/compare-instrumented-vs-release.json`).

The cohort order was one warmup per side, then three counted pairs in ABBA order (1: before, after; 2: after, before; 3: before, after). All 8 runs exited 0 and retained one success each, with exactly 12 checkpoints. Record hashes are identical within each side: before `sha256:af9f5c30cc8f…` ×3, after `sha256:e74207afedaf…` ×3.

Wall seconds per virtual second (medians over reps 1–3; range in brackets):

| Window | before | after |
| --- | --- | --- |
| 0–10 (literal) | 0.646 [0.678] | 0.127 [0.101] |
| 10–60 (literal) | 4.605 [5.936] | 0.159 [0.121] |
| 60–120 (literal) | 17.854 [5.973] | 0.197 [0.190] |
| 30–60 (post-heal steady) | 6.952 [9.789] | 0.209 [0.148] |
| 60–90 | 19.073 [6.311] | 0.206 [0.164] |
| 90–120 | 20.468 [9.466] | 0.187 [0.217] |
| Within-run ratio 90–120 / 30–60 | 1.61 / 2.03 / 4.25 | 1.22 / 0.90 / 0.72 |
| Command wall | 1835 / 1249 / 1245 s | 55 / 48 / 15 s |

The before side grows superlinearly with accumulated history: its 60–120 cost is about 3.9× its 10–60 cost, and every run's late steady window costs more than its early one. The after side is flat within measured spread. The change between the 30–60 and 90–120 medians is −0.022 s/s, below both windows' rep-to-rep ranges (0.148 and 0.217). Within-run ratios straddle 1 (median 0.90). The literal 10/60/120 checkpoints rise less than their own range. Part of that rise comes from the low-traffic partition window (10–30 s), which lowers the 10–60 average, as the retained density analysis records.

Absolute after-side spread is dominated by host load: rep 3 ran on a quieter host. Interleaving and the within-run ratios control for this. Median command-wall speedup is 25.9×. This is a measured result for this workload only and no general performance claim.

## Identity and replay

Before vs after (`compare-before1-after1.json`):

- Equal: module, canonical World, virtual elapsed 120000000001, and per-node choice-tape records, source/target bounds and payload hash. Faults, history, outputs, network and transitions are equal too.
- Different: identity, node specs (helper and implementation SHA in the node config only), spec digest, static identities, the tape implementation binding with its header checksum, and the inspection `state_encoding`/`world`/`world_encoding` fields, which are the declared v2 rotation.

Fresh retained replay of the measured workload: all three counted after-side successes replay with `reproduced=true diagnostic=false result=success choice-replay=none` (`replay/after-{1,2,3}.out`). An uninstrumented build of the same source (`39694cd12b`, helper `sha256:02169018a92e…`) ran the 120 s workload to success and replayed it with `reproduced=true` (`release-check/`). The same build ran the checked-in 90 s scenario seeds 7 and 19 to success with their fault-phase proof, and both replayed (fn-224.3).

## Gates on the integrated tip

These ran on `3ad88e1cde` (+ `.flow`-only `4466cc160e`), private Cargo targets, in `gates-tip/`:

- Helper release suite: exit 0, 162 library tests plus integration targets.
- `helper_corpus` under diagnostic-profile, diagnostic-fuel-off, guest-profile and guest-profile+diagnostic-fuel-off: exit 0.
- `go test -tags test_dep` with `GOMAD_STOCK_GO`:
  - `internal/backends/wasm/cluster/...`: exit 0, 337.8 s.
  - `internal/model/world/...` and `internal/model/network`: exit 0.
  - `internal/backends/wasm/command`: 74 PASS, 1 FAIL. The failure is `TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs/gomad.wasi-cooperative/v1` with `finalize artifact manifest: invalid choice trace identity`. That test passes on `92c69594f2` (`at92c6-installed-operator.log`), so it is the `38572d7e49` Go-host regression (choice trace v4 vs Go record validation v3), not this change. It belongs to fn-232; see the fn-224 artifact.

The first `go test` attempt (`go-scenario-world-network.log`) omitted `GOMAD_STOCK_GO` and is retained as an infrastructure receipt.

## Limits

- The measurement used `92c69594f2` and not the tip. The tip's Go launcher cannot run cooperative cluster scenarios (`decision/trace mapping`, `38572d7e49`), and the inspection/commitment source between them is unchanged (`git diff 38572d7e49 gomad -- wasmhost/src` touches only a moved test).
- Qualification of the rotated contract on the frozen Rust candidate stays with fn-232.13. This task supplies the fresh-replay, corpus and identity evidence for the rotation.

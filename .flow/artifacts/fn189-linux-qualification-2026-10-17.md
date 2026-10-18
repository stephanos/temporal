# fn-189 Linux WASM qualification

Stopped at the user's request on 2026-10-17T21:21:17Z. All qualification-worker
processes are terminal. This record does not establish fn-189 R3 acceptance.
The owner retains commits; no commit, push, PR, CI dispatch or Flow completion
was performed by the qualification worker.

## Candidate and actual host

The initial candidate was `bad3c401b5e7dbdf8bc8afaca3d08cad4da154fe` on
`gomad`, executed on Linux/aarch64, kernel
`7.0.14 #1 SMP PREEMPT Mon Sep 28 07:01:01 UTC 2026`. The machine reported
21,242,953,728 physical memory bytes and no swap. Host observations, exact
compiler paths and source state are in `.flow/tmp/fn189-linux/host-identity.log`.
Darwin history is not execution evidence for this host.

| Input | Actual identity |
| --- | --- |
| Stock compiler | Go 1.27.1 Linux/arm64; `tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go` |
| Compiler executable SHA-256 | `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64` |
| Rust compiler | `rustc 1.94.1 (e408947bf 2026-03-25)` |
| Release helper SHA-256 | `4f136bdc6bfd2b9a9df88c1ffcc73fbc000617fc7b0d51a4dfee40b0875bbabc` |
| Runtime fixture module SHA-256 | `97a745597433330e23ebca2090925df9c4fc20355a6bfc6f24288c37a74f75a2` |
| Cooperative runtime identity | `7c220e2c89e83713598a0549b82d731b01e560d711dea865c5d62c301d14bc12` |
| Runtime implementation | `sha256:6940d4218be85ff27527c83573d5f00474382960964b1b37f56f06b0ececb079` |
| WASI model | `sha256:9482b9c5e810464896c29d569c269d9b7e6d841872949ffa6cd20cfd5a7b0744` |
| Engine | Wasmtime 47.0.3, `cranelift-fuel-nan-canonical-nounwind-v2`, Linux/aarch64 |

`make -C tools/gomad/internal/backends/wasm helper` exited 0 using Rust 1.94.1
and `--release --locked`. Its Cargo dependency file names this actual checkout's
Rust sources; a copy is retained as `helper-source-dependencies.d`. The helper
is a regular single-link file. No historical Darwin helper was reused.

The initial 5,149-entry regular-source manifest SHA-256 is
`58a60cce5bebde164aafa0eac0e6cd3137e0110446cf6db17311fce4fa7eba19`.
It covers tracked Go/module/Rust/Cargo/model/fixture/config/schema/tooling inputs
selected by `setup.sh`; `.flow`, `.plans` and `.github` are excluded.
Two tracked symlink mappings are separately retained in `source-symlinks.log`;
their regular target files are in the manifest. The earlier 5,151-entry manifest
included symlinks and was correctly refused by integrated-source validation.
That invalid setup attempt remains `workloads-invalid-manifest.log`.

## Verified gates and interruptions

All paths below are relative to `.flow/tmp/fn189-linux/`.

| Gate / exact command owner | Exit and evidence |
| --- | --- |
| `make -C tools/gomad/internal/backends/wasm helper` | 0; `helper-build.log` |
| `make -C tools/gomad wasm-test-helper`, before fixture repair | 2 (Cargo 101); 248 unit and 5 clock tests passed; cluster 18 passed/1 failed; `helper-tests.log` |
| `cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --test cluster cluster_instances_match_their_single_runs_and_isolate_failures -- --exact --nocapture` | 101; deterministic reproduction, `helper-cluster-fuel-repro.log` |
| `make -C tools/gomad wasm-test-helper`, after owner fixture repair | 0; 420 passed across unit/clock/cluster/engine/network/World, 2 existing ignored; `helper-tests-after-fuel-fix.log` |
| `TestIntegratedFailureArtifactFreshObservedReplays`, fresh pinned-Go backend test binary | 0; 20/20 fresh helper replays with identical raw output/evidence after captured-input removal and unavailable compiler; `artifact-replay.log`, `artifact-replay/` |
| Canonical `qualify-runtime` under initial regular-source manifest | 2; seeds 0–17 each passed 25 recordings and 1 exact replay; seed 18 correctly refused the owner's changed Rust test fixture; `runtime.log`, `runtime/` |
| `TestPrepareTemporalQualificationInputs` and staged fixture `go list`/`cmp` preflight | 0; configuration-only producer, `prepare-inputs.log`, `inputs/`, `temporal-staged/` |
| Initial canonical `qualify-workloads` | Explicitly terminated after two passing controlled fixture cells; concurrent source/runtime work makes these preliminary correctness observations, not final cost acceptance; `workloads-preliminary.log`, `workloads-preliminary/` |
| Dedicated Temporal inspection | Shard completed all assertions and published its retained inspection; deliberately interrupted as workflow began, per root's integrated-batch decision. Parent exit 1; no three-cell pass claimed. `inspection.log`, `inspection/shard/` |
| Historical cluster baseline source controls | User stop interrupted compilation before selected tests ran; exit 143. Validated process group 252845 and descendants were terminated and reaped; `cluster-baseline-source-controls.log` |

The partial runtime reports are not a completed 32-seed qualification. The source
manifest changed because the owner repaired the Rust fixture; no runtime
behavior divergence was observed. A fresh final source freeze and full run
remain required.

The completed preliminary shard case retained record
`sha256:e9a4ba49ad269802f12ee0c1582924f5e378c70535cc4beedd2e373c876a7944`,
module `3b81c8fccdfeb679d9b8d6dec5ac9dd455a5ac2e9fe3da29c29f67de5a564111`,
`inspection/shard/inspection/debug.json` (6,241 bytes) and `heap.pprof`
(5,126 bytes). The final success log follows every shard assertion, including
detached exact replay, authenticated stopped-prefix inspection, useful snapshot
and probe bytes, heap/stacks and unchanged retained artifact evidence. Workflow
and onebox did not execute. The enclosing process was intentionally interrupted
after `=== RUN .../workflow`, so its exit 1 is not represented as a full gate pass.

## Fuel fixture diagnosis and owner repair

The Rust cluster test expected its `2-fuel` instance to exhaust the unchanged
20,000-fuel allowance. Its 2,000-iteration WAT guest actually consumed 12,024
fuel under the pinned engine: an allowance of 20,000 exited successfully with
7,976 remaining; an allowance of 10,000 exhausted fuel; 1,000,000 exited with
987,976 remaining. This was deterministic and separate from the historical
automatic-cluster finish-order race.

`fuel_probe.py`, `cluster-fuel-guest.wat`, `helper-cluster-fuel-probe.jsonl` and
the exact failing selector retain the reproducer. The owner changed only the
test guest loop from 2,000 to 4,000 iterations. The 20,000 allowance, expected
capacity outcome, single/cluster equality assertions and production limits
are preserved. The repaired guest consumes 24,024 fuel; the same 20,000
allowance now exhausts fuel, while 1,000,000 exits with 975,976 remaining.
The post-repair probe and full canonical Rust suite both exited 0. Production
helper bytes and runtime/model identities did not change.

## Required remaining qualification

The final candidate still requires the complete 32×25 runtime gate and 32 fresh
exact replays; canonical ten-cell workload corpus plus aggregate; all four SQLite
selected crash states plus catalogue aggregate; Temporal inspection's three cells;
Temporal storage/network/independent-cluster gates; and actual installed Linux
kill/resume and detached replay coverage. Source-gate results are owned by the
root's source worker and must be combined with these results.

`run-workloads.sh` supplies the exact canonical corpus environment.
`run-extra.sh` supplies dedicated inspection, storage, network, independent-cluster,
installed-operator and selected-SQLite commands. Every cost acceptance run must
have an exclusive host window. The preserved limits are preparation ≤10 minutes,
execution/replay ≤15 minutes, compiler and cold single-helper peak ≤6 GiB,
warm single-helper peak ≤4 GiB, aggregate cluster helper peak ≤8 GiB, fresh
operator lifetime peak ≤2 GiB, at most two resident guests and one visible
continuation. Missing measurements and capacity refusals remain failures.

The current qualification contract separately retains the fn-178 R4/R8 release
cluster comparison: baseline revision `15b491281c`, actual collector descendant
`51604889c9`, seeds 7/19/42, one warmup and three measured repetitions per seed,
all crash/reopen/prefix/full replay phases. Candidate authorized execution must
be lower on every matched seed; parent CPU and helper/compiler observations
remain separate. The historical external reports are absent, so fresh producers
must authenticate fresh source/helper/module inputs. The reconstructed baseline
checkout is `/home/agent/wt/fn189-cluster-baseline`; its actual current-rendered
instrumentation patch SHA-256 is
`10cc04990d1cfc27b315c1154481283126d8592b9a8cefe0b7ede66f629bf7c3`.
It is an unmodified checkout of the historical collector commit, not an invented
current-Go-reference replacement.

The five-run warm preparation comparator belongs to completed fn-173 R6/task .5,
not fn-189/187's current mandatory workload/resource contract. Its historical
source/helper/model/runtime identities and receipts remain historical. Root
confirmed that no renewed fn-173 optimization claim is required; admission,
threshold and negative-control preservation remain required, as do the current
canonical warm-memory and preparation budgets. Its old external baseline paths
are absent. Its clean-commit and baseline requirements were not bypassed.
The explicitly transferred fn-182 full/slim and fn-184 command timings remain
with fn-185. No performance improvement is claimed here.

## Retained workspace

This worker created the clean detached historical collector checkout
`/home/agent/wt/fn189-cluster-baseline` at `51604889c9` for the required fresh
release comparison. It remains needed and is worker-owned for eventual removal.
Other worktrees are inherited or owned by the root's concurrent workers and
must not be removed by this worker.
`.flow/tmp/fn189-linux/` is retained for active qualification and owner review,
including raw evidence, compiler caches and test-owned preparations. The stopped
inspection's interrupted workflow staging copy was moved out of `service/history`
into `interrupted-workflow-staging/`; it is recoverable and contains no original
production source. No qualification helper remains active from that stopped run.
Final cleanup/archive awaits the integrated final freeze, active gates and
evidence handoff.

At stop, no worker-owned helper, qualification test, runtime loop, inspection
watcher or baseline compiler remained. No `service/history/.gomad-inspection-*`
staging directory remained. Retained worker-owned directories are the clean
`/home/agent/wt/fn189-cluster-baseline` checkout and
`.flow/tmp/fn189-linux/{artifact-replay,inputs,inspection,interrupted-workflow-staging,runtime,runtime-superseded,temporal-staged,workloads-preliminary}`.
The sibling `source-gates/` directory is root/source-worker owned. These are kept
for the requested stop and future continuation; nothing was deleted, reverted,
committed or published during stop handling.

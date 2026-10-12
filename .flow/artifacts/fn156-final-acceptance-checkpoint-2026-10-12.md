# fn-156 final acceptance checkpoint — 2026-10-12

Work resumed from the `wasm-layout-checkpoint-2026-10-12.md` stop state under the owner's goal to complete fn-156. It stopped again when the sandbox root filesystem became read-only (EROFS on `/tmp` and tool output), probably from disk exhaustion by worktrees and evidence under `/tmp/fn156-final`. This is not Flow completion. Raw logs referenced below lived under `/tmp/fn156-final` and may not survive the sandbox.

Host: linux/arm64 sandbox, stock Go 1.27.1, Rust 1.94.1, Wasmtime 47.0.3. Native execution is unsupported on this host; native-host test failures (deterministic I/O platform check, missing `.toolchain/bin/go`) are excluded and remain with fn-128/fn-149.

## Commits on `gomad`

| Commit | Change | Task |
| --- | --- | --- |
| 7299fb6be9 | Resume and shard validation use the recorded profile environment | fn-164.5 |
| d36476f98a..b3eee0867d | Purity checker: promoted/externally satisfiable interface calls, callback hooks, stdlib stream wrappers, archcheck dead code and per-checker budgets | fn-165.4/.6 |
| 8d584361a7..8e249316e3 | Native installation precedence and top-level grammar, WASM native-import guard, gomad3sim CI routing, backend-selector error order, checked-in pre-move inspect fixtures | fn-164.5–.9 |
| 956331b83a | CLI derives choice-only coverage values from Runner | fn-164.7 |
| 6283a4ec90 | Pin integrated compiled model identity `sha256:f7fa0246…` | fn-164.6 |
| 6f024c292b | Portable tests follow relocated native owners | fn-164.3 |
| 4e9140c637 | Fresh campaign/plan/exploration validation uses the effective environment (fixed one-box `invalid_input`) | fn-164.5 |

## Gate results

Candidate 1 = 6283a4ec90 (frozen manifest, 1388 files). Candidate 2 = 4e9140c637.

| Gate | Candidate 1 | Candidate 2 |
| --- | --- | --- |
| `make -C tools/gomad3 validate` | 0 | 0 |
| Architecture/purity/public signatures, `GOMAD3_ARCH_ALL_PLATFORMS=1` | 0 (41s) | 0 (35s) |
| `make lint-code-fast` (base e43c7a59c7, Linux tools) | 0, 0 issues | 0 |
| Portable shared packages | stale-test failures | native-host failures only |
| `make -C tools/gomad3 test-wasm` | 0 (491s) | 0 (429s) |
| Rust helper test/fmt/clippy | 0 | not rerun (unchanged) |
| `qualify-runtime`: 32 seeds × 100 fresh + 32 exact replays + aggregate | 0 (805s); 30 choice traces, 32 outputs, 2 outcomes | runtime path unchanged |
| `prototype` (virtual time, network, scheduling, Temporal workflow, frontend smoke) | 0 | — |
| Temporal crash/reopen (`TestTemporalCommittedWorkflowFreshGuestCrashReopen`) | 0 (242s) | — |
| Temporal named network reset + exact replay | 0 (285s) | — |
| Two-cluster partition/heal/whole-cluster crash + replay | 0 (385s) | — |
| 100 fresh retained-artifact replays | 0 | 0 |
| `qualify-workloads` | fixture and SQLite cells passed; one-box `invalid_input` | unknown at stop |
| Installed operator gate (inputs deleted; positive minimization) | weak first version | 0 |

The `wasi` guest gates do not execute the changed Runner operations, so candidate 1 runtime evidence was retained for candidate 2. Helper fixes below change the helper identity, so every guest gate must rerun on candidate 3.

## Independent reviews

- fn-164 and fn-165 reviews: findings fixed in the commits above.
- fn-156 review of candidate 2: not ready. R6 workload corpus and .8 current-source matched costs unmet; G1 (retained cluster replay executes artifact-supplied helper bytes) high; G2–G5 and helper R1–R3 medium/low; CLI/README claims about 5× gate, installed proof and projection checks overstated.

## In flight at stop

- `fn156-rust-fixes` 0a089f32ec (unmerged): output charged by bytes written, remote transcript limit, cache failures classified as infrastructure. Helper `58e618e8…`. Check `backends/wasm/command` corrupt-cache exit status.
- `fn156-go-fixes` (unmerged, worktree `/tmp/fn156-final/wt-go`): a29f6b5304 G3 restart validation, 7f3b649e68 G2 ambient GOWASM/GODEBUG, c2d2f23148 G4 interrupted cache publication, 3bc609e5dd G5 shard environment, 650e334296 G1 installed-helper-only retained replay, 9f21cd3f00 matched-cost rigor (4 seeds, alternating order, derived toolchain identity, separate baseline cache), f95938c9fb documentation and `test-wasm` projection selection. Worker RED/GREEN passed and backend/simulation/command/qualification packages exited 0. Open: diff `./runner` failures against 4e9140c637 (three failures not yet proven native-only: two "compatibility pack identity is missing", one empty I/O profile in `TestManifestForRunBindsIOProfileIdentity`).

## Resume

1. Free disk space: remove `/tmp/fn156-final/{baseline-wt,fixwt,wt-*}` worktrees after salvaging unmerged branches, old dry-run evidence and duplicate `.wasm` modules; `git worktree prune`.
2. Finish and merge the Go and Rust fix branches; rebuild the helper.
3. Freeze candidate 3 and rerun every gate above, plus the SQLite selected-crash/catalogue gates and `TestActualTemporalClusterQualification` within `qualify-workloads`.
4. Run the full smoke prototype (`GOMAD_WASM_PROTOTYPE_ONLY=smoke`) to build `tests.wasm`, then `TestHelperMatchedSuiteQualification` against the baseline helper rebuilt from 436adbf127 (`bb16eeee…`, the helper behind the 2.911x measurement), with a separate empty baseline cache.
5. Independent re-review; complete fn-164, fn-165 and fn-156 tasks in Flow with one commit per task; update MILESTONES.md.

No PR, push or CI action was taken.

# WASM section acceptance evidence — 2026-10-12

One evidence handover for the integrated fn-164 layout, fn-165 checker and fn-156 productionization batch. It supersedes the stop states in `wasm-layout-checkpoint-2026-10-12.md` and `fn156-final-acceptance-checkpoint-2026-10-12.md`. Raw logs were local under `/tmp/fn156-final/cand3` and are not retained in Git.

## Candidate and environment

- Candidate 3: `0e3eb30e6c` on `gomad`; frozen source manifest of tracked `tools/gomad3`, `tests/gomadfunctional`, `go.mod` and `go.sum` files; postfreeze check verifies unchanged bytes and HEAD.
- Host: linux/arm64 sandbox, 12 CPUs, 15 GiB RAM. Stock Go 1.27.1 (copy outside the module cache), Rust 1.94.1, Wasmtime 47.0.3.
- Helper built from this checkout: `8a0a6038b22f5c063c02c31cb2fc29e79650ebfa29ef1b237111dcfb03a7a161`. Rust binaries embed source paths, so the same source built from another checkout has a different digest.
- Compiled WASI model `sha256:f7fa0246c20c821642ed3e13b5512be477f5bca77dd8667ad24b62f5467619fb`; cooperative overlay/runtime identity `b602475b745717550ff72f854755b769f885a51e5f221c81395fe78d45d0350b`.
- Native execution is unsupported on this host. Native-host failures (deterministic I/O platform check, missing `.toolchain/bin/go`, native CLI build) are excluded from claims and remain with fn-128/fn-149. The `./runner` failure set is identical before and after this batch's Runner changes (53 lines, compared against `4e9140c637`).

## Source gates (candidate 3)

| Gate | Command | Exit |
| --- | --- | --- |
| Rust helper tests | `make -C tools/gomad3 wasm-test-helper` | 0 |
| Rust fmt and clippy `-D warnings` | `cargo +1.94.1 fmt --check`, `clippy --release --locked` | 0 |
| Generation/protocol/boundary validation | `make -C tools/gomad3 validate` | 0 |
| Architecture, purity, public signatures, CI routing, three-platform vet | root selection with `GOMAD3_ARCH_ALL_PLATFORMS=1` plus `./internal/archcheck ./internal/puritycheck` | 0 |
| Canonical affected lint | `make lint-code-fast GOLANGCI_LINT_BASE_REV=e43c7a59c7 GOLANGCI_LINT_FIX=false` with Linux tools | 0, 0 issues |
| Portable shared packages | every non-backend package | native-host failures only |
| Complete WASM suite | `make -C tools/gomad3 test-wasm` | 0 |
| Workflow lint | `actionlint .github/workflows/gomad3.yml .github/workflows/gomad3-smoke.yml` (v1.7.7) | 0 |

## Independent reviews

- fn-164 and fn-165 reviews (candidate 6a99172b91): fixed by d36476f98a..b3eee0867d, 8d584361a7..8e249316e3, 956331b83a, 6f024c292b.
- fn-156 review (candidate 4e9140c637): fixed by 84ff3b0ae4..0e3eb30e6c (restart validation, ambient compiler settings, interrupted cache publication, shard environment, installed-helper-only retained replay, matched-cost rigor, documentation, helper output/transcript/cache classification).
- Guest verification found and fixed resumed-plan (7299fb6be9) and fresh-campaign (4e9140c637) environment validation defects.

## fn-156 final candidate (2026-10-13)

Final candidate `9d6012b52b` integrates the candidate 3 batch plus:
- the operator/helper memory fixes: VmHWM helper peaks; retained schema v2 references the module by digest; payloads released early; replay compares by record digest;
- the allowed-`hostfs` import fix;
- the independent re-review fixes: interrupted-incarnation wait upper bounds kept out of helper-image peaks; versioned refusal of retained v1 before installation/payload reads; target module bound before copy; symlink tests on genuine bytes; smaller Rust ballast;
- the revocation-after-reply race fix (`TestTransportRevocationAfterReplyInterruptsTheCrossingCall`; 8/8 under 12 busy loops).

Gates ran under the 2026-10-12 owner budgets. Raw logs were local under `/tmp/fn156-final/cand6` and are not retained in Git.

- Frozen manifest: 1,407 tracked files, digest `sha256:178ee82e4c344a25affdb248f5a55068f457f7c6630dbccfb1a64b6e5975c21b`. Postfreeze bytes and HEAD were verified after all guest gates.
- Helper `c2e4ff2f398101d9204aa86aba29ce6a0e220a366aca3366faee4371d5cc3857`.
- Compiled WASI model `sha256:9ef9041f502f4b489bc5d7cac906bb03467b797c885a1deae9e703150eb25cd6`; it rotated from `f7fa0246…` with the race fix in `wasi/process.go`.

| Gate | Result |
| --- | --- |
| Rust helper tests, fmt/clippy, `validate`, architecture (all platforms), canonical lint | exit 0 |
| Portable shared packages | native-host failures only; identical set to candidate 3 |
| `make test-wasm` | exit 0 |
| Scheduling qualification | 32 seeds × 25 fresh instances (800 record observations) plus 32 fresh forced exact replays; exit 0 |
| Retained-artifact fresh observed replays | 20/20; exit 0 |
| Installed operator gate | 79 steps, `fail=0` (minimization, exact replay, changed-helper refusal status 3) |
| Temporal committed-workflow crash/reopen | pass |
| Temporal named network reset/retry exact replay | pass |
| Temporal independent clusters partition/heal/whole-cluster crash | pass |
| `make qualify-workloads` | fixture ×4, SQLite durable success/canary, one-box success/canary, multi-instance success/canary, mandatory corpus aggregate: all pass |

Multi-instance success cell budget: WithinBudget.

| Measurement | Peak | Limit |
| --- | --- | --- |
| Operator | 1,856,929,792 B | 2 GiB; 4.92 GB on candidate 3 |
| Cold helper | 4,085,874,688 B | 4 GiB |
| Warm helper | 1,272,758,272 B | — |
| Conservative resident helper sum | 2,545,500,160 B | — |
| Compiler | 2,560,475,136 B | — |

No helper sample needed the wait-figure upper bound.

Failed attempts are preserved as history; none is relabelled:
- A first concurrent cluster run was OOM-killed by the host kernel: four parallel guest gates plus another session's JVMs exceeded 15 GiB. Run alone, it passed.
- The first scheduling run was refused by the 8 GiB cold-memory guard under concurrency. Run alone, it passed.
- The first workloads run failed only its multi-instance cell, because the driver passed the staged Temporal source without its `./` prefix (a Go import-path error). After correcting the driver, all cells passed.

SQLite supplementary selection gates need a native crash catalogue that this linux/arm64 host cannot produce. They remain native-owned (fn-128/fn-149).

Matched costs (`TestHelperMatchedSuiteQualification`, run alone, four alternating-order seeds, separate empty 0700 caches):
- Baseline helper rebuilt from `436adbf127`, the helper behind the 2.911x measurement.
- Module: a `./tests` module compiled with the baseline toolchain (`GOMAD_WASM_PROTOTYPE_ONLY=smoke-workflow` at `436adbf127`, 337,915,879 B). Modules from the current runtime import `gomad_wasm_v2::net_stream`, which the original helper refuses (`unknown import`). A matched module is therefore necessarily baseline-era, as it was for the accepted 2.911x figure.

| Suite | Authorized execution, baseline → candidate | Full wall, baseline → candidate |
| --- | --- | --- |
| workflow | 10.139 s → 2.712 s (3.739x) | 16.668 s → 19.196 s (0.868x) |
| activity | 49.294 s → 10.633 s (4.636x) | 55.221 s → 27.822 s (1.985x) |

The candidate's same-module, same-seed, same-tape execution speedup holds at or above the owner-accepted 2.911x. Workflow full wall remains below 1x because candidate preparation dominates that small suite. Earlier attempts were preserved:
- the current one-box module was refused by the baseline helper because of the ABI;
- the candidate refused a non-0700 compile-cache directory (fail-closed, operator error).

Final independent review of `0e3eb30e6c..9aa7e6634a` found nothing blocking; its five findings, nits and the gate-found race are fixed in `a936df0b75..9d6012b52b`.

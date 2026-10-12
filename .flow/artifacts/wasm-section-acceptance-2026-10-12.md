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

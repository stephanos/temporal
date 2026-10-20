# Gomad integration checkpoint — 2026-10-20

This is source-integration evidence, not candidate qualification or milestone completion.
Source candidate: `57d8b5f1ad9c7c597421d85a75524d42815a68bf`.

## Integrated source

- Sweep source `cc84570e2838dda3ec185764b5a099b284796ec9`: method/registration and nested-module discovery, selected-method verdicts, authenticated compile-only cache, bounded preparation/execution, clean-source admission and torn-journal repair. The independent sweep reviewer reported SHIP for the repaired patch; the compile-cache reviewer separately reported SHIP. Earlier wrapper-only v1 proof remains historical evidence and does not establish method coverage. Corrected live proof uses the isolated frozen sweep checkout, not this later integrated candidate.
- Determinism source `161c61798ad705e8942200f7b84a5fc064cb3e80`: genuine Runner stress retention/replay adapter, Go event-order parser/comparator, actual trace-to-Decision mapping, full-cohort perturbations and shared executable CAS. Exact Rust event diagnostics, capture, real CLI prefix replay and full stress execution are not established by this integration.
- fn-222 source mapping correction `5bc5a16988`: mandatory smoke is `TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds`, not `TestFrontendSystemInfo`. The latter remains required as a separate supplementary cell for the first completed support list. No gate, capacity or original minimum frontend requirement is removed.

## Focused verification

Compiler: `tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go` (Go 1.27.1); commands run from `tools/gomad`, with `GOMAXPROCS=2`, `-p=1` and `-tags test_dep`.

| Command after the compiler path | Exit | Scope |
| --- | --- | --- |
| `test -p=1 -tags test_dep ./internal/backends/wasm/cmd/wasmsweep ./internal/backends/wasm/command` | 0 | Sweep command and installation command contracts. |
| `test -p=1 -tags test_dep -run '^Test(Stress.*\|Compilation.*\|CachePublication.*\|OptionsObservationIsNotSerialized)$' ./internal/backends/wasm/backend` | 0 | Stress admission, compile-only cache and publication source regressions; no live guest gate. |
| `test -p=1 -tags test_dep ./internal/backends/wasm/internal/sweep ./internal/backends/wasm/determinism/... ./internal/backends/wasm/installation ./internal/backends/wasm/command ./cmd/gomad/internal/cli` | 0 | Source compatibility of the integrated batches and existing complete setup policies. |
| `test -p=1 -tags test_dep -run 'Stress\|CompilationCache\|CompilationIdentity\|Retention\|TemporalConfig' ./internal/backends/wasm/backend ./internal/backends/wasm/cluster ./internal/backends/wasm/internal/measure ./internal/backends/wasm/wasi` | 1 | Backend selection also matched live integration tests without `GOMAD_STOCK_GO`; those refused setup. Cluster, measure and wasi package selections exited 0. No guest failure or integration pass is inferred. |

Two earlier command-selection mistakes are retained here rather than hidden: the first source command named nonexistent packages `internal/backends/wasm/cmd/gomadsweep` and `internal/command` (exit 1; corrected above). A broader backend selection matched `TestProvenanceRejectsChangedIdentityBeforeExecution` and `TestProviderRejectsEmptyModelIdentity`, which require explicit integration inputs (exit 1); the source-only selection above then exited 0. These are not substitutes for the required integration gates.

`make lint-code-gomad GOLANGCI_LINT=/home/agent/wt/fn222/.bin/golangci-lint-v2.13.0 ERRORTYPE=/home/agent/wt/fn222/.bin/errortype GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=9cb5bc3149` exited 2. Log: `/home/agent/wt/evidence/integration-2026-10-20/lint-57d8b5f1ad.log`. Findings: two gci import-format issues (`determinism/event_order_test.go`, `determinism/observed/inspection_test.go`) and one testifylint assertion-style issue (`backend/determinism_integration_test.go`). The determinism owner is repairing them; this candidate is not lint-green.

`go test -p=1 -tags test_dep ./internal/archcheck` exited 0 in 58.469 s on the same source candidate. Log: `/home/agent/wt/evidence/integration-2026-10-20/archcheck-57d8b5f1ad.log`. `flowctl validate --all --json` exited 0: 59 specs, 420 tasks, zero errors and 36 warnings (including historical evidence/coverage warnings and fn-227's new heading/R-ID formatting warnings). Structural validity does not establish acceptance for any open task.

## Resource preservation

Removed unused, clean, merged checkouts `/home/agent/wt/cand2-frozen` and `/home/agent/wt/qualdriver` after checking their commits are ancestors of the integrated branch and no live process used them. The source commits remain in Git, the qualification-driver branch remains, and external evidence is retained. Only reproducible checkout/generated preparation copies were removed. Available disk rose to approximately 15 GiB; subsequent builds can change that figure.

## Subsequent integrated checks

At `28d2c881475249902cc4627bff975133ed7c2e45`, attribution source and all four feature-mode sidecar updates through `f172c17499` were integrated with determinism through `c60ae60207`. The independent attribution source/fixture reviewer reported SHIP. The registry merge retains both stress and cluster-owner producers. The broker merge retains both diagnostic World spans and supplemental event contexts. An actual compile check caught duplicate cluster-stderr accessors; integration retains one detached accessor, one completed-result clone and a regression that directly mutates the detached result to verify session isolation.

- Integrated `make lint-code-gomad` with the exact overrides above exited 0, zero issues. Log: `/home/agent/wt/evidence/integration-2026-10-20/lint-28d2c88147.log`.
- `cargo check --locked --all-targets --features diagnostic-profile` exited 0 (6.44 s), and the separate `--features stress-jitter` check exited 0 (5.90 s). Logs: `/home/agent/wt/evidence/integration-2026-10-20/rust-{diagnostic,stress}-28d2c88147.log`.
- An earlier combined `diagnostic-profile,stress-jitter` check exited 101 because the source explicitly forbids that feature combination. That guard remains intact; it was not a supported-role build failure. Log: `/home/agent/wt/evidence/integration-2026-10-20/rust-combined-28d2c88147.log`.
- After integrating context fixes through `791a61ed2d`, at root `dc5af510c610d7c577c22f6c9928fbe0e3fb9502`, fresh focused determinism/observed, sweep, measurement/measure and CLI package tests exited 0. `go vet -p=1 -tags test_dep` over backend, cluster, wasi, measure and CLI also exited 0. Exact supplemental state-commit diagnostics and live prefix probes still have their own fn-188 acceptance gates.

The corrected fn-221 two-package proof and same-candidate resume are retained in `fn221-method-proof-2026-10-20.md`. Root reran both committed v2 sidecar verifiers: exit 0 each. Independent final evidence review returned SHIP; `flowctl done` and a subsequent `show` verified fn-221.1 done before commit `c3ec0975e6`. Full survey and reason ranking remain open. The actual fn-185 early activity/runtime-cost proofs are separately retained in `fn185-attribution-port-2026-10-20.md`; they do not replace integrated qualification or full matched measurements.

The unused clean merged `/home/agent/wt/cand2` checkout was subsequently removed with its branch preserved. The adapter owner separately removed its own unused clean `/home/agent/wt/fn188-runner-adapter`, preserving all original source commits on its branch and all required evidence externally. No live checkout or raw execution evidence was removed.

At `1739cad84f0f4ed180275dcc2009f9adc2eb6e2f`, supplemental namespace/recovery commit diagnostics through `20a81a8191` are integrated. `cargo fmt --all -- --check` and separate locked all-target checks for empty features, `diagnostic-profile`, `stress-jitter` and `determinism-fixture` all exited 0. Logs are `/home/agent/wt/evidence/integration-2026-10-20/rust-{fmt,release,diagnostic,stress,fixture}-1739cad84f.log`. Live handoff/prefix acceptance remains separate from compilation.

Qualification preflight's static equivalence check found 101 reference stages minus five retired native stages, exactly 96 required DAG stages, identical commands and no problems. Initially suspicious tool paths were verified, not changed: stock trees `2198cf19890232fc` and `878c9b2935e2c9e6` have identical relevant source/compiler inventories and compiler SHA-256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`. The retained Linux/AArch64 lint binaries execute, have the same mandated versions/build inventories as fn-222's copies, and verify against their original manifest. Original driver bindings remain intact. No full candidate qualification has started at this checkpoint.

After checking no process depended on them, root removed reproducible Rust incremental and generator compilation caches. The unused root-owned `q1` frozen checkout was also removed after preserving its exact current helper at `/home/agent/wt/evidence/q1/retained-worktree-gomad-wasmhost` (SHA-256 `ed6cbaeafab2fb5bf9631100d943276da9015525a1648aee744dc6906209d189`) and both staged source fixtures under `/home/agent/wt/evidence/q1/retained-staged-fixtures/`. Copies were compared byte-for-byte before removal. Historical qualified helper/module/receipt evidence under `evidence/q1` remains untouched; the saved worktree helper is not relabeled as the qualified ordinary helper. The checkout's source is recoverable at ancestor `e50c6476fba40d3b31ddb20d1030abf08807412c`. An old matched-pair driver naming `q1` must recreate that checkout and its staged files; future measurements may bind a new frozen candidate. The transferred matched-release baseline checkout `fn189-cluster-baseline` remains. Free disk after removal was 14,332,912 KiB; this is preflight headroom, not a promise about later resource use.

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

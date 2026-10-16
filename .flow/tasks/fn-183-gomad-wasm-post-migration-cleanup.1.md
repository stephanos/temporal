---
satisfies: [R1, R3]
---
# fn-183-gomad-wasm-post-migration-cleanup.1 Isolate reference execution and remove residual migration routes

## Description
Audit the delivered fn178 routing and remove demonstrated residual production reference selection/obsolete plumbing for R1/R3.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/cluster/provider.go, provider_test.go, wasi/process.go, installation/installation.go, related routing tests
**Touches:** [tools/gomad/internal/backends/wasm/cluster/**, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/*test.go, tools/gomad/internal/backends/wasm/installation/**]

### Approach
- Re-anchor to fn178's completed routing. Inventory actual callers across cooperative clusters, standalone/stock execution, Go-broker differential tests and native/shared consumers. Do not reimplement fn178.9.
- Remove production selectors only if any remain; keep explicit test-only reference selection and existing stock/standalone ownership. Delete adapters/branches only after caller/profile analysis proves they are transitional and redundant. If a claimed leftover is already gone, record verification rather than invent replacement work.
- Pin behavior with Go-broker versus Rust-owner parity fixtures comparing replies, per-incarnation outcomes, evidence bytes and global order before/after cleanup. Retain conformance generation and reference-source drift checks.
- Test that normal installation/config cannot silently choose reference execution, and that reference tests still select it deliberately.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/cluster/provider.go
- tools/gomad/internal/backends/wasm/cluster/provider_test.go:593
- tools/gomad/internal/backends/wasm/wasi/process.go
- tools/gomad/internal/backends/wasm/installation/installation.go
- tools/gomad/internal/backends/wasm/wasi/integration_test.go

### Quick commands
From tools/gomad: `go test -tags test_dep ./internal/backends/wasm/installation ./internal/backends/wasm/cluster ./internal/backends/wasm/wasi` using the documented actual helper setup.

## Acceptance
- [ ] Final routing inventory separates supported production/reference/stock/native consumers; only proven leftovers removed
- [ ] Normal configuration cannot select a residual reference path; parity tests still can
- [ ] Before/after parity harness preserves per-incarnation evidence and shared order
- [ ] Standalone/stock/native source consumers, reference models and drift guards retained

## Done summary
Verified production Rust routing and private Go-reference selection; the caller audit found no additional obsolete adapter that justified deletion. Supported stock/standalone/native-source consumers and differential/drift coverage remain. Source review is SHIP at 18882a5f2a.

Evidence and exact candidate/commands are consolidated in `.flow/artifacts/fn180-current-batch-2026-10-16.md`, including final installed matrix `/tmp/fn181-classification-operator-20261016-7jBRX1/matrix.log`, portable closure checks and preserved lower-level qualification. The spec's final integrated qualification task remains open. No native qualification or fn-180 R6/fn-182 empirical completion is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 406e6bdf87, 3932262536, 86f25ef560, 18882a5f2a
- Tests: CARGO_TARGET_DIR=/tmp/gomad-wasm-current-target cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --locked, GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact GOMAD_WASM_CONFORMANCE_HELPER=/tmp/gomad-fn180-conformance-helper-compact GOFLAGS=-p=2 make -C tools/gomad test-wasm GOMAD_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go, GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact GOMAD_WASM_CONFORMANCE_HELPER=/tmp/gomad-fn180-conformance-helper-compact go test -tags test_dep ./internal/backends/wasm/wasi ./internal/backends/wasm/cluster -run '^TestStorageEvidenceLargeEnumerationOwnerParity$|^TestStorageEvidenceEscapedMetadataOwnerParity$|^TestStorageEvidenceStreamAdmissionKeepsIndependentBounds$|^TestProviderIndependentGuestsLifecycleReplay$|^TestRetainedLifecycleRejectsHostCallbacksBeforeReplay$' -count=1 -v, go test -tags test_dep ./internal/archcheck -count=1
- PRs:
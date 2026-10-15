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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

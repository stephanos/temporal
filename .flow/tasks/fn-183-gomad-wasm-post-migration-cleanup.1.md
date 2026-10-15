---
satisfies: [R1, R3]
---
# fn-183-gomad-wasm-post-migration-cleanup.1 Isolate reference execution and remove residual migration routes

## Description
Audit the delivered fn178 routing and remove demonstrated residual production reference selection/obsolete plumbing for R1/R3.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/cluster/provider.go, provider_test.go, wasi/cluster.go, wasi/process.go, wasi/integration_test.go, installation/installation.go, related cluster/installation routing tests
**Touches:** [tools/gomad/internal/backends/wasm/cluster/**, tools/gomad/internal/backends/wasm/wasi/cluster*.go, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/installation/**]

### Approach
- Re-anchor to fn178's completed routing. Inventory actual callers across cooperative clusters, standalone/stock execution, Go-broker differential tests and native/shared consumers. Do not reimplement fn178.9.
- Remove production selectors only if any remain; keep explicit test-only reference selection and existing stock/standalone ownership. Delete adapters/branches only after caller/profile analysis proves they are transitional and redundant. If a claimed leftover is already gone, record verification rather than invent replacement work.
- Pin behavior with Go-broker versus Rust-owner parity fixtures comparing replies, per-incarnation outcomes, evidence bytes and global order before/after cleanup. Retain conformance generation and reference-source drift checks.
- Test that normal installation/config cannot silently choose reference execution, and that reference tests still select it deliberately.
- Inventory final fn-178 OpenCluster/ClusterSession.Start, delayed/never-started nodes, restart holds/Finish, source-less retained replay/recovery and every shipped profile/isolation consumer. Keep standalone Go owners accessible where supported; changing a shipped request into a refusal is not a valid deletion proof.
- Keep test/reference-only and Go conformance consumers intentionally, plus final fn-176 full-context host/helper admission and chunked transport. Those are independent coverage/trust-boundary contracts, not dead production callbacks.
- .1/.2 parallel development requires disjoint actual source/test surfaces; serialize any common owner and validate only the joined candidate in .3.

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
- [ ] Each removed route has a final-candidate consumer/profile proof; no supported isolation, default behavior or source-less replay/recovery capability disappears
- [ ] Final fn-176 context admission and fn-178 delayed-start/holds/Finish/streamed evidence semantics remain intact

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

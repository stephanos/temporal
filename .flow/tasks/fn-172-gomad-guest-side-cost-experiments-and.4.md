---
satisfies: [R3]
---
# fn-172-gomad-guest-side-cost-experiments-and.4 Multi-instance cluster baseline

## Description
Record a re-runnable diagnostic baseline (phase, ABI and nested buckets) for the multi-instance cluster selection, for fn-178 R4 and R8. The network-v2 baseline was dropped (owner decision 2026-10-13): network-v2 runs always attach a broker, so the cluster baseline covers them.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/diagnostic_profile_test.go`, `tools/gomad/internal/backends/wasm/wasi/integration_test.go` (decode per-helper profile lines in the cluster pair runner), `.flow/artifacts/fn172-baselines-<date>.md` (new)
**Touches:** [tools/gomad/internal/backends/wasm/wasi/diagnostic_profile_test.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go, .flow/artifacts/fn172-*]

### Approach
- Cluster: drive `TestTemporalIndependentClustersPartitionHealWholeClusterCrash` (`:3594`, `temporalRunPair` at `:3709`). Decode every helper's `result.helperStderr` profile line with `decodeDiagnosticReport` and record per-instance and summed buckets plus parent pprof.
- Gate: run on the post-fn-154 integrated candidate (fn-154 tasks through .12 landed). The tracker cannot express this cross-spec task edge, so check it at start. Record the commit, helper and module digests, seeds, host and exact command.
- If fn-166.6 has renamed `internal/backends/wasm/cluster` or the gates moved, follow the current paths.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:3378-3450`, `:3594-3760`, `:1912` (`temporalRecoveryRequest`)
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:4653` (`runDiagnosticProfile`)

### Quick commands
```bash
(cd tools/gomad && GOMAD_WASM_NETWORK_TEMPORAL_ROOT=/abs/root GOMAD_WASM_TEST_HELPER=/abs/diag/gomad3-wasmhost env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 60m -run '^TestDiagnosticNetworkV2Baseline' ./internal/backends/wasm/wasi)
(cd tools/gomad && GOMAD_WASM_TEMPORAL_RECOVERY_ROOT=/abs/root GOMAD_WASM_TEST_HELPER=/abs/diag/gomad3-wasmhost env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 90m -run '^TestDiagnosticClusterBaseline' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] The artifact holds the cluster baseline (phases, ABI and nested buckets per instance and summed, plus the parent pprof summary), with commit, digests, seeds, host and re-run command.
- [ ] Baselines were captured on a candidate containing all landed fn-154 source; the commit is recorded.
- [ ] Driver tests skip without their env vars; the per-helper decoding has a unit test.

## Done summary
R3 cluster diagnostic baseline captured on Linux/arm64 (TestDiagnosticClusterBaseline, seed 7, measure 15b491281c with fn-154 .1-.12 receipts verified): 4/4 pairs complete with per-instance and summed phase/ABI/nested buckets and parent pprof summary. Artifact: .flow/artifacts/fn172-baselines-2026-10-14.md. Impl review SHIP r1/r2.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 15b491281c, 6e5dec62d3, 6be76036ed, 13a997350f, b7d1393c37, 9e7270cf91
- Tests: TestDiagnosticClusterBaseline on Linux/arm64 (C3): PASS, TestDiagnosticClusterPair*: PASS
- PRs:
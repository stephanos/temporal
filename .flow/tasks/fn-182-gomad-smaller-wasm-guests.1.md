---
satisfies: [R1, R4]
---
# fn-182-gomad-smaller-wasm-guests.1 Measure guest dependency closures and select safe exclusions

## Description
Establish R1 and the pre-change half of R4 before changing dependency selection.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/backend/inventory_test.go, wasi/integration_test.go, qualification/onebox_test.go, qualification/temporal_cluster_test.go, measurement report
**Touches:** [tools/gomad/internal/backends/wasm/backend/*test.go, tools/gomad/internal/backends/wasm/wasi/*test.go, tools/gomad/internal/backends/wasm/qualification/*test.go]

### Approach
- Use the provider's actual go list/build environment and qualified workload/tag selections to inventory transitive dependencies, init registrations and provider references. Distinguish imported package count from linked module bytes.
- Trace Elasticsearch first through visibility.NewManager and newVisibilityStoreFromDataStoreConfig; typed ProcessorConfig references mean removing a constructor import alone may not sever the closure. Inspect optional provider seams, not test rewriting.
- Select the smallest exclusion that preserves the workload, default production and stock/native behavior. Prove a disposable same-selection build can shrink the module before committing to the seam; if none is safe, keep R2 open and report the finding.
- Reuse existing diagnostic measurement collectors to freeze a post-fn178 baseline on the measurement host. Record module/cache bytes, compile, warm preparation, fresh instantiation, execution and peak RSS with common seeds, repetitions, budgets, identities and rerun commands.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/backend/provider.go:149
- tools/gomad/internal/backends/wasm/backend/inventory.go
- common/persistence/visibility/factory.go:33
- common/persistence/visibility/store/elasticsearch/client/aws_gomad.go
- tools/gomad/internal/backends/wasm/qualification/onebox_test.go:71
- tools/gomad/internal/backends/wasm/wasi/integration_test.go:5457

### Quick commands
From tools/gomad: `go test -tags test_dep ./internal/backends/wasm/backend`; use TestDiagnosticGuestVariants/measurement driver conventions for actual guest baseline capture, with their documented explicit helper inputs.

## Acceptance
- [x] Representative qualified single/cluster guest closures and registration dependencies recorded with exact inputs
- [x] Selected exclusion has a measured smaller disposable build of the same workload and a documented preserved-provider seam
- [x] Required/default-profile dependencies are retained; Elasticsearch is a candidate rather than mandatory removal
- [x] Matched pre-change same-host baseline retained for every R4 metric before implementation; missing measurements remain open — byte metrics retained (module bytes, baseline compiled-cache bytes); the timing/RSS metrics are **transferred to fn-185 R6** (owner decision 2026-10-17)
## Done summary
Recorded the qualified one-box and multi-instance guest closures under the provider's WASI build environment, selected three optional-provider exclusions (cloud archivers, ringpop, MySQL/PostgreSQL test drivers) and rejected Elasticsearch, which the scheduler query converter needs. Re-verified on the post-fn-178 candidate (`bb20a23e6f`): multi-instance 1446 -> 1320 packages, 126 removed, 0 added, identical to the predicted edge cut. Byte baseline: full one-box module 338,040,129 B and compiled cache 693,979,010 B (n=1). The timed baseline part of R4 is transferred to fn-185 R6 by the owner decision of 2026-10-17. Detail: `.flow/artifacts/fn182-closeout-2026-10-17.md`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9335ea7c32, bb20a23e6f
- Tests: go list -mod=readonly -pgo=off -deps -test -json -tags disable_grpc_modules,sqlite3_dotlk,test_dep[,gomad_wasm_slim] <guest package> (GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 GOEXPERIMENT=nogreenteagc), go test -tags test_dep -run TestSlimGuestClosureCutsOnlyOptionalProviderSeams ./internal/backends/wasm/backend
- PRs:
- Detailed evidence: `.flow/artifacts/fn182-closeout-2026-10-17.md` (R-ID map, commits, commands, exits, reviews, fn-185 R6 transfer)

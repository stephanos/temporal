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
- Reuse existing diagnostic measurement collectors to freeze a post-fn178 baseline on real Linux/arm64. Record module/cache bytes, compile, warm preparation, fresh instantiation, execution and peak RSS with common seeds, repetitions, budgets, identities and rerun commands.
- Freeze the final integrated untrimmed candidate and every R4 production metric before any provider exclusion; preserve actual qualified workload/test/argument/tag/config inputs. Reuse collectors without adopting diagnostic/fuel-off/GC variants as acceptance baselines; matched helpers/runtime/engine/fuel/clock/resident selections stay fixed.
- Trace public ProcessorConfig/type, constructor, registration and test edges before choosing exact files. Record the minimal guest-only opt-in tag and complementary stock/native/default source set, explicit primary/secondary provider refusal and a disposable same-selection byte-reduction proof; Elasticsearch remains one candidate.
- Serialize heavy production measurements, separate cache roots and record explicit cold/warm states/repetition order. A size proof or historical report cannot substitute for a missing baseline phase; no safe exclusion leaves R2 open.

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
- [ ] Representative qualified single/cluster guest closures and registration dependencies recorded with exact inputs
- [ ] Selected exclusion has a measured smaller disposable build of the same workload and a documented preserved-provider seam
- [ ] Required/default-profile dependencies are retained; Elasticsearch is a candidate rather than mandatory removal
- [ ] Matched pre-change Linux/arm64 baseline retained for every R4 metric before implementation; missing measurements remain open
- [ ] Production helper/fuel/runtime selections and cold/warm cache states match final fn-178 workload inputs; no diagnostic/fuel-off substitute or smaller test workload
- [ ] Exact provider/type/registration seam and opt-in guest tag justified without default API redesign or mandatory Elasticsearch removal

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

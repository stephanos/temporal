---
satisfies: [R1, R2]
---
# fn-223-gomad-delete-the-native-backend.2 Remove the linkname glue, Make targets, CI jobs, archcheck tables and native build tags

## Description
Delete the native go:linkname glue in internal/model/cluster/runtime_domain.go, runtime_network.go, runtime_process.go, runtime_process_model.go and runtime_volume.go, and remove the native-only runtime_time_toolchain.go selection, preserving stock unavailable-runtime behavior. Update the cluster embedded-source inventory for removed files without weakening coverage. Remove the native targets from tools/gomad/Makefile and the root Makefile (toolchain, test-host, test-toolchain, overlay-test, test-simulation, core-qualification, upgrade-dossier, compatibility-pack-qualification, test-builder, test-live-capability, test-runtime, test-upstream, tests-qualification-generate, generate-native, validate-native, soak, GOMAD_LINT_TAGS selection), the native jobs and native path-filter output from .github/workflows/gomad.yml, and all native Linux/Darwin smoke jobs and routing from gomad-smoke.yml. Delete the smoke workflow if no retained jobs remain, otherwise keep valid workflow structure; validate source without dispatching CI. Remove native ownership tables, inventories, exclusions, NativeSelected and nativeModulePresent from internal/archcheck. Clean native path/build-tag references from every retained Git-tracked document under tools/gomad before the zero-reference gate, including README.md, ARCHITECTURE.md, CLI.md and internal/backends/wasm/README.md; preserve remaining WASM operator instructions when retiring obsolete native guide content. Task .3 finishes historical wording and qualification links. List every //go:build gomad site and decide each; preserve live WASM inspection/determinism instrumentation as well as gomad_wasm_slim and its guest stubs.

**Size:** M
**Files:** tools/gomad/internal/model/cluster/runtime_*.go, tools/gomad/Makefile, Makefile, .github/workflows/gomad.yml, .github/workflows/gomad-smoke.yml, tools/gomad/internal/archcheck/**, tools/gomad/**/*.md, //go:build gomad sites
**Touches:** Cluster runtime glue and embedded inventory; native Make/CI/archcheck wiring; native-only selections; reference-bearing Gomad documentation.

## Acceptance
- [ ] Git-tracked reference scan over tools/gomad, both Makefiles and .github/workflows reports no internal/backends/native or gomad_native match (git grep exit 1); scan errors fail
- [ ] go test -tags test_dep ./internal/archcheck ./internal/model/cluster/... passes
- [ ] Every //go:build gomad site listed with its decision; gomad_wasm_slim untouched
- [ ] Live gomad-tagged WASM instrumentation retained; changed cluster identities and their transitive bindings accounted for under task .3
- [ ] All native smoke jobs/routing retired, every retained workflow structurally valid; source validation only, no CI run

## Done summary
R1/R2: Removed native glue, targets, CI/smoke jobs/routing, archcheck inventories and native-only selections; retained WASM instrumentation/slim seams and source-hash coverage. Exact tracked native tree/tag reference scan exits1 with no matches, not an error. Build-tag inventory classifies every site. Architecture/model/router/workflow source checks exit0; no CI dispatched. Review found lint-only routing regression, fixed91abbc3051; both real-switch regressions pass on root. Independent combined source review SHIP at07ebfdb4ee; final runtime qualification remains pending under fn232.

Evidence index: /home/agent/wt/evidence/fn232/fn223-deletion-inputs-2026-10-20/README.md; original producer handovers and narrow review-fix appends remain authenticated independently. Source acceptance only; spec closure requires separate completion review.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cafabfb200, 46ce5a58bb, 91abbc3051
- Tests: go test -tags test_dep ./internal/archcheck ./internal/model/cluster/... (retained frozen source receipts), workflow YAML/needs validation; exact tracked native reference scan; build-tag site inventory, root pinned Go TestCIBackendGatesRoutePaths exit0
- PRs:
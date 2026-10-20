---
satisfies: [R1, R2]
---
# fn-223-gomad-delete-the-native-backend.2 Remove the linkname glue, Make targets, CI jobs, archcheck tables and native build tags

## Description
Delete the native go:linkname glue in internal/model/cluster/runtime_domain.go, runtime_network.go, runtime_process.go, runtime_process_model.go and runtime_volume.go, and remove the native-only runtime_time_toolchain.go selection, preserving stock unavailable-runtime behavior. Update the cluster embedded-source inventory for removed files without weakening coverage. Remove the native targets from tools/gomad/Makefile and the root Makefile (toolchain, test-host, test-toolchain, overlay-test, test-simulation, core-qualification, upgrade-dossier, compatibility-pack-qualification, test-builder, test-live-capability, test-runtime, test-upstream, tests-qualification-generate, generate-native, validate-native, soak, GOMAD_LINT_TAGS selection), the native jobs and native path-filter output from .github/workflows/gomad.yml and the native manifest reference from gomad-smoke.yml (edited as source only). Remove native ownership tables, inventories, exclusions, NativeSelected and nativeModulePresent from internal/archcheck. List every //go:build gomad site and decide each; preserve live WASM inspection/determinism instrumentation as well as gomad_wasm_slim and its guest stubs.

**Size:** M
**Files:** tools/gomad/internal/model/cluster/runtime_*.go, tools/gomad/Makefile, Makefile, .github/workflows/gomad.yml, .github/workflows/gomad-smoke.yml, tools/gomad/internal/archcheck/**, //go:build gomad sites

## Acceptance
- [ ] grep for internal/backends/native and gomad_native over tools/gomad, both Makefiles, .github/workflows and internal/archcheck returns nothing
- [ ] go test -tags test_dep ./internal/archcheck ./internal/model/cluster/... passes
- [ ] Every //go:build gomad site listed with its decision; gomad_wasm_slim untouched
- [ ] Live gomad-tagged WASM instrumentation retained; changed cluster identities and their transitive bindings accounted for under task .3
- [ ] Workflows edited as source only; no CI run

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

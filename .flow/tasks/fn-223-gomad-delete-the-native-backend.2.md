---
satisfies: [R1, R2]
---
# fn-223-gomad-delete-the-native-backend.2 Remove the linkname glue, Make targets, CI jobs, archcheck tables and native build tags

## Description
Delete the go:linkname declarations against internal/gomadio and internal/gomadsim in internal/model/cluster/runtime_network.go, runtime_process.go and runtime_volume.go, leaving the ErrRuntimeUnavailable implementations. Remove the native targets from tools/gomad/Makefile and the root Makefile (toolchain, test-host, test-toolchain, overlay-test, test-simulation, core-qualification, upgrade-dossier, compatibility-pack-qualification, test-builder, test-live-capability, test-runtime, test-upstream, tests-qualification-generate, generate-native, validate-native, soak, GOMAD_LINT_TAGS selection), the native jobs and native path-filter output from .github/workflows/gomad.yml and the native manifest reference from gomad-smoke.yml (edited as source only). Remove native ownership tables, inventories, exclusions, NativeSelected and nativeModulePresent from internal/archcheck. List every //go:build gomad site and decide each; gomad_wasm_slim and its guest stubs stay.

**Size:** M
**Files:** tools/gomad/internal/model/cluster/runtime_*.go, tools/gomad/Makefile, Makefile, .github/workflows/gomad.yml, .github/workflows/gomad-smoke.yml, tools/gomad/internal/archcheck/**, //go:build gomad sites

## Acceptance
- [ ] grep for internal/backends/native and gomad_native over tools/gomad, both Makefiles, .github/workflows and internal/archcheck returns nothing
- [ ] go test -tags test_dep ./internal/archcheck ./internal/model/cluster/... passes
- [ ] Every //go:build gomad site listed with its decision; gomad_wasm_slim untouched
- [ ] Workflows edited as source only; no CI run

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

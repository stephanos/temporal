---
satisfies: [R1, R3, R4]
---
# fn-184-gomad-native-backend-isolation.3 Apply the isolation to gates, archcheck and CI sources

## Description
Apply the chosen mechanism to the native tree, then rewire the gates: Makefile default targets (test-wasm, lint-code, validate, test-harness, world-test) operate on the default set and native targets (toolchain, test-host, test-toolchain, overlay-test, test-simulation, core-qualification, upgrade-dossier, compatibility-pack-qualification, test-builder, test-live-capability, test-runtime, test-upstream, tests-qualification-generate) pass the selection; the root lint-code-gomad target and `cmd/tools/lintcode` cover the default set and, with the selection, the native set. Update internal/archcheck ownership tables, inventories and exclusions so the default run inspects the default set without reporting native packages as uncovered and the selected run keeps the both-source-set checks; keep the WASM layout and architecture tests. Keep the projection generators, model/cluster linkname glue, version-pin validation and pin-impact working on both sets. Edit .github/workflows/gomad.yml and gomad-smoke.yml as source so wasm-linux and host-tools-linux use the default set and native jobs use the selection; do not run CI. Run the native gates under the selection to prove nothing was lost, and the WASM gates on the default set.

**Size:** L
**Files:** tools/gomad/internal/backends/native/** (mechanism only), tools/gomad/Makefile, Makefile (lint-code-gomad), cmd/tools/lintcode/**, tools/gomad/internal/archcheck/**, tools/gomad/internal/backends/wasm/{layout_test.go,architecture_test.go}, tools/gomad/internal/model/volume/internal/projection/generate.go, tools/gomad/internal/model/network/generate/main.go, .github/workflows/gomad.yml, .github/workflows/gomad-smoke.yml

## Acceptance
- [x] Default set: build, vet, lint-code, test-wasm, test-harness, world-test and archcheck pass with zero native packages compiled
- [x] Selected set: every existing native Make target and generator runs unchanged; nothing deleted or relocated beyond the mechanism — native execution stays deferred under fn-128/fn-149
- [x] Projection generators, linkname glue, version-pin and pin-impact validate on both sets; generated outputs unchanged or regenerated with the rotation recorded
- [x] Workflow sources updated; no CI run, PR or push
## Done summary
Applied the nested module to Make gates, archcheck, lint and CI sources (`d40debd4db`, `424cc83bbf`, `6898eceb5a`, `bc569fb2a3`). Default set: build, vet, archcheck, lint, validate, test-harness, world-test and test-wasm exit 0 with zero native packages. Selected set: native build/vet/test compile, archcheck, lint and validate exit 0; remaining selected CLI/gomadtool failures equal the baseline and need the deferred native toolchain. Identities unchanged; four inherited pin-impact invalidations unchanged. Workflows edited as source only. Detail: `.flow/artifacts/fn184-closeout-2026-10-17.md`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d40debd4db, ac48f5cc9f, 424cc83bbf, 6898eceb5a, bc569fb2a3
- Tests: cd tools/gomad && go test -count=1 -tags test_dep ./internal/archcheck, cd tools/gomad && go test -count=1 -tags test_dep,gomad_native ./internal/archcheck, make -C tools/gomad validate (default and GOFLAGS=-tags=gomad_native), make lint-code-gomad GOLANGCI_LINT_BASE_REV=3dae95b99f GOLANGCI_LINT_FIX=false (default and GOFLAGS=-tags=gomad_native), make -C tools/gomad test-harness world-test
- PRs:
- Detailed evidence: `.flow/artifacts/fn184-closeout-2026-10-17.md` (R-ID map, commits, commands, exits, reviews, fn-185 R6 transfer)

---
satisfies: [R1, R4, R6]
---
# fn-219-gomad-promote-the-model-projections-and.3 Rewire Makefile, archcheck and CI to the default set

## Description
tools/gomad/Makefile: generate and validate-toolchain run protocolgen without tags; move version-generate, boundary-generate, compatibility-pack, patch-validate, script-validate, qualification-manifest-generate and TestHostPacksBindCurrentProfile plus OVERLAY_FILES, VERSION_INPUTS, BOUNDARY_INPUTS and COMPATIBILITY_INPUTS to generate-native and validate-native which set the selection themselves; native targets that depended on validate depend on validate-native. internal/archcheck: native tables and inventories only under NativeSelected, remove the projection generator entries, add a default-set test that no untagged file outside internal/backends/native contains the string internal/backends/native (tagged fixtures allowed). .github/workflows/gomad.yml: drop the overlay/schema/generator to wasm=true mapping, run make validate without GOFLAGS=-tags=gomad_native, run validate-native only when native is true. Edit workflows as source only.

**Size:** M
**Files:** tools/gomad/Makefile, Makefile (lint-code-gomad if affected), tools/gomad/internal/archcheck/**, .github/workflows/gomad.yml, .github/workflows/gomad-smoke.yml (only if it calls validate)

## Acceptance
- [ ] make -C tools/gomad generate validate test-wasm run no command with -tags gomad_native
- [ ] GOFLAGS=-tags=gomad_native make -C tools/gomad generate-native validate-native exit 0 and every native target that depended on validate still runs
- [ ] go test -tags test_dep ./internal/archcheck passes on the default set with the new string check, and fails when a native path string is introduced in an untagged shared file
- [ ] gomad.yml filter and host-tools-linux steps updated as source; no CI run, PR or push

## Done summary
Moved native Make generators/validators, ownership tables and inventories behind explicit selection; default generation/validation and CI routing no longer read the native tree. Added rejecting default-source path checks and default Make dry-run checks; retained native-absent generate/validate/build/vet/model/architecture checks. Fixed the lint router to read split native tables and classified split native module-contract tests consistently with their original excluded counterparts, preserving checker code/test limits and oversized negative tests. Fresh integrated lint-router tests and full default archcheck exit 0 (integrated-archcheck-fixed.log, 70.541s); default/native generation validation exit 0. Independent source review SHIP, including the budget classification correction. Native selected archcheck/gomadtool baseline failures remain recorded; no runtime, CI, PR or push claim. Evidence: .flow/artifacts/fn219-acceptance-audit-2026-10-20.md and /home/agent/wt/evidence/fn219/.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aa43211daf, ab052d04e9, b6a8296c1f, 5f9e92af84, 8aca21a9df, cbfd5f75a3, a84a807829, 38b4c5c6f2
- Tests: go test -tags test_dep -count=1 ./internal/model/volume/... ./internal/model/network/... ./internal/model/generation/protocol/... ./internal/model/cmd/protocolgen/... ./internal/archcheck (exit 0), make generate validate (exit 0), GOFLAGS='-tags=gomad_native -p=2' make generate-native validate-native (exit 0), go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool protocol-generate -check (exit 0), required-helper TestNamespaceConformance and TestVolumeConformance (exit 0), cargo +1.99.1 test --release --locked --test network_corpus --test world_conformance --test clock_conformance (exit 0), integrated go test -p=1 -tags test_dep ./cmd/tools/lintcode (exit 0), integrated go test -p=1 -tags test_dep ./internal/archcheck (exit 0; integrated-archcheck-fixed.log), independent source review and budget correction review: SHIP
- PRs:
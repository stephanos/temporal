---
satisfies: [R1, R5]
---
# fn-184-gomad-native-backend-isolation.1 Decide and prototype the isolation mechanism

## Description
Prototype both candidates on a throwaway branch: a nested Go module at internal/backends/native (module path go.temporal.io/server/tools/gomad/internal/backends/native, parent replace, same pinned versions) and a file-level gomad_native build constraint over all native sources. For each, show `go list -deps ./cmd/gomad` with the CLI native imports stubbed contains no internal/backends/native package, `go build ./... && go vet ./...` of the default set passes, and the native set builds under the selection. Check whether either rotates a retained choice, model, helper or runtime identity (compare the generated identity outputs and version-pin validation). Record the baseline measurement for R5 before any change: default-set package count, production/test lines, and wall time for build, vet, lint-code, test-harness, world-test and test-wasm on a quiet host. Write the decision with the trade-offs (file count touched, go.sum duplication, pin-impact handling, archcheck inventory impact) into the spec's Decision Context.

**Size:** M
**Files:** tools/gomad/go.mod, tools/gomad/internal/backends/native/** (prototype only), tools/gomad/cmd/gomad/internal/cli/launcher.go, .flow/specs/fn-184-gomad-native-backend-isolation.md

## Acceptance
- [x] Both mechanisms prototyped; `go list -deps ./cmd/gomad` on the default set lists no native package for the chosen one
- [x] Native set builds and `go vet` passes under the selection
- [x] Identity check recorded: no rotation, or the exact identities that rotate and why
- [x] Baseline numbers for R5 retained (package count, lines, gate wall times, host identity) — package and line counts and host identity retained; gate wall times **transferred to fn-185 R6** (owner decision 2026-10-17)
- [x] Decision and trade-offs written into the spec (Decision Context)
## Done summary
Prototyped both mechanisms on throwaway branches: nested module (`1d72581886`) gives zero native packages with default build/vet exit 0; pure build tags (`ffac982ecb`) fail because overlay packages stay visible to stock Go. Chose the nested module. Identity probes show no rotation (choice, volume, network, World, WASM model). Baseline counts: 80 packages, 118,144 production / 131,789 test lines at `3dae95b99f`. Gate wall times are transferred to fn-185 R6 by the owner decision of 2026-10-17. Detail: `.flow/artifacts/fn184-closeout-2026-10-17.md`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d40debd4db
- Tests: cd tools/gomad && go list -deps ./cmd/gomad | grep internal/backends/native, cd tools/gomad && go build ./... && go vet ./..., BASE_SHA_EXPECTED=3dae95b99f AFTER_SHA_EXPECTED=f22f8ef227 bash /home/agent/wt/evidence/fn184/measure.sh structural
- PRs:
- Detailed evidence: `.flow/artifacts/fn184-closeout-2026-10-17.md` (R-ID map, commits, commands, exits, reviews, fn-185 R6 transfer)

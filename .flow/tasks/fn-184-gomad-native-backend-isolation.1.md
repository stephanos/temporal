---
satisfies: [R1, R5]
---
# fn-184-gomad-native-backend-isolation.1 Decide and prototype the isolation mechanism

## Description
Prototype both candidates on a throwaway branch: a nested Go module at internal/backends/native (module path go.temporal.io/server/tools/gomad/internal/backends/native, parent replace, same pinned versions) and a file-level gomad_native build constraint over all native sources. For each, show `go list -deps ./cmd/gomad` with the CLI native imports stubbed contains no internal/backends/native package, `go build ./... && go vet ./...` of the default set passes, and the native set builds under the selection. Check whether either rotates a retained choice, model, helper or runtime identity (compare the generated identity outputs and version-pin validation). Record the baseline measurement for R5 before any change: default-set package count, production/test lines, and wall time for build, vet, lint-code, test-harness, world-test and test-wasm on a quiet host. Write the decision with the trade-offs (file count touched, go.sum duplication, pin-impact handling, archcheck inventory impact) into the spec's Decision Context.

**Size:** M
**Files:** tools/gomad/go.mod, tools/gomad/internal/backends/native/** (prototype only), tools/gomad/cmd/gomad/internal/cli/launcher.go, .flow/specs/fn-184-gomad-native-backend-isolation.md

## Acceptance
- [ ] Both mechanisms prototyped; `go list -deps ./cmd/gomad` on the default set lists no native package for the chosen one
- [ ] Native set builds and `go vet` passes under the selection
- [ ] Identity check recorded: no rotation, or the exact identities that rotate and why
- [ ] Baseline numbers for R5 retained (package count, lines, gate wall times, host identity)
- [ ] Decision and trade-offs written into the spec

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

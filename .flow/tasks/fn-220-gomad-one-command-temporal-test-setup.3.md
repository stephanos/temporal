---
satisfies: [R4]
---
# fn-220-gomad-one-command-temporal-test-setup.3 Document the one-command workflow

## Description
Update internal/backends/wasm/CLI.md and tools/gomad/README.md (the paragraph at README lines 150-172) to describe the one-command workflow and remove the statement that arbitrary Temporal tests lack a setup command. State that support is limited to the named test and that qualification is a separate spec.

**Size:** S
**Files:** tools/gomad/README.md, tools/gomad/internal/backends/wasm/CLI.md

## Acceptance
- [ ] CLI.md and README describe the install and explore commands for a Temporal test package
- [ ] No claim of support beyond TestFrontendSystemInfo; qualification named as separate

## Done summary
README and WASM CLI guide now document install --temporal-checkout/--temporal-package, explicit explore environment and retention flags, inspection/replay, bounded capacities and fixture profile differences. Removed the claim that setup requires hand-authored configuration. Support wording stays confined to the named TestFrontendSystemInfo observation; qualification belongs to fn-222. Actual documented install/explore/replay workflow executed successfully on 29f49b17bc. Independent review SHIP, changed-code lint and diff checks pass. Evidence: .flow/artifacts/fn220-setup-proof-2026-10-20.md.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fb8044eca5, 7c8ef42504, ff719ea3b4, 8896829710, fc04600872, 93b12381b7
- Tests: go test -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/installation ./cmd/gomad/internal/cli -count=1 (exit 0), TestTemporalConfigPreservesPrototypeBytes; TestTemporalInspectionConfigPreservesFixtureBytes; historical qualification smoke bytes (exit 0), compiler-backed TestBuildSelectionChangesInvalidateCompilationCache (exit 0), network connection-limit mutation 4096 to 4095: command+inspection tests exit 1; restored exit 0, derived install and discover on clean 29f49b17bc (exit 0), explore TestFrontendSystemInfo retained success on clean 29f49b17bc (exit 0), fresh gomad replay retained success: reproduced=true choice-replay=exact (exit 0), make lint-code-fast GOLANGCI_LINT_FIX=false (exit 0), independent source review after 93b12381b7: SHIP
- PRs:
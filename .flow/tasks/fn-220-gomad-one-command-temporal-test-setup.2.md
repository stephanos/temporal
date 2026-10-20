---
satisfies: [R1, R2]
---
# fn-220-gomad-one-command-temporal-test-setup.2 Expose the one-command setup and run TestFrontendSystemInfo end to end

## Description
Wire the chosen consumer path into internal/backends/wasm/command and cmd/gomad/internal/cli with explicit inputs only (checkout, package, compiler, helper, runtime root, output), refusing inherited GOMAD_* variables. Missing inputs are refused with the existing installation status and a message naming the input. From a clean checkout: one install-side command, then gomad explore go-test ./tests/gomadfunctional -- -test.run=^TestFrontendSystemInfo$ to a pass with retained evidence, then gomad replay of that result. Record the exact commands and exits.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/command/**, tools/gomad/cmd/gomad/internal/cli/**, CLI fixtures under tools/gomad/cmd/gomad/testdata/**

## Acceptance
- [ ] One install-side command produces the cooperative installation with schema mount, working directory, PWD, writable roots, limits and slim tags; no GOMAD_TEMPORAL_CONFIG
- [ ] TestFrontendSystemInfo passes under gomad explore and the retained result replays
- [ ] Missing compiler, helper or schema source refused with the existing status and a named input
- [ ] Existing WASM CLI fixtures pass byte-for-byte

## Done summary
One explicit install command captures immutable schema, guest cwd/PWD, writable roots, capacities and slim tags without manual JSON or GOMAD_TEMPORAL_CONFIG. Missing compiler/helper/schema, package escapes and inherited GOMAD_* settings fail with existing named-input status; existing command/installation/shared CLI suites pass. Clean integrated candidate 29f49b17bc ran TestFrontendSystemInfo through explore: attempted=1, succeeded=1, zero failures/watchdogs/divergences, one retained success. Fresh replay exited 0 with reproduced=true, result=success and choice-replay=exact. The actual named guest PASS, commands, numeric exits, complete tape/trace/module/helper/compiler/model/runtime digests and local artifact path are retained in .flow/artifacts/fn220-setup-proof-2026-10-20.md. Independent source review SHIP after the independent configuration-expectation correction. Full WASM requalification is separately owned by fn-219.4/fn-222; this is not a broader support claim.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fb8044eca5, 7c8ef42504, ff719ea3b4, 8896829710, fc04600872, 93b12381b7
- Tests: go test -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/installation ./cmd/gomad/internal/cli -count=1 (exit 0), TestTemporalConfigPreservesPrototypeBytes; TestTemporalInspectionConfigPreservesFixtureBytes; historical qualification smoke bytes (exit 0), compiler-backed TestBuildSelectionChangesInvalidateCompilationCache (exit 0), network connection-limit mutation 4096 to 4095: command+inspection tests exit 1; restored exit 0, derived install and discover on clean 29f49b17bc (exit 0), explore TestFrontendSystemInfo retained success on clean 29f49b17bc (exit 0), fresh gomad replay retained success: reproduced=true choice-replay=exact (exit 0), make lint-code-fast GOLANGCI_LINT_FIX=false (exit 0), independent source review after 93b12381b7: SHIP
- PRs:
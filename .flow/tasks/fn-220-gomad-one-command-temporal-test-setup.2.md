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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R2, R6]
---
# fn-166-gomad-readable-package-structure-and.3 Consolidate WASM, CLI and gomadtool tests by behavior

## Description
Merge backends/wasm/wasi helper_* tests per production file and move guest-compiling tests into integration_test.go; consolidate backends/wasm/backend, cmd/gomad, cmd/gomad/internal/cli and cmd/gomadtool tests into per-file and per-command files.

## Acceptance
wasi has at most 14, cli at most 15 and gomadtool at most 12 test files; test-wasm and its -run selections still select the same tests; no coverage dropped.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

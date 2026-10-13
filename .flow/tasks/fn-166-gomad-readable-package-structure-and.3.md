---
satisfies: [R2, R6]
---
# fn-166-gomad-readable-package-structure-and.3 Consolidate WASM, CLI and gomadtool tests by behavior

## Description
Merge backends/wasm/wasi helper_* tests per production file and move guest-compiling tests into integration_test.go; consolidate backends/wasm/backend, cmd/gomad, cmd/gomad/internal/cli and cmd/gomadtool tests into per-file and per-command files.

## Acceptance
wasi has at most 14, cli at most 15 and gomadtool at most 12 test files; test-wasm and its -run selections still select the same tests; no coverage dropped.

## Done summary
WASM, CLI and gomadtool tests organized by behavior: wasi 28 to 11, cli 24 to 12, gomadtool 20 to 9, wasm backend 17 to 7, cmd/gomad e2e 4 to 1. Test-name lists and pass/skip/fail outcomes identical; only test files changed; no hashed source changed (WASI pin 9ef9041f unchanged); Make/CI selectors unchanged; test-wasm, validate, architecture and lint pass.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3b635b38c2, 23bc923c68, 09e74dcefd, 8842f360c1, 0159a7e012
- Tests: make -C tools/gomad3 test-wasm, go test -list per package (identical), architecture selection, lint-code-fast, make -C tools/gomad3 validate, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs:
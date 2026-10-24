---
satisfies: [R11, R12]
---

# fn-232-gomad-rewrite-the-host-side-in-rust.19 Restore retained Go lint and router gates
# Restore retained Go lint and router gates

## Description
Size: S

**Touches:** `cmd/tools/lintcode/lint_policy_test.go`, `tools/gomad/internal/backends/wasm/toolchain/cmd/runtimeinventory/main.go`, `tools/gomad/internal/model/cmd/sqlite-native/main.go`, `tools/gomad/internal/model/volume/recovery_image.go`

Cycle2 on abbea52d43 passed validate but failed lint-router, lint-router-vet and lint-module with actual source diagnostics. Router listing/vet cannot compile lint_policy_test.go (unused strings, missing go/format for format.Source); retirement commit b64f3dcb89 changed its imports. Module lint reports S1017 on conditional TrimPrefix, errcheck on read-only stream.Close, QF1001 on digest character condition. Retain failed command/receipts unchanged in resume-closure-cycle2; these supply no cycle success.

Repair only these demonstrated findings using existing conventions. Preserve all policy assertions/exclusions, runtimeinventory output, SQLite input/file/capacity checks and digest accepted language. Do not change linter configuration, suppressions, base revision, assertions, budgets or unrelated formatting. No dependency changes. Inspect surrounding tests and patterns first. Existing failed compile/lint commands are the regression; add no tests that mirror mechanical syntax changes. Run corrected router tests/vet plus retained Go package tests and exact module lint command. Preserve red and green logs. Root owns full mandatory gate, independent review, source-binding preparation and actual fresh cycle. New source-byte changes require real source-bound input refresh where needed; no rewriting historical receipts.

## Acceptance
- [ ] Router package compiles and its existing policy/router tests pass with pinned Go; go vet -tags test_dep ./cmd/tools/lintcode exits 0.
- [ ] Exact lint-code-gomad gate with retained tool paths and unchanged base revision exits 0; no source finding is suppressed.
- [ ] Runtimeinventory mapping, SQLite input handling and digest validation retain behavior, supported by existing relevant package tests and focused review.
- [ ] Retained cycle2 failure evidence remains unchanged; fresh full-cycle and all receiving obligations stay with .13.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

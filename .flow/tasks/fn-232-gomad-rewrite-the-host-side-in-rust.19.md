---
satisfies: [R11, R12]
---

# fn-232-gomad-rewrite-the-host-side-in-rust.19 Restore retained Go lint and router gates
# Restore retained Go lint and router gates

## Description
Size: S

**Touches:** `cmd/tools/lintcode/main.go`, `cmd/tools/lintcode/main_test.go`, `tools/gomad/rust/driver/static_stages.py`, `tools/gomad/rust/driver/tests/test_static_stages.py`, `cmd/tools/lintcode/lint_policy_test.go`, `tools/gomad/internal/backends/wasm/toolchain/cmd/runtimeinventory/main.go`, `tools/gomad/internal/model/cmd/sqlite-native/main.go`, `tools/gomad/internal/model/volume/recovery_image.go`

Cycle2 on abbea52d43 passed validate but failed lint-router, lint-router-vet and lint-module with actual source diagnostics. Router listing/vet cannot compile lint_policy_test.go (unused strings, missing go/format for format.Source); retirement commit b64f3dcb89 changed its imports. Module lint reports S1017 on conditional TrimPrefix, errcheck on read-only stream.Close, QF1001 on digest character condition. Retain failed command/receipts unchanged in resume-closure-cycle2; these supply no cycle success.

Repair only these demonstrated findings using existing conventions. Preserve all active policy assertions/exclusions, runtimeinventory output, SQLite input/file/capacity checks and digest accepted language. Do not change linter configuration, suppressions, base revision, budgets or unrelated formatting. No dependency changes. Inspect surrounding tests and patterns first. Existing failed compile/lint commands are the regression; add no tests that mirror mechanical syntax changes. Run corrected router tests/vet plus retained Go package tests and exact module lint command. Preserve red and green logs. Root owns full mandatory gate, independent review, source-binding preparation and actual fresh cycle. New source-byte changes require real source-bound input refresh where needed; no rewriting historical receipts.

Further defects revealed by the initial repair are in scope. Retirement b64f3dcb89 removed exactly two obsolete Go-runner panic exceptions; the policy count must reflect the 17 active rules while preserving their path/negative assertions. Router package selection is lexically sorted; update obsolete wasi/cluster expectation order. CI still enforces the retained Go lint step in renamed reference-tools-linux (the old host-tools-linux job no longer exists); test the actual job without removing the requirement. The relocated internal/model/wasi fixture tree currently falls through broad archcheck exclusions, accepting unknown runtime_cost_extra. Restore its original closed fixture boundary: known runtime_cost/namespace_owner Go fixture roots receive their declared disposition, unrecognized hidden/testdata sources remain refused. Preserve the existing negative test and add relocated-root positive/negative boundaries where needed.

The cycle's lint-router action currently omits LINT_POLICY_GOLANGCI, so TestLintPolicyRealGolangci would skip and strict verify_go would correctly refuse. Wire the already sealed plan.tools.golangci into that action's producer and independent reader command environment; retain actual policy fixture execution and strict skips, never add an exclusion. Add a red/green regression for this missing action environment and run the actual policy suite using the exact pinned linter. This scope extension follows observed regression evidence and source/call tracing; it is not a blanket permission to change assertions to observed output.

## Acceptance
- [ ] Router package compiles and its existing policy/router tests pass with pinned Go; go vet -tags test_dep ./cmd/tools/lintcode exits 0.
- [ ] Exact lint-code-gomad gate with retained tool paths and unchanged base revision exits 0; no source finding is suppressed.
- [ ] Runtimeinventory mapping, SQLite input handling and digest validation retain behavior, supported by existing relevant package tests and focused review.
- [ ] Router tests preserve closed relocated WASI fixture boundaries and actual CI enforcement; the cycle supplies the pinned policy linter without skipped real fixtures. Focused static-driver tests and full router suite pass.
- [ ] Retained cycle2 failure evidence remains unchanged; fresh full-cycle and all receiving obligations stay with .13.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

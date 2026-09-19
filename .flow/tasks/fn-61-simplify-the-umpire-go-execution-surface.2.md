---
satisfies: [R1, R2, R5]
---
# fn-61-simplify-the-umpire-go-execution-surface.2 Migrate generated and end-to-end callers to portable plans

## Description
Move repository-level test generation and handwritten end-to-end coverage to the facade from Task `.1` (R1-R2). Remove duplicate low-level assembly and assertions from top-level tests; retain detailed engine behavior in package-local tests.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-gen-tests-go/**`, `tools/umpire/temporal/nexus/testdata/**`, `tests/umpire4_caller_closure_generated_test.go`, `tests/umpire4_caller_closure_test.go`, `tests/umpire4_portable_executor_test.go`, `tests/umpire4_run_evaluation_test.go`, `tests/umpire4_run_evaluation_negative_control_test.go`, `tests/umpire4_testenv_test.go`
**Touches:** [tools/umpire/cmd/umpire-gen-tests-go/**, tools/umpire/temporal/nexus/testdata/**, tests/umpire4_caller_closure_generated_test.go, tests/umpire4_caller_closure_test.go, tests/umpire4_portable_executor_test.go, tests/umpire4_run_evaluation_test.go, tests/umpire4_run_evaluation_negative_control_test.go, tests/umpire4_testenv_test.go]

### Approach
- Change generated operational tests to load/construct the checked-in portable protobuf plan and call the root executor instead of embedding `runner.InputBinding` and invoking `runner.Run`.
- Keep generated inputs deterministic and ahead-of-time; a test invocation must not start Lean or spawn one Go subprocess per verification.
- Replace handwritten top-level runner/runtime/Nexus assembly with result-level assertions through the facade. Move only unique low-level invariants to their existing package-local suites; delete duplicate checks already covered there.
- Migrate the legacy HTTP canary integration to direct facade or generated gRPC execution before its imported packages are removed in later tasks.
- Separate offline Run Evaluation from live resident execution: keep the live normal/negative proof on `ExecutionResult`, and feed the offline checker from exact checked-in four-member execution-set fixtures. Add the missing normal fixture beside the existing duplicate-delivery run set and keep corruption/ambiguity controls in the checker suite.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-gen-tests-go/generate.go:13-145,234-326` — current generated runner and checker template
- `tests/umpire4_caller_closure_generated_test.go:114-285` — emitted binding/runtime workflow
- `tests/umpire4_caller_closure_test.go:415-517,800-919` — handwritten adapter and execution assembly
- `tests/umpire4_run_evaluation_test.go:45-95,141-200` — live runner-to-offline-checker coupling
- `tests/umpire4_run_evaluation_negative_control_test.go:83-180` — paired live negative-control coupling
- `tests/umpire4_testenv_test.go:23-59` — shared integration setup

**Optional** (reference as needed):
- `tools/umpire/temporal/nexus/*_test.go` — detailed adapter invariants that should remain local

### Key context
The generator must consume stable plan artifacts, not reconstruct behavioral intent from current model code at test runtime. Preserve the complete `^TestUmpire` tagged-test naming convention so the live gate continues to select every migrated test.

### Acceptance
- [ ] Generated and handwritten end-to-end tests construct one executor and submit protobuf plans; none import runner, runtime, local/Nexus binding, evaluation-contract, or raw Evidence types.
- [ ] Generated output remains deterministic and its drift check passes.
- [ ] Runtime details removed from top-level tests are either redundant or retained in package-local tests with identical negative-case coverage.
- [ ] Offline Run Evaluation tests consume exact normal and duplicate-delivery four-member fixtures and no longer construct those sets through the resident runner; the tagged live suite independently proves both plans against the disposable cluster.
- [ ] The former HTTP integration uses the facade/gRPC path before executorhttp, runner, runtime, or Temporal adapter packages are deleted.
- [ ] Tests verify decisions, typed statuses, evidence/result identity, run freshness, eventual completion, and cluster cleanup through `ExecutionResult`.
- [ ] `make umpire-check-generated-go-test` and the focused tagged integration tests pass.

## Acceptance
- [ ] Repository-level callers advance R1-R2 through portable plans only.
- [ ] Generated drift and end-to-end behavior remain exact.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

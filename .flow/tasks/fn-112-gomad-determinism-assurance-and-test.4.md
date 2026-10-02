---
satisfies: [R4]
---
# fn-112-gomad-determinism-assurance-and-test.4 Plumb diagnostics through the Runner and add the trace differ

## Description
Host half of the localiser (R4): a `--diagnostics` flag, retention of the diagnostic trace, and a differ that reports the first diverging ordinal for two fresh same-seed runs.

**Size:** M
**Files:** `tools/gomad3/cmd/gomad/internal/cli/cli.go`, `qualify.go`, `tools/gomad3/runner/runner.go`, `tools/gomad3/runner/internal/execution/process.go`, `choicetrace.go`, `tools/gomad3/qualification/qualification.go`, `tools/gomad3/cmd/gomadtool/main.go` and a new subcommand file, the differ in the existing `choice` package, new test files and one fixture directory
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/cli.go, tools/gomad3/cmd/gomad/internal/cli/qualify.go, tools/gomad3/cmd/gomad/internal/cli/diagnostics_test.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/internal/execution/process.go, tools/gomad3/runner/internal/execution/choicetrace.go, tools/gomad3/runner/internal/execution/diagnostics_toolchain_test.go, tools/gomad3/qualification/qualification.go, tools/gomad3/cmd/gomadtool/**, tools/gomad3/choice/*.go, tools/gomad3/internal/gomadtool/conformance/testdata/diagnostic_fault/**]

### Approach
- Mirror how `--choices` and `--choice-bytes` flow from the CLI to the reserved environment list and the process launcher.
- The diagnostic setting is part of execution identity when enabled, like `--choices`; absent, identities are unchanged.
- When `qualify` finds repetitions that differ and diagnostics are on, retain both traces and report the first diverging ordinal and fields beside the existing field-name report.
- The differ is a `gomadtool` subcommand taking two trace paths. It handles traces of unequal length and a divergence at ordinal 0.
- Fixture: use the task 3 fault switch to perturb a draw at a known ordinal and assert the differ names it.
- Add no new package: the differ lives in the existing `choice` package, so `architecture_test.go` is untouched. Put new tests in new files, and run the fixture through the toolchain-test launcher in `runner/internal/execution`, not the conformance campaign.
- Documentation of the flag and the differ is task 10's; record the flag name, subcommand name, and exit statuses in the done summary.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:432-447` and `:861-866` — choice flag parsing
- `tools/gomad3/runner/runner.go:1232-1356` — reserved env and option flow
- `tools/gomad3/runner/internal/execution/process.go:22-28` — descriptor hand-off
- `tools/gomad3/qualification/qualification.go:394-418` — `firstDivergence`
- `tools/gomad3/cmd/gomadtool/main.go:22` — subcommand list

**Optional** (reference as needed):
- `tools/gomad3/runner/evidence.go:51-72` — evidence fields
- `tools/gomad3/architecture_test.go:24` — import allowlist

### Key context
- fn-105 task 12 (D12) waits on this task; a dependency is recorded.
- Replay and forced-prefix executions: state in the docs whether diagnostics apply there; rejecting the combination is acceptable.
## Acceptance
- [ ] `explore` and `qualify` accept `--diagnostics`; without it, plan, Campaign, and Artifact identities are byte-identical to before
- [ ] A `nondeterministic` qualification with diagnostics on retains both traces and reports the first diverging ordinal and fields
- [ ] The differ reports the injected site for the fault fixture, and handles unequal lengths and ordinal 0
- [ ] Invalid or truncated trace input is invalid input, never a partial result
- [ ] Flag, subcommand, and exit statuses are in the done summary for task 10 to document
- [ ] `make -C tools/gomad3 test-host test-runtime` pass on darwin/arm64; linux status recorded
## Done summary
Task 4 implements host diagnostic collection and comparison for R4 in the uncommitted working tree over HEAD 1d7272e654f268f9a45f3fe965918fe2522827c6. Round2 implementation review returned SHIP and confirmed both review findings fixed and R4 met; review-round2.json and review-round2.md retain the verdict. The parent owns Flow completion.

`gomad explore --diagnostics` and `gomad qualify --diagnostics` imply choices, reserve an independent bounded diagnostic descriptor, and bind the diagnostic profile in execution identity. Capacity is derived from choice capacity and rejected above 64 MiB. Diagnostics-off portable-plan, Campaign-plan and Artifact canonical bytes match pre-change captures; normal and failure qualification report hashes match the original implementation. Complete traces are private, immutable, synced Campaign sidecars retained even when successful Artifacts are discarded. Qualification hashes diagnostic evidence, retains each repetition's full evidence and explicit complete/unavailable status, validates reference/digest bindings, and reports the first divergent ordinal and fields with the compared repetition numbers. Watchdog/cancellation reports preserve failure evidence and available traces in either ordering; comparison uses the first complete trace and subsequent complete traces, with no localization for unavailable pairs. Saved reports reject removed/contradictory references, evidence, status and baseline bindings.

`gomadtool diagnostic-diff [--json] EXPECTED_TRACE ACTUAL_TRACE` exits 0 for equal records, 1 for divergence, 2 for invalid arguments or unreadable/malformed/truncated/incomplete traces, and 3 for output failure. Both entire inputs are validated before comparison; unequal lengths and ordinal 0 are covered. The launcher-owned task3 draw fault fixture localizes ordinal 5, field runtime_cheap_rand_draws. Ordinary replay, qualification failure replay, replay-successes and guided replay preserve stored identity while disabling fresh diagnostic collection. Diagnostics with forced-prefix exploration are rejected. Portable plans, shards, guidance, coordinator transport and resume preserve the setting. Descriptor plumbing through the supervisor/bootstrap, plus the existing identity/config restoration paths, accounts for files beyond the task's initial touch list; no package, third-party dependency or architecture-gate exemption was added.

On darwin/arm64, full `make -C tools/gomad3 test-host test-runtime` passed in make-test-host-runtime.log. After the report-only review repairs, full `make -C tools/gomad3 test-host` passed again in review-repair-test-host.log; runtime is reused explicitly because the repair changed only qualification.go and qualification/CLI regression tests, with no runtime or launcher changes. Final focused diagnostic tests and vet passed in review-repair-focused.log and review-repair-vet.log. The refreshed production qualification smoke passed with two complete traces and no retained successful Artifacts; review-repair-live-qualified-report.json retains that result. git diff --check passed. Red/green regression commands and exact command/result logs are recorded in worker-commands.json and worker-repair-commands.json. worker-repair-source-bindings.json binds all 29 final source files; toolchain key is 6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee.

Linux was not run. Root `make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD` exits 2 because root-module package discovery does not include the nested Gomad module; root-lint.log records the existing limitation, so root lint is not claimed clean. Task10 flag/subcommand/exit-code and replay/report semantics are recorded in .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-4/task10-handover.md. No files were staged, no commits were created and no PR was opened.

stage: plan-sync - skipped(config: planSync.enabled != true)
stage: impl-review - ran (model: gpt-6-astra at high) - NEEDS_WORK, SHIP
## Evidence
- Commits:
- Tests: ./.toolchain/bin/go test -count=1 -tags test_dep . ./choice ./cmd/gomadtool ./cmd/gomad/internal/cli ./qualification ./runner -run 'Diagnostic|CoordinatorTransport|TestPublicPackagesDoNotExportTypeAliases' (exit 0; cwd tools/gomad3; diagnostic-focused-repaired.log), env -u GOROOT PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH make -C tools/gomad3 test-host test-runtime (exit 0; cwd .; make-test-host-runtime.log), ./.toolchain/bin/go vet -tags test_dep ./choice ./runner/... ./qualification/... ./cmd/gomad/internal/cli ./cmd/gomadtool (exit 0; cwd tools/gomad3; focused-vet.log), env -u GOROOT PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD (exit 2; known nested-module discovery limitation, not a clean root lint gate; cwd .; root-lint.log), env -u GOMADSEED -u GOMAD3_CHILD_SEED ./.toolchain/bin/go run ./cmd/gomad qualify --diagnostics --seed=7 --repeat=2 --json --toolchain-root=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain --working-dir=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/internal/gomadtool/conformance/testdata --artifacts=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain/diagnostic-task4-smoke go-run ./diagnostic_fault (exit 0; cwd tools/gomad3; live-qualify.log), env -u GOMADSEED -u GOMAD3_CHILD_SEED ./.toolchain/bin/go run ./cmd/gomadtool diagnostic-diff --json ../../.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-4/live-fresh-1.bin ../../.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-4/live-fresh-2.bin (exit 0; cwd tools/gomad3; live-diff.log), ./.toolchain/bin/go test -count=1 -tags test_dep ./qualification -run 'TestDiagnostics(PreserveInterruptedQualificationEvidence|RejectCorruptedSavedReportBindings)' (exit 1; Regression failures before implementation; cwd tools/gomad3; review-repair-red.log), ./.toolchain/bin/go test -count=1 -tags test_dep ./qualification -run 'Diagnostic' (exit 0; cwd tools/gomad3; review-repair-green.log), ./.toolchain/bin/go test -count=1 -tags test_dep ./qualification/... ./cmd/gomad/internal/cli ./runner -run 'Diagnostic' (exit 0; cwd tools/gomad3; review-repair-focused.log), ./.toolchain/bin/go test -count=1 -tags test_dep -overlay=/tmp/gomad-task4-report-off-baseline/overlay.json ./qualification -run TestCaptureOffReportBaseline -v (exit 0; cwd tools/gomad3; report-off-baseline.log), env -u GOROOT PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH make -C tools/gomad3 test-host (exit 0; cwd .; review-repair-test-host.log), ./.toolchain/bin/go vet -tags test_dep ./qualification/... ./cmd/gomad/internal/cli (exit 0; cwd tools/gomad3; review-repair-vet.log), env -u GOMADSEED -u GOMAD3_CHILD_SEED ./.toolchain/bin/go run ./cmd/gomad qualify --diagnostics --seed=7 --repeat=2 --json --toolchain-root=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain --working-dir=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/internal/gomadtool/conformance/testdata --artifacts=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain/diagnostic-task4-repair-smoke go-run ./diagnostic_fault (exit 0; cwd tools/gomad3; review-repair-live-qualify.log), ./.toolchain/bin/go run /tmp/gomad-task4-saved-report-repair-probe.go ../../.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-4/review-repair-live-qualified-report.json (exit 0; cwd tools/gomad3; saved-report-probe-after.log), git diff --check (exit 0; cwd .)
- PRs:
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

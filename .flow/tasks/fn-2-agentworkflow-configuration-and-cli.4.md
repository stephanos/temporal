---
satisfies: [R3, R4, R5]
---
# fn-2-agentworkflow-configuration-and-cli.4 Replace CLI dispatch with Cobra while preserving behavior

## Description
Create the internal Cobra command tree for R3, R4, and R5, leaving `cmd/agentworkflow/main.go` as a thin process entry point. Preserve the existing injected context/writer test seam and explicit exit mapping.

**Size:** M
**Files:** `tools/agentworkflow/internal/cli/**`, `tools/agentworkflow/cmd/agentworkflow/main.go`, `tools/agentworkflow/go.mod`, `tools/agentworkflow/go.sum`
**Touches:** [tools/agentworkflow/internal/cli/**, tools/agentworkflow/cmd/agentworkflow/main.go, tools/agentworkflow/go.mod, tools/agentworkflow/go.sum]

### Approach
- Add Cobra v1.10.2 and construct a fresh root command per execution.
- Use `RunE`, positional validators, command-local flags, injected context, and separate stdout/stderr writers.
- Silence Cobra's automatic usage/error emission and map parse, argument, operational, outcome, and writer errors through the stable exit categories.
- Preserve positional run objectives, repeatable flags, nested `config explain`, help aliases, JSON output, and whole-run model override behavior.

### Investigation targets
**Required** (read before coding):
- `tools/agentworkflow/cmd/agentworkflow/main.go:34-74` — current command dispatch
- `tools/agentworkflow/cmd/agentworkflow/main.go:324-407` — shared flags and request conversion
- `tools/agentworkflow/cmd/agentworkflow/main.go:449-567` — exit, output, and writer behavior
- `tools/agentworkflow/cmd/agentworkflow/main_test.go:21-169` — CLI compatibility coverage
- `tools/agentworkflow/go.mod` — nested module dependencies

**Optional** (reference as needed):
- `https://pkg.go.dev/github.com/spf13/cobra@v1.10.2` — supported IO, args, and error APIs

### Key context
- Use `SetOut` and `SetErr`; deprecated `SetOutput` and package-global command/flag state are excluded.
- Successful machine-readable output remains isolated on stdout.

### Acceptance
- [ ] Failing compatibility tests first cover commands, flags, positionals, help/errors, writers, JSON streams, and exit codes.
- [ ] A fresh Cobra tree is built for every execution.
- [ ] `--model` still overrides configured stage models for run and resume backend identity.
- [ ] The executable entry point delegates to the internal CLI and contains no command implementation.
- [ ] CLI tests and command build pass with `-tags test_dep`.

## Acceptance
- [ ] R4 Cobra command compatibility is covered.
- [ ] R3 CLI model precedence is covered.
- [ ] R5 executable entry point is thin.


## Done summary
Replaced the standalone Agentworkflow command's manual dispatcher with a fresh Cobra v1.10.2 tree under `internal/cli`, leaving `main.go` process-only. Compatibility coverage preserves every command, nested help, local and repeatable flags, positional objectives, stdout/stderr and JSON isolation, model override behavior, writer failures, and stable exit categories.

TDD captured the expected legacy-dispatch RED before the Cobra implementation turned the focused and complete tagged suites GREEN. The command build, focused formatting/import checks, tagged vet/race suites, and task-scoped case-sensitive 13-analyzer lint all pass; the approved inherited branch-wide baseline exception remains unchanged.

stage: impl-review - ran | verdict: SHIP | session: 01a0352e-3edc-7151-ba3c-dfaf13535508
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 753efe97a54b2ef4c297205a7f8ef9771108f09b
- Tests: baseline: green (cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./...; cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow), RED: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./cmd/agentworkflow -run TestCLI compatibility set - legacy dispatcher failed Cobra help, nested help, unknown-flag, and positional-arity expectations, GREEN: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./internal/cli -run TestCLI compatibility set, cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow, cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go test -race -tags test_dep ./..., .bin/gci-v0.13.6 diff --skip-generated -s standard -s default <task Go files>, GOLANGCI_LINT_BASE_REV=6c907aaac45b9faa05eeadd55f4cc1c77c07abf2 GOLANGCI_LINT_FIX=false make LOCALBIN=/tmp/fn2-4-lint-tools lint-code (disposable case-sensitive clone at 753efe97a; 13 analyzers plus errortype vet, 0 issues), NO_RECEIPT: unittest gate receipt not warrantable because shared checkout has pre-existing config/development.yaml changes
- PRs:

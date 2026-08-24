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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

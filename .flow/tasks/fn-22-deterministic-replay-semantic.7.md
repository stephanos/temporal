---
satisfies: [R8, R10]
---
# fn-22-deterministic-replay-semantic.7 Expose the bounded replay command and its report

## Description
Add `umpire-replay run`: the subject from `--case <fixture.json>` and `--run <run.json>`, the set and the Query or exploration target the bridge recovers it by, the deployment through `binding.RegisterFlags`, `--promotion-root`, the fixed limits by name only. One canonical JSON report to stdout with `admission`, `semanticReplay`, the `key` and the Case `identity` apart, `reproduction` and each rerun's outcome, `reduction` (completion and every edit's fate), `limits`, `cleanup`, `proposal` (digest, path, written, or its status) and `failure` as separate fields; bounded progress on stderr. Exit codes: 0 reproduced with a complete reduction and, when a root is named, the proposal written; 1 not-reproduced; 2 indeterminate, incomplete or stopped; 3 tooling failure, which is also a rejected subject (the `admission` field names the reason, nothing ran) and a proposal write failure or existing destination (the `proposal` field names it, the rest of the report stands). `make umpire-replay` and `make umpire-replay-run` wrap it beside `umpire-fuzz`.

### Approach
- The command mirrors `umpire-fuzz run`'s shape: parse and refuse before opening anything, open once, drive, settle, render, cap the report without truncating; the command-edge helpers come from `tools/umpire/internal/cli`.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-replay/`

**Size:** M
**Files:** `tools/umpire/cmd/umpire-replay/main.go`, `tools/umpire/cmd/umpire-replay/run.go`, `tools/umpire/cmd/umpire-replay/run_test.go`, `tools/umpire/replay/report.go`, `tools/umpire/replay/report_test.go`, `Makefile`, `model/README.md`
**Touches:** `tools/umpire/cmd/umpire-replay/**`, `tools/umpire/replay/report*.go`, `Makefile`, `model/README.md`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review round one; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] The command exposes no arbitrary Driver, checker, executable, semantic edit or compatibility option, and refuses the command line before opening anything.
- [ ] Every exit code is pinned, a rejected subject and a proposal failure included; output, cancellation and reporting failure are canonical and bounded; the report cap never truncates.
- [ ] A reporting or proposal failure never installs a regression and never reruns target effects.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

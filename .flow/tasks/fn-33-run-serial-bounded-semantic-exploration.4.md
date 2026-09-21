---
satisfies: [R4]
---
# fn-33-run-serial-bounded-semantic-exploration.4 Expose the closed serial umpire-fuzz command

## Description
Add the bounded `umpire-fuzz run` command over the completed serial coordinator. Emit canonical summary/error output that separates selection, preparation, Run start, decisive result, per-target coverage, unreachable and missed targets, counterexamples, exhaustion, limits, stop/lost iteration, and tooling failure.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-fuzz/**`, `Makefile`, `model/README.md`
**Touches:** [tools/umpire/cmd/umpire-fuzz/**, Makefile, model/README.md]

### Approach
- Flags name the set, the deployment binding (as `umpire-run`), the candidate cap and byte caps; none names a target or widens a Limit.
- Output is one canonical JSON summary on stdout at the end and diagnostics on stderr; exit codes: 0 exhausted, 1 counterexample or violated coverage, 2 limit-reached or stopped, 3 tooling failure.
- `make umpire-fuzz SET=<set>` wraps the command; document it beside `umpire-run` in `model/README.md`.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-run/main.go`, `run.go:26-40` — exit-code and configuration conventions.
- `model/README.md` — the Case production and runtime sections.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-fuzz/... && GOLANGCI_LINT_BASE_REV=<base> make lint-code-fast`

### Re-plan note (2026-09-21)
Re-planned on fn-85's exploratory set after fn-86 R6 deleted the variation Space this task was first written against; see the spec's **Re-plan on fn-85** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] The summary reports selected, prepared, started, decisive, covered, unreachable, missed and counterexample counts without collapsing them, and lists each counterexample.
- [ ] Terminal status is exactly one of exhausted, limit-reached, stopped, tooling-failure, with its exit code.
- [ ] Unexecuted, inconclusive and cleanup-uncertain work is never reported as coverage.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

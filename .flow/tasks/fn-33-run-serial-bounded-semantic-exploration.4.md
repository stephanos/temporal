---
satisfies: [R4]
---
# fn-33-run-serial-bounded-semantic-exploration.4 Expose the closed serial umpire-fuzz command

## Description
Add the bounded `umpire-fuzz run` command over the completed serial coordinator. Emit canonical summary/error output that separates selection, preparation, Run start, decisive result, per-target coverage, unreachable, violated and attempted targets, counterexamples, exhaustion, limits, stop/lost iteration, and tooling failure, with one progress line per candidate on stderr. Starts after `.6`, whose state machine its exit codes rest on.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-fuzz/**`, `Makefile`, `model/README.md`
**Touches:** [tools/umpire/cmd/umpire-fuzz/**, Makefile, model/README.md]

### Approach
- Flags name the set, the deployment binding (as `umpire-run`), the candidate cap and byte caps; none names a target or widens a Limit.
- Output is one canonical JSON summary on stdout at the end; stderr carries diagnostics and one progress line per candidate (identity, selected target key, outcome) so an operator can follow a campaign of hundreds of Runs; exit codes: 0 exhausted, 1 counterexample or violated coverage, 2 limit-reached or stopped, 3 tooling failure.
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
- [x] The summary reports selected, prepared, started, decisive, covered, unreachable, violated, attempted and counterexample counts without collapsing them, and lists each counterexample; stderr carries one progress line per candidate.
- [x] Terminal status is exactly one of exhausted, limit-reached, stopped, tooling-failure, with its exit code.
- [x] Unexecuted, inconclusive and cleanup-uncertain work is never reported as coverage.
## Done summary
The closed serial command. `umpire-fuzz run` names the exploratory set, the deployment as `umpire-run` names it (`--grpc`, `--http`, `--namespace`, `--task-queue`, `--nexus-endpoint`, `--handler-task-queue`, `--create`), the campaign's caps (`--max-candidates`, `--max-case-bytes`, `--max-run-events`, `--max-report-bytes`, `--run-timeout`, `--timeout`) and the bridge's location (`--model-root`, `--bridge`); no flag names a target or widens a declared Limit, and the flag set refuses one. It opens the deployment binding once, spawns the bridge in the model package and initializes it under the Profile identity `umpire-fuzz.<namespace>`, drives the coordinator of task .6, and writes one canonical JSON summary to stdout: the terminal (`status`, `limit`, `failure`, `lost`), set, Profile, machine, budget and Limits, the counters (planned, prepared, started, decisive, rejected, failed, inconclusive, skipped, Case bytes, Run Events), the bridge's coverage counts and per-target ledger copied as answered, the counterexamples and one line per candidate (identity, target, kind, observation, detail, credited keys). Exit codes: 3 tooling failure, then 1 counterexample or violated coverage, then 2 cap or stop, else 0 exhausted. The report cap is checked on the rendered summary and, over it, the terminal, the counters and the counterexamples alone are written (an exhausted campaign as `limit-reached`), never a truncated report. The bridge's stderr is the command's, so the progress line per candidate is the bridge's; the command adds one line naming the terminal. `make umpire-fuzz` builds it and `make umpire-fuzz-run SET=<set>` builds the bridge and runs a campaign against the deployment the `UMPIRE_FUZZ_*` variables name; both documented beside `umpire-run` in `model/README.md`. Tests drive the command over a scripted bridge and scripted binders through exhaustion with coverage copied from the ledger only, a counterexample (exit 1, also at a cap), a candidate cap (exit 2), a stop by timeout naming the lost iteration, a binding failure, a bridge tooling failure and a campaign that cannot open (exit 3), preparation rejections that never count as coverage, the report cap, every command-line rejection before anything opens (a `--target` or `--search` flag included), and the bridge path derived from the model root.
## Evidence
- Commits: f40c42c355f9efead1955106512ee2fb7413adc6, 47e6d5b3479109755100647714c1fb558e75b85e
- Tests: go test -count=1 -timeout 180s -tags test_dep ./tools/umpire/cmd/umpire-fuzz/... ./tools/umpire/campaign/... ./tools/umpire/binding/... ./tools/umpire/cmd/umpire-run/..., go vet -tags 'test_dep integration' ./tools/umpire/..., GOLANGCI_LINT_BASE_REV=HEAD~1 make lint-code-fast, make -n umpire-fuzz-run SET=x
- PRs:
---
satisfies: [R8, R10]
---
# fn-22-deterministic-replay-semantic.8 Expose the bounded replay command; close the matrices, live proof, gates and documentation

## Description
Add `umpire-replay run`: the subject from `--case <fixture.json>` and `--run <recorded-run.json>`, the set and the Query or exploration target the bridge recovers it by, the deployment through `binding.RegisterFlags`, `--promotion-root`, the fixed limits by name only; admission before `binding.Open`. One canonical JSON report to stdout with `admission`, `semanticReplay`, the `key` and the Case `identity` apart, `reproduction` and each rerun's outcome, `reduction` (completion and every edit's fate), `limits`, `cleanup`, `proposal` (digest, path, written, or its status) and `failure` as separate fields; bounded progress on stderr. Exit codes: 0 reproduced with a complete reduction and, when a root is named, the proposal written; 1 not-reproduced; 2 indeterminate, incomplete or stopped; 3 tooling failure, which is also a rejected subject (the `admission` field names the reason, nothing ran) and a proposal write failure or existing destination (the `proposal` field names it, the rest of the report stands). `make umpire-replay` and `make umpire-replay-run` wrap it beside `umpire-fuzz`. Then close the admission, key, semantic replay, rerun, reduction, evidence core, proposal, cancellation, limit and output matrices; run the negative control end to end through `umpire-replay run` against the test cluster in the live suite (reproduced, irreducible or minimized, proposal compiled and written under a scratch root, proving the mechanism only); reconcile the documentation with the Case Runtime, amend the UMPIRE4 spec's Exploration section for the replay classes and the key, and remove active references to replay bundles, Run Evaluation, caller-closure runtime support and SDK replay as proof.

### Approach
- The command mirrors `umpire-fuzz run`'s shape: parse and refuse before opening anything, admit, open once, drive, settle, render, cap the report without truncating; the command-edge helpers and the proposal writer come from `tools/umpire/internal/cli`.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-replay/; cd model && lake build && LEAN_NUM_THREADS=1 make -C .. lint-model; make umpire-check-goldens umpire-check-case-runtime-conformance umpire-check-inventory umpire-check-model-module-index umpire-check-exploration-bridge; go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/...; make lint-code-fast`

**Size:** L
**Files:** `tools/umpire/cmd/umpire-replay/main.go`, `tools/umpire/cmd/umpire-replay/run.go`, `tools/umpire/cmd/umpire-replay/run_test.go`, `tools/umpire/replay/report.go`, `tools/umpire/replay/report_test.go`, `tests/testpilot_nexus_control_case_test.go`, `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** `tools/umpire/cmd/umpire-replay/**`, `tools/umpire/replay/report*.go`, `tests/testpilot_nexus_control_case_test.go`, `Makefile`, `model/*.md`, `.plans/*.md`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review rounds one and two; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] The command exposes no arbitrary Driver, checker, executable, semantic edit or compatibility option, refuses the command line before opening anything, and admits before `binding.Open`; every exit code is pinned, a rejected subject and a proposal failure included; the report cap never truncates.
- [ ] Focused Lean, Go, `-tags test_dep`, integration, formatting and lint gates pass, and the live negative-control proof runs through the command.
- [ ] Docs keep the three replay classes, the key against the identity, the control's proposal as mechanism only, and every retired or deferred boundary; existing comments are preserved or reworded only where the invariant they describe changed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

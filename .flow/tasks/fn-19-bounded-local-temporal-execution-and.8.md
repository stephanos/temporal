---
satisfies: [R7]
---
# fn-19-bounded-local-temporal-execution-and.8 Integrate generated Go tests and synchronize runtime documentation

## Description
### Umpire4 reconciliation (normative)

There is no public `umpire-local-run`, `umpire-run-local`, or separate run-tests CLI. Generate ordinary Go tests through `umpire-gen-tests` / `umpire-gen-tests-go`; those tests call the reusable `tools/umpire/runner` directly and preserve normal `go test` discovery, filtering, breakpoints, and failures. An internal harness may prove adapters but is not user-facing.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Complete R7 with the exact direct/root command, status contracts, public usage documentation, and honest roadmap status.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-local-run/main.go`, `tools/umpire/cmd/umpire-local-run/main_test.go`, `tools/umpire/runtime/README.md`, `Makefile`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [tools/umpire/cmd/umpire-local-run/main.go, tools/umpire/cmd/umpire-local-run/main_test.go, tools/umpire/runtime/README.md, Makefile, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Implement exact three-flag grammar, preflight-before-run behavior, the parent ordered summary/error schemas, boolean publication semantics, and statuses 0/1/2.
- Take Task `.7`'s admitted in-memory set and make the sole production `PublishSet` call; preflight output-root safety where fn-18 permits, publish before stdout, report both booleans on every tooling failure, and never auto-rerun.
- Add only the root `umpire-run-local` Make target with exact required-variable checks.
- Document the two-member input, fixed profile/budgets, four-member output, failed-run artifact behavior, evidence dispositions, local-only authority, and downstream conformance boundary.
- Update C6/C9 and pilot-step implementation status only after the live test is implemented; keep Milestone B unintegrated until live semantic conformance.

### Investigation targets
**Required** (read before coding):
- Task `.7` live API/publication result
- `Makefile:988-1118` Umpire target conventions
- `.plans/UMPIRE4_COMPONENTS.md:303-327,394-416,613-640,706-716`
- parent exact CLI/status and boundary contracts

### Acceptance
- [ ] Direct and root command exact ordered bytes/statuses agree for success, operational failed/incomplete, preflight error, publication error, and post-publication broken summary output.
- [ ] Missing/extra/malformed arguments fail before execution with no stdout or side effect.
- [ ] Documentation gives one copy-paste local run and does not imply semantic conformance or remote support.
- [ ] No model-local Makefile, CI, remote adapter, or prohibited legacy reference/use is added.
## Acceptance
- [ ] R7 command, root UX, docs, and implementation-time roadmap status are complete.
- [ ] All focused suites and the one bounded live command pass.
- [ ] Existing comments remain preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

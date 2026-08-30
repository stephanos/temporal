---
satisfies: [R1, R2, R5, R6, R7]
---

# fn-20-local-execution-semantic-conformance.6 Expose the local Run Evaluation command and immutable publication

## Description
Add the exact direct/root command, sole production publication call, and frozen status/reporting behavior (R1/R2/R5-R7).

**Size:** M
**Files:** `tools/umpire/cmd/umpire-local-run-evaluation/main.go`, `tools/umpire/cmd/umpire-local-run-evaluation/main_test.go`, `tools/umpire/runevaluation/command_test.go`, `Makefile`
**Touches:** [tools/umpire/cmd/umpire-local-run-evaluation/main.go, tools/umpire/cmd/umpire-local-run-evaluation/main_test.go, tools/umpire/runevaluation/command_test.go, Makefile]

### Approach
- Implement the exact two-flag grammar, fixed sibling-pair discovery/handshake, parent summary/error field order, nullable fields, authoritative booleans, and statuses 0/1/2.
- Call Task `.4` and make the sole production fn-18 `PublishSet` call only after complete six-member admission; publish before stdout and never delete/rewrite/rerun on a reporting failure.
- Treat operational/Observation Evaluation/semantic non-success as a published status 2, distinct from admission/checker/output/publication/reporting status 1.
- Add only the repository-root `umpire-check-local-run-evaluation` target; build/install the Go command and Lean checker as a sibling pair and validate SET/OUTPUT_ROOT before invocation.
- Enforce R2 against accidental or tool-controlled pathname substitution: use descriptor-bound execution on Linux and, on Darwin, a private mode-0700 pair with embedded digest and `UF_IMMUTABLE` held on the exact open vnode through child wait. Treat a concurrent same-UID actor able to clear vnode flags, mutate or ptrace the launcher, or otherwise compromise the process as outside the threat boundary.
- Test permission/path/conflict, signal/cancellation, idempotent existing destination, broken stdout/stderr, and post-publication reporting cases with exact bytes and side-effect assertions.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.8.md` — command/publication/status precedent
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.10.md` — immutable publication API and retry behavior
- `Makefile:988-1118` — current-model root target conventions
- `model/lakefile.toml:18-20` — Lean executable output
- parent spec §API Contracts — exhaustive CLI schemas and exit matrix

## Acceptance
- [ ] Direct and Make commands produce exact summary/error bytes and statuses for satisfied, violated, non-accepted, operational failed/incomplete, input, checker, output-invariant, publication, and reporting cases.
- [ ] Missing/extra/malformed arguments and unsafe output roots fail before checker startup; no stdout or partial destination exists.
- [ ] The CLI is the sole production publisher, identical publication is idempotent, and post-publication output failure includes the authoritative destination.
- [ ] The installed-pair contract exposes no checker path/PATH/env/plugin override and detects accidental or tool-controlled substituted siblings under the explicit R2 threat boundary.
- [ ] All Make changes are confined to the repository-root Makefile and existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

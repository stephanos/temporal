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
Implemented the exact local Run Evaluation CLI and repository-root Make target, including strict two-flag parsing, semantic preflight before checker resolution, canonical status/reporting schemas, sole post-admission publication, idempotent immutable destinations, and complete error/cancellation behavior. The installed sibling contract now binds the embedded checker digest to a private mode-0700 pair, executes the verified descriptor on Linux, and holds `UF_IMMUTABLE` on the exact open vnode through child wait on Darwin. The Flow-authoritative R2 re-plan explicitly limits this guarantee to accidental or tool-controlled pathname substitution and excludes a concurrent same-UID actor able to clear vnode flags, mutate or ptrace the launcher, or otherwise compromise the process.

Focused Go tests, the real Make-installed Lean/Go sibling proof, race, fuzz, vet/format, model lint, and the full 240-job Umpire regression are green. Full repository `make lint-code` retains the inherited `tools/umpire/runtime/errors.go:60:9 et:unw` finding outside this task. Codex review session `01a05134-1b1a-7ab3-9d47-41f81e544465` returned SHIP after withdrawing the prior same-UID finding under the approved R2 threat boundary, with zero introduced or pre-existing findings and no unaddressed requirements.
## Evidence
- Commits: 7bcd4b747, c357d39d1, 0f7f414f9, 7ba112a5a, 16095d4e3, a0be5a3dc, 9d8ba2872
- Tests: TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -count=1 -tags test_dep ./tools/umpire/runevaluation ./tools/umpire/cmd/umpire-local-run-evaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -count=1 -race -tags test_dep ./tools/umpire/runevaluation ./tools/umpire/cmd/umpire-local-run-evaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -tags test_dep ./tools/umpire/runevaluation -run '^$' -fuzz '^FuzzDecodeCheckerResponse$' -fuzztime=3s, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go vet -tags test_dep ./tools/umpire/runevaluation ./tools/umpire/cmd/umpire-local-run-evaluation, gofmt -d tools/umpire/runevaluation tools/umpire/cmd/umpire-local-run-evaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH make lint-model, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH make umpire-check-regression, INHERITED_RED: make lint-code reports tools/umpire/runtime/errors.go:60:9 et:unw outside fn20.6, /Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl validate --spec fn-20-local-execution-semantic-conformance --json
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)

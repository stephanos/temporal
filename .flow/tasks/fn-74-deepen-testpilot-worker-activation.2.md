---
satisfies: [R2, R6, R8]
---
# fn-74-deepen-testpilot-worker-activation.2 Integrate activation ownership into both SDK interpreters

## Description
Use task 1's activation owner in workflow and Nexus-handler interpretation (R2, R6, R8), with regression and ownership documentation updates.

**Size:** M
**Files:** `common/testing/testpilot/temporal/worker/interpreter.go`, `values.go`, `sdk_test.go`, relevant worker/session tests and README; `common/testing/testpilot/internal/execution/README.md`.
**Touches:** [common/testing/testpilot/temporal/worker/**, common/testing/testpilot/internal/execution/README.md, common/testing/testpilot/temporal/internal/activation/activation_test.go]

### Approach
- Instantiate state only for an actual interpretation. Replace manual maps, budget subtraction and separate validation/storage calls with task 1's operations. Retire superseded worker helpers only once all consumers migrate, preserving relevant comments.
- Keep prepared DAG traversal, SDK dispatch/futures/timeouts, terminal outcomes, reservation/redelivery cache and completion capability authority in their current owners.
- Preserve workflow SDK-context checks and bounded pure evaluation with the existing context strategy; canceled SDK outcomes must still follow existing outcome-validation paths. Add no wall-clock workflow deadlines or operation cancellation capability.
- Adapt direct interpreter test construction to evaluate before admission, especially Await timeout setup; preserve every existing outcome/timeout assertion. Retain real WorkflowReplayer and Stop-race coverage.
- Move the former private Slot-reference rejection assertion into the activation owner's existing ownership test; do not expose a lookup method merely to retain a worker-package assertion.
- Update worker and execution documentation to explain private state composition while retaining public low-level work-charge contracts.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:33` — workflow traversal and SDK boundaries.
- `common/testing/testpilot/temporal/worker/interpreter.go:138` — Nexus redelivery and fresh interpretation.
- `common/testing/testpilot/temporal/worker/sdk_test.go:75` — replay; Stop race and Await timeout later in the same file.
- `common/testing/testpilot/temporal/worker/session_test.go:123` — opaque authority and publication.
- `common/testing/testpilot/temporal/worker/README.md:3` — adapter ownership.
- `common/testing/testpilot/internal/execution/README.md:131` — returned work versus composition ownership.

### Quick commands
`go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot`
`go test -race -tags test_dep ./common/testing/testpilot/temporal/...`

## Acceptance
- [ ] Both production interpreters use the private module for all R2 responsibilities; SDK and delivery authorities remain separate.
- [ ] R6 replay, Start/Await/Finish, Nexus Respond, Stop/timeout/late-publication and session regressions pass with existing assertions retained.
- [ ] No public signature, Case/protocol byte, evaluator, controller store, retry or cleanup behavior changes under R8.
- [ ] Worker/execution documentation describes the final composition and unchanged work units; relevant existing comments are preserved.
- [ ] Focused unit/race checks pass. Whichever task completes last also runs the combined facade/Driver suites and `make lint-code GOLANGCI_LINT_FIX=false`, reporting exact inherited failures separately.

## Done summary
Implemented task2 composition through the frozen private activation module. Both real interpreters construct fresh state per actual interpretation, evaluate guard/input by instruction index, and admit SDK/terminal outcomes atomically. Removed worker activation reference maps and manual work/validation helpers; cached adapter instruction slices once per interpretation. No public, protocol, Case, evaluator or controller changes.

Worker handover status: in_progress before conductor review and completion. Commits: []. Base commit: 375abfe180dba72da6dd357e6abe33fa75a292a7.
stage: impl-review - ran (SHIP; model: gpt-6-astra at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)

Verification
- Physical cwd verified before reads/operations; read worker charter in chunks, Flow usage then fresh anchor, authoritative task/spec, AGENTS.md, UMPIRE4 spec, required investigation files and applicable skills. All Flow operations were reads; no lifecycle, memory, review or base-record mutations.
- Both required baseline suites passed before edits. Exact commands, logs and terminal exits appear below and in the evidence JSON.
- Red/green regression: the original TestWorkflowFinishRejectsUnevaluatedInstruction failed because Finish returned a response without prior evaluation. It now forms the unevaluated case of TestWorkflowFinishRejectsInvalidAdmission, with a valid text payload so validation alone cannot satisfy the regression. Its second case checks typed admission failure after evaluation; neither failure emits a terminal response.
- TestNexusCanceledEvaluationPreventsResponse checks canceled handler evaluation returns context.Canceled and no response kind/value/token. No operation cancellation expansion was implemented.
- Await timeout test now evaluates Start and Await before executing them, and reads the admitted status via a prepared Finish expression. All three existing expected statuses and elapsed-time assertions remain. Real WorkflowReplayer, SDK Start/Await/Finish, Nexus Respond/redelivery, Stop races, concurrent delivery, opaque authority and post-close publication tests pass.
- Migrated TestActivationValuesOwnValidatedOutcome to New/Evaluate/Admit and a real Finish. The owned-result assertion remains; the old direct private-Slot nil lookup is replaced by a nil Await-input assertion because the new interface exposes no lookup. Unchanged session tests retain opaque capability publication/consumption assertions. This assertion adaptation is called out for conductor review.
- First post-edit unit invocation failed compilation from an unused import left by removing direct lookup; removed it. Subsequent runs passed. Added focused error cases and corrected the unevaluated fixture to retain its meaningful lifecycle regression before final verification.
- Final unit/race commands exited 0. Repository lint initially reported one introduced import-shadowing finding at three sites; renamed delivery parameters. Final lint exited 2 with exactly 1,284 inherited raw findings / 825 normalized distinct (path,message) pairs, zero added and zero removed. Lint is not clean; the later errortype vet recipe step did not run because golangci-lint failed. Comparison: /tmp/fn74-task2-lint-comparison.json.
- Commands were captured once per execution, with terminal exits retained in .exit files; no run was repeated merely to observe its result. Repeated gates followed source/test/style changes.

Scope and ownership
- Only seven permitted paths changed: /tmp/fn74-task2-paths.txt. Task2-only delta: /tmp/fn74-task2.patch. Final SHA256: /tmp/fn74-task2-frozen-hashes.json.
- /tmp/fn74-task2-integrity.json compares all 7,594 preedit files and the supplied 79-file frozen baseline. No outside-scope source or administrative drift, no changed frozen baseline paths outside scope, and no removed existing comments.
- Worker/execution documentation describes private composition and unchanged low-level returned-work units. SDK DAG selection, futures, SDK context checks/background pure evaluation, timeout handling, replay reconstruction, Stop/redelivery cache and opaque completion authority retain their existing owners.
- Task1 activation sources remain frozen. Task3 diagnostics, fn79 cancellation expansion, combined final gate, review and all Flow lifecycle changes remain conductor-owned. No staging, commits, push, worktrees, nested agents or bridges.
- NO-FURTHER-SOURCE-EDITS DECLARATION: the implementation, tests and documentation are frozen at the recorded SHA256 hashes; this worker will make no further source edits after this handover.

Command receipts
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot` — log `/tmp/fn74-task2-baseline-unit.log` — terminal exit 0 (`/tmp/fn74-task2-baseline-unit.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/...` — log `/tmp/fn74-task2-baseline-race.log` — terminal exit 0 (`/tmp/fn74-task2-baseline-race.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/worker -run '^TestWorkflowFinishRejectsUnevaluatedInstruction$'` — log `/tmp/fn74-task2-red.log` — terminal exit 1 (`/tmp/fn74-task2-red.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot` — log `/tmp/fn74-task2-unit.log` — terminal exit 1 (`/tmp/fn74-task2-unit.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot` — log `/tmp/fn74-task2-unit-fixed.log` — terminal exit 0 (`/tmp/fn74-task2-unit-fixed.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot` — log `/tmp/fn74-task2-unit-final.log` — terminal exit 0 (`/tmp/fn74-task2-unit-final.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot` — log `/tmp/fn74-task2-unit-frozen.log` — terminal exit 0 (`/tmp/fn74-task2-unit-frozen.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/...` — log `/tmp/fn74-task2-race-final.log` — terminal exit 0 (`/tmp/fn74-task2-race-final.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false` — log `/tmp/fn74-task2-lint-final.log` — terminal exit 2 (`/tmp/fn74-task2-lint-final.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot` — log `/tmp/fn74-task2-unit-verified.log` — terminal exit 0 (`/tmp/fn74-task2-unit-verified.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/...` — log `/tmp/fn74-task2-race-verified.log` — terminal exit 0 (`/tmp/fn74-task2-race-verified.exit`).
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false` — log `/tmp/fn74-task2-lint-verified.log` — terminal exit 2 (`/tmp/fn74-task2-lint-verified.exit`).

Conductor follow-up: preserved the original private Slot-reference nil assertion in the activation ownership test, alongside the admitted outcome. The nil Await input is not treated as equivalent coverage. Task Touches explicitly includes this test-only move. `go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation` passed (log /tmp/fn74-task2-slot-assertion.log, exit 0). Full make lint-code GOLANGCI_LINT_FIX=false retained the exact issue set (log /tmp/fn74-task2-conductor-lint.log, exit 2). Final task delta is eight paths; production activation code unchanged.

Conductor verified all final hashes and terminal test receipts. Independent review SHIP, zero findings: /tmp/fn74-task2-impl-review.json. Reviewer rerun blocked by read-only Go temporary directory; recorded passing execution remains authoritative.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/..., TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/worker -run '^TestWorkflowFinishRejectsUnevaluatedInstruction$', TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/..., TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/..., TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false, go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation — conductor retained Slot assertion, exit 0; /tmp/fn74-task2-slot-assertion.log, make lint-code GOLANGCI_LINT_FIX=false — conductor final exit 2; exact inherited lint issue set; /tmp/fn74-task2-conductor-lint.log
- PRs:
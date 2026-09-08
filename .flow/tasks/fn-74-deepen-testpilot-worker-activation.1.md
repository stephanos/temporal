---
satisfies: [R1, R3, R4, R5]
---
# fn-74-deepen-testpilot-worker-activation.1 Encapsulate pure activation state and work accounting

## Description
Build the private activation owner and its isolated tests (R1, R3–R5). Interpreter migration follows in task 2.

**Size:** M
**Files:** new `common/testing/testpilot/temporal/internal/activation/activation.go` and `activation_test.go`.
**Touches:** [common/testing/testpilot/temporal/internal/activation/**]

### Approach
- Construct from public `testpilot.EntrypointPlan`; reject the comparable zero plan before getters and reject controller context. Cache the instruction plans once and derive the existing production work ceiling.
- Move reference resolution, guard/input evaluation, cumulative charging and atomic owned outcome storage behind construct/evaluate/admit. Delegate semantics to existing public plan methods; do not import generic Testpilot internal packages.
- Track each instruction's evaluated/skipped/admitted/failed state. Reject repeated evaluation and admission without an enabled evaluation; do not enforce DAG traversal or introduce scheduling.
- Preserve returned success/error work charges and existing copy units. Exact remaining allowance can succeed; subsequent operations at zero reject. Private tests may inject a tight allowance without exposing an override API.
- Test guard/presence behavior, invalid indices/contexts, mutation isolation, rejected partial fields, canceled contexts, and independent sequential/concurrent activations sharing a prepared plan.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:304` — current composition and charges.
- `common/testing/testpilot/temporal/worker/values.go:9` — resolver and snapshot ownership.
- `common/testing/testpilot/driver.go:221` — public evaluation/validation authority.
- `common/testing/testpilot/internal/execution/program.go:233` — getter preconditions and guard-first semantics (read-only reference).
- `common/testing/testpilot/internal/execution/values.go:156` — ceiling and work-unit semantics (read-only reference).
**Optional:**
- `common/testing/testpilot/runtime_fixture_test.go` — admitted plan fixture pattern.

### Quick commands
`go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation`
`go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation`

## Acceptance
- [ ] Private construct/evaluate/admit interface satisfies R1 without widening public plan APIs or importing Testpilot internals.
- [ ] Interface-level tests cover R3's enabled/skipped/error states and repeated evaluation/admission rejection without resets.
- [ ] R4 accounting tests pin exact-bound success, one-less failure, consumed error-path charges, zero allowance and per-activation isolation against existing work units.
- [ ] R5 tests retain immutable input/outcome snapshots and atomic failure behavior across malformed, nil, oversized and canceled inputs plus concurrent independent states.
- [ ] Focused unit/race commands pass; preserve existing comments and all unrelated changes.

## Done summary
Implemented the new private activation package with `New`, `Evaluate`, and `Admit`. It caches public instruction plans, rejects zero/controller plans, tracks single-use evaluation/admission, consumes returned work on success and error, and retains atomically validated owned fields. It introduces no SDK calls, scheduler, futures, locks, timers, I/O, registry, public plan changes, or budget override API.

Worker handover status: in_progress before conductor review and completion. No commits; base_commit: 375abfe180dba72da6dd357e6abe33fa75a292a7.
stage: impl-review - ran (SHIP; model: gpt-6-astra at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)

Verification and scope
- Physical cwd verified as /Users/stephan/Workspace/temporal/umpire before other operations. Read worker charter, usage then authoritative anchor, AGENTS.md, UMPIRE4 spec and applicable skills. Only the two new activation Go files were written in the repository.
- Activation directory absent at baseline. Fresh existing-suite baseline passed; no nonexistent-package baseline was attempted. Baseline lint exited 2 with the inherited 1,284 findings.
- Meaningful red/green: initial fixture compile error corrected (public PreparedProgram exposes Entrypoints); red-behavior then failed on assertions against minimal unimplemented operation stubs. Green run passed after implementation.
- Additional presence fixture initially failed admission because an optional VALUE read needs an explicit presence guard. Corrected it to a required STATUS read under a true guard, which prepares and fails at evaluation when not yet admitted. No preparation semantics changed.
- Final unit and race tests passed. Initial post-change lint identified two introduced test-style findings; both fixed. Final full repository make lint-code exited 2 with exactly 1,284 inherited findings, 825 unique normalized (path,message) pairs, zero additions/removals. The subsequent errortype go vet recipe step did not run because lint failed.
- Exact commands, output logs and terminal exits are listed below and in the evidence JSON. Repeated runs followed actual test/implementation/style changes, never merely re-observed results.

Requirement evidence
- R1: TestConstruction rejects the comparable zero plan before getters and controller context; workflow and handler accepted. TestEvaluationLifecycle rejects invalid indices. Public plans remain the evaluation/validation authority.
- R3: TestGuardAndOwnership, TestEvaluationLifecycle, TestPresenceAndMissingRequiredInput cover enabled/no-input/skipped/error states, guard-first/presence/boolean short-circuiting, missing reads, repeated evaluations after every terminal state, and invalid admission order. Await evaluation before Start demonstrates the module does not enforce DAG traversal.
- R4: TestWorkAccounting pins literal-input cost at 19 units (one expression visit plus two 9-byte copies), exact-bound success and 18-unit failure retaining 10 consumed units. Admission work is compared against public ValidateOutcome, with exact and one-less budgets, cumulative evaluation/admission and consumed validation failures. Zero, negative and MaxInt64 allowances reject; no production budget override exists. TestPresenceAndMissingRequiredInput pins six consumed units for true guard followed by an absent required read.
- R5: TestRejectedOutcomesAreAtomic covers nil/unspecified/unknown/protocol status, wrong type, missing/oversized/undeclared values and failed retry rejection, including Start payload rejection. Tight admission failure publishes no fields even when validation has completed its field copies. TestCanceledAndNilContexts verifies canceled/nil contexts and no retry/publication. TestRepeatedReadsOwnTheirValues checks raw outcomes, public snapshots and evaluated inputs cannot alter subsequent reads. TestIndependentActivations exercises sequential and ten concurrent distinct states sharing a plan, with per-state values and a final immutable-plan check.

Integrity and limitations
- /tmp/fn74-task1-integrity.json compares all 7,592 existing tracked/untracked nonignored files against the preedit snapshot and the supplied 77-path baseline. No existing source file changed. Two Flow spec files changed externally: .flow/specs/fn-74-deepen-testpilot-worker-activation.json and .md. Removing only the newly added HTML render-link line reproduces the preedit Markdown hash; semantic contract is unchanged. Neither file was written or reverted by this worker.
- Changed paths: /tmp/fn74-task1-paths.txt. Final SHA256: /tmp/fn74-task1-frozen-hashes.json. Lint comparison: /tmp/fn74-task1-lint-comparison.json.
- One activation is serial-only. SDK integration/task 2, diagnostic facade/task 3, cancellation expansion/fn79, reviews, memory and all Flow lifecycle mutations remain conductor-owned/out of scope.
- No staging, commits, push, worktrees, delegation, review verdict, done, or base-file mutation performed.
- NO-FURTHER-EDITS DECLARATION: implementation and tests are frozen at the recorded hashes; this worker will make no further source edits after this handover.

Command receipts
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot` — log `/tmp/fn74-task1-baseline-go.log` — terminal exit 0.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false` — log `/tmp/fn74-task1-baseline-lint.log` — terminal exit 2.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-red.log` — terminal exit 1.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-red-behavior.log` — terminal exit 1.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-green.log` — terminal exit 0.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-unit.log` — terminal exit 1.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-unit-final.log` — terminal exit 0.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-race.log` — terminal exit 0.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false` — log `/tmp/fn74-task1-lint.log` — terminal exit 2.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-unit-frozen.log` — terminal exit 0.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation` — log `/tmp/fn74-task1-race-frozen.log` — terminal exit 0.
- `TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false` — log `/tmp/fn74-task1-lint-final.log` — terminal exit 2.

Conductor verification: frozen source hashes and terminal unit/race receipts verified. Independent review SHIP, no findings: /tmp/fn74-task1-impl-review.json. Reviewer test reruns could not create build directories in read-only sandbox; prior verified runs remain the test evidence.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false
- PRs:

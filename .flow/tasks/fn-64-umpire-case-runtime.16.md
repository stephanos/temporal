---
satisfies: [R2, R4, R7, R9]
---
# fn-64-umpire-case-runtime.16 Share prepared worker outcome and value semantics

## Description
Expose the prepared worker value boundary required by task6, while keeping controller and SDK validation identical.

**Size:** M
**Files:** `tools/umpire/internal/execution/{program,dataflow,projection,values}.go`, focused execution tests, `tools/umpire/host.go`, execution README, bounded IR snapshot traversal and focused regression tests
**Touches:** [tools/umpire/internal/execution/**, tools/umpire/host.go, tools/umpire/internal/ir/catalog.go, tools/umpire/internal/ir/read_test.go, tools/umpire/internal/ir/runtime_value_test.go]

### Approach
- Extract opcode-aware outcome validation, declared-field lookup and ownership copies from `stageOutcome`; controller staging delegates to the same implementation.
- Expose a small immutable prepared outcome/value interface through the existing root plan wrappers, with the prepared runtime-work ceiling. Keep compiled types and mutable controller stores private.
- Let worker adapters supply replay-local lookups to already compiled expressions. Operations must be deterministic, bounded and free of controller-store locks, SDK calls, networking or goroutine scheduling.
- Preserve deterministic work-limit error precedence in shared protobuf snapshots: bound and charge present-field/map-key ordering before traversal, preserving presence and extension semantics. Include focused IR normal and race regression gates.
- Reject VALUE declarations on StartNexusOperation; retain Await and evaluated Finish/RespondNexus result semantics. Preserve guard and failure precedence.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/execution/projection.go:58` — existing stageOutcome authority
- `tools/umpire/internal/execution/values.go:153` — prepared runtime work and value ownership
- `tools/umpire/internal/execution/program.go:113` — immutable plan accessors
- `tools/umpire/internal/execution/dataflow.go:164` — outcome admission
- `tools/umpire/host.go` — public adapter seam
- `.flow/tmp/fn64-task6-investigation.md` — SDK constraints

### Key context
A future is an SDK handle, not an IR Value. No proto/Lean generation or second evaluator is needed.
## Acceptance
- [ ] Controller and worker callers use one outcome validator; tests compare statuses, fields, snapshots, work exhaustion and error precedence through both paths.
- [ ] Tests reject Start VALUE during preparation; Await accepts exactly its declared result type, with malformed/missing/undeclared/oversized failure cases.
- [ ] Concurrent activation lookups cannot mutate prepared data or another activation; false guards and missing guarded results retain existing semantics.
- [ ] Focused tagged execution/root tests and race tests, make fmt-imports and scoped make lint-code pass; document the worker API handoff.
## Done summary
Exposed shared prepared outcome/value operations through root plan wrappers: bounded validation and independent outcome/declared-field snapshots, cloned declared schemas, cached runtime ceiling and guarded compiled-input evaluation with activation-local lookup. Controller staging delegates the same validator; StartNexusOperation VALUE rejects at preparation while Await and evaluated Finish/RespondNexus results retain their semantics.

Baseline green: focused tagged execution/root tests, make fmt-imports and authorized scoped no-fix lint all exited 0, recorded in `.flow/tmp/fn64-task16-baseline-results.json` and `.flow/tmp/fn64-task16-baseline-style-results.json`. Final normal/race IR/execution/verification/root, formatting and scoped lint all exited 0; exact commands/environments/timestamps/logs are in `.flow/tmp/fn64-task16-final-results.json`. Initial missing-API red, parity failure, static-budget compatibility failure and first final lint failures were corrected before review. No global lint green claim: the inherited mainbase backlog remains outside scope.

Tests: `TestPreparedStartRejectsValue`, `TestPreparedOutcomeParity`, `TestPreparedInputActivationIsolation`, `TestPreparedTerminalResultsAndDeclaredTypes`, and `TestRuntimeSnapshotDeterministicFailureOrder` cover R2/R4/R7/R9 task obligations. They cover controller/worker status/field/error/work parity, missing/malformed/undeclared/oversized values, exact and one-less budgets, repeated tight-budget field/map ordering, false guards, required missing guarded reads, concurrent independent lookups and schema/outcome/field/input ownership. Existing root/import and execution tests remain green. The worker README documents validated immutable activation-local callback inputs and aggregate work ownership; no new evaluator, mutable store, locks, SDK calls, I/O or goroutines were added to the shared operations.

The conductor approved a narrow IR `catalog.go` / `runtime_value_test.go` scope extension after parity tests exposed protobuf Range nondeterminism. Runtime fields/map keys are bounded and ordered before validation/copying; static binding preserves inherited finite accounting units. This is required by deterministic task16 failure precedence, not an added runtime capability. The task scope was re-anchored after its authoritative update.

stage: impl-review - ran (codex:gpt-5.6-sol:high; SHIP first pass; 2026-09-05T02:45:49.533386Z; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.16.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: concurrent-wave - skipped(policy: shared checkout; one writer)
Tracker sync: n/a (bridge inactive).

No review findings or unaddressed requirements. The read-only reviewer attempted a focused Go command but could not create its temporary build directory; the writer's explicit final gate exits above are authoritative. The reviewed owned tree `a6ad409f851c7abfd8ca0dad0fe0a0b628d56400` matches all owned source exactly against start tree `6900676746da4d715a03d2b08948de922255aa09`; comparison exited 0 before lifecycle completion. Start HEAD `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`, actual HEAD `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`. No commits, pushes or worktrees: user owns commits, evidence commits are empty; all pre-existing staged changes were preserved.

Gate classification FULL. NO_RECEIPT: worktree dirty outside the ignore set (.plans/UMPIRE4_ORDER.md) - receipt not warrantable. No HEAD-based green receipt is claimed for uncommitted source. SDK runtime/carrier delivery/ledger work remains with tasks 17/18/6; no future live integration or cutover gates claimed.
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./tools/umpire/internal/ir ./tools/umpire/internal/execution ./tools/umpire/verification ./tools/umpire, go test -count=1 -race -tags test_dep ./tools/umpire/internal/ir ./tools/umpire/internal/execution ./tools/umpire/verification ./tools/umpire, make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, baseline: green (task16 baseline tests/style result files), All final gate command exits, environments, timestamps and logs: .flow/tmp/fn64-task16-final-results.json, Owned source matches reviewed tree: git diff --exit-code a6ad409f851c7abfd8ca0dad0fe0a0b628d56400 -- <owned_paths>; exit 0, NO_RECEIPT: worktree dirty outside the ignore set (.plans/UMPIRE4_ORDER.md) - receipt not warrantable
- PRs:
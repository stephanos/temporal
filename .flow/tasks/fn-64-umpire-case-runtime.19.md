---
satisfies: [R7, R10]
---
# fn-64-umpire-case-runtime.19 Resolve completion-review runtime and documentation gaps

## Description
Resolve the fn-64 spec-completion review findings without expanding the Case Runtime surface. Fix the Temporal worker SDK interpreter so admitted terminal VALUE outcomes for Finish and RespondNexus retain their evaluated result, and enforce each Await instruction's admitted timeout with replay-safe workflow APIs and the existing typed timeout outcome semantics. Correct the Lean compiler architecture documentation so it describes assembly/lowering accurately and assigns static admission to PrepareCase. Preserve status-only outcome behavior, existing comments, public APIs, deterministic/replay-safe execution, and all established Case Runtime boundaries.

## Acceptance
- Workflow Finish with a declared VALUE outcome records the evaluated input value; Finish without VALUE preserves its existing status-only behavior.
- Nexus-handler RespondNexus with a declared VALUE outcome records the evaluated input value; RespondNexus without VALUE preserves its existing status-only behavior.
- Await enforces its own `TimeoutMilliseconds` independently from StartNexusOperation, using replay-safe workflow APIs and producing the existing typed timeout outcome when the Await bound expires.
- Focused SDK tests cover both terminal VALUE opcodes, status-only compatibility, and distinct Start-versus-Await timeout bounds.
- `model/Umpire/ARCHITECTURE.md` states the actual Lean Case compiler assembly/lowering boundary and identifies `PrepareCase` as the static admission boundary.
- Relevant tagged worker/unit tests, `make umpire-check-regression`, `make fmt-imports`, and the task-owned lint-equivalence gate pass.
- A fresh implementation review returns SHIP; the resumed spec-completion review remains the post-task spec gate.
## Done summary
Implemented schema-aware terminal VALUE outcomes for Finish and RespondNexus while preserving status-only behavior, enforced Await's independent bound with replay-safe Temporal workflow timers, and corrected the Lean compiler/static-admission architecture boundary.

Baseline: green (`go test -count=1 -tags test_dep ./tools/umpire/temporal/worker/...`, `/tmp/fn64-task19-baseline.log`). TDD regression demonstrated both VALUE failures and Await waiting five seconds despite its one-second bound (`/tmp/fn64-task19-red.log`). Extended existing SDK tests cover terminal schemas and status-only compatibility; `TestSDKAwaitUsesItsOwnTimeout` covers Await-first timeout, Start-first timeout, and completion before either bound. Existing history-replayer coverage remains green, with no extra timer for an already-ready Nexus result.

Verification logs: `/tmp/fn64-task19-green.log`, `/tmp/fn64-task19-bounds.log`, `/tmp/fn64-task19-units-final.log`, `/tmp/fn64-task19-race-final.log`, `/tmp/fn64-task19-fmt-final.log`, `/tmp/fn64-task19-regression.log`, `/tmp/fn64-task19-lint-final.log`, `/tmp/fn64-task19-owned-lint.log`, `/tmp/fn64-task19-errortype.log`. The aggregate regression gate passed its exact inherited live-failure identity comparison; this does not claim those inherited tests pass. Broader unit tests initially hit a local compiler-header error and passed with `CC=/usr/bin/clang`. Task-relative full lint plus errortype passed with zero new issues; no claim is made that the repository's inherited lint backlog is empty. Gate receipts were unavailable because the user-owned staged tree is dirty relative to HEAD; tests actually ran and none were skipped.

Replanned the final acceptance bullet to require task implementation review first and retain spec-completion review as the post-task spec gate, removing the circular requirement that this task be complete before its own completion gate. The conductor owns the resumed completion review.

No commit or push: the user owns commits, HEAD remains `b0862eace5aa7922a46d3333ee3d839d46f4845a`. Original staged tree `799b4ed7c2bd78c59a22175837778f6b8a8db501` is preserved. Review round 1 delivered NEEDS_WORK on an empty HEAD..HEAD range; it remains recorded. The established temporary-index adapter supplies the actual immutable task-owned staged diff without changing HEAD or creating a commit.

Reviewed staged tree: `68126b72a6e2fb208131bbf47656307aa6a1a2fe`. Official review receipt: `/tmp/impl-review-receipt-fn-64-umpire-case-runtime.19.json` (SHIP). The NEEDS_WORK-to-SHIP transition corrected only review scope, not code; no non-trivial review fix required memory capture.

stage: impl-review - ran (codex:gpt-6-astra:medium; SHIP; round 1 empty-range NEEDS_WORK retained, round 2 staged-tree SHIP)
## Evidence
- Commits:
- Tests: baseline: green (go test -count=1 -tags test_dep ./tools/umpire/temporal/worker/...), TDD_RED: both terminal VALUE opcodes rejected missing values; Await elapsed 5s instead of 1s, CC=/usr/bin/clang go test -count=1 -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire/verification/... ./tools/umpire/temporal/server/... ./tools/umpire/temporal/worker/..., go test -count=1 -race -tags test_dep ./tools/umpire/temporal/worker/..., make umpire-check-regression (exact inherited live-failure identities matched), make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=b0862eace5aa7922a46d3333ee3d839d46f4845a GOLANGCI_LINT_FIX=false, go vet -tags disable_grpc_modules,test_dep -vettool=.bin/errortype -style-check=false ./tools/umpire/temporal/worker/..., REVIEW_SHIP: staged tree 68126b72a6e2fb208131bbf47656307aa6a1a2fe; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.19.json
- PRs:
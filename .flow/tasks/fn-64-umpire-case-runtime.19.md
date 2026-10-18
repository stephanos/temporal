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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
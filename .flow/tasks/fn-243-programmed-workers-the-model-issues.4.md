---
satisfies: [R5]
---
# fn-243-programmed-workers-the-model-issues.4 DESIGN.md R40 architecture checkpoint and code pointers; unheld proof deferred

## Description
After .2 and .3. fn-228.7 writes DESIGN.md's Runner and Run section after this spec, so R40 is written once in its final form there. This task checks that section states: a worker in any language follows a Model-issued program and the Run records what happened; the Shim is superseded; the proposal vocabulary is action, bootstrap, read; the hold fault; the executor kinds and where they live; and that the design note is marked design of record. It adds pointers from `programmed-workers.md` to the shipped code and goldens.
## Acceptance
- [ ] Check R40 against the actually admitted held-before-start and terminal-cancellation slice, name the exact temporary refusal and fn-243.5 obligation, and avoid claiming full unheld cancellation conformance or closing the parent spec.
- [ ] DESIGN.md states R40 in its final form with the proposal vocabulary, hold fault and executor kinds; the design note points to the code
## Done summary
Documented the shipped programmed-worker architecture as the design of record, linked its code and wire goldens, and checked DESIGN.md's R40 section against the supported held-before-start and terminal-cancellation slice. The exact unheld refusal, original counterexample, recorded non-Go gap and fn-243.5 proof obligation remain explicit. Both fn-243 and fn-228 remain open.

Tier: session (jev-unavailable(no_key)). No actual implementer model is inferred from the dispatch. Actual reviewer metadata records gpt-6.1-sol/high, the writer family; no cross-family independence is claimed.

Similar code search extended existing model/design/programmed-workers.md and model/DESIGN.md. Source cross-checks covered Reset issuance/live facade, proposal and history projection, program decoding/raw/SDK workers, Driver composition/hold controls, generic Answer retention, Run and proposal wire goldens. No executable or Model semantics changed; no source count was recomputed.

Baseline: green, actual `cd model && mise exec -- cargo test -p umpire-temporal --test program_history --test proposal` exit 0, 12+7 tests and zero filtered, captured in `.flow/tmp/fn2434-baseline.log`. No honorable receipt existed for that command at baseline. Post-edit Verify ran the same full spec Quick command once, exit 0, 12+7 tests, `.flow/tmp/fn2434-verify.log`; unittest receipt was minted at b2d4383f. The classifier conservatively returned `FULL: unmatched: model/DESIGN.md`; no docs-only skip or inherited broad green is claimed. The conductor explicitly bounded this prose checkpoint to source/link checks and the spec Quick command, with no expensive Model/live rerun. All 35 local links across the changed design note and Runner section resolve; `git diff --check` and Flow spec validation passed.

Primary receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-243-programmed-workers-the-model-issues.4.json`. Fresh correctness/contracts/integration draws all returned SHIP with zero introduced/pre-existing findings; finalization succeeded exit 0. Sidecars are `.flow/review-fanout/750a79b73d3a4e36a47cb677ad585d03/`. Reviewer Cargo attempts hit read-only build-lock restrictions and receive no execution credit. The worker's actual baseline/Verify supply the 19-test evidence. A concurrent worker Flow check prematurely replayed the in-flight refund journal; the original dispatch nevertheless completed normally and the documented finalizer recorded all actual draws without retry, force or manual state repair. Raw dispatch/finalize evidence is `.flow/tmp/fn2434-review-dispatch.json` and `.flow/tmp/fn2434-review-finalize.json`.

stage: impl-review - ran (codex:gpt-6.1-sol:high, first-round correctness/contracts/integration draws all SHIP, zero findings, merged receipt finalized)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b2d4383ff12d8a2754aa6f27bcc937119ead509c
- Tests: cd model && mise exec -- cargo test -p umpire-temporal --test program_history --test proposal, 35 local Markdown links resolve; shipped source and wire golden pointers checked, git diff --check, flowctl validate --spec fn-243-programmed-workers-the-model-issues --json
- PRs:
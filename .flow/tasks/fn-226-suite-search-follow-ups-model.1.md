---
satisfies: [R1, R2, R3]
---
# fn-226-suite-search-follow-ups-model.1 Suite search: predicate panics, Model-state budget, panic relevance

## Description
Apply the three follow-ups and the cosmetic fixes from the re-review of df992fd8e8 as listed in the spec, in model/umpire/suite (search.rs, space.rs, run.rs) with fixtures; README and Space::budget doc aligned. Verify make model-check and make model-time.

## Acceptance
- [ ] Predicate panics refused as such, naming the waypoint
- [ ] find/along budget counts Model states; large bounded pattern fixture
- [ ] Irrelevant deeper panics do not refuse an answer
- [ ] make model-check passes

## Done summary
Suite searches now count distinct Model states, catch waypoint predicate panics with their index, and retain answers when skipped action panics cannot shorten them. `Run::along_with_budget` exposes the same budget, and cached graph search preserves shortestness through nondominated labels and an explicit cut frontier.

Tier: session (jev-unavailable(no_key)).

stage: impl-review - ran [2026-10-20T15:59:36Z..2026-10-20T16:02:00Z]

The fresh-context Codex review finalized SHIP with no findings across correctness, contracts and integration. The explicit reviewer was codex:gpt-6.1-sol:high, from the writer's family. Receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-226-suite-search-follow-ups-model.1.json; fanout and finalize outputs are .flow/tmp/fn226/review-fanout.json and review-finalize.json. Reviewers' fresh Cargo commands were blocked by read-only build locks, so they supply code-review evidence and the worker gates supply execution evidence.

Acceptance evidence:
- R1. The space_find fixture pins start, later-step and non-string waypoint panics as distinct refusals.
- R2. The search_budget fixture pins a billion-step unreachable gap, a twelve-step witness over three Model states, along's budget override, cut-frontier ties, omitted starts, and queued-label dominance. Existing space_find later-heating behavior remains pinned. Checker rows use their enforced source ordering for binary slice lookup, avoiding an O(VE) adjacency scan without adding public API.
- R3. The space_skipped fixture covers no witness and panic depths below, equal to and above answers across find, refuted and along.
- Cosmetics. Completed labels return without successors; starts are deduplicated in declaration order; the stale index-zero comment was removed with the replaced wrapper.

Baseline: green. Both make model-check and make model-time exited 0 before edits. The baseline timing gate took 59.288s, exceeding its 30s soft target; its inner loop took 6.105s. Logs are .flow/tmp/fn226/baseline-check.log and baseline-time.log.

Verification before review: focused suite/checker tests and clippy passed; make model-check and make model-time exited 0. The timing gate measured 11.409s with a 1.786s inner loop. Logs are .flow/tmp/fn226/focused-checker.log, clippy.log, pre-review-check.log and pre-review-time.log. Gate receipts were written after the implementation commit; the first attempt before committing was declined because source files were dirty.

Final verification after SHIP: gate classify required full verification. make model-check and make model-time each exited 0 (.flow/tmp/fn226/final-check.log and final-time.log). Final timing measured a 12.732s gate and 4.923s inner loop. Both green receipts were written at baea1f875ff18a2ea251b0ee039982380c412daf; no gates were skipped. The final fixture tree has no pending snapshots, and git diff --check passed.

Defect route:
- prior fixes: git history across local refs contains the original df992fd8e8 implementation and no later competing suite-search fix. GitHub PR and issue checks were unchecked because gh returned HTTP 401 Bad credentials. Relevant search identity and lazy-path memory entries were read.
- diagnosis: the red fixture reproduced a twelve-step witness consuming the three-state product budget. Predicate failures escaped starts or became action failures; unconditional skipped-edge refusal erased shallow witnesses. The retained graph, local judge catch and answer-length relevance rule address these mechanisms.
- introduced by: df992fd8e8 fix(suite): fn-213 review follow-ups. Bisect skipped because the report names no known-good revision.
- base: the fixture suite failed before production edits at 4265c5b7f8; .flow/tmp/fn226/regression-red.log records the budget refusal. The budget override compile control failed because along_with_budget was absent (.flow/tmp/fn226/budget-api-red.log). Head controls passed in focused-checker.log.
- live: no live surface; these are library fixtures.

The initial failing-test commit also included the conductor's pre-existing MILESTONES.md status edit through the required catch-all staging. The implementation commit deletes the pending snapshots produced by the red run; the final fixture tree contains none.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4265c5b7f89728a92f428205e79e1a95e9e0d6ca, baea1f875ff18a2ea251b0ee039982380c412daf
- Tests: cd model && mise exec -- cargo test -p umpire-suite -p umpire-checker, cd model && mise exec -- cargo clippy -p umpire-suite --all-targets -- -D warnings, make model-check, make model-time, git diff --check
- PRs:
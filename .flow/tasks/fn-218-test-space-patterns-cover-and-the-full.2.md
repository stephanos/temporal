---
satisfies: [R1, R2, R3, R5]
---
# fn-218-test-space-patterns-cover-and-the-full.2 cover and sample in umpire-suite with expansion and coverage goldens

## Description
Implement full-grammar find/Run::along plus cover(&space, Cases(n), path) and seeded finite random-walk sample in umpire-suite. Cases mismatch must fail before Model stepping and list alternatives; count over budget and unwitnessed alternatives fail explicitly. Reuse one explored Model graph, retaining exact edge outcomes/facts and per-alternative shortestness. Cover fails on a cut; along rejects alternatives before search. Covered exposes runs/len/all/coverage/expansion. Expansion prints count, choices, every action with outcome and final state; coverage shows touched/untouched declared actions, outcomes, faults and claims by typed/registration identity, honestly labeling undeclared metadata. Sample must implement actual bounded seeded random walks in the admitted pattern, not just pick cover shortest witnesses, and report seed/experiment size. Register expansion/coverage golden kinds in testing and pin mismatch, impossible alternative, count/permutation limits, cut, predicate failures and reproducible seed controls. **Touches:** `model/umpire/suite/**`, `model/umpire/testing/**`. **Quick:** `cd model && mise exec -- cargo test -p umpire-suite -p umpire-testing`; full gates `make model-check` and `make model-time`.

Leading structural action patterns must preserve the existing action-first actor-to-state inference before later untyped state predicates. Repair the confirmed integration E0282 without requiring an explicit state prefix, duplicating/moving expressions or weakening zero-repeat behavior. The additional implementation surface is narrowly `model/umpire/dsl/src/path_build.rs`, `model/umpire/dsl/macros/src/path.rs` and DSL regression fixtures/tests; do not otherwise reopen the grammar task. Include focused DSL/macros tests for this repair in addition to the suite/testing Quick commands.

**Additional Touches:** `model/umpire/dsl/src/path_build.rs`, `model/umpire/dsl/macros/src/path.rs`, `model/umpire/dsl/tests/**`.

## Acceptance
- [ ] Action-first structural alternatives followed by bare state closures compile through actual find/cover consumers; expression order/single evaluation, zero repeats and legacy/manual-world routes stay intact.
- [ ] Model-state, alternative-admission and finite random-walk budgets are independently controlled; a one-state graph covers two alternatives with Model-state budget one. Sampling reports the walk bound and explicitly refuses a selected alternative that cannot finish within it.
- [ ] Cases mismatch is first, with count and alternatives; impossible alternatives and over-budget/permutation counts refuse without vacuity.
- [ ] One shortest witness per alternative uses one retained Model graph, with explicit cut handling; along rejects multiple alternatives.
- [ ] Typed coverage and expansion goldens account for actual outcomes/actions/faults/claims and honestly report undeclared catalogs.
- [ ] Sample implements bounded pattern-admitted random walks reproducibly by seed and reports its finite experiment.
- [ ] Focused suites, make model-check and make model-time pass.

## Done summary
# fn218.2 handover

Implemented full-grammar shortest find/along, cover, and genuine finite seeded pattern walks over one retained Model graph. Added typed witness coverage, expansion/coverage golden consumers, independent state/alternative/walk limits, and the narrowly authorized leading-pattern inference integration repair.

Tier: session (jev-unavailable(no_key))

Baseline: green; focused suite/testing, default model-check, actual required TLC model-check, and timing passed before edits. See baseline.md; the first focused invocation had an unobserved completion and was recorded inconclusive before the properly observed baseline retry.

R1 controls: path_leading_patterns.rs, manual-world controls in path_language_patterns.rs, and actual find/along consumers in full_patterns.rs. R2 controls: full_patterns.rs and alternative_search.rs cover admission ordering, independent budgets, globally shortest alternatives, cut/frontier behavior, along alternatives and skipped-step relevance. R3 controls: retained.rs, coverage unit tests, fixture test_space, and testing suite_consumers.rs cover retained non-Clone outcomes/facts, registration/catalog identity, honest metadata, and mixed CHECKER/SUITE output ownership. R5 controls: bounded seeded loop experiments, reproducibility, seed variation and explicit horizon refusal; sampled runs are not selected cover-shortest witnesses.

Authoring routes changed: full-grammar find/Run::along, cover/sample, Space alternative_budget/walk_bound, and expansion/coverage expected kinds. API and task-.3 golden composition guidance are in api-handoff.md. The conductor owns the feature map, MILESTONES, real-model examples and subsequent tasks. Its inherited MILESTONES/task clarification edits were included unchanged by required catch-all staging.

Independent quality audits and final reference probe: quality-audit.md, quality-audit-followup.md and quality-audit-final.md retain findings/dispositions. matching-probe-result.md records 25,000 actual public-find comparisons against an independent exact-age epsilon/product reference. No coverage, sampling, parsing or panic claim is inferred from that matcher probe.

The fixture macro's named fourth `extra = Files` expression retains the top-level negative-claim hook and merges without output replacement. registry-fixture-red.log and registry-fixture-named-{green,clippy}.log record runtime controls. ../fn218-3-hook-proof-named-evidence.md records independent actual Lift::source/Tla::emit comparisons of Kettle's original hook, `extra = suite_files()?`, and a claim-shaped helper RHS: all retain the identical single negative claim and emitted metadata/configuration. Named syntax isolates arbitrary helper call shapes from the lifter's supplemental-claim parser.

Formal review first fanout: correctness/contracts/integration ran as gpt-6.1-sol at high, evidenced by review-fanout.json. Their duplicate contextual-manual-world findings merged into issue 1; structural along's missing skipped-step refusal is issue 2. The macro now carries deferred actor-owner hints, lets concrete state and authored predicate constraints win, and resolves ownership only where needed. Direct/nested successive-predicate controls and source-order controls pass. Along checks skipped steps only for negative Unbridged searches or relevant successful witnesses, preserving predicate/resource errors. See leading-dsl-handoff.md, along-skipped/along-precedence red/green logs, and quality-audit-run-fixture-delta.md/quality-audit-carrier-delta.md; final independent rechecks have no residual findings.

Review artifacts are retained under .flow/review-fanout/4cb743418067413da62b798da8695330 and `/tmp/impl-review-receipt-8f37faba39e2-fn-218-test-space-patterns-cover-and-the-full.2.json`. The single resumed review returned SHIP with both findings fixed and no remaining blocking findings. review-round2.json records actual reviewer model gpt-6.1-sol at high; the writer's executed model was not exposed and is not inferred from the requested implementer.

Committed-fix pre-review verification at 2f62da777c: suite/testing Quick and strict all-target Clippy pass; complete DSL/macros tests and strict Clippy pass. Full default model-check and separately required actual TLC model-check exit zero. Timing exits zero: gate 12.697s (target 30s), inner 2.122s (target 10s). Logs: review2-focused-final.log, review2-clippy-final.log, leading-carrier-complete-verification.log, and review2-model-{check,check-tlc,time}.log. All commands joined and the three full-gate receipts were minted at this clean HEAD.

Final Verify classified the cumulative task diff FULL. All three exact gate checks honored 2f62da77 after the review: the complete path history since that verified commit contains only .flow/ memory/review records. Formatting and strict all-target Clippy for suite/testing/DSL/macros ran again and passed (post-review-style.log). No executable edits followed the full gates or final 25,000-case reference probe. The nontrivial review fix updated the existing macro-inference memory entry, preserving predecessor lessons rather than creating a duplicate.

GATE_SKIPPED:model-check:green-receipt 2f62da77

GATE_SKIPPED:model-check-tlc:green-receipt 2f62da77

GATE_SKIPPED:model-time:green-receipt 2f62da77

stage: impl-review - ran [2026-10-20 18:07:15 UTC..2026-10-20 18:44:36 UTC] (codex; evidenced reviewer gpt-6.1-sol at high; SHIP after first fanout and one bounded single re-review)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2ad24d881929b485e88f07e19cb9ddf30ee58f33, 2f62da777cb811197b3b7efe45a8925317eba6d7, b3ba2c4b2423ea2af82a585cca5d6ff8a8d47d2b
- Tests: cd model && mise exec -- cargo test -p umpire-suite -p umpire-testing, cd model && mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros, cd model && mise exec -- cargo fmt --all -- --check, cd model && mise exec -- cargo clippy -p umpire-suite -p umpire-testing -p umpire-dsl -p umpire-dsl-macros --all-targets -- -D warnings, make model-check, UMPIRE_BACKENDS=require make model-check, make model-time, GATE_SKIPPED:model-check:green-receipt 2f62da77, GATE_SKIPPED:model-check-tlc:green-receipt 2f62da77, GATE_SKIPPED:model-time:green-receipt 2f62da77
- PRs:
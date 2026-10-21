# Task 1 verification and review context

Task: fn-231-declarative-realizations-the.1. Full task base: b9f26782e39f815d740ea4b89925a723303c861d. Preformal implementation checkpoint: 22482ab2cc5a4cc592d5bbb52cb483b37668add4. The first formal round reviewed through 653fd40331fb5740940d6f91df8ad3c234ab9c4e and returned NEEDS_WORK with two introduced findings. Round 2 reviewed their committed fix f8df139c1cb23586972f4841ffb9358942cf74a1 and returned SHIP, both fixed and no blockers. All three final full gates subsequently returned 0. Source remains unchanged since the reviewed fix.

## Scope and preservation

The implementation adds proper realization attributes, Step and Reply authoring types, descriptor-admitted field classifications and explicit state projection. The actual activity binding uses the generated dispatch. Catalog parsing, descriptor admission and deterministic input discovery are shared; API and macro freshness retain every visited directory, including empty directories.

Model source, its runtime assertions and all existing activity goldens remain unchanged. The approved phase-helper source exception replaces wildcards without changing status-first/run-second conversion, error messages, precedence or mappings. Independent controls retain all 63 status/run pairs and unknown enum precedence. A compile control includes the actual phase source with added upstream variants and pins three non-exhaustiveness errors across both enum families.

No ABI/protocol, generic runtime, Go SDK panic path, third-party dependency or version was changed. Callback/reply messages remain partial SDK messages. Unknown Cancel codes retain raw i32 behavior and refusal order. Complete/Fail literals, external-payload clearing, Current/terminal/unexplained projection behavior and single reply-envelope packing are covered by native controls and unchanged expansion output.

Root-owned commit 1b71756bcdd5e80d533ec329b464d064f14b9bea only updates the fn230 retirement artifact. It is inside the full task range but is not fn231 implementation. Foreign MILESTONES.md changes and untracked fn233 plans/tasks are excluded from this task's staging.

## Baseline before edits

At the full task base, selected native realization/helper/admission controls, the complete Temporal fixture/cover/boundary package, generic guest build, actual live slice, actual live gaps/privacy and default model-check all returned 0. The gap baseline collected and passed 2336 unique subtests. Logs are retained under .flow/tmp/fn2311-baseline-*.log and summarized in fn2311-baseline.md.

The first native launch failed before collection because of a nested relative cwd. Its corrected absolute invocation established the baseline. That first observation is inconclusive, not a passing test. No baseline handoff or fabricated receipt was used. The default full-gate reuse check refused because MILESTONES.md was dirty; the command actually ran.

## Focused regression evidence

- Shared schema graph and reverse RPC edges have actual missing-behavior RED followed by same-control GREEN. Unsupported fields use genuine optional/oneof presence, with opaque paths stopping traversal.
- The original macro lane retains actual parser/expansion/compile-refusal REDs and ordinary GREEN runs in .flow/tmp/fn2311-macro-lane. Existing workspace dependencies provide trybuild, insta and prettyplease. No existing golden was fitted.
- Phase upstream-variant control failed because the old helper accepted the additions, then passed with the intended compiler errors. Existing pair/unknown-precedence controls were unchanged.
- Direct aliases, renamed variants and external reexports initially compiled when refusal was expected. Final ordinary UI runs contain ten passing cases. The observed source-scope compiler/fallback panic was not accepted as a pin; its replacement is the intended located refusal.
- Wrapper positions preserve Ok/Err and Some/None identity. Valid Err fallback, local Model types and byte-policy fallbacks remain allowed. Unknown external, conditional and unsupported alias coverage is refused, not silently classified Other.
- Fresh correctness findings each had a separately collected RED: concrete wrapper aliases, dereferenced generated enums and deferred initialization. Six tests collected with three intended failures and three passes, followed by six passes. Related inferred-payload and binding-scope REDs are retained separately.
- The custom-Deref addendum has an actual one-failure/five-pass RED and six-test GREEN. Only proved built-in references retain dereference identity; overloaded results are unresolved. Borrowed wrapper matching is normalized separately without rewriting expressions.

The raw per-axis audit history, addendum and resolutions are retained verbatim in task-1-quality-audits.md. These audits were read-only. Their cited runtime results were not executions by the auditors. Preformal quality-audit resolution counts were zero for correctness and standards. The subsequent formal round found the two additional gaps recorded below. Dispatches were pinned to gpt-6.1-sol at high; this is same-family evidence, not a cross-family independence claim.

## Preformal post-edit Quick observations at 22482ab

All owner commands use CARGO_BUILD_JOBS=4, the serially leased fn2301-unwind Cargo target and Go cache. Every started command is observed through its terminal handle.

| Command | Observation | Saved log |
| --- | --- | --- |
| cargo test -p umpire-temporal-api -p umpire-temporal-annotate-syntax -p umpire-temporal-realization-macros -p umpire-temporal -p umpire-temporal-tests --all-features | exit 0; real tests collected, API freshness, ordinary UI, snapshots, runtime/helpers, cover, two boundary and seven fixture controls | fn2311-final-quick-native.log |
| cargo clippy with the same five packages, --all-targets --all-features -- -D warnings | exit 0 | fn2311-final-quick-clippy.log |
| cargo fmt --all --check | exit 0 | fn2311-final-quick-fmt.log |
| make umpire-guest | exit 0; actual cmp against captured eddb guest and leased output both 0, SHA256 unchanged | fn2311-final-quick-guest.log |

Cargo commands run through mise in model/. The generic make target is the admitted guest builder. Source-only changes after the last live run require a regenerated guest byte comparison; changed bytes require another actual live run.

## Exact live evidence

The refreshed eddb guest is 3329990 bytes with SHA256 10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06. Its actual renewed slice returned 0, package 1.115s. Its complete renewed gaps run returned 0, package 351.609s, with the unchanged inner 20-minute limit and an outer 1800-second bound.

Regular-log inventory comparison found 2336 unique RUN/PASS pairs, baseline inventory missing 0, extra 0, invalid/duplicate pairs 0, and the privacy control passed. The generated report compared exactly with the unchanged golden and retained report copies, SHA256 a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0. Logs are fn2311-combined-{slice,gaps}.log. Earlier fe98 live logs and report copies remain untouched and are not silently reassigned to another asset.

Both the post-05f448d rebuild and the final post-22482ab rebuild returned 0 and compared exactly with the captured eddb guest and leased build output. SHA256 remains 10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06. The renewed eddb live proof therefore covers the exact final guest bytes. Exact captured guest copies are retained for actual cmp, not just inferred semantic equivalence.

## Formal review risk steering

The checker is bounded source analysis, not compiler type inference. A classification may be preserved across an operation only when its semantics/type identity are proved. Familiar method or operator spelling alone cannot establish identity. This focus was supplied before round 1. All three formal draws subsequently found unproved method-result propagation; correctness also found rest-pattern coverage gaps. Both valid introduced groups were retained. All distinct map_err, custom-method and tuple-unwrap examples remain controls despite deduplication. Raw reviews, merge rationale and round-1 outcome are retained in task-1-formal-review.md and the actual receipt snapshot.

The formal-fix RED collected eight tests, with both aggregated finding controls failing and the original six passing. The retained fn2311-formal-r1-red.log lists all three method examples, the explicit cast-target check and all five rest examples. It does not stop at the first case. The same eight controls then passed in fn2311-formal-r1-green.log. A narrow sibling control for the identical ExprTry wrapper-shape transformation subsequently failed as intended: fn2311-formal-r1-try-red.log collected eight tests, with the method control failing only on that case and the other seven passing. Final ordinary native verification passed all eight, ten UI cases and unchanged snapshots; strict clippy, formatting and diff checks also returned 0. Logs are fn2311-formal-r1-final-{native,clippy,fmt}.log. The child returned its frozen source/cache lease with every command terminal; owner integrated Quick and guest verification follow before re-review. No second review or full final-gate result is claimed at this entry.

Unproved method and `?` result shapes are Unknown, rather than inheriting the receiver's shape. Casts use their explicit target type, retaining reference identity while permitting raw scalar casts. Tuple/variant coverage carries proved arity so rests cover only the actual omitted positions and suffixes align correctly. Unknown rest arity is not guessed. The controls retain explicit generated prefixes/suffixes, non-enum rest payloads, raw Result error fallback and existing activity byte-policy behavior.

## Post-fix integrated Quick before round 2

Owner five-package all-feature native tests, strict all-target/all-feature clippy and workspace formatting each returned terminal 0, in fn2311-round2-quick-{native,clippy,fmt}.log. Native collection includes all eight syntax controls, ten ordinary UI cases, unchanged expansion, fourteen realization and five helper controls, added-variant phase refusal, API inputs, cover/fixtures and both boundary controls. The nine preserved Model/test/golden hashes still match; the approved phase source exception remains separately controlled.

The admitted generic guest rebuild returned 0 after 30.06s in fn2311-round2-quick-guest.log. Actual cmp against both the captured eddb guest and the leased target output returned 0. SHA256 remains 10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06, so the retained renewed slice, complete 2336-case gaps, privacy and byte-identical report evidence still applies to the exact rebuilt guest. No redundant live rerun or full-gate receipt was fabricated. The fix is ready for the bounded single round-2 review, not yet SHIP.

## Final Verify after SHIP

The cumulative base diff classified full, not docs-only. All commands actually ran after SHIP with the authorized serial cache lease, unchanged limits and captured regular logs:

| Full command | Terminal result | Log |
| --- | --- | --- |
| make model-check | 0; default backend mode, complete gate and host checks | fn2311-final-model-check.log |
| UMPIRE_BACKENDS=require make model-check | 0; real required TLC agreement/semantic suites and complete gate | fn2311-final-model-check-required.log |
| make model-time | 0; instrumented complete gate and edited-checker inner loop | fn2311-final-model-time.log |

Timing measured total 85.714s versus the unchanged 30s soft target, and inner loop 1.634s versus 10s. Dominant steps were Temporal guest freshness 33.172s, tests 24.554s and generic guest controls 14.474s. The predecessor fn230 final measurement was 85.295s with inner 1.648s (task-2-verification.md); this remains the existing full-gate soft-target overrun, not a newly green latency claim. No budget was extended or note suppressed.

Gate receipt attempts used unittest/default, required/required-backend and timing/model-time identities. All returned 1 with written:false because MILESTONES.md is dirty outside the ignore set. Their exact outputs are fn2311-final-{model-check,model-check-required,model-time}-receipt.json. Actual passes stand independently; no gate skip or reusable receipt was fabricated. The foreign dirt was never stashed or hidden.

Phase 4.5 captured the nontrivial review fix after a narrow no-match search and low-score overlap check as bug/integration/prove-result-shape-before-propagating-2026-10-21. Original audit reports and first-round reviews remain retained. Final source diff is empty against reviewed f8df139, nine preservation hashes match, and guest cmp against the actual renewed-live capture remains 0/SHA1072 unchanged. Every started command is terminal.

## Successor ownership

Task .2 owns the full realization.snap.txt consumer/coverage golden and DESIGN changes. fn232 owns cross-platform proof. Seven broad legacy umbrella failures are inherited conductor evidence and remain outside this task; these focused/model gates do not claim those failures green. No parent completion, milestone edit or next-task admission is performed by this worker.

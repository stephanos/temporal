---
satisfies: [R1, R3]
---
# fn-218-test-space-patterns-cover-and-the-full.1 Full path! grammar: actions, inputs, outcomes, alternatives, repetition, permutations; alternative enumeration

## Description
Extend Path/path! in umpire-dsl with actions, structured inputs (ranges, sets, exact values, `_`/`..`), actor wildcard, typed outcomes and outcome alternatives, alternatives, exact/bounded repetition, unordered permutations, and exclusion gaps. Enumerate every authored finite alternative and count before search with checked arithmetic and an overridable default permutation limit of six. Preserve state-only paths, closure inference and two-state progress claims; malformed grammar and test-only claim forms have pinned trybuild diagnostics. Implement the normal Action payload surface using required finite #[inputs(...)] field domains, full-value equality/hash/clone/formatting, and typed actor projection; never narrow Model catalogs to requested paths. Supply the supporting typed outcome/fault catalog surface for R3: generated stock/unit outcomes and explicitly marked fault actors, borrowed classifiers, and honest undeclared metadata for manual Machines. The spec's Implementation decisions and .flow/tmp/fn218-design-handoff.md explain the grounded design; API details may improve while preserving the requirements. **Touches:** `model/umpire/dsl/**`. **Quick:** `cd model && mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros`; full gates `make model-check` and `make model-time`.

## Acceptance
- [ ] All requested grammar forms construct typed paths and enumerate finite alternatives in declared order, with checked counts and limit controls.
- [ ] Structured derived actions have explicit complete finite input domains, payload-sensitive identity/formatting and typed actor conversion.
- [ ] Outcome/fault metadata supports later complete coverage without imposing Clone/Eq on generic outcomes or treating missing metadata as declared empty.
- [ ] Malformed grammar and every test-only form in a progress claim are pinned by trybuild; existing state-only paths and claim behavior remain valid.
- [ ] Focused tests, make model-check and make model-time pass.

## Done summary
Implemented the full typed finite `path!` grammar, payload action domains and complete borrowed outcome/fault metadata, preserving state-only inference and two-state progress claims. Suite matching/cover/sample and real-model demonstrations remain assigned to tasks .2/.3; the stable typed consumer contract is in `api-handoff.md` beside this summary.

Tier: session (jev-unavailable(no_key)). Two disjoint in-host implementation delegates joined before integration; no executed writer-model metadata was exposed, so no actual writer model is claimed.

Baseline: green for focused DSL/macros suites, `make model-check`, and `make model-time` (10.732s gate / 1.712s inner). Final 84ce1e73 verification is green. Canonical focused suites passed (`final-focused.log`), default full gate passed (`final-model-check.log`), required TLC full gate passed (`final-model-check-tlc.log`), and timing passed (`final-model-time.log`, 12.016s gate / 3.616s inner). TLC-required mode supplies actual execution/agreement evidence; the timing gate's normal backend skip is not claimed as TLC evidence.

R1 controls: `path_language_patterns.rs` covers all grammar kinds, ordered finite choices, positional duplicates, fresh repeat choices, borrowed outcomes, closure-first/action-first/qualified-state inference, manual typed adaptation, and resource boundaries. `path_domains.rs` pins wide integer domains/overflow, legacy clone-before-budget refusal, accurate repeated-domain labels and oversized empty-alternative refusal before labeling. Trybuild pins malformed grammar, test-only claim forms, state-with-outcome refusal, missing ordinary claim gaps, duplicate structured fields including raw aliases, and public-actor/private-state visibility. Action fixture controls pin finite declaration requirements and payload identity/formatting. R3 controls: `catalog.rs` pins all stock/unit outcomes, explicit fault independence, complete actor projections/catalogs, borrowed non-Clone outcomes and undeclared versus declared-empty metadata.

Independent root audits found three introduced issues (duplicate fields, empty-alternative storage, missing claim gaps), each reproduced red then repaired in c9b52fb5. Both fresh follow-ups report zero remaining findings (`quality-audit-followup.md`); initial reports and bounded actual runtime probes remain retained. The root's separate Rust reference enumerator agrees on all 19,995 admitted small patterns (`enumeration-probe-result.md`). Stock catalog ownership was also moved beside `Outcome`, with generated delegation and two intentionally updated expansion snapshots. Earlier snapshots intentionally changed for the new generated action/catalog/projection surface and renamed macro expansion origin; no assertions were weakened.

Formal fanout merged eight introduced findings. de4e35e8 fixes source-order and single-evaluation semantics, shares owned payload domains across outcome alternatives, preserves qualified Rust state structs, infers nested state closures through a typed construction context, validates wildcard field names, preserves const-generic declarations and separates authored field names from finalization. `path_expression_semantics.rs`, `action_inputs_builders.rs` and the unknown-field UI fixture pin these repairs. Red logs include the `[0,2,1]` waypoint reversal, twice-evaluated repeat count, E0382 owned payload move, E0282 nested closure inference and accidental unknown-field compilation. Both independent delta audits cleared all eight fixes (`quality-audit-formal-fixes.md`). 84ce1e73 removes the unused superseded single-outcome helper. The path expansion and state-outcome error snapshots intentionally reflect the shared-domain helper/context shape.

Authoring routes changed: `path!`, `#[derive(Action)]` field `#[inputs(...)]` / actor `#[fault]`, and `#[machine]` generated metadata/actor association. The conductor owns any feature-map updates. Source changes remain under `model/umpire/dsl/**`; the inherited root MILESTONES in-progress edit was included unchanged in the implementation commit.

The formal review returned SHIP on 84ce1e73 after one three-axis fanout and one resumed single review. All eight findings have status fixed, and no unaddressed R-IDs remain for task .1. Review execution metadata confirms `gpt-6.1-sol` at high effort; this names the reviewer only. The read-only reviewer could not acquire Cargo's build lock and used cached binaries/compiler probes. The writer-owned full gates above supply fresh suite evidence. Full review output is in `review-fanout.json`, `review-finalize.json` and `review-rereview.json`; the live receipt is `/tmp/impl-review-receipt-8f37faba39e2-fn-218-test-space-patterns-cover-and-the-full.1.json`.

The nontrivial NEEDS_WORK-to-SHIP repair was captured in `bug/integration/preserve-rust-expression-and-inference-2026-10-20`, linked by the memory tool to the existing macro-invocation lesson.

GATE_SKIPPED:model-check:green-receipt 84ce1e73 - baseline reused from prior post-gate pass
GATE_SKIPPED:model-check-tlc:green-receipt 84ce1e73 - baseline reused from prior post-gate pass
GATE_SKIPPED:model-time:green-receipt 84ce1e73 - baseline reused from prior post-gate pass

stage: impl-review - ran [2026-10-20T17:02:56Z..2026-10-20T17:20:40Z] (SHIP; codex:gpt-6.1-sol:high; two rounds)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e5d3268650f589d5fe70354ff6635c103eac0187, c9b52fb541a620585db9aaf80fd0130401fe36de, de4e35e8dd6bb61ebac8271a2ad0854de87d4a26, 84ce1e73201ea5e9cf1e4d9ff58abb466540ef02
- Tests: cd model && mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros, make model-check, UMPIRE_BACKENDS=require make model-check, make model-time, cd model && cargo clippy -p umpire-dsl -p umpire-dsl-macros --all-targets -- -D warnings, GATE_SKIPPED:model-check:green-receipt 84ce1e73 - baseline reused from prior post-gate pass, GATE_SKIPPED:model-check-tlc:green-receipt 84ce1e73 - baseline reused from prior post-gate pass, GATE_SKIPPED:model-time:green-receipt 84ce1e73 - baseline reused from prior post-gate pass
- PRs:
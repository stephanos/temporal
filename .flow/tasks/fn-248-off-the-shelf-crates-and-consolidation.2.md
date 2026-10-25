---
satisfies: [R2]
---
# fn-248-off-the-shelf-crates-and-consolidation.2 One copy of each repeated helper

## Description

**Size:** M
**Files:** model/umpire/backends/native/**, model/umpire/backends/apalache/src/replay.rs, model/umpire/author/suite/**, model/umpire/catalog/**, model/umpire/front/**, model/umpire/dev/testing/**, model/umpire/protocol/runtime/tests/**, model/umpire/reference/kettle/probes.rs, model/temporal/realization/src/probes/generic.rs, model/temporal/feature/activity/**, model/umpire/dev/fixtures/fixtures/world_fairness/tests.rs
**Touches:** [model/umpire/backends/native/**, model/umpire/backends/apalache/src/replay.rs, model/umpire/author/suite/**, model/umpire/catalog/**, model/umpire/front/**, model/umpire/dev/testing/**, model/umpire/protocol/runtime/tests/**, model/umpire/reference/kettle/probes.rs, model/temporal/realization/src/probes/generic.rs, model/temporal/feature/activity/**, model/umpire/dev/fixtures/fixtures/world_fairness/tests.rs, model/Cargo.lock, .plans/UMPIRE_MODULES.md]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Collapse repeated helpers to one copy each.

- `panic_message(payload: &(dyn Any + Send)) -> &str` in umpire-checker (`model/umpire/backends/native`), replacing the downcast chains in `model/umpire/author/suite/src/{coverage.rs ~77, pattern_search.rs ~102, search.rs ~121}`, `model/umpire/backends/native/src/explore.rs` (~261), `model/umpire/backends/apalache/src/replay.rs` (~556). One fallback text; update assertions on the old texts.
- One SplitMix64: `model/umpire/catalog/src/value.rs` (~8-23) and `model/umpire/author/suite/src/pattern_search.rs` (~46-62). Use one deterministic seeded implementation without requiring the old sequence or `% len` index; place it in a crate both already depend on, or justify a new edge.
- One applicability mapping returning the prost `ProbeApplicability` in umpire-catalog, used by `model/umpire/reference/kettle/probes.rs` (~31-38, magic `0..3` today) and `model/temporal/realization/src/probes/generic.rs` (~336-357, including `matches!(.., 0 | 3)`).
- One `Errors` accumulator (`push`, `finish(value) -> syn::Result<T>`) in umpire-front, replacing `model/umpire/front/src/declare/action.rs` (~82-93), `declare/fields.rs` (~27-31), `declare/mod.rs` (~700-711).
- `replace_once`, `replace_in`, `replace_code` in umpire-testing, replacing copies in `model/temporal/feature/activity/policy_checks.rs` (~365, ~594), `proof_checks.rs` (~17), `model/umpire/dev/fixtures/fixtures/world_fairness/tests.rs` (~36).
- One hex decoder in `model/umpire/protocol/runtime/tests/support/mod.rs`, replacing `tests/wire.rs` (~24-32), `core_protocol.rs` (~61), `proposal_wire.rs` (~62).

### Placement and error controls

Place SplitMix64 in the existing lower checker dependency chain, which both suite and catalog already reach; add an explicit local dependency only if needed and record it. No cycle or new general utility crate is required. Test fixed-seed reproducibility, index/domain boundaries and semantic witness replay independently of snapshot text. Preserve panic downcast behavior for &str/String/non-string payloads, applicability variants/unknown wire values, accumulated error ordering and edit-helper missing/duplicate match failures.

## Acceptance
- [ ] Each helper exists once; no former copy remains (grep evidence in the task summary).
- [ ] A fixed seed reproduces the new outputs; generated values stay within their declared domains and emitted witnesses replay and satisfy their recorded claim. Review and regenerate pattern-search/value-generation goldens; matching old sequences is not required.
- [ ] Focused checks below pass; joined full validation belongs to the final integration task.


## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-checker -p umpire-suite -p umpire-catalog -p umpire-front --lib --locked
cd model && mise exec -- cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
Model authors and probe consumers now share one implementation of each helper named by R2. The checker owns panic payload decoding and SplitMix64; catalog owns typed probe applicability, front owns ordered syntax-error accumulation, testing owns source edits, and runtime test support owns hex decoding.

Tier: session (jev-unavailable(no_key))
Status: in_progress. The conductor owns review, integration and lifecycle state.
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

The shared stream retains the existing algorithm. Fixed-seed stream/index controls, generated-value domain checks, and seeded witness replay all pass. The strengthened seeded-walk control replays each recorded choice through World::step and checks the requested final state/action. Suite fixture outputs, catalog goldens and Kettle coverage still match their current baselines; this lane produced no snapshot or wasm diff.

The panic controls cover &str, String and non-string payloads. Applicability controls cover every variant and refuse unknown prost values. Front controls retain combined-error ordering and the success value. Source-edit controls retain Unicode/surrounding text and refuse missing/duplicate matches, missing/duplicate delimiters and empty code anchors. The original replace_in accepts duplicate region ends; original-source-edit-red.log records the expected exit 101 against base-commit helper bodies, while verify-source-edit.log records two passing controls. The initial crate-level Errors test also failed before implementation. Literal protocol vectors remain independent byte oracles; malformed hex controls retain refusal.

Both mandatory focused commands passed before edits (22 library tests and 6 wire tests) and after consolidation (27 library tests and 7 wire tests). The affected-scope checks pass, including generation/black-box controls, suite fixtures, Kettle probes, generic probe metadata, world-fairness source mutants, activity global-budget source mutants, all 15 catalog tests, formatting and selected Clippy across the changed crates/test targets. validation.json records exact commands, exits, counts, logs and source hashes. The first Clippy run refused a single-variant wildcard; the final run passes with the explicit Applicable arm.

The authorized scope additions are model/Cargo.lock (one existing local dependency edge) and .plans/UMPIRE_MODULES.md (catalog may use checker for the shared stream). dependency-check.json verifies the resolved normal/build dependency graph is acyclic. The direct catalog-to-checker edge enables no new package versions or named package-feature selections in the Kettle/Temporal guest closures (79 package versions and 136 feature pairs before/after). The conductor supplied the exact pre-integration feature receipt; both trees and delta receipts are retained here.

One-owner grep evidence is in helper-owners.txt, produced by rg over model/umpire and model/temporal for fn panic_message, the SplitMix64 mixing constant, fn probe_applicability, struct Errors, fn replace_(once|in|code) and fn decode_hex. It reports one owner for each named helper. All former call sites import or call those owners, including the additional panic chain in testing/interpreted.rs. No new utility crate or third-party dependency was added.

The additive testing/lib.rs export overlaps task1 and needs the central merge already coordinated with that lane. Task7 must rerun affected checks after conflict resolution and owns the required full Model pipeline, reviewed joined snapshots and any current guest publication. This handover certifies only this isolated source lane.

Conductor joined source at bc78db241d14836141a8df039c0d0630b4e16107 from actual premerge base 2f195824aa7cfba880aea880cd4f3048982a7807. The additive catalog manifest conflict retains both thiserror and umpire-checker. Postconflict mandatory helper libraries passed27 tests and literal runtime wire controls passed8; the four affected libraries passed Clippy with -D warnings, and workspace formatting passed. Current locked metadata independently verifies the normal/build dependency graph is acyclic with both catalog dependencies. Historical lane dependency/guest feature inventories retain their original scope and are not substituted for the joined current graph or final publication.

The authentic fresh readonly three-draw review uses codex:gpt-6.1-sol:high, honestly same-family, with raw outputs and receipt in task2/review. The source ledger records the actual merge base; prior helper negative controls/semantic witness checks remain evidence. No snapshot was reaccepted and no guest/full R5 pipeline or parent closure was run. Deferred task7 remains unstarted.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 418d6bd06e18aba85e2d2455946f24f385df3fe8, 44a42e1a84f2b27ab76b457b2d2d59123fd41165, f10ff7a4fa6304f802a33afe6976c56a964f563e, bc78db241d14836141a8df039c0d0630b4e16107
- Tests: cd model && mise exec -- cargo test -p umpire-checker -p umpire-suite -p umpire-catalog -p umpire-front --lib --locked, cd model && mise exec -- cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --locked, cd model && mise exec -- cargo test -p umpire-testing --lib source_edit_tests --locked, cd model && mise exec -- cargo test -p umpire-catalog --test generation --test black_box --locked, cd model && mise exec -- cargo test -p umpire-suite --test fixtures --locked, cd model && mise exec -- cargo test -p umpire-reference-kettle --test probes --locked, cd model && mise exec -- cargo test -p umpire-temporal --lib probes::generic::tests --locked, cd model && mise exec -- cargo test -p umpire-fixtures --test fixtures world_fairness_source_only_selectors_and_leaves_discriminate --locked, cd model && mise exec -- cargo test -p umpire-temporal-tests --test fixtures source_only_global_budget_controls_keep_laws_and_discriminate --locked, cd model && mise exec -- cargo clippy -p umpire-checker -p umpire-suite -p umpire-catalog -p umpire-front -p umpire-testing -p umpire-apalache -p umpire-reference-kettle -p umpire-temporal -p umpire-temporal-tests -p umpire-fixtures --all-targets --locked -- -D warnings, cd model && mise exec -- cargo clippy -p umpire-runtime --all-targets --locked -- -D warnings, cd model && mise exec -- cargo test -p umpire-catalog --test catalog --locked, cd model && mise exec -- cargo fmt --all --check, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-checker -p umpire-suite -p umpire-catalog -p umpire-front --lib --locked, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --locked, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo clippy -p umpire-checker -p umpire-suite -p umpire-catalog -p umpire-front --lib --locked -- -D warnings, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo metadata --locked --offline --format-version 1, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo fmt --all --check
- PRs:
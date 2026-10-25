---
satisfies: [R1]
---
# fn-248-off-the-shelf-crates-and-consolidation.4 heck, darling and quote in the macro and build crates

## Description

**Size:** M
**Files:** model/umpire/catalog/syntax/**, model/umpire/catalog/build/**, model/temporal/annotate/syntax/**, model/umpire/catalog/tests/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml
**Touches:** [model/umpire/catalog/syntax/**, model/umpire/catalog/build/**, model/temporal/annotate/syntax/**, model/umpire/catalog/tests/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Replace hand-written code generation and parsing in the macro and build crates.

- `heck` for case conversion: `model/umpire/catalog/build/src/wire_schema.rs` (~456, ~473), `model/umpire/catalog/syntax/src/derive.rs` (~632), `model/temporal/annotate/syntax/src/realization.rs` (~558, ~575). The two existing `camel` copies differ (one filters empty parts); decide from the snapshot diffs and record which behavior is kept.
- `darling` 0.24 (syn 3) for catalog `ApiAttributes` (`model/umpire/catalog/syntax/src/derive.rs` ~76-299: `Parse`, `set_once`, `attributes`, `merge`) and `proposal_handler` (`model/temporal/annotate/syntax/src/realize.rs` ~812-845): flags, `key = value`, rename, duplicate detection, error accumulation. Hand-written parsing stays for source-ordered tag/constraint families, positional `also(uuid, ...)`, ordered `oneof(...)`, `rejects(...)`, the rule/mirror split, and the `Declaration` parser (`realize.rs` ~105-155). If darling makes a part worse, leave that part and record why.
- catalog/build Rust generation (`wire_schema.rs`, `services.rs`, ~85 `write!`/`push_str` calls) moves to `quote!` + `prettyplease`, for generated Rust only (protobuf schema text stays string-built).

Re-accept trybuild `.stderr` and expansion snapshots after reviewing the diffs.

### Dependency and refusal controls

`heck` 0.5.0 is already resolved through prost-build; quote and prettyplease 0.3/syn3 are already used. darling is neither declared nor resolved in the audited lock: verify its proposed 0.24/syn3 dependency and R18 admission before selecting features/resolving it. Record new direct/transitive packages and host/build versus guest contexts; do not assume all transitive packages are admitted. Use the existing dependency policy and a current source/evidence admission ledger; identify the exact fn210 R18 note if that acceptance reference requires a metadata update, without rewriting completed task evidence.

Test duplicate attributes, mixed source-ordered families, positional/oneof/rejects forms and malformed offending spans, alongside acronym/underscore/empty-part name cases and generated Rust compile controls. Keep the stated manual grammar boundaries; do not treat `deny_unknown_fields` or darling defaults as a substitute for existing admission checks. Prepare reviewed UI/expansion diffs; final reacceptance follows the join.

## Acceptance
- [ ] No hand-written snake/camel converter remains; generated names match prost-build's.
- [ ] `ApiAttributes` and `proposal_handler` parse through darling; malformed attributes still refuse at the offending span.
- [ ] catalog/build emits generated Rust through `quote` + `prettyplease`.
- [ ] fn-210's R18 note records heck and darling; Focused checks below pass; joined full validation belongs to the final integration task.


## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax --locked
cd model && mise exec -- cargo test -p umpire-catalog --test ui --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
Implemented the source lane for fn-248-off-the-shelf-crates-and-consolidation.4. Catalog scalar options and proposal hooks use darling 0.24.1; naming uses heck 0.5; catalog service and wire Rust generators emit quote tokens formatted by prettyplease 0.3. The protobuf schema text generator remains separate.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran (codex:gpt-6.1-sol:high, same family; three first-round draws NEEDS_WORK, two deduplicated findings fixed, same-session round 2 SHIP at 2026-10-25T02:54:03Z)

The current source is identified by source-sha256.json. commands.json records commands, observed exits and log paths. baseline.exit and baseline-ui.exit are both 0; final focused, UI, Clippy, format and dependency-ban checks passed. The initial baseline log was streamed without retaining its terminal exit, so that observation is inconclusive; baseline-observed.log and baseline.exit record the properly captured pre-edit baseline. The first guest tree attempted the retired package name umpire-wasm and refused with exit101; the corrected six-package inventory succeeded and is not a guest build or execution claim.

Scalar options retain their original metas through attribute-line merging. Darling detects duplicates and accumulates independent duplicate-key errors across lines with original spans. Bare Path/LitStr admission remains explicit because darling's default path reader also accepts quoted paths. Supplemental is darling's word-only Flag. Source-ordered tags/constraints, positional also, ordered oneof, rejects and presence rules, rule/mirror separation and Declaration remain manual. Expectations retain the offending source span across lines.

Naming follows heck/prost segmentation: acronym HTTPInput becomes HttpInput, foo__Bar becomes foo_bar and empty underscore segments disappear. Both former camel implementations effectively contributed nothing for empty segments (one filtered them, the other mapped them to empty strings); the retained behavior is heck's, with its acronym/boundary normalization. Build module identifiers use prost-build's Module sanitizer; build type names handle Self/numeric prefixes. No legacy generated-name compatibility layer was introduced.

The new attribute_controls and proposal_options_controls tests cover duplicate flags/keys/directions, accumulated cross-line errors, malformed values, ordering and positional grammar. parser-red.log proves the previous repeated supplemental acceptance and incorrect underscore conversion before the fix. wire_generated is a real rustc compile/run control over independently authored prost-shaped carriers: acronym/underscore fields, optional/repeated/map/oneof mutation, absent cleanup and collection limits. wire-red.log shows the old generator's intended acronym/member errors; the post-change build controls pass, including the existing service compile-pass and exhaustive compile-fail controls.

The new UI stderr was generated by the failing ui-candidate run, inspected, then copied as the expected diagnostic. Duplicate supplemental/context/api highlight second occurrences, and the invalid quoted path highlights its literal. Existing UI fixtures and expansion snapshots did not change; the actual focused expansion tests passed with their existing expectations. No automatic blanket snapshot acceptance occurred.

Dependency admission is in dependency-inventory.json, backed by the crates.io API bytes in crates-io-admission.json and the resolved lockfile. New packages are darling/core/macro 0.24.1 (2026-08-20 releases, MIT, syn3) and ident_case 1.0.1 (finished, widely used; 533M downloads at admission). Darling defaults are off, avoiding suggestions/strsim. Heck was already resolved through prost-build and is now direct in the three host crates. Existing syn3/quote/proc-macro2/prettyplease packages are reused. six-guest-normal-tree.txt excludes these packages from all six guest normal closures when proc-macro edges are removed; darling-inverse-tree.txt shows the macro/build routes. No dependency cycle was added. The conductor appended the resolution clarification to the existing fn210 R18 admission note centrally; this lane did not edit completed Flow records.

Join notes: task5 independently changes WKT descriptor/import/type mappings in wire_schema.rs, realization.rs and inferred_type in derive.rs. Merge its semantic WKT changes into the new quote structures; preserve its primitive-wrapper leaf handling through descriptor::well_known_primitive. The rewritten wire generator parses Rust types as syn::Type, including unit; task5 keeps catalog mirror types as Path and explicitly refuses named-field primitive WKT mirrors. The codegen child coordinated those adaptations with task5. The final integration task owns joined snapshot review/reacceptance, regenerated guests and canonical R5 proof. This lane does not certify joined inputs or publish guests.

Conductor review closure: exact integration base ea98df9ff47c6e98508e7b71b3512d4291df5c1f. Root joined validation passed before review. Fix f40149592b293c958b1b636ff4089299a36a6314 preserves generic scalar-option paths through direct ExprPath construction and shares prost-compatible keyword/numeric sanitization across inference, descriptor realization and build generation. Meaningful controls failed before fixes. The final affected three-package tests passed 52 tests, catalog UI harness passed its three unchanged compile-fail fixtures and new compile-pass fixture, and all-target/all-feature Clippy passed with -D warnings. Shared naming is compared to real prost-build module naming and generated special-type output. All intermediate failures and corrected independently verified numeric-name expectations are retained in task4/review/fix-evidence.md and raw logs.

The authentic backend receipt is task4/review/receipt.json; both findings are fixed, no unaddressed R-IDs remain. The reviewer independently ran updated cached targeted binaries; Cargo reruns were blocked by its readonly build lock. Writer-run focused gates exited zero. Gate classification returned FULL for Cargo.lock; this component uses its required focused gates, with no fabricated full-pipeline receipt or skip. Final task7 still owns the covering pipeline, joined snapshot acceptance and current guest publication. The guest normal-edge inventory is metadata evidence only; host macro/build routes are not a compiled-guest exclusion proof.

stage: memory-capture - ran (bug/build-errors/preserve-generic-paths-and-prost-2026-10-25)

The source-only fix was handed to the isolated task5 join actor; its WKT changes must retain shared sanitization and generic-path construction. No new Cargo or bootstrap changes arose in the fix. Final source hashes and review provenance are durable under task4/review. Root owns remaining in-flight wrap-up; this closure starts no other task.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7996cb97a945a348d1496d4b447de017e80fa2d0, 7ef93ed5e2b03af50e588f1898b6219e07aa43ba, f9507adf06378fb9d4af80f6e7ebd88088da4673, 4b6246f89979facd3d412928c68fa539961cd9e2, 4c4e99ca7e89d87f6c64e7b30cdbbb764294d758, f40149592b293c958b1b636ff4089299a36a6314
- Tests: CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2484-target mise exec -- cargo clippy -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax --all-targets --all-features --locked -- -D warnings, CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2484-target mise exec -- cargo test -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax --locked, CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2484-target mise exec -- cargo test -p umpire-catalog --test ui --locked, CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2484-target mise exec -- cargo fmt --package umpire-catalog-syntax --package umpire-catalog-build --package umpire-temporal-annotate-syntax --check, CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2484-target mise exec -- cargo tree -p umpire-temporal-guest -p umpire-kettle-guest -p umpire-stock-guest -p umpire-control-guest -p umpire-caught-panic-guest -p umpire-overflow-guest --edges normal,no-proc-macro --target wasm32-wasip1 --locked, CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2484-target mise exec -- cargo tree -i darling --target wasm32-wasip1 --locked, CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2484-target mise exec -- cargo deny check bans, cd model; BASH_ENV=/dev/null CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax --locked, cd model; BASH_ENV=/dev/null CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-catalog --test ui --locked, cd model; BASH_ENV=/dev/null CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo clippy -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax --all-targets --all-features --locked -- -D warnings, cd model; BASH_ENV=/dev/null CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo fmt --package umpire-catalog-syntax --package umpire-catalog-build --package umpire-temporal-annotate-syntax --check, git diff --check
- PRs:
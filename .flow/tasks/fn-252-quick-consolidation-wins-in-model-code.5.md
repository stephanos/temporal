---
satisfies: [R5, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.5 Test consolidation: trybuild cases, guest harness, protocol::Method, catalog-syntax tests

## Description
Implement **trybuild cases (B5)**, **Guest test harness (B6)** and **Catalog-syntax tests (B7)** from the spec. Where cases or tests live in `temporal/annotate`, follow fn-251's facade removal if it has landed.

## Acceptance
- [ ] Every removed UI case maps to a table row or unit test covering the same semantic admission/refusal condition with a useful source location. Exact old wording/rendered spans are not required; rustc-dependent type-checking cases remain.
- [ ] `GuestExt`/`ResponseExt`, `protocol::Method` and `assert_additions_refused` exist once.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
Consolidated catalog and realization admission controls, shared native guest/response helpers behind the nondefault runtime `testing` feature, and descriptor addition checks. Catalog grammar tests now live in catalog-syntax; runtime/live dispatch consumes the existing generated `protocol::Method` without changing the schema or exporting a fixture.

Tier: session (jev-unavailable(no_key)).
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Task: `fn-252-quick-consolidation-wins-in-model-code.5`; status remains `in_progress`.
Workspace: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2525-tests`.
Commit range: `b107cb6381f4f5fd7f897b9a82782a836cc19ca9..bc73f4bcd2e5225f84b75848394e917b2b9bc3c7`. Tree clean after commit.

### Validation

Baseline: green, all eight preparation commands before edits. Captured logs are in `fn2525-logs/baseline*.log` beside this handover. The preparation file's SHA-256 was verified as `f84e04c2efd24614fa2ee6e2665e2f67c0042d47e07abceb0b08cbb0b8e5029f` against integrated base `b107cb6381`.

The full affected-crate run passed: 316 tests, zero failures, one previously ignored test; 65 nonempty suites and 18 trybuild compiler checks. The final minor helper cleanup (shared admission explanation extraction and exact refusal-code comparison) was followed by passing focused consumer tests. Strict scoped Clippy across all targets/features, formatting and whitespace checks passed. No test or lint suppression was added.

All Cargo commands ran from `model/` with `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2525`, `CARGO_PROFILE_DEV_DEBUG=0`, `CARGO_PROFILE_TEST_DEBUG=0`, `CARGO_INCREMENTAL=0`, `CARGO_BUILD_JOBS=2`. Gate commands ran in the foreground with captured output; final suite/Clippy observations are recorded in the corresponding `.exit` files. No other cache was used.

- `mise exec -- cargo test -p umpire-catalog-syntax -p umpire-catalog -p umpire-temporal-realization-macros -p umpire-temporal-annotate-syntax -p umpire-temporal -p umpire-runtime -p umpire-reference-kettle -p umpire-temporal-api -p umpire-catalog-build --all-features --locked` — exit 0; `fn2525-logs/affected-crates.log`.
- `mise exec -- cargo test -p umpire-runtime --all-features -p umpire-temporal --test admission --test proposal --test realization --test annotations --test catalog --test stateful --test selected_start --locked` — exit 0; `fn2525-logs/final-helper-consumers.log`.
- `mise exec -- cargo clippy -p umpire-catalog-syntax -p umpire-catalog -p umpire-temporal-realization-macros -p umpire-temporal-annotate-syntax -p umpire-temporal -p umpire-runtime -p umpire-reference-kettle -p umpire-temporal-api -p umpire-catalog-build --all-targets --all-features --locked -- -D warnings` — exit 0; `fn2525-logs/clippy-clean.log`.
- `mise exec -- cargo fmt --all -- --check` — exit 0; `fn2525-logs/fmt-check.log`.

Initial harness adoption correctly failed before `testing.rs` existed (`harness-red.log`). Intermediate checks caught incorrect test token selection, actual syn span extents, missing context in the new compiler control, and mechanical helper adoption/Clippy issues. Those were corrected; no intermediate red was recorded as green. Catalog's new row assertion log is `catalog-moved.log`; realization unit location checks passed in `realize-green.log` and again in the full suite. The final full-suite log proves the compact final realization snippets and all retained compiler cases.

### Per-case migration map

Every entry below removes both its `.rs` and `.stderr` file. Catalog semantic replacements are rows of `MIGRATED_REFUSALS`, exercised by `migrated_ui_refusals_name_the_offending_token` in `model/umpire/catalog/syntax/tests/admission.rs`. Each checks the stated semantic message and exact start/end coordinates of the token or construct. Direct `DeriveCatalog::parse` preserves derive token locations; direct `Catalog` parsing handles the annotation grammar. No proc-macro2 feature was added.

| Removed case | Replacement/refusal | Exact offending span |
| --- | --- | --- |
| `catalog/tests/admission-ui/contradictory_roles` | `contradictory_roles`: contradictory roles or kind/tag pair | `id` (last complete token occurrence) |
| `catalog/tests/admission-ui/empty_ignored_reason` | `empty_ignored_reason`: valid literal reason or positive consequence | `ignored` (last complete token occurrence) |
| `catalog/tests/admission-ui/ignored_kind_type` | `ignored_kind_type`: opaque fields require Bytes or Secret | `value` (last complete token occurrence) |
| `catalog/tests/admission-ui/ignored_rule` | `ignored_rule`: ignored fields cannot appear in rules | `opaque` (first rule argument) |
| `catalog/tests/admission-ui/incompatible_custom_kind` | `incompatible_custom_kind`: custom tag Visibility is incompatible with kind Uuid | `Visibility` (last complete token occurrence) |
| `catalog/tests/admission-ui/incompatible_tag` | `incompatible_tag`: tag IdempotencyKey is incompatible with kind Counter | `idempotency_key` (last complete token occurrence) |
| `catalog/tests/admission-ui/missing_custom_consequence` | `missing_custom_consequence`: expected identifier | `}` (last complete token occurrence) |
| `catalog/tests/admission-ui/missing_request_expectation` | `missing_request_expectation`: request constraint requires an expectation | `id` (last complete token occurrence) |
| `catalog/tests/admission-ui/missing_rule_expectation` | `missing_rule_expectation`: request rule requires an expectation | `any_of` (last complete token occurrence) |
| `catalog/tests/admission-ui/missing_validator` | `missing_validator`: unknown catalog attribute | `covers` (last complete token occurrence) |
| `catalog/tests/admission-ui/opaque_semantic_kind` | `opaque_semantic_kind`: opaque fields require Bytes or Secret | `value` (last complete token occurrence) |
| `catalog/tests/admission-ui/random_decided` | `random_decided`: contradictory roles or kind/tag pair | `bytes` (last complete token occurrence) |
| `catalog/tests/admission-ui/secret_observed` | `secret_observed`: contradictory roles or kind/tag pair | `secret` (last complete token occurrence) |
| `catalog/tests/admission-ui/wildcard_action` | `wildcard_action`: action must name a variant; wildcard bindings hide additions | `_` (last complete token occurrence) |
| `catalog/tests/ui/malformed_attributes` | Four independent rows: `duplicate_supplemental`, `duplicate_context`, `duplicate_api`, `quoted_context` | Second `supplemental`, second `context = Second`, second `api = second`, and the quoted `"crate::Context"` literal respectively; duplicate and expected-identifier messages checked separately |
| `realization/tests/ui/supplemental_missing_covers` | `supplemental_missing_covers`: unknown catalog attribute | `covers` |
| `realization/tests/ui/supplemental_missing_expectation` | `supplemental_missing_expectation`: request constraint requires an expectation | `value` |
| `realization/tests/ui/supplemental_presence_missing_expectation` | `supplemental_presence_missing_expectation`: request rule requires an expectation | `required` |
| `realization/tests/ui/supplemental_blank_covers` | `supplemental_blank_covers`: request constraint requires an expectation | `value` |
| `realization/tests/ui/realization_duplicate` | `migrated_realization_refusals_retain_source_locations`, duplicate observation handler | Entire second `#[observe(...)]` attribute |
| `realization/tests/ui/realization_fields` | Existing `field_declarations_use_the_admitted_catalog_graph` rows strengthened: empty ignore reason, overlapping paths, opaque nested path | Exact `" "` literal; exact `WorkflowService::DescribeActivityExecution` path for overlap and opaque-path declarations |
| `realization/tests/ui/realization_produces` | Migrated realization row, answer requires at least one produces request context | `PollActivityTaskQueueResponse` in the answer attribute |
| `realization/tests/ui/realization_return` | Migrated realization row, expected Projection, Step, or state value with project | `usize` |
| `realization/tests/ui/realization_unknown` | Migrated realization row, unknown `temporal.api.activity.v1.ActivityExecutionInfo.not_a_field` | Exact handler `WorkflowService::DescribeActivityExecution` path |
| `realization/tests/ui/realization_wildcard` | Migrated realization row, generated enum matches must name every variant | `_` arm |
| `realization/tests/ui/realization_alias` | Migrated realization row using `expand_in_source` with original aliased enum import | `_` arm |
| `realization/tests/ui/realization_variant_alias` | Migrated realization row using `expand_in_source`, original variant import and typed `sample()` result | `_` arm |
| `catalog/tests/admission-ui/added_enum_variant` | Retained exact generic `catalog/tests/ui/added_enum_variant` compiler control | Rust's missing `Status::Added` exhaustiveness diagnostic and source location |
| `catalog/tests/admission-ui/omitted_field` | Retained generic `catalog/tests/ui/omitted_field` compiler control | Rust's omitted `omitted` field diagnostic at derive expansion |
| `reference/kettle/tests/ui/omitted_enum` | Same retained generic added-enum compiler control | Rust exhaustiveness over the actual enum variants |
| `reference/kettle/tests/ui/omitted_field` | Same retained generic omitted-field compiler control | Rust's exhaustive mirror pattern rejects an omitted field |

31 original UI cases were consolidated. Their semantic coverage is 22 catalog table rows, seven schema-backed realization rows and three strengthened existing field-declaration rows; four duplicate compiler cases reuse the retained generic controls. `supplemental_blank_covers` additionally retains its independent rustc obligation in the new minimal `supplemental_unregistered_api` UI control: valid `#[rule(accepts)]` prevents an admission error from masking rustc's unregistered `#[api]` rejection. `ui-pass/generated_api.rs` still compiles.

All remaining catalog admission compiler controls remain: added action/actor/oneof, response checks, kind/type mismatch and repeated bytes/text scalar handling; generic added enum and omitted field controls remain under `catalog/tests/ui`. Realization identity, source-scope, external-alias and phase-enum-addition compiler controls remain. Kettle omitted-action and omitted-method controls remain. Existing Temporal syntax `tests/realize.rs`, including its extensive alias controls, stays in place.

### Shared harness and descriptor controls

`runtime/src/testing.rs` owns `GuestExt::{call, judge, reset, propose}` and `ResponseExt::{refusal, explanation, unexplained}` once. All successful typed calls publish through `Guest::dispatch_to`; `GuestExt::call` uses UFCS at consumers because the inherent three-argument ABI method remains `call`. Response helpers return the full refusal or explanation, preserving message, step and outcome assertions. Runtime catalog/direct-dispatch tests retain direct runtime calls where dispatch is the subject, while sharing response extraction. Stateful/selected-start/proposal native guest publication controls use shared successful calls and preserve false publication, panic, oversize, freeze, rollback and raw malformed-byte controls. Kettle uses `Guest<KettleRuntime>` for its successful typed harness. No generic fixture or `Model` export was introduced.

`catalog/syntax/tests/support/mod.rs::assert_additions_refused` is shared by path includes from Kettle descriptor tests, Temporal API descriptor tests and catalog-build neutral tests. It validates the baseline once, clones that descriptor independently for each added field/enum value/method, mutates a selected fully qualified declaration and requires the refusal to name the added symbol. Neutral tests retain all three omission controls and the newly integrated generated-dispatch controls. Service generator implementation, probe implementation, Go generated files, TLA/Apalache replay and dev/testing were untouched.

### Goldens and manifests

The expansion snapshot moved to `model/umpire/catalog/syntax/tests/snapshots/expansion__typed-catalog.snap`; only its source metadata changed. Its body is byte-identical, SHA-256 `cf4c3d6854010e866d02374a78cd7561de4f71755cdd84ce907f6970a1627108`. The only new diagnostic golden is `realization/tests/ui/supplemental_unregistered_api.stderr`, reviewed to contain exactly `cannot find attribute api` at the authored helper. Removed diagnostic goldens are accounted for in the migration map; all remaining goldens passed without overwrites. No `.snap.new` or unreviewed trybuild wip file is retained.

Touched manifests: `model/umpire/protocol/runtime/Cargo.toml` (nondefault testing feature and self dev-dependency), `model/temporal/realization/Cargo.toml` and `model/umpire/reference/kettle/Cargo.toml` (enable testing for tests), `model/umpire/catalog/syntax/Cargo.toml` and `model/temporal/annotate/syntax/Cargo.toml` (move insta/test targets), and `model/temporal/api/Cargo.toml` (direct catalog-syntax test support dependency). `model/Cargo.lock` records only those existing package edge changes; no package/version was added.

### Conductor obligations

Integrate the commit, obtain independent review, regenerate the current guest artifacts if required by runtime/live source changes, and run the joined `make model-check`. Neither that full model gate nor guest regeneration ran in this worker, per dispatch. No gate receipt or review verdict is claimed. The task remains `in_progress`; the conductor owns Flow completion, any tracker projection and MILESTONES changes. No mapped app route changed.

### Review follow-up: integration references

The conductor's independent review found two integration failures introduced by the test move. The moved admission fixture used a system-specific protobuf name under the generic source root; the snapshot registry still pointed at the old location. Both existing tests reproduced before the fix at `88531598eea0c661827474a9dbfb68ca98d86e0c`:

- `mise exec -- cargo test -p umpire-temporal-tests --test boundary --test snapshot_ownership --locked` — exit 101, `generic_source_boundary` rejected `catalog/syntax/tests/admission.rs:7`; Cargo stopped before the snapshot target. Evidence: `fn2525-logs/review-integration-red.log` and `.exit`.
- `mise exec -- cargo test -p umpire-temporal-tests --test snapshot_ownership --locked` — exit 101, `current_snapshots_have_explicit_owners` reported that the moved expansion snapshot had no consumer. Evidence: `fn2525-logs/review-ownership-red.log` and `.exit`.

Fix commit: `bc73f4bcd2e5225f84b75848394e917b2b9bc3c7`. The fixture now uses `example.v1.Request`. `READERS` names the moved catalog-syntax snapshot and its concrete catalog-syntax test command. Only those references changed in source; boundary and registry checks, semantic rows, Models, generated files and goldens were unchanged. The conductor explicitly authorized including its preexisting finalized review ledger without edits; this checkpoint carries that ledger too.

Final follow-up checks (all exit 0):

- `mise exec -- cargo test -p umpire-temporal-tests --test boundary --test snapshot_ownership -p umpire-catalog-syntax --locked` — `fn2525-logs/review-integration-green.log` and `.exit`.
- `mise exec -- cargo test -p umpire-catalog-syntax --locked` — `fn2525-logs/review-catalog-syntax.log` and `.exit`.
- `mise exec -- cargo clippy -p umpire-catalog-syntax -p umpire-temporal-tests --all-targets --all-features --locked -- -D warnings` — `fn2525-logs/review-clippy.log` and `.exit`.
- `mise exec -- cargo fmt --all -- --check` — `fn2525-logs/review-fmt.log` and `.exit`.
- `git diff --cached --check` — checked staged source/ledger whitespace.

The integration run executed four tests (two boundary, two snapshot ownership), including both negative controls. The separate catalog-syntax run executed all 36 crate tests; none failed. Cargo checks used the same task-owned cache/debug/incremental/jobs settings as the original handover. This follow-up leaves the task `in_progress`, the worktree clean and independent re-review/full joined validation with the conductor. No review was dispatched by this worker.

Defect route:
- prior fixes: conductor confirmed exclusive ownership of these two references; local file history identifies the original move commit; bounded search found no sibling stale reference; memory search found no existing fix. External PR/tracker searches not done: this is a conductor-assigned local review repair with exclusive ownership.
- diagnosis: source-boundary negative controls passed while the live generic scan identified the moved fixture name; registry negative controls passed while the actual snapshot inventory identified its moved location. This confirms stale integration references rather than weakened scanner/registry behavior.
- introduced by: `88531598eea0c661827474a9dbfb68ca98d86e0c`; bisect not done because the immediately preceding implementation commit is the known source move under repair and both failing paths are visible directly in that diff.
- base: both unchanged existing controls failed at `88531598eea0c661827474a9dbfb68ca98d86e0c`; head: both pass at the follow-up commit, with their negative controls retained. No new failing-test commit was needed because the reproduction tests already existed in committed history.
- live: no live surface; source scanning and snapshot ownership are library/tooling controls.

Conductor acceptance: integrated in umpire at code checkpoint d30d679fc1; task implementation review reached actual SHIP (including supplemental fixes). The joined UMPIRE_APALACHE=require make model-validate exited 0 on the unchanged model tree, including required TLC/Apalache controls, strict workspace clippy/docs/tests, embedded guest freshness, all standard wasm/native controls and all three Go hosts. Gate 1308.170s exceeds the 60s soft target; edit/test 8.882s meets 10s. Logs: consolidation-retry-model-validate.log and .exit. Final boundary/snapshot checks and reviewed golden/guest updates are integrated.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 88531598eea0c661827474a9dbfb68ca98d86e0c, bc73f4bcd2e5225f84b75848394e917b2b9bc3c7, d30d679fc1
- Tests: mise exec -- cargo test -p umpire-catalog-syntax --locked, mise exec -- cargo test -p umpire-catalog --test admission --test ui --locked, mise exec -- cargo test -p umpire-temporal-realization-macros --lib --locked, mise exec -- cargo test -p umpire-temporal --locked, mise exec -- cargo test -p umpire-runtime --all-features --locked, mise exec -- cargo test -p umpire-reference-kettle --locked, mise exec -- cargo test -p umpire-temporal-api --test descriptor --locked, mise exec -- cargo test -p umpire-catalog-build --test neutral --locked, mise exec -- cargo test -p umpire-catalog-syntax -p umpire-catalog -p umpire-temporal-realization-macros -p umpire-temporal-annotate-syntax -p umpire-temporal -p umpire-runtime -p umpire-reference-kettle -p umpire-temporal-api -p umpire-catalog-build --all-features --locked, mise exec -- cargo test -p umpire-runtime --all-features -p umpire-temporal --test admission --test proposal --test realization --test annotations --test catalog --test stateful --test selected_start --locked, mise exec -- cargo clippy -p umpire-catalog-syntax -p umpire-catalog -p umpire-temporal-realization-macros -p umpire-temporal-annotate-syntax -p umpire-temporal -p umpire-runtime -p umpire-reference-kettle -p umpire-temporal-api -p umpire-catalog-build --all-targets --all-features --locked -- -D warnings, mise exec -- cargo fmt --all -- --check, git diff --cached --check, mise exec -- cargo test -p umpire-temporal-tests --test boundary --test snapshot_ownership -p umpire-catalog-syntax --locked, mise exec -- cargo test -p umpire-catalog-syntax --locked, mise exec -- cargo clippy -p umpire-catalog-syntax -p umpire-temporal-tests --all-targets --all-features --locked -- -D warnings, mise exec -- cargo fmt --all -- --check, git diff --cached --check, UMPIRE_APALACHE=require make model-validate (exit 0; TLC and Apalache required; 1308.170s full gate, 8.882s inner loop; unchanged model tree), Actual Codex gpt-6.1-sol:high SHIP task implementation review and supplemental re-review; same-family CLI reviewer
- PRs:
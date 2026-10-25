---
satisfies: [R3, R4]
---
# fn-246-realization-as-a-checked-refinement-a.12 Shared descriptor catalog, claims and per-case opaque probe accounting

## Description
Implement R3's descriptor catalog/claims and the inventory semantics needed by R4. Keep this admission change separate from lens relocation so probe loss cannot hide inside a directory move.

**Size:** M
**Files:** Temporal descriptor catalog generation/field kinds, service declaration admission, generic catalog extension only where necessary, Temporal inventory preparation and exact delta controls.
**Touches:** [model/temporal/shared/catalog/**, model/temporal/foundation/**, model/temporal/annotate/syntax/src/services.rs, model/temporal/annotate/syntax/src/realization.rs, model/temporal/annotate/tests/**, model/temporal/api/src/lib.rs, model/temporal/api/tests/build_inputs.rs, model/temporal/feature/activity/api.rs, model/temporal/feature/activity/claims.rs, model/temporal/feature/activity/probe_gaps.rs, model/temporal/realization/src/probes.rs, model/temporal/realization/src/probes/generic.rs, model/temporal/realization/tests/catalog.rs, model/temporal/realization/tests/probes.rs, model/temporal/realization/tests/start_mirror.rs, model/temporal/realization/tests/probe_accounting.rs, model/temporal/realization/tests/data/probes-before-claims.binpb, model/umpire/catalog/**, model/README.md, model/temporal/realization/tests/data/.gitattributes]

### Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Capture the complete previous descriptor-driven probe inventory and accepted-gap IDs, mutations, applicability and predictions before changing admission. Preserve `wire_cases`' generic shape-only behavior and Kettle's inventory.
2. Generate the shared WorkflowService mirrors/service inventory from the pinned descriptors through the current generator owner. Define shared field kinds and reject patterns once. Claims cover actual reads and fills; unclaimed descriptor fields become Ignored, with an explicit generated reason. Check exhaustiveness against independent claimed descriptor/type inventories.
3. Apply opaque suppression in Temporal's claim-aware inventory assembly. Remove inferred mutations under genuinely unclaimed Ignored subtrees, including empty/negative/undefined-enum/length cases. Suppress unsupported inferred Secret bounds while retaining explicit required token presence and any explicit supported constraint. No blanket generic mutation suppression.
4. Implement the conductor-approved R4 clarification. Conditional queue-name and retry-ordering cases remain present, with independently justified typed semantic rules and preserved paired controls. Validate actual API/World applicability before giving strict predictions; retain visible gaps and fail acceptance if the rule cannot be justified. Do not fit a rule to a live rejection.
5. Produce an exact per-ID old/new inventory reconciliation. A retained case may gain a strict prediction; a removed case must cite its actual opaque/claim rule. Remove no claimed-field/enum/request-evidence/response-validation controls.

**Dispatch exclusion:** Descriptor generation also owns the existing Temporal API include/build-input seams. Serialize production schema/API integration with .14; its completed SDK-only proof is disjoint. Serialize with .11 on annotation syntax/tests; the parent overlap guard applies even when the task DAG otherwise permits parallel work. Preserve existing dependencies.

### Investigation targets

**Required:**

- `model/temporal/realization/src/probes/generic.rs:33-90` for inventory, ordering and prediction reconciliation.
- `model/umpire/catalog/src/black_box.rs:117-166`, `catalog.rs:698,745`, `kind.rs:198` for shape-only mutations and kind/rule admission.
- `model/temporal/annotate/syntax/src/services.rs` for reuse of generic service syntax.
- `model/temporal/foundation/api.rs:64-74,147` for retry and queue-name rules.
- `model/temporal/feature/activity/probe_gaps.rs` and `api.rs:245` for all original gaps and required Secret presence.
- `common/tqid/task_queue_validator.go:110`, `service/frontend/workflow_handler.go:6718` and pinned protocol evidence as compatibility facts; they cannot be used to repair independent Model semantics.

### Verification

From model/: `mise exec -- cargo test -p umpire-catalog -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate -p umpire-temporal`; run focused existing descriptor freshness/controls. Live retained queue/retry controls are mandatory with .3/.6's rebuilt guest; no success claim from inventory-only checks.

## Acceptance
- [ ] Descriptor catalog and explicit claims drive shared field kinds and inventory. Added claimed fields/variants, invalid claim/rule combinations and wrong descriptor/result/request/response identities fail; unclaimed fields have auditable Ignored reasons.
- [ ] Exact old/new case manifest explains every removal by actual opaque/claim policy. Ignored empty/negative/enum cases and conditional queue/retry cases are accounted individually; explicit Secret presence and paired semantic controls survive.
- [ ] Retained nonopaque semantic cases have independently justified strict predictions or visible acceptance failures. Deleting probes, changing Models or fitting observed SUT outcomes cannot empty the gap list.
- [ ] Generic/Kettle inventories remain exact, focused catalog/syntax tests and existing freshness controls pass, and the exact pending live-control obligations are handed to .3/.6.


## Done summary
Final-source review fixes are committed. Describe include_outcome/include_last_failure are claimed Flags and all twelve original controls are restored unchanged. The current Rust/owned-Go read/fill audit also claims Poll response.header and Start response.link/Activity identity. Failure.source has an explicit comparison-copy ignore reason; untouched Default fields, opaque forwarding and unrelated SDK-managed fields do not acquire invented semantics. Source hashes and the complete audit are in read-fill-audit.md and read-fill-audit-sources.json.

Retry ordering now calls the existing shared View<kinds::Duration> normalization/cap before validity and relative-order checks. Cap equality and excess-nanosecond inversion controls reproduce both original defects and now pass. Disabled/default applicability and mixed-sign validity behavior remain intact. Inversion construction rechecks the same projection and refuses atomically at the cap. No Model policy changed.

Original baseline accounting is 2,317 cases. Current inventory is 1,141 cases (1,060 generic and 81 strict). Every original strict case retains its metadata (retained_strict_count=79; after_strict_count=81). The complete manifest records 1,122 actual Ignored removals, 56 unsupported Secret-bound removals, four strengthened semantic IDs and two added strict cases. Its reasons carry exact descriptor identities, including ignored oneof leaves; the provenance assertion remains unchanged. Restored unpredicted controls may expose live gaps and no empty-gap/live success claim is made.

Final seven-package tests passed 281 cases with one existing ignored live-import target. Strict all-target/all-feature clippy, warning-denied docs, workspace formatting and both boundaries exited 0. All helpers joined. Original and previous-round red/green evidence, handovers and review records are preserved. Kettle tests passed in the same suite; its earlier full 176-case original/current Engine equality proof remains recorded.

The fn24612 Cargo cache is released to the conductor. Worker environment remains CARGO_PROFILE_DEV_DEBUG=0, CARGO_PROFILE_TEST_DEBUG=0, CARGO_INCREMENTAL=0 and CARGO_BUILD_JOBS=2. Inherited cached/embedded guest hashes are listed in fn24612-round3-guest-hashes.json, explicitly marked pre-fix; this worker built no guest. Root owns combined guest refresh, resumed primary review, joined verification and the unresolved full required gate. Tasks .3/.6 retain the exact actual queue/retry/token live obligations. No task7-owned source, lifecycle state, MILESTONES or remote ref changed.

Tier: implementer gpt-6.1-sol at medium (explicit dispatch).
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn24612-catalog
Original range: a8e4067edba5ba8b68380ea19840752f43920635..755fd7b9b94d823cd759baff804c286cacabbc47
Current fix range: da3ece49611e245a49e3f50106170b15578313db..755fd7b9b94d823cd759baff804c286cacabbc47
Evidence: /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/milestones-resume-20261024/fn24612-evidence.json
Proofs: /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/milestones-resume-20261024/fn24612-proof
Task status: in_progress

Conductor closure evidence: fix755fd7b9b9 was cherry-picked asa70f54ec5b and original ancestry merged at143fc6de7c with identical model tree83aeb79433bd665989a5e836fe6fab7b886fdca1. All audited source and accounting hashes match the integrated checkout. The joined eight-package suite passed294 tests with one existing ignored live-import target (139.584s). Current Temporal and six standard guests/native panic reports built, all three Go host modules passed, and guest freshness passed. Embedded guest SHA2560f560e1073b6b4b25feed794fcfdffd0c94b4ad409c7c711fefbfd2d0f67acc4. Actual final primary re-review SHIP at143fc6de7c, session01a17334-6fe9-7291-80d1-36915da0a5ce, codex:gpt-6.1-sol:high (same family); four prior findings fixed, no surviving blockers. Two nontrivial correction lessons captured in Flow memory.

The parent full required gate remains outstanding after its preserved Apalache subprocess refusal; isolated reruns are not a passing full gate. Original queue/retry/token live obligations remain .3/.6 owned, actual cooperative worker proof remains fn237.4, and the original baseline tool-exit ambiguity remains recorded rather than promoted to a green receipt. This per-task closure uses focused verification on the rolling route and claims no live-service or parent-completion result. stage: plan-sync skipped(policy: rolling scheduler).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3dd5486d3098a4348ab4cd1ba7681289c3d70c09, da3ece49611e245a49e3f50106170b15578313db, 755fd7b9b94d823cd759baff804c286cacabbc47, a70f54ec5b258f4e735c152c2f79c9ae6d129ee2, a479ed1a1b, 143fc6de7c27bdbea5d2514c6db8b7ca252de478
- Tests: timeout 600 mise exec -- cargo test -p umpire-catalog -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax -p umpire-temporal-api -p umpire-temporal -p umpire-reference-kettle, timeout 600 mise exec -- cargo clippy -p umpire-catalog -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax -p umpire-temporal-api -p umpire-temporal -p umpire-reference-kettle --all-targets --all-features -- -D warnings, RUSTDOCFLAGS='-D warnings' timeout 600 mise exec -- cargo doc -p umpire-catalog -p umpire-catalog-syntax -p umpire-catalog-build -p umpire-temporal-annotate-syntax -p umpire-temporal-api -p umpire-temporal -p umpire-reference-kettle --all-features --no-deps, mise exec -- cargo fmt --all -- --check, timeout 600 mise exec -- cargo test -p umpire-temporal-tests --test boundary, python3 .flow/tmp/fn24612-compile-dump.py, python3 .flow/tmp/fn24612-account.py, Conductor joined8pkg locked suite:294passes/1existingignore;139.584s, Current Temporal+six standard guests/native panic, all3 Go modules and freshness: all exit0
- PRs:
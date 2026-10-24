---
satisfies: [R3, R4]
---
# fn-246-realization-as-a-checked-refinement-a.12 Shared descriptor catalog, claims and per-case opaque probe accounting

## Description
Implement R3's descriptor catalog/claims and the inventory semantics needed by R4. Keep this admission change separate from lens relocation so probe loss cannot hide inside a directory move.

**Size:** M
**Files:** Temporal descriptor catalog generation/field kinds, service declaration admission, generic catalog extension only where necessary, Temporal inventory preparation and exact delta controls.
**Touches:** [model/temporal/shared/catalog/**, model/temporal/foundation/**, model/temporal/annotate/syntax/src/services.rs, model/temporal/annotate/tests/**, model/temporal/realization/src/probes/generic.rs, model/temporal/realization/tests/{catalog,probes,start_mirror}.rs, model/umpire/catalog/**]

### Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Capture the complete previous descriptor-driven probe inventory and accepted-gap IDs, mutations, applicability and predictions before changing admission. Preserve `wire_cases`' generic shape-only behavior and Kettle's inventory.
2. Generate the shared WorkflowService mirrors/service inventory from the pinned descriptors through the current generator owner. Define shared field kinds and reject patterns once. Claims cover actual reads and fills; unclaimed descriptor fields become Ignored, with an explicit generated reason. Check exhaustiveness against independent claimed descriptor/type inventories.
3. Apply opaque suppression in Temporal's claim-aware inventory assembly. Remove inferred mutations under genuinely unclaimed Ignored subtrees, including empty/negative/undefined-enum/length cases. Suppress unsupported inferred Secret bounds while retaining explicit required token presence and any explicit supported constraint. No blanket generic mutation suppression.
4. Implement the conductor-approved R4 clarification. Conditional queue-name and retry-ordering cases remain present, with independently justified typed semantic rules and preserved paired controls. Validate actual API/World applicability before giving strict predictions; retain visible gaps and fail acceptance if the rule cannot be justified. Do not fit a rule to a live rejection.
5. Produce an exact per-ID old/new inventory reconciliation. A retained case may gain a strict prediction; a removed case must cite its actual opaque/claim rule. Remove no claimed-field/enum/request-evidence/response-validation controls.

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

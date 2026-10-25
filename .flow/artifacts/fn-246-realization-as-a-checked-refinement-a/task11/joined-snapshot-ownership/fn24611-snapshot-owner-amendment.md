---
satisfies: [R2, R3]
---
# fn-246-realization-as-a-checked-refinement-a.11 Generic variant metadata and Temporal action/entity/read registry

## Description
Implement the R3 action-declaration seam after capabilities. This task owns generic metadata forwarding and Temporal registry/entity/read syntax, with synthetic consumers proving the interface before activity migration.

**Size:** M
**Files:** generic front/action macro metadata forwarding, Temporal annotation syntax/derive support and focused pass/fail controls.
**Touches:** [model/temporal/testing/snapshot_ownership.rs, model/umpire/front/src/**, model/umpire/author/dsl/macros/src/**, model/umpire/author/dsl/src/action.rs, model/umpire/author/dsl/src/lib.rs, model/umpire/author/dsl/**/tests/**, model/umpire/author/umpire/tests/consumer/**, model/temporal/annotate/**, model/temporal/realization/macros/**, model/temporal/realization/src/registry.rs, model/temporal/realization/src/lib.rs, model/temporal/realization/tests/registry.rs, model/temporal/realization/tests/registry/**, model/temporal/realization/tests/declarative.rs, model/temporal/realization/tests/capabilities.rs, model/temporal/realization/tests/realization.rs, model/temporal/realization/tests/realization_admission.rs, model/temporal/realization/tests/admission/**, model/temporal/realization/tests/ui/**, model/temporal/realization/tests/ui-pass/**]

### Approach

2026-10-25 joined-gate amendment: the second required gate passed all 39 activity fixture tests (2 inherited ignored), then rejected the new typed-action-registry snapshot as unowned. Reopen .11 and add its exact existing typed_registry_expansion consumer to the canonical READERS list. This additional test-metadata owner is admitted after the failed gate; preserve the orphan/pending/missing-reader negative controls and snapshot bytes. Run the ownership target and real snapshot consumer, scoped fmt/clippy, then renew review. No broad scanner exception or new behavior is authorized.

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Reanchor the final fn-237.6/fn-237.3 action declaration context. Extend the existing action-variant expansion with inert per-variant metadata forwarding only. Keep machine body, transition semantics, catalogs and backend lowering unchanged.
2. Add Temporal-owned rpc/environmental/entity/read declarations using descriptor/type identity admission and existing realization syntax owners. Normalize registry rows once and expose typed classification, fill and evidence access to the capabilities; consumers must not rebuild per-action tables.
3. Preserve actual generated enum/result type proofs. Test omitted/added actions and claimed fields against independent descriptor/type inventories rather than the declaration being checked. Keep correlation/read configuration typed and reject conflicting/unknown/duplicate declarations with located diagnostics.
4. Prove a generic non-Temporal Machine still expands and keeps the same checking results (goldens may be re-accepted after review). Use synthetic Temporal examples to compile client proposal, worker fill and probe-baseline access before the activity switches. Refinement syntax is owned by .4 after its typed consumer exists.

Synthetic registry controls belong in `tests/registry.rs` and `tests/registry/**`, retaining existing declarative, capability, admission, UI and renamed controls. Per-feature program codec/history controls remain with .14. This ownership follows the synthetic-before-activity migration boundary; any additional source owner requires a fresh admission check without reducing verification.

**Dispatch exclusion:** Serialize with .12 on annotation syntax/tests; the parent overlap guard applies even when the task DAG otherwise permits parallel work. Preserve existing dependencies.

### Investigation targets

**Required:**

- `model/umpire/front/src` and `model/umpire/author/dsl/macros/src/action.rs` for current action admission/expansion; resolve exact owner after reanchor.
- `model/temporal/annotate/syntax/src/realize.rs:739,845,968` for action coverage, result shape and generated dispatch.
- `model/temporal/annotate/syntax/src/realize/patterns.rs` for existing pattern/type validation.
- `model/temporal/realization/macros/src/lib.rs` for the actual `#[realization]` entry point, and `model/temporal/realization/src/lib.rs` for typed facade support.
- `model/temporal/realization/tests/realization_admission.rs`, `tests/ui/**` and `model/umpire/catalog/tests/admission-ui/**` for existing descriptor/action/type diagnostics; add task-specific controls under the current realization test owner.

### Verification

From model/: `mise exec -- cargo test -p umpire-front -p umpire-dsl-macros -p umpire-temporal-annotate-syntax -p umpire-temporal -p umpire-temporal-realization-macros`; include generic renamed-facade consumer and syntax snapshots. Run scoped fmt/strict clippy and existing generic source-boundary checks.

## Acceptance
- [ ] One typed variant registry serves synthetic proposal, worker row and probe baseline/evidence consumers. Generic metadata support contains no Temporal imports or semantics.
- [ ] Added/omitted actions, unknown/duplicate attributes, wrong descriptor/request/response/result type, conflicting entity/read settings and uncovered claimed variants produce located compile diagnostics.
- [ ] Generic bounds, empty enums and renamed facade controls pass; existing non-Temporal Model rows keep their checking results. No transition/body/backend semantics change.
- [ ] Focused macro/front tests, compile controls, snapshots, strict clippy and boundary verification pass; activity migration can consume the interface without another per-action table.


## Done summary
Integrated typed variant metadata and the generic Temporal registry. Corrected both initial review findings: empty-path root admission now invokes the authored predicates with conflict refusal, and semantic scanning strips opaque variant metadata only from its scan clone while preserving genuine reject!/stay! behavior.

The worker's cohesive six-crate suite passed 175 tests with no ignored tests, with strict clippy/fmt and boundary controls. Root renewed metadata, RegistryReach, expansion, Host parity and guest freshness checks; all source/proof manifests were verified. Final style-only semicolons received scoped lint/fmt checks without repeating the unchanged broad suite. The original primary reviewer resumed the merged findings and returned SHIP at f85d9399c4, explicitly confirming both fixes and no new blockers.

The joined full gate is running, not claimed complete by this task; task .15 retains that closure requirement. Activity migration, compatible frontiers, protocol/refinement, worker policies, live gates and final counts remain assigned downstream.

Tier: AGENTS implementer gpt-6.1-sol/medium; reviewer gpt-6.1-sol/high (same family).
stage: impl-review - SHIP (original primary session 01a173e8-01cb-73d2-a411-26134e411a9e).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:

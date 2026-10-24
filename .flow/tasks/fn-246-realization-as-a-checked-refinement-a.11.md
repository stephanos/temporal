---
satisfies: [R2, R3]
---
# fn-246-realization-as-a-checked-refinement-a.11 Generic variant metadata and Temporal action/entity/read registry

## Description
Implement the R3 action-declaration seam after capabilities. This task owns generic metadata forwarding and Temporal registry/entity/read syntax, with synthetic consumers proving the interface before activity migration.

**Size:** M
**Files:** generic front/action macro metadata forwarding, Temporal annotation syntax/derive support and focused pass/fail controls.
**Touches:** [model/umpire/front/src/**, model/umpire/author/dsl/macros/src/**, model/umpire/author/dsl/**/tests/**, model/temporal/annotate/**, model/temporal/realization/macros/**]

### Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors, golden hashes and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Reanchor the final fn-237.6/fn-237.3 action declaration context. Extend the existing action-variant expansion with inert per-variant metadata forwarding only. Keep machine body, transition semantics, catalogs and backend lowering unchanged.
2. Add Temporal-owned rpc/environmental/entity/read declarations using descriptor/type identity admission and existing realization syntax owners. Normalize registry rows once and expose typed classification, fill and evidence access to the capabilities; consumers must not rebuild per-action tables.
3. Preserve actual generated enum/result type proofs. Test omitted/added actions and claimed fields against independent descriptor/type inventories rather than the declaration being checked. Keep correlation/read configuration typed and reject conflicting/unknown/duplicate declarations with located diagnostics.
4. Prove a generic non-Temporal Machine still expands and its artifacts remain byte-identical. Use synthetic Temporal examples to compile client proposal, worker fill and probe-baseline access before the activity switches. Refinement syntax is owned by .4 after its typed consumer exists.

### Investigation targets

**Required:**

- `model/umpire/front/src` and `model/umpire/author/dsl/macros/src/action.rs` for current action admission/expansion; resolve exact owner after reanchor.
- `model/temporal/annotate/syntax/src/realize.rs:739,845,968` for action coverage, result shape and generated dispatch.
- `model/temporal/annotate/syntax/src/realize/patterns.rs` for existing pattern/type validation.
- `model/temporal/annotate/macros/src/lib.rs` and `model/temporal/realization/macros/src/lib.rs` for derive entry points.
- `model/temporal/annotate/tests/ui.rs` and `tests/ui/{added_action,wildcard_action,added_enum_variant}.rs` for diagnostics.

### Verification

From model/: `mise exec -- cargo test -p umpire-front -p umpire-dsl-macros -p umpire-temporal-annotate-syntax -p umpire-temporal-annotate -p umpire-temporal-realization-macros`; include generic renamed-facade consumer and syntax snapshots. Run scoped fmt/strict clippy and existing generic source-boundary checks.

## Acceptance
- [ ] One typed variant registry serves synthetic proposal, worker row and probe baseline/evidence consumers. Generic metadata support contains no Temporal imports or semantics.
- [ ] Added/omitted actions, unknown/duplicate attributes, wrong descriptor/request/response/result type, conflicting entity/read settings and uncovered claimed variants produce located compile diagnostics.
- [ ] Generic bounds, empty enums and renamed facade controls pass; existing non-Temporal Model rows and artifact hashes remain exact. No transition/body/backend semantics change.
- [ ] Focused macro/front tests, compile controls, snapshots, strict clippy and boundary verification pass; activity migration can consume the interface without another per-action table.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

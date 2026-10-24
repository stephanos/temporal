---
satisfies: [R3]
---
# fn-246-realization-as-a-checked-refinement-a.2 Shared Temporal entity lens and filler-driven probes with equivalence pins

## Description

Extract R3's shared entity lens and filler-driven probe plumbing after capabilities, registry and catalog contracts land. Preserve activity semantics through a compatibility adapter until .3 switches the feature declaration.

**Size:** M
**Files:** existing Temporal live/support/probes owners and their tests, shared lens/probes/support modules, crate exports and guest imports.
**Touches:** [model/temporal/realization/src/**, model/temporal/realization/tests/**, model/temporal/shared/{lens,probes,support}/**, model/temporal/realization/Cargo.toml, model/temporal/guest/**, model/temporal/testing/boundary.rs]

## Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Pin serialized old-vs-new Reset/Propose/Judge responses, receipt transitions, freeze behavior, actual evidence validation, read finality, hold instructions and probe inventory at the integrated prerequisite tip. Use exact existing fixture/live captures as an equivalence harness with a closed dynamic-value projection where necessary.
2. Move admission, immutable receipt/evidence, whole-Runtime replacement, correlation, read scheduling/finality, hold and program-envelope assembly through the existing owners. Keep complete request/response/evidence validation before feature delegation. Preserve publication/commit and failed-allocation behavior. Do not create a sibling lens lifecycle or decision interpreter.
3. Parameterize shared mechanics by declared entity identity/bootstrap/read and assembled capabilities. Keep current activity adapter compiling. Move protocol API names freely but reject imports of feature modules, hard-coded feature phase/World rules and scenario identities through a focused boundary control.
4. Make baseline/catalog probes call registered fillers; consume catalog's exact inventory reconciliation rather than reproducing gap logic. Keep generic raw mutation generation and Kettle unchanged. Keep the journal/gates and cooperative actual-worker live evidence owned by future fn-237.4; the moved executor continuation capability has its own task.
5. Rewire guest/test imports and workspace paths only as needed; add no Cargo dependency beyond those admitted under fn-210 R18. Record shared handwritten/generated ownership for later counts. Do not charge feature-semantic helpers as shared merely because they moved.

## Investigation targets

**Required:**

- `model/temporal/realization/src/live.rs:28-39` and `annotations.rs` for receipt/admission/transactions.
- `model/temporal/realization/src/support.rs` for typed envelope/identity validation.
- `model/temporal/realization/src/probes.rs` and `probes/generic.rs:33-90` for shared inventory owners.
- `model/temporal/realization/tests/admission/publication.rs` and `tests/realization.rs` for rollback/freeze/Reset.
- `model/temporal/feature/activity/temporal.rs:149` for existing proposal correlation, reanchored after .3 prerequisite.
- `model/temporal/testing/boundary.rs` and `model/temporal/guest` for imports and genericity controls.

## Verification

From model/: `mise exec -- cargo test -p umpire-temporal -p umpire-temporal-tests`; build the Temporal guest through its current builder and run focused Go-host parity with `test_dep`. Run existing annotation/probe/facade controls plus the exact equivalence harness, scoped fmt and strict clippy. Full fresh-guest/live gate remains .6.

## Acceptance

- [ ] Before/after serialized judgments, receipt/freeze/Reset, pending correlation, read finality, hold/program envelope and exact probe inventory agree through the named equivalence harness.
- [ ] Failed publication/allocation and wrong/missing/crossed evidence leave state unchanged; only successful publication commits. Existing correlation and actual worker-evidence validation retain their precedence.
- [ ] Shared modules import no feature module and contain no feature-phase/World/scenario policy. API/domain Activity names remain allowed; planted feature-coupling controls fail deterministically.
- [ ] Filler-driven probes and compatibility adapter compile across guest/host consumers. Focused Rust/Go parity, boundary, fmt and strict clippy pass without Cargo dependencies outside fn-210 R18 or claimed future fn-237.4 evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

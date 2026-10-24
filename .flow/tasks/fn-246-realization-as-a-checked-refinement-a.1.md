---
satisfies: [R1]
---
# fn-246-realization-as-a-checked-refinement-a.1 Transactional bounded Reach and Kettle proof point

## Description

Implement R1 at the existing generic judgment owner and establish the Kettle proof point. Keep Realize's assembled interface available while the later capability task migrates consumers.

**Size:** M
**Files:** generic runtime projection/judgment/replay and its direct consumer signatures, runtime controls, Kettle realization and focused expected outputs.
**Touches:** [model/umpire/protocol/runtime/src/**, model/umpire/protocol/runtime/tests/**, model/umpire/reference/kettle/**, model/temporal/annotate/syntax/src/realize.rs, model/temporal/realization/macros/src/snapshots/**]

## Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. At execution admission, require committed fn-237.3 and whole-fn-247 closure, reanchor their World/evidence/Rust build-helper contracts and capture current artifacts, transaction controls and Kettle outputs. Do not inherit research commit results as this task's baseline. All other fn-246 root candidates share these prerequisite gates; source, Cargo/schema/codegen overlap still serializes dispatch.
2. Extend Projection with a typed Reach target and admitted scope, not a snapshot-manufactured full State. Generic ReachTarget<State> owns one pure candidate-compatibility predicate; Exact<State> preserves complete-state matching. Full candidates come only from current-state checked_step/replay. Explicitly adapt each direct/generated Realize target type, including stock/runtime controls, before selecting Reach; do not rely on unstable associated-type defaults. Update generated observe signatures/snapshots in the same compilable commit. The later feature owns its validated partial-observation target; generic runtime learns no Temporal fields or clocks.
3. Implement primitive search inside existing preparation using catalog, checked_step and replay, built on the suite's budgeted search core (`model/umpire/author/suite/src/search.rs`, `pattern_search.rs`, lifted or exposed as needed) rather than a third BFS. Typed paths retain action, enabled choice, emitted Outcome AND Facts plus source/target; exact expiry cause cannot use phase/outcome alone. Preserve v1 textual report meaning and avoid new arbitrary Outcome/Fact Clone bounds. Resolve canonical precedence before a report witness; longer cyclic work does not force a found witness to refuse or earn uniqueness. Complete no-match is Unexplained, truncated no-match/unresolved precedence is Refused. Prefix2 frontier/publication semantics belong to the later prerequisites; do not prune partial hidden alternatives here. Scope depth/work explicitly through realization metadata.
4. If deduplicating, key visited nodes by Model state, evidence cursor/consumption state, path position/depth and every evidence-relevant history component. Preserve distinct typed witness prefixes unless future evidence admission and canonical-precedence equivalence is established. Add a same-state/different-cursor control that would fail state-only dedup, plus a short successful path with longer cyclic frontier that must remain explained.
5. Require positive evidence for current/zero-step success. Preserve separate emitted Model self-loop and full-state-preserving declared stutter evidence. A visible-equal hidden budget change is not a full-state stutter and target compatibility cannot authorize an evidence-free Tick. Add filter-level partial-target controls with distinct reachable budgets without fabricating either and preserve edge/cause predicates. This task proves primitive Reach/Exact behavior; the separate ambiguity prerequisite owns bounded compatible histories and the explicit immutable report contract before non-singleton feature integration. Do not enable a greedy partial-state runtime or estimate hidden budgets here. Keep observation/pending correlation validation ahead of delegation.
6. Switch Kettle to Reach and list its changed outputs per file for review and re-acceptance. Exercise prepare/publication/commit, pending state, failure retry and Reset with the existing transaction controls. No new interpreter or publication path.

## Investigation targets

**Required** (coordinates at d4eff7eedb; reanchor):

- `model/umpire/protocol/runtime/src/runtime.rs:21-138,345-422,581-689` for Projection, judgment, choice-aware replay and commit.
- `model/umpire/protocol/runtime/src/lib.rs:51` for response publication contract.
- `model/umpire/reference/kettle/realization.rs` for an independent consumer.
- `model/umpire/protocol/runtime/tests/support/mod.rs:103` and `tests/proposal.rs` for direct implementers and correlation controls.
- `model/temporal/annotate/syntax/src/realize.rs:968` for generated Realize dispatch.

**Optional:** `model/umpire/protocol/runtime/src/stock.rs:13` and runtime guest manifests for non-Temporal consumers.

## Verification

From model/: `mise exec -- cargo test -p umpire-runtime -p umpire-reference-kettle`; compile the affected macro/generated Temporal consumer and every existing runtime guest using the repository's current guest builder. Run scoped fmt and strict clippy for changed crates. Broader full gates belong to .6.

## Acceptance

- [ ] First-valid and multiple-path cases preserve canonical action/choice order and outcome matching; a short replayable match remains a normal prepared judgment despite longer cyclic frontier, without global uniqueness/exhaustiveness claims. Same-state/different-outcome and distinct-choice controls discriminate.
- [ ] No-match with no remaining evidence-compatible frontier prepares Unexplained; a viable unexamined frontier at depth/work bounds and work exhaustion before resolving canonical precedence return named Refused bounds. Exact/beyond-depth and cycle controls never loop, silently truncate precedence or refuse an already valid first match.
- [ ] Same Model state with different evidence cursors/consumption histories/position remains distinct. A state-only dedup mutant loses the required explanation and fails; merged prefixes require demonstrated future-admission/canonical-precedence equivalence.
- [ ] Typed partial target filter matches distinct legitimately reached full budget states without filling Remaining values; contradiction, unknown/absent/present-zero substitution and evidence-free Tick/full-state-changing stutter mutants fail. Kettle Exact stays available. Non-singleton feature runtime waits for the separate bounded-ambiguity/publication prerequisite; greedy hidden-state commitment is never claimed sound.
- [ ] Positive Current/zero-step, Model self-loop, declared stutter, missing predicate, changed-state stutter and checked-step/catalog/choice failure controls retain their separate outcomes.
- [ ] Failed publication/allocation leaves Model state, pending proposal and evidence unchanged; retry, successful commit, freeze and Reset controls pass. Kettle uses Reach with its golden changes listed and reviewed; other fixtures keep the same checking results (goldens may be re-accepted after review). Reach reuses the suite's search core; no third BFS.
- [ ] Direct/generated signatures and all runtime/Kettle/Temporal guest consumers compile with the compatibility path. Focused tests, fmt and strict clippy pass with command/output evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

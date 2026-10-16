---
satisfies: [R6, R7]
---
# fn-173-reachable-exploration-drop-full-tables.11 Finite record input domains and input-reading native guards

## Description
Complete native finite domain and guard authoring on fn165's settled record inputs; use small framework-only examples. Full six-sample reachable-graph equivalence remains .6's later proof.

**Size:** M
**Files:** framework DSL Action, InputSource, Syntax, Rules tests, small samples and framework-local guidance.
**Touches:** [model/framework/dsl/**, model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Add action-local/per-field finite oneOf domains, including explicit Range expansion, using the record frontend. Do not copy stale token-based design spellings or install global Finite[Int]. Distinguish finite oneOf values from expect's Unit branch alternatives.
- Supply explicit record-field domains before whole-record/per-field Finite lookup: a modifier on an already constructed RecordAction cannot solve missing Finite[Int]. Reuse concrete typed InputField descriptors, including independent domains for two Int fields, without standalone tokens or author-supplied field-name strings. Range expansion preserves its iteration order; collection-valued arguments remain individual values, not implicitly flattened iterables.
- Preserve a selected whole-record witness's correlations, order and actual object identities. Do not silently replace it with a Cartesian product of projections. Make whole-record versus per-field selection modes explicit; reject a call explicitly supplying both until an intersection/order contract is defined. No-override declarations keep their existing selected-witness path.
- Bind typed where((s,in)=>...) through existing rule dispatch and show actual input-dependent enabling/results. Preserve no-input guard behavior and class/default/domain order.
- Pass the same retained actual input to guard and effect, filtering exact class targets before guard evaluation. Preserve ordered short-circuit across state/input conditions and skip effects for false guards. Preserve state-only no-input, pinned-class and heterogeneous-target behavior; refuse unsupported input-reading contexts explicitly rather than through callback casts.
- New explicit oneOf domains reject emptiness and duplicate values when constructed, without sorting or silent deduplication. Ordinary selected Finite metadata stays lazy, including intentional empty/throwing-catalog controls. Installing field domains must not invoke record constructors/default getters; materialized classes, including omitted defaults, must satisfy selected field and whole-record membership before use. An already supplied scalar default may be validated immediately without evaluating it again. Preserve receiver/argument/default evaluation order and exactly-once behavior.
- Reject malformed domain bindings, invalid materialized/default values, wrong guard arity/types and unsupported native DSL forms. Full collection-expression subset/capture refusals remain original .5's obligation: Data evidence or a successful host guard callback does not establish operation liftability, and this task must not add a Scala semantic checker. Native samples demonstrate the settled author surface without reached-graph/Go or export proofs.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/InputSource.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/Rules.test.scala; model/design/bounded-collections.md.

### Verification
Run touched native input/default/rule/expect tests and check/lint selections with nonzero counts. Keep original .5 lifting/IR and .6 graph-isomorphism proofs held.

## Acceptance
- [ ] Explicit finite domains retain order/defaults without global numeric enumeration; malformed domains and out-of-domain defaults reject.
- [ ] Per-field domains work without global Finite[Int]; correlated whole-record witnesses retain their joint membership/order/identity, and unexplained explicit mixed selection modes refuse.
- [ ] New oneOf emptiness/duplicates reject immediately; ordinary metadata remains lazy, domain installation invokes no record/default getter, and materialized omitted defaults are validated before use with valid paired controls.
- [ ] Actual native dispatch depends on both state and supplied record input; wrong guard shapes fail compilation.
- [ ] Guard/effect input identity, exact-class filtering, state/input short-circuit and false-guard effect suppression hold through native facets/derivations without weakening no-input/class-specific/heterogeneous state-only behavior.
- [ ] Domain oneOf and expectation oneOf coexist without ambiguous authoring; native examples/docs and focused tests pass.
- [ ] Full liftability, numeric interpreter parity and six-sample reached-graph equivalence remain unclaimed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

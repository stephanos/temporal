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
- Bind typed where((s,in)=>...) through existing rule dispatch and show actual input-dependent enabling/results. Preserve no-input guard behavior and class/default/domain order.
- Reject empty domains, invalid/default-outside-domain values, wrong guard arity/types and unsupported collection operations. Native samples demonstrate the settled author surface without running reached-graph/Go or export proofs.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/InputSource.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/Rules.test.scala; model/design/bounded-collections.md.

### Verification
Run touched native input/default/rule/expect tests and check/lint selections with nonzero counts. Keep original .5 lifting/IR and .6 graph-isomorphism proofs held.

## Acceptance
- [ ] Explicit finite domains retain order/defaults without global numeric enumeration; malformed domains and out-of-domain defaults reject.
- [ ] Actual native dispatch depends on both state and supplied record input; wrong guard shapes fail compilation.
- [ ] Domain oneOf and expectation oneOf coexist without ambiguous authoring; native examples/docs and focused tests pass.
- [ ] Full liftability, numeric interpreter parity and six-sample reached-graph equivalence remain unclaimed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

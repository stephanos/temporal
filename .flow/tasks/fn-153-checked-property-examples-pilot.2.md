---
satisfies: [R2]
---
# fn-153-checked-property-examples-pilot.2 Declare and lift typed Property illustrations

## Description
Add the typed authoring attachment and lift it into .1's carriers. Use tiny fixtures to prove same-step and transition metadata transport before touching the three production Properties.

**Size:** M
**Files:** `model/framework/Claims.scala`, `model/irgen/Claims.scala`, `model/irgen/test/Fixtures.test.scala`, new scoped `model/irgen/testdata/lifts` fixtures and expected outputs
**Touches:** [model/framework/Claims.scala, model/irgen/Claims.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/lifts/**]

### Approach
- Re-anchor these seams after fn-141. Keep captured values in the constructed declaration exporter if that baseline replaced AST declaration folding; do not revive deleted folding or add a second declaration evaluator.
- Attach while `PropertyBuilder[S,O,F]` retains all types (`framework/Claims.scala:21`), or preserve equivalent typed evidence in the finished declaration. Reuse `Step[S,O,F]` from `Machine.scala:10`. Choose the smallest spelling consistent with the post-chain API. Do not weaken outcome/fact types or require casts; add compile-negative fixtures for cross-machine and mismatched types.
- Carry ordered metadata and each declaration Position with its resolved Property association through the existing Property registration (`irgen/Claims.scala:278`) and source mapping (`Context.scala:264`). Keep explanation text exact. Reject invalid declaration metadata at its nested location.
- Fixture same-step and before/after Properties, omitted metadata, multiple illustrations, duplicate labels, empty explanations and wrong associations. Preserve original IR/Cases in a baseline comparison; fixture changes alone must not introduce production illustrations.

### Investigation targets
**Required:**
- `model/framework/Claims.scala:17` - completed Property versus typed builder
- `model/framework/Machine.scala:10` - concrete Step types
- `model/irgen/Claims.scala:278` - registered Property transport, or its fn-141 replacement
- `model/irgen/Context.scala:264` - declaration source mapping
- `model/irgen/test/Fixtures.test.scala:2502` - scoped admission/lifting fixtures
**Optional:**
- `model/check/Gate.scala` - established schema and fixture generation paths

### Quick commands
```bash
make lint-model
```
Run the lifter fixture runner selected by the current gate, with a test count proving the new fixtures ran. Preserve required compiler roots and the current equality/nullness policies.

## Acceptance
- [ ] Both supported forms author typed illustrations without casts and lift their values, order, owner, labels, explanations and exact source positions.
- [ ] Compile-negative type/owner tests and located metadata refusal fixtures run and pass; missing transition and extraneous same-step before-state follow the parent contract.
- [ ] A Property without illustrations retains its original emitted behavior and identity; unchanged production Model outputs compare to .1's baseline.
- [ ] Attachment uses the current declaration transport and does not reintroduce retired declaration evaluators or add a Scala predicate evaluator.
- [ ] The new lifter fixtures execute under the current compiler roots and model lint passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

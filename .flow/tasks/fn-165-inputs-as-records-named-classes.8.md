---
satisfies: [R3, R5]
---
# fn-165-inputs-as-records-named-classes.8 Native record execution and structural shared-input selectors

## Description
Bind native record inputs through effects/rules and framework-owned shared-input helpers using .7's settled representation.

**Size:** M
**Files:** model/framework/dsl/{Action,Syntax,RuleBindings}.scala, realize helpers and native effect/input tests.
**Touches:** [model/framework/dsl/**, model/framework/realize/**]

### Approach
- Add S=>Steps and (S,In)=>Steps paths sharing the existing rule execution seam. Preserve field values, class restrictions and selected input domains; no second evaluator.
- Replace shared-token identity in generic native helpers with checked field selectors and structural same-label/same-type contracts. Prove a framework-local helper if the historical deadlines consumer exists only in disabled Temporal.
- Add wrong arity/type, missing field, same-name wrong-type and foreign-selector negatives. Preserve source label/field order for later export without implementing capture.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/RuleBindings.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/Effects.test.scala; model/framework/realize.

### Verification
Run touched native Inputs/Effects/Rules/realization selections with nonzero tests and independent old/new step-value pins. Full exporter and Temporal deadlines migration remain held.

## Acceptance
- [ ] Native rule functions read the exact supplied record through (S,In), and no-input rules retain S-only behavior.
- [ ] Structural shared-input helper and selector tests pass without reflection, casts or unchecked string lookup.
- [ ] Wrong fields, types, selector owners and arities reject; focused native regressions pass.
- [ ] No complete R5/Temporal consumer or lifted field-read proof is claimed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

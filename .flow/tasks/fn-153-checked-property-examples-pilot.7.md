---
satisfies: [R2, R4, R6]
---
# fn-153-checked-property-examples-pilot.7 Native typed Property illustrations on the final expect API

## Description
Implement the framework-native attachment/validation portion of .2 without depending on .1's IR carriers. Keep .1-.6 transport, Go classification, rendered pilot, freshness and compatibility proofs open.

**Size:** M
**Files:** framework DSL Claims/Syntax and focused expectation/illustration tests plus framework-local guidance.
**Touches:** [model/framework/dsl/**, model/framework/README.md]

### Approach
- Re-anchor typed optional illustration attachment to fn164's final expect API and fn165's record inputs. Preserve same-step versus transition state evidence, exact action owner, class inputs, outcomes/facts, labels, explanations and ordered metadata without casts.
- Validate omitted/multiple illustrations, missing transition before-state, extraneous same-step before-state, foreign owners, incompatible/domain-invalid values, duplicate/empty labels, empty explanations and unknown/unspecified expectations at native construction.
- Use tiny framework-owned same-step/transition/applicability fixtures; preserve Property predicate authority and distinguish hypothetical illustration from reachable witness evidence. Do not invent a second predicate evaluator or require generated documents.
- Prove no-illustration declarations retain native behavior with independent predicate/selector pins; document the native API locally.

### Investigation targets
**Required:** model/framework/dsl/Claims.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Machine.scala; model/framework/dsl/Witness.test.scala; model/framework/dsl/Expect.test.scala.

### Verification
Use isolated native compiler/runtime and syntax checks with external-author negative fixtures and nonzero test counts. No IR/schema/Go/Temporal or generated pilot gates.

## Acceptance
- [ ] Same-step and transition illustrations attach through final expect API with exact typed data, association and order.
- [ ] Native type/domain/owner/metadata refusals cover the listed invalid cases; omitted metadata preserves independent native Property behavior.
- [ ] Local examples clearly distinguish illustrations from witnesses and tests pass.
- [ ] No full R2 transport, R3 classifier, R5 rendered document, R6 pilot or whole-spec completion is claimed.

## Done summary
Blocked:
Owner-ordered native DSL queue. Release only after fn-123.10's native fault/durability checkpoint is committed on umpire. Cross-spec task edges are unsupported; this admission hold records queue order without IR-bearing spec closure. Typed illustrations use the final expect/record/domain APIs. IR/Go/backend work and all model/temporal inputs remain held.
## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R2, R3, R4, R6, R7, R8]
---
# fn-164-expect-blocks-for-properties.10 Finish native expect callers, public API and colon authoring

## Description
Finish the framework-native portions of R2-R4/R6-R8 after .2. Original .3/.7-.9 retain their exact IR, exporter and complete migration proofs; this checkpoint cannot close them.

**Size:** M
**Files:** model/framework/dsl/{Claims,Syntax,Machine}.scala, native tests, realize, samples, check/lint and framework guidance.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/samples/**, model/framework/check/lint/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Refresh .flow/tmp/fn16410-native-caller-inventory.md after committed .2. At checkpoint 8d45f95573 it pins 33 native finisher calls (22 sample Properties, six IrFileDiscovery callers, five internal pattern constructors), with live .2 additions explicitly unstable. Account for every author caller, internal constructor, core oracle and intentional negative fixture. Migrate author callers to expect, preserving predicates, explicit contextual names, exact selectors and native truth/exception behavior. Keep direct core oracles independent.
- Remove public holds/holdsAcross together, including renamed/deprecated compatibility routes. Retain private/direct core construction without core importing sugar. Disabled Temporal and held exporter fixtures do not constrain the final public API.
- Convert native authoring examples to colon blocks using the pinned formatter and existing scanners. BlockFormRule currently scans only model/temporal and recommends braces; add explicit framework-only source roots and colon diagnostics before treating it as native enforcement. Test positive executed scan counts, a rejected native optional-brace block and accepted language-required braces, comments/strings and deliberate invalid fixtures. Check formatting twice for idempotence. Never restore Temporal sources to make this scanner run.
- Update Core form comments and framework-local migration guidance. Deferred export fixtures adapt later; do not keep a public legacy API merely to compile held IR tests.

### Investigation targets
**Required:** model/framework/dsl/Claims.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Witness.test.scala; model/framework/dsl/Effects.test.scala; model/framework/check/lint/BlockFormRule.scala; model/framework/check/lint/SyntaxRule.scala.

### Verification
Use .2's isolated native compile/test recipe. Run touched Expect, Witness, Effects, Rules and lint selections with nonzero executed counts; compare old independent core predicates with new expect predicates on explicit truth/exception cases. No make check-framework, capture/IR, Go, schema, regeneration or model/temporal inputs.

## Acceptance
- [ ] Native callers use expect or inventoried independent core constructors; external author-package compile negatives reject holds/holdsAcross and compatibility routes.
- [ ] Native equivalence pins preserve selector/parameter identity, predicate truth values, holes and short-circuiting across witnesses, patterns and monitors.
- [ ] Colon-style scanner visits nonzero native source inputs and rejects reintroduced optional brace DSL blocks without comment/string or language-required-brace false positives; second formatter pass is unchanged; guidance matches final API.
- [ ] Focused native/check suites pass; original .3/.7-.9 exact IR obligations and all Temporal migration remain open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

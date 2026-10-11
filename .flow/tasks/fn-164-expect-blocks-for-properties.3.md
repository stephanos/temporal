---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-164-expect-blocks-for-properties.3 Lift expectation predicates with exact core IR and temporal bridges

## Description
**Size:** M
**Files:** context recognition, claim/predicate lifting and positioned fixture harness.
**Touches:** [model/irgen/Context.scala, model/irgen/Claims.scala, model/irgen/Syntax.scala, model/irgen/Expressions.scala, model/irgen/Declarations.scala, model/irgen/Monitors.scala, model/irgen/Capabilities.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/expect/**, model/irgen/testdata/lifts/**, model/framework/Syntax.scala, model/framework/*Expect*test.scala]

Implement R4/R5's lifting and reusable predicate bridges after the runtime task. Do not migrate production callers until the paired core fixtures pass.

### Approach
- Extend existing block recognizers deliberately: inline Property bodies and reusable generic capability predicates are legal; machine-section-only is/effect placement does not accidentally apply to all expectation arguments. Centralize expectation statement validation/lowering behind existing source-aware claim registration.
- Lower requirements and alternatives without identity literals, preserving association, if/else versus implication shape, eager helper-let placement and function reference/parameter identities. Recognize pure forwarding single-check blocks as the forwarded core predicate instead of introducing a wrapper. Explicit contextual names reify original parameters; do not normalize parameter names away in comparisons.
- First pin heartbeat-shaped AND/implication and reset-shaped eager restarted let plus six/three alternatives against exact ordinary-core twins. Include named defs, `s`, `(b,a)`, compiler placeholders, nested conditionals and branch conjunctions. Compare complete functions/declarations with an explicit source-span map; helpers that dereference function references alone do not prove whole-IR equality.
- Implement and lift ordinary function predicate bridges for never, stays.unless, sticky, stickyAcross and Monitor.readAfter, inline and named; retain S=>Boolean predicates for once/stays/from/leadsTo. Include generic Declares capabilities and composition fixtures, and reject wrong-level uses rather than treating a path claim as a Boolean.
- Reuse exact owner-specific composed fact binding, retaining machine type checks and verifying selectors with shared state types/two same-named facts. No suffix matching or all-member widening.
- Located refusals cover empty block/branch, insufficient alternatives, bare final Boolean, malformed statements, nested unsupported forms and wrong-level expressions. Preserve existing errors and source placement.
- Record the settled sugar contract for fn-141's later generic definition expansion; no permanent independent lowering implementation is exempt from that refactor.

### Investigation targets
**Required:** `model/irgen/Context.scala:69-106`; `model/irgen/Declarations.scala:161-173`; `model/irgen/Claims.scala:274-302`; `model/irgen/Syntax.scala:201` and composed fact helper; `model/irgen/Expressions.scala:271`; `model/irgen/test/Fixtures.test.scala:76-109`.
**Optional:** `model/framework/{Syntax,Monitor,Assume}.scala`; `model/temporal/features/activity/standalone/protocol/{Heartbeat,Protocol}.scala` (read-only proof examples).

### Quick commands
```sh
mise exec -- scala-cli test model/irgen --test-only '*Expect*'
mise exec -- scala-cli test model/irgen --test-only '*Fixtures*'
make lint-model-syntax
```

Prefer a focused suite without eagerly rebuilding every fixture for iteration, but preserve and run canonical focused Quick obligations. All output candidates are scratch, not production regeneration.

## Acceptance
- [ ] Full paired fixture IR is identical except explicitly mapped positions, including heartbeat/reset trees, eager lets, parameter names and named-def forwarding (R5).
- [ ] All listed temporal/pattern/monitor predicate bridges construct and lift inline/named/generic/composed expectations without changing outer semantics (R4).
- [ ] Source-located invalid-block, branch, statement and wrong-level refusals are covered; exact composed field/fact binding is independently tested (R1-R4).
- [ ] Runtime API preservation choices are proved and recorded for migration/fn-141; no generic normalization or function-wrapper exemption substitutes for equality (R5).
- [ ] Quick commands pass and managed artifacts/schema/Go remain unchanged.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

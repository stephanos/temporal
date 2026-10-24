---
satisfies: [R10]
---
# fn-189-the-reify-plugin-keeps-only-the-body.10 After .3/.9: one tree-normalization pass; delete typedRemainder (R10)

## Description
**Recut 2026-10-18 (one-language series):** 'Go evaluates a role pattern the same way' becomes: the Scala checker and the Scala emitters (fn-206/207) evaluate it the same way.

Touches: model/framework/reify/body/Expressions.scala, BodyEngine.scala, BodySyntax.scala, BodyContext.scala, Constants.scala

### Description
Two body-lifter simplifications that change no DSL and no IR. (1) One normalization pre-pass over a term before `lift` sees it: strip `Typed`, `Inlined(_, Nil, _)`, `Block(Nil, _)` and `NamedArg`; substitute the synthetic argument vals a call with out-of-order named arguments arrives in (`arguments`/`synthetic`); drop `copy$default$N` and other `$default$` getters where the lifter today recognises them; keep the original tree for diagnostics (the `Inlined` call is what names a sugar in a refusal under R7). Replace the local `unwrapped` copies (`through`, `forwardedExpectation`, `factoryConstruction`, `stepFactsView`, `BodyEngine.unwrapped`, `Constants.resolve`'s wrapper arms) and the wrapper arms at the head of `lambda`, `forwardedDef`, `fieldPath`, `blockCall`, `standaloneExpectation`, `requirementStatements`, `requirement`, `stepFunctionOrigin`, `alternative` and `lift` itself with the one pass. (2) Delete `typedRemainder`: a final `case _: Closed` that exhausts the enum is lifted as the role pattern `rolePattern` already produces, not rewritten to a wildcard; Go evaluates a role pattern the same way. The three refusals `typedRemainder` raised ("typed remainder has a binding/guard", "overlaps", "incomplete") go with it; the compiler's exhaustiveness check under the strict flags covers the incomplete case.

Do this after .3 (the expansion engine is the largest consumer of the wrapper arms) and after .9 (fewer parameters to carry through the pass). If fn-190.9 has reshaped `lift` by then, apply the pass to the reshaped lifter; the list above names today's sites, not a contract.

**Size:** M

### Verification
- The six samples and the framework fixtures lift to the same IR as before this task (positions aside, per .9), including a Model whose match ends in `case _: Closed`.
- Refusal texts and positions of the existing refusal fixtures are unchanged, except the three `typedRemainder` refusals, whose fixtures are retired.
- grep shows one definition of the wrapper-stripping pass in `reify/body` and no `typedRemainder`.
## Acceptance
- [ ] One normalization pass strips wrappers, synthetic argument vals and default getters; no other `unwrapped`-style helper remains in `reify/body`.
- [ ] Sugar refusals still name the sugar from its call site.
- [ ] `typedRemainder` is gone; an exhaustive final `case _: Role` lifts as a role pattern and the sample IR is otherwise unchanged.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
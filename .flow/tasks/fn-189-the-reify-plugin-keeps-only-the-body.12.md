---
satisfies: [R11]
---
# fn-189-the-reify-plugin-keeps-only-the-body.12 After .5/.10: helpers inline at the call site; roots keep their names; remove namedCopy, the function table and the cardinality proof (R11)

## Description
**Recut 2026-10-18 (one-language series):** 'Go reachable graph unchanged' becomes 'the Scala checker's reachable graph unchanged'.

Touches: model/framework/reify/body (Expressions: `callee`, `defCallee`, `call`, `forwardedDef`, `choose`, `namedCopy`, `functionCopies`; BodyContext: `functions`, `reachableFunctions`, `lifting`), model/framework/reify/plugin (store and capture: the stored body and its splice), model/framework/dsl/Reify.scala (`liftedIr` payload), export/testdata reify fixtures, model/README.md plugin section

### Description
Phase 2, after .5 (one lifting rule and one carrier) and .10 (the normalization pass). Implement R11: a lifted body's call of a non-root def of the lifted sources is replaced by the def's body with one `Let` per parameter bound to the lifted argument, in parameter order; a root (effect, transition, claim member, monitor function, `end`, `evidence`, refinement map) keeps its name and its `ir.Function`. The store phase stores a covered def's body and parameter list as its `liftedIr`; a capture that reaches a def in another file splices the stored body. Delete `namedCopy`, `functionCopies`, `reachableFunctions` and the cross-function cardinality proof; a `choose` alternative that calls a helper is checked on the inlined steps with the existing "not one step written out" refusal. Keep the recursion refusal through the inline stack. Refuse `require` inside a helper with the R11 text; keep `require` on roots as `ir.Function.requires`. Update the plugin section of model/README.md and the fixtures that asserted a helper `ir.Function` or a `$choice` copy. Record the body-lifter line count in the counts receipt.

**Size:** M

### Verification
- The six samples compile with -Xplugin and lift; the Scala checker's reachable graph of each is unchanged from before this task (positions aside).
- The warm-Bloop test of R2 still recompiles a dependent Model when only a helper's body changes.
- A fixture with a helper called from two `choose` alternatives lifts both alternatives' steps named, with no `$choice` function; a fixture with `require` in a helper fails with the R11 text.
- grep shows no `namedCopy`, `functionCopies` or `reachableFunctions` in model/framework/reify.
## Acceptance
- [ ] No `ir.Call` or `ir.Function` is emitted for a non-root helper; roots keep their names and functions.
- [ ] `namedCopy`, `functionCopies`, `reachableFunctions` and the cardinality proof are gone; a helper called from a choose alternative lifts named steps inline.
- [ ] `require` in a helper is refused with the R11 text; the warm-Bloop invalidation test and the six samples pass with an unchanged reachable graph.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
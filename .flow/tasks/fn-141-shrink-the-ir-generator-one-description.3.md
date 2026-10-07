---
satisfies: [R3, R4]
---
# fn-141-shrink-the-ir-generator-one-description.3 Generic sugar expansion: function-body sugar is lifted from its definition (proof for Part A)

## Description
Part A, first half. The lifter stops matching `enter`, `stay`, `reject`, `disabled`, `in`, `implies` and `records` by name. One expansion replaces a call of a sugar definition with that definition's body, the call's arguments bound, and the ordinary lift reads the result.

**Size:** M
**Files:** `model/irgen/{Lift,Context,Expressions,Syntax,Claims}.scala`, `model/umpire/Syntax.scala`, `model/irgen/testdata/lifts/Sugar.scala`, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/irgen/**, model/umpire/Syntax.scala]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Index the framework's and the kit's TASTy too. Today `lifted` drops every `umpire/` entry; keep that exclusion for roots and for the lints, and lift it for the definition index.
- What marks a definition as sugar: the syntax lint already holds sugar to the `Syntax.scala` files, so membership of such a file can be the mark. Decide between that and an annotation, and record it.
- Reuse the binding the lifter already has for helper defs of the Models (`bodyOf`, `binding`, `boundValues`, `boundFunctions`, `boundTypes`).
- Positions: every IR node from an expanded body takes the call's position, as the hand lowering gives today.
- Add the small generic reductions the bodies need, each one a rule about core Scala and none about a sugar: a field read of a known constructor call (`ok.outcome` of `Ok(o)`), a varargs list built from `first` and `rest`.
- Rewrite a sugar body where it is outside the liftable subset. The composition form of `records` tests a suffix at run time while its lowering tests a composed key; give it a body the lifter can read and the JVM can run, with the same answers.
- Refuse a sugar body that cannot be lifted, naming the sugar and its definition's position, and a sugar that reaches itself.
- Delete the matching arms of the lifter's `sugar` as each form moves.

### Investigation targets
**Required:**
- `model/irgen/Syntax.scala` (`sugarCall`, `sugar`, `outcomeOf`, `composedFact`)
- `model/irgen/Claims.scala` (`declaring`, `bodyOf`, `valued`) and `Context.scala` (`binding`)
- `model/irgen/Expressions.scala` (`lift`: where `sugared` is asked first; `call`, `callee`)
- `model/irgen/Lift.scala` (`lifted`: the `umpire/` exclusion)
- `model/umpire/Syntax.scala` (each definition's body against its `Core form:` comment)
- `model/check/SyntaxRule.scala` (what the lint already guarantees about sugar files)

## Acceptance
- [ ] No lifter code names `enter`, `stay`, `reject`, `disabled`, `in`, `implies` or `records`.
- [ ] The sugar fixture lifts both spellings to the same IR as before, positions included.
- [ ] A fixture kit defines a sugar of its own; a Model using it lifts with no lifter change.
- [ ] Reject fixtures: an unliftable sugar body (named with its definition's position) and a sugar cycle.
- [ ] The framework's munit tests pass: each rewritten body gives the same run-time answers.
- [ ] `scala-cli test model/irgen` and `make umpire-check-model` pass; `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

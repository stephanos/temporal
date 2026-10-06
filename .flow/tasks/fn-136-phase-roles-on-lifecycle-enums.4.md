---
satisfies: [R8]
---
# fn-136-phase-roles-on-lifecycle-enums.4 in[R] and p.is[R]: short role-test spellings, lifted like in(...) and isInstanceOf

## Description
Adds the two spellings that let callers read a role directly: `in[R]` in rule headings and `p.is[R]` on phase values. Framework, lifter and fixtures only. No Model changes yet, so model/ir does not change.

**Size:** M
**Files:** model/umpire/Syntax.scala (`in[R]` on `Rules`, `is[R]` extension, each with a `Core form:` comment), model/check/SyntaxRule.scala (sugar list), model/irgen/Syntax.scala (recognize `umpire.Rules.in` type-arg form, around :24 and :52-60), model/irgen/Expressions.scala (`is[R]` beside the `isInstanceOf` lowering from task .1), model/irgen/Roles.scala (reuse the role closure), model/irgen/testdata/lifts/Roles.scala + expected/roles.json, model/irgen/testdata/lifts/Rejects.scala + expected/rejects.txt, model/irgen/test/Fixtures.test.scala, model/umpire/Rules.test.scala
**Touches:** [model/umpire/Syntax.scala, model/umpire/Rules.test.scala, model/check/SyntaxRule.scala, model/irgen/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `in[R](using TypeTest[P, R])` is a third `in` overload on `Rules`, next to `in[Q](first, rest*)` (:322) and `in(set)` (:326). The planning probe on Scala 3.9.0 showed the three overloads resolve unambiguously (`in[Closed]`, `in(a, b)`, `in(pred)`).
- `extension [P](p: P) def is[R](using TypeTest[P, R]): Boolean`. Keep it distinct from fn-135's `is { }` block, which takes no receiver.
- Lifter: lower `in[R]` through the rules' projection to the role's case set, reusing the `in(...)` → OP_CONTAINS path and the role-closure module. Lower `p.is[R]` exactly as `p.isInstanceOf[R]` is lowered in task .1.
- Compile-time refusal of `in[R]` in rules with no projection: with the defaulted phase type a `TypeTest` may still be synthesized, so `TypeTest` alone won't refuse. Add evidence that the phase type is not the no-projection default (fn-137 plans the same `P` is not `Nothing` evidence for both other `in` forms; share it).
- If fn-137 has landed, the projection comes from `Phased`. Otherwise it comes from `Rules(_.phase)`. Write against whichever the tree has.

### Investigation targets
**Required:**
- model/umpire/Syntax.scala:300-365: `in` overloads and `PhasesOf`
- model/irgen/Syntax.scala:20-60: `in` recognition and lowering
- model/irgen/Roles.scala and the `isInstanceOf` branch added by task .1

### Acceptance
- [ ] Fixture: `in[Closed]` lifts to JSON identical to `in(<Closed cases in declaration order>)`, and `s.phase.is[Live]` to the same IR as `s.phase.isInstanceOf[Live]`, including an inherited role.
- [ ] Refusal fixtures: `in[R]` in rules with no projection, `is[R]`/`in[R]` against a non-role, and on a non-enum value, one message each.
- [ ] Rules.test.scala: `in[R]` fires in exactly the role's phases.
- [ ] No unchecked warning under `-Werror`; `make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model` and `make lint-model` pass with model/ir unchanged.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Acceptance
- [ ] TBD

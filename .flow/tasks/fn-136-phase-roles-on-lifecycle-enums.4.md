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
Adds the short role-test spellings, `when[R]` for rule cases and `p.in[R]` on phase values (not `p.is[R]`; reason below), lifted to exactly the IR of the hand-written `in(...)` and of `isInstanceOf[R]`.

stage: impl-review - skipped(config: REVIEW_MODE=none - DSL batch: reviews run once at the batch's end)
Tier: lane B, IMPLEMENTER claude-opus-5-5 at high

### Spelling decision: `p.in[R]` instead of `p.is[R]` (owner unavailable; my recommendation)
`extension [P](p: P) def is[R](using TypeTest[P, R])` cannot sit beside fn-135's top-level `is[S, O, F](using Owner)(body: View[S] ?=> Boolean)`. With both in scope, `is` is overloaded. Overload resolution then types the block's argument without its context-function type, so every `is(phase == ...)` / `is { flag }` fails with "No given instance of type View[...]", and `is(true)` silently picks the extension. I checked this on Scala 3.9.0 with a probe of the real signatures.

The only placements that keep `is` (a member of `Finite`, or a given in `Machine`) break the sugar-layering lint, since a core file would define or name sugar. Weakening that lint is not my call.

`p.in[R]` reuses the membership name the owner already approved for roles (`in[R]` was R8's rule form), and it reads as fn-139's remaining meaning of `in`: membership. It overloads cleanly with `p.in(a, b)`, both at the top level and inside `Rules`. Rule cases use `when[R]` as the batch order says. If the owner wants `is`, the lifter needs only the sugar name changed. The framework needs one of the lint exceptions above.

### What changed
- `model/umpire/Syntax.scala`: `Rules.when[R](using TypeTest[P, R], ClassTag[R], ProjectsPhases[P])`. The `ClassTag` names the role in overlap messages (`when[Closed]`). `extension [A](value: A) def in[R](using TypeTest[A, R])` is defined at the top level and as a `Rules` member, beside the existing pair. `ProjectsPhases[P]` is evidence that the rules declare a projection (`NotGiven[P =:= Nothing]`), with an `implicitNotFound` naming the fix. It was named `Projected` first, but that name is used in `umpire/realize`, so the syntax rule refused it. fn-137 can share it for the other `in` forms.
- Lifter: `Declarations` adds `Heading.Role`, lowered to `OP_CONTAINS(projection, <role's cases>)` through `PhaseRoles.roleSet`. It refuses `when` in rules without a projection ("when names the phases of a role, and X's rules declare no projection: `Rules(_.phase)`"), and a derivation still refuses any phase heading. `irgen/Syntax.sugar` lowers `x.in[R]` through `roleTest`, the same code as `isInstanceOf`. `Order.appliesAtOnce` now includes `when`.
- `model/check/SyntaxRule.scala`: `when` and `ProjectsPhases` were added to `sugarNames`.
- Fixtures: `Shortened` (`when[R]`, `p.in[R]`) against `Written` (hand-written `in(...)`). Its states mirror `Roled`/`Listed`, so `p.in[R]` matches `isInstanceOf[R]`, including the inherited `Waiting` and the model role `Expired`. The roles test requires identical machines and functions, but for names and positions. Refusals, one line each: `when` with no projection (lifter; compile time in `Rules.test.scala`), `when`/`in` against a non-role, and `when`/`in` on a non-enum value.
- The non-enum refusal of `.1` now suggests `s.phase.in[R]`, since the Models' lint forbids `isInstanceOf`.

### Expected IR delta at the batch regeneration
None. The Models' lift is byte-identical to the pre-lane baseline.

### For later tasks
- **fn-136.2/.3/.5:** write role tests as `p.in[Closed]` (and rule cases as `when[Closed]`). `p.isInstanceOf[...]` is refused by `lint-model-models` (DisableSyntax). `p.in[R]` lowers byte-equal (but for positions) to `p.in(<cases in declaration order>)`, and `when[R]` to `in(<cases>)`.
- **fn-139 (parallel lane):** it adds `when(p1, …)`/`when(set)` as overloads beside this `when[R]`. Both lanes append `"when"` to `SyntaxRule.sugarNames`, so keep one. The lifter's `heading` match gains a `("when", List(List(test, _, _)))` arm for the role form. fn-139's `when(...)` arms must not shadow it (their argument lists differ). Its "no projection" refusal for `when(...)` could reuse `ProjectsPhases` and this message.
- **fn-137:** `ProjectsPhases[P]` is the shared "P is not Nothing" evidence the plan asked for.
- `p.is[R]` from the spec's API Contracts (R8) is not provided. The spec/R8 text and docs should say `p.in[R]`. That is for the conductor or owner to apply.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 98d64663e8
- Tests: baseline: green via fn-136.1's final run at 985f65ed5 (same lane, same commands), mise exec -- scala-cli test model/project.scala model/umpire (umpire_tests=0; includes RulesTest 'when[R] fires in exactly the phases with the role R' and its compile-time no-projection refusal), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check (green after renaming the evidence to ProjectsPhases), UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test model/irgen (expected/roles.json grown by Shortened/Written; 5 new rejects.txt lines; no existing roles.json declaration changed), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (suite_rc=0, 99 passed), Models lift into a scratch dir: diff -r against the pre-lane baseline lift is empty (no model/ir change), scala-cli fmt --check model/project.scala model/umpire model/temporal model/irgen model/check: clean, make lint-model-irgen, lint-model-irgen-lifts, lint-model-models, lint-model-check, lint-model-syntax: rc=0 each (one by one), GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: no model gate per task (worker-notes.md)
- PRs:
## Acceptance
- [ ] TBD

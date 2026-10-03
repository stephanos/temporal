---
satisfies: [R1, R2, R5, R16]
---
# fn-112-make-the-standalone-activity-scala.3 Add machine derivation and canonical step helpers

Touches: [model/umpire/Machine.scala, model/umpire/Domain.scala, model/umpire/Syntax.scala, model/lifter/Declarations.scala, model/lifter/Expressions.scala, model/lifter/Syntax.scala, model/lifter/test/**, model/lifter/testdata/**, model/gate/**, model/.scalafix.conf, model/.scalafmt.conf, tools/umpire/internal/golden/**]

## Description
Implement the framework/lifter surface for behavior-preserving machine derivation and compact step expressions, and land the function projection of R1 that every later lowering needs.

**Size:** M
**Files:** model/umpire machine DSL (core) and `model/umpire/Syntax.scala` (new: the step sugar); model/lifter Declarations.scala, Expressions.scala and `model/lifter/Syntax.scala` (new: sugar matching); dedicated positive/refusal fixtures; `tools/umpire/internal/golden/original.go` and its tests (function projection).

### Approach
- **Harness first (R1, Edge Cases "Behavior is frozen").** `MatchOriginal` (`tools/umpire/internal/golden/original.go:252`) runs `proto.Equal` over the whole projected Model, so function bodies and the `functions` inventory are compared; `in` (`OP_CONTAINS` where `||` chains lift to `OP_OR`/`OP_EQ`), the task-4 patterns and `accept`/`stay` (which retire named private helpers from the inventory) would all fail it. Extend `ProjectBaseline`/`ProjectCurrent` to project out function bodies, the `functions` list and the names by which steps, Properties, monitors and evidence refer to functions, and prove function meaning through the derived outputs `model.json`/`lower.json` already digest (step tables, Property verdicts over every row, monitor and evidence catalogs, refinement rows, Query answers, fingerprints) and Case bytes. Add a mutation control: a semantically changed function body (one flipped guard) still fails. Do not widen anything else; Definition-ID probes and the "do not widen" rule of task 1 are untouched.
- Follow existing restrict-copy lifting for rebind, extend and unmonitored (which drops monitors and the refinement together with its visibility projections, as `restrict` does). Preserve starts, ends, results, assumptions, monitors and refinement metadata; replace only bound actions, add only unbound actions and retain binding order.
- Add `refining(product)(map)` to replace only the source product/map while preserving visible facts/outcomes, and `assuming(as*)` to append unique assumptions in declaration order.
- Add accept, disabled, stay, finite membership, implication and fact-recording declarations that lower directly to existing IR, following the operator policy in the spec's Decision Context (`.plans/DSL_OPERATORS.md`): words, no new symbol, no `inline`/macro, each a plain `def`/`extension` the lifter matches by name.
  - `a implies b`: `extension (a: Boolean) infix def implies(b: => Boolean)`, lowered to `or(not a, b)` so the by-name right side is read only when the left holds (a hole in `b` is not reached when `a` is false). Alphabetic infix has the lowest precedence, so `a == b && c implies d` reads as intended. No `iff`, no `and`/`or`/`not`.
  - `phase.in(a, b, c)`: `extension [A](a: A) def in(first: A, rest: A*)`, dotted (never infix), lowered to `OP_CONTAINS(a, list(first, rest...))`, the `a in b` SEMANTICS.md already defines. The signature forbids `x.in()`.
  - `step.records(fact)`: dotted like `contains`, lowered to `OP_CONTAINS(fact, field(step, facts))`, exactly what `s.facts.contains(f)` lifts to today. This is the definition fn-112.4's composition form `after.records(_.member, fact)` shares.
- **Core and sugar (spec Architecture; Decision Context "Core and sugar").** `accept`, `stay`, `disabled`, `in`, `implies` and `records` are sugar: they go in `model/umpire/Syntax.scala`, each with a doc comment naming the core form it stands for (`List(Step(m.accepted, state, facts))`, `List(Step(m.accepted, s, Nil))`, `Nil`, `List(a, b, c).contains(x)`, `!a || b`, `s.facts.contains(f)`); `Machine.scala`/`Domain.scala` (core: `rebind`, `extend`, `refining`, `assuming`, `unmonitored`, `Step`, `because`) import nothing from it. Their lifter matching goes in `model/lifter/Syntax.scala`, reached from `Expressions.scala`/`Declarations.scala` through one hook each, and lowers to the IR the core spelling produces; one fixture per sugar form declares the sugar and its core spelling side by side and the expected JSON proves the trees equal.
- The core/sugar rule is enforced by `make lint-model` from this task on (spec R16 errors): a lint rule (in the gate's lint stage or a scalafix rule under `model/`) fails on a definition in a `Syntax.scala` that is not documented with its core form, on a sugar-named definition (`implies`, `in`, `records`, `accept`, `stay`, `disabled`, the patterns, `:=`, `sticky`) outside a `Syntax.scala`, and on an import of a `Syntax.scala` from a core file of `model/umpire`, `model/temporal/realize` or `model/lifter`. Tasks 4, 5 and 9 and fn-114.4 add their forms under the same rule.
- A step function arm that is a wildcard (`case _ => Nil`) lifts as today; the match pattern kind is already in the IR and fn-120.3's lint reads it (`.plans/MODALITIES.md` H1). Do not refuse or rewrite it here.
- Refuse rebind of an unbound action, extend of a bound action, duplicate assumptions, incompatible refinement state/map types, duplicate bindings and cyclic aliases with located diagnostics. `in()` with no member is a compiler refusal by signature.
- Prove derived Definition IDs and complete tables against the task-1 archive.
## Acceptance
- [ ] The harness projects function bodies, inventory and references out of the IR text check, proves function meaning on the derived outputs and Case bytes, and a mutation control (flipped guard) fails; nothing else is widened.
- [ ] rebind, extend, refining, assuming and unmonitored lift without a new IR field and preserve all untouched machine metadata and ordering; `unmonitored` drops monitors, the refinement and its visibility projections.
- [ ] accept/because, disabled, stay, in, implies and records lower to the existing step/expression IR: `implies` to `or(not a, b)` with a by-name right side, `in` to `OP_CONTAINS` over a list literal, `records` to `OP_CONTAINS` over the step's facts; no new IR node.
- [ ] `implies` is an `infix` extension of `Boolean`; `in` and `records` are dotted, `in` takes at least one member by its signature; none is `inline`/`transparent inline`; no `and`/`or`/`not`/`iff` word is added and no new symbolic operator appears.
- [ ] The six sugar forms live in `model/umpire/Syntax.scala` with their core form documented, their matching in `model/lifter/Syntax.scala`, core files import no `Syntax.scala`, and one fixture per form proves IR equality with its core spelling (including the short-circuit case where the right side of `implies` is not read).
- [ ] `make lint-model` fails on a sugar definition outside a `Syntax.scala` and on a core file importing one (a negative fixture or test proves each).
- [ ] Positive and refusal fixtures cover every new form, refinement replacement, assumption append and invalid binding case; a wildcard step-function arm lifts unchanged.
- [ ] Full tables, IDs, fingerprints and Query answers equal the original baseline under the R1 projection.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

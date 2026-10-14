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
Added machine derivations, the six step/claim sugar forms with their core/sugar split and lint rule, and the R1 function projection of the original-baseline harness. No production Model changed; model/ir and model/cases are byte-identical.

**What changed**
- Harness (`tools/umpire/internal/golden/original.go`): the IR check drops `Model.functions` and reads each set function reference (`Call.function`, `StepBinding.function`, `Machine.evidence`, `Refinement.map/visible/visible_outcomes`, `Monitor.next/violated/after`, `Property.holds`, `Progress.from/to`) as one token; empty stays empty, so presence and the monitor evaluation oneof are still compared. Meaning is held by the existing derived outputs and Case bytes. `TestOriginalBaselineRejectsAFlippedGuard` is the mutation control: a flipped guard passes the IR check and fails the semantics digest; a renamed, double-negated function passes both. `lower.json` was re-derived from the unchanged archive (exploration candidate digests use the projection); model.json, owners.json and archive/*.gz are untouched.
- `model/umpire/Machine.scala` (core): `rebind`, `extend`, `refining(product)(map)`, `assuming`, `unmonitored`, plus `steps.because("…")`. `restrict` now lifts through the same chain.
- `model/umpire/Syntax.scala` (sugar, each doc says `Core form:`): `Accepted[O]`, `accept`, `stay`, `disabled`, `x.in(first, rest*)`, `a implies b` (infix, by-name), `step.records(fact)`. Lifted in `model/lifter/Syntax.scala` through one hook in `Expressions.lift` (plus a guard in `stepFunction`).
- Lifter refusals: rebind of an unbound action, extend by a bound one, a doubled binding, a repeated assumption, `refining` with no refinement or with a product of other types, a machine derived from or aliased to itself (both used to recurse forever), `in` over a splatted list, `because` on steps not written out, an `Accepted` given that is computed. Compiler refusals: `x.in()` and a cross-typed `refining` map (crossed fixture).
- Fixtures: `lifts/Derived.scala` and `lifts/Sugar.scala` pair each form with its spelled-out core, and new tests in `Fixtures.test.scala` require equal IR after dereferencing function names. This covers the `implies` short-circuit over a hole and a wildcard arm.
- Lint (`model/gate/SyntaxRule.scala`, `gate --check-syntax`, `lint-model-syntax` in `make lint-model`). It refuses three things:
  - an undocumented definition in a Syntax.scala;
  - a sugar-named top-level, object or extension definition outside one;
  - a core file that imports Syntax names, or, in umpire, refers to them.

  `model/metrics/Metrics.scala` lists the new gate file.

**Decisions (autonomous)**
- **`accept`/`stay` read the accepted outcome from `given Accepted[O] = Accepted(Outcome.accepted)`.** This is typed and explicit, with no case-name convention and no inline derivation. It costs one line per outcome type in task 6.
- **`refining` requires the source to declare a refinement, and the replacement to have the same state, outcome and fact types.** The preserved visibility projections name those types.
- **`because` is core,** as the spec classifies it: an extension on `List[Step]` in Machine.scala.
- **`Delta.Functions` (function-name substitutions) is removed.** The projection subsumes it.
- **No Declarations sugar hook.** No task-3 sugar form is a declaration; task 4's patterns add it.
- **No new expected JSON.** The fn-115 inventory is closed, so the pairs are assertion tests, as in fn-112.2.
- **Outside Touches:** `model/lifter/Context.scala` (alias-cycle guard), `Lifting.scala`, `Makefile`, `model/metrics/Metrics.scala`, `model/README.md`, `tools/umpire/model/original_migration_test.go` and the crossed fixture.
- **Parallel work:** two opus subagents built the Go harness and the lint rule.

**Review:** claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`; the writer and reviewer are the same family (Opus).
- Round 1: SHIP with 2 P3s, both fixed in 808e2e3499: the step-binding parsing is shared, and `restrict` chains with the other derivations.
- Round 2: SHIP.

**Deferred P3/FYI:** an IOException inside `--check-syntax` prints a stack trace instead of a GateError (reviewer confidence 50).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d9d0609caf, cda3b19165, 808e2e3499
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), scala-cli test model/lifter (exit 0), scala-cli test model/gate (exit 0), go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 193 s), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs:
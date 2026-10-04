---
satisfies: [R9, R14, R16]
---
# fn-112-make-the-standalone-activity-scala.5 Add captured typed action inputs and bounded counters

Touches: [model/umpire/Action.scala, model/umpire/Domain.scala, model/umpire/Syntax.scala, model/lifter/Declarations.scala, model/lifter/Expressions.scala, model/lifter/Syntax.scala, model/lifter/Types.scala, model/lifter/test/**, model/lifter/testdata/**]

## Description
Implement the settled input-token syntax and field-local finite counter bounds without changing state keys.

**Size:** M
**Files:** model/umpire Action/input/Finite APIs (core) and `model/umpire/Syntax.scala` (`:=`, sugar); model/lifter declaration/expression/finite lifting and `model/lifter/Syntax.scala` (`:=` matching); focused fixtures.

### Approach
- Add captured `input[A]` tokens, Action declarations that consume them and `token := value` calls. Reorder supplied tokens into declaration order and default omissions to the first finite value.
- `:=` is the only symbolic operator fn-112 adds (spec Decision Context, operator policy). Define it on the typed input token with `@targetName("set")`, as a plain `def` (no `inline`, no macro) the lifter matches by its source name. It means exactly "this named slot receives this value" and nothing else; no other operator takes that meaning, and fn-112.9 reuses this same operator and meaning for typed request fields inside `rpc`/`poll`, so define it in a way that scope can share (one token-typed left side, one value right side) rather than something specific to action inputs.
- **Core and sugar (spec Decision Context "Core and sugar").** `input[A]` tokens and the Action declaration are core (`Action.scala`); `:=` is sugar for positional supply, so it lives in `model/umpire/Syntax.scala` with the positional call it stands for documented (`start(scheduleToStart := expires)` is `start(unset, expires, unset)`), `Action.scala` imports nothing from it, and its matching in `model/lifter/Syntax.scala` reorders and defaults into exactly the positional argument list; one fixture declares a named-token call beside its positional spelling and the expected JSON proves the trees equal.
- Refuse duplicate and foreign tokens, wrong value types and an input absent from the Action; compiler errors count where invalid Scala cannot produce TASTy (`:=` on a foreign token is refused by the token's type).
- Add UpTo[N] finite enumeration and use a fixture to pin `0,1,2` order for UpTo[2]. Keep Active as its current enum.
- Preserve explicit `results("Delivery")` IR metadata while separating removal of the dead Scala enum.
## Acceptance
- [ ] Named-token calls support partial and reordered inputs while lifting exactly the original positional argument list and defaults; a fixture proves IR equality between a named-token call and its positional spelling.
- [ ] `:=` carries `@targetName("set")`, is a plain `def` in `model/umpire/Syntax.scala` the lifter matches by name in `model/lifter/Syntax.scala`, and has the one meaning "a named slot receives a value"; no other symbolic operator is added and `:=` is given no second reading; `Action.scala` imports no `Syntax.scala`.
- [ ] The positive/negative matrix covers repeated input types, duplicate/foreign tokens, wrong value types and missing inputs; the foreign-token refusal is recorded at the layer (compiler or lifter) that catches it.
- [ ] UpTo[2] enumerates exactly 0,1,2 and Protocol-state fixture keys match the original baseline; Active is unchanged.
- [ ] No action input-name string literal or new IR field is required.
## Done summary
Added input tokens, named inputs with `:=` and the `UpTo[N]` bounded counter. No production Model changed; model/ir, model/cases and every lifts/expected/*.json are byte-identical, and rejects.txt gains 7 refusals.

**What changed**
- Core (`model/umpire/Action.scala`): `trait Slot[A]`, `final class Input[A]` (compared by identity), `def input[A]: Input[A]` (named after its val by the lifter), `Action.input(token)`. The positional calls are now Action members typed per position by the total match type `InputAt[I, Size, N]` (falls back to the uninhabited `OtherInputs`).
- Core (`model/umpire/Domain.scala`): `opaque type UpTo[N <: Int] <: Int = Int`, `UpTo(n)` and `given Finite[UpTo[N]]` (0..N).
- Sugar (`model/umpire/Syntax.scala`): `Assigned[A]`, `extension [A](slot: Slot[A]) @targetName("set") def :=`, and the named call `start(scheduleToStart := expires)`, documented as the positional `start(unset, expires, unset)`.
- Lifter: token inputs in `Declarations.actionOf`/`inputToken` (tokens recorded in `Context.inputTokens`). Named calls in `lifter/Syntax.scala` (`namedClass`/`named`): supplied inputs reordered into declaration order, omitted ones defaulted to the first catalog value (`firstValue`). Hooked from `Claims.classOf`, which also reads the member positional form. `UpTo` lifts as `IntRange(0, N)` (`Types`), and `UpTo(n)` lifts as `n` (`Expressions`).
- Fixtures: `lifts/Inputs.scala` and a new pair test. Named and positional Scenarios and `when` classes lift to equal JSON: partial, reordered, enum-with-fields and Boolean defaults. `Counted` (production `Phase`/`Timeout`, `attempts: UpTo[2]`) lifts to exactly the record of `temporal.standaloneactivity.ProtocolState`, so Go keys both catalogs alike. `steer.results("Delivery")` keeps the text with no Delivery type lifted.
- Refusals:
  - Lifter (Rejects.scala): foreign token; token supplied twice; supply kept in a val; token given to a name-string action; token declared twice; token no val names; `UpTo[-1]`.
  - Compiler (crossed/NamedInput.scala): `:=` value of another type; `:=` on a non-slot; wrong positional type; wrong arity.
- `model/umpire/Inputs.test.scala`: runtime UpTo[2] = 0,1,2; named = positional; foreign and duplicate throw. README documents tokens, `:=`, UpTo, core/sugar.

**Decisions (autonomous)**
- **Positional calls became members.** Scala 3 forbids overloading a top-level extension `apply` across files ("overloaded methods must all be defined in the same group of toplevel definitions"). A member that does not apply falls back to extensions, so this is the only way to keep the named call in Syntax.scala while core imports nothing from it. Production compiled unchanged and its IR is byte-identical.
- **The foreign-token refusal is in the lifter.** `Input[A]` is not tied to an action, so a token of another action with the same value type compiles. A wrong value type, and `:=` on a non-slot, are compiler refusals.
- **`:=` is defined on the generic `Slot[A]`** and returns `Assigned[A]`, so fn-112.9's request field can extend `Slot` and reuse the same operator and meaning.
- **`UpTo(n)` lifts as identity.** Go range-checks bounded fields as it does today. The per-record `given Finite[Int] = Finite.upTo` rule stays for Models not yet migrated (nexuscaller and the feature).
- **Delivery:** the production `enum Delivery` (dead, absent from IR types) is left for fn-112.6/.8. This task proves the `results("Delivery")` metadata is independent of it.
- **No new expected JSON** (the inventory is closed). The proofs are assertion tests, as in fn-112.2-.4.
- **Files outside Touches:** `lifter/Claims.scala` (classOf hook), `lifter/Context.scala`, `lifter/test/Fixtures.test.scala`, `testdata/crossed/NamedInput.scala`, `model/umpire/Inputs.test.scala`, `model/README.md`.
- **Parallel work:** one opus subagent wrote the crossed compile-refusal fixture and the README.

**Review:** claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- Round 1: SHIP with one P3, a needless deepCopy, fixed in 9add08ddde.
- Round 2: SHIP.

**Deferred P3/FYI:** `literalValue` does not read `UpTo(n)`, so a counter-typed action input cannot be given positionally or in an `example` (unused today). `inputToken`'s message for a token val whose right side is not literally `input[...]` could be clearer.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8323044169, 9add08ddde
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), scala-cli test model/lifter (exit 0), scala-cli test model/project.scala model/umpire model/temporal (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs:
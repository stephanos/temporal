# fn-114-state-every-scala-model-declaration-once.11 Drop type annotations the compiler and lifter do not need

## Description
Owner request (2026-10-04): the Models carry many type annotations the compiler infers anyway, e.g. `val rejectAfterCloseQueries: Vector[Query] = designQueries(rejectAfterClose)` or `val all: Vector[Query] = Vector(...)`. A count on 2026-10-04 found about 112 annotated `val`/`def` declarations under `model/temporal` (72 `List[...]`, 48 `Boolean`, 24 `Vector[...]`, plus `Progress`, `Assumption`, `Party`, `Entity`, `Role`, `Realization`, ...). No rule asks for them: `.scalafix.conf` has no explicit-result-type rule, and the lifter reads `ValDef.tpt.tpe`, which TASTy records for inferred types too.

**Entry gate:** after fn-114.7 (every Model restated), before fn-114.10 and fn-114.9, so each file is edited once more at most.

**Size:** M
**Files:** `model/temporal/**` (Model, Properties, Queries, Realization, IrFiles files), `model/umpire/realize/Kit.scala` only where a helper's local vals carry redundant types, `model/README.md` (authoring guidance), `.flow/tmp/fn114-11/**`.
**Touches:** [model/temporal/**, model/umpire/realize/**, model/README.md, .flow/tmp/fn114-11/**]

### Approach
- Remove an annotation when the inferred type is the same type. Keep it only where it is needed, and say why in the done summary by category:
  - the lifter classifies the declaration by its declared type in a way inference would change (e.g. a `Machine[S, O, F]` val whose types fn-112.2 lets the val type state instead of `machine[S, O, F]`);
  - inference would produce a different type (a singleton/literal or union type, a narrower subtype, a different overload, an expected-type-driven conversion or given);
  - a cyclic-reference error between top-level vals;
  - a public framework signature in `model/umpire/**` (out of scope here; framework signatures stay as they are).
- Do not change behavior: `make umpire-gen-model` must leave `model/ir/**`, `model/cases/**` and `lifts/expected/**` byte-identical except source positions.
- Add one sentence of authoring guidance to `model/README.md`: write a type only where inference would differ or the lifter needs it.

### Investigation targets
**Required:** `model/temporal/**/*.scala`, `model/lifter/Lifting.scala:20-60` (declared-type classification), `model/lifter/Types.scala:170-190` (`Finite` read from the declared type).

## Acceptance
- [ ] No `val`/`def` under `model/temporal/**` keeps a type annotation that inference reproduces; each kept annotation falls in a recorded category, with counts before and after in `.flow/tmp/fn114-11/`.
- [ ] `model/ir/**`, `model/cases/**` and `lifts/expected/**` are byte-identical apart from source positions; the original-baseline and migration goldens, the model gate, `lint-model`, the Go tooling suite and `lint-code-fast` pass.
- [ ] `model/README.md` states when a Model writes a type.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

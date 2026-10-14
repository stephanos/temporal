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
Dropped the type annotations that inference reproduces under `model/temporal/**`. Commit 43d8521a1a.

**Counts** (val/def/given declarations with a body; `.flow/tmp/fn114-11/counts.txt`, per declaration in `kept.tsv`)
- 269 before, 25 after; 244 removed.
- By kind: def 179 → 18, val 86 → 3, given 4 → 4.
- By type: List 75 → 15, Boolean 48 → 0, Vector 25 → 0, Long 3 → 3.

**Kept, by the task's categories**
- **Inference would produce a different type: 21.**
  - 8: a step whose body is `disabled`/`stay` infers no facts (`List[Nothing]` / `Step[S, O, Nothing]`).
  - 7: in taskqueue, `accept`'s outcome type and its companion-scoped `given Accepted` come only from the expected type.
  - 3: in Kit, a `Long` written as an `Int` literal.
  - 1: `saturatingSucc`, where `UpTo[2]` needs the expected type for `ValueOf[2]`.
  - 1: `atMostOneActive`, where `never` infers the narrower `Never[S, O, F]` instead of `Property[S]`.
  - 1: `decided`, where `Condition.equal`'s type argument comes from the expected type.
- **Lifter classifies by declared type: 0.** TASTy records inferred types, and the inferred root types (Machine, Composition, Realization, Vector[Query], Progress) are identical.
- **Cyclic references: 0.**
- **Framework signatures: none touched.** `model/umpire/**` is out of scope.
- **Outside the four categories:** the 4 `given family: Family` aliases, whose type is required syntax.

**Method** (scripts in `.flow/tmp/fn114-11/`)
- Stripped every annotation and compiled with `-Vprint:typer`.
- Diffed the typed trees per unit against HEAD, with synthetic names normalized.
- Restored each declaration whose tree changed, and pruned cascade victims one at a time.
- Final proof (`r400.show`): the only remaining differences are step results printed as the dealiased `Step[...]` instead of the alias, which is the same type.

**Behaviour**
- `make umpire-gen-model` left `model/ir`, `model/cases` and `lifts/expected` byte-identical, positions included: only in-line edits, and scalafmt needed no reflow.
- `model/README.md` now states when a Model writes a type.

**Decisions**
- Removed annotations whose inferred type is the alias written out (`List[ProtocolStep]` vs `List[Step[ProtocolState, Outcome, ProtocolFact]]`): this is the same type, and the IR is identical.
- Edited `model/temporal/realize/Kit.scala` (task Files/Touches, `model/temporal/**`).
- Left `model/umpire/realize/**` untouched, per conductor.
- Edits are annotation removals only: no reformatting or reordering, so merges with fn-122 and fn-114.12 stay line-local.

**Setup**
- Copied `model/gen` (without history/) and `proto/api.binpb` from the main checkout.
- Touched `api/umpire/v1/ir.pb.go` and `proto/api.binpb` so the gate's staleness checks accept the unchanged files.

**Review**
- claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- Round 1: SHIP, no findings.
- FYI P3s, deferred:
  - the README sentence is long;
  - mixed annotated and unannotated steps in one file look arbitrary but follow the rule;
  - Kit's shared helpers now rely on inferred types (in scope by the spec).
- Three parallel Opus read-only folder reviews (nexuscaller, standaloneactivity, taskqueue/worker/realize) found nothing.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 43d8521a1a
- Tests: CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; model/ir, model/cases, lifts/expected byte-identical), mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, 118s), mise exec -- go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, 93s), mise exec -- go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 484s), mise exec -- make lint-model (exit 0, 115s), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast (exit 0, 123s, 0 issues), scala-cli compile -Vprint:typer before/after typed-tree diff (.flow/tmp/fn114-11/r400.show: alias-only differences), git diff --check (exit 0)
- PRs:
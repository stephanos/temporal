---
satisfies: [R1, R2, R4, R5]
---
# fn-169-a-temporal-free-framework-the-lifter.1 Mechanical move: generic export/check/migrate and the DSL into model/framework, Temporal parts into model/temporal; Temporal-free build boundary; docs

## Description
Runs after fn-166.13 and before fn-141 (owner, 2026-10-12). Read the spec's survey and target layout first.

1. **Move.** Generic code from `model/irgen`, `model/check` and the `model/migrate` harness moves into `model/framework/{export,check,migrate}` with `framework.*` packages. Temporal-specific parts go to `model/temporal/{export,check,migrate}`: the realization-kit lifting, layout conventions, Temporal API use, the Gate, ProtoLiterals, kit syntax rules, and every task rule with its proof. Where a file mixes both, split it only as far as a mechanical move allows; the full plug-in seam is fn-169.2, after fn-141.8. Record the residual coupling it leaves.
   **`check/` layout (owner, 2026-10-12):** `framework/check/` (package `framework.check`) holds the gate driver (`Gate`, `Tools`, `ModelSources`) and metrics; `framework/check/lint/` the generic source rules (comment, block-form, syntax-placement, the string-blanking scanner of `ProtoLiterals`), where fn-141.1's order/structure/marker lints will land; `framework/check/equivalence/` the step-table equivalence, rename maps and ledgers (`Equivalence`, `StepLedger`), used by the gate and the migration harness. Temporal side: `temporal/check/` the Temporal gate configuration (which protos, which Go checks), `temporal/check/lint/` the kit syntax rules and the Temporal API proto-name rule.
   **Generic proto support (owner, 2026-10-12):** protobuf is framework, not Temporal. Move the generic proto authoring sugar in `temporal/realize/Syntax.scala` (`ProtoValueOf`, `FieldScope`, `field`, `proto[M <: GeneratedMessage]`, `read`, and anything else in it that names no Temporal API) into `framework/realize`, after confirming each definition is Temporal-free; leave only Temporal kit vocabulary in `temporal/realize`. The registration seam for a system's API protos is fn-169.2.
   **Colocated tests (owner, 2026-10-12):** test files sit next to what they test, as the framework already does (`X.test.scala` beside `X.scala`). Dissolve `model/irgen/test/`, `model/check/test/` and `model/migrate/test/` (and any other `test/` folder in the moved code) into their new homes beside the tested sources. A suite that spans many files, such as `Fixtures.test.scala`, goes beside the module's entry point (the lifter's `Lift.scala`); fn-141 splits it per construct as it rewrites each one. Fixture sources stay in a `testdata/` folder, excluded from the build because some must not compile. Update `model/check/Coverage.scala` and the scala-cli project files for every moved suite.
2. **Build boundary.** `model/framework` compiles with no Temporal API jar and no `temporal.*` source on its classpath. A gate/source check fails on a violation.
3. **Migration harness (owner, 2026-10-12):**
   a. Parameterize it by feature. `prove-mini.sh` becomes `prove.sh`, selecting the feature with `FEATURES` (default `activity_mini`) and deriving the copied source set from fn-166.13's scope logic instead of today's hard-coded list (`framework`, `temporal/foundations`, `actors`, `capabilities`, `realize`, `Bounds.scala`, `Activity.scala`, `features/activity_mini`). A rule's rewrite target is the runner's argument, not a hard-coded `activity_mini/protocol`.
   b. Reuse `framework/check/equivalence` for the before/after lift comparison, including the declared-delta case (moved source lines, recorded identity maps) that proofs now handle with their own scripts (`RewriteGapsMini compare`, `check-*-delta.mjs`). Retire the plain `cmp` comparison and the per-proof comparison scripts that this replaces.
   c. Split by home: the generic harness goes in `framework/migrate`; the per-rule proof runners stay beside their rules in `temporal/migrate`. Move `GuideSourcesTest` (a docs-excerpt check) out of `migrate/test` into `framework/check/lint`.
   Every existing rule proof still passes through the new harness.
4. **DSL sub-package (owner, 2026-10-12).** The authoring DSL (today's `model/framework/*.scala`, with `outcomes` and the other DSL members) moves into `model/framework/dsl/`, package `framework.dsl`, so a Model's `import framework.dsl.*` brings only the authoring API into scope; today's `import framework.*` would otherwise expose the `lift`, `check`, `migrate` and `realize` package names, and `check` collides with fn-164's planned `check(condition)`. Update mini and shared imports in source, and ship a Scalafix import rule (proven on copied mini) for the real features at the sweep. **IR names may change (owner, 2026-10-12):** the framework's DSL types are named by their new Scala package in the IR (`framework.dsl.Step`, `framework.dsl.outcomes.Outcome`, ...). Record the complete identity delta as a rename map and prove mini step-table equivalence under it. The Go interpreter's hard-coded `framework.Step`/`framework.Delivery` (`tools/umpire/interp/eval.go`) and Go test strings follow in held fn-169.4; until then the checked-in mini IR stays stale like the rest.
5. **Follow the move.** Update Make targets, scala-cli project files, fn-166.13's coverage registry, the worker docs and every doc naming the old paths: `model/README.md`, `model/migrate/README.md`, `AGENTS.md`, `.plans/UMPIRE_MODULES.md` (record fn-114.12/fn-122.8 as superseded).
6. **Proof.** The mini lift is byte-identical except for recorded source-path changes; mini step-table equivalence holds; the moved suites pass at their new homes.
## Acceptance
- `model/framework/{export,check,migrate}` exist with `framework.*` packages; Temporal parts are under `model/temporal/{export,check,migrate}`; old `model/irgen`, `model/check` and `model/migrate` directories are gone or hold only what the receipt names.
- The framework builds with no Temporal API jar or `temporal.*` source; a deliberate violation fails the boundary check.
- `prove.sh` proves every existing rule on mini through `framework/check`'s comparison, with feature selection by `FEATURES`; the per-proof comparison scripts it replaces are removed; `GuideSourcesTest` runs from `check`.
- Mini lift byte-identical apart from recorded source paths; mini equivalence passes; docs updated.
- The DSL is in `framework.dsl`; mini/shared import `framework.dsl.*`; the import rule passes its copied-mini proof; the IR's DSL type names follow `framework.dsl` with a recorded rename map; mini equivalence passes under it.
- No `test/` folder remains in the moved code; every suite is colocated with what it tests and registered at its new path.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

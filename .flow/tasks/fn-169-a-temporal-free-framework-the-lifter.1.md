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
   **Scratch retention (owner, 2026-10-13):** today every lifter-suite run builds in a new `model/build/history/<name>.<random>` directory and nothing deletes it ("kept for inspection: nothing is deleted, so another process's build state is never removed"). That grew to about 600 runs and 5 GB in a week. Instead, each run deletes its own directory on success. A failed run keeps its directory for inspection, and at most the newest 10 failed runs are kept. Each process prunes only directories that are not its own and are older than an hour, so concurrency stays safe. Apply the same rule to the scratch directories the migration harness and suites create under `.flow/tmp` (`mini-migration.*`, `facet-*`, `observations.*`, ...).
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
- A passing suite run leaves no scratch directory behind; failed runs are capped at 10.
## Done summary
### Done (fn-169.1)

Requirement evidence:
- R1 move: model/irgen -> model/framework/export (package framework.`export`); model/check split into
  model/framework/check (Tools, ModelSources, ChangeCheck over a generic Registry, Json, lint/: comment,
  block-form, syntax, boundary, literal scanner; equivalence/: Equivalence, StepLedger, Lifts) and
  model/temporal/check (Gate, Temporal coverage registry, source metrics, lint/ProtoLiterals, baselines/);
  migration rules/proofs/suites -> model/temporal/migrate, harness -> model/framework/migrate/prove.sh;
  lifter plug-in hook (framework/export/Plugin.scala, Java service) with TemporalPlugin (Temporal module map)
  in model/temporal/export. Old model/irgen, model/check, model/migrate are gone. Tests colocated, no test/
  folders; fixtures in framework/export/testdata. A Temporal tool builds from both halves, framework first.
- R2 boundary: framework/dsl/project.scala builds the core with no Temporal API; Boundary.scala rule +
  `make check-model-boundary` (also compiles framework core, export, check alone); run by
  model-check-changed. Deliberate violations (import temporal.realize.Kit in Lift.scala; api-scalapb.jar
  directive) fail it (.flow/tmp/fn169/task1/boundary). It found one real crossing (DSL test used a Temporal enum).
- Harness: prove.sh parameterized by FEATURES, sources from framework.check.selectedSources, target is the
  runner's argument, comparison via framework/check/equivalence/Lifts.scala (byte-for-byte or declared delta);
  cmp, RewriteGapsMini compare mode and check-*-delta.mjs retired; GuideSourcesTest in framework/check/lint.
  All 13 rule proofs pass (.flow/tmp/fn169/task1/proofs/final, RewriteMini rerun in final-rerun after a
  broken-incremental-class flake).
- Retention: Retained (Tools.scala) + RetainedScratch mixin; passing runs delete scratch, failed keep it
  marked (<name>.failed), newest 10 kept, others' dirs pruned only when older than 1 h; applied to lifter
  and migrate suites, the gate's own scratch and prove.sh. A passing model-check-changed left no new
  history directories.
- Proto sugar: temporal/realize/Syntax.scala (all Temporal-free) -> framework/realize/Syntax.scala.
- DSL: framework/dsl, package framework.dsl; mini/shared/kit import framework.dsl.*. IR type renames
  (framework.Step, framework.outcomes.Outcome/Rejection, framework.Domain$package$.UpTo gain .dsl) recorded as
  `ir` entries in model/temporal/check/baselines/activity_mini.map (RenameMap learned `ir old new`, left to the
  IR in steps-only mode). Mini lift equals the pre-task lift under those renames plus the Syntax.scala path move.
- Coverage.scala path rules and modules updated; docs: model/README.md (layout, boundary, residual coupling),
  AGENTS.md, .plans/UMPIRE_MODULES.md (fn-114.12/fn-122.8 superseded), migrate READMEs, sweep recipe.

Verification: make model-check-changed passed, .flow/tmp/checks/20261012-221229/record.md (boundary, compile,
lift, mini step-table equivalence, models 139, lifter 275, check 139, migrate 68 tests). Earlier checkpoints:
20261012-185201, -191004, -194243, -201617, -204457.

Notes: the owner committed the in-progress DSL rename and docs as `wip` (4fcd8c2b5b); d088870ccd adds the
sweep recipe. Fn169Task1DslPackage + its proof and OnLegacyDsl were built before the owner dropped the
rule-per-change policy; they are kept, the recipe is the migration of record.

### Open obligations
- fn-169.2: residual Temporal coupling as text in framework/export (kit lifting, layout conventions,
  Switches), framework/check (ModelSources, Syntax/BlockForm trees, StepLedger constant), whole Gate in
  temporal/check, SourceMetrics' API category (listed in model/README.md).
- fn-169.3: IR namespace io.temporal.server.api.umpire.v1 still on the framework's classpath.
- fn-169.4 (held, Go): Go tests read lifter fixtures at model/irgen/testdata paths (now
  model/framework/export/testdata) and hard-code framework.Step/Delivery; the checked-in mini IR is stale.
- Sweep: real features are behind (old imports, old package framework tests); apply the recipe; add the
  `ir` renames to the sweep's map.
- Flaky in-process fixture races seen (FacetsTest "zip file is empty", PropertyGroupsFixtures native run):
  pre-existing, pass on rerun.

### Deferred gates
lint-model (scalafmt of whole model, scalafix of Models/lifter, lint-model-conventions with the new jar
prefixes), umpire-check-model, make model-mini, full Go, independent review: batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c6f9f309ca, 25ffe44e08, 2031e5bb58, bc09cd51f6, a3d1321015, 4fcd8c2b5b, d088870ccd
- Tests: make model-check-changed: .flow/tmp/checks/20261012-221229/record.md (passed), make check-model-boundary with deliberate violations: .flow/tmp/fn169/task1/boundary, prove.sh x13: .flow/tmp/fn169/task1/proofs/final
- PRs:
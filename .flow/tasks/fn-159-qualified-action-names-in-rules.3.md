---
satisfies: [R9]
---
# fn-159-qualified-action-names-in-rules.3 Qualify every phase reference; refuse phase-member imports

## Description

Close the current Scala source/lift proof for qualified phases and phase-member import refusal.
Enforcement and the historical source migration are already integrated; re-anchor rather than
repeating them. The owner Scope limits this task to the mini feature, framework/lifter and shared
code. Standalone/Nexus/legacy sources and generated artifacts remain untouched for the sweep.

**Size:** M
**Files:** typed phase-import/rule-heading checks, isolated qualification fixtures, source-rule
proof and author docs; mini/shared source only where an unmet contract requires a change.
**Touches:** [model/irgen/Conventions.scala, model/irgen/Declarations.scala, model/irgen/test/QualifiedPhases.test.scala, model/irgen/testdata/qualifiedPhases/**, model/irgen/testdata/conventions/PhaseImports.scala, model/irgen/testdata/lifts/RuleRejects.scala, model/framework/Syntax.scala, model/temporal/features/activity_mini/**/*.scala, model/temporal/foundations/taskqueue/**/*.scala, model/migrate/**, model/README.md, model/framework/SEMANTICS.md]

### Approach

- Begin after the committed .1 qualification closure and preserve its source-spelling/refusal
  work, focused fixture infrastructure and identities. Do not re-run broad suite baselines.
- The relevant phase type is the `P` of `Phased[S, P]`, including inherited projections reached
  by derivations/compositions, not an enum's name. Preserve unrelated enum/member imports.
  Independently test a differently named finite phase enum and accepted non-phase imports.
- Inspect current `Conventions.phaseImport` and `Declarations.qualifiedPhase`. Pin real authored
  wildcard/selective/renamed-member imports at file, machine, nested-section and rules-block levels,
  including an import written after another token on the same line. The current text check uses
  `trim.startsWith("import")`; do not assume the existing three whole-line fixtures prove all
  authored import positions. Keep compiler-generated enum companion imports distinct, not an
  excuse to suppress actual imports or malformed/source-unavailable authored positions.
- The lifter must refuse each applicable import at its actual source position and name the type
  and qualified remedy. It also refuses bare arguments of phase-form `when`, including a bare
  eta-expanded named set. Accepted `when(Phase.x)`, qualified set and role forms remain valid;
  Property-builder `when` is not this rule-case surface.
- There is no requested framework runtime phase check: varargs `when` is not inline. Preserve
  that boundary and all qualified mini effects/init/states/properties/choice alternatives.
  The native contract .1 establishes for action names is a different surface.
- Ship an actual task-named Scalafix source rule under the shared convention. Focused copied-mini
  legacy-to-qualified transformation, ambiguity/refusal, idempotence and compiled identity/meaning
  controls prove source correctness. Do not apply rules to real features or claim the held loader
  works. Reuse the existing rule-source/test harness; no competing generic rewrite engine.
- Documentation uses qualified phases and the complete mini's actual cited source. Keep the
  source-citation guard passing by fixing inaccurate excerpts/line references, not the oracle.

### Source and consumer boundary

R9's current source check is on selected mini/shared input and isolated fixtures. Real-feature
source/lift equivalence, managed IR/Case regeneration, Go/lint and canonical Make gates stay at
the sweep. No protected baseline, pin, expected IR or accepted lint debt is rewritten to obtain
source closure. Preserve the earlier task/spec history and actual failed attempts.

### Verification

Fresh mini/shared compile, focused native tests where changed and a new isolated QualifiedPhases
lifter suite are the per-step gates. The broad Fixtures class packages all fixture roots in
beforeAll, so do not invoke it merely to filter a phase test. Freeze each proof's source/jar inputs
while it runs. Scratch pre/post mini lifts must preserve selected declaration identities and
semantics, and state explicitly any justified rule-heading/source-span difference rather than
normalizing it away. Run direct `FEATURES=activity_mini` Scala steps-only equivalence against
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`
with a separate justified mini map, never modifying either baseline or the batch-1d map. The complete
selected baseline includes shared query roots: 16 tables/11,325 rows from the pre-change task8 jar;
the first 10-table mini-reachable capture is retained unchanged.
Run focused migration-rule/source-citation checks and changed-file formatting/diff checks.
Lint/manual acceptance is NOT RUN. Broad validation/independent review is deferred to the ONE
mini batch-end gate; canonical Make/Go/regeneration/live remains held, never reported as green.

## Acceptance

- [ ] Actual phase-member imports are refused for wildcard/selective/renamed forms at all authored
  levels, including inline same-line locations, with independently asserted positions/type/remedy;
  compiler-generated companion imports and unrelated enum imports are not falsely rejected.
- [ ] Bare phase-form `when` and bare named sets are refused; qualified phase/set and role forms
  are accepted, with Property `when` unaffected.
- [ ] Selected mini/shared source contains only qualified phase references and no phase-member
  imports; its scratch lift and native tables preserve the contract, while real-feature sources,
  artifacts, pins and old baseline/map remain unchanged.
- [ ] The task's actual migration-rule source has focused transformation/refusal/idempotence and
  compiled copied-mini proof; real-feature application and production loading are explicitly held.
- [ ] README/framework/guide examples accurately show qualified phases and retain checked source
  citations. Focused source gates and direct mini native equivalence pass; any exact lift differences
  are explained and independently checked without laundering source positions/identities.
- [ ] Failed attempts and deferred canonical/consumer/lint/broad-review gates are recorded honestly;
  no full-set, live, broad-suite or SHIP credit is claimed.

## Done summary
The mini/shared R9 source closure now refuses phase-member imports at every authored position while preserving compiler-generated enum imports. A task-named Scalafix rule, isolated lifter controls and copied-mini proof preserve the qualified mini's identities and meaning; real-feature and consumer closure remains held.

Tier: session (jev-unavailable(no_key)); project implementer routing retained
stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers broad validation and independent review to the ONE Scala batch-end gate)

Baseline: green, fresh selected mini/shared package and eight existing migration/source-citation tests, without handoff or receipt reuse. No full-suite Quick command is defined for this scoped step. Gate classification returned FULL (IDEAS.md in the committed cumulative range); the task's focused source gates ran. No docs-only or green-receipt skips are claimed.

R9 enforcement:
- QualifiedPhasesTest compiles its isolated source universe against the task3 frozen scoped-model.jar/classpath, never the broad Fixtures class.
- Twenty-four authored import controls cover wildcard/selective/renamed forms at file, machine, nested-section and rules levels, with whole-line and inline layouts independently asserted. Each refusal independently names the actual source line, imported member/form, Stage phase type and Stage.<case> remedy. An unrelated enum is deliberately named Phase: its member imports remain accepted.
- Malformed and unavailable authored-source controls cannot pass as enum synthesis. The synthetic exemption requires the actual enum owner and an empty imported-expression span, independently confirmed by compiler diagnostics and positive mini/shared lifts. Removing the fail-closed source branch makes the malformed-source control lift successfully (RED sensitivity control).
- Bare phase-form when and bare eta-expanded sets refuse at their authored lines; qualified phase/set, role headings and Property-builder when accept. A derived machine retains its source phase. Existing qualifiedPhase, native source capture/TASTy and ownership, initializer checks, framework Syntax and authored mini/shared declarations are unchanged. No runtime phase check was added.

Migration and author docs:
- Fn159Task3QualifiedPhases is actual compiled Scalafix 0.14.9/JDK25 source. Its four focused tests cover every phase-reference surface, wildcard/selective/renamed qualification, idempotence, unrelated enum/Property controls, ambiguity, qualifier/member shadowing, mixed importers, competing wildcards, unsupported selectors, external wildcard refusal, and lexical scope.
- The rule reads P from an explicit local Phased[S,P]. It rewrites external selective/renamed imports; a wildcard requires enum cases in the same document. Documents whose phase is inherited with no explicit local Phased and external wildcards have a documented manual recipe. No generic rewrite engine or production loader was introduced.
- Copied-mini proof directory: .flow/tmp/mini-migration.wWHmcw. RewriteQualifiedPhasesMini introduces a renamed scheduled phase import only in copied Dispatching.scala, transforms both states/rules references, saves accepted/legacy/qualified sources, proves exact accepted-source restoration, idempotence and atomic ambiguity refusal. Before/after jars are independently compiled and their complete raw activity-mini.json and waiver bytes compare identically. Both IR hashes: b38f5fe304ee9d6c0de6aad646e39a68c36cc5ecad3b071cf169e57c801d92d9. No source-position/identity projection, normalization or expected-byte rewriting.
- README examples now consistently qualify phase references. SEMANTICS explains P, all import levels and the lifter/native boundary. GuideSourcesTest preserves all 30 exact source-cited excerpts.

Native proof:
- Direct FEATURES=activity_mini steps-only equivalence writes and compares all 16 selected tables against the immutable COMPLETE mini-selected-baseline-00d22597d2 capture (11,325 rows).
- The task3 mini.map contains only a justification comment, with zero identity changes/removals/projections. Both COMPLETE and original ten-table mini-baseline-00d22597d2 integrity scripts pass; the old batch-1d map and protected captures were not edited.
- All source and jar inputs remained frozen for the proof/equivalence commands; no overlapping source edits or formatting occurred. After those finished, only independent fixture-shape/diagnostic assertions were strengthened, then the four focused lifter tests and changed-file format check passed again. Runtime lift/migration/native inputs are unchanged.

Final GREEN logs under .flow/tmp/fn159/task3:
- baseline-package.log, baseline-migrate.log
- lifter-position-final.log: four isolated lifter tests
- migration-final.log: twelve migration/citation tests
- proof-final.log: exact copied-mini compiled IR/waiver proof
- equivalence-final.log: sixteen native tables equivalent
- format-check-final.log; git diff --check and staged diff check

Recorded failed/RED observations:
- imports-red.log: original line-prefix heuristic refuses twelve whole-line controls and misses all twelve inline controls; initial accepted-property count lacked the property root. Failing reproduction pinned in f5affca448 before the fix.
- span-diagnostic.log: retained inline-import RED plus incorrect initial nested property-root spelling; corrected to the actual symbol root.
- lifter-first.log and owner-diagnostic.log: refuted companion-owner synthetic exemption falsely rejects generated enum imports; replaced using observed enum owner and empty expression span. Temporary instrumentation was removed.
- migration-red.log: the initial no-op rule fails transformation/refusal controls.
- migration-first.log: deprecated Template.stats rejected by Werror; replaced with established body.stats.
- migration-second.log: enum-case declarations were wrongly treated as imported uses; excluded while preserving actual bare references.
- source-unavailable-red.log: first sensitivity attempt also reached an invalid inline-effect fixture; replaced fixture effects with valid named defs.
- source-unavailable-red-final.log: with the source guard deliberately bypassed, the valid malformed-source control lifts (exit 0), failing the expected refusal. Bypass reverted before final gates.
No failed observation is represented as green or as whole-suite credit.

Defect route:
- prior fixes: current integrated phase enforcement and recent Conventions history inspected; bug memory search found no phase-import-position fix. GitHub PR/issue searches unchecked because gh PR search returned HTTP401 (bad credentials).
- diagnosis: original whole-line source heuristic misses twelve actual inline import spans; compiler span/owner evidence confirms authored imports contain import text, while generated enum imports have the enum owner and empty expression span. Synthetic-owner hypothesis was refuted and removed.
- introduced by: skipped, no known-good inline-import refusal revision was provided.
- base: f5affca448 reproduction plus imports-red.log lacks twelve required inline refusals; head: four final isolated tests pass, including source-integrity refusal sensitivity.
- live: no live surface for this static lifter/source-rule task; Cases/live remain held.

Protected/deferred:
- NOT RUN: lint/manual acceptance; broad validation and independent review (owner ONE batch-end deferral); canonical Make/linked-protobuf routes; Go/proto/CI/schema/runtime/Case/live/regeneration; production custom-rule loader; standalone/Nexus/legacy/real-feature migration or lift.
- No SHIP verdict, whole-spec completion, real-feature source-complete or consumer credit is claimed.
- External seventeen owner Scope specs, MILESTONES and untracked fn168 planning files remain dirty and unstaged. Owner IDEAS commit e81c2c3e6326283f3726d945057ae427c9ce35d1 is preserved in the contiguous range and is not this task's implementation.
- No push, branch switch, merge, rebase, amend or history rewrite. Conductor owns local integration and checkout return to umpire.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f5affca44822bdb9defa01a2eed65ad409748a31, e81c2c3e6326283f3726d945057ae427c9ce35d1, eac089656b8ce3868f1b9b6c568d1cf3cdcc5315
- Tests: timeout 600 mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/framework model/temporal/Bounds.scala model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/realize model/temporal/features/activity/Activity.scala model/temporal/features/activity_mini -f -o .flow/tmp/fn159/task3/scoped-model.jar, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.QualifiedPhasesTest, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/migrate --test-only 'umpire.migrate.*Test', timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteQualifiedPhasesMini, FEATURES=activity_mini timeout 600 mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2 /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task3/mini.map --steps-only, node /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/integrity.mjs, node /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-baseline-00d22597d2/integrity.mjs, timeout 600 mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/irgen/Conventions.scala model/irgen/test/QualifiedPhases.test.scala model/irgen/testdata/qualifiedPhases model/migrate/rules/Fn159Task3QualifiedPhases.scala model/migrate/test/QualifiedPhases.test.scala model/migrate/proof/RewriteQualifiedPhasesMini.scala, git diff --check; git diff --cached --check, baseline: green, fresh scoped compile and eight migration/citation tests; no handoff or receipts, RED: imports-red.log (12 inline refusals missing); migration-red.log (no-op rule sensitivity); source-unavailable-red-final.log (guard-bypass sensitivity), FAILED SETUP/IMPLEMENTATION: initial property root/count; synthetic companion-owner exemption; deprecated Template.stats; enum-case declarations considered bare uses; invalid inline-effect fixture in first unavailable-source control, NOT RUN: lint/manual acceptance, broad validation and independent review (owner ONE batch-end deferral), canonical Make, Go/consumer, Cases/live/regeneration, production loader and real-feature application
- PRs:
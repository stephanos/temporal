---
satisfies: [R13]
---
# fn-159-qualified-action-names-in-rules.6 Drop Observation's free-text read field

## Description

Close the current source/API/lift proof for `Observation(name, on)`. The removal is already
integrated in commit `50bc14c2fa9deec636528c0524dcd6cf0a851f90`; inspect and reuse it, rather
than deleting the field again. Its parent contains the actual retired three-argument signature.
The current owner Scope permits the framework/lifter, shared code and `features/activity_mini`
only. Real-feature uses, realizations, lift proofs and consumer closure remain at the sweep.

**Size:** S
**Files:** isolated Observation API/name/typed-read controls, task-named migration/source proof,
current framework/lifter comments and author docs where a demonstrated gap requires a change.
**Touches:** [model/framework/Action.scala, model/framework/Observation.test.scala, model/irgen/Constants.scala, model/irgen/test/Observations.test.scala, model/irgen/testdata/observations/**, model/migrate/**, model/README.md, model/framework/SEMANTICS.md]

### Approach

- Start from local `umpire` after the committed central-Bounds source checkpoint. Re-anchor the
  current two-field API, named-by-val folding in `Constants.constString`, source comments and
  README. The mini has no author-maintained `Observation` declaration; do not add an artificial
  observation to the permanent mini merely to make this task's proof look exercised.
- Independently prove the native API has only name/on and rejects the retired named `read`
  argument. Prove inferred and explicit observation names, aliases and machine evidence references
  in isolated compiled/lifted fixtures, rather than merely searching for the removed parameter.
- A typed `Observed[Message]` names the protobuf message; the actual read's typed response selector
  names the response field. Do not claim an `Observed` token by itself proves that field.
  An isolated realization/read fixture must independently assert the observation identity/message,
  exact typed response path and matching hint comment. Include an independent wrong/invalid-path
  control at the existing compiler/lifter boundary, without adding a new runtime policy.
- Verify the retired text was not carried by the Scala lowering or shared IR observation transport.
  Existing `Constants` folds the declaration's first name argument; the schema's `Observed` carries
  id/position/message, and actual read paths are separate metadata. Bounded read-only inspection
  of existing Go lint/lowering may establish their use of observation ids/messages and read targets,
  but no Go tests/changes, live/Case verdict or whole-consumer credit is authorized. If a concrete
  consumer of the old free text is found, report it before changing behavior.
- Ship actual compiled `Fn159Task6...` Scalafix source under the established convention. Remove
  the provably literal retired label from supported Observation constructors, preserving name/on,
  typed read selectors, comments and unrelated APIs. Ambiguous/shadowed constructors, unsupported
  argument shapes and nonliteral expressions whose evaluation cannot safely be dropped refuse
  atomically with a concrete manual recipe. Prove transformation, idempotence and refusal controls.
- Reuse the existing `model/migrate` harness. A copied-mini source proof may carry an explicitly
  isolated scratch Observation probe alongside the copied feature, since the actual mini has no
  Observation call to migrate. The actual compiled rule must transform that probe and restore its
  exact accepted source. Compile/lift the restored probe and copied mini independently. Label
  scratch-probe evidence separately; do not claim a real mini observation was changed or exercised.
  If retired-API pre/post compilation is needed, use the actual old signature only in an isolated
  scratch framework, never the current checkout or real-feature tree. Do not normalize away
  metadata/positions or substitute a no-op run for transformation evidence.
- Keep framework comments/README accurate and all thirty source-cited mini guide excerpts exact.
  Existing integrated documentation may already satisfy this; no manufactured source edits.

### Verification and boundaries

Fresh selected mini/shared compile, focused native Observation API controls, isolated name/typed-read
lifter fixtures, focused migration/source-citation tests, exact copied-source restoration and complete
raw mini IR/waiver comparison are this source checkpoint's gates. Do not invoke the broad Fixtures
class to filter one case. Freeze source/jar inputs before proofs and do not overlap formatting/edits.
Use matching immutable jar-based classpaths rather than mutable Scala compiler output roots.
Independently check emitted fixture metadata and preserved mini declarations/positions.

Run direct `FEATURES=activity_mini` Scala `--check-equivalence ... --steps-only` against
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`
with a separate justified task map. It is the complete pre-change task8-jar selected capture,
sixteen tables/11,325 rows including shared Query roots. Both this and the retained original
ten-table baseline must pass their integrity scripts; neither capture nor the old batch-1d map
may be changed or recaptured. Run changed-file formatting/diff checks and retain actual failed
attempts. Checked-in generated IR/Cases and real-feature/legacy payloads stay untouched.

Real-feature observation removals/hint proofs, managed regeneration and canonical/consumer gates
remain held; existing Duration/protobuf prerequisites do not block this source-only closure.
No Go/proto/schema/Make/CI/runtime/Case/live/production loader changes are authorized.
Lint/manual acceptance is NOT RUN. Broad validation and independent review run at the ONE
Scala batch-end gate, not here. No SHIP, canonical-green, full-consumer or real-feature credit.

## Acceptance

- [ ] The native two-field API and retired-argument refusal are independently tested; isolated
  compiled/lifted inferred/explicit observation names, aliases and machine evidence stay correct.
- [ ] Typed observation identity/message and the actual response selector/path are independently
  asserted, with a meaningful invalid-path control and an accurate nearby response-field hint.
- [ ] Bounded source/schema/consumer inspection finds no reader of the retired free text; actual
  typed read metadata is distinguished from that inert label. Unrun consumer gates stay held.
- [ ] Task-named compiled Scalafix source has real literal-label transformation, idempotence and
  atomic ambiguity/nonliteral refusals; an explicitly isolated copied-mini probe proves source
  restoration and compilation/lift, without inventing an authored mini observation.
- [ ] Fresh focused source gates, raw complete mini IR/waiver preservation, direct sixteen-table
  native equivalence and both frozen baseline integrity checks pass; protected sources/artifacts,
  pins, positions and historical map are unchanged.
- [ ] Framework/lifter/README comments remain accurate and exact guide citations pass; failed
  attempts, owner-held real/consumer obligations and NOT RUN lint/broad-review gates are recorded
  honestly rather than reported as passing.

## Done summary
Temporal authors now have independent proof that Observation's two-field API preserves inferred and explicit evidence names, aliases and typed response reads. A compiled task-named Scalafix rule removes supported literal retired labels, with atomic manual refusals for ambiguous, effectful or unsupported forms; the authored mini remains unchanged because it declares no Observation.

Tier: session(jev-unavailable(no_key)); AGENTS implementer gpt-6.1-sol medium retained.
stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers broad validation and independent review to the ONE Scala batch-end gate)

Baseline: green fresh selected mini/shared package and exact GuideSourcesTest (one executed test), before any task edit. No full-suite Quick command is defined for this source checkpoint. Gate classify returned FULL on external MILESTONES.md; only the authorized focused gates ran, with no docs-only skip or full-suite receipt claimed.

Native and typed lift proof:
- ObservationTest executes two tests. Product fields are exactly name/on, arity two, explicit names survive and the native default name is empty. Named read is refused with the actual retired-parameter diagnostic; only its arity companion is permitted. Positional third arguments fail with the arity diagnostic.
- ObservationsTest packages only its isolated fixture against the fresh task6 scoped-model.jar and frozen dependency classpath. Two tests assert inferred and explicit IDs, alias identity, exact machine evidence text, ActivityExecutionInfo message identity, response path info and both observe targets. The nearby hint states that info is the whole message, rather than its attempt scalar.
- A separately compiled wrong-name source emits wrong-info and fails the independent explicit-info expectation. A constant non-field selector compiles but fails at the existing lifter field-path boundary and produces no IR. No new native validation policy was added.
- Constants.scala folds constructor name first and resolves default Observation names through the declaring val. Realizations.scala emits typed Observed id/message separately from actual response read metadata. The shared schema's Observed contains only id/position/message. Bounded read-only lowering/lint inspection uses observation id/message and observe/lift targets, with no reader of the retired free text found. This is bounded source inspection, not consumer gate credit.
- The existing removal commit 50bc14c2fa9deec636528c0524dcd6cf0a851f90 and its parent's actual retired signature were inspected; the removal was not repeated. README and framework comments were corrected where they conflated Observed message identity with a scalar selector, and the README distinguishes native blank names from lifter val-name folding.

Migration and copied-source proof:
- Fn159Task6ObservationRead is actual compiled Scalafix 0.14.9 source in the existing Scala 3.9/JDK25 harness. Four table-driven tests exercise named and positional literals, selective/wildcard and rooted constructors, idempotence, preserved comments/strings/typed selectors/unrelated APIs, shadowed bindings/imports/packages, unsupported arity/keys/constructor aliases/new/explicit .apply forms, and nonliteral expressions.
- Bare constructors require one unambiguous framework import; framework.Observation requires an unshadowed package. _root_.framework.Observation resolves that package explicitly. Refusals are whole-document atomic and retain a concrete manual recipe. No generic resolver, new dependency or production loader was added.
- Final proof directory is .flow/tmp/mini-migration.UP54vn. Its separate observation-probe contains accepted.txt, legacy.txt, fixed.txt and independently compiled accepted/fixed jars and lifts. The actual compiled rule transforms both retired forms, restores exact accepted source bytes, converges and refuses an effectful label atomically. Complete probe metadata and positions compare byte-identically. Legacy text is parsed and transformed; no old whole-repository tree or old-API compilation is claimed.
- Copied mini/shared roots compile and lift before/after independently. Complete raw mini IR and waiver bytes compare identically, including every identity, body and position. Both final IR SHA256 values are b38f5fe304ee9d6c0de6aad646e39a68c36cc5ecad3b071cf169e57c801d92d9. Fourteen declaration-section inventories and twenty located Queries were independently checked on the fresh copied output. No position normalization or baseline rewrite occurred.
- GuideSourcesTest executes one passing test requiring exactly thirty source-cited excerpts, each matching its source and line. SEMANTICS and authored mini sources are unchanged.

Native equivalence and freeze:
- Direct FEATURES=activity_mini steps-only equivalence passed against the complete immutable mini-selected-baseline-00d22597d2 capture. It wrote and compared sixteen native tables; the baseline integrity verifies 11,325 rows and its unchanged original ten-table subset.
- Task6 mini.map contains only a justification comment, with no identity/removal/projection entries. Both complete and original baseline integrity scripts passed. The original task8 scoped-model.jar still has SHA256 92c23185a3acd23ab00b81ce73dddc99295562c7f6c96d5a319b05e96778666e. Neither baseline nor the historical batch-1d map was edited or recaptured.
- All source and formatting edits finished before the final UP54vn proof. No edits or formatting overlapped its frozen package/lift inputs. Every lift uses frozen jars in place of mutable first classpath entries. All owned process handles reached terminal exit before closure.

Final green observations under .flow/tmp/fn159/task6:
- baseline-package.log, baseline-guide.log, package-final.log and native-final.log (two tests).
- lifter-refrozen.log (two tests, including the compiled wrong-name control), migration-unsupported-final.log (four tests), guide-final.log (one thirty-excerpt test).
- proof-unsupported-final.log and equivalence.log, both exit 0.
- format-unsupported-check.log confirms all seven changed Scala files formatted; working and staged git diff --check passed.

Retained failed/setup observations:
- migration-red.log exited 1 with all four tests failing against the no-op rule. migration-first.log exposed an unknown named-key constructor escaping atomic refusal; migration-second.log passed after that gap closed.
- migration-package-shadow-red.log exposed an imported framework package escaping qualified-constructor refusal. Rooted package resolution and package-shadow guards corrected it. migration-unsupported-red.log exposed a rooted new constructor escaping atomic refusal; the bounded new/explicit .apply refusal completed the error table.
- native-first.log/native-diagnostic.log/native-second.log failed on an arbitrary single-diagnostic assumption. The logs retain the compiler's actual retired-parameter and arity diagnostics; the final assertion checks the intended refusal and excludes unrelated diagnostics.
- lifter-first.log failed under Werror for an unused import and discarded intercept result. lifter-second.log lacked frozen dependency jars and failed fixture compilation; lifter-third.log used an incorrect top-level Scala root. lifter-fourth.log looked up Function.id instead of Function.name. Those setup errors were corrected without weakening emitted-value expectations; the final typed proofs pass.
- proof-final.log at ONSFpJ refused the fixture's bare constructors under competing wildcard imports. The isolated accepted probe now spells _root_.framework.Observation explicitly; ZjsKpJ, APXUjI and final UP54vn proofs pass.
- format-check.log returned 0 while reporting ParseError on Scalafmt's own wrapped guard. That observation is INCONCLUSIVE, not green. A shorter guard was formatted and checked successfully with the same formatter; the final seven-file check reports all formatted.
- Two apply_patch attempts failed exact-context matching and made no changes; read-only discovery named a nonexistent proof path and an absent expected JSON path. Actual established paths were subsequently used.
No failed or inconclusive observation supplies passing gate credit.

Protected and deferred obligations:
- NOT RUN: lint/manual acceptance; broad validation and independent review; canonical Make, linked-protobuf/Duration consumer prerequisites, Go/proto/schema/CI/runtime/Case/live/regeneration, production custom-rule loading, standalone/Nexus/legacy migration or lift, and full-set gates.
- The owner retains real-feature Observation removal/hint proofs and consumer/canonical closure at the sweep. This checkpoint claims no SHIP, whole-spec completion, canonical-green or full-consumer proof.
- The nine admitted model files and this Flow receipt are the task's scope. All seventeen external owner Scope specs, MILESTONES and untracked fn168 planning files remain preserved and unstaged. Runtime source changes are limited to one framework comment; prior macro/phase/Bounds behavior, generated payloads, pins, historical captures/maps and guide positions remain intact.
- No branch switch, merge, push, amend, rebase or history rewrite occurred. The conductor owns integration into local umpire and next-task admission.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9335740963b1144c7de56ed25a17176117f074a7
- Tests: mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/framework model/temporal/Bounds.scala model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/realize model/temporal/features/activity/Activity.scala model/temporal/features/activity_mini -f -o .flow/tmp/fn159/task6/scoped-model.jar, mise exec -- scala-cli test --server=false model/project.scala model/framework --test-only framework.ObservationTest, mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.ObservationsTest, mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.ObservationReadTest, mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.GuideSourcesTest, timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteObservationMini, FEATURES=activity_mini mise exec -- scala-cli run --server=false model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/ /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task6/mini.map --steps-only, node .flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/integrity.mjs, node .flow/tmp/scala-dsl/mini-baseline-00d22597d2/integrity.mjs, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/Action.scala model/framework/Observation.test.scala model/irgen/test/Observations.test.scala model/irgen/testdata/observations/Observations.scala model/migrate/rules/Fn159Task6ObservationRead.scala model/migrate/test/ObservationRead.test.scala model/migrate/proof/RewriteObservationMini.scala, git diff --check, git diff --cached --check
- PRs:
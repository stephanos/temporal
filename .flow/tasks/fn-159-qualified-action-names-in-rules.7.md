---
satisfies: [R14]
---
# fn-159-qualified-action-names-in-rules.7 enters(<phase>) replaces trivial phase-only effects

## Description

Close and complete the current mini/shared source migration to `enters(Phase.X)`. The framework
primitive, typed Phased writer and most selected uses are already integrated. Reuse them rather
than choosing a competing writer design or replaying the real-feature migration. Two actual
phase-only effects remain in the mini's protocol Answering facet: `backOff` and `resume`.
The owner Scope excludes standalone/Nexus/legacy edits, lifts, regeneration and consumer work.

**Size:** M
**Files:** existing enters primitive/lifter, focused native/isolated lift controls, the remaining
mini phase-only effects and their concrete phase writer, task-named migration proof and accurate docs.
**Touches:** [model/framework/Syntax.scala, model/framework/Effects.test.scala, model/framework/Entering.test.scala, model/framework/Phased.test.scala, model/irgen/Syntax.scala, model/irgen/Declarations.scala, model/irgen/Context.scala, model/irgen/Conventions.scala, model/irgen/test/Entering.test.scala, model/irgen/testdata/entering/**, model/irgen/testdata/lifts/Entering.scala, model/temporal/features/activity_mini/**/*.scala, model/temporal/actors/worker/Serving.scala, model/temporal/actors/worker/Worker.scala, model/temporal/foundations/**/*.scala, model/temporal/capabilities/**/*.scala, model/migrate/**, model/README.md, model/SEMANTICS.md]

### Approach

- Begin after the committed qualification/source checkpoints are integrated into local `umpire`.
  Current `framework.enters` uses the explicit writer of `Phased[S,P]`, inherited through derived
  subjects/facets, returns one ok step and records exactly the target phase's declared status when
  its type is Recorded; an unrecorded phase adds no facts. Preserve this existing design/boundary.
- Establish focused native controls for rule effects and actual choose alternatives, Recorded
  and plain phases, retained non-phase fields, inherited/nested writer paths and existing missing/
  miswritten writer refusals. Do not merely assert the new form constructs or compare it to itself.
  Reuse the existing EffectsSuite evidence and isolated Entering fixture where appropriate.
- Package isolated lifter roots rather than invoking broad Fixtures.beforeAll. The old Entering
  fixture has Recorded Stage, a narrower role-bearing case and a counter; its core and sugar
  subjects are useful controls. Independently assert outcomes, target state, exact facts and choice
  alternatives, with explicit identity/source-span accounting. Do not launder unrelated fields by
  globally removing positions or blindly renaming arbitrary JSON strings.
- Survey only selected mini/shared sources for phase-only effects and aliases. Current
  `protocol/Answering.scala` has `backOff`/`resume`, each `enter(s.copy(phase = AnswerPhase.X))`;
  replace the rule uses by qualified enters and remove those definitions. Keep poll's attempt
  increment, send/channel logic, receive/trust's conditional phase and evidence facts, rejection,
  loss and hole reach intact. Shared `actors/worker/Serving.scala` already uses enters for stop/
  resume; retain its meaningful serve and verify that source closure rather than repeating it.
- `Answering` currently has no Phased parent, and concrete `AnswerDispatch` is a plain Machine
  with an explicit end. Supply the existing typed projection/writer through the facet/concrete
  machine so enters works and lifts: exactly copy the phase, preserve attempts/inbox and preserve
  the explicit end. Its AnswerPhase enum is plain/unrecorded; do not add status facts or roles to
  make the migration convenient. Do not bypass initialization or add a competing local writer
  that the lifter cannot validate. If a writer changes emitted metadata, account for it explicitly
  and prove it does not change states/rows/outcomes/facts/claim or Query semantics.
- Ship actual compiled `Fn159Task7...` Scalafix source under the shared convention. Supported
  phase-only effect definitions and their rule/call/choose uses migrate together and unused
  definitions disappear; preserve genuine logic. Prove the phase writer and Recorded/status
  relationship from actual source/type information, not a name heuristic or absence of a local
  enum. Ambiguous/external/unproven phase status or unsafe exported usages refuse atomically with
  a concrete manual recipe. Reuse the pinned harness, no generic rewrite engine or production
  loader. Prove real transformation/refusal/idempotence on copied-mini sources and compiled
  semantics; a no-op run is not migration evidence.
- Model README, SEMANTICS and exact source-cited mini guide must show the current core form,
  supported rule/choose forms and typed writer. Update actual excerpts/line references accurately,
  not the oracle to conceal a mismatch.

### Verification and boundaries

Capture an immutable selected current pre-edit source jar/lift in task-specific scratch before the
mini edits. Preserve the earlier task8 jar, complete native baseline and old batch-1d map; do not
fold new source into any protected baseline. Run fresh mini/shared compile and targeted native,
isolated lifter and migration/citation checks. Freeze all source/jar inputs before final proofs;
no overlapping formatting or source edits while package/lift/proof runs.

This task can change function identities/call bodies/fingerprints and source spans when phase-only
definitions are removed. Compare complete raw pre/post lifts and explicitly inventory each expected
function/body/span change. Independently require unchanged action/class/type/state/property/Query/
realization identities, budgets, totals and ordinary/negative-control semantics. No blanket position
projection, removal map or golden rewriting supplies that proof. Exact native step-table comparison
must preserve every selected row/outcome/fact. If a copied migration restores exact accepted source,
its full raw IR/waiver bytes also compare exactly; keep that separate from the authored pre/post diff.

Run direct `FEATURES=activity_mini` Scala `--check-equivalence ... --steps-only` against
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`
with a separate justified task map. The complete immutable selected task8-jar capture contains
sixteen tables/11,325 rows including shared Query roots. Both its and the retained original
ten-table baseline's integrity scripts must pass. Never recapture/edit either or alter the old map.
Run changed-file formatting/diff checks and preserve all actual failed/RED attempts.

Real-feature migration/IR/Case/fingerprint publication and canonical consumer gates stay held.
No Go/proto/schema/Make/CI/runtime/Case/live/production loader or checked-in generated changes.
Lint/manual acceptance is NOT RUN. Broad validation and independent review occur at the ONE
Scala batch-end gate; no per-task SHIP, full-set, consumer or canonical-green claim.

## Acceptance

- [ ] Existing enters writer semantics are independently proven for rules/choose, Recorded/plain
  phases and retained fields, with meaningful missing/miswritten writer refusals and isolated lifts.
- [ ] Selected mini/shared sources retain no demonstrably phase-only named effect or alias;
  actual Answering backOff/resume uses migrate with the correctly typed concrete phase writer,
  preserving the plain enum, explicit end, meaningful effects, facts, hole and rejection behavior.
- [ ] Complete raw authored pre/post lift differences are explicitly inventoried and independently
  limited to justified function/call/span/writer bookkeeping; ordinary declarations, identities,
  Query budgets/totals, state/outcome/fact semantics and all sixteen native tables remain unchanged.
- [ ] Task-named compiled Scalafix source genuinely transforms supported phase-only definitions/
  uses, converges and refuses unproven/unsafe cases atomically; actual copied-mini compilation/lift
  proof is recorded separately from the authored migration and protected baselines are unchanged.
- [ ] Docs and exact guide excerpts reflect current source; focused source/native gates, both
  frozen baseline integrity checks and changed-file format checks pass with recorded real exits.
- [ ] Real/consumer/regeneration obligations and lint/manual/batch-end broad review are recorded
  as held or NOT RUN, never credited as passes; protected payloads/pins/maps and prior findings stay.

## Done summary
Mini authors now enter AnswerPhase.backingOff/waiting through AnswerDispatch's explicit typed Phased writer. Both trivial effects disappear; attempts, inbox, plain-enum facts, explicit end, poll/send, conditional receive/trust, rejection, loss and interrupt hole retain their semantics.

Tier: session(jev-unavailable(no_key)); AGENTS gpt-6.1-sol medium implementer retained.
stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers broad validation and independent review to the ONE Scala batch-end gate)

Baseline was green from fresh pre-edit selected packaging and complete lift. Immutable sources/jar/classpath/IR are under .flow/tmp/fn159/task7/pre, based at 86072c481bad61dc620904a13b1a0ca63a190c8b. No full-suite Quick command is defined. Gate classify returned FULL on external MILESTONES.md; authorized focused gates supplied the evidence, with no full-suite receipt credit.

Final focused checks pass with recorded exit 0. EffectsTest executes 8 tests; AnswerEnteringTest 3, including exact plain steps, retained fields and an inherited nested writer; EnteringTest 2 isolated Recorded/core/choose and missing-writer controls; EntersTest 4 transform/convergence/atomic-refusal tests; GuideSourcesTest 1 exact thirty-excerpt test. All eight changed Scala files are formatted, and working/staged diff checks pass. See native-final.log, mini-final.log, isolated-frozen.log, migration-final.log, guide-first.log and format-check-final.log under .flow/tmp/fn159/task7.

The compiled Fn159Task7Enters proves explicit state/phase types, projection/copy writer and actual plain or Recorded status declarations. Private owners work locally; facets/public source sets require explicit context and caller inventory. Unknown/exported escapes, lookalike/competing imports and missing/miswritten writers refuse atomically with a manual recipe. Meaningful facts/state writes/rejection/choices stay. The actual copied-mini transformation restores accepted source bytes and complete IR/waivers exactly in .flow/tmp/mini-migration.PTGgMk. Final proof-final.log passes; both IR hashes are b153c4b8eb2e0314d33add4a35b43eda712cb8bf9d3591086aeb1dcd2e5b0728. Legacy text is parsed/transformed; old-API compilation is not claimed.

raw-delta.json inventories complete authored pre/post lifts. Exactly two functions disappear and two rule calls inline their independent core Step bodies. Its closed source-coordinate ledger accounts for 381 position/body changes across Answering/AnswerDispatch. All other values/identities/order, fourteen sections and twenty Query budgets/totals match. Six independently mutated fact/state/outcome/choice/total/source controls fail. Byte digests are not Go Behavior Fingerprints. Direct steps-only equivalence passes all sixteen tables against the immutable complete baseline. Both integrity scripts pass, including 11,325 rows and the original ten-table subset; task8 jar SHA256 remains 92c23185a3acd23ab00b81ce73dddc99295562c7f6c96d5a319b05e96778666e. The task map has comments only. See raw-delta.log, delta-controls.log and equivalence.log.

proof-notes.md retains detailed reasoning and RED/setup observations. migration-red.log fails four tests against the no-op rule; mini-red-corrected.log records absent Phasing before the fix. Compilation/import/root-selector/setup failures remain in their logs. The jar-overlap isolated-second run is INCONCLUSIVE and supplies no credit; isolated-frozen is its fresh replacement. Final inputs remained frozen and all owned process handles are terminal. The Flow receipt was refreshed solely to shorten this summary and replace one displayed classpath placeholder with the actual shell command.

NOT RUN or held: lint/manual acceptance, ONE batch-end broad validation/independent review, standalone/Nexus/legacy sweep/lift, production loader, canonical Make/linked-protobuf/Duration, Go/proto/schema/CI/runtime/Case/live/regeneration, checked-in payload/fingerprint publication and full-set/consumer gates. No SHIP, whole-spec or canonical-green claim. Thirteen admitted model files and this task receipt form commit scope. Seventeen external Scope specs, MILESTONES and untracked fn168 plans remain preserved/excluded. Protected captures/pins/maps/payloads and prior findings/fixes are unchanged. The conductor owns integration and next-task admission; no branch switch, merge, push or history rewrite occurred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8160e9ea1d7747170e1404c6ba36d1d527ca90d2
- Tests: mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/framework model/temporal/Bounds.scala model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/realize model/temporal/features/activity/Activity.scala model/temporal/features/activity_mini -f -o .flow/tmp/fn159/task7/scoped-model.jar, mise exec -- scala-cli test --server=false model/project.scala model/framework --test-only framework.EffectsTest, mise exec -- scala-cli test --server=false model/project.scala model/temporal/features/activity_mini/protocol/Entering.test.scala --classpath "$(<.flow/tmp/fn159/task7/scoped-model.classpath)" --test-only framework.AnswerEnteringTest, mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.EnteringTest, mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.EntersTest, mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.GuideSourcesTest, timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteEntersMini, node model/migrate/proof/check-enters-delta.mjs .flow/tmp/fn159/task7/pre/lift/activity-mini.json .flow/tmp/fn159/task7/post-formatted/activity-mini.json .flow/tmp/fn159/task7/raw-delta.json, node model/migrate/proof/test-enters-delta.mjs .flow/tmp/fn159/task7/pre/lift/activity-mini.json .flow/tmp/fn159/task7/post-formatted/activity-mini.json .flow/tmp/fn159/task7/delta-controls, FEATURES=activity_mini mise exec -- scala-cli run --server=false model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/ /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task7/mini.map --steps-only, node .flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/integrity.mjs, node .flow/tmp/scala-dsl/mini-baseline-00d22597d2/integrity.mjs, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/Effects.test.scala model/irgen/test/Entering.test.scala model/temporal/features/activity_mini/protocol/Entering.test.scala model/temporal/features/activity_mini/protocol/Answering.scala model/temporal/features/activity_mini/protocol/AnswerDispatch.scala model/migrate/rules/Fn159Task7Enters.scala model/migrate/test/Enters.test.scala model/migrate/proof/RewriteEntersMini.scala, git diff --check, git diff --cached --check
- PRs:
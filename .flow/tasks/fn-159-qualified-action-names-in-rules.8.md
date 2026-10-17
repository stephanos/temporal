---
satisfies: [R15]
---
# fn-159-qualified-action-names-in-rules.8 Rename capability kinds Retries → Retryable, Deadline → Expirable

## Description

Close the current mini/shared source proof for the owner-confirmed capability names Retryable and
Expirable. Both classes/companions, files and selected uses already have those names under
`model/temporal/capabilities/system/`. Inspect and prove the existing migration rather than replay
it or ask for the settled names again. Real-feature references, managed IR/Cases and Go consumers
remain at the sweep under the current owner Scope.

**Size:** S–M
**Files:** existing system capability source/tests, independent name/origin and selected-reference
controls, actual task-named source migration/proof and accurate author documentation.
**Touches:** [model/temporal/capabilities/system/Retryable.scala, model/temporal/capabilities/system/Retryable.test.scala, model/temporal/capabilities/system/Expirable.scala, model/temporal/capabilities/system/Expirable.test.scala, model/temporal/capabilities/Names.test.scala, model/temporal/features/activity_mini/**/*.scala, model/temporal/foundations/**/*.scala, model/temporal/actors/**/*.scala, model/irgen/test/CapabilityNames.test.scala, model/irgen/testdata/capabilityNames/**, model/migrate/**, model/README.md, model/framework/SEMANTICS.md]

### Approach

- Start from the preceding committed source checkpoint integrated into local umpire. Freshly
  compile selected mini/shared roots and run focused current capability tests. Do not run the
  broad Properties suite: it imports real activity/Nexus Models, which this Scope excludes.
- Independently assert the two actual class/companion names, declaration files and current
  selected uses. Test generated Property kind/origin IDs and actual mini instance identities,
  not only strings returned by a rewritten fixture. Existing mini Completion has Retryable,
  Expirable, a retry override and a scheduleDeadline binding that must remain named as it is.
- Keep Property bodies, applicability, waiver/override reasons, Query budgets/totals and all
  native step outcomes/facts unchanged. Other deadline vocabulary (timer objects, inputs, kit
  values, binding names, Timeout types) is explicitly not part of the rename. Independently
  inventory those relevant selected declarations as preservation controls.
- Ship actual compiled Fn159Task8... Scalafix source in the shared naming convention. Rewrite
  provably owned legacy capability declarations/references/imports and override/bound member
  uses, leaving unrelated Deadline/Retries symbols, strings, comments and deadline binding names
  untouched. Refuse ambiguous imports, shadowing and unsupported ownership atomically with a
  manual recipe. Source rule file-name handling outside syntactic patches must be documented;
  production loading and real-feature application stay held.
- Prove transformation, idempotence and refusal with meaningful independent controls and an
  actual copied-mini legacy-to-current application. Restore exact accepted source and independently
  compile/lift it. Legacy text can be parsed/transformed without old-API compilation; label that
  boundary honestly. Reuse the pinned harness; no generic rename engine or new dependencies.
- README/SEMANTICS and exact source-cited mini excerpts must use current capability names. Existing
  source/docs may already satisfy the contract; do not manufacture runtime edits to close missing
  proof. Migration documentation may explicitly explain the old-to-new names.

### Verification and boundaries

Fresh selected compile, focused Retryable/Expirable native tests, isolated name/origin lifter
fixtures, actual mini metadata/inventory controls, targeted migration/citation tests and raw complete
copied-mini IR/waiver comparison are this step's gates. Capture current artifacts after task7;
do not assume its earlier b38... IR digest remains current once phase-only effects are removed.
Freeze all source/jar inputs before final proofs; no overlapping edits/formatting. Use frozen jars
in place of mutable first classpath roots. Compare complete bytes, not position-normalized output.

Run direct FEATURES=activity_mini Scala `--check-equivalence ... --steps-only` against
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`
with a separate justified task map. This complete immutable task8-jar capture is sixteen native
tables/11,325 rows including shared Query roots. Both its and the original ten-table baseline's
integrity scripts must pass; neither baseline nor the old batch-1d map/pins may be changed or
recaptured. Run changed-file formatting/diff checks and record actual failed/RED attempts.

The historical renamed real-feature origins/IR/Case/Go proof and canonical gates remain held.
No Go/proto/schema/Make/CI/runtime/Case/live/production-loader or generated-artifact changes/runs.
Lint/manual acceptance is NOT RUN. Broad validation and independent review occur at the ONE
Scala batch-end gate. No SHIP, canonical-green, full-consumer or real-feature credit.

## Acceptance

- [ ] Current classes/companions/files and selected references use the confirmed Retryable/Expirable
  names; independent emitted kind/origin and actual mini-instance controls pass.
- [ ] Other deadline vocabulary, actual Property bodies/applicability, waivers/overrides, declaration
  identities and Query budgets/totals are preserved; no real-feature or consumer proof is claimed.
- [ ] Task-named compiled Scalafix source performs supported owned-symbol renames, preserves
  unrelated names and converges, with atomic ambiguity/shadow/unsupported refusals and documented
  file-name/manual/production boundaries. Actual copied-mini transformation compiles and lifts.
- [ ] Full current mini IR/waiver bytes and metadata/position inventories remain exact; all sixteen
  native tables compare with the frozen baseline, both integrity scripts pass and protected
  sources/payloads/pins/maps are unchanged.
- [ ] Accurate docs and exact guide citations pass focused gates and changed-file checks; failed
  attempts and held consumer/lint/manual/batch-end broad-review obligations are recorded honestly.

## Done summary
Proved the already-current Retryable/Expirable classes, companions, declaration files and selected mini uses without runtime edits. Fn159Task8CapabilityNames migrates explicitly owned declarations/imports/references, overrides and bounds, preserves unrelated vocabulary, converges and refuses ambiguous/shadowed/unsupported ownership atomically with a manual recipe.

Tier: session (jev-unavailable(no_key)); AGENTS implementer gpt-6.1-sol medium retained.
stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers broad validation and independent review to the ONE Scala batch-end gate)

Baseline: green from fresh selected package/complete lift and seven exact native Retryable/Expirable tests. No full-suite Quick command is defined; gate classify reports FULL for external MILESTONES.md, with no canonical/full-suite receipt credit. Final focused gates pass: five migration tests, one compiled class/companion test, two isolated actual-kind/mini metadata tests and one exact thirty-citation guide test. Changed-file formatting, working/staged diff, frozen-source/jar and external-preservation checks pass.

Final copied-mini proof is .flow/tmp/mini-migration.oqLfde: actual Completion legacy references restore accepted source exactly; accepted/restored packages and full raw IR/waivers compare byte for byte, including positions. IR remains b153c4b8eb2e0314d33add4a35b43eda712cb8bf9d3591086aeb1dcd2e5b0728. Legacy text is parsed/transformed, not old-API compiled. Actual mini identities, applicability, Property bodies, override reason and Query budgets/totals remain fixed. Direct steps-only equivalence compares all sixteen tables; both protected integrity scripts pass (11325 complete rows; original subset retained for integrity only). The map permits no renames/removals/projection.

proof-notes.md and ignored logs retain no-op RED, fixture setup errors and the observed import-visibility RED/fix. Final inputs stayed frozen and all command handles are terminal. README/SEMANTICS were already current; migration docs state file-name/manual/production boundaries.

Held/NOT RUN: lint/manual acceptance, ONE batch-end broad validation/independent review, real-feature/legacy sweep/lift, production loader, canonical Make/linked-protobuf, Go/proto/schema/CI/runtime/Case/live/regeneration, full-set/consumer gates and payload/fingerprint publication. No SHIP or canonical-green claim. Prior findings/fixes and protected payloads/pins/maps/baselines are unchanged. Owner Scope specs, MILESTONES and fn168 plans are preserved/excluded; no branch/history/push operation occurred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 95f404d11d33ddf997e4ea6089415627d8d1f5d5
- Tests: mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/framework model/temporal/Bounds.scala model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/realize model/temporal/features/activity/Activity.scala model/temporal/features/activity_mini -f -o .flow/tmp/fn159/task8/scoped-model.jar, mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/irgen -M umpire.irgen.lift -- --ir /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task8/scoped-model.jar=model/ .flow/tmp/fn159/task8/scoped-model.classpath .flow/tmp/fn159/task8/current activity-mini, mise exec -- scala-cli test --server=false model/project.scala model/temporal/capabilities/system/Retryable.test.scala model/temporal/capabilities/system/Expirable.test.scala --classpath "$(<.flow/tmp/fn159/task8/scoped-model.classpath)" --test-only 'framework.Capability*Suite', mise exec -- scala-cli test --server=false model/project.scala model/temporal/capabilities/Names.test.scala --classpath "$(<.flow/tmp/fn159/task8/scoped-model.classpath)" --test-only framework.CapabilityNamesSuite, mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.CapabilityNamesTest, mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.CapabilityNamesTest, mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.GuideSourcesTest, timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteCapabilityNamesMini, FEATURES=activity_mini mise exec -- scala-cli run --server=false model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/ /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task8/mini.map --steps-only, node .flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/integrity.mjs, node .flow/tmp/scala-dsl/mini-baseline-00d22597d2/integrity.mjs, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/migrate/rules/Fn159Task8CapabilityNames.scala model/migrate/test/CapabilityNames.test.scala model/migrate/proof/RewriteCapabilityNamesMini.scala model/temporal/capabilities/Names.test.scala model/irgen/test/CapabilityNames.test.scala model/irgen/testdata/capabilityNames/Names.scala, sha256sum -c .flow/tmp/fn159/task8/frozen-sources.sha, cmp .flow/tmp/fn159/task8/external.patch .flow/tmp/fn159/task8/external-final.patch, sha256sum -c .flow/tmp/fn159/task8/external-untracked.sha, git diff --check, git diff --cached --check
- PRs:
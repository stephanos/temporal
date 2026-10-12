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
**Touches:** [model/temporal/capabilities/system/Retryable.scala, model/temporal/capabilities/system/Retryable.test.scala, model/temporal/capabilities/system/Expirable.scala, model/temporal/capabilities/system/Expirable.test.scala, model/temporal/capabilities/Names.test.scala, model/temporal/features/activity_mini/**/*.scala, model/temporal/foundations/**/*.scala, model/temporal/actors/**/*.scala, model/irgen/test/CapabilityNames.test.scala, model/irgen/testdata/capabilityNames/**, model/migrate/**, model/README.md, model/SEMANTICS.md]

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

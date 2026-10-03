---
satisfies: [R2, R3, R4, R5, R6, R7, R8, R9]
---
# fn-117-type-the-temporal-api-in-the-models.6 Migrate the standalone activity to typed API declarations

## Description
Migrate the activity's author surface without performing fn-112's showcase rewrite.

**Size:** M
**Files:** model/temporal/standaloneactivity/Model.scala and Realization.scala; regenerated model/ir activity outputs only when positions move; model/lifter/Realizations.scala and focused lifter fixtures for the typed Long-operand folding gap exposed by the migration.
**Touches:** [model/temporal/standaloneactivity/Model.scala, model/temporal/standaloneactivity/Realization.scala, model/ir/**, model/umpire/realize/Realize.scala, model/lifter/Realizations.scala, model/lifter/test/Fixtures.test.scala, model/lifter/testdata/lifts/Typed.scala, model/lifter/testdata/typedInvalid/Invalid.scala, model/cases/manifest.json, model/lifter/testdata/lifts/expected/admission.json, model/lifter/testdata/lifts/expected/realizations.json, .flow/tmp/fn117-6/**]

### Approach
- Snapshot the current decoded IR and Case/fixture bytes before editing. Replace every API schema/method/path/enum/constant-message declaration in these two files with tasks3-5's typed forms, including typed status/helper parameters. Keep action/machine/property/Scenario/Query/realization IDs, scripts, waits, roles and behavior unchanged.
- Preserve comments and layout outside the affected code. Do not split System.scala, derive/copy machines differently, introduce value-name capture or the script helper kit; those are fn-112/fn-114.
- Regenerate through make umpire-gen-model, inspect the complete IR diff, and independently compare every decoded IR field except source positions. The fn-113 golden projections are not permission for new function/type/order/parameter differences. Case JSON bytes remain unchanged. Generated expected-fixture `position` fields and Query-manifest `Position` strings may update to their corresponding moved Scala declarations; all other data, order and source files remain equal to exact pre-edit snapshots.
- Run the relevant lifter/model gate, focused migration goldens with tagged serial Go settings and model lint once ready; reuse unaffected full package results and rerun only invalidated checks. Record removed proto string counts by category and any unknown typed operand exceptions.

### Migration-discovered integration fix
- The first activity lift refuses `Operand.number(300L)` because typed operands still use the Int-only folder. Add the same Long literal and bound-helper folding already used by typed ProtoValue. Include focused equivalence coverage for both typed Operand forms, preserve the existing IR Number and Go validation, and keep the fix limited to this branch and its fixtures. This amendment is necessary for the approved typed API migration.

- The typed inline Projected origin reports the DSL definition file instead of the author file in three activity-race RunEvent key paths, which the frozen golden correctly refuses. Reuse the existing projected-path lowering helper to preserve the author callsite and add a focused origin-file assertion. Preserve root validation and all frozen Go projections. The positions-only comparator must retain source filenames, allowing only line/column changes.

- Testpilot messages are included in the no-proto-string contract. Add the missing generic typed Observed message factory, using the existing generated-descriptor full-name seam; migrate the three CorrelatedEvidence declarations, add nonmessage compiler refusal and typed-vs-legacy IR equivalence. The legacy string bridge remains only until task8. No real message construction or handwritten name table.

### Investigation targets
**Required:**
- model/temporal/standaloneactivity/Model.scala
- model/temporal/standaloneactivity/Realization.scala
- model/README.md:300-329
- tools/umpire/model/migration_golden_test.go:211-228
- tools/umpire/lower/migration_golden_test.go:223
- model/lifter/test/Fixtures.test.scala

### Quick commands
CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks; independent decoded IR positions-only comparison; focused migration goldens with -tags test_dep -p 1 -parallel 1; check-mode model gate and lint-model.

### Execution constraints
User owns commits; no worktrees/push. Preserve existing comments. No behavior, IR schema, Go lowering or fn-112 showcase work. Keep heavy Go/generation checks serial.
Migrate each Poll to the typed reference of its actual Recorded.Read/Single evidence declaration, preserving the original evidence ID and projection; do not manually annotate a root unrelated to that evidence.
## Acceptance
- [ ] The activity has no free-text proto message/method/field/enum author declarations and helper parameters are typed.
- [ ] All six decoded IRs equal the task's before snapshot except positions; Case JSON bytes and non-position fixture/manifest data remain equal and frozen migration goldens pass.
- [ ] Applicable focused/gate/lint checks pass, removed string counts and dynamic-operand exceptions are recorded.
## Done summary
Migrated the standalone activity's Temporal and Testpilot protobuf author declarations to typed schemas, methods, fields, enums, symbolic messages, and observations. Added the minimal typed Long-operand, projected-origin source-location, and generic `Observed[Message](id)` lowering needed by the migration. Machine/action/property/realization IDs, roles, scripts, waits, declaration order, and emitted behavior remain unchanged; legacy DSL forms remain for task 8.

baseline: green via task-5 handoff at HEAD 2093a63f2da7c6c66f3509414dd78476eb4d5c04 (same-session review-fixtures-2.log 23/23, review-gen-model.log, review-check-model-final.log, review-lint-model-final.log). No pre-edit gate rerun. Exact pre-edit source mirrors are under preedit/<repo-relative-path> for every touched Scala/generated file; touched-files.json lists all 13 task paths. Before editing, six IRs and hashes of 1,439 Case/fixture files were saved. The three generated files whose positions changed were later reconstructed with the mirrored pre-edit sources, and all three recovered SHA-256s matched the original frozen hashes; see reconstruct-baseline.log and before/model/{cases/manifest.json,lifter/testdata/lifts/expected/admission.json,lifter/testdata/lifts/expected/realizations.json}.

Independent final comparator: `python3 .flow/tmp/fn117-6/compare_artifacts.py` (exit 0; artifact-compare-observed-final.log). All six decoded IRs have identical keys, values, order, and source filenames; only position line numbers moved. Of 1,439 frozen Case/fixture files, 1,436 are byte-identical. The three generated metadata files are identical except same-file source-position line numbers. Manifest references resolve to the current Evidence.runEvent and activityScript declarations. No golden input, projection rule, IR schema, or Go source was changed.

Final checks (wall times measured from log creation to final write):
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- scala-cli test --suppress-outdated-dependency-warning model/lifter --test-only 'umpire.lift.Fixtures*' -- '*typed*'` — exit 0, 23.1s, green-observed-fixtures.log (four focused typed checks).
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` — exit 0, 97.2s, gen-model-observed-final.log.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/model -run '^TestMigrationGoldens$'` — exit 0, 23.8s, model-golden-observed-final.jsonl.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/lower -run '^TestMigrationGoldens$'` — exit 0, 79.9s, lower-golden-observed-final.jsonl.
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` — exit 0, 101.0s, check-model-observed-final.log (ends == ok).
- `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make lint-model` — exit 0, 15.2s, lint-model-observed-final.log. Zero current compiler errors/unused parameter diagnostics; the caught JDK 27 Scalafix NoSuchFieldException trace is inherited.
- `git diff --check` — exit 0. All original comment lines in the seven touched Scala sources remain present.

Focused controls: typed Long-operand lift first failed on `300L` (red-long-fixture.log), then passed. The projected-origin file check first failed with `umpire/realize/Realize.scala` instead of the author file (red-origin-fixture.log), then passed. Typed `Observed` first failed to compile (red-observed-positive.log), then failed in the old lifter (red-observed-lifter.log), then matched the legacy whole-realization IR. `Observed[String]` produces compiler refusal at Invalid.scala:165:49 (red-observed-invalid.log shows the new diagnostic before its expected-list update). Before the projected-origin fix, both focused Go goldens failed against current inputs and passed with exact pre-edit inputs (model-golden.jsonl, lower-golden-red.jsonl and baseline-{model,lower}-golden.jsonl).

Measured free-text author sites removed in the two activity sources: 14 protobuf message literals (including three Testpilot CorrelatedEvidence observations), 8 method-suffix literals, 13 assignment targets, 6 compact evidence-field paths, 9 projected paths, 3 symbolic Proto fields, and 4 EnumName constructor sites. No quoted Temporal/Testpilot protobuf full name remains. The only dynamic message roots are the two explicit `Operand.Projected.as[InstructionOutcome]` RunEvent payload origins; no unknown generated enum value is used.

No files were staged or committed. HEAD/base remain 2093a63f2da7c6c66f3509414dd78476eb4d5c04. Task status remains in_progress for conductor review/completion.

Independent task-local implementation review reached SHIP with no findings; session 01a10220-64ea-7b22-b43c-3f437a69499b. Conductor verified all 78 pinned inputs, current compiler diagnostics, strict artifact comparison and diff check.

stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: wave-dispatch - ran (model: gpt-6-sol at high)
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- scala-cli test --suppress-outdated-dependency-warning model/lifter --test-only 'umpire.lift.Fixtures*' -- '*typed*', CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks, python3 .flow/tmp/fn117-6/compare_artifacts.py, CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/model -run '^TestMigrationGoldens$', CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -tags test_dep -p 1 -parallel 1 ./tools/umpire/lower -run '^TestMigrationGoldens$', CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make lint-model, git diff --check, Independent implementation review SHIP, session 01a10220-64ea-7b22-b43c-3f437a69499b; all 78 pinned hashes verified, Unchanged full Go source coverage reused from fn-113.16; focused model/lower migration goldens rerun against current inputs
- PRs:
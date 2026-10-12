---
satisfies: [R17]
---
# fn-159-qualified-action-names-in-rules.10 Explicit input defaults: start() is every declared default, no first-value rule

## Description

Make omitted inputs read their token's explicit default rather than catalog order, in the
framework and lifter. Apply this to activity_mini and shared sources only. Preserve positional
calls and :=; the later input-record redesign belongs to fn-165. Current mini defaults are
Timeout.unset and MaxAttempts.one, not the real activity's MaxAttempts.unlimited.

**Size:** M
**Files:** action/input and named-call framework, action/token/default collection and all lifter
consumers, selected input declarations, focused native/isolated fixtures, migration and docs.
**Touches:** [model/framework/Action.scala, model/framework/Syntax.scala, model/framework/InputSource.scala, model/framework/*.test.scala, model/irgen/Context.scala, model/irgen/Declarations.scala, model/irgen/Constants.scala, model/irgen/Claims.scala, model/irgen/Syntax.scala, model/irgen/Types.scala, model/irgen/Realizations.scala, model/irgen/test/InputDefaults.test.scala, model/irgen/test/DeadlinePresets.test.scala, model/irgen/test/FixturePositions.test.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/inputDefaults/**, model/irgen/testdata/lifts/Inputs.scala, model/irgen/testdata/lifts/ManyInputs.scala, model/irgen/testdata/rejects/**, model/temporal/features/activity_mini/**/*.scala, model/temporal/features/activity/Activity.scala, model/temporal/foundations/**/*.scala, model/temporal/actors/**/*.scala, model/migrate/**, model/README.md, model/SEMANTICS.md]

### Approach

- Start from the preceding committed bounded-integer checkpoint on umpire. Capture the selected
  pre-edit jar/full lift and inventory every omitted input with its old concrete value, including
  shared tokens and realization deadline presets. Real-feature sources/lifts remain excluded.
- Input[A] carries Option[A] default; input[A](default = v) declares it, input[A] does not.
  Empty and partial named calls fill only declared defaults; omission without one fails at class
  construction with the input's actual name, including machine/Scenario/realization initialization.
  Native token inputs currently carry empty declaration names: address that evidence-backed gap
  with a narrowly scoped name-capture mechanism, not a positional label or catalog/name guess.
  Named string inputs with no token remain required; zero-input actions/timers still work.
- Preserve full positional and named calls, ordering/typing, token identity, duplicate/foreign-slot
  refusals and arities one through six. Independently test non-first defaults, false/zero defaults,
  two tokens of one type, aliases/shared action bindings and required tokens supplied explicitly.
  A missing default must not fall back to domain order even if its catalog is nonempty.
- Preserve the supported finite value shapes, not just nullary enums: the old Types.firstValue
  recursively constructs record fields and enum cases with fields, including the synthetic
  Option catalog. Prove explicit record/optional/field-bearing defaults with focused controls
  and reuse existing constant-value lowering for their declarations. If literalValue's current
  example-oriented subset lacks an already-supported shape, close that narrow gap; do not add
  an arbitrary expression evaluator or leave a previously liftable empty class unexpressible.
- The lifter reads the actual token declaration's default into internal metadata, without new
  schema/transport fields. Preserve action binding inheritance and token ownership. Update both
  Claims.firstClass/firstInput and Syntax.named, plus Realizations.deadlinesItem: all currently
  fill inputs through firstValue. The latter computes defaults for every input even when a preset
  supplies non-Timeout inputs. Require defaults only for inputs truly omitted or whose default
  the deadline contract needs, preserving explicit required non-Timeout presets and existing
  Timeout-default checks. Keep unrelated firstValue uses if any remain; remove obsolete ones.
- MiniActivity's scheduleToStart/startToClose use Timeout.unset; maxAttempts uses MaxAttempts.one.
  scheduleToStart is also taskQueue.schedule's token. Shared failure currently appears explicitly
  in selected respondFailed calls: retain it as required unless the inventory proves an omission.
  Preserve all eight authored deadline presets, maxAttempts.one/two and seven evidence bindings.
- Isolated lift controls include a genuinely non-first default, empty/partial/full/zero-input
  calls, reused tokens/bound actions and located required-input refusal. Exercise deadline presets
  with an explicitly supplied required non-Timeout input, not merely the all-default mini case.
  Prove native initializer and lifter refusals separately; do not invoke broad Fixtures.beforeAll.
  Update relevant old framework/lift fixtures to declare their previous values, retaining their
  assertions and expected IR. Reuse the existing exact-coordinate fixture contract only if these
  required declaration changes move a source position; preserve its preimage and assert every
  non-coordinate leaf. Do not expand this step into unrelated fixture cleanup.
- Ship a compiled task-named Scalafix rule under model/migrate's existing convention. Use actual
  input/default and omission ownership evidence or a documented explicit migration inventory;
  preserve existing defaults and required inputs, refuse ambiguous/unproven old catalog values
  atomically with a manual recipe. Prove transformation, convergence and refusal on actual copied
  mini declarations, independently compiling/lifting the accepted/restored source. Legacy input
  text may be parsed rather than compiled by the new API; label that boundary honestly.
- README/SEMANTICS describe token defaults and required-input omission, never a catalog-first
  policy. Keep exact mini source-citation assertions accurate. Production loading remains held.

### Verification and boundaries

Run a selected mini/shared compile, focused input/changed-file native tests, isolated input-default
and targeted deadline lifter controls, migration/citation tests and complete raw mini IR/waiver
comparison. Source positions are part of the raw evidence: no blanket stripping, golden rewrites,
padding or blind name substitution. Retain genuine differences explicitly if unavoidable; don't
claim expected-byte equality without proving it. Freeze source/jar inputs after formatting.

Direct FEATURES=activity_mini --check-equivalence --steps-only uses the complete immutable baseline
/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/
with a separate justified task map: sixteen tables, 11,325 rows including shared roots. Both that
capture's and the original ten-table integrity scripts must pass. Neither baseline, original frozen
jar, old batch-1d map, protected payload nor pin changes or is recaptured. Record actual RED/failures.

No real-feature edits/lifts/migration; no Go/proto/schema/Make/CI/runtime/Case/live/production-loader
or regeneration. Lint/manual acceptance NOT RUN. Broad validation and independent review happen
once at the Scala batch end; no per-task SHIP/canonical/consumer/full-set credit.

## Acceptance

- [ ] Empty and partial named calls use only explicit defaults; omission without one refuses with
  the actual input name in native construction/initialization and at the lifter call.
- [ ] Non-first/false/zero defaults and full calls, ordering, ownership, arities and binding reuse
  are independently proven; no framework/lifter input filler reads catalog order.
- [ ] Mini/shared omitted-input values and required inputs follow the captured inventory; deadline
  presets retain explicit required non-Timeout values, all eight variants and evidence bindings.
- [ ] Actual compiled migration transforms copied mini declarations, converges, refuses unsafe
  cases atomically and documents supported/manual/production boundaries; raw accepted/restored
  mini IR/waivers, semantics and protected native tables remain unchanged.
- [ ] Docs/guide and focused gates match current source, both baseline integrity checks pass,
  failures remain recorded and all owner-held broad/consumer/real/lint obligations stay open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

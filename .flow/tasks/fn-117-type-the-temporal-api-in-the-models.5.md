---
satisfies: [R5, R6, R7, R10]
---
# fn-117-type-the-temporal-api-in-the-models.5 Add typed constant messages, enum values and map entries

## Description
Complete the symbolic constant-message author surface, using the same typed field/operand interface.

**Size:** M
**Files:** model/umpire/realize typing module and Realize.scala; model/lifter/Realizations.scala; focused compile/lift fixtures/tests.
**Touches:** [model/umpire/realize/**, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, .flow/tmp/fn117-5/**]

### Approach
- Declare a symbolic message by generated message type and typed selected fields, rather than string Proto/ProtoField/ProtoEntry names. Support all existing nested/oneof/repeated/default and bytes/UTF-8 shapes without constructing a real message. Reuse descriptors for field names and enum names; no handwritten emitter table.
- Accept generated enum values with their full type. Unknown enum cases and values for the wrong enum must fail compilation; do not concatenate enum strings. Preserve exact emitted enum names/default absence.
- Type map key/value construction for Payload.metadata (string to bytes), including the existing encoding -> Utf8(json/plain) entry. Map keys are application data and remain values; they are not proto-name string violations. Do not introduce map-key read paths.
- Support dynamic helper parameters with typed field/enum/operand values, so task6/7 can replace status(value: String) and response(variant: ProtoField) without a string escape hatch or altered IR.
- Add compile-negative wrong message field, wrong scalar/enum/map key/value type and unknown enum fixtures, plus typed nested/oneof/repeated/map constant messages. Compare against existing lifted fixture meaning and refusal positions.

### Investigation targets
**Required:**
- model/umpire/realize/Realize.scala:326-349
- model/lifter/Realizations.scala:150-269
- model/temporal/nexuscaller/Realization.scala:239-323
- model/temporal/standaloneactivity/Realization.scala
- model/lifter/testdata/lifts/Realizations.scala:121-124
- tools/umpire/lower/descriptor.go:182-194
- model/lifter/test/Fixtures.test.scala

### Quick commands
Focused typed message/enum/map negative and lift fixtures; full lifter, check-mode model gate and lint-model once ready. Reuse unchanged Go baseline.

### Execution constraints
No real Scala message construction, Go lowering/schema changes, new path grammar, Model ID helpers or API behavior hints. Preserve comments and verification scopes; user owns commits.

## Acceptance
- [ ] Symbolic messages and enum values are typed, including nested/repeated/oneof/default/bytes and the current string-to-bytes map; mismatched fields/values and unknown enum cases fail compilation.
- [ ] Generic lowering emits exactly the existing message/field/enum names and data values modulo source positions.
- [ ] Typed helper parameters work without a free-text escape hatch, and positive/negative fixtures and scoped gates pass.

## Done summary
Typed symbolic `Proto[Message]`, field/value/entry carriers and generic descriptor lowering now cover the existing constant forms. Positive fixtures compare nested Failure and Duration messages (including a bound `2L` seconds helper), enum Command, oneof StartOperationResponse, omitted fields, vector-spliced field declarations, and Payload string-to-bytes metadata/UTF-8 with the legacy IR after only realization IDs and positions are removed. Base-typed generated enum and typed-field helpers both lower through bound parameters. Thirteen compile-negative examples reject wrong message roots, scalar/enum/nested/map key/map value types, unknown enum names, direct and field-context `Unrecognized` values, unknown typed-operand enums, missing fields, and direct carrier construction. Broad generated-enum helper parameters remain accepted; if supplied an `Unrecognized` value, the lifter refuses it before writing IR. The existing string constructors remain as the task-8 migration bridge; no Temporal message is constructed or sent.

Review round 1 fixes: `ProtoValue.number(Long)` now folds Long literals and bound Long helper values in the lifter. Both `ProtoValue.enumValue` and `Operand.enumValue` use separate declared and actual type parameters with `NotGiven[Actual <:< UnrecognizedEnum]`; this rejects statically unknown generated values while preserving helpers typed with the exact generated enum base type. Both lowering paths validate named values against the generated enum descriptor. The IR/Go format cannot encode values for a repeated protobuf field, but the existing repeated symbolic field-declaration list (`fields*`) works and is covered by a helper fixture; no new repeated-value IR form was introduced.

baseline: green via task-4 handoff at the same HEAD (review-fixtures-3.log 21/21; review-check-model.log; review-lint-model.log; review-format-check.log; review-artifacts.log). No Go inputs changed; prior fn-113.16 Go gate evidence remains applicable. Initial positive fixture failed for absent typed APIs (red-positive.log), and reviewer-regression controls showed a prior `2L` lift refusal and a missing contextual unknown-enum rejection (review-red-fixtures.log). Final focused Fixtures suite passed 23/23 (review-fixtures-2.log, exit 0), including the new dynamic-helper refusal. Model regeneration passed (review-gen-model.log, exit 0). Final check-mode gate passed, including lifter and IR/Case comparison (review-check-model-final.log, exit 0). Final lint-model and configured format check passed with zero current compiler errors (review-lint-model-final.log, exit 0); its caught JDK 27 Scalafix NoSuchFieldException trace is inherited. Targeted formatting passed (review-format.log and review-format-2.log, exit 0). All 30 pre-existing artifact hashes passed (review-artifacts-final.log); git diff --check passed. Every pre-existing comment line in touched source files remains present. Exact touched paths are in touched-files.json; pre-edit source mirrors are under preedit/ and pre-edit hashes in preedit.sha256. No source was staged or committed.

Independent implementation re-review reached SHIP in the same session 01a101da-8d29-7943-afb5-1fe4690d8f1f after both findings were fixed. Conductor verified all 63 pinned inputs, current lint diagnostics, unchanged artifacts and diff check.

stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: wave-dispatch - ran (model: gpt-6-sol at high)
## Evidence
- Commits:
- Tests: mise exec -- scala-cli test model/lifter --test-only umpire.lift.Fixtures (exit 0; 23/23; .flow/tmp/fn117-5/review-fixtures-2.log), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; .flow/tmp/fn117-5/review-gen-model.log), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; .flow/tmp/fn117-5/review-check-model-final.log), mise exec -- make lint-model (exit 0; zero current compiler errors; .flow/tmp/fn117-5/review-lint-model-final.log), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf <task Scala sources> (exit 0; .flow/tmp/fn117-5/review-format.log and review-format-2.log), shasum -a 256 -c .flow/tmp/fn117-3/artifacts-before.sha256 (exit 0; 30/30; .flow/tmp/fn117-5/review-artifacts-final.log), git diff --check (exit 0), Independent implementation re-review: SHIP, same session 01a101da-8d29-7943-afb5-1fe4690d8f1f; conductor verified all63 pinned inputs, Unchanged Go source inputs reuse fn-113.16 test/vet/lint evidence under MILESTONES.md
- PRs:
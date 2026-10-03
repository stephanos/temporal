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
Typed symbolic `Proto[Message]`, field/value/entry carriers and generic descriptor lowering now cover the existing constant forms. Positive fixtures compare nested Failure and Duration messages (including a bound `2L` seconds helper), enum Command, oneof StartOperationResponse, omitted fields, vector-spliced field declarations, and Payload string-to-bytes metadata/UTF-8 with legacy IR after only realization IDs and positions are removed. Typed enum and field helpers lower through bound parameters. Thirteen compile-negative examples reject wrong message roots, scalar/enum/nested/map key/map value types, unknown enum names, direct and field-context `Unrecognized` values, unknown typed-operand enums, missing fields, and direct carrier construction. Existing string constructors remain as the task-8 migration bridge; no Temporal message is constructed or sent.

Review fixes fold Long literals and bound Long helper values, reject statically unknown generated enum values while preserving helpers typed with the generated enum base, and validate named values against generated enum descriptors. The IR/Go format cannot encode values for a repeated protobuf field; existing repeated symbolic field declarations work and are covered. Final focused Fixtures passed 23/23, model regeneration and check-mode gate passed, model lint and formatting passed with zero current compiler errors, all 30 pre-existing artifact hashes passed, and `git diff --check` passed. Independent re-review reached SHIP; the conductor verified all 63 pinned inputs and unchanged artifacts.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: mise exec -- scala-cli test model/lifter --test-only umpire.lift.Fixtures (exit 0; 23/23), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), mise exec -- make lint-model (exit 0), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf <task Scala sources> (exit 0), shasum -a 256 -c .flow/tmp/fn117-3/artifacts-before.sha256 (30/30, exit 0), git diff --check (exit 0), Independent implementation re-review SHIP; conductor verified 63 pinned inputs and unchanged artifacts
- PRs:
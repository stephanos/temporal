# Migrate elapsed-time fields to protobuf Duration

## Current admission order (owner override, 2026-10-16)

The framework DSL is priority 0. Native implementation and fn-188.3's formal batch gate are complete (SHIP). Next is .5: close only native Duration value/presence coverage and author guidance against the final public surface. Construction/retention is not owning-field admission; no new native validation policy is implied. Existing passing native evidence is retained.

Tasks .1-.4 remain blocked integration work, not prerequisites of .5. Their schema/Go/lifting/runtime/equivalence/artifact obligations remain intact. The former whole-spec fn-145/fn-140/fn-123/fn-149 scheduling edges are removed only to admit .5; their original integration baselines below still bind .1-.4. Completion of .5 does not release them. No Temporal source, including disabled bodies, may be read or consumed. No new Scalafix rule, verifier or manifest is requested; earlier mini migration wording below is historical and superseded for current execution.

> HTML render lens: .flow/artifacts/fn-147-migrate-elapsed-time-fields-to-protobuf/spec.html — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

Model authors, lowering and Testpilot currently exchange elapsed-time quantities as millisecond integers with field-specific presence rules. Move those quantities to `google.protobuf.Duration` across the Umpire and Testpilot boundaries, preserve their defaults and monotonic meaning, and simplify polling and scalar-presence wrappers in the same successor-format migration.

## Architecture & Data Models
<!-- scope: technical -->

Seven Testpilot quantities migrate together. They cover instruction timeout, evidence polling interval, wait-hint bound, two Program duration ceilings, Contract elapsed deadline and Run Event elapsed coordinate. Umpire realization hints and operands that lower into those fields use the same checked conversion contract.

Case format 4.0 includes Duration and scalar-presence changes under fn-146.1's shared batch contract. Replace the old fields directly and migrate authoring, lowering and runtime consumers together; public emission/admission activates only at fn-148.6, and managed artifacts and recorded companions regenerate once at fn-148.7. Retired formats 1.0, 2.0 and 3.0 are unsupported; no parallel field spellings, decoders or compatibility conversion paths are retained. The schema/producer and consumer tasks form one breaking integration batch: the intermediate tree may be red, and only schema generation plus generated-API checks gate the schema stage. Consumer-dependent tests wait until consumers compile; managed-artifact tests and full gates wait for fn-148.7.

Counts, attempts, ordinals, percentages, logical transition bounds and wall-clock timestamps remain their existing types. An absent Duration, an explicit zero and a positive value keep field-specific meanings rather than sharing one global default.

## Edge Cases & Constraints
<!-- scope: technical -->

- Existing consumers operate at whole-millisecond precision. Finer precision is rejected rather than truncated.
- Signed protobuf Duration values require field-specific nonnegative or positive admission.
- Conversion checks seconds and nanos before integer conversion so overflow cannot wrap.
- Run Event elapsed time remains monotonic duration since Run opening, not an absolute timestamp.
- Polling simplification must distinguish one read from repeated polling without losing omission semantics.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The seven elapsed-time fields use `google.protobuf.Duration` in the successor Testpilot format, and corresponding Umpire realization fields lower through one checked conversion contract. Counts, ordinals, percentages, logical bounds and timestamps do not change. Errors: negative values where forbidden, non-positive polling intervals, sub-millisecond precision and conversion overflow are rejected at admission with the owning field named.
- **R2:** Each field preserves its absent, explicit-zero and default behavior, including instruction defaults, Program ceilings, Contract deadlines and Run Event monotonic coordinates. Errors: missing required presence, explicit zero where positive is required and a decreasing elapsed coordinate are rejected with their existing error class.
- **R3:** Successor-format evidence reads perform one read when interval is absent and poll when a positive interval is present. This intentionally changes the old omitted-policy refusal. Scalar-only singleton oneofs use proto3 presence where their arm carries no additional domain meaning. Errors: present-zero or negative intervals, polling without an admitted timeout or default, and retired field spellings are rejected.
- **R4:** Scala authoring, lifting, Go lowering, Testpilot preparation, execution, verification, conformance and recorded artifacts use the same checked conversions. Errors: a source value that cannot be represented exactly fails before artifact generation or Driver I/O; retired formats are rejected rather than converted.
- **R5:** Generated Models, Cases, fixtures and canary bindings are regenerated once with categorized identity changes, and the semantics, module map and milestone overview document the new units and presence rules. Umpire's elapsed-time hint/default fields and the seven Testpilot fields are inventoried together. Errors: any artifact delta outside that inventory, its derived identities and R3's explicit presence cleanup stops the migration.

## Early proof point

Implementation starts after committed fn-146-adopt-cel-for-runtime-predicates-and.6 source completion, not fn-146 whole-spec closure; all three specs close together at fn-148.7.

Task fn-147-migrate-elapsed-time-fields-to-protobuf.1 validates the conversion and presence contract against boundary values before schema consumers move. If it fails, re-evaluate the field-specific defaults and successor representation before Task fn-147-migrate-elapsed-time-fields-to-protobuf.2.

## Boundaries
<!-- scope: business -->

- Timestamps, event counts, attempts, ordinals, percentages and operation-transition bounds do not migrate.
- No implicit rounding or saturation is allowed.
- Evidence declaration unification and correlated state normalization land in the following Testpilot consolidation spec.
- No new duration configuration surface is added.

## Decision Context
<!-- scope: both -->

The seven fields describe one unit family but currently duplicate integer conversion and presence rules. One coordinated breaking migration avoids mixed Cases whose producer and runtime disagree about units. Folding the read policy and scalar presence cleanup into this change removes wrapper APIs that exist only because integer scalars lacked presence.

## Execution baseline

The structural prerequisites are integrated: fn-145 split the Umpire schema; fn-142/fn-143 and `fn-151-split-standalone-activity-into-smaller` established the current framework and Activity source layout. Start this batch only from the integrated, committed closure of fn-156 and authoring Batch 2 (fn-140/fn-123, with their shared close at fn-123.8). Before implementation, re-anchor schema fields, Scala declarations, lifter/lint owners, tests and artifact identities against that closed baseline. fn-123's fault metadata and fn-156's compiler checks are still pending at this preparation point; consume their actual committed APIs and pins, without assuming names or results here. Retain the single format 4.0 activation at fn-148.6 and the shared regeneration, full-gate, review and live-run close at fn-148.7.

## Quick commands

Current native-only task .5 uses:

```bash
mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --test-only framework.realize.RealizationConstructionTest --require-tests
```

Retain already valid native evidence and run only reached controls during iteration. The broad native batch gate runs at batch end if source changes invalidate its evidence.

Historical integration commands below remain held, not instructions for .5:

```bash
go test -tags test_dep ./common/testing/testpilot/... ./tools/umpire/lower/... ./tools/umpire/conformance/...
make umpire-gen-model
make umpire-check-cases
```

## Migration policy

The owner authorized breaking IR changes on 2026-10-06. Preserve supported behavior and domain authority, not historical wire compatibility. Regenerate managed artifacts and recorded companions under the current schema; remove superseded fields and runtime machinery. Format checks reject retired artifacts explicitly.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The seven elapsed-time fields use `google.protobuf.Duration` in the successor Testpilot format, and corresponding Umpire realization fields lower through one checked conversion contract. Counts, ordinals, percentages, logical bounds and timestamps do not change. Errors: negative values where forbidden, non-positive polling intervals, sub-millisecond precision and conversion overflow are rejected at admission with the owning field named. | fn-147-migrate-elapsed-time-fields-to-protobuf.1, fn-147-migrate-elapsed-time-fields-to-protobuf.2, fn-147-migrate-elapsed-time-fields-to-protobuf.3, fn-147-migrate-elapsed-time-fields-to-protobuf.5 | — |
| R2 | Each field preserves its absent, explicit-zero and default behavior, including instruction defaults, Program ceilings, Contract deadlines and Run Event monotonic coordinates. Errors: missing required presence, explicit zero where positive is required and a decreasing elapsed coordinate are rejected with their existing error class. | fn-147-migrate-elapsed-time-fields-to-protobuf.1, fn-147-migrate-elapsed-time-fields-to-protobuf.2, fn-147-migrate-elapsed-time-fields-to-protobuf.3, fn-147-migrate-elapsed-time-fields-to-protobuf.5 | — |
| R3 | Successor-format evidence reads perform one read when interval is absent and poll when a positive interval is present. This intentionally changes the old omitted-policy refusal. Scalar-only singleton oneofs use proto3 presence where their arm carries no additional domain meaning. Errors: present-zero or negative intervals, polling without an admitted timeout or default, and retired field spellings are rejected. | fn-147-migrate-elapsed-time-fields-to-protobuf.3 | — |
| R4 | Scala authoring, lifting, Go lowering, Testpilot preparation, execution, verification, conformance and recorded artifacts use the same checked conversions. Errors: a source value that cannot be represented exactly fails before artifact generation or Driver I/O; retired formats are rejected rather than converted. | fn-147-migrate-elapsed-time-fields-to-protobuf.2, fn-147-migrate-elapsed-time-fields-to-protobuf.3, fn-147-migrate-elapsed-time-fields-to-protobuf.4, fn-147-migrate-elapsed-time-fields-to-protobuf.5 | — |
| R5 | Generated Models, Cases, fixtures and canary bindings are regenerated once with categorized identity changes, and the semantics, module map and milestone overview document the new units and presence rules. Umpire's elapsed-time hint/default fields and the seven Testpilot fields are inventoried together. Errors: any artifact delta outside that inventory, its derived identities and R3's explicit presence cleanup stops the migration. | fn-147-migrate-elapsed-time-fields-to-protobuf.4 | — |

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope.

---
satisfies: [R10]
---
# fn-139-actor-grouped-rules-per-rpc-actions.8 Export shared rejection-to-RPC status mapping

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). This schema/export half runs before the batch's single regeneration. Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task. Framework, lifter, reader and focused unit tests still run here. Commit the task on its own.

Investigation found no existing realization metadata slot that can carry the shared `Rejection` to gRPC status-code mapping, so the former task 8 is split as its own size rule requires. Add one table in the Temporal realization layer, written as an exhaustive Scala match over every `umpire.outcomes.Rejection` case: `notFound` -> `NOT_FOUND`, `alreadyExists` -> `ALREADY_EXISTS`, `failedPrecondition` -> `FAILED_PRECONDITION`, and `invalidArgument` -> `INVALID_ARGUMENT`. Attach that table centrally through `temporalRealization`, export it on each Temporal `Realization` through a new IR field, and admit/validate it in the Go realization reader. There must be no independently maintained Go mapping.

Remove `RecordOverQueue.states.closedAnswer` and any other ad-hoc rejection-answer string. Replace the composition binding with a typed/shared derivation of the composed outcome key; do not inline the old string. Preserve every Query assessment. Add focused Scala/lifter and Go reader tests proving the mapping is exhaustive, exported on each Temporal realization, unique, complete, and correctly read. Declare the new IR metadata and the answer-string removal for the batch diff. The post-regeneration conformance consumer is task 9.

**Files:** `model/temporal/realize/` (shared mapping and central attachment), `model/umpire/realize/Realize.scala`, `model/temporal/features/activity/standalone/system/WithTaskQueue.scala`, any framework helper needed to derive a composed outcome key without hand-written strings, `model/irgen/`, `proto/internal/temporal/server/api/umpire/v1/ir.proto`, and the Go realization reader/tests.

**Touches:** [model/temporal/realize/**, model/umpire/**, model/temporal/features/**, model/irgen/**, proto/internal/temporal/server/api/umpire/v1/**, tools/umpire/realization/**]

Required investigation already established:
- `Realization` currently ends at field 17; allocate a new field without renumbering existing fields.
- `temporalRealization` in `model/temporal/realize/Kit.scala` is the central attachment point for every Temporal realization.
- fn-133.8 derived carriers from existing performance metadata, but rejection codes are not derivable from any existing IR field.
- Run instruction outcomes already retain lower-case protocol codes; task 9 owns correlating them to Model steps and enforcing conformance.

## Acceptance
- [ ] One shared Temporal Scala table exhaustively maps every `Rejection` to its gRPC status code with no wildcard; removing a case's mapping fails compilation.
- [ ] A new realization IR field carries that table from Scala through the lifter. Every Temporal realization receives it through `temporalRealization`; no second Go-side mapping exists.
- [ ] The Go realization reader admits only a unique, complete mapping and a Go test proves every exported shared `Rejection` case has exactly one code.
- [ ] `closedAnswer` and every equivalent hand-written rejection-answer string are gone. Composition capability bindings derive the shared rejected outcome key without inlining its encoded string, and existing Query assessments remain unchanged.
- [ ] Focused Scala framework/lifter tests and Go realization reader tests pass. No checked IR, fixture golden, or Cases regeneration occurs; the schema and metadata delta is declared for the batch regeneration.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

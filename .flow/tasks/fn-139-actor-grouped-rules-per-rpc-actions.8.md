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
Implemented the shared Temporal rejection-to-gRPC-code table end to end: the Scala kit owns and attaches it, generic realization lifting emits it, generated IR carries it, and Go validates it from generated enum metadata. Replaced the hand-written composed rejection string with a typed `Composition.composedOutcome` helper and proved both runtime behavior and the actual `RecordOverQueue.capabilities` lift.

Focused verification:

- `go test -tags test_dep ./tools/umpire/realization -run '^TestRejectionCodesAreReadAndAdmittedAsAUniqueCompleteTable$'` — green.
- `mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only 'temporal.realize.RejectionsTest'` — green.
- `mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only 'temporal.capabilities.CapabilityPropertiesTest'` — green.
- Separate actual-root lifts for activity standalone, Nexus workflow, and Nexus standalone — green; every emitted realization carried the exact four-entry table.
- Actual `temporal.features.activity.standalone.system.RecordOverQueue$.capabilities` lift — green, exercising the typed composed-outcome fold.
- `mise exec -- scala-cli test model/irgen` — all behavioral tests green, with only the five approved regeneration-deferred artifacts stale: `taskqueue.json`, `hints.json`, `rejections.json`, `hintsRefused.json`, and `rejects.txt`.
- Targeted Scalafmt check, `gofmt -d`, and `git diff --check` — green.

Expected batch deltas: the later single regeneration adds `rejectionCodes` to every checked Temporal realization IR and refreshes the five named fixture artifacts (including queued source-position/type deltas from the batch). This task intentionally changed no checked `model/ir`, Cases, expected lifter fixtures, or `MILESTONES.md`.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3ff210b307757bf44d9ab0cc7ab3b883e14f489d
- Tests: go test -tags test_dep ./tools/umpire/realization -run '^TestRejectionCodesAreReadAndAdmittedAsAUniqueCompleteTable$', mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only 'temporal.realize.RejectionsTest', mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only 'temporal.capabilities.CapabilityPropertiesTest', three separate actual-root lifter invocations: activity standalone; Nexus workflow; Nexus standalone, actual capability-root lift: temporal.features.activity.standalone.system.RecordOverQueue$.capabilities, mise exec -- scala-cli test model/irgen (expected nonzero only for regeneration-deferred taskqueue.json, hints.json, rejections.json, hintsRefused.json, rejects.txt), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check <10 changed Scala files>, gofmt -d tools/umpire/realization/rejection_codes.go tools/umpire/realization/rejection_codes_test.go tools/umpire/realization/validate_realization.go, git diff --check
- PRs:
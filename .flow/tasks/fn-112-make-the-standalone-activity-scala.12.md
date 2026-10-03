---
satisfies: [R20]
---
# fn-112-make-the-standalone-activity-scala.12 Extract the reusable Temporal task-queue entity and shared provider claims

## Description
Extract the dispatch/matching queue from standalone activity into reusable `model/temporal/taskqueue`, following the existing `temporal/worker` entity boundary.

**Size:** M
**Touches:** [model/temporal/taskqueue/**, model/temporal/standaloneactivity/System.scala, model/temporal/standaloneactivity/compositions/**, model/temporal/test/**, model/lifter/testdata/**, model/gate/Roots.scala, model/README.md, .plans/UMPIRE_MODULES.md, model/ir/**]

Move QueueView/QueueDetail, custody and delivery domains, enqueue/deliver/acknowledge, matching internal steps, crashes/ack-loss/storage-loss assumptions, opaque/detailed contracts and matching/lossy/forgetful/volatile providers into the shared package. Define the queue Entity keyed by taskQueue and attach queue machines/actions where supported. Preserve the one-message/two-delivery bound; do not add multi-message configurations. Shared queue Properties and provider Queries move with the code, in Model/Properties/Queries files. Parameterize no behavior merely to anticipate future consumers. Keep the feature's dispatch/admission/settle synchronization and cross-entity claims in standaloneactivity. Preserve old symbol IDs with DefinitionScope, and retain existing machine/family/query names through explicit declaration overrides where needed. Update gate registration without duplicating model inventories. Use an independent minimal consumer fixture to prove reuse without an activity import. Compare every output against task 1's exact authorized entity delta and original behavior; never accept arbitrary fingerprint drift.

## Acceptance
- [ ] Shared temporal/taskqueue owns the queue Entity, vocabulary, opaque/storage-loss contracts, four providers and their shared properties/Queries; it imports no standaloneactivity types or helpers.
- [ ] Standalone activity consumes the shared model by typed composition and retains only its own admission/synchronization/cross-entity claims.
- [ ] Forgetful/volatile providers remain reusable negative controls with the same counterexamples and ordinary/lossy refinement targets.
- [ ] An independent consumer fixture compiles/lifts/checks the shared queue; no copied transitions/properties or empty files remain.
- [ ] Definition IDs, finite state/class catalogs, transitions, refinements, Query answers and Case bytes match the original baseline. Entity-sensitive fingerprints equal baseline plus precisely the new queue entity metadata.
- [ ] Focused Scala/Go gate checks pass, docs state ownership and finite bounds, and standalone-only plus combined source metrics are recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

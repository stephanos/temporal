---
satisfies: [R5]
---
# fn-132-group-the-nexus-and-activity-models-by.4 Spike: one action shared by two forms' entities, and their outcomes

## Description
**Size:** M
**Touches:** [model/umpire/**, model/temporal/**, model/irgen/**, tools/umpire/ir/**, tools/umpire/interp/**, tools/umpire/check/**, tools/umpire/realization/**, tools/umpire/lower/**, MILESTONES.md]

**Required investigation:** `model/umpire/Domain.scala`, `model/umpire/Action.scala`, `model/umpire/Refine.scala`, `model/umpire/realize/Realize.scala`, `model/irgen/test/Fixtures.test.scala`; parent Edge Cases and R2/R3/R5. The stamp prototype is historical inspiration, not an available dependency.
Use disposable fixtures before editing production Models. Preserve admitted identity, Case/verdict behavior and entity typing; no extra configurable identity framework or library. A necessary minimal seam is within the owner's autonomous milestone direction, but must be measured, planned and tested before task 5 uses it.

Settle the two questions that block Parts B and C, in lifter fixtures before touching the Models.

1. **Entity.** The handler's `reply`/`complete` and the worker's `poll`/`respond` are bound with `.on(entity)`, and fn-126 decision 28 infers a machine's entity from its actions. The forms name their entity differently: the workflow form by `scheduledEvent` with `refer = Map("caller" -> workflow)`, the standalone form by `operationId` / `activityId`. Try, in order:
   - (a) **Identity relative to a parent scope**, as the `stamp` prototype keyed its models (`common/testing/stamp/mdl.go`: key `parent/Type[id]`, parent declared by a typed `Scope[*Parent]`). One kind-level entity whose parent is the workflow in the workflow form and the namespace in the standalone form. The parent is declared with a type, not the string-keyed `refer`. This mirrors the server, where `Store chasm.ParentPtr[...]` points to the workflow or is nil.
   - (b) One kind-level entity whose key each form's realization binds.
   - (c) Per-form actions, with the shared product written over the kind's actions and each form's refinement mapping its own.
   Pick the smallest binding proved by the DSL, lifter, Go reader and lowering that meets R2/R3. A types-only result cannot complete this task or satisfy those requirements. If a framework seam is necessary, record its bounded change and implement/test it under the owner's autonomous direction before task 5 starts; do not choose a weaker result merely because it passes existing tests.
2. **Outcome.** The shared product catalog contains both `notFound` and `alreadyCompleted`; prove that `visibleOutcomes` affects stutter observation only after catalog admission. A negative fixture with a missing source outcome must still fail even when visibility excludes it. Preserve current rejection/repeat results.
3. **`terminated`.** Whether `NexusProduct` gains a `terminated` phase or the standalone refinement hides it.
4. **Facts and carriers.** Prove both forms' asynchronous starts and terminal completions have a product carrier. The current checker maps facts by member name, not meaning; use the parent's finite standalone-status → product-fact ledger rather than inventing a broad projection API. Classify scheduled/cancel-request/terminated facts and visible outcomes explicitly. Cover starts, sync/async completions, controls/repeats, termination and stutters, with a wrong/missing fact carrier negative. Keep history and Describe read contracts unchanged under those declared identities.
5. **Product signature closure.** Identify the exact existing network/timeout declarations and input types the shared product uses; move that closure in task 5 so kind code does not import either form. Distinguish disabled extra input classes and canonical fingerprint changes from previously enabled transition behavior.

Record each decision, with what it changes, in the spec's Decision Context and close its Parked unknown.
## Acceptance
- [ ] Passing lifter and checker-level fixtures prove one kind action bound for both forms, required fact carriers and both outcome catalog members, while missing/wrong carriers or catalog members fail even under visibility exclusion.
- [ ] The entity, outcome and `terminated` decisions are in the spec's Decision Context; their Parked unknowns are removed.
- [ ] Any required binding seam is recorded, implemented and verified before task 5 starts; its positive and wrong-entity/invalid-mapping cases preserve the exact requirement and artifact contracts. No unproved or types-only fallback is reported as success.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

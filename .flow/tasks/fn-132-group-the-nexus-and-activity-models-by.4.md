---
satisfies: [R5]
---
# fn-132-group-the-nexus-and-activity-models-by.4 Spike: one action shared by two forms' entities, and their outcomes

## Description
**Size:** M
**Touches:** [model/umpire/**, model/temporal/**, model/irgen/**, tools/umpire/ir/**, tools/umpire/interp/**, tools/umpire/check/**, tools/umpire/realization/**, tools/umpire/lower/**, MILESTONES.md, .flow/specs/fn-132-group-the-nexus-and-activity-models-by.md, .flow/specs/fn-132-group-the-nexus-and-activity-models-by.json]

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

**Decision-record scope:** The parent authoring paths above are writable only to record this spike's proved entity, outcome, termination, fact-carrier and Product-signature-closure decisions and remove their genuinely resolved Parked unknowns. Preserve R1-R5, task dependencies/statuses and unrelated deferred workflow questions. This aligns the write surface with the existing decision-record acceptance; it does not widen the behavior change.
## Acceptance
- [ ] Passing lifter and checker-level fixtures prove one kind action bound for both forms, required fact carriers and both outcome catalog members, while missing/wrong carriers or catalog members fail even under visibility exclusion.
- [ ] The entity, outcome and `terminated` decisions are in the spec's Decision Context; their Parked unknowns are removed.
- [ ] Any required binding seam is recorded, implemented and verified before task 5 starts; its positive and wrong-entity/invalid-mapping cases preserve the exact requirement and artifact contracts. No unproved or types-only fallback is reported as success.

## Done summary
Model authors can bind one kind-declared action to each form's Entity using existing `.on`/`.creates` aliases. Task 4 proves both actual Nexus forms and the Activity extraction shape, records all five decisions in the parent spec, and leaves production PartB/C moves to tasks 5/6.

Tier: session (jev-unavailable(no_key))

The current validation packet is `.flow/tmp/fn132-4/verification-ledger.json`. Canonical unfiltered GEN passed with the full fixture phase; complete affected checker/lower packages passed 649 events with zero failures/skips; current Case check, full Scala lint and read-only Go lint passed. Each gate's logs, numeric exit and independent wrapper wall are retained. Lock-queued walls include wait time. Applicable task2/task3 unaffected evidence was reused under MILESTONES; no full CHECK/whole-Go receipt or live/backend result is claimed.

Source-derived fixtures and the minimal maintained binding/relay fixtures prove canonical action identity/metadata, local creates/control inference, history scheduled-event versus standalone operation-ID correlation and wrong-key refusals. Checker proofs cover shared Property answers on both forms, admitted outcome catalogs before visibility, actual visible terminal/start facts, missing/wrong carriers, invalid mapping, disabled unsupported classes, and exact previous enabled source rows. Lowering preserves all 17 executable programs under the explicit source/fact ledger; fresh projection fingerprints and standalone's mapped Product fields are deliberate augmentation. `tools/umpire/check/testdata/grouping.md` records provenance and these bounds.

The complete 63-artifact proof in `user-position-proof.json` permits only the predeclared exact Activity Product declaration/span Position ledger. The user's Product comment/declaration relocation and its seven generated mirrors remain unstaged and uncommitted. All 1040 changed Positions map exactly; all other content, Case programs/fingerprints, fixture/canary pins and 34 other production sources remain unchanged. See `user-position-ledger.md`, `production-source-proof.txt` and the immutable `artifacts-before/` copies. Checkout release preserves these eight user-owned dirty paths.

Disposable actual-tree sources live in `source-snapshot/`, with 18-file inventory `grouping-source-inventory.sha256`; the three new checked-in IR fixtures equal actual proof5 emitted bytes. Only task-created duplicate source files were deleted after byte comparison, and the retained copies recover them. The original outer-cwd/wrong-surface lift and early fixture construction failures are INCONCLUSIVE, retained separately. Intended alias refusal and lazy-cycle StackOverflow red-to-green diagnostics are recorded. First canonical GEN failed on fixture naming shadowing; its repaired full rerun passed. Failed full old-Contract equality was removed after catalog/refinement augmentation refuted it, with logs retained.

The inherited default-cluster ShutdownWorker race remains red/inconclusive under the MILESTONES owner exception. This offline gate claims no live/backend success. Task 4 ends here; tasks 5–7 remain todo and the parent remains open. Follow-up authoring syntax, Phase roles, enum/comment conventions are outside this task.

stage: impl-review - ran [2026-10-06T15:26:52Z..2026-10-06T15:31:05Z], SHIP. Three fresh gpt-6.1-sol/high draws, zero findings. Receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-132-group-the-nexus-and-activity-models-by.4.json`; full reports `.flow/review-fanout/7cc7dba6b47448a29b3f310ad78a52fb/`. Reviewer reruns were read-only-sandbox-blocked and claim no fresh test pass.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d30f5d031b0966ac4cf23b65c4f768c28b2e33a6, 7b16a388b68c8fa36971c90e6f45ee1f9448c6bf, e073f1af33d479d1ad52aa626330fb87db9bca7e, 6dfc82fe77f7a69f28cbf6a42db7e6ce2c3d27c1, a58dd377f75183b17ba8094a0ced8cf6b87d71d6
- Tests: make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model, mise exec -- go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/check ./tools/umpire/lower, make lint-model, make umpire-check-cases, GOLANGCI_LINT_BASE_REV=21b9964965c8f6e383ee0a751a5fc3cb32d172db GOLANGCI_LINT_FIX=false make lint-code-fast, python3 .flow/tmp/fn132-4/verify-user-positions.py
- PRs:
---
satisfies: [R20]
---
# fn-112-make-the-standalone-activity-scala.12 Extract the reusable Temporal task-queue entity and shared provider claims

## Description
Extract the dispatch/matching queue from standalone activity into reusable `model/temporal/taskqueue`, following the existing `temporal/worker` entity boundary.

**Size:** M
**Touches:** [model/temporal/taskqueue/**, model/temporal/standaloneactivity/System.scala, model/temporal/standaloneactivity/compositions/**, model/temporal/test/**, model/lifter/testdata/**, model/gate/Roots.scala, model/README.md, .plans/UMPIRE_MODULES.md, model/ir/**]

Move QueueView/QueueDetail, custody and delivery domains, enqueue/deliver/acknowledge, matching internal steps, crashes/ack-loss/storage-loss assumptions, opaque/detailed contracts and matching/lossy/forgetful/volatile providers into the shared package. Define the queue Entity keyed by taskQueue and attach queue machines/actions where supported. Preserve the one-message/two-delivery bound; do not add multi-message configurations. Shared queue Properties and provider Queries move with the code, in Model/Properties/Queries files. Parameterize no behavior merely to anticipate future consumers. Keep the feature's dispatch/admission/settle synchronization and cross-entity claims in standaloneactivity. Preserve old symbol IDs with DefinitionScope, and retain existing machine/family/query names through explicit declaration overrides where needed. Update gate registration without duplicating model inventories. Use an independent minimal consumer fixture to prove reuse without an activity import. Compare every output against task 1's exact authorized entity delta and original behavior; never accept arbitrary fingerprint drift.

Declare the queue's shared properties through one `def queueLaws(m)` taking the contract machine (the shape `providerQueries(m)` at `System.scala:757` already has, and the shape `fn-122-capabilities-and-their-laws`'s catalog later generalizes; spec R20, `.plans/SEMANTIC_PROTOCOLS.md` item 6): `delivers` and `committedStays` are declared there once and read on every provider, with `committedStays` written as `stays(_.custody != Custody.nowhere).unless(_.records(QueueFact.acknowledged))` (task 4's pattern) and each instance keeping its frozen name and `*.any.*` Query spelling through the explicit-name form, with each instance's total passed as task 11's integer argument. `storageLossDrops` is not in `queueLaws`: only `lossyMatchingQueue` binds `storageLoss` (`System.scala:797`), a Property on the other providers would be new behavior the Boundaries forbid, and `fold` cannot lift a conditional declaration; it stays declared once on the lossy provider.
## Acceptance
- [ ] Shared temporal/taskqueue owns the queue Entity, vocabulary, opaque/storage-loss contracts, four providers and their shared properties/Queries; it imports no standaloneactivity types or helpers.
- [ ] Standalone activity consumes the shared model by typed composition and retains only its own admission/synchronization/cross-entity claims.
- [ ] Forgetful/volatile providers remain reusable negative controls with the same counterexamples and ordinary/lossy refinement targets.
- [ ] `queueLaws(m)` declares `delivers` and `committedStays` once over the contract machine, `committedStays` with `stays/unless`, each instance with its own total argument and the original Property rows and answers on every provider; `storageLossDrops` stays on the lossy provider alone and no provider gains a Property.
- [ ] An independent consumer fixture compiles/lifts/checks the shared queue; no copied transitions/properties or empty files remain.
- [ ] Definition IDs, finite state/class catalogs, transitions, refinements, Query answers and Case bytes match the original baseline. Entity-sensitive fingerprints equal baseline plus precisely the new queue entity metadata.
- [ ] Focused Scala/Go gate checks pass, docs state ownership and finite bounds, and standalone-only plus combined source metrics are recorded.
## Done summary
Extracted the standalone activity's dispatch/matching queue into the reusable `temporal.taskqueue` (model/temporal/taskqueue: Model, Properties, Queries). Commits 0f83fa247f, 7366a57e81.

**What moved / what stays**
- taskqueue owns: `taskQueueEntity` = Entity("taskQueue", key = "taskQueue"); QueueView/Outstanding, QueueDetail/Custody/Delivered, QueueOutcome, QueueFact; enqueue/deliver/acknowledge/addActivityTask/persistTask/syncMatch (`internal on taskQueueEntity`) and the faults crash/ackLoss/storageLoss (party `fault`, no entity, like the worker's faults); dispatchQueue, dispatchQueueUnderStorageLoss, matchingQueue, lossyMatchingQueue, forgetfulQueue, volatileQueue (all `entity: taskQueue`); Limits `seven`, `twelve`.
- Properties.scala: `queueLaws(m): QueueLaws(delivers, committedStays)`, declared once over a provider, `committedStays` as `stays(_.custody != Custody.nowhere).unless(_.records(QueueFact.acknowledged))`; `storageLossDrops` on the lossy provider alone.
- Queries.scala: `providerQueries(m, anyTotal)` (reads `laws.delivers`/`laws.committedStays`; totals 2880/2880/2880/3240), the storage-loss Query.
- System.scala keeps the admission designs, both composition families (`import taskqueue.*`), the dispatch/admit/settle syncs and the cross-entity claims; Realization names `taskqueue.ackLoss`. Gate roots name the provider Queries under `temporal.taskqueue.Queries$package$` (same order). Imports nothing of standaloneactivity.

**Lifter (with lift + refusal fixtures)**
- Type names follow a file pin: a top-level type of a file whose declarations pin a former file owner `pkg.File$package$` takes `pkg.<Type>`; two types one lift would name alike are refused (MovedRejects.scala/`movedNameTaken`). DefinitionScope probe now also asserts moved types keep their names.
- Claim bundles: a case class whose every field is a Property/Scenario/Query folds to its claims and is read by field (Totals.scala bundled-vs-direct IR equality); a mixed case class is refused (`mixedBundle`), as is a field read naming no claim. Fixed a `None.get` crash for a non-bundle case-class constructor in claim position.

**Identity**: family "temporal.activity.standalone.system", Definition IDs (one DefinitionScope pin of `temporal.standaloneactivity.System$package$`) and IR type names (`temporal.standaloneactivity.Queue*`) unchanged. Only delta: the 12 entity attachments in original.json. Original baseline passes; Case bytes and activity.json unchanged.

**Decisions (autonomous)**
- Faults name no entity (worker precedent); this also keeps ackLoss, which admissionResponseLoss shares, out of the activity race's Cases.
- `queueLaws(m)` returns a case-class bundle (needed a small lifter rule) so Properties.scala and Queries.scala split cleanly; the free `any.committedStays` total stays a `providerQueries` argument.
- Type-name pin instead of a type-rename allowance: R1 allows no type-name delta, and fingerprints must stay exact.
- fn-115's migration golden (closed inventory, by-file positions) needed: `later_inventory` (the new consumer fixture, must exist, not compared), `source_path_splits` (taskqueue files compare as System.scala), `source_root_moves` (moved roots in Model.source), and the original.json attachments applied to its frozen inputs (`golden.Attached`, `Config.Unsplit`), with a unit test.
- Compositions stay in System.scala; task 8 owns the folder layout.

**Consumer fixture**: lifts/TaskQueue.scala (job over dispatchQueue, over matchingQueue replacing it, and over forgetfulQueue as the control); tools/umpire/model/taskqueue_test.go pins every answer (found/verified; forgetful RefinementRejected with its enqueue, addActivityTask, crash witness).

**Metrics**: standaloneactivity 2628 lines / 200 literals before; after 2281 / 179, taskqueue 418 / 25, combined 2699 / 204 (+71 lines: headers, givens, docs).

**Review**: claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`; writer and reviewer are the same family (Opus). Round 1 SHIP. P3s: unknown bundle field now refused at its line (7366a57e81, lifter tests re-run); `golden.Attached` re-decodes original.json per Match (cheap, left).

**For task 8**: type pins cover only top-level types of a file pinning a former `$package$` owner; enums moved into vocabulary objects would change IR type names unless the rule is extended. When System.scala is split, extend `source_path_splits` (and `source_root_moves` for moved roots) in golden/config.json.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0f83fa247f, 7366a57e81
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; lifter tests inside), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|MigrationProjection|IRInventory' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), scala-cli test model/lifter (exit 0, after 7366a57e81's change)
- PRs:
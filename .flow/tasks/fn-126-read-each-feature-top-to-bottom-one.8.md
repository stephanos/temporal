---
satisfies: [R9, R10, R11, R18, R19, R20]
---
# fn-126-read-each-feature-top-to-bottom-one.8 Rename batch, level-name lint, Product and System docs; close

## Description
Also decision 29: add the kit's `trait Client extends Actor` and rename the standalone activity's and standalone Nexus operation's `caller` to `object client extends Client` (the Nexus caller feature keeps `caller`/`handler`).

Also: rename the Nexus caller's `ForgedCompletion` object and its machine to `TrustingCaller`/`trustingCaller` (decision 17, moved here from task 5).

Also decision 28's task-8 items: Scenario names from their `val`, and the intent note in each feature header and `AGENTS.md`.

Task 6c of the split closing batch: the R18 rename batch with decision 19 and the task-queue rows, R20 (b) level names, R10 retired names, R19 and R9 docs, R11 evidence, and close the spec. Carry-forward constraints (`.flow/tmp/fn-126/carry-forward.md`): `callerSide.close`/`handlerSide.finish` (a recorded deviation), `reply` into `Inputs.reply` in both Nexus features, `result` stays top-level (fix the sketch). Prove with a name map applied to the before-IR and `projtool`. Scope the R10 retired-name check to `model/`, `tools/{umpire,canary}`, `tests/testcore/testpilot`, `common/testing/testpilot` and the R9 docs, word-bounded. Plan sections 3-4.

Plan: `.flow/tmp/fn-126/plan6.md` (read-only planning of 2026-10-05, built after task 4 against task 5 in progress; re-verify against the merged task 5). Host decisions on its open points: IDs follow decision 23's rule literally (section objects included, e.g. `…ActivityRecord.monitors.atMostOneActive`); the Kit's family comes from the lifter substituting the realization's package when it folds a Kit call (no macro); level files keep their subject's own types and signature; `TrustingCaller`'s `inspect` moves into the feature's `object caller` (party unchanged), ending the shadowing; a `shared/` folder has at most one `object exports`. The golden harness is retired by fn-124.7 before these tasks: every proof is by projection (`projtool` + a before/after IR projection), never a golden re-capture.

## Acceptance
- [ ] Every item of this task's description is done, with the plan's verification list for its section passed.
- [ ] Equality proved by projection as described; any table, answer, verdict or fingerprint difference beyond the stated renames stops the task.
- [ ] All gates of the spec's Verification pass.


## Done summary
# fn-126.8 done summary

The closing rename batch is applied across Scala, IR, Cases, generated fixtures, canary data, Go expectations, and the R9 documentation. Product/System structure is now enforced independently (including the shared prefix and required System refinement), retired vocabulary is checked across the full live `model/` tree with explicit runtime-kind exceptions, and the projection remains byte-identical after applying the declared name map.

Tier: session (jev-unavailable(no_key))

baseline: red (`make umpire-check-model` failed pre-edit on stale generated protobuf, then inherited Scala unused implicits; that tooling state was repaired before feature edits). The canonical combined command's final model and Scala-lint portions pass; its default `lint-code-fast` leg remains inherited-red because it compares this long-lived branch to `main` and reports 708 unrelated findings. The read-only task-base form reports 0 issues.

stage: impl-review - ran [round 1 NEEDS_WORK..round 2 SHIP] (receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-126-read-each-feature-top-to-bottom-one.8.json`)

### What changed

- R18/R19: `ActivityProtocol`/`NexusProtocol` became Product/System vocabulary; history designs became ActivityRecord/TrustingActivityRecord/HeldDispatch/LostStartAnswer and RecordOver*/TrustingRecordOver*; task-queue levels became TaskQueueProduct/TaskQueueSystem; the listed actions were renamed everywhere. `ForgedCompletion`/machine `forgedCompletion` became `TrustingCaller`/`trustingCaller`, while the Query `forgedCompletion` intentionally keeps its name.
- Decision 29: the kit has `trait Client extends Actor`; standalone activity and Nexus operation use `client`, while Nexus caller keeps its Nexus `caller` role. `CauseKind.handlerReply` remains the retained runtime cause kind.
- Decision 28: Scenario names derive from their vals, including cross-file initialization support; every feature header and `AGENTS.md` carries the independent-model/human-review intent.
- R20(b): Product and System machines are resolved and validated independently, both suffixes are required, prefixes must match, and System must refine Product whenever both level files exist. The `boiler` both-wrong and `urn` missing-refinement fixtures pin the review regressions.
- R10: the retired-vocabulary gate scans all live `model/` content (excluding build/cache output), plus the named Go/Testpilot/docs roots, with explicit retained-runtime exceptions. Stale `IrFile.scala` and protocol prose were corrected.
- R9: README, SEMANTICS, Umpire plans, and module docs describe Product as what the client reads, System as how the server provides it, and history records rather than system contracts.

### R11 measurements

Before task 1 (`.flow/tmp/fn-126/baseline-counts.txt`):

| Folder | Files | Lines | Same-package cross-file refs | disabled | inverted |
| --- | ---: | ---: | ---: | ---: | ---: |
| standaloneactivity | 6 | 1060 | 81 | 22 | 8 |
| standaloneactivity/admission | 4 | 425 | 23 | 11 | 6 |
| standaloneactivity/compositions | 4 | 267 | 15 | 0 | 0 |
| nexuscaller | 5 | 1337 | 65 | 13 | 7 |
| nexuscaller/closepolicy | 4 | 1095 | 68 | 8 | 8 |
| nexusoperation | 6 | 335 | 25 | 5 | 5 |
| shared/taskqueue | 3 | 412 | 24 | 12 | 8 |
| shared/worker | 1 | 85 | 0 | 3 | 3 |

After the closing batch (`.flow/tmp/fn-126/task8/r11-after-counts.txt`):

| Folder | Files | Lines | Same-package refs | Parent-package refs | disabled | inverted |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| standaloneactivity | 2 | 550 | 12 | 0 | 0 | 0 |
| standaloneactivity/product | 1 | 134 | 0 | 9 | 1 | 0 |
| standaloneactivity/system | 3 | 1225 | 6 | 41 | 0 | 0 |
| nexuscaller | 2 | 760 | 11 | 0 | 0 | 0 |
| nexuscaller/product | 1 | 86 | 0 | 9 | 1 | 0 |
| nexuscaller/system | 3 | 1852 | 2 | 47 | 0 | 0 |
| nexusoperation | 2 | 347 | 5 | 0 | 0 | 0 |
| shared/taskqueue | 1 | 131 | 0 | 0 | 0 | 0 |
| shared/taskqueue/product | 1 | 92 | 0 | 9 | 0 | 0 |
| shared/taskqueue/system | 1 | 296 | 0 | 15 | 0 | 0 |
| shared/worker | 1 | 78 | 0 | 0 | 0 | 0 |

Totals moved from 74 disabled arms and 45 inverted guards to 2 declarative disabled actions and 0 inverted guards. The `activityProduct` state-to-Query read is now 3 files/361 lines (`StandaloneActivity.scala` 213 + `product/Product.scala` 134 + `shared/Bounds.scala` 14); `nexusProduct` is 3 files/342 lines (`NexusCaller.scala` 242 + `product/Product.scala` 86 + `Bounds.scala` 14). Same-package references return within `system/` by design (decision 22).

### Complete R5 delta ledger from tasks 1-5

- Task 1: IR/Case source paths and lines; source-root strings; 49 Function symbols for effects, the `admission`→`record` and `compositions`→`withTaskQueue` package moves, and `ends`→`end`; two display-only law binding strings; golden root/package/path move configuration. IDs, types, declaration names, tables, answers, lint findings, and Contracts stayed fixed.
- Task 2: IR/Case positions; 30 source roots; 91 Function moves into module objects/sections, including `Operation.ends`→`end`; one display-only Nexus-operation law binding; five lint `because` prose entries; golden root/function/path merge/split configuration. Two dead function substitutions were removed. The merged fn-124.6 baseline already contained the forgedCompletion conformance reason.
- Task 3: line positions only (including two wait-hint lines); display-only law binding owners moved to actor-qualified `caller` paths; two refusal-message paths changed. No root or Function moved, and all semantic names/outputs stayed fixed.
- Task 4: IR/Case/manifest positions; roots; 25 Function substitutions from step functions to `<machine>.rules.<action>`; activity lint prose changed from disabled effects to disabled rules; hints/hintsRefused function names and the new rules fixture. Tables and Check receipts were byte-identical.
- Task 5: positions, roots, and Function symbols for remaining `<machine>.rules.<action>`/section moves; two Nexus lint acceptance reasons name rules; the `disabled-by-default` acceptance for `nexusProduct`'s `handlerReply-handlerError-true in scheduled` was removed because class-ruled lowering now emits explicit Nil cases. Silent-rejection and never-enabled findings stayed unchanged.

The per-task machine-readable ledgers remain at `.flow/tmp/fn-126/fn126-{1,2,3,4,5}/ir-deltas.json` (and task 2/3 `lifts-deltas.json`).

### Complete closing rename ledger

Direct object/machine/type/composition renames:

- `ActivityProtocol/activityProtocol`→`ActivitySystem/activitySystem`; `NexusProtocol/nexusProtocol`→`NexusSystem/nexusSystem`; `ProtocolState`→`SystemState`; `ProtocolFact`→`SystemFact`.
- `CurrentAdmission/currentAdmission`→`ActivityRecord/activityRecord`; `StaleAdmission/staleAdmission`→`TrustingActivityRecord/trustingActivityRecord`; `HeldAdmission/heldAdmission`→`HeldDispatch/heldDispatch`; `AdmissionResponseLoss/admissionResponseLoss`→`LostStartAnswer/lostStartAnswer` (the `FaultKind` spelling is retained).
- `CurrentRecord/currentRecord`→`RecordMember/recordMember`; `StaleRecord/staleRecord`→`TrustingRecordMember/trustingRecordMember`.
- `CurrentOverQueue/currentOverQueue`→`RecordOverQueue/recordOverQueue`; `StaleOverQueue/staleOverQueue`→`TrustingRecordOverQueue/trustingRecordOverQueue`; `CurrentOverMatching/currentOverMatching`→`RecordOverMatching/recordOverMatching`; `StaleOverMatching/staleOverMatching`→`TrustingRecordOverMatching/trustingRecordOverMatching`; `CurrentOverForgetful/currentOverForgetful`→`RecordOverForgetful/recordOverForgetful`; `CurrentOverVolatile/currentOverVolatile`→`RecordOverVolatile/recordOverVolatile`; `CurrentOverLossyMatching/currentOverLossyMatching`→`RecordOverLossyMatching/recordOverLossyMatching`.
- `DispatchQueue/dispatchQueue`→`TaskQueueProduct/taskQueueProduct`; `DispatchQueueUnderStorageLoss/dispatchQueueUnderStorageLoss`→`TaskQueueProductUnderStorageLoss/taskQueueProductUnderStorageLoss`; `MatchingQueue/matchingQueue`→`TaskQueueSystem/taskQueueSystem`.
- `ForgedCompletion/forgedCompletion`→`TrustingCaller/trustingCaller` for the object/machine and all target/action/state/state-field/outcome/fact owners; Query `forgedCompletion` stays unchanged.
- Activity and Nexus-operation actor `caller`→`client`; `protocolFact`→`systemFact`; IR file `activity-system`→`activity-record`; export val `activitySystem`→`activityRecord`.

Action/action-class renames:

- `attemptStart`→`poll`; `attemptResult`→`respond` and `attemptResult-{completed,failed-*,canceled}`→`respond-{completed,failed-*,canceled}`.
- `answerDelivery`→`answerMatching`; `handlerReply`→`reply` and `handlerReply-{syncSuccess,async,operationFailed,operationCanceled,handlerError-*,syncFailure,syncCanceled}`→the corresponding `reply-*` classes. `CauseKind.handlerReply`, `cause.handlerReply`, and `visibility.handlerReply.*` remain.
- `transportFault`→`fault`; `callerClose`→`close`; `handlerFinish`→`finish` and its succeeded/failed/canceled classes; `workerStop`→`stop`; `workerResume`→`resume`. The retained `FaultKind` spellings remain.

Every renamed Query (suffix braces mean every listed full key):

- `activityProtocol.{cancelIsRequested,terminateSettles}`→`activitySystem.{cancelIsRequested,terminateSettles}`.
- `admissionResponseLoss.committed`→`lostStartAnswer.committed`; `heldAdmission.staleDelivery`→`heldDispatch.staleDelivery`.
- `currentAdmission.{admittedBeforePause,any.atMostOneActive,duplicateDelivery,duplicateDelivery.monitored,pausedIsNotDispatched,product.pausedIsNotDispatched,scheduleToCloseFirst,scheduleToStartFirst,staleDelivery,startedAfterCompletion.monitored,terminalStatesAreFinal}`→the same suffixes under `activityRecord`; the identical `staleAdmission.*` set→`trustingActivityRecord.*`.
- `currentOverQueue.{admittedBeforePause,any.atMostOneActive,duplicateDelivery,failedCommit,pausedIsNotDispatched,staleDelivery,terminalStatesAreFinal}`→`recordOverQueue.*`; the identical `staleOverQueue.*` set→`trustingRecordOverQueue.*`.
- `currentOverMatching.{admittedBeforePause,any.atMostOneActive,crashAfterAdmissionCommit,deliveredAgainAfterLostAck,pausedIsNotDispatched,staleDelivery,terminalStatesAreFinal}`→`recordOverMatching.*`; the identical `staleOverMatching.*` set→`trustingRecordOverMatching.*`.
- `currentOverLossyMatching.{admittedBeforePause,any.atMostOneActive,crashAfterAdmissionCommit,deliveredAgainAfterLostAck,pausedIsNotDispatched,staleDelivery,terminalStatesAreFinal}`→`recordOverLossyMatching.*`.
- `matchingQueue.{any.committedStays,crashAfterAcknowledgment,crashAfterDelivery,crashAfterInvocation,crashAfterPersistence,crashAfterSyncMatch}`→the same suffixes under `taskQueueSystem`.

Every renamed Case file:

- `activity-race-admissionResponseLoss.committed-case.json`→`activity-race-lostStartAnswer.committed-case.json`.
- `activity-race-heldAdmission.staleDelivery-case.json`→`activity-race-heldDispatch.staleDelivery-case.json`.
- `activity-activityProtocol.cancelIsRequested-case.json`→`activity-activitySystem.cancelIsRequested-case.json`.
- `activity-activityProtocol.terminateSettles-case.json`→`activity-activitySystem.terminateSettles-case.json`.

Every renamed law claim:

- `currentAdmission.{pausedIsNotDispatched,terminalStatesAreFinal}`→`activityRecord.*`; `staleAdmission.{pausedIsNotDispatched,terminalStatesAreFinal}`→`trustingActivityRecord.*`.
- `currentOverQueue.{pausedIsNotDispatched,terminalStatesAreFinal}`→`recordOverQueue.*`; `staleOverQueue.{pausedIsNotDispatched,terminalStatesAreFinal}`→`trustingRecordOverQueue.*`.
- `currentOverMatching.{pausedIsNotDispatched,terminalStatesAreFinal}`→`recordOverMatching.*`; `staleOverMatching.{pausedIsNotDispatched,terminalStatesAreFinal}`→`trustingRecordOverMatching.*`.
- `currentOverLossyMatching.{pausedIsNotDispatched,terminalStatesAreFinal}`→`recordOverLossyMatching.*`.
- `activityProtocol.{cancelIsRequested,terminateSettles}`→`activitySystem.*`.

Every renamed lint owner/key prefix:

- `admissionResponseLoss`→`lostStartAnswer`; `heldAdmission`→`heldDispatch`; `activityProtocol`→`activitySystem`; `currentAdmission`→`activityRecord`; `staleAdmission`→`trustingActivityRecord`; `currentRecord`→`recordMember`; `staleRecord`→`trustingRecordMember`.
- `dispatchQueue`→`taskQueueProduct`; `dispatchQueueUnderStorageLoss`→`taskQueueProductUnderStorageLoss`; `matchingQueue`→`taskQueueSystem`; `nexusProtocol`→`nexusSystem`.
- `currentOverQueue`→`recordOverQueue`; `staleOverQueue`→`trustingRecordOverQueue`; `currentOverMatching`→`recordOverMatching`; `staleOverMatching`→`trustingRecordOverMatching`; `currentOverLossyMatching`→`recordOverLossyMatching`.
- Lint subject/action-class tokens follow the exhaustive action/action-class map above. No lint kind, verdict, or acceptance meaning changed.

The authoritative machine-readable maps are `.flow/tmp/fn-126/tools/renames.json` and `.flow/tmp/fn-126/task8/additional-renames.json`.

### Verification

- `bash .flow/tmp/fn-126/tools/prove.sh --renames .flow/tmp/fn-126/tools/renames.json --renames .flow/tmp/fn-126/task8/additional-renames.json --repo "$PWD" .flow/tmp/fn-126/task8/before "$PWD" .flow/tmp/fn-126/task8/prove-final-2`: `RESULT: OK`; projection byte-identical (88,856,698 bytes), 21/21 Cases, all seven IR files, lint/laws/generated/canary/names.
- `make umpire-check-model && make lint-model`: exit 0 (`.flow/tmp/fn-126/task8/final-model-and-lint-green.log`).
- `GOLANGCI_LINT_BASE_REV=8bf917bf0b84dc263b9e732a25c498678b3053f4 GOLANGCI_LINT_FIX=false make lint-code-fast`: 0 issues (`lint-code-fast-task-base.log`).
- `go test -count=1 -tags test_dep -p 2 ./tools/umpire/...`: exit 0 (`final-go-suite.log`).
- `make umpire-check-cases && make umpire-check-fixtures && make canary-check-case`: exit 0 (`final-smoke.log`).
- Strict red/green receipts include the R20 both-wrong/missing-refinement fixtures and the full-model retired-vocabulary coverage (`review-r20-{red,green}.log`, `review-r10-{red,green}.log`).

Review round 1 found three introduced P2 gaps; all were fixed and round 2 returned SHIP. The non-mechanical paired-validator lesson was captured at `.flow/memory/bug/integration/paired-level-validators-must-check-each-2026-10-06.md`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 90491988a349c4f1bd88bc0dfa9ce1e67b422c24, f98632f94aeaf477c41bd4a136b59c1d20fc8222, 9e0f0b923b339a75793bf573672409edca4e9428, ebeaa0a62b8254f926e13f32d32c0d1c951c66a6, b919ece45d6381fc4687f3cbfb3d9f44e1690b44, e6b1192bb81bcf7c6a55c115ce6df073edb3f343, 86d7cde58ed88026785c92f34cdd3ef5e80a4736, 7e8e559755044d72e38c4f2ed5dc1ffa56ddde38
- Tests: make umpire-check-model && make lint-model, GOLANGCI_LINT_BASE_REV=8bf917bf0b84dc263b9e732a25c498678b3053f4 GOLANGCI_LINT_FIX=false make lint-code-fast, go test -count=1 -tags test_dep -p 2 ./tools/umpire/..., make umpire-check-cases && make umpire-check-fixtures && make canary-check-case, bash .flow/tmp/fn-126/tools/prove.sh --renames .flow/tmp/fn-126/tools/renames.json --renames .flow/tmp/fn-126/task8/additional-renames.json --repo "/Users/stephan/Workspace/skunkworks/umpire/temporal" .flow/tmp/fn-126/task8/before "/Users/stephan/Workspace/skunkworks/umpire/temporal" .flow/tmp/fn-126/task8/prove-final-2
- PRs:
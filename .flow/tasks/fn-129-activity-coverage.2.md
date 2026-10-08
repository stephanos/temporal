---
satisfies: [R2, R5]
---
# fn-129-activity-coverage.2 Respond by ID: a service actor

## Description

Implement R2's independent service rows and typed controller bindings, with required completion/failure/cancellation R5 Cases and the bounded held external-settlement bridge. Consume .1's integrated source; commit focused source proof independently. Production/full/live acceptance stays at .5.

**Size:** L by file count, bounded to the service-answer/external-settlement slice under the owner's fixed five-task decomposition.
**Files:** standalone signature/Product/System/Realization and co-located by-ID/rejection/refinement tests; exact visibility, typed external-settlement/publication basis, carrier/lowering and absent-worker fixtures; bounded Program/Profile/VM/native schema closure; owning protocol docs.
**Touches:** [model/temporal/features/activity/standalone/**, model/temporal/realize/Realize.scala, model/temporal/realize/Kit.scala, model/temporal/realize/Modules.scala, model/temporal/realize/Behavior.scala, model/umpire/realize/Scripts.scala, model/irgen/Realizations.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/realizationRefusals/**, proto/internal/temporal/server/api/umpire/v1/ir.proto, proto/internal/temporal/server/api/testpilot/v1/program.proto, proto/internal/temporal/server/api/testpilot/v1/instruction.proto, proto/internal/temporal/server/api/testpilot/v1/run.proto, api/umpire/v1/ir*.pb.go, api/testpilot/v1/program*.pb.go, api/testpilot/v1/instruction*.pb.go, api/testpilot/v1/run*.pb.go, common/testing/testpilot/contract.go, common/testing/testpilot/contract/profile.go, common/testing/testpilot/contract/profile_test.go, common/testing/testpilot/internal/execution/**, common/testing/testpilot/temporal/profile.go, common/testing/testpilot/temporal/profile_test.go, common/testing/testpilot/temporal/driver_test.go, common/testing/testpilot/temporal/worker/interpreter.go, common/testing/testpilot/temporal/worker/sdk.go, common/testing/testpilot/temporal/worker/driver.go, common/testing/testpilot/temporal/worker/session.go, common/testing/testpilot/temporal/worker/*activity*test.go, tools/umpire/realization/**, tools/umpire/lower/**, model/README.md, model/SEMANTICS.md, common/testing/testpilot/README.md, common/testing/testpilot/temporal/README.md, common/testing/testpilot/temporal/worker/README.md]

## Approach

- Add the service actor's completed, failed and canceled by-ID actions, bound to the existing activity entity. Author independent phase/outcome/rejection rows and Product refinement. Separate scheduled/paused force completion from held-attempt failure/cancellation; enumerate retry policy, exhaustion, fatality and controls. The comparator has no CanceledByID event. Consult frontend synthesis and cancellation preconditions without assuming a token cancellation row is the by-ID specification.
- Reuse `rpc`, `RequestBase`, generated WorkflowService method constants, response reads and generic unary transport. Learn `StartActivityExecutionResponse.run_id` as `activityExecutionRunID` and bind the namespace/activity/run of subsequent calls. Leave workflow_id empty for standalone. Never use the Testpilot Run ID as the activity execution run or invent an attempt number. Preserve the public count as raw scheduling data.
- Add exact method-to-Describe visibility hints with current transaction citations. Completion/cancellation empty responses and accepted failure RPCs prove their call outcomes only; Describe supplies terminal status independently. Retain unknown/wrong descriptor, incomplete/crossed assignment and missing visibility refusals.
- Author a separate controller-only realization/Query for scheduled completion with no workers section, SDK entrypoint, reservation or fault. Use the existing `temporal/driver.go:120` path. Pin emitted Program absence and a runtime fixture with zero SDK registration/poll; distinguish omitted, nil-instruction and empty-instruction worker entrypoints. No generic worker optionality subsystem is needed.
- Require authored realized scheduled no-poll completion, held ById failure and held ById cancellation Cases. Choose held fatal failure without cancellation for an unambiguous terminal failure witness; cancellation first establishes the required cancel request and then uses the canceled-by-ID API. Independent Model checks still cover retry/fatal/control/rejection rows. Their live evidence remains pending .5, not optional or replaced by scheduled completion.
- Extend .1's immutable zero/one-heartbeat-prefix plus single-disposition group only with the smallest explicit typed external-control settlement basis for these two held Cases. Bind the declaration to the typed ById method/request, activity execution run and reservation/delivery identity, with a positive bounded settlement/cleanup contract. No new by-ID transport opcode is needed. External-control basis is distinct from request-field timeout basis: a valid external-control declaration with no selected timer is admitted; missing/unknown basis rejects. Preserve .1's requirement for positive heartbeat basis plus one selected timer on its timeout paths. Never derive basis/visibility from action names or treat a pending local return as server acceptance.
- Prove a bounded activity-local pending-publication handoff using the existing Program/Profile/VM outcome/slot seam, extending only its typed activity identity if absent. AwaitCommand is workflow-only, not this barrier. Before the controller's ById answer, require actual local pending record publication and bounded Describe held evidence; after the RPC require the typed response plus bounded Describe settlement. Return exact unwrapped SDK ErrResultPending with no worker answer RPC, drain that reservation immediately, retain authority for cleanup and external server closure, and lower chronologically without moving late evidence backward. A controller answer is not a worker script answer. Pin source/native fixtures first; any missing narrow contract is a located blocker, never unsupported live credit.
- Test valid timer-free external control, missing/unknown/crossed basis, wrong method/request identity, no held attempt/cancel request, duplicate external response, early/late/absent publication, heartbeat timer basis with no selected occurrence, external response misclassified as a worker answer, absent versus empty worker and incomplete cleanup. Focused recording/replay must retain expected terminal status/reason and raw Describe count. The proven publication seam is the one .3 consumes; no generic recorder/control subsystem, synthetic timeout waiver or extra per-attempt disposition.
- Preserve fn-138's known retryable cancellation-requested failure disagreement; keep independently authored service treatment explicit. Never copy the server Failed branch into the Model merely to reconcile that disagreement. New live disagreement requires human judgment. Retain focused source row/rejection mutations and scratch current-source lift/lower plus all three Case identities/expected-run proof. Necessary changed native schema mirrors and ignored model/build ScalaPB jars may be regenerated for compilation; production Model IR/Cases/managed fixtures and unrelated generated outputs stay unchanged.

## Investigation targets

**Required** (read before coding):
- `model/temporal/features/activity/standalone/Standalone.scala:62` and `model/temporal/features/activity/standalone/system/System.scala:270`.
- `model/temporal/features/activity/standalone/system/Realization.scala:38`.
- `model/umpire/realize/Scripts.scala:80` and `model/temporal/realize/Behavior.scala:15`.
- `tools/umpire/realization/carriers.go:107` and `tools/umpire/lower/waits.go:540`.
- `common/testing/testpilot/temporal/driver.go:120` and `common/testing/testpilot/temporal/internal/primitive/primitive.go:85`.
- `service/frontend/workflow_handler.go:1727` and `chasm/lib/activity/model/model.go:101`.
- `common/testing/testpilot/internal/execution/scheduler.go:881`, worker pending/Drain seams from .1 and `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` AwaitSlot/AwaitInstruction.

## Quick commands

```bash
mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*Activity*' --require-tests
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/realization ./tools/umpire/lower -run 'ByID|ById|Carriers|Activity|ReadsWait'
go test -tags test_dep -p 2 -timeout 30m ./common/testing/testpilot/temporal ./common/testing/testpilot/temporal/worker -run 'ControllerOnly|AbsentWorker|Composite|Activity'
go test -tags test_dep -p 2 -timeout 30m ./common/testing/testpilot/internal/execution -run 'Activity|External|Publication|ReservationOutcome'
```

Serialize heavy scratch/Go commands with the shared flock. Run focused lifter/descriptor fixtures and changed-file format/lint. Record actual command selectors and nonzero test counts. Full gates and live run remain .5.

## Acceptance

- [ ] Independent service phase/rejection matrix is pinned with retry/fatal/control cases, Product refinement and mutation negatives; comparator's absent cancellation form and retained disagreement are documented.
- [ ] All three typed ById request carriers/lowerings use true learned execution run identity and have authored realized Cases. Required held failure/cancel scratch Cases prove the declared external-control basis, actual holding, publication before answer, typed RPC and terminal Describe; live evidence awaits .5. Descriptor, crossed identity, malformed/incomplete request and missing visibility negatives fail closed.
- [ ] Scheduled completion scratch Case has no worker entrypoints/reservations and fixture execution registers/polls none. Empty/nil worker scripts do not pass the no-poll proof. Raw public attempt count is not delivery evidence.
- [ ] Native source fixture admits valid timer-free external settlement, returns exact unwrapped pending with no SDK answer RPC and immediate reservation Drain, then proves bounded publication/settlement/cleanup. Missing/unknown basis, false timer credit, wrong order/identity/method and duplicate/incomplete negatives fail closed; .1 heartbeat admission rules remain intact.
- [ ] Focused scratch recording/replay retains exact authored assessment/reason and Case/Program/Contract identities for completion/failure/cancellation, with original source/native commit and new Query/Case inventory for .5. No production regeneration/full suite/live acceptance is claimed here.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R3]
---
# fn-83-author-a-live-case-from-a-model-file.5 Fault lines in the Scenario and the worker-outage Model

## Description
Add `fault <kind> before|after <hook>` to the `scenario` command, lower it in `Umpire.Case.Producer` through the existing intent and realization types with the hook-to-instruction map and the dependency-edge rewriting the spec's R3 defines, add the outage-order rule whenever a Scenario carries faults, and re-author the worker-outage Case as a Model on the `workflow` template (R3). The hand-written outage file is deleted. Depends on .4 because both edit the outage live test and the artifact tests.

Paths are pre-fn-82; `Umpire.Space` is `Umpire.Variations` afterwards.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Success/Syntax.lean` (fault arm on `scenario`; hook and duplicate diagnostics in the `case` elaborator), `model/Umpire/Case/Producer.lean` (fault lowering, edge rewriting, outage-order rule), `model/Umpire/Case/Tests/Producer.lean`, `model/Temporal/Feature/Workflow/Outage.lean` (new Model file; final path a task decision under `Temporal/Feature/Workflow/`), `model/Temporal/Testpilot/WorkerOutage.lean` (deleted), `model/Temporal/Testpilot.lean`, `model/Temporal/TestpilotTests.lean`, `model/Temporal/Feature/Nexus/Success/Tests.lean` (two fault `#guard_msgs`), `tests/testcore/testpilot/worker_outage_artifact_test.go` (rule ID, terminal state, Deadline pin), `tests/testpilot_worker_outage_case_test.go` (Verdict shape)
**Touches:** [model/Temporal/Feature/Workflow/**, model/Temporal/Feature/Nexus/Success/Syntax.lean, model/Temporal/Feature/Nexus/Success/Tests.lean, model/Umpire/Case/**, model/Temporal/Testpilot/**, model/Temporal/Testpilot.lean, model/Temporal/TestpilotTests.lean, tests/testcore/testpilot/worker_outage_artifact_test.go, tests/testpilot_worker_outage_case_test.go]

### Approach
- Grammar: a repeated `fault` line after `actions exactly`; the `scenario` macro stores the lines as data; the `case` elaborator resolves hooks against the template and rejects duplicates (same kind, hook, placement). Keep the Scenario's checked value free of template knowledge.
- Lowering: for each line build a `FaultIntentDeclaration.atOccurrence` with the hook as occurrence and the kind's capability ID, and a `FaultRealization` whose `instructionId` is `<hook>-<kind>` and `roleId` is the template's task-queue role; insert the lowered instruction into the controller entrypoint. `lower` keeps ignoring the occurrence.
- Edge rewriting: controller nodes run concurrently unless an edge orders them, and a realization can only make the fault depend on something. For `before`, the hook's instruction gains a dependency and a succeeded-guard on the fault; for `after`, every successor of the hook gains a dependency on the fault. Today's hand-written Program is the oracle: `start-workflow` depends on the stop, `history` depends on the resume. A Producer unit test asserts the resulting edges and guards for both placements, not only the inserted instruction.
- Outage-order rule: generalize the existing `outageOrderRule` (bounded liveness over `FAULT_INJECTED` events in declared order, `rule_events` Deadline, terminal state `resumed`) into the Producer, parameterized by the fault lines and `Realization.faultRuleId`. The Deadline value (16 today) becomes a template field; record the choice.
- Outage Model: states `pending`, `completed`; Action `awaitCompletion`; no Fact (the state `completed` already says what happened); Property requiring `state completed`; the Scenario from the spec; `limits`; a `find` Query; `case ... fixture "worker-outage" as workflow "umpire-worker-outage-workflow" evidence awaitCompletion ← history workflowExecutionCompleted`.
- Verdict shape changes: the hand-written safety rule `worker-outage-workflow-completed` is replaced by one Correlated clause rule per `require`, so the Verdict is the outage-order rule plus two clause rules. Update the live test's rule-count and rule-ID assertions and the artifact test accordingly; the completed history event is still read from the supporting evidence.

### Investigation targets
**Required:**
- `model/Umpire/Space/Lowering.lean:12-58` — the header stating the occurrence is unread, capability IDs, `FaultRealization`, `lower`
- `model/Temporal/Testpilot/WorkerOutage.lean:60-146, 148-226` — intents, realizations, the Program's edges (`start-workflow` after stop, `history` after resume), `outageOrderRule`, `workflowCompletedRule`, and the Case assembly
- `common/testing/testpilot/internal/execution/README.md:80-110` — controller scheduling: independent nodes run concurrently
- `tests/testcore/testpilot/worker_outage_artifact_test.go:22` — the pinned rule ID, violation state, and Deadline
- `tests/testpilot_worker_outage_case_test.go:31-130` — the two live tests and `requireWorkerOutageEvidence`
- `model/Temporal/Feature/Nexus3/Syntax.lean:225-241` — the `scenario` macro to extend

### Key context
- EVD-20 (Driver-realized faults) is drafted and pending human approval; build on it as fn-80 did and note it in the receipt.
- A stop with no resume is not an elaboration error; the outage-order rule answers it at Run time.
## Acceptance
- [ ] The worker-outage file is deleted; the outage Case is a Model file with two `fault` lines; its fixture regenerates through the registry
- [ ] A Producer unit test asserts the rewritten edges and guards for `before` and `after`
- [ ] Both live outage tests pass with the outage-order rule (same ID, terminal state `resumed`) and both clause rules satisfied; the artifact test pins the `rule_events` Deadline
- [ ] `#guard_msgs` pins unknown hook and duplicate fault line
- [ ] `make umpire-check-regression` passes
## Done summary
Blocked:
Blocked 2026-09-10 pending a redesign of the `case` abstraction (decided with the user, not yet a spec).

The per-Case `case` block (one Query, one hand-picked realization template, per-Case evidence lines) is being replaced by:

- **Sets per purpose.** A developer declares query sets by kind: functional and canary sets list Queries explicitly; exploratory sets state a coverage goal and a budget over a variation space.
- **One Case per Query.** A set compiles to many Cases run together; "Case = one Program + one Contract" stays.
- **A separate Temporal binding.** Runtime metadata (how an Action is caused, which recorded event confirms a step result, which resources a role needs) lives in a Temporal-owned binding declaration beside the behavioral Model, so Programs and Contracts are assembled from the Model plus its binding rather than from a whole-Program template chosen per Case.

This task builds on the `case` block, whole-Program templates, per-Case evidence, or the Case-registry shape that redesign replaces. Unblock or rewrite it once the redesign spec exists.
## Evidence
- Commits:
- Tests:
- PRs:

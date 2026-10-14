---
satisfies: [R17]
---
# fn-93-simplify-the-lean-model.30 Shared realization scaffolding and the declared-action check (A7)

## Description
Lane A7. Move the Realization copies (`workerNamespaceBinding`, `taskQueueBinding`, `rpc`, `historyAssignments`, `startWorkflowNode`, `awaitCloseNode`, `workflowTypeOf`, `sharedObservations`, `sharedRoles`, `historyNode`) from `Realization/{Workflow,Nexus}.lean` and `Conformance.lean`'s private role copies and `historyNode` into `Temporal/Case/Support.lean`. Take method paths from the generated API and event attribute field names from `EventKind.attributesField?`. Build realization action IDs through `DefinitionFamily`; `case` rejects at elaboration an action ID the set's Model does not declare.

**Size:** M
**Files:** `model/Temporal/Case/Realization/{Workflow,Nexus,Rpc}.lean`, `model/Temporal/Case/Support.lean`, `model/Temporal/Testpilot/Conformance.lean`, `model/Temporal/Case/Syntax.lean` (the check), `model/Temporal/Case/EventKind.lean`, `model/Temporal/Case/ReadKind.lean`, a new negative test under `model/Temporal/Case/Tests/**`
**Touches:** [model/Temporal/Case/**, model/Temporal/Testpilot/Conformance.lean]
**Depends on other specs:** fn-92 (Realization/Workflow cited in its spec), fn-89.5 (`Conformance.lean`).

### Approach
- Generated method names are dot-separated (`fullName` in `Temporal/API.lean`); the slash form used in Case literals (`Support.lean:33-37`, `ReadKind.lean:38,48`) is derived by one helper, byte-identical.
- `startWorkflowNode` differs (Workflow takes `instructionId`): unify with a parameter, not two copies.
- The new rejection is a located error naming the action and the Model; pin it with `#guard_msgs`.
- Case fixtures and the conformance corpus are the oracle.

### Investigation targets
**Required:**
- `model/Temporal/Case/Realization/Workflow.lean:20-100`, `model/Temporal/Case/Realization/Nexus.lean:160-260,620-630`
- `model/Temporal/Case/Support.lean`
- `model/Temporal/Testpilot/Conformance.lean:120-130,240-260`

### Quick commands
```sh
cd model && lake build Temporal TemporalModelTests
make umpire-check-case-runtime-conformance umpire-check-goldens canary-check-case
```

## Acceptance
- [ ] One scaffolding set in `Case/Support`; no copies in Realization or Conformance
- [ ] Method paths and attribute names come from generated data; action IDs built through `DefinitionFamily`
- [ ] Undeclared action ID rejected with a pinned located error; every Case fixture and the corpus byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:

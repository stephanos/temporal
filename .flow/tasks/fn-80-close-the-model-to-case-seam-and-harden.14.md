---
satisfies: [R1]
---
# fn-80-close-the-model-to-case-seam-and-harden.14 Emit ScopedEvidence from a Program-declared projection

## Description
Unblocks task .4 and closes fn-77's Known Gap `bounded-completion-is-model-only`. Both specs reached the same wall from opposite sides.

**Size:** L
**Files:** proto/internal/temporal/server/api/testpilot/v1/**; common/testing/testpilot/internal/verification/**; common/testing/testpilot/internal/execution/**; model/Testpilot/**
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/internal/execution/**, model/Testpilot/**]

### Why this task exists
Task .4 escalated `DEPENDENCY_BLOCKED` with this evidence:
1. `bindScoped` (`internal/verification/scoped_prepare.go:71-73`) rejects any Case whose `Contract.scoped` names an `evidence_observation_id` that is not a Program Observation typed exactly `temporal.server.api.testpilot.v1.ScopedEvidence`. The async-nexus Program declares only `history-event`, so a regenerated Case would not even `Prepare`.
2. An Observation is filled only by an `InvokeRPC` response projection, and the version-one `Instruction` table has no instruction that can produce a `ScopedEvidence` value. `run.proto:125` says it outright: ScopedEvidence is supplied only through the capability's declared typed Observation. No Temporal WorkflowService RPC returns that message.
3. So a scoped clause on such a Program receives zero evidence and `scopedRun.answer` (`internal/verification/scoped.go:622-642`) returns **SATISFIED vacuously** — shipping that would delete the only rule proving the Nexus operation completed and leave a hollow regression test.

fn-77's completion review named the same follow-up: a Program-declared source that lifts recorded history into `ScopedEvidence` with `identity`, `operation`, `kind` and `fields`.

### Scope
Add that projection so a scoped capability can run on the real Driver. **A clause that receives no evidence must not answer SATISFIED** — fix the vacuous-satisfaction hole as part of this, or state precisely why it is correct to leave it.

## Acceptance
- [ ] A Program can declare a source that lifts recorded history into a `ScopedEvidence` Observation carrying `identity`, `operation`, `kind` and `fields`.
- [ ] A Case whose `Contract.scoped` names that Observation passes `bindScoped` and `Prepare`.
- [ ] A scoped clause that receives zero evidence no longer answers SATISFIED vacuously — either it is inconclusive/violated, or the summary argues precisely why vacuous satisfaction is correct here.
- [ ] The scoped bounded-response clause runs on the real Driver, closing fn-77's Known Gap `bounded-completion-is-model-only`; the Known Gap record is updated rather than left stale.
- [ ] Regenerated fixture bytes are produced through owning targets and accounted for; conformance gates green.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

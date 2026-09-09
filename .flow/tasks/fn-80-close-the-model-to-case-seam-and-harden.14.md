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
Blocked:
BLOCKED: EXTERNAL_BLOCKED — acceptance item 4 is only observable against a live Temporal cluster,
and this environment has none. Deferred in favour of the fully offline-verifiable R2 chain
(.11 -> .12 -> .13); reset this task to `todo` and take it in an environment with a cluster.

Why acceptance 4 is live-gated
- "The scoped bounded-response clause runs on the real Driver, closing fn-77's Known Gap
  `temporal.nexus3.typed-nexus.bounded-completion-is-model-only`". That Known Gap is carried by the
  typed-nexus Case (`model/Temporal/Feature/Nexus3/TypedNexus.lean:759-768`), and the only thing
  that runs that Case against the real Driver is `tests/testpilot_typed_nexus_case_test.go`, an
  `integration`-tagged live test. The offline conformance runner uses the facade Driver, so a green
  conformance gate would not close the gap.
- The task this one unblocks (.4) is live-gated for the same reason (its own acceptance requires
  `TestTestpilotAsyncNexusCase` satisfied in both environments), so the whole chain terminates in a
  cluster requirement. Nothing is lost by taking it where the cluster is.

Design worked out this session — do not re-derive it
1. The mechanism already exists end to end; only the *source* of the value is missing.
   `common/testing/testpilot/scoped_facade_test.go:52-107` shows a scoped Case that Prepares and Runs
   today: it declares an Observation typed `ScopedEvidence`, and an `InvokeRPC` whose method
   *response type is `ScopedEvidence`* projects straight into it. `bindScoped`
   (`internal/verification/scoped_prepare.go:71-73`) is satisfied by that shape. So a Program CAN
   declare and fill a ScopedEvidence Observation — what no real Temporal RPC can do is *return* one.
2. Therefore the additive wire change is a lift, not a new instruction. `ProjectionTarget`
   (`proto/internal/temporal/server/api/testpilot/v1/instruction.proto:20-24`, a oneof over
   `slot_id` / `observation_id`) gains a third variant that names the ScopedEvidence Observation and
   carries the bindings that build the message from the projected value:
     - a guard `FieldPath` that must be present for the rule to fire (for Nexus history this is the
       `attributes` oneof selector, so `kind` is a literal per rule and the oneof case selects it —
       `CaseSupport.historyAttribute` already builds exactly that path shape);
     - `identity.scope` bindings (field_id + FieldPath, text), `identity.source` (literal),
       `identity.ordinal` (FieldPath to `event_id`);
     - `operation` (FieldPath), `kind` (literal), and `fields` bindings (field_id + FieldPath).
   One rule per history event kind (scheduled / started / completed) gives the whole lifecycle.
3. Go work: `internal/execution/dataflow.go:282-310` (`bindProjectionSinks`) type-checks the new
   variant at Prepare — target Observation must be singular `ScopedEvidence`, every bound path must
   type-check against the projected message; `internal/execution/projection.go` builds the message
   at run time and skips the rule when the guard path is absent.
4. Vacuous satisfaction (acceptance item 3) is a small, separable fix and does NOT need a cluster:
   `scopedRun.answer` (`internal/verification/scoped.go:622-642`) falls through to SATISFIED when
   `r.operations` is empty, so a clause that received zero evidence answers green. The fix is to
   return INCONCLUSIVE when no evidence was accepted at all. Check `scoped_test.go` for an existing
   expectation on the empty run before changing it. If a future run wants a landable slice of this
   task, that is the one to take first.

Impact: task .4 stays blocked behind this one.
Suggested resolution: run this task where `go test -tags 'test_dep integration' ./tests -run
TestTestpilot` can reach a cluster; implement (2)+(3)+(4) above, regenerate fixtures through their
owning targets, and update the Known Gap record rather than deleting it.
## Evidence
- Commits:
- Tests:
- PRs:

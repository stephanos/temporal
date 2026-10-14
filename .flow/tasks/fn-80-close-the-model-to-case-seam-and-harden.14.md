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
A Program can now declare a source that lifts recorded history into a `ScopedEvidence` Observation:
`ProjectionTarget` gains a `ScopedEvidenceProjection` variant whose guarded rules build `identity`,
`operation`, `kind` and `fields` from the projected value, with declared literals for the Run
coordinates a recorded fact does not carry and a dense per-Run source ordinal only the emitting
instruction can count. The typed-nexus Case routes its bounded-response clause through
`Umpire.Case.Scoped.lower` and satisfies it live on the real Driver, which closes fn-77's Known Gap
`bounded-completion-is-model-only`; the record is replaced by `completion-identity-is-unrecorded`,
which names what the lift still cannot say (a completed Nexus event records no operation identity,
so the operation key is the scheduled event it references and one representative completed step is
released).

Vacuous satisfaction is closed on both sides: a scoped capability that admitted no evidence answers
unresolved rather than reading silence as the empty-obligation satisfaction a total model trace has.
The Lean portable interpreter (`Testpilot.Scoped.Run.answers`), the Go runtime
(`scopedRun.answer`) and a new `unobserved` fixture scenario agree; the model kernel's own
`Umpire.Property.Scoped` semantics are untouched, because a model trace is total and a recorded
evidence stream is not.

Two changes outside the immediate surface, both load-bearing and argued in place:
- `Umpire.Observation.Projection.Coverage` now rebuilds the first element of a repeated field, as it
  already rebuilds an established optional one; a later index still rejects. The shipped capture
  path walks `history.events[0]`, so without this no real Case could request coverage. The
  `FieldLowering` pin moves to `.index 1` and gains a positive `.index 0` case.
- The Driver's hard `max_work_per_event` ceiling is sized for a scoped capability. The scoped
  stage's reservation is cubic in the accepted evidence count and is charged into the same
  per-event bucket, so the expression-evaluation ceiling it carried rejected every multi-operation
  scoped Case at its own first evidence event. A `CONSIDER` records the better fix (charge the
  reservation's increment).

Reviewer findings addressed in a second commit: a declared evidence source belongs to one
instruction on a controller entrypoint (dense ordinals cannot restart), a presence-terminal guard
rejects at Prepare (it answers false rather than absent, so it could never select a rule), and the
two-operation Case is now Prepared offline in `tests/testcore/testpilot/typed_nexus_artifact_test.go`.

Second-round reviewer P3s also addressed: `ScopedEvidenceRule` renumbers to 1..7 with no hole, one
shared limits check replaces the two copies that overwrote the work ceiling before validating it, a
plain Contract still rejects at its own modest per-event value so the raised Driver ceiling stays
visible, and the plan narrative records that this task closed both halves it described as open.

stage: impl-review - ran [33df4e80..b80786fc] SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 90d2cf3f60827afdbec357a9919cda022a0ec13f, 0ca7bcaa9d9f1a3a9d94aea14769871269ddc65c, b80786fcf2bf68f56f670e33b5739e8f539c7ad9
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make lint-model (169 errors, unchanged baseline, all in generated Temporal/API), make umpire-check-case-runtime-conformance, CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., go test -tags 'test_dep integration' ./tests -run '^TestTestpilotTypedNexusOperationsCase$' (scoped bounded-completion clause SATISFIED live), make umpire-check-live-tests (empty failure set across 4 passing identities), make umpire-check-regression, make lint-code GOLANGCI_LINT_FIX=false (128: errcheck 1, govet 4, revive 106, staticcheck 17 - unchanged baseline), go vet -tags test_dep ./... (15 pre-existing diagnostics, unchanged)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)

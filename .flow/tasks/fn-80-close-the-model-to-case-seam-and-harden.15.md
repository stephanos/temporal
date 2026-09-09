---
satisfies: [R1]
---
# fn-80-close-the-model-to-case-seam-and-harden.15 Replace the Nexus3 clause-for-clause equality gate

## Description
R1's separable product defect, extracted from blocked task .4. Task .4's handover states explicitly that this is worth closing on its own and should be re-planned rather than silently substituted into .4's acceptance — this task is that re-plan.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Testpilot.lean
**Touches:** [model/Temporal/Feature/Nexus3/Testpilot.lean]

### Why this task exists
The spec's Goal section names this as defect 1: `produceCompletionCase` takes the checked values, throws on any scoped clause, then compares each input to a fixed expected model. A model edit therefore produces a **lowering error instead of a different Case**, which is the single behaviour most at odds with the vision — the model is supposed to be the source of truth.

This half needs no `ScopedEvidence` evidence path, so it does not wait on the projection task or on .4.

### Scope
Replace the clause-shape match and the hand-written success rule with lowering driven by the checked model. Of the protections the equality gate provided, preserve: witness absence stays a `LoweringError`, and an unexpressible clause stays a `LoweringError` naming the clause. A changed Target, Behavior, Query or witness must become **different Case bytes rather than an error**.

Coordinate with the ordered-list task above if both are in flight — both touch `Testpilot.lean`'s `supportsSuccessProperty`.

## Acceptance
- [ ] A changed Target, Behavior, Query or witness produces different Case bytes, not a lowering error.
- [ ] Witness absence still produces a `LoweringError`; an unexpressible clause still produces a `LoweringError` that names the clause.
- [ ] The hand-written success rule and the clause-shape equality match are gone.
- [ ] Regenerated fixture bytes are accounted for through owning targets; the async-nexus Case still satisfies live, or the reason it cannot is recorded rather than worked around.
- [ ] `make lint-model` adds nothing to the 169 baseline.

## Done summary
Replaced the Nexus3 Producer's clause-for-clause equality gate with lowering driven by the checked
model: `produceCompletionCase` now carries the checked Target, Property, Behavior, Query and
witness into the Case instead of comparing each against one expected value, and the
correlated-history monitor rule is derived from the Facts the selected witness records through
per-Fact Nexus history evidence projections the Producer declares. `sameTarget`, `sameQuery`,
`supportsSuccessProperty` and the hand-written `successRule` are gone.

The derived rule reproduces the deleted hand-written rule byte-for-byte for the shipped model, so
`make umpire-check-case-runtime-conformance` passes with no fixture regeneration and the live
async-nexus Case behaviour is provably unchanged.

Preserved rejections: a scoped clause, an absent witness, a Fact with no declared Nexus history
evidence, and a clause no witness step carries all reject as `LoweringError` naming the subject.

stage: impl-review - ran | verdict SHIP (model: claude-fable-5-1 at high); three P3 findings, two
addressed in a follow-up commit (Property-axis and one-Fact-chain guards, docstring wording), the
third accepted and documented in the code.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 331373dd2, faf868d27
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make lint-model (169 errors, all generated Temporal/API; unchanged from baseline), make umpire-check-case-runtime-conformance, CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., GATE_SKIPPED:live-integration:go test -tags 'test_dep integration' ./tests -run TestTestpilot needs a live cluster; the generated Case bytes are unchanged (conformance gate diffs both trees), so live behaviour is unchanged by construction
- PRs:
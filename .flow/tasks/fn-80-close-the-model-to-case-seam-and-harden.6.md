---
satisfies: [R8]
---
# fn-80-close-the-model-to-case-seam-and-harden.6 Add the query all verify form

## Description
Dependency correction, 2026-09-09: this task carried a stale `depends_on` edge to task .5, which was
superseded by the .10-.13 sequence. Task .5's own Superseded note records that .6 "was re-pointed to
depend on .11"; the .11 edge was added but the .5 edge was never removed, so .6 stayed blocked on a
task that will never complete. The edge was dropped in `.flow/tasks/…6.json` — flowctl has no
`dep remove` verb — and nothing else about the task changed.

Implements R8 (spec §R2 and R8). Replaces the always-erroring `query ... all ...` macro with elaboration through `QueryForm.verify`, and makes the Producer reject a verify Query as witness-absent.

**Size:** S
**Files:** `model/Temporal/Feature/Nexus3/Syntax.lean`, `model/Temporal/Feature/Nexus3/Authoring.lean`, `model/Temporal/Feature/Nexus3/Tests.lean`, `model/Temporal/Feature/Nexus3/RaceSyntaxTests.lean`, `model/Temporal/Feature/Nexus3/Testpilot.lean` (witness-absent path)
**Touches:** [model/Temporal/Feature/Nexus3/**]

### Approach
- The stub at `Syntax.lean:127-128` becomes a real macro expanding to `Authoring.check` with `QueryForm.verify` (`Umpire/Query/Language.lean:119-126`: `.verify` yields `.universal` / `.verifiedWithinLimits`).
- `Authoring.CheckedModel` must carry an optional witness so `produce` can reject a verify Query with `witness.absent` (the rejection retained from task .4).
- Add a verify Query over the R2 race lifecycle and one over an unsatisfiable Behavior asserting `unsatisfiable`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:100-140` — `QueryForm` and quantifiers
- `model/Temporal/Feature/Nexus3/Authoring.lean:411-459` — `CheckedModel` and `check`

**Optional** (reference as needed):
- `model/Umpire/Query/Authoring.lean:243-300` — `query%` elaborator

## Acceptance
- [ ] `query <name> on <model> all <property> in <behavior> limits <limits>` elaborates and checks on the race lifecycle
- [ ] A verify Query over an unsatisfiable Behavior reports `unsatisfiable` (PLN-05), asserted with `#guard`
- [ ] `produce` rejects a verify-form `CheckedModel` with a `witness.absent` `LoweringError`, asserted with `#guard`
- [ ] `lake build Temporal TemporalModelTests` and `make lint-model` pass

## Done summary
Replaced the always-erroring `query ... all ...` stub with real elaboration through
`QueryForm.verify`. The command expands to the same `Authoring.check` owner under a new
`QueryFormKind.verifyClaim`, which selects the exhaustive planner policy the verify form requires
(`validateStrategy` rejects any other strategy for that form).

`CheckedModel.witness` became `Option BehaviorTrace`: a verify Query establishes its claim over
every trace the Behavior admits and selects none. The Producer therefore rejects a verify-form model
as `witness.absent` without any new code — a Case realizes one selected trace.

`AdmissionError.noWitness` became `notSelected (outcome : PlanningOutcome)`, so a failed Query says
what planning actually delivered. That is what makes the two error cases assertable: an unsatisfiable
Behavior reports `.unsatisfiable` (PLN-05), and a requirement an admitted trace violates reports
`.found _ .violatingCounterexample` rather than slipping through as verified.

Verify Queries now check on both lifecycles: `verifiedCompletion` over the success slice and
`cancellationVerified` over the second lifecycle task .13 added.

Dependency correction: this task carried a stale `depends_on` edge to task .5, which the .10-.13
sequence superseded. Task .5's own Superseded note records that .6 "was re-pointed to depend on
.11"; the .11 edge had been added but the .5 edge never removed, so .6 was blocked on a task that
will never complete. The edge was dropped in `.flow/tasks/…6.json` — flowctl has no `dep remove`
verb — and recorded at the top of this task's description.

stage: impl-review - ran | verdict SHIP (model: claude-fable-5-1 at high); 2 P3 findings, the
counterexample-coverage one addressed in a follow-up commit, the other (witness optionality
correlated with the claim only by convention) is the shape the task's own Approach section asked
for and is recorded, not changed.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cd67a1f2b, HEAD
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make umpire-check-case-runtime-conformance (green, no fixture bytes moved), make lint-model (169 errors, all generated Temporal/API; Umpire.Lint and Shared clean; unchanged from baseline), CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., GATE_SKIPPED:live-integration:go test -tags 'test_dep integration' ./tests -run TestTestpilot needs a live cluster; the conformance gate proves the generated Case bytes are unmoved
- PRs:
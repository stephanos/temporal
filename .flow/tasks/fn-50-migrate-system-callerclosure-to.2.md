---
satisfies: [R2, R3, R4, R5]
---
# fn-50-migrate-system-callerclosure-to.2 Pin CallerClosure target and correspondence compatibility

## Description
Verify the checked Target and every live cross-layer consumer against the FiniteMachine-backed representation (R2-R4).

**Size:** M
**Files:** `model/Temporal/System/Nexus/CallerClosure.lean`, `model/Temporal/System/Nexus/Tests.lean`, `model/Temporal/System/Nexus/ImplementationLink.lean`, `model/Temporal/System/Nexus/ImplementationLinkTests.lean`, `model/Temporal/ImplementationLinkTests/Nexus.lean`, `model/Temporal/Tool/RunEvaluationMutationTests.lean`
**Touches:** [model/Temporal/System/Nexus/CallerClosure.lean, model/Temporal/System/Nexus/Tests.lean, model/Temporal/System/Nexus/ImplementationLink.lean, model/Temporal/System/Nexus/ImplementationLinkTests.lean, model/Temporal/ImplementationLinkTests/Nexus.lean, model/Temporal/Tool/RunEvaluationMutationTests.lean]

### Approach
- Compare checked target identity, behavior description, fingerprint, providers, capabilities, and planner output to the existing baselines.
- Compile the System Observation and Feature/System correspondence through the retained public equality/conjunction seams, with no new representation unfolding or list-membership rewrites at consumer sites.
- Retain independent Run Evaluation mutation oracles and exact caller-closure trace/result expectations.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/CallerClosure.lean:180-305` — definitions, target, and planning outputs
- `model/Temporal/System/Nexus/ImplementationLink.lean:424-674` — all direct domain and authority consumers, including the trailing conjunction destructuring proofs
- `model/Temporal/System/Nexus/Tests.lean` — checked Target compatibility pattern
- `model/Temporal/System/Nexus/ImplementationLinkTests.lean` — public authority seam
- `model/Temporal/ImplementationLinkTests/Nexus.lean` — exact composed correspondence
- `model/Temporal/Tool/RunEvaluationMutationTests.lean` — independent System trace oracle
## Acceptance
- [ ] Checked target JSON, fingerprint, definitions, providers, capabilities, and planner action order are unchanged.
- [ ] Observation, Implementation Link, and Run Evaluation consumers use only retained public seams and compile unchanged semantically.
- [ ] Existing Implementation Link `change`/`rcases` forms through the complete authority-consumer range compile without list-membership rewrites.
- [ ] Existing trace, Evidence Link, verdict, result, artifact, and mutation expectations remain exact.
- [ ] Temporal System, composed link, model, and experimental suites pass.
## Done summary
Pinned System CallerClosure's exact definition inventory, composition, checked planning, canonical JSON, and Behavior Fingerprint, plus the checked System-to-Feature correspondence identity, mapping order, and fingerprints. Existing public authority seams and exact Observation, Evidence Link, trace, verdict, result, artifact, and mutation oracles compile unchanged; the inherited branch-wide Go lint baseline was reproduced with no task-path findings.

stage: impl-review - ran [2026-09-01T04:53:46Z..2026-09-01T04:55:45Z]
## Evidence
- Commits: 83200adc68c6781a3c26bc2dac30cf0dced7ad21, b11138e6b8699e5db9933339747a51c13e785a77
- Tests: cd model && mise exec -- lake build Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus, cd model && mise exec -- lake build TemporalModelTests TemporalExperimentalTests UmpireTests, make umpire-check-regression, make lint-model, make lint-code (accepted inherited branch-wide red: 1373 findings — errcheck 220, exhaustive 6, forbidigo 211, govet 5, revive 792, staticcheck 135, testifylint 4; no task Go paths; six --fix edits exact-inverse-restored), baseline: red (make lint-code failed pre-edit with the identical 1373 inherited findings; accepted by standing roadmap policy)
- PRs:
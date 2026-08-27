---
satisfies: [R2, R6]
---
# fn-28-authorized-remote-staging-black-box.2 Bind the remote RuntimeConfiguration and public evidence mapping

## Description
Implement R2's exact remote RuntimeConfiguration and Temporal-owned public-boundary Observation mapping while preserving the semantic kernel.

**Size:** M
**Files:** `model/Temporal/System/Execution/RemoteStaging.lean`, `model/Temporal/System/Execution/RemoteStagingTests.lean`, `model/Temporal/Feature/Nexus/Execution.lean`, `model/Temporal/Feature/Nexus/ExecutionTests.lean`, `model/Temporal/Tool/RunEvaluation/**`, `model/Temporal/Tool/RunEvaluationTests.lean`, `tools/umpire/temporal/nexus/testdata/caller-closure-remote-input-set/**`
**Touches:** [model/Temporal/System/Execution/RemoteStaging.lean, model/Temporal/System/Execution/RemoteStagingTests.lean, model/Temporal/Feature/Nexus/Execution.lean, model/Temporal/Feature/Nexus/ExecutionTests.lean, model/Temporal/Tool/RunEvaluation/**, model/Temporal/Tool/RunEvaluationTests.lean, tools/umpire/temporal/nexus/testdata/caller-closure-remote-input-set/**]

### Approach
- Define `temporal.runtime-profile.remote-staging-public-grpc` with exact phase, authority-capability, action, API-call, workflow, worker, evidence, timeout, retry, and Known Gap Limits; include no raw target coordinate or credential.
- Produce the exact two-member input fixture with the byte-identical caller-closure ExperimentSpec and one distinct admitted RuntimeConfiguration.
- Add a closed public-evidence mapping branch for participant output, public history, control receipt, and cleanup receipt; reuse the existing Observation coordinates, Query, Property set, Behavior, transition kernel, and evaluation-outcome rules.
- Pin unknown versus unsupported versus conflict behavior and prove internal evidence or payload-derived facts cannot enter the mapping.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.2.md` — disposable CI runner and RuntimeConfiguration binding pattern
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime phases, profiles, and evidence source closure
- `.flow/tasks/fn-20-local-execution-semantic-conformance.2.md` — Lean-owned mapping/evaluator boundary
- `model/Temporal/Feature/Nexus/CallerClosure.lean` — unchanged semantic target and kernel
- `model/README.md:145-162` — compile/inspect versus runtime/Evidence Limitary

### Key context
Equivalent accepted observations may share semantic outcome identity across environments; configuration, run, provenance, receipt, and set identities must remain distinct.

### Acceptance
- [ ] The remote fixture changes only the RuntimeConfiguration member and preserves the ExperimentSpec bytes.
- [ ] The fixed public mapping derives every required coordinate or returns the exact unknown/unsupported/conflict status.
- [ ] No target coordinate, credential, internal evidence, payload meaning, alternate Property, or second evaluator enters the model.
- [ ] Local/CI model fixtures and generated regressions remain unchanged.

## Acceptance
- [ ] R2 remote configuration and fixed public evidence mapping are complete.
- [ ] Focused model, fixture identity, mapping mutation, and sibling regression suites pass.
- [ ] Existing semantic comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R2, R6]
---
# fn-29-bounded-production-canary-execution-and.2 Bind the no-fault canary RuntimeConfiguration and public evidence mapping

## Description
Implement R2/R6's distinct canary RuntimeConfiguration and exact Temporal-owned public Observation mapping while preserving the ExperimentSpec and semantic kernel.

**Size:** M
**Files:** `model/Temporal/System/Execution/ProductionCanary.lean`, `model/Temporal/System/Execution/ProductionCanaryTests.lean`, `model/Temporal/Feature/Nexus/Execution.lean`, `model/Temporal/Feature/Nexus/ExecutionTests.lean`, `model/Temporal/Tool/Conformance/**`, `model/Temporal/Tool/ConformanceTests.lean`, `tools/umpire/temporal/nexus/testdata/caller-closure-canary-input-set/**`
**Touches:** [model/Temporal/System/Execution/ProductionCanary.lean, model/Temporal/System/Execution/ProductionCanaryTests.lean, model/Temporal/Feature/Nexus/Execution.lean, model/Temporal/Feature/Nexus/ExecutionTests.lean, model/Temporal/Tool/Conformance/**, model/Temporal/Tool/ConformanceTests.lean, tools/umpire/temporal/nexus/testdata/caller-closure-canary-input-set/**]

### Approach
- Define `temporal.runtime-profile.production-canary-public-grpc` with exact phases, authority/action/API/workflow/worker/evidence/time/retry bounds and explicit empty fault/traffic/deployment/configuration capabilities.
- Produce a two-member input fixture containing the byte-identical caller-closure ExperimentSpec and one distinct admitted RuntimeConfiguration.
- Reuse the remote public execution-source schemas where identical; add a closed canary admission branch and an isolation receipt that can affect qualification but never Observation or Result.
- Derive the existing Observation coordinates only from admitted participant/history/control/cleanup facts and pin unknown, unsupported, and conflict behavior.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.2.md` — remote configuration/public mapping seam
- `.flow/tasks/fn-20-local-execution-semantic-conformance.2.md` — Lean-owned mapping/evaluator boundary
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime phases and evidence closure
- `model/Temporal/Feature/Nexus/CallerClosure.lean` — unchanged semantic target and kernel
- `model/README.md` — compile/inspect versus runtime/evidence boundary

### Key context
Equivalent qualified facts may share semantic outcome identity; configuration, run, environment provenance, receipt, and set identities must differ. An isolation attestation cannot supply semantic truth.

### Acceptance
- [ ] Canary changes only RuntimeConfiguration and preserves ExperimentSpec bytes.
- [ ] The fixed public mapping returns exact qualified/unknown/unsupported/conflict outcomes for every admitted source combination.
- [ ] No authority/isolation assertion, target coordinate, internal evidence, payload meaning, alternate Property, or second evaluator enters semantics.
- [ ] Local, CI, and staging fixtures/regressions remain unchanged.

## Acceptance
- [ ] R2/R6 canary configuration, mapping, and semantic purity are complete.
- [ ] Focused model, fixture-identity, mapping-mutation, and sibling regression suites pass.
- [ ] Existing semantic comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

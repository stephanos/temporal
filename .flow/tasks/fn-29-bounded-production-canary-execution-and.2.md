---
satisfies: [R2, R6]
---
# fn-29-bounded-production-canary-execution-and.2 Bind the no-fault canary RuntimeConfiguration and public evidence mapping

## Description
Bind R2/R6's exact no-fault production-canary configuration and public Evidence mapping into the caller-closure PortableTestPlan supplied by fn-52. The plan retains the byte-identical ExperimentSpec/model identities and is the only semantic input accepted by the canary.

**Size:** M
**Files:** `model/Temporal/System/Execution/ProductionCanary.lean`, `model/Temporal/System/Execution/ProductionCanaryTests.lean`, `model/Temporal/Feature/Nexus/Execution.lean`, `model/Temporal/Feature/Nexus/ExecutionTests.lean`, `model/Temporal/Tool/PortableEvaluationContract.lean`, `tools/canary/testdata/caller-closure-canary-plan/**`
**Touches:** [model/Temporal/System/Execution/ProductionCanary.lean, model/Temporal/System/Execution/ProductionCanaryTests.lean, model/Temporal/Feature/Nexus/Execution.lean, model/Temporal/Feature/Nexus/ExecutionTests.lean, model/Temporal/Tool/PortableEvaluationContract.lean, tools/canary/testdata/caller-closure-canary-plan/**]

### Approach
- Define the exact production-canary runtime profile with explicit empty fault/traffic/deployment/configuration capabilities and the existing bounded phase/action/evidence Limits.
- Use fn-52's Lean compiler to emit one typed PortableTestPlan containing the retained ExperimentSpec binding, no-fault runtime program, public Observation mapping, Implementation Link, Properties, closure rules, and required/advisory external obligations.
- Pin the plan checksum and model-compiled provenance expected by the protected canary; do not add a canary-specific plan vocabulary or caller override.
- Derive semantic coordinates only from admitted participant/history/control/cleanup facts. Isolation and authority facts remain Claim Assessment inputs, never semantic Evidence.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-52-caller-neutral-grpc-portable-test-plans.md` — typed plan and authority contract
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — canonical Run Evaluation semantics
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime phases and Evidence closure
- `model/Temporal/Tool/PortableEvaluationContract.lean` — existing checked-Test lowering
- `model/Temporal/Feature/Nexus/CallerClosure.lean` — unchanged model target and kernel

### Key context
The fn-29 `production-canary-public-grpc` name describes downstream Temporal access. The caller-to-executor ingress is the distinct fn-52 UmpireExecutor gRPC interface. The protected canary accepts model-compiled provenance only; it never downgrades to plan-local authority.
## Acceptance
- [ ] The pinned canary PortableTestPlan retains the exact ExperimentSpec identity and contains the complete no-fault execution and fixed public verification program.
- [ ] Model provenance, plan checksum, runtime profile, mapping, Property, Limit, Known Gap, and obligation bindings are deterministic and mutation-tested.
- [ ] External/unvalidated plans, arbitrary target/action input, internal/payload Evidence, unknown, unsupported, ambiguity, conflict, or unresolved required obligations cannot qualify.
- [ ] Local, CI, staging, and fn-28 compatibility fixtures remain unchanged.
- [ ] R2/R6 focused model, plan-identity, mapping, mutation, and sibling regression suites pass.
- [ ] Existing semantic comments are preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

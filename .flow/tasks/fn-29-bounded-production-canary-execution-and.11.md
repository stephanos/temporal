---
satisfies: [R3, R4, R5, R7, R8, R9]
---
# fn-29-bounded-production-canary-execution-and.11 Run adversarial authority, containment, and security matrices

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, conformance, and qualification interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Expand R3-R5/R7-R9's harness into bounded adversarial matrices for the production authority, target, lease, lifecycle, recovery, secrets, and forbidden capabilities.

**Size:** M
**Files:** `.github/workflows/umpire-production-canary-qualification.yml`, `tools/umpire/temporal/canary/**`, `tools/umpire/remotequalification/**`, `tools/umpire/canaryqualification/**`, `tools/umpire/cmd/umpire-qualify-production-canary/**`
**Touches:** [.github/workflows/umpire-production-canary-qualification.yml, tools/umpire/temporal/canary/**, tools/umpire/remotequalification/**, tools/umpire/canaryqualification/**, tools/umpire/cmd/umpire-qualify-production-canary/**]

### Approach
- Mutate authority closure, credential/ref/SHA context, TLS target/routing, isolation assertion, run-owned identity collision, lease reuse/conflict/fence, ambiguous starts, redelivery/idempotency, cancellation, target drift, cleanup/recovery state, and progress limits with independent expected outcomes.
- Test controller RPC partitions at N/N+1, prove the 24-call cleanup/reconcile reserve cannot be borrowed or reset through RemoteRecoveryRecord v2, and prove staging v1 stays unchanged. Separately test idle polling, response retry, redelivery, cancellation, and timeout using one-slot `WorkflowTaskPollerBehavior`/`NexusTaskPollerBehavior`, a one-slot WorkerTuner, `LocalActivityWorkerOnly:true`, zero legacy concurrency fields, and no activity registrations.
- Perform race, bounded fuzz, secret/redaction, path/permission, and capability-surface scans; prove no reachable API constructs customer traffic, deployment/configuration, namespace/endpoint/task-queue, fault, release-output, or unrelated-resource mutation.
- Verify the repository-controlled credential-free default-ref guard and exact-SHA checkout. Treat the external protected-environment branch restriction as a runbook/provisioning assertion that tests cannot authenticate from repository bytes.
- Verify synthetic mode cannot write a configured production destination or retain an accepted result, while explicitly proving that receipt schema validation alone does not authenticate origin.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.3.md` — authority/scope failures
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.4.md` — reuse, fence, budget, and cleanup contract
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.9.md` — recovery/progress/trusted-ref workflow contract
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.10.md` — controlled harness and terminal fixtures
- `tools/common/artifactio/set_test.go` — failure/recovery mutation pattern

### Key context
This task owns operational and security adversaries, not exhaustive artifact-version compatibility or repository-wide aggregate gates.

### Acceptance
- [ ] Every authority/ref/SHA/scope/lease/fence/lifecycle/recovery/progress mutation has one deterministic no-side-effect or non-success outcome.
- [ ] Actual controller RPC accounting and worker transport bounds are tested under idle, redelivery, cancellation, ambiguity, cleanup, and N/N+1 cases.
- [ ] Worker startup does not panic, regular/sticky workflow plus Nexus tasks run with exact pinned SDK options, and the harness observes no activity poll or activity-task response.
- [ ] Race/fuzz/secret/capability scans prove sensitive values and forbidden mutation surfaces cannot cross the closed interfaces.
- [ ] Workflow tests prove repository guards honestly and docs retain the unverifiable external-environment prerequisite.
## Acceptance
- [ ] R3-R5/R7-R9 adversarial authority, containment, and security verification is complete.
- [ ] Focused race, bounded-fuzz, secret, permission, capability, and workflow-policy suites pass.
- [ ] Existing security and lifecycle comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

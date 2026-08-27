---
satisfies: [R9]
---
# fn-29-bounded-production-canary-execution-and.13 Publish the canary runbook and synchronize the component roadmap

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Finish R9 by documenting the implemented protected production-canary boundary and updating C12 only to the truth proved by Task `.12`.

**Size:** S
**Files:** `docs/admin/umpire-production-canary-claim-assessment.md`, `docs/README.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [docs/admin/umpire-production-canary-claim-assessment.md, docs/README.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Add one operator runbook covering protected-environment provisioning, required reviewers, the external deployment-branch rule restricted to the protected default branch, non-secret required variable names, invocation, fixed scope/limits, monitoring, abort, cleanup/reconciliation, runner-loss backstop, result/status interpretation, retention/redaction, immutable-output recovery, and incident escalation.
- Require operators to verify the external branch rule before secrets are provisioned or a run is approved; distinguish that external prerequisite from repository-tested ref/SHA guards.
- State that the canary never touches customer traffic or deployment/configuration, performs no customer rollback, trusts a protected ownership/isolation assertion without independent audit, and never produces release eligibility.
- State that synthetic tests prove protocol behavior only, harnesses refuse retained accepted output, receipt bytes are not self-authenticating, and future consumers require a separately trusted retained-artifact channel.
- Link the guide from the admin index and reconcile C12 state/Active Flow wording only to implemented artifacts; keep release aggregation and broader production automation separate.

### Investigation targets
**Required** (read before coding):
- `docs/README.md` — operator guide index
- `.plans/UMPIRE4_COMPONENTS.md` — C12 doctrine, Active Flow table, and deferred canary item
- `.flow/specs/fn-29-bounded-production-canary-execution-and.md` — profile, trust, status, and non-goals
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.9.md` — trusted-ref/recovery/progress operations
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.12.md` — final proof scope

### Key context
Do not edit `.plans/UMPIRE4_VISION.md`. Do not change generated regression documentation because this slice changes environment Claim Assessment, not semantic catalog input.

### Acceptance
- [ ] An authorized operator can verify trusted-ref policy, provision, invoke, monitor, abort, reconcile, interpret, retain, and escalate without secret values in docs.
- [ ] Docs distinguish accepted/rejected/incomplete/tooling outcomes, trust/authenticity limitations, and every observability/release Known Gap.
- [ ] Roadmap/docs claim no more than implemented/tested scope and keep trusted-channel release aggregation separate.
- [ ] Links resolve, VISION remains untouched, and unrelated docs/comments are preserved.
## Acceptance
- [ ] R9 operator documentation and roadmap synchronization are evidence-backed.
- [ ] Documentation link and scoped wording checks pass.
- [ ] Existing comments and unrelated roadmap content are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

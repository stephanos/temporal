---
satisfies: [R9]
---
# fn-28-authorized-remote-staging-black-box.11 Publish the operator runbook and synchronize the component roadmap

## Description
Finish R9 by documenting the implemented protected-staging boundary and updating C12 only to the truth proved by Task `.10`.

**Size:** S
**Files:** `docs/admin/umpire-remote-staging-claim-assessment.md`, `docs/README.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [docs/admin/umpire-remote-staging-claim-assessment.md, docs/README.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Add one focused operator runbook covering protected-environment provisioning responsibility, required secret names without values, invocation, exact blast radius and limits, progress/terminal records, result interpretation, abort, recovery/reconciliation, runner-loss backstop, escalation, retention, redaction, and immutable-output recovery.
- State explicitly that synthetic tests prove protocol behavior only, accepted staging receipts can come only from the manual protected workflow, and no result from this profile is release-eligible.
- Link the guide from the existing admin documentation index and preserve unrelated documentation.
- Reconcile the C12 current-state text and Active Flow row only to the artifacts and workflow actually implemented; keep canary and release aggregation separate.

### Investigation targets
**Required** (read before coding):
- `docs/README.md:13-20` — operator guide index
- `.plans/UMPIRE4_COMPONENTS.md:495-517` — C12 profile-specific Claim Assessment doctrine
- `.plans/UMPIRE4_COMPONENTS.md:730-742` — deferred canary/general-remote boundary
- `.flow/specs/fn-28-authorized-remote-staging-black-box.md` — exact profile, Known Gaps, statuses, and non-goals
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.9.md` — workflow/recovery/progress operations
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.10.md` — proof scope and synthetic-claim limitation

### Acceptance
- [ ] The runbook lets an authorized operator provision, invoke, monitor, abort, reconcile, interpret, retain, and escalate without exposing secret values.
- [ ] The docs distinguish accepted/rejected/incomplete/tooling outcomes and state every trust/observability/release Known Gap.
- [ ] The roadmap and docs claim no more than the implemented/tested profile and retain canary/release work as separate.
- [ ] Links resolve and unrelated docs/comments remain unchanged.

## Acceptance
- [ ] R9 operator documentation and roadmap synchronization are complete and evidence-backed.
- [ ] Documentation link checks and scoped wording checks pass.
- [ ] Existing comments and unrelated roadmap content are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

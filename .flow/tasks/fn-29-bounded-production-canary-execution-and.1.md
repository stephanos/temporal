---
satisfies: [R1, R10]
---

# fn-29-bounded-production-canary-execution-and.1 Define the canary Assessment Profile outside Umpire
## Description
Define the domain-neutral assessment vocabulary needed by fn-26 and one canary-owned fixed policy under `tools/canary`. Bind environment, authority, isolation, evidence, cleanup, trust, Limits, Known Gaps, claim strength, and structural `releaseEligibility:false` without adding canary policy to Umpire.

**Size:** M
**Touches:** `tools/canary/assessment/profile.go`, `tools/canary/assessment/profile_test.go`, `model/Umpire/Evaluation.lean`

## Acceptance
- [ ] Unknown, duplicate, contradictory, broadened, secret-bearing, or N+1 policy rejects.
- [ ] Reusable Umpire types contain no Temporal target, credential, lease, workflow, or canary authority.
- [ ] Package/import tests prove Umpire does not import `tools/canary`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

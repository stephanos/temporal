---
satisfies: [R2, R3, R4, R7]
---

# fn-26-local-qualification-receipts-and-staged.3 Implement offline local Claim Assessment

## Description
Add `tools/umpire/evaluation/assess.go`: `Assess(subject *Subject, profile Profile) Decision`, pure: evaluate every condition the Profile's reason table names against the subject's recorded fields -- disposition, cleanup, Verdict status, the Known Gap kinds present, each rule's support, the recorded Profile name when the Profile requires it -- accumulate every reason that holds in the table's order, and decide `rejected` if any rejecting reason holds, else `incomplete` if any incomplete reason holds, else `accepted`. A satisfied Verdict alone never accepts: an unclosed cleanup, a blocking Known Gap, an unsupported rule or an unmet requirement keeps the decision below `accepted`. The `Decision` keeps disposition, Verdict status, cleanup, Known Gaps and trust as their own fields beside the decision and its reasons, never folded into it. No Driver, deployment, preparation, Run or Contract evaluation; the same subject and Profile give the same Decision.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/evaluation/ -run Assess`

**Size:** M
**Files:** `tools/umpire/evaluation/assess.go`, `tools/umpire/evaluation/assess_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Accepted, rejected, and incomplete decisions accumulate all reasons deterministically.
- [ ] Cancellation, protocol failure, missing evidence, status drift, and N+1 never produce accepted.
- [ ] Repeated or different-Profile assessments create no Run and preserve the source subject.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

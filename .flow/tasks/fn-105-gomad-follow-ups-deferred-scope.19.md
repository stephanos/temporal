---
satisfies: [R19]
---
# fn-105-gomad-follow-ups-deferred-scope.19 D19: investigate activity fairness backlog readiness

## Description
Covered by the 2026-09-30 blanket investigation approval. TestFairnessSuite/Test_Activity_Basic and TestFairnessAutoEnableSuite/Test_Activity_Basic measure dispatch fairness before all workflow activity transfers are known to have reached matching. The recorded virtual-time runs initially dispatch from only 3-4 of 15 workflows. Establish whether the test setup needs an explicit backlog-readiness condition or whether there is a product fairness defect, then propose the correction.

## Acceptance
- Compare native and Gomad runs on seeds 11 and 17, retaining task-transfer, queue-readiness, dispatch, and unfairness evidence for both suites.
- Establish the intended fairness contract and whether the complete backlog is actually eligible before measurement starts.
- Evaluate a bounded readiness condition using existing queue/testcore observations; distinguish test setup bias from a matcher fairness defect.
- Record the cause, correction owner, proposed next action, and qualification criteria in fn-105 for a subsequent decision; retain any needed fix as explicit open work.
- Preserve the fairness assertion and distribution, keep the skips until verification supports removal, and do not substitute a relaxed threshold or Gomad-only source rewrite for evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

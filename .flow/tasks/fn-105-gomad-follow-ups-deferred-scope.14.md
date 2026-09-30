---
satisfies: [R1, R2]
---
# fn-105-gomad-follow-ups-deferred-scope.14 D14: darwin chasm replay residual

## Description
Origin: F7 (2026-09-29). TestSignalWorkflowTestSuiteChasm is intermittent on darwin: about one seed-11 replay in 28 differs in stderr because two heap-span refills (size classes 11 and 50) swap order at decision 46 of cluster start; the allocating goroutine is not identified. Likely the same host-timing class as D12. Revive with D12, or when the rate rises.

## Acceptance
- the allocating goroutine is identified and the channel fixed or recorded; the suite qualifies on darwin


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

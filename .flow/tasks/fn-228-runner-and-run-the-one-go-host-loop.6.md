---
satisfies: [R6]
---
# fn-228-runner-and-run-the-one-go-host-loop.6 After .4: delete verification, evaluation, cel, predicate, contract, casefile, campaign, replay; shrink ir and execution; sizes reported

## Description
Remove internal/verification, evaluation, cel, predicate, the IR CEL bridge and cel-go, contract, casefile, campaign and replay; shrink internal/ir to instruction decoding and internal/execution to act-and-observe; delete generated Case trees and the seven legacy failing tests, replacing their coverage with Runner questions over the activity Model. Umbrella green. Report non-test lines per package before and after, excluding generated v1; targets at most 5,000 generic and 11,000 total, else owner exception in writing. May run in parallel with .5 on disjoint packages.

## Acceptance
- [ ] Listed packages gone; ir and execution shrunk; cel-go out of go.mod
- [ ] Case trees and seven legacy failures removed; umbrella green
- [ ] Per-package counts reported; targets met or exception recorded

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

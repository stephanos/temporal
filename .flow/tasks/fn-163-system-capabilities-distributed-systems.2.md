---
satisfies: [R1, R3, R4, R5]
---
# fn-163-system-capabilities-distributed-systems.2 Inventory and named system capabilities: Exclusive, Fenced, Idempotent, SingleOutcome

## Description
Inventory the hand-written distributed-systems claims and give each pattern one named definition in capabilities/system: Exclusive (atMostOneActive Property, atMostOneActiveAttempt monitor), Fenced (staleDeliveryRejected), Idempotent (action-targeted: a repeated request answers as the first; closedRejectsOrRepeats, repeatedRequestsAnswer), SingleOutcome (singleOutcome monitor). Laws lower to the existing Property and Monitor IR; Models call them instead of hand-writing.

## Acceptance
- Inventory recorded in the spec (R1)
- Each pattern is one named definition, used by the Models (R3)
- Laws lower to existing IR; history laws stay monitors, with the reason recorded (R4)
- Step tables unchanged; IR differences only function names (R5)

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

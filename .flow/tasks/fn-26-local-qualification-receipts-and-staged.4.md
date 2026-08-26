---
satisfies: [R2, R3, R4, R6]
---
# fn-26-local-qualification-receipts-and-staged.4 Implement pilot-gated offline local qualification

## Description
**Size:** L
**Files:** tools/umpire/qualification/**, model/Temporal/Tool/QualificationProfile.lean
**Touches:** pilot admission and offline qualification controller

Implement tools/umpire/qualification QualifyLocal. Strictly read fn-14 evidence through its public reader, enforce the fixed 10-second/1-MiB/zero-stderr sibling local-profile protocol and kill/reap rules, apply every condition in the exact accumulating pilot/status/phase/source/cleanup table, construct the receipt and admitted v2 set in memory, and preserve explicit absent formal evidence and canonical omissions. No caller can supply a decision/profile or invoke execution/conformance.

## Acceptance
Only recomputed LEAN_FIRST_GO plus the exact all-green local set yields accepted with reasons [accepted]. Valid facade/no-go/inconclusive and every single/compound operational/evidence/semantic/phase/source/cleanup row yield the complete sorted reason union and specified rejected-over-incomplete precedence; malformed pilot/source/profile/protocol data yields no receipt. Controller tests prove bounds/cancellation/reaping and no network, Temporal, raw-fact interpretation, Property evaluation, or alternate profile path.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

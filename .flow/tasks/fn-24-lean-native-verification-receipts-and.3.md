---
satisfies: [R1, R3]
---
# fn-24-lean-native-verification-receipts-and.3 Gate violated claims through canonical replay

## Description
Implement the generic `Umpire.Formal.Replay` fail-fast gate. Recompute candidate identity and lineage, admit the canonical setup and initial state through the checked target, enforce typed Limits, admit each exact ordered transition result through the target kernel, require the checked Behavior and violating selection reason, then run the existing pure Property evaluator and accept only false. Retain exact clause evaluation, derive the exact replay identity, and construct violated/concrete-replay receipts only by pairing the proof-bearing matched result with its opaque native context so planner diagnostics survive without becoming identity-bearing. Add one Temporal-owned early test file that drives the unchanged caller-closure force-close trace through the generic gate to `property-satisfied`; reusable implementation/tests remain domain-neutral. Preserve comments.

**Size:** M
**Files:** `model/Umpire/Formal/Replay.lean`, `model/Umpire/Formal/Tests/Replay.lean`, `model/Umpire/Formal/Tests/ReplayMutations.lean`, `model/Temporal/Feature/Nexus/CallerClosureReplayTests.lean`
**Touches:** [model/Umpire/Formal/Replay.lean, model/Umpire/Formal/Tests/Replay.lean, model/Umpire/Formal/Tests/ReplayMutations.lean, model/Temporal/Feature/Nexus/CallerClosureReplayTests.lean]

## Acceptance
The unchanged caller-closure force-close trace reaches `property-satisfied` through a Temporal-owned consumer of the generic API without any Temporal branch in reusable Umpire. Each exact replay status and replay-identity preimage is independently covered; target/query/Property/kernel/Limits/setup/initial/action/outcome/state/observation/order/reason/behavior mutations and forged status/identity/context pairings fail closed. Only a matched replay paired with its originating opaque native context exposes the violated-receipt factory, preserves exact diagnostics, and emits exact counterexample/evidence/identity bytes; non-matched results cannot become violated or promotion input.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

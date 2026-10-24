---
satisfies: [R1, R3, R4, R5, R6]
---
# fn-163-system-capabilities-distributed-systems.2 Inventory and named system capabilities: Exclusive, Fenced, Idempotent, SingleOutcome

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

Inventory the hand-written distributed-systems claims and give each pattern one named definition in capabilities/system: Exclusive (atMostOneActive Property, atMostOneActiveAttempt monitor), Fenced (staleDeliveryRejected), Idempotent (action-targeted: a repeated request answers as the first; closedRejectsOrRepeats, repeatedRequestsAnswer), SingleOutcome (singleOutcome monitor). Laws lower to the existing Property and Monitor IR; Models call them instead of hand-writing.

### Observable realization evidence (R6)

Before the Scala source close, inventory which named capability promises are
observable by each existing realization, particularly idempotent repeats and
the by-ID witnesses currently expecting `inconclusive(explanationsDisagree)`.
Compare the actual typed realization observations, paths and Query bindings
with the promise's existing Property; source aliases or a capability name alone
are not evidence. Where the current observation surface already makes the
promise observable, prove its exact lowering and preserve that evidence through
the naming migration. Otherwise record the precise missing observation and
retain the existing expected verdict, with any Go/live evidence work explicitly
deferred rather than claimed complete. Do not silently change an inconclusive
expectation, broaden what the law promises, weaken a guard or manufacture an
observation to earn a passing result. R5's unchanged answers/outcomes and the
original R6 observable-evidence obligation both remain binding.

This is the existing R6 requirement, not a new capability API or permission to
run Go, generate managed artifacts, review or execute live Cases. Preserve the
user-owned SEMANTICS draft when updating final author documentation.
## Acceptance

- Inventory recorded in the spec (R1)
- Each pattern is one named definition, used by the Models (R3)
- Laws lower to existing IR; history laws stay monitors, with the reason recorded (R4)
- Step tables unchanged; IR differences only function names (R5)

- [ ] Every observable named capability promise has an explicit realization-evidence inventory and source/lift proof; by-ID inconclusive expectations are revisited against actual observations without silently changing their verdicts, and missing consumer/live evidence is retained as a named deferred obligation rather than passing credit (R6; R5).

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
---
satisfies: [R21]
---
# fn-105-gomad-follow-ups-deferred-scope.21 D21: investigate host-clock reporting escapes and policy-compatible remedies

## Description
Covered by the 2026-09-30 blanket investigation approval. Existing inventory classifies host-clock escapes including MemStats.LastGC, debug.GCStats, the Prometheus last-GC gauge, FIPS monoTime, execution-tracer snapshots, and Linux syscall.Gettimeofday behind an exact pack gate. LastGC is target-visible reporting state. Assess practical evidence exposure and remedies within the current prohibition on collector/runtime-assembly patches. D11 remains the separate deferred dynamic Linux audit. Investigation does not authorize collector changes or silently accept the limitation.

## Acceptance
- Retain minimal exposure evidence and inventory paths on the qualified platforms, distinguishing target-visible reporting from runtime control flow and gated operations.
- Establish whether the escapes can affect supported target evidence and whether any are related to D12/D14, with evidence rather than a shared-cause assumption.
- Evaluate remedies that preserve collector behavior and the current deterministic boundary; explicitly identify any option that would require a separate patch-policy decision.
- Record the feasibility result, affected claims, correction owner, and proposed fix or documented limitation in fn-105 for a subsequent decision. Retain any selected fix as explicit open work.
- Do not close D12/D14 from classification of a reporting escape, widen generic syscall access, alter collector policy, or treat an unavailable audit as passing evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R15]
---
# fn-105-gomad-follow-ups-deferred-scope.15 D15: support larger choice traces when a workload needs them

## Description
Origin: fn-106.3 and the original D13 capacity proposal. Split from D13 on 2026-09-30 when routine qualification tracing became opt-in. This extension remains deferred. Record a named workload requiring a retained decision tape beyond 64 MiB for debugging, replay verification, exploration, or minimization before implementation. Choose a configurable larger bound or streaming from workload evidence. Completion of the opt-in policy does not implement this extension.

## Acceptance
- Record the named workload and evidence that revives this deferred extension before implementation.
- Support its larger complete choice trace with explicit resource bounds, bound artifact identity, fail-visible overflow, replay validation, and compatibility.
- The named workload retains a complete trace and passes verified choice-tape replay; measure trace/storage costs.
- Verify capacity boundaries, interruption, malformed input, and existing-trace compatibility. An untraced seed rerun cannot satisfy this extension's acceptance.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

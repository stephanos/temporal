---
satisfies: [R1]
---
# fn-112-gomad-determinism-assurance-and-test.14 Preserve parent cancellation classification when an exploration round finishes

## Description
Native Linux CI run37027023410 at67dbe0266 failed TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/cancellation/simulation-exploration: target_supervision/context canceled instead of cancelled/context canceled, with identical committed counts and retained runs. Trace the round-drain select boundary in both exploration strategies, reproduce from the committed source, fix the cause without weakening retention assertions, and restore native CI. Serialize integration with fn114.7 shared Runner work.

## Acceptance
A focused regression demonstrates cancellation classification is independent of whether the final completion or parent context Done channel wins the select. Choice and simulation exploration preserve cancelled/overall_timeout domain classification, retained committed evidence, and ordinary supervisor-error boundaries. Focused tests pass repeatedly and the full host gate plus native Linux CI pass on the repaired revision. No runtime/toolchain identity change is needed for a host-loop correction.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

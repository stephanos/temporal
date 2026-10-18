---
satisfies: [R3]
---
# fn-209-spike-the-scala-checker-as-a-native.3 Scala Native build of the checker with one sample

## Description
**Owner 2026-10-18:** fallback for the live-deployment sidecar only; it no longer gates the go/no-go. Run after .5 and only if the sidecar path is wanted.

---
satisfies: [R3]
---
# fn-209-spike-the-scala-checker-as-a-native.3 Scala Native build of the checker with one sample

### Description
Cross-build the checker and one sample with Scala Native and ScalaPB's native runtime, replacing any reflection or bytecode dependency found by task 1; run verifyAll, compare verdicts, record size, startup and Bank wall time, or record the blocking failure.

**Files:** .flow/tmp/fn209/native/, model/design/native-checker-spike.md.

**Quick commands:** per the recipe recorded in the note

### Acceptance
- Note section with measurements or the blocker.

### Done summary
TBD

### Evidence
- Commits:
- Tests:
- PRs:
## Acceptance
- Note section with measurements or the blocker.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

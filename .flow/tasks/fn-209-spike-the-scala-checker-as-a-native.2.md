---
satisfies: [R2]
---
# fn-209-spike-the-scala-checker-as-a-native.2 GraalVM native-image build of the checker with one sample

## Description
Build the checker plus one sample suite with native-image (pinned through mise), supply reachability metadata where needed, run verifyAll, compare verdicts with the JVM, record binary size, startup and Bank verifyAll wall time, or record the blocking failure.

**Files:** .flow/tmp/fn209/graal/, model/design/native-checker-spike.md.

**Quick commands:** per the recipe recorded in the note

## Acceptance
- Note section with measurements or the blocker.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

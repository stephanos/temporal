---
satisfies: [R1]
---
# fn-200-model-lint-is-native-findings-from-the.1 Static lint kinds over DSL declarations

## Description
Compute unasked Property, unperformed action, unread refinement, unread observation, explicit wait, witness-only Query and unevidenced fact from a checked instance's declarations, with the Go kind names and subject spellings. Define the closed `Kind` enum and the `Finding(kind, owner, subject, position)` shape.

**Files:** new model/framework/lint/Native.scala (or a `framework.lint.native` package), tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/lint --require-tests`

## Acceptance
- Each static kind reproduced on a synthetic Model per kind.
- Kind names equal the Go kind names.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

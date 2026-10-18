---
satisfies: [R1]
---
# fn-204-the-conformance-descriptor-ships-with.1 Scala emits the conformance descriptor from the native graph

## Description
From `Checker.rows` cut at the Query's start under `Limits`, serialize interned states, rows with ordered results, hole marks, per-result Property bits, monitor automata, progress and end marks, fairness classes and evidence roles into `ConformanceDescriptor`; attach it to each Case whose Model declares claims; record sizes for the samples; refuse a slice above the owner-set ceiling with the size.

**Files:** model/framework/export/testpilot/Descriptor.scala, tests, a size receipt in model/design.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/export/testpilot --require-tests`

## Acceptance
- Every sample Case carries a descriptor; sizes recorded.
- Ceiling refusal names the size.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

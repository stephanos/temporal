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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
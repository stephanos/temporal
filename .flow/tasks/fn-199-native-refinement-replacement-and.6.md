---
satisfies: [R7]
---
# fn-199-native-refinement-replacement-and.6 Docs: README, SEMANTICS and the measurement note; rule-disjointness disposition recorded

## Description
State the native refinement, replacement and composed fairness contracts in the framework README and SEMANTICS; remove the "no native check reads" wording; update the measurement note's obligation table to point here; record the owner's rule-disjointness disposition in the spec's Decision Context without implementing it.

**Files:** model/framework/README.md, model/framework/SEMANTICS.md, model/design/scala-checker-measurement.md, .flow/specs/fn-199-*.md.

**Quick commands:** `grep -rn "no native check reads" model/framework` returns nothing.

## Acceptance
- Docs updated; grep clean.
- Decision Context carries a dated disjointness line.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
---
satisfies: [R7]
---
# fn-200-model-lint-is-native-findings-from-the.5 Owner decision on modality tables and H1/H2/H3/H5

## Description
**Owner decision 2026-10-18:** drop the tables and H1, H3, H5. H2 is decided after reviewing the 691 accepted reasons: keep and instrument alone (firing rule or absence per pair) only if accepted H2 findings led to Model fixes; otherwise drop. This task now does the review and records the H2 outcome; no instrumentation unless H2 is kept.

Present the three options (keep over the IR until fn-208; instrument native step bodies for decision tracing; drop the tables) with the current acceptance counts per kind, obtain the owner's decision, record it in the spec's Decision Context, and file the follow-up task if the decision is to instrument. Until decided, the native lint reports these kinds as not computed.

**Files:** .flow/specs/fn-200-*.md, MILESTONES.md.

**Quick commands:** none.
## Acceptance
- Dated decision line in Decision Context.
- Follow-up task filed if required.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
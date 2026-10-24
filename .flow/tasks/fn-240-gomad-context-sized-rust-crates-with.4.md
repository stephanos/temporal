---
satisfies: [R4]
---
# fn-240-gomad-context-sized-rust-crates-with.4 Measure and decide next cuts

## Description
Compare lines per crate, clean+incremental build wall time, public item counts and pub-for-test counts before/after .2 and .3. Ask one worker to do a bounded task in replay/ against gomad-evidence and report whether it opened neighbouring code. Record go/no-go for continuing to the Amendment 4 target layout (tasks .6–.13) rather than choosing candidates; if the clean build is >20% slower, record which fold from the Amendment 4 "Decisions recorded" list widens the DAG before continuing.
## Acceptance
Decision artifact .flow/artifacts/fn-240/decision.md with measurements and a reason per go/no-go; follow-up specs filed for approved cuts or an explicit stop.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

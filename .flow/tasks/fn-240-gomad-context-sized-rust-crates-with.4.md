---
satisfies: [R4]
---
# fn-240-gomad-context-sized-rust-crates-with.4 Measure and decide next cuts

## Description
Compare lines per crate, clean+incremental build wall time, public item counts and pub-for-test counts before/after .2 and .3. Ask one worker to do a bounded task in replay/ against gomad-evidence and report whether it opened neighbouring code. Record go/no-go per remaining candidate (wasm-abi, wasm-engine, wasm-cluster, gomad-campaign, consumer crates, bin glue). Stop further cuts if clean build >20% slower until DAG widened.

## Acceptance
Decision artifact .flow/artifacts/fn-240/decision.md with measurements and a reason per go/no-go; follow-up specs filed for approved cuts or an explicit stop.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

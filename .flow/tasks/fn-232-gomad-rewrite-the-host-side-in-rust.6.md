---
satisfies: [R5]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.6 replay in Rust (exact and observed) on the helper library tape path

Touches: tools/gomad/rust/gomad/src/cli/replay.rs, gomad/src/replay/{preflight.rs,divergence.rs}

## Description
Implement preflight (helper, module, runtime and implementation identity checks without reopening build inputs), exact replay through the helper library's fn-228 tape path, observed replay for the stock profile, divergence projection with ordinal, reason, expected and observed decisions, and the storage-crash interrupted path. Add replay to the harness.

## Acceptance
Harness replay lane shows the same match/first-divergence ordinal and reason as Go on its own artifacts across the corpus; identity mismatch, missing payloads and divergence produce today's statuses and messages.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R6]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.7 seed explore, resume and merge in Rust

Touches: tools/gomad/rust/gomad/src/cli/{explore.rs,resume.rs,merge.rs}, gomad/src/campaign/{seeds.rs,ledger.rs,retention.rs,failure_policy.rs}

## Description
Port campaign control for seed strategy: deterministic seed selection order, slots, failure policy, retention of failures and selected successes, hash-linked round ledgers, resume from last complete round with changed-bound and identity refusals, merge of shards. Go-lineage campaign roots refused. Add to the harness.

## Acceptance
Harness explore lane matches selection order, outcome classes and failure grouping; interrupted campaign resumes from the last complete round; changed bounds refused; merge of two shards equals one unsharded run's ledger.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

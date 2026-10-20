---
satisfies: [R6, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.7 seed explore, resume and merge in Rust

Touches: tools/gomad/rust/gomad/src/cli/{explore.rs,resume.rs,merge.rs}, gomad/src/campaign/{seeds.rs,ledger.rs,retention.rs,failure_policy.rs}

## Description
Port campaign control and `execute-shard` for seed strategy: deterministic seed selection order, slots, failure policy, retention of failures and selected successes, hash-linked round ledgers, resume from last complete round with changed-bound and identity refusals, merge of shards. Go-lineage campaign roots refused. Add to the harness.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness explore lane matches selection order, outcome classes and failure grouping; interrupted campaign resumes from the last complete round; changed bounds refused; execute-shard and merge of two shards equal one unsharded run's ledger; all three public lanes plus resume have parity and golden fixtures.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

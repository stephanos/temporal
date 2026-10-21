---
satisfies: [R6, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.7 seed explore, resume and merge in Rust

**Touches:** tools/gomad/rust/gomad/src/cli/{explore.rs,resume.rs,merge.rs}, tools/gomad/rust/gomad/src/campaign/{seeds.rs,ledger.rs,retention.rs,failure_policy.rs}

## Description
Port campaign control and `execute-shard` for seed strategy: deterministic seed selection order, slots, failure policy, retention of failures and selected successes, hash-linked round ledgers, resume from last complete round with changed-bound and identity refusals, merge of shards. Go-lineage campaign roots refused. Add to the harness.

Receive fn-229 R2's campaign-inspect integration: Go's ExecutionInspection.divergence is projected from the validated execution ledger, not artifact choice metadata. When adding the Rust campaign inspection consumer, preserve the helper-owned typed ordinal/reason/tape_records and optional expected/observed decisions with their raw site keys; reuse fn-229's immutable symbol lookup against each execution's identity-matching authenticated retained module. No external module fallback, text-path parsing or duplicate divergence codec. Coordinate the choice-round producer/fixtures with .8; fn-229.2 stays open until both decisions are covered by public inspect fixtures on actual validated campaign evidence. This receiving work does not waive R6/R13/R14 or the existing campaign/ledger integrity checks.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness explore lane matches selection order, outcome classes and failure grouping; interrupted campaign resumes from the last complete round; changed bounds refused; execute-shard and merge of two shards equal one unsharded run's authenticated semantic ledger view under task .4, not raw diagnostic bytes; independently verify both original ledgers' hash links, record references, order and bounds; all three public lanes plus resume have parity and golden fixtures.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

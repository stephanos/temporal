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
Rust seed explore, resume, execute-shard and merge pass the mandatory campaign lane (fresh frozen-Go reference, Rust public runtime/failure cases, typed scenario cases, interrupted resume, shard/merge) on the fresh candidate-v3 fixture binary 31b3187b. The Rust tree d6b57a08 equals gomad; gomad's later helper change only rotates a test-only network corpus pin. Regeneration1b and regeneration2 (via --verb merge) are byte-identical; strict (via execute-shard) and strict2 (via resume, with the reworked identity fixture) pass and leave goldens unchanged.

Conductor decision (owner-delegated, documented in tools/gomad/docs/CLI.md): frozen Go WASM has no positive plan/execute-shard/merge surface. Its public CLI does not register the verbs (status 2, `unknown gomad command`), and the original Runner API refuses the WASM bundle layout (status 3, `prepared campaign target is outside its private preparation directory`). The reference driver verifies both refusals exactly on every run. The lane compares Rust execute-shard and merge semantic views (ordinal, seed, outcome, termination, failure group) against the fresh unsharded Go public explore view, alongside Rust's own exact equality with the unsharded records. Go explore/resume keep positive parity. No Go policy or preparation owner changed.

Lane corrections: merge of a failing campaign now expects the same status as the other verbs (1 with failures; the first failed attempt is preserved). The changed-identity refusal now rechains a resumable interrupted prefix and requires `campaign implementation identity mismatch`, with an unchanged rechained prefix as the success control. The bounds refusal message is asserted. Golden changes vs HEAD are identity rotation plus fn230's v3->v4 choice-trace profile and creation-site feature keys; seeds, outcomes, virtual time, stdout and transcripts are equal. Independent codex review: NEEDS_WORK (identity fixture masked) -> fixed -> re-review SHIP.

stage: impl-review - ran (independent codex review NEEDS_WORK -> fixed -> re-review SHIP R6/R13/R14)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 92e583b3f0, 5d7c29f10a
- Tests: campaign-regeneration1b exit 0, campaign-regeneration2 exit 0 (byte-identical goldens), campaign-strict exit 0, campaign-strict2 exit 0, independent codex review NEEDS_WORK then re-review SHIP; receipts /home/agent/wt/evidence/fn232-campaign-closeout
- PRs:
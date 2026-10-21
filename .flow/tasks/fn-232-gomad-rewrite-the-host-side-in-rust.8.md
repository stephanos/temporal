---
satisfies: [R7, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.8 choice exploration and simulation exploration in Rust

Touches: gomad/src/campaign/exploration/{choice.rs,simulation.rs,round.rs,plan.rs}, cli/plan.rs

## Description
Port forced-prefix choice exploration (bounded BFS rounds, frontier order, round transactions, explored-envelope report) and combined simulation exploration across runtime, scenario, network, storage, fault and crash dimensions with explicit bounds and stop reasons. Plan file validation refuses malformed plans on open. Add to the harness.

Reuse task .6's typed linked cluster replay/inspection execution seam over the existing helper core, including pinned module bytes, retained scenario and multiple incarnations, rather than duplicating cluster execution or restoring subprocess transport. This task adds exploration control/transactions and preserves that seam's clock/broker/storage/fresh-store semantics and bounds.

Receive fn-229 R2 with .7's campaign inspection consumer: actual choice-round divergence must retain the helper-owned typed ordinal/reason/tape_records and optional expected/observed raw decisions in the authenticated ledger. Wire symbol enrichment through the shared immutable resolver and each execution's identity-matching retained module; do not move divergence into artifact choice metadata or parse standalone replay's textual path. Public inspect fixtures must exercise actual validated campaign divergence, both expected/observed symbols, missing-site behavior and unchanged raw ledger/record identities. Keep fn-229.2/R2 open until this consumer integration is proven; no producer/qualification claim follows from a typed-only fixture.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness lane matches frontier order and envelope reports on the corpus plans; exhausted bounds report the envelope; malformed plan refused on open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

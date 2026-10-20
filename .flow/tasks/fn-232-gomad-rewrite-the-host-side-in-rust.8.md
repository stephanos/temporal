---
satisfies: [R7, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.8 choice exploration and simulation exploration in Rust

Touches: gomad/src/campaign/exploration/{choice.rs,simulation.rs,round.rs,plan.rs}, cli/plan.rs

## Description
Port forced-prefix choice exploration (bounded BFS rounds, frontier order, round transactions, explored-envelope report) and combined simulation exploration across runtime, scenario, network, storage, fault and crash dimensions with explicit bounds and stop reasons. Plan file validation refuses malformed plans on open. Add to the harness.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness lane matches frontier order and envelope reports on the corpus plans; exhausted bounds report the envelope; malformed plan refused on open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

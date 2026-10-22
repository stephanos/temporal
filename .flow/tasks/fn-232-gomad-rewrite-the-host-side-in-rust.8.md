---
satisfies: [R7, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.8 choice exploration and simulation exploration in Rust

**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/linked_cluster.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/**, tools/gomad/testdata/parity/**, tools/gomad/rust/gomad/src/campaign/exploration/{choice.rs,simulation.rs,round.rs,plan.rs}, tools/gomad/rust/gomad/src/cli/plan.rs

## Description
Receiving fn-227 R1/R2: this task owns the Rust implementation and consumption of fn-227.1's varied-insertion-order corpus. At an admitted live-session shutdown boundary, stop remaining incarnations in exact (ID, Incarnation) order and retain matching World/network evidence; the current linked_cluster incomplete-lifecycle abort is not a substitute for this explicit cleanup path. Preserve refusal of genuinely malformed/incomplete scenario plans and keep cleanup distinct from declaring those plans successful. Validate storage-crash context and selection volumes in sorted volume order before execution so multiple invalid entries choose the same refusal and message. Extend the linked core only as required to realize these existing Go-reference semantics; the prior instruction to preserve the linked seam does not waive this receiving obligation.

Port forced-prefix choice exploration (bounded BFS rounds, frontier order, round transactions, explored-envelope report) and combined simulation exploration across runtime, scenario, network, storage, fault and crash dimensions with explicit bounds and stop reasons. Plan file validation refuses malformed plans on open. Add to the harness.

Reuse task .6's typed linked cluster replay/inspection execution seam over the existing helper core, including pinned module bytes, retained scenario and multiple incarnations, rather than duplicating cluster execution or restoring subprocess transport. This task adds exploration control/transactions and preserves that seam's clock/broker/storage/fresh-store semantics and bounds.

Receive fn-229 R2 with .7's campaign inspection consumer: actual choice-round divergence must retain the helper-owned typed ordinal/reason/tape_records and optional expected/observed raw decisions in the authenticated ledger. Wire symbol enrichment through the shared immutable resolver and each execution's identity-matching retained module; do not move divergence into artifact choice metadata or parse standalone replay's textual path. Public inspect fixtures must exercise actual validated campaign divergence, both expected/observed symbols, missing-site behavior and unchanged raw ledger/record identities. Keep fn-229.2/R2 open until this consumer integration is proven; no producer/qualification claim follows from a typed-only fixture.

## Acceptance
- [x] Consume fn-227's multiple-live-node and multiple-invalid-volume cases as Rust behavioral/golden fixtures. Varied insertion order preserves exact stop order, World/network evidence, refusal class and message; malformed/incomplete scenarios still refuse.
- [x] Retain linked shutdown/cleanup proof and deterministic pre-execution crash-selection refusal evidence, then hand actual implementation/model identities to fn-232.13 final qualification and reconcile fn-227.1. A source-only inventory update or existing incomplete-lifecycle abort cannot close this receiving acceptance.
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness lane matches frontier order and envelope reports on the corpus plans; exhausted bounds report the envelope; malformed plan refused on open.

## Done summary
Choice and simulation exploration in Rust, including fn-227 R1/R2 ordering. Candidate-v6 exploration lane: two identical regenerations plus strict. Codex review SHIP. Identities handed to .13; fn-227.1 note appended. Merged f16f53e59d.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c04d3a4ba4, c1c3383b74, dda8a82900, 487051cd96, f16f53e59d
- Tests: six-lane refresh v6 (2 regen + strict each), make validate, codex re-review SHIP
- PRs:
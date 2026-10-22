---
satisfies: [R2]
---
# fn-238-shared-front-end-one-model-reading-for.9 Derive the IR token printer instead of hand-writing it

## Description
Owner approved 2026-10-22 ("yes"). `dsl/macros/src/lifted.rs` hand-writes the printer that turns the admitted `umpire-ir` tree back into Rust tokens for the embedded Model (277 lines: one `record!` line per struct, explicit arms per enum variant). Every new IR node or field needs a matching printer edit, about 5-15 lines per construct, and body-level and Quint-parity constructs add many nodes.

Add a small proc-macro crate (about 100-150 lines, e.g. `umpire-ir-derive`) with a `#[derive]` that emits the token-printing impl for structs and enums in `umpire-ir`, and delete the hand-written printer. fn-238.4's reviewer estimated about -190 net lines. No third-party dependency beyond what the workspace already uses (`syn`, `quote`, `proc-macro2`). R30's rule that proc-macro crates stay small applies.

Schedule after fn-227 closes or before the next batch of new IR constructs.

### Verification
All goldens byte-identical, including expansion snapshots, and the fn-238 IR dump unchanged. The new crate enters `umpire-dsl`'s lockfile closure, so refresh the guest once in a separate commit and run slice and gaps. Quick then Full gates; report line deltas.

## Acceptance
- [ ] IR token printing is derived; the hand-written printer is gone; adding an IR field needs no printer edit (control).
- [ ] All goldens and the IR dump unchanged; guest refreshed once with slice/gaps green.
- [ ] Line deltas reported (target about -190 net).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R1]
---
# fn-254-gomad-deeper-rust-host-testing-mutation.1 Survey targets for each deep-testing tool

## Description
R1: a read-only survey that produces `.flow/artifacts/fn-254/survey.md`. For each tool, it names the target modules, the reason, and what the tool can't reach. Starting points from the 2026-10-25 grep:
- **Threading (shuttle and loom candidates):** `wasmhost/src/linked*.rs`, `linked_cluster`, `linked_cancellation`, `linked_conditions`, `event_order`, `diagnostic`, `guest_profile`, `cluster/handoff/broker`, `gomad/src/backstop.rs`, `slots.rs`, `cycle/process/{registry,domain}`, `campaign/launcher`, `measurement/cancellation`, `stress/process`. Split them into shuttle-sized pieces and loom-sized primitives.
- **Decoders (fuzz and proptest):** the ≈110 decode/parse/from_bytes sites. Prioritise the formats that are retained or replayed (evidence records, wire codecs, transcript/tape, deflate world, the receipts the cycle verb reads).
- **`unsafe` (Miri):** 48 sites. Mark which ones are reachable without wasmtime JIT execution.
- **Mutation scope:** which packages and paths cargo-mutants covers, and which it excludes (generated codecs, test fixtures). Include the expected runtime of a full run against the scheduled-tier budget.
The survey also records where it overlaps the fn-240 crate cuts, so seams land with the cut instead of being refactored twice.

## Acceptance
- `survey.md` names target modules, a reason and blind spots for each of the seven tools, plus the fn-240 overlap; tasks .2–.6 cite it.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

---
satisfies: [R9, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.10 qualify in Rust

Touches: gomad/src/qualify/{cells.rs,report.rs}, cli/qualify.rs

## Description
Port `qualify`, `qualify-set`, `merge-set` and `compare-support`, preserving current WASM supported/refused cases: repeated seeds and fresh instances per cell, replay of every retained execution, report bound to source, helper, module, model, compiler, host, profile and limits in the new schema, for the fixture cell and the SQLite workload first, then the one-box and cluster cells. Add verdict comparison to the harness.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
All owned public-command lanes have parity and golden fixtures; qualify matches verdicts per cell; a failed repetition or replay fails the cell with an explicit report; reports are retained under the new schema.

## Done summary
Not completed; returned to todo:
Provisional qualification admission is withdrawn under the user's closure-first request. Its fn232.9 predecessor has not completed acceptance. The actual WASM dispatch uses qualify --spec FILE, not the dormant generic parser; preparatory generic Rust code was removed before integration. No qualification implementation or positive acceptance remains in flight. Checkpoint 280ff70271 retains only actual live-grammar RED fixtures and contract notes. Return this task to todo after this dependency block; this is scheduling, not completion or a scope waiver. Resume the full Spec/Freeze/Run/cost/Result/receipt port after the current closure batch.
## Evidence
- Commits:
- Tests:
- PRs:

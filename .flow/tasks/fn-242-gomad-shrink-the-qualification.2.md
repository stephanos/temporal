---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.2 Rust cycle verb: stage DAG, resource scheduler, serial and parallel dispatch

## Description
Add a gomad verb (name decided here, documented in CLI.md) that loads the qualification config and its sha256, builds the 96-stage DAG and dispatches stage processes through .1's explicitly mapped Rust orchestration endpoints, preserving every unchanged leaf-command argv byte-for-byte, in serial (predecessor edge between rows) and parallel mode with the original resource scheduler: core and memory caps, locks, scratch declarations, the 8 GiB live-memory guard, heavy/budgeted isolation, exclusive timing stages, twelve-worker busy-boundary groups reaped before return, and the configured 3,600-second timing policy. Preserve the current stop_at_deadline=False behavior: continue execution and evidence collection through final validation, report elapsed time, and return deferred-perf success only when every correctness/resource/replay requirement passes. Actual stage or infrastructure failures retain their original nonzero classifications; a timing-only overrun alone is not an infrastructure failure. Retain command exits/signals and candidate identities per stage. Golden: compare all unchanged leaf-command argv, bounds, resources and isolation exactly with .1's preserved original table, and validate each orchestration replacement against its explicit original-to-Rust mapping; reject an undeclared substitution or changed leaf argument; mutation fixtures for each scheduler bound. No Python.

Preserve the current global allowance of TWO new full dispatches, both unused at transfer, across candidates and restarts. Catch three policy failures identified by .1. (1) Model child creation followed by KeyboardInterrupt or an otherwise ambiguous spawn outcome; retain the reservation/attempt charge until child absence is proved. (2) After initial enrollment, refuse missing/mismatched fixed journal bindings even when retained prior started outputs exist; never create a replacement history that restores the allowance. (3) Recheck45GiB HOST/20GiB overlay/12GiB inner admission immediately before reservation, including a fixture where space drops after preparation. Use deterministic process/resource seams for these fixtures, not actual full qualification. Historical reference is launch-current-source-v1.py:342/:98/:973 under the fn232-split-closure evidence root. No old Python repair belongs here. After fn-232.30, this implementation/cutover precedes all fn-240 extractions; actual final qualification belongs to fn-246.

Consume every mandatory retained-unproved pending_regression contract assigned to this task by .1. Implement and actually execute its catching fixture or controlled mutation and valid control against the Rust check, preserving the original concrete failure and trust boundary. Retain the inventory IDs, source bindings, exact commands, observed outcomes and numeric exits for .7's proof audit. A named future test, an assertion in source or an unexecuted fixture does not resolve proof. A shared fixture may cover multiple rows only when the evidence maps its catching outcome to each original failure/trust boundary. This task cannot claim its assigned proof complete while any assigned row is unresolved; split oversized surfaces through Flow before implementation and make cutover wait for every proof-producing successor. No full qualification dispatch belongs to this proof work.

**Touches:** tools/gomad/rust/gomad/src/cycle/**, tools/gomad/rust/gomad/src/cli/**, tools/gomad/rust/gomad/src/main.rs, tools/gomad/rust/gomad/src/lib.rs, tools/gomad/CLI.md

## Acceptance
- [ ] Every assigned pending_regression row has actual executed catching/control proof with inventory IDs, commands, outcomes, numeric exits and current Rust source bindings; future test names and unexecuted fixtures supply no proof
- [ ] Dry-run plan preserves all 96 stage contracts, unchanged leaf argv, bounds/resources/isolation exactly; every orchestration substitution matches .1's ledger and an undeclared substitution or altered leaf argument fails a catching fixture
- [ ] Serial and parallel modes dispatch the same stage processes; parallel honors caps, locks, scratch, live-memory guard and exclusive stages, each with a mutation fixture
- [ ] A deterministic overrun regression preserves stop_at_deadline=False, complete final-reader evidence, elapsed-time reporting and deferred-perf success for an otherwise passing run; actual correctness/resource/replay/infrastructure failures retain their original exits
- [ ] Per-stage exits/signals and identities retained; no Python added
- [ ] Deterministic regressions catch ambiguous spawn charge loss, journal re-enrollment/reset and omission of any45GiB HOST/20GiB overlay/12GiB inner reservation-time floor; no full attempt is dispatched by these tests

## Done summary
Blocked:
The design-only worker returned with no product changes and no live commands. The approved decomposition requires .11–.15 and genuine endpoint integration .52 before this original acceptance aggregate can execute. Retain every original acceptance/proof requirement; no completion or failure credit. Resume this aggregate after its actual prerequisites pass.
## Evidence
- Commits:
- Tests:
- PRs:


## Owner-authorized batch B15

Implementation and independent review run under fn-242-gomad-shrink-the-qualification.51 with the complete batch scope in `.flow/artifacts/fn242-execution-batches.json`. This original task retains all acceptance and proof ownership. Its leader dependency prevents separate worker admission. After whole-batch SHIP, start and complete this receipt-only member in original topological order using the actual reviewed source, shared review receipt and its own executed proof.

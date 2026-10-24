---
satisfies: [R1, R2, R5]
---
# fn-242-gomad-shrink-the-qualification.7 Cutover: delete the Python driver, repoint Make, CI, docs and the determinism inventory; record sizes

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
Before deleting any original check or closing cutover, complete the global R1 proof audit against .1's exhaustive source/site inventory. Reconcile every original row and source predicate to actual retained catching execution proof on the current Rust implementation, or a valid fold/deletion reason with actual replacement coverage of the same concrete failure and trust boundary. Earlier Python catching evidence remains provenance and requires proof for its Rust replacement. No pending_regression, unresolved proof, unclassified/dropped site or future named test can count as a proved row. Resolve pending rows only from actual commands, observed mutation/control outcomes, numeric exits and source-bound evidence supplied by responsible port tasks; inspect semantic coverage rather than accepting hashes or rewritten statuses alone. Require zero unresolved pending proof rows and preserve the audit trail of each resolution. Global R1 remains unproved until this audit passes. Task .7 cannot complete, delete unproved checks or authorize .8 completion while any row remains unresolved. This is focused implementation proof; actual full qualification remains fn-246-owned.

After .9/.10 replace composition/preparation and every retained producer endpoint, delete tools/gomad/rust/driver/ and its 71 test files and workload_tools Python that only served it. Repoint Makefile targets (qualify, validate-wasm-determinism driver suite), .github/workflows/gomad.yml source, QUALIFICATION.md one-box gates and ARCHITECTURE.md to the verb. Migrate the determinism inventory driver: evidence references (5 distinct today) to the Rust tests that replaced them so inventory_all_evidence_references_resolve stays green. Run the gomad-source-size/v1 counter before and after against the fn-232.13 ledger row. Before deletion, execute a bounded fresh-input preparation-through-dispatch-and-validation fixture through the supported Rust entry points, including representative genuine producer paths and catching input-authentication/endpoint mutations. Each replacement has executable contract proof; generated bindings must reference surviving endpoints only. This focused proof dispatches no full96 or full production cohort. The actual final-candidate roles, current measurement-role producers, SQLite oracle, runtime, live and R4 cohort remain fn-246.4-owned. Keep approved Go generators, conformance models and the guest toolchain with their original contracts. Merge gate green. Touches: tools/gomad/rust/driver (removed), workload_tools Python that served it (removed), Rust cutover fixture tests, Makefile, .github/workflows/gomad.yml, QUALIFICATION.md, ARCHITECTURE.md, determinism/inventory.json, .flow/artifacts/fn242-size.md, .flow/artifacts/fn242-obligation-inventory proof-resolution/audit append.

## Acceptance
- [ ] Global R1 audit covers every original inventory/source-predicate row with no dropped, unclassified or unresolved pending proof sites; each retained Rust check has actual catching execution evidence, or a valid fold/deletion reason and same-failure/trust-boundary replacement proof
- [ ] Zero pending_regression/unresolved proof rows remain before check deletion or .7/.8 completion; retain row-level commands, control/mutation outcomes, numeric exits, source bindings and resolution provenance, and do not credit future tests or status rewrites as proof
- [ ] Every deleted Python orchestration/preparation/producer endpoint has a mapped executable Rust replacement with catching proof; generated bindings name only surviving endpoints and no Python bridge is needed for the supported cycle
- [ ] Bounded fresh-input preparation, representative genuine production, dispatch and validation pass with retained commands/exits; stale inputs and deleted/unmapped endpoints refuse; no full attempt or expensive final production cohort is launched
- [ ] No Python remains under tools/gomad/rust/ except what another open spec owns
- [ ] make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism exits 0 with no driver: reference unresolved
- [ ] QUALIFICATION.md, ARCHITECTURE.md, CLI.md and CI source name the verb
- [ ] Size table recorded with counter version, baseline and final revisions

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


## Owner-authorized batch B16

Implementation and independent review run under fn-242-gomad-shrink-the-qualification.5 with the complete batch scope in `.flow/artifacts/fn242-execution-batches.json`. This original task retains all acceptance and proof ownership. Its leader dependency prevents separate worker admission. After whole-batch SHIP, start and complete this receipt-only member in original topological order using the actual reviewed source, shared review receipt and its own executed proof.

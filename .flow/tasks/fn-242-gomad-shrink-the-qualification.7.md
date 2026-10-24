---
satisfies: [R2, R5]
---
# fn-242-gomad-shrink-the-qualification.7 Cutover: delete the Python driver, repoint Make, CI, docs and the determinism inventory; record sizes

## Description
After .9/.10 replace composition/preparation and every retained producer endpoint, delete tools/gomad/rust/driver/ and its 71 test files and workload_tools Python that only served it. Repoint Makefile targets (qualify, validate-wasm-determinism driver suite), .github/workflows/gomad.yml source, QUALIFICATION.md one-box gates and ARCHITECTURE.md to the verb. Migrate the determinism inventory driver: evidence references (5 distinct today) to the Rust tests that replaced them so inventory_all_evidence_references_resolve stays green. Run the gomad-source-size/v1 counter before and after against the fn-232.13 ledger row. Before deletion, execute a bounded fresh-input preparation-through-dispatch-and-validation fixture through the supported Rust entry points, including representative genuine producer paths and catching input-authentication/endpoint mutations. Each replacement has executable contract proof; generated bindings must reference surviving endpoints only. This focused proof dispatches no full96 or full production cohort. The actual final-candidate roles, current measurement-role producers, SQLite oracle, runtime, live and R4 cohort remain fn-246.4-owned. Keep approved Go generators, conformance models and the guest toolchain with their original contracts. Merge gate green. Touches: tools/gomad/rust/driver (removed), workload_tools Python that served it (removed), Rust cutover fixture tests, Makefile, .github/workflows/gomad.yml, QUALIFICATION.md, ARCHITECTURE.md, determinism/inventory.json, .flow/artifacts/fn242-size.md.

## Acceptance
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

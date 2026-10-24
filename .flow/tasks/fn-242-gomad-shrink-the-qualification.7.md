---
satisfies: [R2, R5]
---
# fn-242-gomad-shrink-the-qualification.7 Cutover: delete the Python driver, repoint Make, CI, docs and the determinism inventory; record sizes

## Description
Delete tools/gomad/rust/driver/ and its 71 test files and workload_tools Python that only served it. Repoint Makefile targets (qualify, validate-wasm-determinism driver suite), .github/workflows/gomad.yml source, QUALIFICATION.md one-box gates and ARCHITECTURE.md to the verb. Migrate the determinism inventory driver: evidence references (5 distinct today) to the Rust tests that replaced them so inventory_all_evidence_references_resolve stays green. Run the gomad-source-size/v1 counter before and after against the fn-232.13 ledger row. Merge gate green. Touches: tools/gomad/rust/driver (removed), Makefile, .github/workflows/gomad.yml, QUALIFICATION.md, ARCHITECTURE.md, determinism/inventory.json, .flow/artifacts/fn242-size.md.

## Acceptance
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

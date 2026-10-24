---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.3 Port the kept stage readers and receivers into the verb validation step (core, receiving, cells)

## Description
Port the readers the .1 inventory keeps from receiving.py, core_*.py, cell_reader.py, runtime_gate.py, preparation.py and the typed receipt readers into Rust as the verb validation step, each with its mutation fixture from the inventory. Delete nothing here that .1 did not mark for deletion; record each deletion reason in the ledger. Golden/e2e first; a unit test names its invariant. Touches: tools/gomad/rust/gomad/src/cycle/readers/**.

## Acceptance
- [ ] Every kept core/receiving/cell check has a Rust implementation and a fixture that fails when the checked fact is mutated
- [ ] Deleted checks each cite their .1 deletion reason
- [ ] Verb validation step refuses missing, duplicate, skipped or unexecuted items as required-final.json does today

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

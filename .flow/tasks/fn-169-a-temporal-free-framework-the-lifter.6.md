# fn-169-a-temporal-free-framework-the-lifter.6 Port the Python interpreter-table comparator to Scala in framework/check/equivalence; pins to temporal/check

## Description
Owner decision, 2026-10-12: fold the Python interpreter-table comparator in `model/testdata/table-pins/` (fn-156) into the fn-169 layout, so the model tooling is Scala and Go only. Runs after fn-169.1 and before the sweep needs the full interpreter route.

Today: `replay.py` runs the sealed fn-155 Go table dumper over generated IR and compares every record (catalogs, rows, ordered alternatives, outcomes, facts, explanations) with `pins.json` (61 tables, seven activity/Nexus artifacts, 13.9M records); `mutants.py` seeds named mutations to prove the comparator catches them; `test_replay.py` holds its tests. `Gate.scala` calls `python3 replay.py` only on the full equivalence route (`--dumper`, `make umpire-check-equivalence DUMPER=...`).

Do:
1. Port the comparator to Scala in `framework/check/equivalence/`, beside the step-table comparison, sharing its rename-map handling; the gate calls Scala instead of `python3`. Keep every failure class (missing/extra artifacts or subjects, incomplete dumps, catalog/row/result/order changes, changed interpreter scope, input mutation during observation) and the heavy-gates lock.
2. Port the mutation controls and `test_replay.py` into that module's colocated Scala tests.
3. Move `pins.json` and its README to the Temporal side (`temporal/check/`), as the sweep's "before" data for the real features. Delete the Python files.
4. Prove parity: on the fixture streams the Python tests use, and on a mini dump if a dumper is available, the Scala comparator reaches the same verdicts as the Python one did before deletion. A full real-feature replay stays with the sweep.

## Acceptance
- No Python remains under `model/`; the gate's full equivalence route calls the Scala comparator.
- Every mutation control fails as before; the ported tests pass; parity with the Python verdicts is recorded.
- `pins.json` lives under `temporal/check/` with its README.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

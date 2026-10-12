# fn-169-a-temporal-free-framework-the-lifter.5 framework/samples: one-file sample Models (Kettle, Bank, TwoPhaseCommit) comparable with P and Quint

## Description
Owner request, 2026-10-12: `model/framework/samples/` holds one-file sample Models, Temporal-free, over a range of complexity, kept very manageable. Some match well-known P and Quint examples so the three can be compared side by side.

Samples (one Scala file each, in rising complexity):
1. `Kettle.scala`: one machine with phases (off, heating, boiling), rules and one safety Property. Compare with Quint's `examples/tutorials/repl/kettle.qnt`.
2. `Bank.scala`: P's Client Server tutorial. Clients withdraw through a bank server that keeps balances in a database. Safety: the bank always responds with the correct balance. Liveness: every withdrawal gets a response. Compare with P's ClientServer tutorial and the bank in Quint's README.
3. `TwoPhaseCommit.scala`: one coordinator and a bounded number of participants (two or three), composed. Safety: atomicity (no participant commits while another aborts). Include one negative control, for example a coordinator that commits on a partial vote, which the check must refute. Compare with P's TwoPhaseCommit tutorial and Quint's `examples/classic/distributed/TwoPhaseCommit/two_phase_commit.qnt` (adapted from TLA+).

Rules:
- Each sample compiles against `framework` alone (no `temporal.*`, no Temporal API). The samples therefore also prove fn-169's build boundary, and the boundary check covers them.
- Each sample states its counterpart in a short header comment, with the upstream path, and stays small: well under 200 lines for Kettle and Bank, under 300 for 2PC.
- The gate (or fn-166.13's scoped check) compiles, exports and checks each sample with its Queries answered as expected; samples are discovered, never listed.
- Model-only: no realization. fn-169.2 may give `Bank` a tiny proto API of its own as its generic-proto proof.
- `model/README.md` links the samples as the framework's starting point; the mini activity stays the Temporal running example.

## Acceptance
- `model/framework/samples/{Kettle,Bank,TwoPhaseCommit}.scala` exist, each one file, compiling with no Temporal source or API on the classpath.
- Each exports to IR and its Properties/Queries check as expected; the 2PC negative control is refuted.
- Each names its P/Quint counterpart in its header; README links them.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

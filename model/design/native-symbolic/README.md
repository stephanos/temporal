# Native symbolic back end spike

Kettle, read from the compiled World's embedded admitted tree (`umpire_ir::Lifted`, what the
shared front end produces), emitted as a BTOR2 word-level transition system, checked by rIC3
(IC3, k-induction, BMC), every counterexample replayed through the native `checked_step` and the
violated claim re-judged natively. An SMT-LIB 2 bounded check of the same system through a pipe
to z3 is the second data point.

Verdict and measurements: `../native-symbolic-backend.md`.

- `src/btor2.rs` emitter: fields, World constants, first-match branches, one-step history for
  postconditions, checked arithmetic as a latched fault, one `bad` per property.
- `src/witness.rs` BTOR2 witness decoding (state/input declaration indices) and native replay.
- `src/smt.rs` the BTOR2 node graph unrolled as QF_BV SMT-LIB, piped to a solver.
- `src/main.rs` driver: baseline Kettle, two controls that mutate the admitted tree only (the
  cross-check must flag them), and a scaled Kettle (boiling at 1 000 000).
- The baseline includes `umpire/dev/fixtures/fixtures/sample_kettle/model.rs` directly.
  `build.rs` compiles the same authored source with its boiling threshold scaled from 100 to
  1 000 000 for the big Model; it refuses a missing or ambiguous threshold declaration.
- `out/` and `out-big/` are ignored, regenerable reports, emitted `.btor` files, solver logs and
  replayed paths. `run.sh` regenerates the baseline and mutation controls; `RUNS=big OUT=out-big`
  selects the scaled Model. Set `OUT` to retain evidence elsewhere.

Built against `model/umpire/{dsl,ir,checker}` at f76f622a79 with the ric3 binary built from
rIC3 v1.5.2 (see `run.sh`). The crate is standalone (`[workspace]`), not a member of the model workspace; it lives beside its verdict in `model/design/`.

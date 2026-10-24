# fn-242-length-arithmetic-parity-across.1 handoff (lane C)

Branch `work/fn2421-lengths` (worktree `.worktrees/fn2421-lengths`), based on c3c95eb00b.

## Commits
- 7bd76093f9 fix(model): refuse length sums and products; type TLC evaluation errors

## Decision: refuse in the front (the spec's recommendation)
- The front refuses `+` and `*` when the left operand has the index type (`Vector::len()`, indices), at the binary expression's span (`body/expr.rs`). The message is "`+` and `*` on a Vector length are refused: Rust computes lengths in usize and TLC in 32-bit integers, ...". Every syntactic position is covered, including `1 + len()` and indices such as `get(len() + 0)`; compound assignment only reaches fields, which never have the index type.
- Comparison, `-`, `/` and `%` stay admitted. Both operands are in `0..=i32::MAX`, so the result cannot pass TLC's range, and a negative difference fails all three back ends alike (Rust usize underflow panic, interpreter range check, lifter `Assert`).
- Why not bounded emission: it needs length-bound inference, and an out-of-range result would still be a refusal. Refusal costs no admitted Model.
- Interpreter simplification: `checker/src/interpret.rs` drops its `usize` range special case. Lengths now range-check as `0..=i32::MAX`, which is exact for every form that is still admitted. The differential check stays green.

## Acceptance
- [x] R1, refused or agreeing.
  - `usize_lengths::lengths` (`len() * 1_000_000_000`) is refused at `lengths.rs:40:20`. It has no `Interprets` impl and still checks Verified natively. Control: `interpreted::length_sums_and_products_are_refused`.
  - Admitted forms agree across compiled, interpreted and TLC. Control: `interpreted::length_differences_agree_on_every_back_end`.
    - `differences`: `-`, `/` and `%`. Verdicts are halves Verified and below_two Refuted, with 4 states in TLC.
    - `underflow`: `1 - len()` fails compiled and interpreted with "attempt to subtract with overflow". TLC reports it as an `Evaluation` in `Invariant_remaining` at `values = <<1, 1>>`.
  - Lifter-level spans: `lift/tests/vector.rs::length_sums_and_products_refuse_at_their_span` covers 4 refused and 3 admitted forms.
- [x] R2, typed TLC evaluation errors.
  - The runner now parses nested messages; it previously failed with "mismatched TLC message terminator" on them.
  - New `umpire_tlc::Error { Refusal, Evaluation(Box<Evaluation>) }`, returned by `check` and `check_module`.
  - `Evaluation` has these fields:
    - `property`: from 2111/2113, or the run's single property. `None` means the initial predicate or the next-state relation.
    - `message`: the nested cause plus TLC's 2103 expression positions.
    - `trace`: the decoded behavior.
    - `state`: TLC's text for the failing state.
  - An error with neither a property name nor positions is still a `Refusal` ("TLC execution error N").
  - Controls:
    - The unit test `runner::tests::evaluation_errors_name_the_property_and_state` replays recorded TLC output for invariant overflow, Next-relation `Assert`, Init overflow and the unnamed-in-property-run case.
    - The real-TLC control `tests/controls.rs::real_tlc_evaluation_errors_are_typed_and_located` drives TLC into "Overflow when computing 3*1000000000" in the invariant `Wide` at `x = 3`. It also covers a failed `Assert` in Next at `x = 1` and an Init overflow. Evidence is in `.flow/tmp/tlc-controls/evaluation_errors.txt` (worktree).
- [x] Default and required gates pass (below).

## Changed goldens and fixtures
- No golden changed.
- The only Model using the refused form was the fn-238.5 probe `umpire/fixtures/lengths.rs::lengths`. It is kept as the refused control, and the `differences` and `underflow` modules were added.
- Test expectations updated:
  - `lift/tests/narrow_remainder.rs` now expects "TLC evaluation error in the next-state relation" instead of "TLC execution error 2132".
  - `interpreted::lengths_compute_in_usize` was replaced by the two length controls.
  - The TLC unit and control tests now match on `to_string()`.
- Signature ripple: `Scratch::finish` is generic over the error type. Three `process.rs` tests and one Apalache test gained a `::<_, Refusal>` turbofish.
- Docs: model/README.md (Vector subset) and model/DESIGN.md (Vector table and TLC paragraph).

## Gates (worktree, `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2421-target`, each run alone)
- Focused:
  - `cargo test -p umpire-front -p umpire-lift -p umpire-tlc -p umpire-testing -p umpire-fixtures -p umpire-checker` exited 0 (`focused-default.log`).
  - The `umpire-tlc` tests and the length controls also pass with `UMPIRE_BACKENDS=require`.
- `cargo fmt --all --check` and `cargo clippy --workspace --all-targets --all-features -- -D warnings` both exited 0.
- `make model-check` exited 0 in 1352 s (`model-check-default.log`).
- `UMPIRE_BACKENDS=require make model-check` exited 0 in 1700 s (`model-check-require.log`).
  - The first attempt failed on the environment, not the code. Host load average was 67 on 12 cores, and the TLC subprocess hit the 60 s limit in `progress_scope_exit` (`Progress_reaches_target`), which this change does not touch. Log: `model-check-require-1-timeout.log`.
  - The rerun passed with load around 14 to 25.
- Not run, per instructions: model-time, review, flowctl.

## Open items
- An evaluation failure in an initial state carries no state, because TLC prints none; Display says "in an initial state".
- `1 + len()` style author ergonomics are cut. If wanted later, admit sums with a proven length bound, from the authored growth guard, below `i32::MAX`.

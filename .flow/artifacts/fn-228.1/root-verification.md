# fn-228.1 root verification

Current reviewed source: aa0c9d9970, based on f76f622a79. Final required gate remains live and is not claimed here.

Independently read the implementation-review result at `.flow/tmp/fn-228.1/rereview.json`: SHIP, codex gpt-6.1-sol high, with the diagnostic path finding fixed and no unaddressed items.

Independently compared SHA-256 for all 785 entries in `.flow/tmp/fn-228.1/protected-before.json`, mapping only the two renamed crate directories. All bytes match except two explicit permitted changes. For each exception, also loaded the original bytes from f76f622a79 and verified that the sole substitution exactly equals the current file:

- realization macro snapshot: `umpire_wasm` to `umpire_runtime`.
- reference Kettle `omitted_action.stderr`: `../../tests/fixtures/sample_kettle/model.rs` to `../../fixtures/fixtures/sample_kettle/model.rs`.

Independently read live slice terminal PASS and package result (6.242s). Independently counted exactly 2336 final probe PASS entries in live-gaps.log and verified top-level PASS for TestTestpilotModelGaps and TestTestpilotModelGapLoggerPreservesFailuresWithoutPrivateOutput, with package PASS at396.610s. The intermediate RUN-line loss described in live-gaps-observation.txt remains disclosed; final probe PASS evidence is intact.

Neither this report nor the separate native-symbolic design document expands fn-228.1 scope. Full required gate terminal status/timings and Flow done/committed closure still need verification before dependent admission.

Final required timed gate subsequently exited0. Root read the full output and matched every one of the18 named Make stages against its successful timing row. Gate total991.181s exceeds the60s soft target; inner8.588s meets10s. The post-edit default full gate remains pending, so Flow completion is still not claimed here.

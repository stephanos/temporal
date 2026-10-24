---
satisfies: [R4, R5]
---
# fn-248-off-the-shelf-crates-and-consolidation.1 Goldens on insta: hand-rolled comparisons and fixture-runner text snapshots

## Description
Move every hand-rolled golden comparison onto insta and move the fixture runner from binary to text snapshots.

- `model/umpire/reference/kettle/tests/probes.rs` (~93-98): replace the `UMPIRE_UPDATE_KETTLE_COVERAGE` write-then-compare switch with a named insta snapshot.
- `model/umpire/reference/kettle/tests/realization.rs` (~139), `model/temporal/realization/tests/realization.rs` (~377, ~389): replace `assert_eq!` against `include_str!("*.snap.txt")` with named insta snapshots. Grep `model/` for further `include_str!(...snap.txt)` comparisons and `UMPIRE_UPDATE_*` switches and convert them too.
- `model/umpire/dev/testing/src/runner.rs`: `snapshot` uses `assert_snapshot!` (one `<name>.snap` per golden) instead of `assert_binary_snapshot!` (metadata `.snap` + payload `.snap.<ext>`). `logical_file` becomes a `.snap`/`.snap.new` suffix strip; `golden_files` returns the snapshot and its pending file. Consumer ownership, the orphan check and the skipped-backend incomplete/pending checks stay. Regenerate every fixture golden (`Graph@tla.snap.tla`/`Graph@cfg.snap.cfg` pairs become `Graph.tla.snap`/`Graph.cfg.snap` or similar; pick one naming and record it).
- Readers of payload files: `model/umpire/protocol/runtime/tests/dispatch.rs` (~107, ~134), `model/umpire/protocol/runtime/guests/caught-panic/src/bin/panic-control-reports.rs` (~18), `model/temporal/testing/realization_control.rs` (~67), `model/temporal/testing/fixtures.rs` (~64), `model/umpire/dev/fixtures/fixtures.rs` (~55), `model/umpire/dev/testing/tests/{suite_consumers,external_consumers}.rs`, and Go tests reading sibling goldens (e.g. `kettle_annotations_test.go` ~183). They read through one umpire-testing helper that strips the insta header (Go: a small equivalent), or assert their own snapshot.
- Add `cargo insta test --unreferenced=reject` (or the equivalent check) to the model gate.

Do this task first: later tasks re-accept their golden changes through insta.

## Acceptance
- [ ] No test compares goldens by `assert_eq!`/`include_str!` or an env-var update switch.
- [ ] `runner.rs` contains no decoding of insta binary-snapshot file names; every fixture golden is a text snapshot.
- [ ] Payload readers (Rust and Go) use the header-stripping helper or their own snapshot; their tests pass.
- [ ] Orphaned, pending or unowned goldens fail the gate; a skipped backend with an incomplete golden still refuses.
- [ ] `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

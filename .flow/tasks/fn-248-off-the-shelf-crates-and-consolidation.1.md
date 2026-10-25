---
satisfies: [R4]
---
# fn-248-off-the-shelf-crates-and-consolidation.1 Goldens on insta: hand-rolled comparisons and fixture-runner text snapshots

## Description

**Size:** M
**Files:** model/umpire/dev/fixtures/fixtures.rs, model/umpire/dev/testing/**, model/**/tests/**, model/**/expected/**, model/umpire/protocol/runtime/guests/caught-panic/**, model/umpire/reference/kettle/**, model/umpire/hosts/go/**, model/temporal/testing/**, Makefile
**Touches:** [model/umpire/dev/fixtures/fixtures.rs, model/umpire/dev/testing/**, model/**/tests/**, model/**/expected/**, model/umpire/protocol/runtime/guests/caught-panic/**, model/umpire/reference/kettle/**, model/umpire/hosts/go/**, model/temporal/testing/**, Makefile, model/temporal/realization/Cargo.toml, model/temporal/testing/Cargo.toml, model/Cargo.lock, model/README.md, model/design/second-opinion.md, tests/testpilot_model_gaps_test.go]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Move every hand-rolled golden comparison onto insta and move the fixture runner from binary to text snapshots.

- `model/umpire/reference/kettle/tests/probes.rs` (~93-98): replace the `UMPIRE_UPDATE_KETTLE_COVERAGE` write-then-compare switch with a named insta snapshot.
- `model/umpire/reference/kettle/tests/realization.rs` (~139), `model/temporal/realization/tests/realization.rs` (~377, ~389): replace `assert_eq!` against `include_str!("*.snap.txt")` with named insta snapshots. Grep `model/` for further `include_str!(...snap.txt)` comparisons and `UMPIRE_UPDATE_*` switches and convert them too.
- `model/umpire/dev/testing/src/runner.rs`: `snapshot` uses `assert_snapshot!` (one `<name>.snap` per golden) instead of `assert_binary_snapshot!` (metadata `.snap` + payload `.snap.<ext>`). `logical_file` becomes a `.snap`/`.snap.new` suffix strip; `golden_files` returns the snapshot and its pending file. Consumer ownership, the orphan check and the skipped-backend incomplete/pending checks stay. Regenerate every fixture golden (`Graph@tla.snap.tla`/`Graph@cfg.snap.cfg` pairs become `Graph.tla.snap`/`Graph.cfg.snap` or similar; pick one naming and record it).
- Readers of payload files: `model/umpire/protocol/runtime/tests/dispatch.rs` (~107, ~134), `model/umpire/protocol/runtime/guests/caught-panic/src/bin/panic-control-reports.rs` (~18), `model/temporal/testing/realization_control.rs` (~67), `model/temporal/testing/fixtures.rs` (~64), `model/umpire/dev/fixtures/fixtures.rs` (~55), `model/umpire/dev/testing/tests/{suite_consumers,external_consumers}.rs`, and Go tests reading sibling goldens (e.g. `kettle_annotations_test.go` ~183). They read through one umpire-testing helper that strips the insta header (Go: a small equivalent), or assert their own snapshot.
- Extend ownership/pending validation inside the existing test pass; do not add `cargo insta test` or a new mandatory installed CLI. Register current fixture outputs, moved named snapshots and intentional Rust/Go/live/external readers explicitly. Treat source/data reads and skipped-backend completeness as ownership, never execution evidence. Reject orphaned/unowned/pending files without a broad ignore list. The gate must catch an extra text snapshot outside the producer fixture root as well as an unowned fixture file.

Publish the agreed text-snapshot name/payload/ownership interface early; .2–.6 may implement concurrently against current or agreed interfaces in isolated copies. Final reviewed snapshot reacceptance and current guest publication happen after all lanes join, not once per lane.

### Early proof and boundary controls

Prove one ordinary text fixture, one TLA/CFG filename pair, a skipped-backend complete/incomplete/pending trio, an intentional cross-crate/Go reader, and an orphaned text snapshot before mass conversion. Naming is logical `<file>.snap` (or one documented equivalent), with no binary metadata/extension filename decoding. Validate reader payload extraction with body delimiters/blank lines, malformed or missing headers and pending files; do not normalize semantic output to hide changes.

Do not add umpire-testing (which depends on TLC/TLA/native code) as a wasm guest dependency to obtain a tiny reader. Use one std-only shared Rust reader implementation reachable by native readers/control binaries, plus the small Go equivalent; preserve wasm closure isolation. Existing host-only insta APIs may be used where appropriate without an installed cargo-insta requirement.

Keep Go/live producers and their semantic checks independent. Remove ad-hoc Go update environment switches too: choose a narrow update bridge that feeds their produced candidate into the existing Rust insta workflow, while Go remains the live producer/comparator. Record the concrete invocation and ownership; do not copy a Rust-produced expected answer into Go as an independent oracle, or classify an external declaration as a test run. External snapshot declarations need the new layout and pending controls rather than retaining the old blanket rejection of `.snap` names.

## Acceptance
- [ ] Active mutable golden producers use insta instead of hand-rolled update switches/comparisons. Registered Rust/Go native-output readers still compare against snapshot payloads through the helper; historical originals and decoder/fixture inputs are not migrated or re-accepted as current goldens.
- [ ] `runner.rs` contains no decoding of insta binary-snapshot file names; every fixture golden is a text snapshot.
- [ ] Payload readers (Rust and Go) use the header-stripping helper or their own snapshot; their tests pass.
- [ ] Orphaned, pending or unowned current goldens fail inside the existing test pass, including intentional cross-crate/Go reader and non-fixture snapshot ownership controls; a skipped backend with an incomplete golden still refuses.
- [ ] Focused checks below pass; joined full validation belongs to the final integration task.


## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-testing --tests --locked
cd model && mise exec -- cargo test -p umpire-reference-kettle --test probes --test realization --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


Conductor scope clarification: the listed mechanical consumer/manifest/documentation additions are required by the already specified consolidation. Isolated source lanes retain meaningful focused validation; final .7 owns the joined canonical pipeline and necessary current asset/snapshot acceptance before parent closure. Historical/live candidates are not fresh execution evidence.

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
Implemented named insta text snapshots (`<logical filename>.snap`), std-only Rust/Go payload readers, explicit current snapshot ownership and pending/incomplete refusals. Migrated 349 binary pairs and six raw current goldens with their semantic text preserved; historical originals remain untouched.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)
Task status: in_progress; no Flow lifecycle/review commands called by the lane.

Source and evidence are committed in the assigned workspace. Authoritative source-bound commands, exits, logs, scope hashes and limitations: `.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/task1/checks.json`; migration maps and inspected payload hashes are adjacent.

Focused checks are green: required umpire-testing tests and Kettle probes/realization; current snapshot ownership; shared fixtures; Temporal fixtures/boundary; native runtime golden parity; Temporal named report/ordered-row producer; native caught-panic/overflow payload control; Go snapshot unit; affected-crate Clippy with warnings denied; formatting and diff whitespace. The ordinary text-layout regression was observed red on the original runner. The Go pending test was observed red with the pending guard removed, then green with it restored. Incomplete/missing/pending skipped artifacts, external pending/unowned ownership and nonfixture orphan/pending/missing-reader controls run in ordinary tests. No installed cargo-insta CLI is required.

Baseline: pre-edit runner/testing suite green; a separate pre-edit Kettle baseline was not captured. No full Model baseline/final pass was run under the source-wave policy.

Go Kettle and main live-gap reader checks using `-run '^$'` are compile-only, with zero tests executed. No live Go/wasm parity, live gap execution, required TLC/Apalache agreement or guest publication is claimed. Native panic-hook output is intentionally caught (control exit 0). Explicit ignored Go-candidate import controls compile but were not invoked; an import is never execution evidence. Concrete live-export/import commands are documented in the updated READMEs.

Authorized scope additions: model/temporal/realization/Cargo.toml and model/umpire/reference/kettle/Cargo.toml insta dev-dependencies; model/Cargo.lock reconciliation; tests/testpilot_model_gaps_test.go reader; current snapshot workflow sections in model/README.md and current-file links in model/design/second-opinion.md. The already in-scope model/temporal/testing/Cargo.toml adds a native insta dev-dependency and ordinary snapshot_ownership test. Instantiated dependencies stay dev/host-only; native/guest control binaries path-import the std-only Rust reader. The Go equivalent lives in hosts/go/snapshot and uses stdlib only. Coordinate the additive testing/lib.rs reexport with task2 exports at join.

Final7 owns joined snapshot reconciliation/reacceptance, current guest publication/freshness and the single complete required pipeline. No canonical wasm changed. This lane alters snapshot representation and readers only; live producer semantics and payload expectations were preserved. Re-run affected focused controls after central conflict resolution, including the committed-source added-catalog-field mutation control after its coverage checkpoint joins.

Conductor used actual premerge base 95c90138253fd9e7f3d129536e3c56b183fa7a49 and source merge6f147474bf plus evidence merge8fc503884f. Both merges were clean and retain the reviewed generic/prost naming fixes, native WKT mappings and shared helper exports. Joined mandatory checks passed34 runner/testing tests and10 Kettle tests; current snapshot ownership/boundary passed4. Snapshot updates were disabled (INSTA_UPDATE=no); no current payload was reaccepted at this join. The prior B1 compiled/interpreted agreement remains accurately scoped historical joined evidence, not a fresh full gate.

The authentic fresh three-draw readonly review uses codex:gpt-6.1-sol:high, same-family. Raw outputs, current source hashes, command logs and receipt are durable in task1/review. Go compile-only and ignored import observations remain compile-only rather than live independent producer evidence. ParentR5, covering pipeline, canonical guest publication and joined snapshot reacceptance remain deferred to the unstarted final7.

Affected-package Clippy initially failed E0599 because the leased cross-worktree target held stale IR/front check metadata. Current source unconditionally defines children_mut and front reexports it. Only umpire-ir/front package artifacts in the leased private task1 target were invalidated (2,122 files/607.9MiB); the identical Clippy command then exited0 after rebuilding, with no source change. Both logs and cache-repair output are retained. Workspace fmt passed. Reviewers independently checked all355 migrated payload bodies against their predecessors; their Cargo attempts were blocked before execution by readonly build locks.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4b0602c57f393e5f10fa7ca4304694bd21dfcca3, b4193d6f320656cf2c2d1077ce080d15b1b5380d, f44fb2b6bdf4a2160f1f4f69b23396edca20d12d, 98304099d2fcbf707c6864d56f5ff7a3966fcdb0, 6f147474bf43317bf511d3bc157ff32629681b02, 8fc503884f41a87601aebfadd96c59eb6fa74baf
- Tests: cd model && mise exec -- cargo test -p umpire-testing --tests --locked, cd model && mise exec -- cargo test -p umpire-reference-kettle --test probes --test realization --locked, cd model && mise exec -- cargo test -p umpire-temporal-tests --test snapshot_ownership --locked, cd model && mise exec -- cargo test -p umpire-fixtures --test fixtures fixtures --locked, cd model && mise exec -- cargo test -p umpire-temporal-tests --test fixtures --test boundary --locked, cd model && mise exec -- cargo test -p umpire-runtime --test dispatch generated_dispatch_matches_trait_kettle_goldens --locked, cd model && mise exec -- cargo run -p umpire-caught-panic-guest --bin panic-control-reports --locked -- /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-native-controls, cd model/umpire/hosts/go && mise exec -- go test -tags test_dep ./snapshot, cd model/umpire/reference/kettle && mise exec -- go test -tags test_dep -run '^$' ., cd . && mise exec -- go test -tags test_dep,integration ./tests -run '^$', cd model && mise exec -- cargo clippy -p umpire-testing -p umpire-temporal-tests -p umpire-reference-kettle --tests --locked -- -D warnings, cd . && mise exec -- cargo fmt --manifest-path model/Cargo.toml --all --check, cd model && mise exec -- cargo test -p umpire-temporal --test realization named_guest_adapter_preserves_the_independent_report_and_ordered_rows --locked, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-testing --tests --locked, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-reference-kettle --test probes --test realization --locked, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-temporal-tests --test snapshot_ownership --test boundary --locked, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo clippy -p umpire-testing -p umpire-temporal-tests -p umpire-reference-kettle --tests --locked -- -D warnings, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo fmt --all --check
- PRs:
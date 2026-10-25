# fn-248.7 completion-review fixes: handoff (STOPPED on disk)

Worktree: temporal/.worktrees/fn250-gate, branch work/fn2504-gate.

## Commits
- 713edcb334 fix(temporal): qualify generated AnyMessage calls and refresh the dispatch snapshot
- 43b10a5dc5 docs(model): name prost-types as the well-known type owner
- eb44d3113c refactor(front): derive the lifted Model name with heck
- e4f0d148c5 refactor(catalog): share the unchanged-probe refusal text

## Fixes
1. HIGH, stale dispatch snapshot. This was also a second, separate failure: fn-248.5 added
   `use ::umpire_runtime::AnyMessage as _;` as the first item of the expansion, so `activity_expansion`
   failed its 3-item shape check ("unexpected realization expansion items") before it reached insta.
   The test now admits a leading `Use` item and checks that it is exactly that import.
   I accepted one snapshot, scoped with INSTA_UPDATE=always to `tests::activity_expansion`:
   `umpire_temporal_realization_macros__tests__activity-realization-dispatch.snap`. Its diff is only
   `::umpire_runtime::google::protobuf::Any` -> `::umpire_runtime::prost_types::Any` (4 lines: the matches, continue_polling
   and observe signatures). The authored and metadata snapshots did not change. Activity has no answer
   handlers, so the qualified unpack/required calls do not appear in it. Result: `cargo test -p umpire-temporal-realization-macros` 3/3 ok.
2. MEDIUM, unqualified AnyMessage calls. realize.rs now emits
   `<::umpire_runtime::prost_types::Any as ::umpire_runtime::AnyMessage>::{pack,unpack,required}` everywhere
   (reply validation, task unpack, Reply pack, callback unpack, required task). I removed
   `use umpire_runtime::AnyMessage as _;` from feature/activity/temporal.rs.
   Evidence: `cargo clippy -p umpire-temporal --all-targets -- -D warnings` passes, and annotate-syntax tests pass.
   Decision: I KEPT the macro's module-level anonymous `use ::umpire_runtime::AnyMessage as _;` (with a comment explaining why).
   The author's own hand-written code in temporal.rs calls `Any::pack`/`.unpack()` about 20 times. Before fn-248 those were
   inherent methods. Inherent methods on the foreign prost_types::Any are impossible, so this import is
   what lets the author file drop the import and keep the pre-fn-248 surface. `as _` binds no name, so it cannot
   collide. Other `AnyMessage as _` imports (tests, realization/src, kettle, runtime) belong to framework or test code
   and call the trait themselves, so I left them.
3. LOW, docs. model/README.md (also removed the now-false "Shared WKT ownership ... rebuilds" sentence),
   umpire/protocol/proto/README.md (WKTs come from prost-types via prost-build's default mapping, re-exported as
   `umpire_runtime::prost_types`; strict admission is in `AnyMessage`), and DESIGN.md (`realization.txt.snap`).
   Not changed: runtime build.rs still lists google/protobuf/*.proto inputs and `disable_comments([".google.protobuf"])`. With
   default WKT mapping these generate nothing. That is harmless dead config, and I left it out of scope.
4. LOW, `pascal`. Replaced with `heck::ToUpperCamelCase` (front gains `heck.workspace = true`; Cargo.lock adds the edge only).
   Evidence: a temporary test asserted that pascal(n) == heck(n) for all 116 inline-module idents in the workspace
   (including __lifted, __probe, self_, sample0..3, v1) plus a__b, x1y, a_1b, v2_api, sample_10, _x, x_, ab9c_d2.
   It passed, and I removed it before committing. The outputs differ only for idents with internal capitals (e.g. HTTPServer); none exist in the workspace.
5. LOW, literal. Added `pub const UNCHANGED_REFUSAL` beside `probe_applicability` in umpire/catalog/src/black_box.rs, used by
   black_box.rs, temporal/realization/src/probes/generic.rs (2 sites) and kettle/probes.rs (the review also missed this third
   duplicate there). The black_box unit test keeps the literal on purpose, to pin the wire text.

## Gates
- `cargo fmt --all --check`: exit 0 (after the fixes).
- `cargo test --workspace --locked --no-fail-fast --quiet`: exit 101, NOT A VALID RECEIPT. Both attempts ran the
  overlay out of space. Log: .flow/tmp/fn2487-workspace-test.log (second run, CARGO_INCREMENTAL=0).
  136 test binaries ok, 23 FAILED, 0 insta snapshot mismatches. Every failure I inspected is ENOSPC (os error 28):
  trybuild suites (malformed_paths, missing_display, refusals, private_state, kettle exhaustiveness,
  phase_conversion, realization_declarations, supplemental_requests) failed while building their trybuild deps
  under target/tests/trybuild, and the xtask apalache/identity tests failed creating temp dirs. Overlay `/` was at 0 bytes free at the end;
  the target dir is 24 GB. No failure attributable to fn-248 code was observed.
- clippy (workspace), guest freshness, and any guest regeneration: NOT RUN (stopped on disk).
  Expect guest freshness to report STALE: the macro expansion changed (qualified calls) and temporal.rs lost a line,
  and both feed the guest. If so, run `make umpire-guest` and update GUEST_BUILDS.md's fn-248.7 section.

## Needed from the conductor
Free overlay space (e.g. `target/tests/trybuild`, about 0.8 GB, or other stale target dirs), then rerun with CARGO_INCREMENTAL=0:
the workspace test, `cargo clippy --workspace --all-targets --locked -- -D warnings`, and guest freshness/regeneration.
Trybuild `.stderr` files that mention `google::protobuf` may still need review once they can run;
`grep -rn "google::protobuf" --include=*.stderr model` would find them.

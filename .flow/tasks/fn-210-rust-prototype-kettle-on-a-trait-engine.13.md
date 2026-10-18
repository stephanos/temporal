---
satisfies: [R29, R30]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.13 Shared end-to-end fixtures: umpire-tests crate, consumers selected by expected files

## Description
**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner.

Runs alone, after .12 (R29).

Preserve the authoring option sketches until .7: relocate samples/options/ to model2/options/ while removing samples, and update its links. Do not delete this comparison input prematurely.

R30 tooling: audit model2/scripts/time-gate.sh so failed timed commands propagate failure while budget overruns remain warnings. A confirmed command-substitution failure-masking regression has a minimal reproduction in this run's timing-wrapper.md notes; add an executable negative control and keep measured steps aligned with the actual gate.

- Create crate `umpire-tests` at `model2/umpire/tests/` (workspace member, `publish = false`, `autotests = false`, one `[[test]] name = "fixtures" path = "fixtures.rs"`), with a `build.rs` that generates the `#[path]` modules from `fixtures/*/model.rs`, and fixtures directly under `model2/umpire/tests/fixtures/<name>/`.
- Move `kettle`, `kettle_trait` and `kettle_cut` from `samples/tests/fixtures/`, and `choices`, `deep_chain`, `edge_checks`, `shared_starts`, `vacuity` from `checker/tests/fixtures/`, with their `expected/` byte-identical. Move `samples/src/kettle.rs` into `umpire/tests/fixtures/kettle/model.rs` (the fixture's own Model, not a re-export) and remove the `samples` crate and folder from the workspace.
- Give `umpire-testing` a consumer registry: each consumer declares the expected file names it owns and renders them; the runner runs a fixture through every consumer whose files it has, fails on an expected file no consumer owns, and on a fixture with none. Register the checker consumer now; TLC (.8) and Apalache (fn-211) register theirs later.
- Update the README's testing and layout sections.

**Files:** Makefile, model2/scripts/, model2/options/ (relocated sketches), model2/Cargo.toml (members), model2/umpire/tests/{Cargo.toml,build.rs,fixtures.rs,fixtures/}, model2/umpire/samples/ (removed), model2/umpire/checker/tests/fixtures/, model2/umpire/testing/src/, model2/README.md.

**Quick commands:** `make model2-check`
## Acceptance
- `model2/umpire/tests/fixtures/` holds the eight shared fixtures; checker-only fixtures remain in `checker/tests/fixtures/`; the `samples` crate and folder no longer exist; Kettle's source is its fixture's `model.rs`.
- Every moved golden is byte-identical.
- An unrecognised expected file, and a fixture with none, each fail naming the folder.
- `make model2-check` passes.

## Done summary
The eight shared Models and their native goldens now live in `model2/umpire/tests/fixtures/`, with a single `umpire-tests` runner selecting consumers from declared expected filenames. The trait Kettle executes natively with report, states and rows goldens, and timed command failures propagate while budget overruns remain warnings.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)
stage: impl-review - ran [2026-10-18T21:09:20Z..2026-10-18T21:14:30Z]

Review reached SHIP after the three-lens first round and one resumed review. The selected reviewer was `codex:gpt-6.1-sol:high`, from the same model family as the implementer. Receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.13.json`. Draws and merge are under `.flow/review-fanout/d190a3ae62c34c709ec8cedae0e705c8/`; the final response is `.flow/tmp/fn210-13-review-round2.log`. Read-only reviewer Cargo observations were inconclusive; the worker's executed checks below provide the gate evidence.

baseline: green (`make model2-check`, before edits; `.flow/tmp/fn210-13-baseline.log`). No gate was skipped.

Verification:
- Fresh final `make model2-check` passed. Log: `.flow/tmp/fn210-13-final-gate.log`; receipt: `.flow/tmp/green-receipts/abc6ce02-unittest.json`.
- The unchanged checker oracle passed with its configured 256 worlds. Log: `.flow/tmp/fn210-13-oracle256.log`.
- Release checker fixtures, including effect panic and arithmetic overflow, passed. Log: `.flow/tmp/fn210-13-release-fixtures.log`.
- `cargo deny check bans` passed, with the existing allowed duplicate-version warnings. Log: `.flow/tmp/fn210-13-deny-bans.log`.
- Default testing controls cover missing `model.rs` and `expected/`, unknown expected files, empty fixtures, orphan aliases/payloads/metadata/pending files, selected and unselected consumers, same-stem `.tla`/`.cfg` storage, and generated module compilation for hyphens, underscores, escape-like names, uppercase, digits and Unicode. The module-name regression failed before its fix. Logs: `.flow/tmp/fn210-13-discovery-red.log` and `.flow/tmp/fn210-13-review-fixes-focused.log`.
- The explicitly invoked optional CLI control `accepts_generic_pending_snapshots_with_cli -- --ignored` passed using real insta pending snapshots and `cargo insta accept`. It does not add a cargo-insta prerequisite to the default gate. Log: `.flow/tmp/fn210-13-insta-accept-final.log`.
- `kettle_trait/model.rs` compiled as a production Rust library with only `umpire-dsl`, without `cfg(test)` or `umpire-testing`. Log: `.flow/tmp/fn210-13-trait-pure.log`; emitted metadata: `.flow/tmp/fn210-13-trait-model.rmeta`.
- The shell timing regression passed its success, exit-1 and exit-37 controls; the actual `--step intentional_failure /bin/false` invocation returned 1. A zero-budget actual timing run returned 0 and printed both budget warnings. Log: `.flow/tmp/fn210-13-timing-warning-control.log`.

Preservation and paths:
- `.flow/tmp/fn210-13-source-paths.log` lists the eight final fixture Model sources. Kettle's canonical source is `model2/umpire/tests/fixtures/kettle/model.rs`; the production-includable trait source for .9 is `model2/umpire/tests/fixtures/kettle_trait/model.rs`. `kettle_cut` reuses the canonical Kettle Model with its existing budget.
- `.flow/tmp/fn210-13-move-manifest.txt` records the original Git identities. `.flow/tmp/fn210-13-byte-comparison.log` records exactly one comparison for each of 63 moved golden artifacts and seven relocated Rust option sketches, all byte-identical. Subsequent commits changed none of those payloads. Options now live in `model2/options/`; its README links and layout were updated.
- The trait fixture's three logical outputs are newly added, replacing the empty `.gitkeep` exception. Its report and states compare byte-for-byte with canonical Kettle, with 81 starts, 283 reached states, 424 emitted steps and depth 62. Its new rows golden contains all 424 emitted rows in structural order. The checker core, oracle and retained checker-specific fixture payloads were preserved.
- The samples crate and directory were removed. The changes are recoverable from Git history.

Owner provenance: owner R31/R32 edits arrived during the catch-all staging boundary and were captured in `47133e551d`, alongside the review fixes. They include the .9 protobuf recut, admitted dependency declarations and the `wasm32-wasip1` toolchain/portability target. The conductor explicitly directed retaining them without rewriting history and committed the matching timing label plus MILESTONES recap separately as `26141fea92`. Final verification used that current `wasm32-wasip1` gate. The .9 implementation remains separate work.

Defect route:
- prior fixes: local history identifies the initial timing helper in `ed1c8279d8`; memory search found no competing timing fix. GitHub PR/issue ownership checks were unavailable because authentication returned HTTP 401.
- diagnosis: the old exact helper called inside `out=$(step ... false)` returned 0 because Bash clears errexit within command substitution. Existing pipefail did not stop the following successful prints. Adding `|| return $?` to the timing assignment made the same reproduction return 1; exit 37 is preserved too.
- introduced by: `ed1c8279d8` initial timing helper. Bisect skipped because no prior working helper revision exists.
- base: failing reproduction committed in `316dbfa16c`; false returned 0 and the empty fixture was silently accepted. Head: both controls pass, and the direct timed false command returns 1.
- live: no live application surface; the shell helper and fixture runner are exercised directly.

Review fixes encoded fixture folder names without collisions and kept the optional cargo-insta CLI test explicit. The generated-identifier lesson was captured in `.flow/memory/bug/integration/encode-fixture-folder-names-before-2026-10-18.md`.

Final `make model2-time` output (exit 0; raw log `.flow/tmp/fn210-13-timing-final.log`):

```text
Time the model2 feedback loop (fn-210 R30)...
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
make[1]: Entering directory '/Users/stephan/Workspace/skunkworks/umpire/temporal'
Check model2 (fmt, clippy, docs, tests)...
timing-controls                   0.013s
fmt                               0.167s
clippy                            0.706s
docs                              0.419s
tests                             1.862s
wasm32-wasip1                     0.143s
make[1]: Leaving directory '/Users/stephan/Workspace/skunkworks/umpire/temporal'
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.033s
gate total                        4.058s  (budget 30s)
inner loop                        1.033s  (budget 3s)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 316dbfa16cb2182ea2c5dc62716b7d2f3a556121, 18977ed1a8b8ac9da51ccaee0081a6c73b9737bb, 47133e551d157dc0eb0860ccaabdce4da8dbece9, 26141fea928cf4984e271256a92be24d817395c3, abc6ce02c0b5c30d5171af47f8fdef01cf1a00b1
- Tests: make model2-check, cd model2 && PROTEST_CASES=256 mise exec -- cargo test -p umpire-checker --test oracle, cd model2 && mise exec -- cargo test --release -p umpire-checker --test fixtures, cd model2 && mise exec -- cargo deny check bans, cd model2 && mise exec -- cargo test -p umpire-testing, cd model2 && mise exec -- cargo test -p umpire-testing --test consumers accepts_generic_pending_snapshots_with_cli -- --ignored, /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model2/scripts/test-time-gate.sh, UMPIRE_INNER_BUDGET=0 UMPIRE_GATE_BUDGET=0 make model2-time, make model2-time, cmp model2/umpire/tests/fixtures/kettle/expected/report.txt model2/umpire/tests/fixtures/kettle_trait/expected/report.txt, cmp model2/umpire/tests/fixtures/kettle/expected/states.txt model2/umpire/tests/fixtures/kettle_trait/expected/states.txt
- PRs:
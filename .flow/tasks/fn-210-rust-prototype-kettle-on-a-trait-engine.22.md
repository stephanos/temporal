---
satisfies: [R8, R23, R29]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.22 Name sample fixtures explicitly and rename progress dead-end control

## Description
Owner-approved naming-only checkpoint: rename model/umpire/tests/fixtures/kettle to sample_kettle, kettle_trait to sample_kettle_trait, and kettle_no_heat to progress_dead_end. The sample_ prefix identifies the canonical attribute and manual-trait authoring examples; focused controls are named for the behavior they test. progress_dead_end demonstrates an unmet progress target on a finite maximal path in the assumption-filtered graph, with no enabled internal action to rescue it. Rename its NoHeat wrapper to a behavior-focused name. Update generated-registry module references in dependent fixtures (currently fixture_kettle), wasm source include path, wasm dispatch and Go host expected payload paths, and live documentation. Use the post-.19/.20 source as authoritative, preserve the .21 payload-only snapshot layout, and do not reintroduce symlinks. Preserve fixture semantics and payload bytes; no golden refresh except strictly necessary path attribution in snapshot review metadata. Do not rename other behavioral fixtures or archived Scala files. Reuse build-time discovery rather than adding name aliases or special-case registration. Touches: model/umpire/tests/fixtures/**, model/umpire/wasm/src/lib.rs, model/umpire/wasm/tests/dispatch.rs, model/umpire/hosts/go/host_test.go, model/README.md, model/DESIGN.md, MIGRATE.md. Quick: cargo test -p umpire-tests -p umpire-wasm; make model-check. This task stays todo until predecessors have committed local closures.

## Acceptance
All three old fixture directories are gone and replacements are discovered exactly once; all source consumers and live documentation resolve the new names. Canonical sample source behavior, graph statistics, verdicts, ordering and snapshot payload bytes remain unchanged. progress_dead_end retains its finite maximal path progress-refutation proof and uses behavior-focused wrapper naming. Snapshot metadata remains valid; no fixture symlinks or compatibility aliases are added. Full make model-check passes, including native shared fixtures, stock wasm and Go host parity; record before/after payload identity.

## Done summary
Renamed the shared fixtures `kettle` → `sample_kettle`, `kettle_trait` → `sample_kettle_trait`, and `kettle_no_heat` → `progress_dead_end`; renamed the control wrapper `NoHeat` → `ProgressDeadEnd`. Updated dependent generated-module imports, WASM source/dispatch includes, Go-host payload readers, and live README/DESIGN references. README now reports the actual inventory of 23 shared Models. Existing discovery is reused without aliases or special registration.

Final acceptance update: the owner replaced the former timing acceptance rule with 10 s inner / 30 s full soft targets. Small isolated overruns are recorded without blocking; substantial or repeated overruns warrant investigation. The original 3.038 s inner / 3.629 s gate observation remains unchanged and fits these targets. The earlier pending-exception wording below records the pre-amendment handoff and no longer blocks closure. Separate policy checkpoint 2d513c3e4e updates the runner and current docs/spec; budget boundary/override controls failed against the old default and passed after the update. Fresh full make model-check exited0 (fn210-soft-budget-model-check.log). The rename implementation is unchanged from its SHIP-reviewed checkpoint caff976929.

Source checkpoint: caff976929a4a5ae0cf369ab930694b758e7c5f3
Base: 19d686ba1071435f0a7938f17905654ff471f267
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal
Status: in_progress; conductor owns review and Flow completion.
Tier: implementer gpt-6.1-sol at medium (requested by conductor). No executed-model identifier was exposed; no actual-model claim is made.

Baseline: green. Both task Quick commands ran before edits, without reused receipts: `cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- cargo test -p umpire-tests -p umpire-wasm` exited 0 (`fn210-22-baseline-focused.log`); `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- make model-check` exited 0 (`fn210-22-baseline-model-check.log`). Both receipt checks returned RUN. Initial tracked tree and staging were clean.

Before/after identity: `.flow/tmp/fn210-22-identity.py` compares every shared expected file against the captured base with exactly the three directory mappings above. All 162 shared payload/metadata files remain byte-identical, including the moved eleven payloads and eleven metadata files. Both canonical sample source files are byte-identical to their base sources; the control source differs only in the generated import and wrapper name. The three old directories are absent, and no fixture symlink or pending snapshot was created. Logs: `fn210-22-identity-before.log`, `fn210-22-identity-after-move.log`, `fn210-22-identity-after-gates.log`, `fn210-22-identity-final.log`. Git's rename heuristic pairs some identical generic metadata across old fixtures; the exact-mapping byte comparison independently validates each intended destination.

The progress control preserves one 20-degree start, two reached states, two emitted edges, eight evaluated actions, six refusals, depth one, no budget cut, and `boils` refuted. Its unchanged counterexample is one `User::PressButton` to heating-on at 20 degrees. DepressButton is excluded by its fixed assumption, Heat remains disabled, and Failover is unavailable at 20 degrees, preserving the finite maximal path proof.

Registry proof: `cargo test --workspace --no-run --message-format=json` exited 0, capturing the actual build-script outputs in `fn210-22-active-artifacts.jsonl` and stderr in `fn210-22-active-artifacts.stderr`. `.flow/tmp/fn210-22-registry-proof.py` verifies all four ACTIVE registries from those outputs, not stale target-directory search results: umpire-dsl-macros (3 fixtures), umpire-dsl (4), umpire-checker (24), umpire-tests (23). Every generated source path is an existing file in this ROOT checkout with no worktree path. Every shared fixture occurs exactly once, old names are absent, and generated modules are `fixture_sample_5fkettle`, `fixture_sample_5fkettle_5ftrait`, and `fixture_progress_5fdead_5fend`. See `fn210-22-active-registry-paths.log`.

Verification: focused `cargo test -p umpire-tests -p umpire-wasm` exited 0 (`fn210-22-focused.log`), including the shared fixture runner and all four dispatch tests. The first post-change full gate caught a rustfmt wrap required by the longer include path (exit 2, `fn210-22-model-check.log`); the ordinary formatting fix was applied. Final `make model-check` exited 0 (`fn210-22-model-check-final.log`), covering fmt, all-target Clippy, docs, native workspace fixtures/refusals, capability negative controls, protobuf freshness, stock WASM builds, core WASM Clippy and uncached CGO-disabled Go-host parity. Logs contain nonzero executed test counts; no suspicious empty suite was treated as coverage. Gate classification required FULL. No fabricated red regression test was added for this naming-only change; byte identity and full parity are its evidence. No goldens were accepted or regenerated.

Exactly one warm `make model-time` ran after source commit, exiting 0 (`fn210-22-model-time.log`), without retry:

```text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check
timing-controls                   0.015s
capability-controls               0.117s
protobuf-freshness                0.081s
fmt                               0.119s
clippy                            0.106s
docs                              0.138s
tests                             1.008s
wasm32-wasip1                     0.063s
core-wasm-clippy                  0.086s
wasm-guest                        0.086s
go-host                           1.541s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            3.038s
gate total                        3.629s  (budget 30s)
inner loop                        3.038s  (budget 3s)
warning: inner loop over budget (fn-210 R30)
```

The timing warning is retained as observed (38 ms above the warning budget); it is not a command failure. The conductor has requested an owner exception for this .22 R30 timing overrun; that exception is pending and no acceptance or completion is claimed. Final identity, registry proof and `git diff --check` passed after timing, and the tracked tree is clean. Read-only `flowctl show` confirms in_progress. All commands are terminal; no subagents were dispatched. No Flow state, MILESTONES, historical Flow/planning/sketch/archive content, manifests, locks, ABI or behavior was changed. MIGRATE.md has no live old fixture reference, so it needed no edit. Gate receipts were not written because the conductor retained Flow state ownership.

Implementation review round1 returned SHIP in all three draws on gpt-6.1-sol at high effort, with zero findings. The requested and observed model match; writer and reviewers share the GPT model family. Receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.22.json has SHA256 e95d75dc7048a4246324fd96fbb69c443a4451709aa68cb8972742d32e560954. The review route closed with no live reservation. Read-only Cargo lock permissions prevented reviewer reruns; the worker's actual root full gate and conductor's independent identity check supply execution evidence. SHIP does not resolve the pending owner timing exception. The task remains in_progress.

stage: impl-review - ran (model: gpt-6.1-sol high)
stage: plan-sync - skipped(config: planSync.enabled=false)
Memory capture: skipped(policy: clean first-pass SHIP and naming-only change)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: caff976929a4a5ae0cf369ab930694b758e7c5f3, 2d513c3e4e
- Tests: baseline: green; cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- cargo test -p umpire-tests -p umpire-wasm (exit 0; .flow/tmp/fn210-22-baseline-focused.log), baseline: green; CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- make model-check (exit 0; .flow/tmp/fn210-22-baseline-model-check.log), python3 .flow/tmp/fn210-22-identity.py --before (exit 0; 162 shared snapshot files, 22 moved files protected), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- cargo test --workspace --no-run --message-format=json (exit 0; .flow/tmp/fn210-22-active-artifacts.jsonl), python3 .flow/tmp/fn210-22-registry-proof.py (exit 0; all four active ROOT registries and exactly 23 shared fixtures; .flow/tmp/fn210-22-active-registry-paths.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- cargo test -p umpire-tests -p umpire-wasm (exit 0; .flow/tmp/fn210-22-focused.log), first post-change CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- make model-check (exit 2: required rustfmt wrap; fixed; .flow/tmp/fn210-22-model-check.log), final CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- make model-check (exit 0; .flow/tmp/fn210-22-model-check-final.log), ONE postcommit CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 mise exec -- make model-time (exit 0; gate 3.629s, inner 3.038s; inner-loop warning retained; owner exception pending at conductor; .flow/tmp/fn210-22-model-time.log), python3 .flow/tmp/fn210-22-identity.py (exit 0 after moves, gates and timing; all 162 payload/metadata files byte-identical under exact path mapping; both sample source files byte-identical; .flow/tmp/fn210-22-identity-final.log), git diff --check (exit 0); git status --short (empty); no pending snapshots or fixture symlinks; flowctl show fn-210.22 confirms in_progress, Conductor independently reran python3 .flow/tmp/fn210-22-identity.py: all162 shared snapshots exact under path mapping and sample sources unchanged; live old-name search empty, Implementation review round1 SHIP, all3 draws gpt-6.1-sol/high; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.22.json SHA256 e95d75dc7048a4246324fd96fbb69c443a4451709aa68cb8972742d32e560954; R30 ownerexception stillpending, Owner amended R30 to soft10s inner/30s full, record minor isolated overruns withoutblocking; original .22 run3.038s/3.629s retained, nowwithinsofttargets; no timingretry, Budget boundary controls first failed on old3s default, then /bin/bash model/scripts/test-time-gate.sh passed on new10s/30s defaults, equality/overrun/override cases; bash -n passed, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check exit0 after ownersoftbudgetchange (.flow/tmp/fn210-soft-budget-model-check.log)
- PRs:
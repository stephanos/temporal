---
satisfies: [R41]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.15 Archive the Scala model under model/archived and rename model2 to model

## Description
Runs alone, after .5 completes, before .6 (R41). No other session may run while this does; every session re-reads its task after it.

1. `git mv model model2/archived`, then move any untracked content left in `model/` (build caches, about 4 GB) into `model2/archived/` with plain `mv`, so nothing is lost and the nested `.gitignore` files keep ignoring it.
2. `git mv model2 model`.
3. Makefile: `MODEL_ROOT := model/archived` (Scala targets stay, unmaintained); the Rust section's `cd model2`, script paths and target names become `cd model`, `model/scripts/...`, and `model-check`, `model-fmt`, `model-update`, `model-time` (keep the `MODEL2_STEP`-style macro, renamed).
4. `AGENTS.md`: Core Mandates and Project Structure describe `model/` as the Rust workspace (Umpire in Rust, gate `make model-check`) and `model/archived/` as the frozen Scala framework, reference only.
5. Workspace references: `model/README.md`, `model/DESIGN.md`, `model/Cargo.toml` comments, `model/scripts/*`, `model/hosts/go`, `model/.cargo/config.toml`; links to `model/framework/...` (Scala) become `model/archived/framework/...`.
6. Flow and docs: open specs and tasks of fn-210, fn-211, fn-212 (`model2/` to `model/`, Scala `model/` paths to `model/archived/`), MILESTONES; the deferred Scala specs are left as they are (their paths now resolve under `model/archived/`, noted in MILESTONES).
7. Run `make model-check` and `make model-time`; grep for leftover `model2` outside `.flow` history and `archived/`.

**Files:** model/ (moved), model2/ (renamed), Makefile, AGENTS.md, MILESTONES.md, .flow/specs/fn-21[012]-*, .flow/tasks/fn-21[012]-*.

**Quick commands:** `make model-check`; `make model-time`; `git status --short | head`; `grep -rn model2 --exclude-dir=.flow --exclude-dir=archived .`

## Acceptance
- `model/` is the Rust workspace and `model/archived/` the Scala tree with its history (`git log --follow` works on a moved file).
- `make model-check` passes and `make model-time` is within R30.
- `AGENTS.md` describes the new layout; no `model2` path remains outside history and `archived/`.
- Nothing from the old `model/` is lost (untracked caches moved, not deleted).

## Done summary
Rust now occupies model/ and Scala is frozen under model/archived/. Current Rust commands, workspace references, AGENTS.md, MILESTONES and the open fn-210/fn-211/fn-212 plans use the new layout. Scala targets point at the archive but remain unmaintained; no Scala/JVM build was attempted.

The canonical physical rename preserved all 480,208 filesystem inventory records, including device/inode identity, size, mtime, mode, owners and symlink targets. Before, after-move and integrated manifests have SHA-256 f549f1782d219a390b9728cd060cb39b7e9e6b5ed20f035bf0dd1c3b0b67a565; both cmp commands passed. This is identity/metadata evidence, not a claim to hash every payload. The archived tracked tree remains cb5223b9a4839233876c98cf6158a46dc9d97a19, equal to the pre-move Scala tree, and git log --follow reaches the original source. No canonical cache was deleted or copied. Details: .flow/tmp/fn21015-canonical-preservation.md.

The first integration conflicted on model/README.md and was aborted without hunk resolution. One serial retry joined the identical pure-move histories, then merged the existing reference update from their shared moved ancestor. Original worker commits remain reachable. The retry's initial wrong-cwd gate and batched-rg status were inconclusive, not green; corrected commands and the original failure are retained in the raw handover.

Review found a migration-induced Go layout regression. The fix separates physical archived Scala scans from historical logical model/... paths, explicitly asserts the live Rust layout, and preserves cache exclusions and vocabulary coverage. Red controls precede their fixes. The full pre-move and final Go IR packages fail the same 47 top-level tests; a pinned actual baseline reproduces TestFrameworkLayout's existing namespace failure. A lost vocabulary match discovered by diagnostic comparison was repaired: the normalized 922-location set is identical, with none added or missing. Existing failures were not skipped or relabeled green. The archived source bytes and production IR/schema contracts were unchanged.

Three review rounds completed in the same receipt lineage: initial NEEDS_WORK, repaired SHIP, then audit-fix SHIP. Final receipt /tmp/impl-review-receipt-c855d6cf59f5-fn-210-rust-prototype-kettle-on-a-trait-engine.15.json has SHA-256 d636896486a612e325569639eeb2f01b51825999ee2a02143587aff48a337f25, model gpt-6.1-sol at high effort, zero introduced findings. Reviewer runtime checks were sandbox-inconclusive; the conductor ran the integrated checks below. Same-family review is recorded honestly.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key).
stage: wave-join - failed(collision: model/README.md; aborted without hunk resolution)
stage: wave-join - ran (one serial retry; ancestry-preserving joins and final fast-forwards)
stage: impl-review - ran (three rounds, same receipt lineage; final SHIP) (model: gpt-6.1-sol high)

Integrated final verification at d98648a0fad86510dd6b9fadd854f4a0953648d5: make model-time exited 0, including the complete actual model-check gate; the final new layout/vocabulary controls passed. The earlier integrated seven-control suite also passed. Logs are .flow/tmp/fn210-15-final-audit-time.log, fn210-15-final-audit-focused.log and fn210-15-final-focused.log. Worker logs are archived under .flow/tmp/fn21015-retry-archived/ and review draws under .flow/review-fanout/e307f1f6c2b249d18d283d16af56392c/. Full legacy Go package red logs and exact pinned comparisons are retained there. A generated baseline snapshot remains recoverable under /tmp/fn21015-baseline-recovery.41zDLw/snapshot; it is not a canonical checkout or cache.

Actual final warm timing output:

```text
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.822s
gate total                        3.596s  (budget 30s)
inner loop                        1.822s  (budget 3s)
```

Both R30 bounds pass. Memory auto-capture after the non-trivial review fix created bug/integration/separate-archived-scan-roots-from-2026-10-18 (low overlap, no matches). No external repository, PR, push or release was changed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0fc76f7805f1ef68f828b03ed78d7f86fcb03cfc, 34ad8dbf926fc4e4a501ddc47a16d4f83810a577, 6264f117607d756d41a97342273ebff189ef77f1, e87ee28f2dbb0bf640b61cee81bb86c9a6b9cf03, 1de206d2c6a6d274c5b0bca6a84541a780aff1be, 45fab9b03d9a2ffdb758e79386734207fe17ce56, 6277eaf5969a9f8fb1630de823d45d2b9acf3ec7, da16692ce9e8698ea70490b9bf7bca4b38dc55a2, c70b93838700656451def3d08129f087616bea83, 5f21aa679be21700ea05019811e472bb98d07caf, fed4d04555, c1b6f1de03842521e25a209d31fa3d7f3772a763, d98648a0fad86510dd6b9fadd854f4a0953648d5
- Tests: Canonical before/after/integrated 480208-entry filesystem manifests: both cmp exit0, same SHA256 f549f1782d219a390b9728cd060cb39b7e9e6b5ed20f035bf0dd1c3b0b67a565, git diff --exit-code 362616295e197a7f492a28be2ce2559362643f8b:model HEAD:model/archived: exit0; exact tracked tree cb5223b9a4839233876c98cf6158a46dc9d97a19, git log --follow on model/archived/framework/samples/Kettle.scala: reaches pre-move source history, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time: integrated final exit0; complete model-check gate; full3.596s inner1.822s; .flow/tmp/fn210-15-final-audit-time.log, mise exec -- go test -tags test_dep ./tools/umpire/ir -run '^(TestModelWorkspaceLayout|TestRetiredVocabularyCoversTheWholeModelTree)$' -count=1: integrated final exit0; .flow/tmp/fn210-15-final-audit-focused.log, Integrated earlier seven-control layout/cache/namespace-detector suite: exit0; .flow/tmp/fn210-15-final-focused.log, Worker final eight focused controls exit0; red cache/vocabulary controls recorded before fixes; .flow/tmp/fn21015-retry-archived logs, Pinned full pre-move baseline and final full tools/umpire/ir package: both RED, same47 failing top-level names; normalized vocabulary coverage exactly922 matches, none added/lost; diagnostic comparison recorded, not claimed green, Same receipt Codex review round3 SHIP, zero introduced; SHA256 d636896486a612e325569639eeb2f01b51825999ee2a02143587aff48a337f25; reviewer runtime checks sandbox-inconclusive, conductor runtime checks above, Tracked current-source model2 scan: only explicitly historical timing command in DESIGN.md; active commands use model paths; archived and completed history excluded, git diff --check: exit0; no Scala/JVM builds
- PRs:
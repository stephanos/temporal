---
satisfies: [R7]
---
# fn-114-gomad-correct-search-path-defects-and.9 Keep one copy of each prepared target per artifact store

## Description
E2 (R7), store half: publication places the target once per store and every artifact in that store shares it; readers verify it before execution. Accounting, pruning, merge, and the measurement are task 10. Depends on tasks 3 and 8 because they edit the corpus and minimizer files this task also changes.

**Size:** M
**Files:** `tools/gomad3/artifact/store.go`, `publication.go`, `open.go`, `store_test.go`, `publication_test.go`, `tools/gomad3/runner/replay_operation.go` and its test, `tools/gomad3/runner/minimize_operation.go`, `tools/gomad3/runner/internal/corpus/corpus.go`, `tools/gomad3/runner/inspect.go`
**Touches:** [tools/gomad3/artifact/**, tools/gomad3/runner/replay_operation.go, tools/gomad3/runner/replay_operation_test.go, tools/gomad3/runner/minimize_operation.go, tools/gomad3/runner/internal/corpus/**, tools/gomad3/runner/inspect.go, tools/gomad3/runner/inspect_test.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/simulation_exploration_campaign.go, tools/gomad3/runner/runner_test.go, tools/gomad3/record/**, .flow/specs/fn-114-gomad-correct-search-path-defects-and.md]

### Approach
- First step: choose the sharing form and record the choice in the spec's Decision Context.
  - Recommended: a content-addressed target pool, keyed by the SHA-256 and size the manifest already records, with each artifact's `target` file a hard link to the pool entry. The manifest, the artifact schema, and every reader stay unchanged, retained artifacts stay readable, and any recursive copy of an artifact directory is a self-contained export.
  - Alternative: the manifest references the pool entry and the artifact directory holds no target. This needs a new target form in the record, a schema decision for retained artifacts, store-level resolution in a reader that is confined to the artifact directory today, and an explicit export step. Take it only if hard links cannot meet R7, and say why.
- Pool ownership: the publication stores in use today are narrower than the sharing boundary. Their roots are per-kind directories and per-round staging directories (`successes` under a staged round, the campaign's successes path, the corpus cases path, the minimizer output root). A pool under each of those would not share across rounds or campaigns. The pool therefore belongs to the artifacts root the command was given (the directory that holds every campaign of that root), and the corpus and the minimizer output root each own one for their own directory. `Store` receives the pool location from its caller; it does not derive it from `Root`.
- A staged round is renamed into place on commit. Links survive the rename, and the pool is outside the staged directory, so an abandoned round leaves at most an unreferenced pool entry for pruning (task 10).
- Publication: write the pool entry by temporary file and rename, then link. On an existing entry verify SHA-256 and size before linking. Two publishers of the same target (parallel executions, shards sharing a store) must both succeed with one pool entry.
- A store on a filesystem without hard links falls back to a private copy and the result says so. It is never a failure and never a silent success of sharing.
- Readers keep verifying the target's hash and size before execution, as they do today. Add tests for a pool entry or link whose content was altered and for a truncated one; both fail before execution.
- An artifact copied out of its store with a plain recursive copy replays on its own.
- Cover the same path for the minimized-artifact store and the corpus store.
- Mode bits are shared across links. The pool entry carries the mode the manifest records for the target today, unchanged; the manifest validator accepts only the two existing modes and readers compare the mode exactly. Do not make targets read-only. Readers already copy the target before executing it, so execution never writes through a link.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/artifact/publication.go:40-80` — target payload placement
- `tools/gomad3/artifact/store.go:65-190`, `:229-300` — publication, staging, store identity, payload copy
- `tools/gomad3/artifact/open.go:19-70`, `:122-180`, `:214-258` — open, directory validation, payload copy
- `tools/gomad3/runner/minimize_operation.go:200-204` — target copy for minimization
- `tools/gomad3/runner/runner.go:923`, `:1805`, `tools/gomad3/runner/choice_exploration_campaign.go:374` — store roots that publication uses today
- `tools/gomad3/record/validation.go:655`, `tools/gomad3/artifact/open.go:311-320` — accepted file modes and the exact mode check

**Optional** (reference as needed):
- `tools/gomad3/runner/replay_operation.go:137-141`, `:477` — target copy and verification before replay
- `tools/gomad3/record/validation.go:89` — file reference validation
- `tools/gomad3/runner/internal/corpus/corpus.go:259-300` — corpus store and byte cap
- `.flow/memory/bug/integration/shard-merge-and-prepared-target-cache-2026-09-29.md` — prior identity bug in the prepared-target cache
- `tools/gomad3/ARCHITECTURE.md:355-381` — artifact layout contract

### Key context
- fn-109 task 12 separates detached Artifact references from owned handles in the same files. If it has landed, build on its handle types; if it has not, keep this change inside the existing functions so that task can still apply.
- Artifact directories are validated file by file against the manifest; an unlisted file fails validation. The pool lives outside artifact directories.
- A hard-linked file passes regular-file checks. Confirm the no-symlink open path treats it as regular on both platforms.
## Acceptance
- [ ] The sharing form is chosen and recorded with its reason
- [ ] An artifacts root with N artifacts of one target holds one copy of the target binary, shown by a test that counts distinct files and spans successes and failures, several exploration rounds, and two campaigns
- [ ] The published target's recorded mode and on-disk mode are the ones published today, and existing artifacts open unchanged
- [ ] Replay, `replay --verify-only`, resume, minimize, and inspect pass their existing tests unchanged
- [ ] Two concurrent publications of the same target into one store both succeed and leave one pool entry
- [ ] An altered or truncated shared target fails before execution; a missing one fails before execution
- [ ] An artifact copied out of its store replays on its own
- [ ] A store without hard-link support publishes private copies and reports that sharing is off
- [ ] `go -C tools/gomad3 test -tags test_dep ./artifact/... ./runner/...` and `make -C tools/gomad3 validate` pass
## Done summary
Artifacts of one target in one artifacts root, corpus, or minimizer output root now share one copy of the target binary (R7/E2, store half). Each artifact's `target` is a hard link to `OWNER/targets/sha256-<hex>`. The manifest, the artifact schema, and the record hash are unchanged, so retained artifacts open as before and `cp -R` of an artifact directory is a standalone export.

What a user sees:

- `ARTIFACTS/targets` appears beside `v1`; a corpus gets `CORPUS/targets`; a minimizer output root gets `OUTPUT/targets`.
- Sixteen artifacts from six campaigns (seed, choice exploration, simulation exploration; successes and failures; three rounds per exploration) hold one target file.
- An altered, truncated, or missing target fails `replay` and `replay --verify-only` before the target starts.
- A pool entry that no longer matches its name fails every later publication of that target in that pool. Nothing repairs it; removing the entry is the recovery until task 10 adds pruning.
- The default minimizer output root is `ARTIFACTS/minimized`, so a parent artifact and its minimized artifact sit in two pools and keep two copies.

How it holds together:

- `artifact.Store.TargetPool` is supplied by the caller; `artifact.TargetPool(owner)` names the directory. An empty pool publishes a private copy, as before.
- `placeSharedPayload` (`artifact/target_pool.go`) writes a missing entry into a staging directory in the pool and publishes it with the existing no-replace rename. A publisher that loses the race links to the winner's entry. The link is verified for mode, size, and SHA-256 before the manifest is written.
- A link that fails with `EXDEV`, `EPERM`, `EMLINK`, or an unsupported operation gives the artifact a private copy and `Artifact.TargetSharing` reports `unshared`. Any other link failure fails the publication.
- The target keeps mode 0700 on disk and in the manifest.
- The minimizer's scratch candidate store pools in its work directory, so a candidate links the target instead of copying it.

Correction to the task's premise: a hard-linked file does not pass the no-symlink open path. `hostfs.OpenRoot` rejects a link count other than one, and `internal/hostfs` is outside this task's Touches. `artifact.openSharedFile` therefore repeats the same checks without the link count, for the one file the manifest names as the target. Every other payload keeps the single-link check. The target's content is hashed at open and again while it is copied for execution.

The sharing form and these reasons are in the spec's Decision Context.

Tests, one per acceptance item:

- One copy across stores, rounds, and campaigns: `TestRunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns`, `TestPublishSharesOneTargetAcrossTheStoresOfOnePool`.
- Mode unchanged and artifacts without a pool unchanged: the same artifact test and the existing store tests.
- Concurrent publishers: `TestPublishConcurrentlyIntoOnePoolLeavesOneEntry` (eight publishers, one entry; green under `-race`).
- Altered, truncated, missing: `TestReplayRejectsDamagedSharedTargetBeforeTargetStart` (replay and verify-only), `TestDamagedSharedTargetFailsOpenAndLaterPublication`.
- Export: `TestReplayRunsAnArtifactCopiedOutOfItsStore` (`cp -R`, then the store and pool are removed), `TestCopiedArtifactOpensWithoutItsStore`.
- No hard links: `TestPublishWithoutHardLinksKeepsPrivateCopiesAndSaysSo`.
- Minimizer and corpus: `TestMinimizeKeepsOneTargetCopyInItsOutputRoot`, `TestCorpusCasesShareOneTarget`.
- The narrowed reader exception: `TestOpenRejectsAnotherLinkToAPayloadThatIsNotTheTarget`.

No existing test changed. Fifteen single-change mutations each turned a named test red (`mutation-checks.json`).

Gates on darwin/arm64 at 577003f98: `test-host` exit 0 (45 packages, 185 s), `validate` exit 0, architecture test, vet, and `-race` on the artifact pool tests exit 0.
GATE_SKIPPED:unittest:green-receipt 577003f9 - Verify at 91c1b5ea5 reused the post-commit pass; only .flow evidence files changed after it

Inconclusive, not counted as a pass: one pre-commit focused run exited 1 while the host load average was 14. `TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs/100_jobs/discard` hit `overall_timeout` after 86 of 100 executions; that row publishes no artifact, its focused rerun passed in 9.6 s, and it passed in the full gate. `TestModelConformanceFilesystem` and `TestModelConformanceTCP` failed in the same run because it lacked the stock-go setup `test-host` provides.

Not met or not run:

- The hard-link fallback test uses an in-package link seam. No filesystem without hard links was mounted.
- linux/amd64 was not run (no native host). The `renameat2` no-replace rename of a regular file and the fallback errnos are compiled and vetted for linux only.
- The spec's literal `go -C tools/gomad3 test` Quick command with an unpatched go was not run; `test-host` covers both package sets.
- "Reports that sharing is off" is met at the publication result only. No campaign result or CLI output reads `TargetSharing`; `cmd/gomad` is outside this task's Touches.

Review: SHIP on the first round from claude-fable-5-1 at high through the `claude` backend (same family as the writer; the reviewer had no shell and read the committed gate evidence). Four P3 findings stay open as follow-ups:

- `openSharedFile` duplicates `hostfs.openRegular`. A link-tolerant variant belongs in `hostfs` once a task may edit it.
- The `artifact.TargetPool` doc comment says the pool is never below a store root. The minimizer's final store uses `Root: OutputRoot` with the pool at `OutputRoot/targets`. The real constraint is a store root that is staged and renamed. Task 10 edits `artifact/**` and should correct the comment before it writes pruning against it.
- On a filesystem without hard links the first publication still creates one pool entry that nothing links to.
- `distinctFiles` exists in both the artifact and runner test packages.

For task 10: unreferenced pool entries come from a failed or capacity-rejected publication, an abandoned round, a discarded corpus case, and a `.publish-*` staging directory left by a crash. Per-artifact `StoredBytes` still counts the target. A resumed campaign derives its pool from two directories above the campaign path.

For task 14's docs: the `targets` directories, the fail-closed pool entry, and the `cp -R` export.

Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-9/`.

stage: impl-review - ran (SHIP, claude:claude-fable-5-1:high, round 1)
## Evidence
- Commits: 577003f9833bbb0163b5e2a9c39851552f98bf42, 91c1b5ea5f1649e061a2672cbc365163b1135b64, 765dc0e4d88d8189312b0b6a063b6ea3e7f57982
- Tests: baseline: green via handoff (full test-host verified at 44dbf795 by the conductor after fn-114.5: 45 packages, exit 0, toolchain build key 245141dc), GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host (at 577003f98: exit 0, 45 packages, 185 s, pinned stock go1.27.1 first on PATH), GATE_SKIPPED:unittest:green-receipt 577003f9 - Verify at 91c1b5ea5 reused the post-commit pass; only .flow evidence files changed after it, make -C tools/gomad3 validate (exit 0), .toolchain/bin/go test -tags test_dep -count=1 -run TestPackageArchitecture . (exit 0), .toolchain/bin/go vet -tags test_dep ./artifact/... ./runner/... (exit 0), .toolchain/bin/go test -tags test_dep -count=1 -race -run 'TestPublish|TestDamaged|TestCopied|TestOpen' ./artifact/ (exit 0), 15 single-change mutations, each red against its named test (.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-9/mutation-checks.json), INCONCLUSIVE: pre-commit focused run '.toolchain/bin/go test -tags test_dep -count=1 ./artifact/... ./runner/... ./cmd/... ./qualification/...' exit 1 under load average 14: TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs/100_jobs/discard hit overall_timeout (focused rerun exit 0 in 9.6 s; green in the full gate) and TestModelConformanceFilesystem/TCP failed on the missing stock-go setup (green in the full gate), NOT RUN: linux/amd64 (no native host); the spec's literal 'go -C tools/gomad3 test' Quick command with an unpatched go; root make lint-code-fast; a real filesystem without hard links
- PRs:
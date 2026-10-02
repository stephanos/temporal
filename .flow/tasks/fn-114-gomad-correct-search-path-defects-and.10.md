---
satisfies: [R7]
---
# fn-114-gomad-correct-search-path-defects-and.10 Account, prune, and merge shared targets and measure retained bytes

## Description
E2 (R7), consumer half: byte accounting, pruning, shard merge, and the corpus cap work with a shared target, and the saving is measured.

**Size:** M
**Files:** `tools/gomad3/artifact/store.go`, `tools/gomad3/qualification/set/prune.go`, `prune_test.go`, `tools/gomad3/runner/internal/campaign/merge.go`, `merge_capacity_test.go`, `tools/gomad3/runner/internal/corpus/corpus.go`, `corpus_test.go`, `tools/gomad3/runner/minimize_operation.go`, `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/retained-bytes.md`
**Touches:** [tools/gomad3/artifact/**, tools/gomad3/qualification/set/**, tools/gomad3/runner/internal/campaign/**, tools/gomad3/runner/internal/corpus/**, tools/gomad3/runner/minimize_operation.go, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/retained-bytes.md]

### Approach
- Stored-bytes semantics: state in one place whether an artifact's stored bytes include the shared target. Recommended: per-artifact stored bytes keep their current meaning, so manifests and identities do not change, and each capacity check that sums them counts a shared target once.
- Apply that rule to the three consumers: the success-evidence byte limit in merge, the corpus byte cap, and the default byte limit for a minimized artifact. Do not raise any limit.
- Corpus: the finding says the 1 GiB cap holds about six `./tests` cases. Show the number of cases the cap holds after the change.
- Pruning: after campaign directories are removed, remove a pool entry only when no retained artifact under that pool's root still shares it. This also collects entries left by abandoned rounds. Never remove one that is still shared.
- Merge: shard batches produced in different stores or on different hosts merge into one store with one copy per target. A shard whose target bytes differ from the plan's target is rejected as today.
- Measure the representative qualification set's retained bytes before and after with the same command, and retain both numbers and the command.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/artifact/store.go:218-228` — stored-bytes computation
- `tools/gomad3/runner/internal/campaign/merge.go:540-580` — evidence byte accounting in merge
- `tools/gomad3/runner/internal/corpus/corpus.go:286-300` — corpus byte cap
- `tools/gomad3/runner/minimize_operation.go:112-120` — default minimized-artifact byte limit
- `tools/gomad3/qualification/set/prune.go:18-64` — campaign pruning

**Optional** (reference as needed):
- `tools/gomad3/qualification/set/prune_test.go` — pruning tests
- `tools/gomad3/runner/internal/campaign/merge_capacity_test.go` — capacity tests
- `tools/gomad3/README.md:334` — the retained-size statement to update in task 14
- root `Makefile:192-198` — representative qualification target and its artifact root

### Key context
- The measurement needs a full representative run on darwin/arm64 (about 11 GiB before). If task 14 will run the set anyway, take the after number there and record the before number here from the current retained set.
- fn-109 task 12 edits `artifact/store.go` and `corpus.go`; rebase onto whichever landed.
## Acceptance
- [ ] The stored-bytes rule is stated in the code contract and applied by merge, the corpus cap, and the minimized-artifact default
- [ ] The corpus cap holds more `./tests` cases than before, with the before and after counts recorded
- [ ] Pruning removes a pool entry only when nothing retained shares it; a test covers an entry still shared by a corpus case
- [ ] Shard merge and `merge --partial` pass their existing tests and leave one copy per target
- [ ] Qualification pruning passes its existing tests
- [ ] The representative set's retained bytes before and after, with the command, are retained in `retained-bytes.md` in the spec's artifacts directory
- [ ] `go -C tools/gomad3 test -tags test_dep ./artifact/... ./runner/... ./qualification/...` and `make -C tools/gomad3 validate` pass
## Done summary
Byte limits over several artifacts now count a shared target once, unshared pool entries are pruned, and the saving is measured (R7/E2, consumer half). The unpruned representative set retains 1.17 GB on disk; the same 112 artifacts with a private target each are 11.44 GB.

What a user sees:

- The 1 GiB corpus cap holds 45 to 1,024 `./tests` cases, depending on what a case holds besides the target, where it held 6. The cap and the 1,024-case cap did not change.
- `gomad merge` charges each distinct target once in the failure bytes and once in the success bytes. Shards that failed to merge over `--success-bytes` only because every artifact carried the target now merge.
- `qualify-set --prune-qualified-artifacts` removes a target from `ARTIFACTS/targets` once the last Campaign linking to it is gone. A corpus does the same when it discards a case, and clears a staging directory a crashed publisher left.
- `gomad inspect` prints `sharing=shared` or `sharing=private` on the `target:` line, and `"sharing"` under `target` in JSON. The inspect schema stays `v5`; the field is additive and omitted where the platform reports no link count.

How it holds together:

- The rule is stated once, on `artifact.RetainedBytes`: per-artifact stored bytes keep the target, so manifests, identities, and single-artifact bounds are unchanged; a sum over artifacts kept together counts a shared target once.
- The corpus reads from disk whether each case links to the corpus pool. A case with a private copy counts in full.
- A merged campaign accounts by target SHA-256, as one store would hold the evidence. `MergedEvidence` gained `target_sha256` and `target_bytes` (omitted when empty); a merged record written earlier has neither and still validates with every artifact in full.
- `artifact.PruneTargetPool` removes an entry only when its link count is 1, through an `os.Root` on the pool, and refuses a pool that is a symbolic link. It needs no list of retained artifacts.
- A publisher whose pool entry is pruned between creation and link creates it again, up to three times.
- The minimized-artifact default needed no arithmetic change: both sides are single-artifact stored bytes. Its comment says so, and a test pins that stored bytes are equal with and without sharing.

Where the task's premise did not hold:

- `gomad merge` publishes a record and copies no evidence, so it cannot leave "one store with one copy per target". Shards of one artifacts root already share one copy and merge leaves it untouched; shards of two roots keep one copy per root. The merged byte limits count one per target either way.
- Merged records failed to open once their executions' success bytes passed the limit, because `validateCampaign` bounds that sum too. `validateMergedCampaign` now hands it limits wide enough for that sum and keeps the capacity check on the retained evidence. The reviewer flagged this as a workaround (P3, below).
- No set run was made at a revision before sharing: it needs 11.4 GB plus the 2 GiB reserve and the host had 5 to 10 GiB free. The before number is the same run counted with no file shared, and agrees with the set report's stored-byte sum (11.42 GB) and the retained 2026-10-01 report (11.42 GB).

Tests, by acceptance item:

- Rule: `TestRetainedBytesCountsASharedTargetOnce`, `TestRetainedBytesRejectsATargetThatIsNotPartOfTheArtifact`, `TestPoolTargetIsTheTargetOnlyOfAnArtifactLinkedToThePool`.
- Corpus cap: `TestCorpusByteCapCountsTheSharedTargetOnce` (6 cases with private copies, 86 with one shared target, at 170 MiB a case).
- Pruning: `TestPruneTargetPoolRemovesOnlyEntriesNothingShares`, `TestPruneTargetPoolRefusesAPoolThatIsNotADirectoryOfItsOwner`, `TestPruneQualifiedCampaignsRemovesASharedTargetOnlyWithItsLastArtifact`, `TestCorpusKeepsItsTargetUntilNoCaseSharesIt` (the entry a corpus case still shares), `TestPublishCreatesAgainAPoolEntryPrunedBeforeItsLink`.
- Merge: `TestMergeCountsTheTargetOfItsSuccessEvidenceOnce` (one root and two roots, a limit one byte short, reopen, `--partial`), `TestOpenMergedCampaignCountsEvidenceWithoutATargetInFull`.
- Inspect: `TestRunInspectReportsWhetherTheTargetIsShared` (text and JSON).

No existing test changed. Fourteen single-change mutations each turned a named test red (`mutation-checks.json`).

Gates on darwin/arm64 at b12b15b1c: `test-host` exit 0 (45 packages, 186 s), `validate` exit 0, architecture test, vet (host and linux/amd64), and `-race` on the artifact pool tests exit 0. The representative set ran unpruned at the same revision: 28/28 qualified, exit 0, 34 minutes.
GATE_SKIPPED:unittest:green-receipt b12b15b1 - Verify at the evidence commits reused the post-commit pass; only .flow files changed after it

Not met or not run:

- linux/amd64 (no native host).
- A pruned set run, and a guided campaign filling a corpus with `./tests` cases. The corpus counts apply the cap's arithmetic to the stored bytes of the 48 real `./tests` artifacts of the measured run.
- The spec's literal `go -C tools/gomad3 test` Quick command with an unpatched go; `test-host` covers the three package sets.
- A platform without link counts: there pruning removes nothing and inspect omits the field. Read from code only.
- The `--success-bytes` limit inside one campaign still charges every artifact its full stored bytes. That check is in `runner/runner.go` and `runner/retention.go`, outside this task's Touches.
- The spec's Decision Context was not updated; the spec file is outside Touches.

Review: SHIP on the first round from claude-fable-5-1 at high through the `claude` backend (same family as the writer; the reviewer had no shell and read the committed evidence). Four P3 findings stay open as follow-ups:

- `validateMergedCampaign` widens the limits it passes to `validateCampaign`. `validateCampaign` should take an explicit way to skip the per-execution success-byte bound.
- Merge counts a target once whether or not the shard stores share it on disk, while the corpus checks the link. The `RetainedBytes` contract should say merge accounts a single hypothetical store.
- `inspect` reports `shared` from the link count alone. The `TargetReport.Sharing` comment says "held with its store's pool", which is not checked, and a pooled store's fallback copy reads `private`, never `unshared`.
- `TestOpenMergedCampaignCountsEvidenceWithoutATargetInFull` exercises the accounting helper; no test opens an older merged record through `OpenMergedCampaign`.

The reviewer also noted, below its reporting threshold: a prune that races a publisher can drop a pool entry an artifact has just linked, which costs deduplication and no data; and a publication that loses its entry three times fails where it could fall back to a private copy.

For task 14: README's "about 11 GiB" for the unpruned set is now about 1.1 GiB; CLI docs need the `sharing` field and the pruning of `targets`; the after number on the qualified candidate is still to be taken there.

Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/retained-bytes.md` and `task-10/`.

stage: impl-review - ran (SHIP, claude:claude-fable-5-1:high, round 1)
## Evidence
- Commits: b12b15b1cb6a3fcd6557380652b88856d96a3965, 982f4c2b232cc2ceb9aedd4e8af5bbe84c87397c, 3defc23d07f9eb0bab8a8495629f35d707d1dbc8
- Tests: baseline: green via handoff (full test-host verified at 577003f9 by fn-114-gomad-correct-search-path-defects-and.9; only .flow evidence commits since), GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host (at b12b15b1c: exit 0, 45 packages, 186 s, pinned stock go1.27.1 first on PATH), GATE_SKIPPED:unittest:green-receipt b12b15b1 - Verify at the evidence commits reused the post-commit pass; only .flow files changed after it, make -C tools/gomad3 validate (exit 0), .toolchain/bin/go test -tags test_dep -count=1 -run TestPackageArchitecture . (exit 0), .toolchain/bin/go vet -tags test_dep ./artifact/... ./runner/... ./cmd/... ./qualification/... (exit 0; GOOS=linux GOARCH=amd64 vet of the changed packages exit 0), .toolchain/bin/go test -tags test_dep -count=1 -race -run 'TestPrune|TestPublish|TestRetained|TestPoolTarget' ./artifact/ (exit 0), make -C tools/gomad3 qualification-set with temporal.json, unpruned, artifacts in scratch (at b12b15b1c: exit 0, 28/28 qualified, 34 min; retained 1,177,358,336 bytes on disk against 11,442,111,182 without sharing; .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/retained-bytes.md), 14 single-change mutations, each red against its named test (.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-10/mutation-checks.json), NOT RUN: linux/amd64 (no native host); a set run at a revision before targets were shared (disk); a pruned set run; a guided campaign filling a corpus with ./tests cases; the spec's literal 'go -C tools/gomad3 test' Quick command with an unpatched go; root make lint-code-fast
- PRs:
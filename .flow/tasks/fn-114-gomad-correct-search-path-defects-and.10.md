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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

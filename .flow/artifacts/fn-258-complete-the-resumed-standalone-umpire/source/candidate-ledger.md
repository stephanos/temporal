# Renewed candidate ownership inventory

Frozen source: `ac60daaf47d2513b41eeff612f0e2758d33e186d`. The [ledger](candidate-ledger.json) records observed refs and registered worktrees separately from the frozen source, with explicit path ownership and original task/proof links. It is a provisional inventory supporting local migration preparation. No original candidate transfer or R6 disposition is complete.

The observation covers 88 repository/ref entries, including every one of the 81 original candidates, all current local branches in both repositories, 39 Temporal worktrees, one standalone worktree, and other observed ref namespaces. Ownership categories are 36 mixed, 15 framework, 28 Temporal, and nine legacy. Merge-base deltas include inherited assemblies; they do not establish exclusive contribution. Twenty-six original candidates still lack a corroborated task assignment.

Eighty original branch refs remain and contain their original heads. The removed branch `refs/heads/work/fn2551-prep-generator` retains its original object `d9a378401bbf9c9e6a7aaf60c03e121c392512f5` through `refs/recovery/fn257-candidates/work-fn2551-prep-generator`, current `umpire`, and other recorded containing refs. No candidate is silently omitted because its former branch name is absent. Missing, shared, experimental and historically mapped candidates retain explicit unresolved actions rather than guessed owners.

[The integrated delta](integrated-framework-delta.json) records every old/new blob and mode from `c88fdc5506` to the frozen source. With rename inference disabled this is 283 path records: 28 modified, 133 added and 122 deleted; ordinary rename presentation reports 161 changed files. The verifier reproduces this delta and its relative binary patch digest.

Standalone checkpoint `58a6989fe5ccd6a1ae734326d568f60dce2dab43` preserves the integrated framework delta and is an ancestor of the observed standalone preparation head `45eb554539fe934f43a1cb906381babd46bfc662`. Consumer lane `work/fn2583-consumers` was observed at `7e655da80b3810807cbd02a3ea29c26ceb9804e4`. The earlier standalone quick and Temporal consumer preparation refs are also retained. These are ongoing migration preparation, not transfer of the original candidates. Their heads and dirty paths may advance after this observation.

Dirty observations include the unresolved `fn254-source-assembly` index, frozen `fn2469-baseline` control, `prep2553-fuzz` sources and ongoing preparation checkouts. Status bytes, per-path ownership, index-stage digest and tracked worktree-diff digest are retained. This ledger does not back up their bytes or inventory ignored files. Before any retirement, checkpoint and verify exact dirty/untracked and unique ignored source/evidence; preserve the assembly index and active source lanes. No cleanup is authorized by this inventory.

Every candidate names original recovery, provisional source ownership, known task records, evidence links, cross-repository prerequisites and required destination actions. Remaining R6 work is to select actual hunks, deduplicate assemblies, resolve unknown task owners, create receiving recovery refs and tracker/obligation records, preserve deferred dependencies/evidence, and verify the exact consumer pair and final source ownership. Old validation-hold references are retained as historical context; resumed fn-258 execution supersedes that hold. Publication remains a separate destination/access decision.

Reproduce verification from the Temporal root:

```sh
python3 .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/candidate-ledger.py
```

The verifier checks all 1,189 manifest records against Git blob hashes/sizes and framework tree modes/paths; all original-candidate and observed local-branch coverage; every frozen ref object's availability; every Temporal candidate's complete committed merge-base delta and ownership mapping; and input digests. It writes [verification results](candidate-ledger-verification.json). Later ref advances are reported as live drift rather than changing the frozen inventory. `--capture` deliberately replaces the renewed observation and ledger; it never modifies refs, checkouts, source, Flow runtime or historical fn-257 artifacts.
